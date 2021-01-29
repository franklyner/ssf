package main

import (
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
