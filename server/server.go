package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// All config properties required for the server to run
const (
	ConfigPort         = "port"
	ConfigReadTimeout  = "readTimeout"
	ConfigWriteTimeout = "writeTimeout"
)

// Server Generic server who is able to load a list of controllers from
// multiple ControllerProviders
type Server struct {
	config         Config
	controllers    []Controller
	statusInfo     *StatusInformation
	repository     *Repository
	serviceMap     map[string]interface{}
	requestHandler http.Handler
}

// GetControllers returns all controllers of the controller provider
func (s *Server) GetControllers() []Controller {
	return s.controllers
}

// CreateServer Factory method to create a Server instance. This is meant
// to be used by main methods and provide the list of ControllerProviders
// the server instance is supposed to serve
func CreateServer(config Config, ctrProviders []ControllerProvider) *Server {
	server := Server{
		config:      config,
		controllers: []Controller{},
		statusInfo:  CreateStatusInfo(),
	}

	r := mux.NewRouter()

	for _, ctrProv := range ctrProviders {
		ctrList := ctrProv.GetControllers()
		for _, ctr := range ctrList {
			server.registerController(r, ctr)
		}
	}

	r.NotFoundHandler = getNotFoundHandler()

	server.registerController(r, StatusController)
	server.requestHandler = r

	server.serviceMap = make(map[string]interface{})
	return &server
}

// SetRepository sets the repository if one is being used.
func (s *Server) SetRepository(repo *Repository) {
	s.repository = repo
}

// RegisterService registers a service to the server
func (s *Server) RegisterService(name string, service interface{}) {
	s.serviceMap[name] = service
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
func (s *Server) initContext(w http.ResponseWriter, r *http.Request) *Context {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = uuid.New().String()
	}

	context := Context{
		Server:            s,
		Request:           r,
		responseWriter:    w,
		StatusInformation: s.statusInfo,
		Repository:        s.repository,
		serviceMap:        s.serviceMap,
		RequestID:         reqID,
	}
	return &context
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
		ctx := s.initContext(w, r)
		if c.IsSecured {
			err := c.AuthFunc(ctx)
			if err != nil {
				ctx.LogError(fmt.Sprintf("Authentication for controller %s failed: %s", c.Name, err.Error()))
				return
			}
		}
		ctx.LogInfo("Executing " + c.Name)
		c.Execute(ctx)
		duration := time.Now().UnixNano() - start

		ctx.LogInfo(formatExecLogMessage(r, duration))
	}
}

func formatExecLogMessage(r *http.Request, duration int64) string {
	uri := r.URL.String()
	method := r.Method
	return fmt.Sprintf("%s %s: processing duration: %d ns", method, uri, duration)
}

func getNotFoundHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		log.Println(formatExecLogMessage(r, 0) + " NOT FOUND")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found!")
	}

	// http.HandlerFunc wraps a function so that it
	// implements http.Handler interface
	return http.HandlerFunc(fn)
}

// todo: will need to make issuer and audience parameterized and create a func
// that creates a handler that can be added to a controller
// how to use jwtMiddleware
// jwtmid := getJWTMiddlewareHanlder()
// ctr = jwtmid.Handler(ctr).(http.HandlerFunc)
// fmt.Printf("Secured controller %s", c.Name)

func getJWTMiddlewareHanlder() *jwtmiddleware.JWTMiddleware {
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// todo: verify issuer and audience?

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}
			result, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			if err != nil {
				fmt.Printf("Error parsing public key from JWKS endpoint: Error: %+v", err)
				return nil, err
			}

			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
	return jwtMiddleware
}

func getPemCert(token *jwt.Token) (string, error) {
	cert := ""

	claims := token.Claims.(jwt.MapClaims)
	url := claims["iss"].(string) + ".well-known/jwks.json"
	fmt.Printf("Getting certificate with: %s\n", url)
	resp, err := http.Get(url)

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

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
