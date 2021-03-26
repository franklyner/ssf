package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/franklyner/ssf/server"
)

var (
	configProperties []string = []string{server.ConfigPort, server.ConfigReadTimeout, server.ConfigWriteTimeout, "name"}
)

func main() {
	server := initServer("./cmd/minimal")
	server.Start()
}

func initServer(configPath string) *server.Server {
	cfg := server.CreateConfig(configPath, "minimal", configProperties)
	ctrProviders := []server.ControllerProvider{minControllerProvider{
		Name: cfg.Get("name"),
	}}

	server := server.CreateServer(cfg, ctrProviders)
	server.RegisterService("hello", helloService{})
	return server
}

type minControllerProvider struct {
	Name string
}

func (m minControllerProvider) GetControllers() []server.Controller {
	srvConf := make(map[string]string)
	srvConf["name"] = m.Name
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
			Config:         srvConf,
			ControllerFunc: service,
		},
		{
			Name:           "JWTController",
			Metric:         "JWTController",
			Methods:        []string{"GET"},
			IsSecured:      true,
			Path:           "/jwt.html",
			ControllerFunc: jwtController,
			AuthFunc:       getJwtAuth(),
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
	ctx.SendHTMLResponse(200, []byte(helloSrv.sayHello(ctx.Controller.Config["name"])))
}

func getJwtAuth() func(ctx *server.Context) error {
	jmw := server.GetJWTMiddlewareHanlder("https://maxbrain-dev.eu.auth0.com/", "")
	return func(ctx *server.Context) error {
		err := jmw.CheckJWT(httptest.NewRecorder(), ctx.Request)
		return err
	}
}

func jwtController(ctx *server.Context) {
	ctx.SendHTMLResponse(http.StatusOK, []byte("if you see this, it worked!"))
}
