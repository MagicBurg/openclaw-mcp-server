# Development

## Prerequisites

- **Go 1.24+** — the project uses `go 1.24.0` in `go.mod`. If your installed version is older, set `GOTOOLCHAIN=auto` and Go will download the right toolchain automatically.

## Project Structure

```
openclaw-mcp-server/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, CLI flags, transport selection
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration loading (env vars, validation)
│   │   └── config_test.go       # 14 tests
│   ├── openclaw/
│   │   ├── client.go            # HTTP client for OpenClaw gateway
│   │   ├── client_test.go       # 12 tests
│   │   ├── registry.go          # Multi-instance worker registry
│   │   ├── registry_test.go     # 9 tests
│   │   └── types.go             # Request/response types
│   ├── tools/
│   │   ├── chat.go              # openclaw_chat tool handler
│   │   ├── invoke.go            # openclaw_tool_invoke tool handler
│   │   ├── cron.go              # openclaw_cron tool handler
│   │   ├── status.go            # openclaw_status tool handler
│   │   ├── instances.go         # openclaw_instances tool handler
│   │   ├── register.go          # Tool registration
│   │   └── tools_test.go        # 15 tests
│   └── server/
│       ├── server.go            # MCP server creation
│       └── server_test.go       # 1 test
├── docs/                        # Documentation
├── plan/                        # Implementation plans
├── go.mod
├── go.sum
├── .env.example
├── CLAUDE.md                    # AI assistant instructions
├── TODO.md                      # Project roadmap
├── README.md
└── LICENSE
```

## Building

```bash
go build -o openclaw-mcp-server ./cmd/server/
```

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

Run tests for a specific package:

```bash
go test ./internal/tools/ -v
```

### Test Coverage

The project has 42 tests covering:

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/config` | 14 | Env loading, single/multi instance, validation errors, edge cases |
| `internal/openclaw` | 12 + 9 | HTTP client (mock server), registry resolution, token isolation |
| `internal/tools` | 15 | All 5 tool handlers (happy path, errors, unknown instances), registration |
| `internal/server` | 1 | Server creation with tool registration |

Tests use `httptest.NewServer` for HTTP client testing — no external dependencies needed.

## Adding a New MCP Tool

1. **Create the handler file** in `internal/tools/` (e.g., `sessions.go`):

```go
package tools

import (
    "context"
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

type SessionsInput struct {
    Instance string `json:"instance,omitempty" jsonschema:"Target worker instance"`
}

type SessionsOutput struct {
    Sessions []any  `json:"sessions"`
    Instance string `json:"instance"`
}

func SessionsHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input SessionsInput) (*mcp.CallToolResult, SessionsOutput, error) {
    return func(ctx context.Context, req *mcp.CallToolRequest, input SessionsInput) (*mcp.CallToolResult, SessionsOutput, error) {
        inst, err := registry.Resolve(input.Instance)
        if err != nil {
            return nil, SessionsOutput{}, err
        }
        // Call gateway, build output...
        return nil, SessionsOutput{Instance: inst.Name}, nil
    }
}
```

2. **Register it** in `register.go`:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "openclaw_sessions",
    Description: "List sessions on an OpenClaw instance.",
}, SessionsHandler(registry))
```

3. **Add tests** in `tools_test.go` or a new test file.

4. **Add a new client method** if needed in `internal/openclaw/client.go`.

## Adding a New Gateway Client Method

If a new tool needs a gateway endpoint not yet covered by the client:

1. Add the method to `Client` in `internal/openclaw/client.go`:

```go
func (c *Client) NewEndpoint(ctx context.Context, params SomeParams) (*SomeResult, error) {
    body, err := c.doJSON(ctx, http.MethodPost, "/new/endpoint", params, "")
    if err != nil {
        return nil, err
    }
    var result SomeResult
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    return &result, nil
}
```

2. Add types to `internal/openclaw/types.go`.

3. Add tests using `httptest.NewServer` in `client_test.go`.

## Code Style

- Standard Go conventions (`gofmt`, `go vet`)
- No external linter configured yet — follow existing patterns
- Keep error handling explicit (no panics in production code)
- Use `jsonschema:"description text"` struct tags for MCP tool schema descriptions
- Fields without `omitempty` in the `json` tag are treated as required by the MCP SDK

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/modelcontextprotocol/go-sdk` | v1.3.1 | Official MCP protocol implementation |

The project intentionally keeps dependencies minimal.
