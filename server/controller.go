package server

// Controller base type of all controllers
type Controller struct {
	Name           string
	Metric         string
	Path           string
	Method         string
	IsSecured      bool
	ControllerFunc func(ctx *Context)
}

// Execute executes the controller in the given context
func (c *Controller) Execute(ctx *Context) {
	ctx.StatusInformation.IncrementMetric(c.Metric)
	c.ControllerFunc(ctx)
}
