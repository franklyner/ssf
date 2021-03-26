package server

// ControllerProvider Interfice providing access the list of controllers
// from another module. If a controller provider requires configuration
// then it is expected to export the parameters as fields and the
// function instantiating the server is responsible to fill them.
// The Config can be used for convenience.
type ControllerProvider interface {
	GetControllers() []Controller
}

// Controller base type of all controllers
type Controller struct {
	Name           string
	Metric         string
	Path           string
	Methods        []string
	IsSecured      bool
	AuthFunc       func(ctx *Context) error
	ControllerFunc func(ctx *Context, ctr *Controller)
	Config         map[string]string
}

// Execute executes the controller in the given context
func (ctr *Controller) Execute(ctx *Context) {
	ctx.StatusInformation.IncrementMetric(ctr.Metric)
	ctr.ControllerFunc(ctx, ctr)
}
