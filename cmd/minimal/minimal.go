package main

import (
	"fmt"
	"net/http"

	"github.com/franklyner/ssf/server"
)

var (
	configProperties []string = []string{server.ConfigPort, server.ConfigReadTimeout, server.ConfigWriteTimeout}
)

func main() {
	server := initServer("./cmd/minimal")
	server.Start()
}

func initServer(configPath string) *server.Server {
	cfg := server.CreateConfig(configPath, "minimal", configProperties)
	ctrProviders := []server.ControllerProvider{minControllerProvider{}}

	server := server.CreateServer(cfg, ctrProviders)
	server.RegisterService("hello", helloService{})
	return server
}

type minControllerProvider struct{}

func (t minControllerProvider) GetControllers() []server.Controller {
	ctrl := []server.Controller{
		IndexController,
		SecuredControlller,
		ServiceController,
	}
	return ctrl
}

type helloService struct{}

func (h *helloService) sayHello() string { return "Hello World!" }

// IndexController Says hello
var IndexController server.Controller = server.Controller{
	Name:      "Index",
	Metric:    "IndexCtrl",
	Method:    "GET",
	IsSecured: false,
	Path:      "/index.html",
	ControllerFunc: func(ctx *server.Context) {
		ctx.SendHTMLResponse(200, []byte("Hello World!"))
	},
}

// SecuredControlller Says hello if secured
var SecuredControlller server.Controller = server.Controller{
	Name:      "SecuredControlller",
	Metric:    "SecuredControlller",
	Method:    "GET",
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

// ServiceController Uses a service to say hello
var ServiceController server.Controller = server.Controller{
	Name:      "ServiceController",
	Metric:    "ServiceController",
	Method:    "GET",
	IsSecured: false,
	Path:      "/service.html",
	ControllerFunc: func(ctx *server.Context) {
		helloSrv := ctx.GetService("hello").(helloService)
		ctx.SendHTMLResponse(200, []byte(helloSrv.sayHello()))
	},
}
