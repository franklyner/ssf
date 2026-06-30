package mcp

import "github.com/franklyner/ssf/server"

// MCPProvider supplies MCP tools and handles tool execution.
type MCPProvider interface {
	ListTools(ctx *server.Context, cursor string) ([]Tool, string)
	CallTool(ctx *server.Context, name string, arguments map[string]interface{}) (CallToolResult, error)
}

// ProviderFunc adapts functions to MCPProvider.
type ProviderFunc struct {
	ListToolsFunc func(ctx *server.Context, cursor string) ([]Tool, string)
	CallToolFunc  func(ctx *server.Context, name string, arguments map[string]interface{}) (CallToolResult, error)
}

func (p ProviderFunc) ListTools(ctx *server.Context, cursor string) ([]Tool, string) {
	return p.ListToolsFunc(ctx, cursor)
}

func (p ProviderFunc) CallTool(ctx *server.Context, name string, arguments map[string]interface{}) (CallToolResult, error) {
	return p.CallToolFunc(ctx, name, arguments)
}
