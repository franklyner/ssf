package main

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/form3tech-oss/jwt-go"
	"github.com/franklyner/ssf/server"
)

var (
	ConfigProperties []string = []string{server.ConfigPort, server.ConfigReadTimeout, server.ConfigWriteTimeout, "name"}
)

func main() {
	cfg := server.CreateConfig("./cmd/minimal", "minimal", ConfigProperties)
	server := initServer(cfg)
	server.Start()
}

func initServer(config server.Config) *server.Server {
	ctrProviders := []server.ControllerProvider{minControllerProvider{
		Name: config.Get("name"),
	}}

	server := server.CreateServer(config, ctrProviders)
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
		},
		SecuredControlller,
		{
			Name:           "ServiceController",
			Metric:         "ServiceController",
			Methods:        []string{"GET"},
			IsSecured:      false,
			Path:           "/service.html",
			ControllerFunc: service,
		},
		{
			Name:           "JWTController",
			Metric:         "JWTController",
			Methods:        []string{"GET"},
			IsSecured:      true,
			Path:           "/jwt.html",
			ControllerFunc: jwtController,
			AuthFunc:       server.GetJwtAuth("https://maxbrain-dev.eu.auth0.com/", claimsValidator),
		},
		{
			Name:           "LogLevelController",
			Metric:         "LogLevelController",
			Methods:        []string{"GET"},
			IsSecured:      false,
			Path:           "/loglevel",
			ControllerFunc: logController,
		},
	}
	return ctrl
}

type helloService struct{}

func (h *helloService) sayHello(name string) string { return "Hello " + name }

func index(ctx *server.Context) {
	r := ctx.Request
	fail := r.FormValue("fail")
	if fail != "" {
		ctx.SendAndLogError(http.StatusBadRequest, "Received fail param!", fail)
		return
	}
	ctx.SendHTMLResponse(200, []byte("Hello World!"))
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
		ctx.SendHTMLResponse(http.StatusBadRequest, []byte(msg))
		return fmt.Errorf(msg)
	},
}

func service(ctx *server.Context) {
	helloSrv := ctx.GetService("hello").(helloService)
	ctrp := ctx.ControllerProvider.(minControllerProvider)
	ctx.SendHTMLResponse(200, []byte(helloSrv.sayHello(ctrp.Name)))
}

func jwtController(ctx *server.Context) {
	ctx.SendHTMLResponse(http.StatusOK, []byte("if you see this, it worked!"))
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
