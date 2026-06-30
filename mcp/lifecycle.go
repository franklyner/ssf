package mcp

import (
	"encoding/json"
	"fmt"
)

func parseInitializeParams(params map[string]interface{}) (InitializeParams, error) {
	if params == nil {
		return InitializeParams{}, fmt.Errorf("params required")
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return InitializeParams{}, err
	}
	var p InitializeParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return InitializeParams{}, err
	}
	if p.ProtocolVersion == "" {
		return InitializeParams{}, fmt.Errorf("protocolVersion is required")
	}
	return p, nil
}

func buildInitializeResult(cfg ServerConfig, negotiatedVersion string) InitializeResult {
	return InitializeResult{
		ProtocolVersion: negotiatedVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name:    cfg.Name,
			Version: cfg.Version,
		},
	}
}

// negotiateProtocolVersion picks the protocol version to use for the session.
func negotiateProtocolVersion(clientVersion, serverDefault string) string {
	if clientVersion != "" {
		return clientVersion
	}
	return serverDefault
}

// ServerConfig identifies the MCP server in initialize responses.
type ServerConfig struct {
	Name    string
	Version string
}
