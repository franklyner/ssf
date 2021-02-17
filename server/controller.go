package server

import (
	"net/http"
)

// Controller base type of all controllers
type Controller struct {
	Name           string
	Metric         string
	Path           string
	Method         string
	IsSecured      bool
	AuthFunc       func(ctx *Context, w http.ResponseWriter, r *http.Request) error
	ControllerFunc func(ctx *Context)
}

// Execute executes the controller in the given context
func (ctr *Controller) Execute(ctx *Context) {
	ctx.StatusInformation.IncrementMetric(ctr.Metric)
	ctr.ControllerFunc(ctx)
}
