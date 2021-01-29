package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// Context intantiated for every request
type Context struct {
	Server            *Server
	Request           *http.Request
	Writer            http.ResponseWriter
	Config            *Config
	Repository        *Repository
	IsResponseSent    bool
	StatusInformation *StatusInformation
	requestBody       []byte
	testResponse      *bytes.Buffer
	RequestID         uuid.UUID
	Subdomain         string
	serviceMap        map[string]interface{}
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

// GetWriter provides response writer or testWriter if in test context
func (ctx *Context) GetWriter() io.Writer {
	if ctx.Writer != nil {
		return ctx.Writer
	}
	if ctx.testResponse == nil {
		ctx.testResponse = new(bytes.Buffer)
	}
	return ctx.testResponse
}

func (ctx *Context) getTestResponse() string {
	if ctx.testResponse == nil {
		return ""
	}
	return ctx.testResponse.String()
}

// GetRequestBody readx the full body and returns it
func (ctx *Context) GetRequestBody() ([]byte, error) {
	if len(ctx.requestBody) == 0 {
		body, err := ioutil.ReadAll(ctx.Request.Body)
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
	if ctx.Writer != nil {
		ctx.Writer.WriteHeader(code)
	}
}

// SendResponseHeader sends the response header
func (ctx *Context) SendResponseHeader(header string, value string) {
	if ctx.Writer != nil {
		ctx.Writer.Header().Set(header, value)
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
	ctx.sendCode(code)
	ctx.SendResponseHeader("Content-Type", contentType)
	w := ctx.GetWriter()
	_, err := fmt.Fprint(w, string(response))
	if err != nil {
		return fmt.Errorf("Error occured while sending response: %d, %s. Error: %w", code, response, err)
	}
	return nil
}

// SendAndLogError helper function that is specialized in sending back an error response to the client
func (ctx *Context) SendAndLogError(code int, message string, data string) {
	ctx.LogError(fmt.Sprintf("%s, Data: %s", message, data))
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

func (ctx *Context) log(msg string) {
	log.Printf("%s: %s", ctx.RequestID, msg)
}

// LogError logs an error
func (ctx *Context) LogError(msg string) {
	ctx.log("Error: " + msg)
}

// LogInfo logs an error
func (ctx *Context) LogInfo(msg string) {
	ctx.log("Info: " + msg)
}
