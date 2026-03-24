# Architecture

## System Overview

```
┌─────────────────────────────┐
│    Agentic Platform         │
│    (MCP Client)             │
└─────────────┬───────────────┘
              │ MCP Protocol
              │ (stdio or Streamable HTTP)
┌─────────────▼───────────────┐
│  OpenClaw MCP Server (Go)   │
│                             │
│  ┌────────────────────────┐ │
│  │    MCP Tool Handlers   │ │
│  │  chat, tool_invoke,    │ │
│  │  cron, status,         │ │
│  │  instances             │ │
│  └───────────┬────────────┘ │
│              │              │
│  ┌───────────▼────────────┐ │
│  │   Instance Registry    │ │
│  │  (worker management)   │ │
│  └───┬───────────────┬────┘ │
│      │               │      │
│  ┌───▼───┐       ┌───▼───┐  │
│  │Client │       │Client │  │
│  │(HTTP) │       │(HTTP) │  │
│  └───┬───┘       └───┬───┘  │
└──────┼───────────────┼──────┘
       │               │
       │ HTTP           │ HTTP
       │               │
┌──────▼──────┐  ┌─────▼──────┐
│  OpenClaw   │  │  OpenClaw  │
│  Gateway 1  │  │  Gateway 2 │
│  (worker)   │  │  (worker)  │
└─────────────┘  └────────────┘
```

## Components

### MCP Tool Handlers (`internal/tools/`)

Each MCP tool is a Go function that:
1. Receives typed input (auto-validated by the MCP SDK against the JSON Schema inferred from Go structs)
2. Resolves the target worker instance via the registry
3. Calls the OpenClaw gateway HTTP API
4. Returns a typed output (auto-serialized by the SDK)

Adding a new tool requires three steps:
1. Define `Input` and `Output` structs in a new file
2. Write the handler function
3. Register it in `register.go` with `mcp.AddTool`

### OpenClaw HTTP Client (`internal/openclaw/client.go`)

A thin HTTP client that communicates with OpenClaw gateways. Provides three methods:

| Method | Gateway Endpoint | Purpose |
|--------|-----------------|---------|
| `Chat()` | `POST /v1/chat/completions` | Send a message, get a response (OpenAI-compatible) |
| `ToolInvoke()` | `POST /tools/invoke` | Invoke any OpenClaw tool directly |
| `Health()` | `GET /health` | Check gateway liveness |

Features:
- Bearer token authentication (`Authorization: Bearer <token>`)
- Session key header (`x-openclaw-session-key`)
- Configurable timeout per instance
- Response size limit (10 MB)
- Error classification (4xx client errors vs 5xx server errors)

### Instance Registry (`internal/openclaw/registry.go`)

Manages multiple OpenClaw gateway instances as named workers.

- Each instance has a name, URL, optional token, optional timeout, and optional default flag
- `Resolve(name)` returns the client for the given instance, or the default if name is empty
- `List()` returns public metadata (name, URL, default flag) without exposing tokens
- Validation: instance names must be 1-64 alphanumeric chars (plus dashes/underscores), URLs must be http/https, no duplicate names, at most one default

### Configuration (`internal/config/`)

Configuration is loaded from environment variables and can be overridden by CLI flags.

**Precedence:** CLI flags > environment variables > defaults.

**Two modes:**
- **Single instance:** `OPENCLAW_URL` + `OPENCLAW_TOKEN` (simplest setup)
- **Multi-instance:** `OPENCLAW_INSTANCES` JSON array (overrides single-instance vars)

### Server (`internal/server/`, `cmd/server/`)

The `server.New()` function creates an MCP server with all tools registered. The `cmd/server/main.go` entry point handles transport selection:

- **stdio:** Reads from stdin, writes to stdout. Used with Claude Desktop and MCP Inspector.
- **Streamable HTTP:** Serves MCP over HTTP at `/mcp`. Supports optional bearer token auth. Health check at `/health`.

## Data Flow

### Chat Request

```
Client → openclaw_chat tool
       → Registry.Resolve(instance)
       → Client.Chat(message, sessionID)
       → POST /v1/chat/completions (OpenClaw gateway)
       ← ChatCompletionResponse (OpenAI format)
       ← ChatOutput (response text, model, usage, instance)
```

### Tool Invocation

```
Client → openclaw_tool_invoke tool
       → Registry.Resolve(instance)
       → Client.ToolInvoke(tool, action, args)
       → POST /tools/invoke (OpenClaw gateway)
       ← ToolInvokeResponse (ok, result/error)
       ← ToolInvokeOutput (ok, instance, result/error)
```

### Cron Management

```
Client → openclaw_cron tool
       → Registry.Resolve(instance)
       → Client.ToolInvoke(tool="cron", args={action, job, ...})
       → POST /tools/invoke (OpenClaw gateway, cron agent tool)
       ← ToolInvokeResponse
       ← CronOutput (ok, instance, result/error)
```

## Security

- **Gateway tokens** are stored in memory and never exposed via the `openclaw_instances` tool or logs.
- **URL validation** at startup rejects non-HTTP schemes (SSRF prevention).
- **Bearer token auth** for HTTP transport (optional but recommended for production).
- **Response size limits** prevent memory exhaustion from large gateway responses.
- **No secrets in config files** — all credentials come from environment variables.

## Extensibility

The architecture is designed to make adding new capabilities straightforward:

**New MCP tool:** Define input/output structs, write handler, register in `register.go`. No changes to the client, registry, or server needed.

**New gateway endpoint:** Add a method to `Client` in `client.go`, then use it from a tool handler.

**New transport:** The MCP SDK supports stdio, SSE, and Streamable HTTP. Transport selection is in `cmd/server/main.go`.

**WebSocket gateway (future):** For real-time operations like tool discovery (`tools.catalog`) and session streaming, a WebSocket client can be added to the `openclaw` package alongside the HTTP client.
