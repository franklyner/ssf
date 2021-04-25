package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
)

// Context intantiated for every request
type Context struct {
	Server             *Server
	Request            *http.Request
	responseWriter     http.ResponseWriter
	Repository         *Repository
	IsResponseSent     bool
	ResponseCode       int
	StatusInformation  *StatusInformation
	requestBody        []byte
	RequestID          string
	Subdomain          string
	serviceMap         map[string]interface{}
	ControllerProvider ControllerProvider
	Controller         *Controller
	LogLevel           string
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// GetService retrieves the specified service by name
func (ctx *Context) GetService(name string) interface{} {
	service, ok := ctx.serviceMap[name]
	if ok {
		return service
	}
	return fmt.Errorf("No service with name %s found", name)
}

// GetRequestBody readx the full body and returns it
func (ctx *Context) GetRequestBody() ([]byte, error) {
	if len(ctx.requestBody) == 0 {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		defer ctx.Request.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Error while reading request body: %w", err)
		}
		ctx.requestBody = body
	}
	return ctx.requestBody, nil
}

// SetRequestBodyManually sets the body. Mainly used for testing
func (ctx *Context) SetRequestBodyManually(body []byte) {
	ctx.requestBody = body
}

func (ctx *Context) sendCode(code int) {
	if ctx.responseWriter != nil {
		ctx.ResponseCode = code
		ctx.responseWriter.WriteHeader(code)
	}
}

// SendResponseHeader sends the response header
func (ctx *Context) SendResponseHeader(header string, value string) {
	if ctx.responseWriter != nil {
		ctx.responseWriter.Header().Set(header, value)
	}
}

// SendJSONResponse uses SendGenericResponse but sets the content type to application/json
func (ctx *Context) SendJSONResponse(code int, response []byte) error {
	return ctx.SendGenericResponse(code, response, "application/json")
}

// SendHTMLResponse uses SendGenericResponse but sets the content type to text/html
func (ctx *Context) SendHTMLResponse(code int, response []byte) error {
	return ctx.SendGenericResponse(code, response, "text/html")
}

// SendGenericResponse sends the response using the context if the response was not sent already.
func (ctx *Context) SendGenericResponse(code int, response []byte, contentType string) error {
	if ctx.IsResponseSent {
		return fmt.Errorf("Response for this request was already sent")
	}
	ctx.SendResponseHeader("Content-Type", contentType)
	ctx.sendCode(code)
	w := ctx.responseWriter
	_, err := w.Write(response)
	if err != nil {
		return fmt.Errorf("Error occured while sending response: %d, %s. Error: %w", code, response, err)
	}
	ctx.IsResponseSent = true
	return nil
}

// SendAndLogError helper function that is specialized in sending back an error response to the client
func (ctx *Context) SendAndLogError(code int, message string, data string) {
	ctx.LogError(fmt.Sprintf("Error response: code: %d, message: %s, Data: %s", code, message, data))
	var resp errorResponse = errorResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	content, err := json.Marshal(&resp)
	if err != nil {
		ctx.LogError(fmt.Sprintf("Error occurred while marshalling error response: repsonse: %+v, error: %v\n", resp, err))
		ctx.SendAndLogError(http.StatusInternalServerError, "Unable to marshal original error response", "")
	}

	err = ctx.SendJSONResponse(code, content)
	if err != nil {
		ctx.LogError(fmt.Sprintf("Error sending back error response: response: %+v, error: %+v\n", resp, err))
	}
}

func (ctx *Context) log(severity string, msg string) {
	log.Printf(`{"req_id": "%s", "severity": "%s", "controller": "%s"} %s`, ctx.RequestID, severity, ctx.Controller.Name, msg)
}

// LogError logs an error
func (ctx *Context) LogError(msg string) {
	ctx.log("Error", msg)
}

// LogInfo logs an error
func (ctx *Context) LogInfo(msg string) {
	ctx.log("Info", msg)
}

// LogDebug logs an error
func (ctx *Context) LogDebug(msg string) {
	if ctx.LogLevel == LogLevelDebug {
		ctx.log("Debug", msg)
	}
}
