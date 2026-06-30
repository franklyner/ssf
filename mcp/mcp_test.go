package mcp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franklyner/ssf/mcp"
	"github.com/franklyner/ssf/server"
)

func newTestServer(provider mcp.MCPProvider) *server.Server {
	cfg := server.CreateConfig("../cmd/minimal", "minimal", []string{
		server.ConfigPort,
		server.ConfigReadTimeout,
		server.ConfigWriteTimeout,
		server.ConfigLogLevel,
		server.ConfigEnablePrometheus,
		server.ConfigEnableOpenAPI,
	})
	cfg.SetProperty(server.ConfigEnablePrometheus, "false")
	cfg.SetProperty(server.ConfigEnableOpenAPI, "false")

	httpProvider := mcp.NewHTTPProvider(mcp.HTTPConfig{
		Path:     "/mcp",
		Provider: provider,
		Server: mcp.ServerConfig{
			Name:    "test-mcp",
			Version: "0.0.1",
		},
	})

	return server.CreateServer(cfg, []server.ControllerProvider{httpProvider})
}

type stubProvider struct {
	tools []mcp.Tool
}

func (s stubProvider) ListTools(_ *server.Context, _ string) ([]mcp.Tool, string) {
	return s.tools, ""
}

func (s stubProvider) CallTool(_ *server.Context, name string, _ map[string]interface{}) (mcp.CallToolResult, error) {
	return mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: "called:" + name}},
	}, nil
}

func postMCP(t *testing.T, handler http.Handler, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.HeaderProtocolVersion, mcp.DefaultProtocolVersion)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSONRPC(t *testing.T, rec *httptest.ResponseRecorder) server.JSONRPCResponse {
	t.Helper()
	respBody, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var resp server.JSONRPCResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", string(respBody), err)
	}
	return resp
}

func TestInitialize(t *testing.T) {
	srv := newTestServer(stubProvider{})
	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": mcp.DefaultProtocolVersion,
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get(mcp.HeaderSessionID) == "" {
		t.Fatal("expected Mcp-Session-Id header")
	}

	resp := decodeJSONRPC(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result object, got %T", resp.Result)
	}
	if result["protocolVersion"] != mcp.DefaultProtocolVersion {
		t.Fatalf("unexpected protocolVersion: %v", result["protocolVersion"])
	}
}

func TestToolsList(t *testing.T) {
	srv := newTestServer(stubProvider{
		tools: []mcp.Tool{{Name: "echo", Description: "echo tool"}},
	})
	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := decodeJSONRPC(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

func TestToolsCall(t *testing.T) {
	srv := newTestServer(stubProvider{})
	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "echo",
			"arguments": map[string]interface{}{"msg": "hi"},
		},
	})

	resp := decodeJSONRPC(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

func TestMethodNotFound(t *testing.T) {
	srv := newTestServer(stubProvider{})
	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "unknown/method",
	})

	resp := decodeJSONRPC(t, rec)
	if resp.Error == nil || resp.Error.Code != mcp.ErrCodeMethodNotFound {
		t.Fatalf("expected method not found, got %+v", resp.Error)
	}
}

func TestInitializedNotification(t *testing.T) {
	srv := newTestServer(stubProvider{})
	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
}

func TestSSEEndpointNotSupported(t *testing.T) {
	srv := newTestServer(stubProvider{})
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()
	srv.GetMainHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestMCPMetricSuffix(t *testing.T) {
	if got := mcp.MCPMetric("initialize"); got != "initialize - mcp" {
		t.Fatalf("unexpected metric: %s", got)
	}
}

func TestRequireSession(t *testing.T) {
	cfg := server.CreateConfig("../cmd/minimal", "minimal", []string{
		server.ConfigPort,
		server.ConfigReadTimeout,
		server.ConfigWriteTimeout,
		server.ConfigLogLevel,
		server.ConfigEnablePrometheus,
		server.ConfigEnableOpenAPI,
	})
	cfg.SetProperty(server.ConfigEnablePrometheus, "false")
	cfg.SetProperty(server.ConfigEnableOpenAPI, "false")

	httpProvider := mcp.NewHTTPProvider(mcp.HTTPConfig{
		Path:           "/mcp",
		Provider:       stubProvider{},
		RequireSession: true,
	})
	srv := server.CreateServer(cfg, []server.ControllerProvider{httpProvider})

	rec := postMCP(t, srv.GetMainHandler(), map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/list",
	})

	resp := decodeJSONRPC(t, rec)
	if resp.Error == nil {
		t.Fatal("expected session error")
	}
}
