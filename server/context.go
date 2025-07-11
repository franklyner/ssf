package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	LogLevelDebug       = "debug"
	LogLevelInfo        = "info"
	ContextKeyRequestID = "request_id"
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
	Subdomain          string
	serviceMap         map[string]interface{}
	ControllerProvider ControllerProvider
	Controller         *Controller
	LogLevel           string
}

// JSONErrorResponse General format of error responses
type JSONErrorResponse struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	LogMessage string `json:"-"`
	RequestID  string `json:"request_id"`
}

func (jer JSONErrorResponse) Error() string {
	return jer.LogMessage
}

func ErrToJSONErrorResponsePreserveCode(err error, message string) JSONErrorResponse {
	var jerr JSONErrorResponse
	if !errors.As(err, &jerr) {
		jerr.Code = http.StatusInternalServerError
		jerr.Message = "internal_server_error"
	}
	if message != "" {
		jerr.LogMessage = fmt.Sprintf("%s, %s", message, err.Error())
	} else {
		jerr.LogMessage = err.Error()
	}
	return jerr
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

// GetResponseWriter returns the response writer if it was set
func (ctx *Context) GetResponseWriter() http.ResponseWriter {
	if ctx.responseWriter == nil {
		return nil
	}
	return ctx.responseWriter
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
func (ctx *Context) SendJSONResponse(code int, response []byte) {
	ctx.SendGenericResponse(code, response, "application/json")
}

// SendHTMLResponse uses SendGenericResponse but sets the content type to text/html
func (ctx *Context) SendHTMLResponse(code int, response []byte) {
	ctx.SendGenericResponse(code, response, "text/html")
}

// SendGenericResponse sends the response using the context if the response was not sent already.
func (ctx *Context) SendGenericResponse(code int, response []byte, contentType string) {
	var err error
	if ctx.IsResponseSent {
		ctx.LogError("Response for this request was already sent")
		return
	}
	ctx.SendResponseHeader("Content-Type", contentType)
	ctx.sendCode(code)
	w := ctx.responseWriter
	if w != nil {
		_, err = w.Write(response)
		if err != nil {
			err = fmt.Errorf("Error occured while sending response: %d, %s. Error: %w", code, response, err)
			ctx.LogError(err.Error())
		}
	}

	ctx.IsResponseSent = true
	return
}

func (ctx *Context) SendRedirect(newurl string, statusCode int) {
	http.Redirect(ctx.responseWriter, ctx.Request, newurl, statusCode)
}

// SendJsonError sends a properly formatted error response
func (ctx *Context) SendJsonError(err error) {
	jerr := ErrToJSONErrorResponsePreserveCode(err, "")
	ctx.LogError(fmt.Sprintf("Error response: code: %d, message: %s, log_message: %s", jerr.Code, jerr.Message, jerr.LogMessage))
	jerr.RequestID = ctx.GetRequestID()
	content, err := json.Marshal(&jerr)
	if err != nil {
		ctx.SendJsonError(fmt.Errorf("Error occurred while marshalling error response: repsonse: %+v, error: %w\n", content, err))
	}
	ctx.SendJSONResponse(jerr.Code, content)
}

func (ctx *Context) log(severity string, msg string) {
	log.Printf(`{"req_id": "%s", "severity": "%s", "controller": "%s"} %s`, ctx.GetRequestID(), severity, ctx.Controller.Name, msg)
}

// LogError logs an error
func (ctx *Context) LogError(msg string) {
	ctx.log("Error", msg)
}

// LogErrorf logs an error with formatting
func (ctx *Context) LogErrorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	ctx.LogError(msg)
}

// LogInfo logs an info message
func (ctx *Context) LogInfo(msg string) {
	ctx.log("Info", msg)
}

// LogInfof logs an info message with formatting
func (ctx *Context) LogInfof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	ctx.LogInfo(msg)
}

// LogDebug logs an debug message
func (ctx *Context) LogDebug(msg string) {
	if ctx.LogLevel == LogLevelDebug {
		ctx.log("Debug", msg)
	}
}

// LogInfof logs an debug message with formatting
func (ctx *Context) LogDebugf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	ctx.LogDebug(msg)
}

func (ctx *Context) GetRequestContextValue(key string) any {
	return ctx.Request.Context().Value(key)
}

func (ctx *Context) SetRequestContextValue(key string, value any) {
	newCtx := context.WithValue(ctx.Request.Context(), key, value)
	ctx.Request = ctx.Request.WithContext(newCtx)
}

func (ctx *Context) GetRequestID() string {
	return ctx.GetRequestContextValue(ContextKeyRequestID).(string)
}

func (ctx *Context) SetRequestID(id string) {
	ctx.SetRequestContextValue(ContextKeyRequestID, id)
}
