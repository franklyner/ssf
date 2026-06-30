package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/franklyner/ssf/server"
)

// DispatcherConfig configures JSON-RPC method routing for MCP.
type DispatcherConfig struct {
	Provider        MCPProvider
	Sessions        *SessionStore
	Server          ServerConfig
	ProtocolVersion string
	RequireSession  bool
}

// Dispatcher routes MCP JSON-RPC methods.
type Dispatcher struct {
	cfg DispatcherConfig
}

// NewDispatcher creates a dispatcher with the given configuration.
func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	if cfg.Sessions == nil {
		cfg.Sessions = NewSessionStore()
	}
	if cfg.ProtocolVersion == "" {
		cfg.ProtocolVersion = DefaultProtocolVersion
	}
	if cfg.Server.Name == "" {
		cfg.Server.Name = "ssf-mcp"
	}
	if cfg.Server.Version == "" {
		cfg.Server.Version = "1.0.0"
	}
	return &Dispatcher{cfg: cfg}
}

// Handle processes a single JSON-RPC request. Returns true if a JSON-RPC response was sent.
func (d *Dispatcher) Handle(ctx *server.Context, req JSONRPCRequest) bool {
	if req.JSONRPC != "2.0" {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidRequest, "Invalid Request: jsonrpc must be 2.0")
		return true
	}
	if req.Method == "" {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidRequest, "Invalid Request: method is required")
		return true
	}

	if req.IsNotification() {
		return d.handleNotification(ctx, req)
	}

	ctx.StatusInformation.IncrementMetric(MCPMetric(req.Method))

	switch req.Method {
	case "initialize":
		d.handleInitialize(ctx, req)
	case "tools/list":
		d.handleToolsList(ctx, req)
	case "tools/call":
		d.handleToolsCall(ctx, req)
	default:
		ctx.SendJSONRPCError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
	return true
}

func (d *Dispatcher) handleNotification(ctx *server.Context, req JSONRPCRequest) bool {
	ctx.StatusInformation.IncrementMetric(MCPMetric(req.Method))

	switch req.Method {
	case "notifications/initialized":
		if d.cfg.RequireSession && !d.hasValidSession(ctx) {
			ctx.SendJsonError(server.JSONErrorResponse{
				Code:       http.StatusUnauthorized,
				Message:    "unauthorized",
				LogMessage: fmt.Sprintf("missing or invalid %s header", HeaderSessionID),
			})
			return false
		}
		ctx.SendGenericResponse(http.StatusAccepted, nil, "application/json")
		return false
	default:
		ctx.SendGenericResponse(http.StatusAccepted, nil, "application/json")
		return false
	}
}

func (d *Dispatcher) handleInitialize(ctx *server.Context, req JSONRPCRequest) {
	params, err := parseInitializeParams(req.Params)
	if err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	negotiated := negotiateProtocolVersion(params.ProtocolVersion, d.cfg.ProtocolVersion)
	session := d.cfg.Sessions.Create(negotiated, params.ClientInfo)
	ctx.SendResponseHeader(HeaderSessionID, session.ID)

	result := buildInitializeResult(d.cfg.Server, negotiated)
	ctx.SendJSONRPCResult(req.ID, result)
}

func (d *Dispatcher) handleToolsList(ctx *server.Context, req JSONRPCRequest) {
	if err := d.requireSessionIfConfigured(ctx); err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidRequest, err.Error())
		return
	}
	if d.cfg.Provider == nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInternalError, "MCP provider not configured")
		return
	}

	cursor := ""
	if req.Params != nil {
		if c, ok := req.Params["cursor"].(string); ok {
			cursor = c
		}
	}
	tools, nextCursor := d.cfg.Provider.ListTools(ctx, cursor)
	ctx.SendJSONRPCResult(req.ID, ListToolsResult{Tools: tools, NextCursor: nextCursor})
}

func (d *Dispatcher) handleToolsCall(ctx *server.Context, req JSONRPCRequest) {
	if err := d.requireSessionIfConfigured(ctx); err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidRequest, err.Error())
		return
	}
	if d.cfg.Provider == nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInternalError, "MCP provider not configured")
		return
	}
	if req.Params == nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, "Invalid params: name and arguments required")
		return
	}

	var params ToolsCallParams
	raw, err := json.Marshal(req.Params)
	if err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, "Invalid params")
		return
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, "Invalid params")
		return
	}
	if params.Name == "" {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, "Invalid params: name is required")
		return
	}

	result, err := d.cfg.Provider.CallTool(ctx, params.Name, params.Arguments)
	if err != nil {
		ctx.SendJSONRPCError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Tool call failed: %s", err.Error()))
		return
	}
	ctx.SendJSONRPCResult(req.ID, result)
}

func (d *Dispatcher) requireSessionIfConfigured(ctx *server.Context) error {
	if !d.cfg.RequireSession {
		return nil
	}
	if !d.hasValidSession(ctx) {
		return fmt.Errorf("valid %s header required", HeaderSessionID)
	}
	return nil
}

func (d *Dispatcher) hasValidSession(ctx *server.Context) bool {
	sessionID := ctx.Request.Header.Get(HeaderSessionID)
	if sessionID == "" {
		return false
	}
	return d.cfg.Sessions.Get(sessionID) != nil
}
