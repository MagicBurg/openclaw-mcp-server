# Plan 001 — OpenClaw MCP Server Foundation

## Goal

Build a Go MCP server that exposes OpenClaw's skills, tools, chat, and cron capabilities to an upstream agentic platform. The server treats multiple OpenClaw instances as workers and routes requests accordingly.

## Architecture

```
Agentic Platform (MCP Client)
    ↕ MCP Protocol (stdio / Streamable HTTP)
OpenClaw MCP Server (Go)
    ├── Instance Registry (worker management)
    ├── OpenClaw HTTP Client
    ├── MCP Tool Handlers
    └── Auth Middleware (bearer token)
    ↕ HTTP
OpenClaw Gateway(s) (workers)
    ├── POST /v1/chat/completions
    ├── POST /tools/invoke
    └── GET /health
```

### Key Design Decisions

1. **HTTP-only for v1** — all gateway communication via HTTP. No WebSocket client yet. Cron is accessed via `POST /tools/invoke` with `tool: "cron"`.
2. **Official Go MCP SDK** — `github.com/modelcontextprotocol/go-sdk` for type-safe tool binding, built-in auth, and transport support.
3. **Multi-instance workers** — registry manages named OpenClaw instances with independent URLs, tokens, and timeouts.
4. **Extensible tool architecture** — adding a new MCP tool should require only: (a) define input/output structs, (b) write handler function, (c) register with `mcp.AddTool`.
5. **Transports: stdio + Streamable HTTP** — stdio for local dev/testing, Streamable HTTP for production.

### Project Layout

```
openclaw-mcp-server/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, CLI flags, transport selection
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading (env, flags, TOML)
│   ├── openclaw/
│   │   ├── client.go            # HTTP client for OpenClaw gateway
│   │   ├── client_test.go
│   │   ├── registry.go          # Multi-instance worker registry
│   │   ├── registry_test.go
│   │   └── types.go             # Shared types (responses, configs)
│   ├── tools/
│   │   ├── chat.go              # openclaw_chat tool
│   │   ├── chat_test.go
│   │   ├── invoke.go            # openclaw_tool_invoke tool
│   │   ├── invoke_test.go
│   │   ├── cron.go              # openclaw_cron tool
│   │   ├── cron_test.go
│   │   ├── status.go            # openclaw_status tool
│   │   ├── status_test.go
│   │   ├── instances.go         # openclaw_instances tool
│   │   ├── instances_test.go
│   │   └── register.go          # Tool registration helper
│   └── server/
│       ├── server.go            # MCP server creation and wiring
│       └── server_test.go
├── docs/
│   └── research.md
├── plan/
├── go.mod
├── go.sum
├── CLAUDE.md
├── TODO.md
└── README.md
```

---

## MCP Tools (v1)

### 1. `openclaw_chat`
Send a message to an OpenClaw instance and get a response.

**Input:**
```go
type ChatInput struct {
    Message   string `json:"message"   jsonschema:"required,the message to send"`
    SessionID string `json:"session_id" jsonschema:"optional session ID for conversation continuity"`
    Instance  string `json:"instance"   jsonschema:"optional worker instance name"`
}
```

**Output:** Assistant response text, model, token usage.

**Gateway call:** `POST /v1/chat/completions`

---

### 2. `openclaw_tool_invoke`
Invoke any OpenClaw tool directly on a worker instance.

**Input:**
```go
type ToolInvokeInput struct {
    Tool       string         `json:"tool"        jsonschema:"required,tool name (e.g. browser, memory_search, web_fetch)"`
    Action     string         `json:"action"      jsonschema:"optional action for the tool"`
    Args       map[string]any `json:"args"        jsonschema:"tool arguments"`
    SessionKey string         `json:"session_key"  jsonschema:"optional session context"`
    Instance   string         `json:"instance"     jsonschema:"optional worker instance name"`
}
```

**Output:** Tool result (ok/error + result payload).

**Gateway call:** `POST /tools/invoke`

---

### 3. `openclaw_cron`
Manage cron jobs on an OpenClaw instance (list, add, update, remove, run, status).

**Input:**
```go
type CronInput struct {
    Action   string         `json:"action"   jsonschema:"required,one of: status, list, add, update, remove, run"`
    Job      map[string]any `json:"job"      jsonschema:"job definition for add action"`
    JobID    string         `json:"job_id"   jsonschema:"job ID for update/remove/run actions"`
    Patch    map[string]any `json:"patch"    jsonschema:"patch fields for update action"`
    Instance string         `json:"instance" jsonschema:"optional worker instance name"`
}
```

**Output:** Cron operation result.

**Gateway call:** `POST /tools/invoke` with `tool: "cron"`

---

### 4. `openclaw_status`
Health check for a worker instance.

**Input:**
```go
type StatusInput struct {
    Instance string `json:"instance" jsonschema:"optional worker instance name"`
}
```

**Output:** `{ status: "ok"|"error", instance: "name" }`

**Gateway call:** `GET /health`

---

### 5. `openclaw_instances`
List all configured worker instances.

**Input:** (none)

**Output:** List of instances with name, URL, default flag. Never exposes tokens.

**Gateway call:** None (local registry).

---

## Phases

### Phase 1 — Project Bootstrap
- Initialize Go module (`go mod init`)
- Add dependencies: official MCP SDK
- Create project structure (`cmd/`, `internal/`)
- Create `cmd/server/main.go` with minimal stdio server
- Create `internal/config/config.go` for configuration loading

**Testing plan:**
- `config_test.go`: parse single instance, multi-instance JSON, env vars, validation errors

---

### Phase 2 — OpenClaw Client & Registry
- Implement `internal/openclaw/client.go`: HTTP client with `Chat()`, `Health()`, `ToolInvoke()` methods
- Implement `internal/openclaw/types.go`: request/response types
- Implement `internal/openclaw/registry.go`: multi-instance management with validation

**Testing plan:**
- `client_test.go`: mock HTTP server, test Chat/Health/ToolInvoke happy path, timeout, error responses, auth header, session header
- `registry_test.go`: single/multi instance, default resolution, unknown instance error, duplicate names, URL validation, token isolation

---

### Phase 3 — MCP Tool Handlers
- Implement all 5 tool handlers (`chat.go`, `invoke.go`, `cron.go`, `status.go`, `instances.go`)
- Implement `register.go` to wire all tools to the MCP server
- Each handler: validate input, resolve instance, call client, format response

**Testing plan:**
- Per-tool test file: happy path, missing required fields, unknown instance, gateway errors, response formatting
- `register_test.go`: verify all tools are registered with correct names/schemas

---

### Phase 4 — Server Wiring & Transports
- Wire everything together in `internal/server/server.go`
- Add stdio transport in `cmd/server/main.go`
- Add Streamable HTTP transport (flag-controlled)
- Add bearer token auth middleware for HTTP transport
- CLI flags: `--transport`, `--port`, `--host`, `--auth-token`
- Env vars: `OPENCLAW_URL`, `OPENCLAW_TOKEN`, `OPENCLAW_INSTANCES`, `MCP_AUTH_TOKEN`

**Testing plan:**
- `server_test.go`: server creation, tool registration count, in-memory transport integration test
- Manual test: run with stdio, connect with MCP inspector

---

### Phase 5 — Documentation & Polish
- Write `README.md` with setup, configuration, usage examples
- Update `docs/` with architecture overview
- Add `.env.example`
- Add `Makefile` or build commands to CLAUDE.md

**Testing plan:**
- Run full test suite
- Verify README instructions work

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENCLAW_URL` | Single instance gateway URL | `http://127.0.0.1:18789` |
| `OPENCLAW_TOKEN` | Bearer token for gateway | (none) |
| `OPENCLAW_TIMEOUT` | Request timeout (e.g., `120s`) | `120s` |
| `OPENCLAW_INSTANCES` | JSON array of instance configs | (none, overrides single-instance) |
| `MCP_TRANSPORT` | `stdio` or `http` | `stdio` |
| `MCP_PORT` | HTTP server port | `8080` |
| `MCP_HOST` | HTTP server bind address | `0.0.0.0` |
| `MCP_AUTH_TOKEN` | Bearer token for MCP client auth | (none) |

### Instance Config (JSON)

```json
[
  { "name": "worker-1", "url": "http://10.0.0.1:18789", "token": "sk-...", "default": true },
  { "name": "worker-2", "url": "http://10.0.0.2:18789", "token": "sk-..." }
]
```

### CLI Flags

```
--transport, -t    Transport mode: stdio, http (default: stdio)
--port, -p         HTTP port (default: 8080)
--host             HTTP bind address (default: 0.0.0.0)
--openclaw-url     Single instance URL
--openclaw-token   Single instance token
--timeout          Request timeout (default: 120s)
--auth-token       Bearer token for MCP client auth (HTTP mode)
```

---

## Post-Execution Report

**Completed:** 2026-03-23

### Implementation Notes

- **Go MCP SDK v1.3.1** used (not v1.4.1 which requires Go 1.25.0 stable). The `go.mod` targets Go 1.24.0 due to transitive dependency `golang.org/x/oauth2`. Go toolchain auto-download handles this via `GOTOOLCHAIN=auto`.
- **jsonschema tags:** The SDK's `jsonschema` struct tag is description-only (no `required` or `description=` prefixes). Required fields are inferred from `json` tags without `omitempty`.
- **Content type:** SDK v1.3.1 uses `[]mcp.Content` (interface) not `[]*mcp.Content`, and has no `NewTextContent` helper. Used `&mcp.TextContent{Text: ...}` directly.
- **Streamable HTTP** transport used for HTTP mode (not legacy SSE).

### Test Results

- 42 tests across 4 packages, all passing
- `internal/config`: 14 tests (env loading, validation, edge cases)
- `internal/openclaw`: 12 tests (client HTTP calls, registry resolution)
- `internal/tools`: 15 tests (all 5 tool handlers, registration)
- `internal/server`: 1 test (server creation)

### Deviations from Plan

- No `Makefile` added — `go build` and `go test` are sufficient for now.
- No `docs/architecture.md` created — README covers the architecture overview, and `docs/research.md` has the detailed reference.

### Known Limitations

- No streaming support for chat (non-streaming only in v1)
- No WebSocket client for gateway — all communication is HTTP
- Bearer token auth is simple static comparison (no OAuth 2.1 yet)
- No async task management (unlike the TypeScript `openclaw-mcp`)
- Tool invoke response parsing is generic (`any`) — structured result types could be added per-tool later
