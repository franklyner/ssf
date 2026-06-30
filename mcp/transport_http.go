package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/franklyner/ssf/server"
)

// HTTPConfig configures the Streamable HTTP MCP transport for SSF.
type HTTPConfig struct {
	// Path is the MCP endpoint path (default "/mcp").
	Path string
	// Metric is the base controller metric name (suffix " - mcp" is applied automatically).
	Metric string
	// Provider handles tools/list and tools/call.
	Provider MCPProvider
	// AuthFunc authenticates MCP requests when IsSecured is true.
	AuthFunc func(ctx *server.Context) error
	// IsSecured enables AuthFunc on the MCP endpoint.
	IsSecured bool
	// Server identifies this MCP server in initialize responses.
	Server ServerConfig
	// ProtocolVersion is the default MCP protocol version (default "2025-06-18").
	ProtocolVersion string
	// RequireSession requires a valid Mcp-Session-Id header for tools/list and tools/call.
	RequireSession bool
	// EnforceProtocolHeader rejects requests when MCP-Protocol-Version header is missing.
	EnforceProtocolHeader bool
}

// HTTPProvider is an SSF ControllerProvider for Streamable HTTP MCP.
type HTTPProvider struct {
	cfg        HTTPConfig
	dispatcher *Dispatcher
}

// NewHTTPProvider creates a ControllerProvider that exposes MCP over Streamable HTTP.
func NewHTTPProvider(cfg HTTPConfig) HTTPProvider {
	if cfg.Path == "" {
		cfg.Path = "/mcp"
	}
	if cfg.Metric == "" {
		cfg.Metric = "MCPJsonRpcController"
	}
	if cfg.ProtocolVersion == "" {
		cfg.ProtocolVersion = DefaultProtocolVersion
	}
	dispatcher := NewDispatcher(DispatcherConfig{
		Provider:        cfg.Provider,
		Server:          cfg.Server,
		ProtocolVersion: cfg.ProtocolVersion,
		RequireSession:  cfg.RequireSession,
	})
	return HTTPProvider{cfg: cfg, dispatcher: dispatcher}
}

// GetControllers returns SSF controllers for the MCP HTTP transport.
func (p HTTPProvider) GetControllers() []server.Controller {
	controllers := []server.Controller{
		{
			Name:           "MCPJsonRpcController",
			Description:    "MCP Streamable HTTP JSON-RPC endpoint",
			Metric:         MCPMetric(p.cfg.Metric),
			Methods:        []string{http.MethodPost},
			Path:           p.cfg.Path,
			IsSecured:      p.cfg.IsSecured,
			AuthFunc:       p.cfg.AuthFunc,
			ControllerFunc: p.jsonRpcController,
		},
		{
			Name:           "MCPSSEController",
			Description:    "MCP SSE listener (not implemented)",
			Metric:         MCPMetric("MCPSSEController"),
			Methods:        []string{http.MethodGet},
			Path:           p.cfg.Path,
			IsSecured:      p.cfg.IsSecured,
			AuthFunc:       p.cfg.AuthFunc,
			ControllerFunc: p.sseNotSupportedController,
		},
	}
	return controllers
}

func (p HTTPProvider) jsonRpcController(ctx *server.Context) {
	if err := validateStreamableHTTPRequest(ctx, p.cfg); err != nil {
		if headerErr, ok := err.(*headerMismatchError); ok {
			ctx.SendJSONRPCError(nil, headerErr.Code, headerErr.Message)
			return
		}
		ctx.SendJsonError(server.JSONErrorResponse{
			Code:       http.StatusBadRequest,
			Message:    "bad_request",
			LogMessage: err.Error(),
		})
		return
	}

	body, err := ctx.GetRequestBody()
	if err != nil {
		ctx.SendJSONRPCError(nil, ErrCodeParseError, "Parse error")
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		ctx.SendJSONRPCError(nil, ErrCodeParseError, "Parse error")
		return
	}

	p.dispatcher.Handle(ctx, req)
}

func (p HTTPProvider) sseNotSupportedController(ctx *server.Context) {
	ctx.SendGenericResponse(http.StatusMethodNotAllowed, []byte("SSE stream not supported"), "text/plain")
}

func validateStreamableHTTPRequest(ctx *server.Context, cfg HTTPConfig) error {
	accept := ctx.Request.Header.Get(HeaderAccept)
	if accept != "" && !acceptsStreamableHTTP(accept) {
		return fmt.Errorf("Accept header must include application/json")
	}

	headerVersion := ctx.Request.Header.Get(HeaderProtocolVersion)
	if cfg.EnforceProtocolHeader && headerVersion == "" {
		return &headerMismatchError{
			Code:    ErrCodeInvalidRequest,
			Message: "Missing MCP-Protocol-Version header",
		}
	}
	if headerVersion != "" && headerVersion != cfg.ProtocolVersion {
		return &headerMismatchError{
			Code:    ErrCodeInvalidRequest,
			Message: fmt.Sprintf("MCP-Protocol-Version mismatch: header %q, expected %q", headerVersion, cfg.ProtocolVersion),
		}
	}
	return nil
}

func acceptsStreamableHTTP(accept string) bool {
	return containsMIME(accept, "application/json") || containsMIME(accept, "text/event-stream") || accept == "*/*"
}

func containsMIME(accept, mime string) bool {
	for i := 0; i <= len(accept)-len(mime); i++ {
		if accept[i:i+len(mime)] == mime {
			return true
		}
	}
	return false
}

type headerMismatchError struct {
	Code    int
	Message string
}

func (e *headerMismatchError) Error() string {
	return e.Message
}
