package mcp

import "encoding/json"

// JSON-RPC 2.0 standard error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

// Streamable HTTP header names.
const (
	HeaderProtocolVersion = "MCP-Protocol-Version"
	HeaderSessionID       = "Mcp-Session-Id"
	HeaderAccept          = "Accept"
)

// DefaultProtocolVersion is the MCP protocol version advertised by the server.
const DefaultProtocolVersion = "2025-06-18"

// JSONRPCRequest is a JSON-RPC 2.0 request or notification.
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	// idPresent is true when the JSON object included an "id" member (even if null).
	idPresent bool
}

// IsNotification reports whether the message is a JSON-RPC notification (no id member).
// Explicit "id": null is not a notification; use HasNullID and reject as invalid.
func (r JSONRPCRequest) IsNotification() bool {
	return !r.idPresent
}

// HasNullID reports whether the request included "id": null.
func (r JSONRPCRequest) HasNullID() bool {
	return r.idPresent && r.ID == nil
}

// parseJSONRPCRequest unmarshals a JSON-RPC message and records whether "id" was present.
func parseJSONRPCRequest(body []byte) (JSONRPCRequest, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return JSONRPCRequest{}, err
	}
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return JSONRPCRequest{}, err
	}
	_, req.idPresent = raw["id"]
	return req, nil
}

// Tool is an MCP tool definition.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// CallToolResult is the MCP tools/call response format.
type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

// ContentItem is a single content item in a tool result.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ListToolsResult is the MCP tools/list response.
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// InitializeParams is the MCP initialize request params.
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      *ClientInfo            `json:"clientInfo"`
}

// ClientInfo identifies the MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the MCP initialize response.
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

// ServerInfo identifies the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsCallParams is the MCP tools/call request params.
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
