package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSONRPCResponse is a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SendJSONRPCResult writes a JSON-RPC 2.0 success response with HTTP 200.
func (ctx *Context) SendJSONRPCResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
	body, err := json.Marshal(resp)
	if err != nil {
		ctx.SendJsonError(JSONErrorResponse{
			Code:       http.StatusInternalServerError,
			Message:    "error marshalling JSON-RPC response",
			LogMessage: fmt.Sprintf("error marshalling JSON-RPC response: %v", err),
		})
		return
	}
	ctx.SendJSONResponse(http.StatusOK, body)
}

// SendJSONRPCError writes a JSON-RPC 2.0 error response with HTTP 200.
func (ctx *Context) SendJSONRPCError(id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	}
	body, err := json.Marshal(resp)
	if err != nil {
		ctx.SendJsonError(JSONErrorResponse{
			Code:       http.StatusInternalServerError,
			Message:    "error marshalling JSON-RPC error",
			LogMessage: fmt.Sprintf("error marshalling JSON-RPC error: %v", err),
		})
		return
	}
	ctx.SendJSONResponse(http.StatusOK, body)
}
