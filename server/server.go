package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"net/http/httptest"
	_ "net/http/pprof"
)

// All config properties required for the server to run
const (
	ConfigPort             = "port"
	ConfigReadTimeout      = "readTimeout"
	ConfigWriteTimeout     = "writeTimeout"
	ConfigLogLevel         = "loglevel"
	ConfigEnableProfiling  = "enable_profiling"
	ConfigEnablePrometheus = "enable_prometheus"
)

var (
	promHttpHist *prometheus.HistogramVec
)

// Server Generic server who is able to load a list of controllers from
// multiple ControllerProviders
type Server struct {
	config              Config
	controllers         []Controller
	statusInfo          *StatusInformation
	repository          *Repository
	serviceMap          map[string]interface{}
	requestHandler      http.Handler
	LogLevel            string
	pathPrefix          string
	isPrometheusEnabled bool
}

// GetControllers returns all controllers of the controller provider
func (s *Server) GetControllers() []Controller {
	return s.controllers
}

// CreateServer Factory method to create a Server instance. This is meant
// to be used by main methods and provide the list of ControllerProviders
// the server instance is supposed to serve
func CreateServer(config Config, ctrProviders []ControllerProvider) *Server {
	return CreateServerWithPrefix(config, ctrProviders, "")
}
func BlankServer() *Server {
	return &Server{serviceMap: make(map[string]interface{})}
}
func CreateServerWithPrefix(config Config, ctrProviders []ControllerProvider, pathPrefix string) *Server {
	server := Server{
		config:      config,
		controllers: []Controller{},
		statusInfo:  CreateStatusInfo(),
		pathPrefix:  pathPrefix,
	}

	r := mux.NewRouter()
	s := r
	if len(pathPrefix) > 0 {
		s = r.PathPrefix(pathPrefix).Subrouter()
	}

	for _, ctrProv := range ctrProviders {
		ctrList := ctrProv.GetControllers()
		for _, ctr := range ctrList {
			ctr.controllerProvider = ctrProv
			server.registerController(s, ctr)
		}
	}

	server.registerController(r, StatusController)

	prof := config.Get(ConfigEnableProfiling)
	if prof == "true" {
		s.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		log.Println("Enabled profiling endpoints on /debug/pprof/")
	}

	prom := config.Get(ConfigEnablePrometheus)
	if prom == "true" {
		server.isPrometheusEnabled = true
		promHttpHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ssf_server_controller_requestcount",
			Help:    "Counts the number of controller invokations",
			Buckets: []float64{1, 10, 50, 100, 200, 400, 800, 1500, 3000, 10000, 30000, 60000},
		}, []string{"controller"})

		s.Handle("/metrics", promhttp.Handler())
		log.Printf("Enabled prometheus metrics endpoint on %s/metrics", pathPrefix)
	}

	s.NotFoundHandler = getNotFoundHandler()
	server.requestHandler = r

	server.serviceMap = make(map[string]interface{})

	ll := config.Get(ConfigLogLevel)
	if ll == "" {
		ll = LogLevelInfo
	} else if (strings.ToLower(ll) != LogLevelDebug) && (strings.ToLower(ll) != LogLevelInfo) {
		log.Panicf("Invalid loglevel provided. Expecting %s or %s", LogLevelDebug, LogLevelInfo)
	}

	server.LogLevel = ll
	return &server
}

// SetRepository sets the repository if one is being used.
func (s *Server) SetRepository(repo *Repository) {
	s.repository = repo
}

func (s *Server) GetRepository() *Repository {
	return s.repository
}

// RegisterService registers a service to the server
func (s *Server) RegisterService(name string, service interface{}) {
	s.serviceMap[name] = service
}

func (s *Server) GetService(name string) interface{} {
	return s.serviceMap[name]
}

// Start starts the previously initialized server
func (s *Server) Start() {

	port := s.config.Get(ConfigPort)
	rt, err1 := s.config.GetDuration(ConfigReadTimeout)
	wt, err2 := s.config.GetDuration(ConfigWriteTimeout)
	if err1 != nil || err2 != nil {
		panic(fmt.Sprintf("%s, %s", err1, err2))
	}
	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		ReadTimeout:  rt,
		WriteTimeout: wt,
		Handler:      s.requestHandler,
	}

	log.Printf("Starting listening on port: %s", port)
	log.Fatal(httpSrv.ListenAndServe())
}

// GetMainHandler Gives access to the mux router for testing purposes
func (s *Server) GetMainHandler() http.Handler {
	return s.requestHandler
}

// initContext initialzes the context for the given request
func (s *Server) initContext(w http.ResponseWriter, r *http.Request, c Controller) *Context {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = uuid.New().String()
	}

	context := &Context{
		Server:             s,
		Request:            r,
		responseWriter:     w,
		StatusInformation:  s.statusInfo,
		Repository:         s.repository,
		serviceMap:         s.serviceMap,
		LogLevel:           s.LogLevel,
		ControllerProvider: c.controllerProvider,
		Controller:         &c,
	}
	context.SetRequestID(reqID)
	return context
}

func (s *Server) registerController(r *mux.Router, c Controller) {
	s.controllers = append(s.controllers, c)

	ctrHandler := http.HandlerFunc(s.getControllerHandlerFunc(c))

	r.Handle(c.Path, ctrHandler).Methods(c.Methods...)
	log.Printf("Registered controller %s", c.Name)
}

func (s *Server) getControllerHandlerFunc(c Controller) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UnixNano()
		ctx := s.initContext(w, r, c)
		ctx.LogDebug(fmt.Sprintf("Executing %s for request: %s", c.Name, r.RequestURI))
		if c.IsSecured {
			err := c.AuthFunc(ctx)
			if err != nil {
				if ctx.IsResponseSent {
					ctx.LogError(fmt.Sprintf("Authentication for controller %s failed with code: %d: %s", c.Name, ctx.ResponseCode, err.Error()))
					return
				}
				ctx.SendJsonError(JSONErrorResponse{
					Code:       http.StatusUnauthorized,
					Message:    "unauthorized",
					LogMessage: fmt.Sprintf("Authentication for controller %s failed: %s", c.Name, err.Error()),
				})
				return
			}
		}
		c.Execute(ctx)
		duration := time.Now().UnixNano() - start
		ctx.LogDebug(formatExecLogMessage(r, duration, ctx.ResponseCode))
		if s.isPrometheusEnabled {
			observed := float64(duration) / 1000000 // calc in ms
			promHttpHist.With(prometheus.Labels{"controller": c.Name}).Observe(observed)
		}
	}
}

func formatExecLogMessage(r *http.Request, duration int64, code int) string {
	uri := r.URL.String()
	method := r.Method
	return fmt.Sprintf("%s %s: processing duration: %d ns, returned code: %d", method, uri, duration, code)
}

func getNotFoundHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		code := http.StatusNotFound
		log.Println(formatExecLogMessage(r, 0, code) + " NOT FOUND")
		w.WriteHeader(code)
		fmt.Fprint(w, "Not Found!")
	}

	// http.HandlerFunc wraps a function so that it
	// implements http.Handler interface
	return http.HandlerFunc(fn)
}

func GetJwtAuth(issuer string, customValidator func(claims jwt.MapClaims) error) func(ctx *Context) error {
	jmw := getJWTMiddlewareHandler(issuer, customValidator)
	return func(ctx *Context) error {
		err := jmw.CheckJWT(httptest.NewRecorder(), ctx.Request)
		return err
	}
}

func getJWTMiddlewareHandler(issuer string, customValidator func(claims jwt.MapClaims) error) *jwtmiddleware.JWTMiddleware {
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)
			if claims["iss"] != issuer {
				msg := fmt.Sprintf("token has wrong issuer: %s", claims["iss"])
				fmt.Println(msg)
				return nil, errors.New(msg)
			}

			err := customValidator(claims)
			if err != nil {
				return nil, fmt.Errorf("custom claims validator returned error: %w", err)
			}

			cert, err := getPemCert(token)
			if err != nil {
				return nil, fmt.Errorf("unable to fetch token certificate with JWKS: %w", err)
			}
			result, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			if err != nil {
				return nil, fmt.Errorf("error parsing public key from JWKS endpoint: %w", err)
			}

			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
	return jwtMiddleware
}

// ok to cache globally as multiple server instance can share the same cache
var keyMap map[string][]byte = make(map[string][]byte)

func getPemCert(token *jwt.Token) (string, error) {
	cert := ""

	claims := token.Claims.(jwt.MapClaims)
	issuer := claims["iss"].(string)
	if _, ok := keyMap[issuer]; !ok {
		url := claims["iss"].(string) + ".well-known/jwks.json"

		resp, err := http.Get(url)

		if err != nil {
			return cert, err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return cert, nil
		}
		keyMap[issuer] = body
	}

	reader := bytes.NewReader(keyMap[issuer])
	var jwks = Jwks{}
	err := json.NewDecoder(reader).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			fmt.Printf("JWKS endpoint result: found kid: %s\n", jwks.Keys[k].Kid)
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}

// Jwks used to marshal the key response from Auth0
type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

// JSONWebKeys used to marshal the key response from Auth0
type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}
