package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/form3tech-oss/jwt-go"
	"github.com/franklyner/ssf/server"
)

const PREFIX = "/min"

var (
	ConfigProperties []string = []string{server.ConfigPort, server.ConfigReadTimeout, server.ConfigWriteTimeout, "name"}
)

func main() {
	cfg := server.CreateConfig("./cmd/minimal", "minimal", ConfigProperties)
	server := initServer(cfg)
	go generateOtherMetric(server)
	server.Start()
}

func generateOtherMetric(srv *server.Server) {
	stats := srv.InitNonRequestContext().StatusInformation
	for {
		time.Sleep(1 * time.Second)
		value := rand.Intn(1000)
		stats.SetMetric("Other Metric", value)
	}
}

func initServer(config server.Config) *server.Server {
	ctrProviders := []server.ControllerProvider{minControllerProvider{
		Name: config.Get("name"),
	}}

	server := server.CreateServerWithPrefix(config, ctrProviders, PREFIX)
	server.RegisterService("hello", helloService{})
	return server
}

type minControllerProvider struct {
	Name string
}

func (m minControllerProvider) GetControllers() []server.Controller {
	ctrl := []server.Controller{
		{
			Name:           "Index",
			Metric:         "IndexCtrl",
			Methods:        []string{"GET"},
			IsSecured:      false,
			Path:           "/index.html",
			ControllerFunc: index,
			Description:    "Basic hello world. returns code bad request if query param fail is set to any non-empty value",
		},
		SecuredControlller,
		{
			Name:           "ServiceController",
			Metric:         "ServiceController",
			Methods:        []string{"GET"},
			IsSecured:      false,
			Path:           "/service.html",
			ControllerFunc: service,
			Description:    "Says hello world using a service",
		},
		{
			Name:           "JWTController",
			Metric:         "JWTController",
			Methods:        []string{"GET"},
			IsSecured:      true,
			Path:           "/jwt.html",
			ControllerFunc: jwtController,
			AuthFunc:       server.GetJwtAuth("https://login.dev.maxbrain.io/", claimsValidator),
			Description:    "Authenticates using a jwt",
		},
		{
			Name:           "LogLevelController",
			Metric:         "LogLevelController",
			Methods:        []string{"GET"},
			IsSecured:      false,
			Path:           "/loglevel",
			ControllerFunc: logController,
		},
		{
			Name:            "SubpathHandler",
			Metric:          "SubpathHandler",
			Methods:         []string{"GET"},
			IsSecured:       false,
			Path:            "/subpath",
			ControllerFunc:  subPathController,
			HandlesSubpaths: true,
		},
	}
	return ctrl
}

type helloService struct{}

func (h *helloService) sayHello(name string) string { return "Hello " + name }

func index(ctx *server.Context) {
	r := ctx.Request
	fail := r.FormValue("fail")
	msg := "Hello World!"
	code := 200
	if fail != "" {
		msg := "Received fail param!"
		ctx.LogError(msg)
		code = 400
	}
	ctx.SendHTMLResponse(code, []byte(msg))
}

// SecuredControlller Says hello if secured
var SecuredControlller server.Controller = server.Controller{
	Name:      "SecuredControlller",
	Metric:    "SecuredControlller",
	Methods:   []string{"GET"},
	IsSecured: true,
	Path:      "/secured.html",
	ControllerFunc: func(ctx *server.Context) {
		ctx.SendHTMLResponse(200, []byte("Hello Secured World!"))
	},
	AuthFunc: func(ctx *server.Context) error {
		secure := ctx.Request.FormValue("secure")
		if secure == "true" {
			return nil
		}
		msg := "secure query param wasn't set"
		return fmt.Errorf(msg)
	},
	Description: "Only executes if query param secure=true is set.",
}

func service(ctx *server.Context) {
	helloSrv := ctx.GetService("hello").(helloService)
	ctrp := ctx.ControllerProvider.(minControllerProvider)
	ctx.SendHTMLResponse(200, []byte(helloSrv.sayHello(ctrp.Name)))
}

func jwtController(ctx *server.Context) {
	token := ctx.Request.Context().Value("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)

	ctx.SendHTMLResponse(http.StatusOK, []byte(fmt.Sprintf("Received token with following claims: %+v", claims)))
}

func claimsValidator(claims jwt.MapClaims) error {
	audiences := claims["aud"]
	fmt.Printf(reflect.TypeOf(audiences).String())
	switch audiences.(type) {
	case string:
		if audiences != "https://cockpit.maxbrain.io/api/" {
			return fmt.Errorf("token contains wrong audience: %s", audiences)
		}
	case []interface{}:
		for _, aud := range audiences.([]interface{}) {
			if aud.(string) == "https://cockpit.maxbrain.io/api/" {
				return nil
			}
		}
		return fmt.Errorf("correct audience not found in token")
	default:
		return fmt.Errorf("aud claim of unexpected type: %T", audiences)
	}
	return nil
}

func logController(ctx *server.Context) {
	ctx.LogDebug("This is a debug message")
	ctx.LogInfo("This is an info message")
	ctx.LogError("This is an error message")
	ctx.SendHTMLResponse(http.StatusOK, []byte("Check your logfile!"))
}

func subPathController(ctx *server.Context) {
	ctx.SendHTMLResponse(http.StatusOK, []byte(ctx.Request.URL.String()))

}
