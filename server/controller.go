package server

// Controller base type of all controllers
type Controller struct {
	Name           string
	Metric         string
	Path           string
	Methods        []string
	IsSecured      bool
	AuthFunc       func(ctx *Context) error
	ControllerFunc func(ctx *Context)
}

// Execute executes the controller in the given context
func (ctr *Controller) Execute(ctx *Context) {
	ctx.StatusInformation.IncrementMetric(ctr.Metric)
	ctr.ControllerFunc(ctx)
}
