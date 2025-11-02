package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (s *Server) handleOpenAPI(r *mux.Router) {
	openAPI := s.config.Get(ConfigEnableOpenAPI)
	if openAPI != "true" {
		return
	}
	s.buildOpenApiDoc()

	// handler for OpenAPI spec
	swaggerJson := "/swagger/doc.json"
	r.Handle(swaggerJson, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := s.openApiDocs.MarshalJSON()
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})).Methods(http.MethodGet, http.MethodHead)

	// Handler for swagger UI
	url := s.pathPrefix + "/swagger/doc.json"
	log.Printf("Enabled OpenAPI endpoint on %s", url)
	r.PathPrefix("/swagger").Handler(httpSwagger.Handler(
		httpSwagger.URL(url),
	))

	// ReDoc (sleek docs)
	r.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<!DOCTYPE html>
		<html>
		  <head><title>API Docs</title></head>
		  <body>
		    <redoc spec-url='`+s.pathPrefix+`/swagger/doc.json'></redoc>
		    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
		  </body>
		</html>`)
	})
	log.Printf("Enabled ReDoc endpoint on %s/docs", s.pathPrefix)
}

func (s *Server) buildOpenApiDoc() {
	isEnabled := s.config.Get(ConfigEnableOpenAPI)
	if isEnabled != "true" {
		return
	}
	s.baseURL = s.config.Get(ConfigBaseURL)
	if s.baseURL == "" {
		s.baseURL = "http://localhost:" + s.config.Get(ConfigPort) + s.pathPrefix
	} else {
		s.baseURL = strings.TrimSuffix(s.baseURL, "/") + s.pathPrefix
	}

	title := s.config.Get(ConfigAppName)
	description := s.config.Get(ConfigAppDescription)
	version := s.config.Get(ConfigAppVersion)
	securitySchemeName := s.config.Get(ConfigSecurityScheme)
	if securitySchemeName == "" {
		securitySchemeName = "bearerAuth"
	}

	if title == "" || description == "" || version == "" {
		log.Printf("Warning: OpenAPI app metadata not fully set. Title: %q, Description: %q, Version: %q", title, description, version)
	}

	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       title,
			Description: description,
			Version:     version,
		},
		Servers: openapi3.Servers{
			&openapi3.Server{URL: s.baseURL},
		},
		Paths: &openapi3.Paths{},
	}

	var secScheme *openapi3.SecuritySchemeRef
	switch securitySchemeName {
	case "bearerAuth":
		secScheme = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		}
	case "apiKey":
		secScheme = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type: "apiKey",
				Name: "X-API-KEY",
				In:   "header",
			},
		}

	}
	doc.Components = &openapi3.Components{
		SecuritySchemes: openapi3.SecuritySchemes{
			securitySchemeName: secScheme,
		},
	}

	for _, c := range s.controllers {
		if c.OpenAPISpec == nil {
			log.Printf("Skipping controller %s for OpenAPI doc generation: no OpenAPISpec defined", c.Name)
			continue
		}
		spec := c.OpenAPISpec
		// Path item (ensure exists)
		pi := doc.Paths.Find(c.Path)
		if pi == nil {
			pi = &openapi3.PathItem{}
			doc.Paths.Set(c.Path, pi)
		}

		// Map each HTTP method to an operation
		for _, m := range c.Methods {
			op := &openapi3.Operation{
				OperationID: firstNonEmpty(spec.OperationID, c.Name+"_"+m),
				Summary:     spec.Summary,
				Description: c.Description,
				Tags:        spec.Tags,
			}

			// Security (optional)
			if c.IsSecured {
				op.Security = &openapi3.SecurityRequirements{
					openapi3.SecurityRequirement{"bearerAuth": {}},
				}
			}

			// Params
			for _, p := range spec.Params {
				sref := p.Schema
				if sref == nil {
					// fall back to string if not provided
					sref = openapi3.NewStringSchema().NewRef()
				}
				op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        p.Name,
						In:          p.In,
						Required:    p.Required,
						Description: p.Description,
						Schema:      sref,
					},
				})
			}

			// Request body
			if rb := spec.RequestBody; rb != nil {
				if rb.Schema == nil {
					rb.Schema = openapi3.NewObjectSchema().NewRef()
				}
				op.RequestBody = &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Description: rb.Description,
						Required:    rb.Required,
						Content: openapi3.Content{
							rb.ContentType: &openapi3.MediaType{Schema: rb.Schema},
						},
					},
				}
			}

			// Responses (need at least one)
			resps := openapi3.NewResponses()
			for _, r := range spec.Responses {
				sref := r.Schema
				if sref == nil {
					sref = openapi3.NewObjectSchema().NewRef()
				}
				key := fmt.Sprintf("%d", r.StatusCode)
				resps.Set(key, &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: &r.Description,
						Content: openapi3.Content{
							r.ContentType: &openapi3.MediaType{Schema: sref},
						},
					},
				})
			}
			if resps.Len() == 0 {
				// provide a default 200 if none declared
				resps.Set("200", &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: ptr("OK"),
						Content: openapi3.Content{
							"application/json": {Schema: openapi3.NewObjectSchema().NewRef()},
						},
					},
				})
			}
			op.Responses = resps

			// Attach op to method
			switch strings.ToUpper(m) {
			case http.MethodGet:
				pi.Get = op
			case http.MethodPost:
				pi.Post = op
			case http.MethodPut:
				pi.Put = op
			case http.MethodPatch:
				pi.Patch = op
			case http.MethodDelete:
				pi.Delete = op
			case http.MethodHead:
				pi.Head = op
			case http.MethodOptions:
				pi.Options = op
			}
		}
	}

	// Optional: validate the doc
	err := doc.Validate(context.Background())
	if err != nil {
		log.Printf("Warning: generated OpenAPI doc is invalid: %s", err)
	}
	s.openApiDocs = doc
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
func ptr[T any](v T) *T { return &v }
