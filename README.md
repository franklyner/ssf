# ssf
Small Server Framework: Lightweight framework for building web and api servers with GO

## Goals
The main goal is as usual to avoid boiler plate code, enforce some standards and provide some functionalities out-of-the-box that just take care of a few things. Also, we want to promote a clean code structure.

## High-Level Approach
The most central concept is that the part of the code that instantiates the server (typically the place where your func main() lives) takes care of wiring up all dependencies. It aligns itself to the concept of an application controller.

The we introduce 3 main parts:
### Controller
A controller is the code that actually handles requests for a specific route. It declares all of the required properties and provide an auth funtion pointer if secured and a controller function pointer.. the code that actually is executed for the request.

Typically you need a bunch of related controllers (such as one for each CRUD operatio for a rest API). To simplify things for the application controller, all related controllers are registered to the server through a ControllerProvider. This allows to nicely organize the different aspects of an application into differen files and provide one controller provider for each file and the controllers therein. This promotes in our opinion a clean code structure.

### Context
For each request a context is instantiated. The context provides access to the request and provides all required functions to send back the response. It also provides access to the service registry.

### Service
A service is nothing else than a struct instance the provides some functions. It is registered to the server by the application controller. On each request the service registry is added to the context and passed down to the controllers for execution. This allows all controllers and inter-dependent services to access other services as long as they get access to the context.

## Configuration Concept
The framework provides a Config type. It's basically a simple map but can optionally use it's pre-implemented viper instanciation. The main guideline is that we want the server to fail upon startup in case of missing configuration. To ensure that each required config property needs to be read by the application controller. The Config functions will immediately panic in case one is missign.

This is the reason why the config object is not passed along with the context. Instead it is expected that each ControllerProvider and Service struct exports fiels for each config property required. It is the application controllers job to populate those properties either through the help of the Config object or by other means. 

Controller functions get access to those by exposing a Config property. The controller provider is responsible to instantiate the config map for each controller (if required) and add it to the controller as part of the getControllers() function.

## Free stuff
Some things come for free:
* Automatic logging of all requests, the corresponding response and measurment of the execution duration.
* Each request gets it's unique UUID. All messages logged through the functions provided by the context will be prefixed with the request id for correlation.
* Some easy to use methods to send html and json responses
* Easy testability: Ther server exposes a GetMainHandler() function that gives access to the main request handler which can then be used for unit testing.
* A status page that gives an overview of how many times each controller has been called and since when the server is running. Controllers and other code can also publish non-controller int metrics via `StatusInformation.SetMetric` / `IncrementMetric`, and string status values via `SetInfo` (shown in the same Non Controller Metrics table).

## New Relic (optional)
- Initialize your New Relic app in the caller and inject it after `CreateServer` and before `Start` using `SetNewRelicApp(app)`.
- When set, each controller invocation runs inside a NR web transaction and MySQL calls (via the repository) are instrumented through `nrmysql` automatically.
- If you do not set the app, nothing changes for existing users.

## MCP (Model Context Protocol)

The `mcp` package (`github.com/franklyner/ssf/mcp`) adds MCP support on top of the existing SSF controller model. It uses **Streamable HTTP** transport: JSON-RPC 2.0 messages sent via `POST`, with optional `GET` on the same path (returns `405` until SSE is implemented).

MCP registers like any other `ControllerProvider` — no changes to the core server lifecycle.

### Quick start

Implement `MCPProvider` (or use `ProviderFunc`) and register `NewHTTPProvider` alongside your REST controllers:

```go
import "github.com/franklyner/ssf/mcp"

provider := mcp.ProviderFunc{
    ListToolsFunc: func(ctx *server.Context, cursor string) ([]mcp.Tool, string) {
        return []mcp.Tool{
            {Name: "echo", Description: "Echoes input", InputSchema: map[string]string{"type": "object"}},
        }, ""
    },
    CallToolFunc: func(ctx *server.Context, name string, args map[string]interface{}) (mcp.CallToolResult, error) {
        // dispatch to your services
        return mcp.CallToolResult{
            Content: []mcp.ContentItem{{Type: "text", Text: `{"ok":true}`}},
        }, nil
    },
}

ctrProviders := []server.ControllerProvider{
    api.UserControllerProvider{},
    mcp.NewHTTPProvider(mcp.HTTPConfig{
        Path:      "/mcp",
        Provider:  provider,
        IsSecured: true,
        AuthFunc:  myAuth,
        Server:    mcp.ServerConfig{Name: "my-api", Version: "1.0.0"},
    }),
}

srv := server.CreateServerWithPrefix(cfg, ctrProviders, "/v1")
```

This exposes `POST /v1/mcp` (JSON-RPC) and `GET /v1/mcp` (405 until SSE is added).

### MCPProvider

```go
type MCPProvider interface {
    ListTools(ctx *server.Context, cursor string) ([]Tool, string)
    CallTool(ctx *server.Context, name string, arguments map[string]interface{}) (CallToolResult, error)
}
```

`ProviderFunc` adapts plain functions to this interface when you don't need a struct.

### HTTPConfig options

| Field | Default | Description |
|---|---|---|
| `Path` | `/mcp` | MCP endpoint path |
| `Metric` | `MCPJsonRpcController` | Base metric name (` - mcp` suffix applied automatically) |
| `Provider` | — | Tool list and execution |
| `AuthFunc` / `IsSecured` | unsecured | Same pattern as REST controllers |
| `Server` | `ssf-mcp` / `1.0.0` | Name and version in `initialize` response |
| `ProtocolVersion` | `2025-06-18` | MCP protocol version |
| `RequireSession` | `false` | When `true`, `tools/list` and `tools/call` require `Mcp-Session-Id` from `initialize` |
| `EnforceProtocolHeader` | `false` | When `true`, reject requests without `MCP-Protocol-Version` header |

### Supported JSON-RPC methods

| Method | Type | Description |
|---|---|---|
| `initialize` | request | Handshake; returns capabilities and sets `Mcp-Session-Id` response header |
| `notifications/initialized` | notification | Client ack after initialize; HTTP 202, no JSON-RPC body |
| `tools/list` | request | Lists tools from `MCPProvider.ListTools` |
| `tools/call` | request | Executes a tool via `MCPProvider.CallTool` |

Clients should send `Accept: application/json, text/event-stream` and may include `MCP-Protocol-Version`.

### Metrics

MCP metrics are suffixed with ` - mcp` so they are distinguishable from REST controller metrics on `/status`:

- `MCPJsonRpcController - mcp` — transport endpoint
- `initialize - mcp`, `tools/list - mcp`, `tools/call - mcp` — per-method counters
- Per-tool metrics (when using controller dispatch in a later release) — `{Controller.Metric} - mcp`

Use `mcp.MCPMetric("myMetric")` when incrementing custom MCP counters.

### JSON-RPC response helpers

The `server` package provides helpers on `Context` for JSON-RPC responses:

```go
ctx.SendJSONRPCResult(id, result)
ctx.SendJSONRPCError(id, -32602, "Invalid params")
```

### Testing

Use `srv.GetMainHandler()` with `httptest`, same as REST controllers. See `mcp/mcp_test.go` for examples covering `initialize`, `tools/list`, `tools/call`, and session enforcement.
