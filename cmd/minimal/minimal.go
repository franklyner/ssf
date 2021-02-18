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
	cfg := server.CreateConfig("./cmd/minimal", "minimal", configProperties)
	ctrProviders := []server.ControllerProvider{minControllerProvider{}}

	server := server.CreateServer(cfg, ctrProviders)
	server.Start()
}

type minControllerProvider struct{}

func (t minControllerProvider) GetControllers() []server.Controller {
	ctrl := []server.Controller{
		IndexController,
		SecuredControlller,
	}
	return ctrl
}

// IndexController just redirects to login
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

// SecuredControlller just redirects to login
var SecuredControlller server.Controller = server.Controller{
	Name:      "SecuredControlller",
	Metric:    "SecuredControlller",
	Method:    "GET",
	IsSecured: true,
	Path:      "/secured.html",
	ControllerFunc: func(ctx *server.Context) {
		ctx.SendHTMLResponse(200, []byte("Hello Secured World!"))
	},
	AuthFunc: func(ctx *server.Context, w http.ResponseWriter, r *http.Request) error {
		secure := r.FormValue("secure")
		if secure == "true" {
			return nil
		}
		msg := "secure query param wasn't set"
		ctx.SendHTMLResponse(http.StatusBadRequest, []byte(msg))
		return fmt.Errorf(msg)
	},
}
