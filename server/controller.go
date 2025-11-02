package server

import "github.com/getkin/kin-openapi/openapi3"

// ControllerProvider Interface providing access the list of controllers
// from another module. If a controller provider requires configuration
// then it is expected to export the parameters as fields and the
// function instantiating the server is responsible to fill them.
// The Config can be used for convenience.
type ControllerProvider interface {
	GetControllers() []Controller
}

// Controller base type of all controllers
type Controller struct {
	Name               string
	Metric             string
	Path               string
	HandlesSubpaths    bool
	Methods            []string
	IsSecured          bool
	AuthFunc           func(ctx *Context) error
	ControllerFunc     func(ctx *Context)
	controllerProvider ControllerProvider
	Description        string

	OpenAPISpec *OASpec
}

// Execute executes the controller in the given context
func (ctr *Controller) Execute(ctx *Context) {
	ctx.StatusInformation.IncrementMetric(ctr.Metric)
	ctr.ControllerFunc(ctx)
}

// Open API specification structure
// For schema generation:
type SchemaProvider interface {
	OpenAPISchemaRef() *openapi3.SchemaRef
}

// Lightweight param/request/response descriptors:
type OAParam struct {
	Name        string
	In          string // "path", "query", "header"
	Required    bool
	Description string
	Schema      *openapi3.SchemaRef // use openapi3gen to build this if nil
}

type OARequestBody struct {
	Description string
	ContentType string // "application/json"
	Schema      *openapi3.SchemaRef
	Required    bool
}

type OAResponse struct {
	StatusCode  int
	Description string
	ContentType string // "application/json"
	Schema      *openapi3.SchemaRef
}

type OASpec struct {
	OperationID string
	Tags        []string
	Summary     string
	Params      []OAParam
	RequestBody *OARequestBody
	Responses   []OAResponse
}
