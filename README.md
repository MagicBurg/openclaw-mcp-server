# OpenClaw MCP Server

A Go MCP (Model Context Protocol) server that exposes OpenClaw's skills, tools, chat, and cron capabilities. Designed to orchestrate multiple OpenClaw instances as workers for an upstream agentic platform.

## Architecture

```
Agentic Platform (MCP Client)
    ↕ MCP Protocol (stdio / Streamable HTTP)
OpenClaw MCP Server (this project)
    ├── Instance Registry (worker management)
    ├── OpenClaw HTTP Client
    └── MCP Tool Handlers
    ↕ HTTP
OpenClaw Gateway(s) (workers)
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `openclaw_chat` | Send a message to an OpenClaw instance and get a response |
| `openclaw_tool_invoke` | Invoke any OpenClaw tool directly (browser, memory, web_fetch, etc.) |
| `openclaw_cron` | Manage cron jobs (list, add, update, remove, run, status) |
| `openclaw_status` | Check health status of a worker instance |
| `openclaw_instances` | List all configured worker instances |

## Quick Start

### Prerequisites

- Go 1.24+
- One or more running OpenClaw gateway instances

### Build

```bash
go build -o openclaw-mcp-server ./cmd/server/
```

### Run (stdio mode)

```bash
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-gateway-token \
./openclaw-mcp-server
```

### Run (HTTP mode)

```bash
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-gateway-token \
MCP_AUTH_TOKEN=your-mcp-auth-token \
./openclaw-mcp-server --transport http --port 8080
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENCLAW_URL` | Single instance gateway URL | `http://127.0.0.1:18789` |
| `OPENCLAW_TOKEN` | Bearer token for gateway auth | _(none)_ |
| `OPENCLAW_TIMEOUT` | Request timeout (Go duration) | `120s` |
| `OPENCLAW_INSTANCES` | JSON array of instance configs (overrides single-instance vars) | _(none)_ |
| `MCP_TRANSPORT` | Transport mode: `stdio` or `http` | `stdio` |
| `MCP_PORT` | HTTP server port | `8080` |
| `MCP_HOST` | HTTP server bind address | `0.0.0.0` |
| `MCP_AUTH_TOKEN` | Bearer token for MCP client auth (HTTP mode) | _(none)_ |

### CLI Flags

```
--transport, -t    Transport mode: stdio, http
--port, -p         HTTP port
--host             HTTP bind address
--openclaw-url     Single instance URL
--openclaw-token   Single instance token
--auth-token       Bearer token for MCP client auth
```

CLI flags override environment variables.

### Multi-Instance (Workers)

To connect to multiple OpenClaw instances, set `OPENCLAW_INSTANCES` as a JSON array:

```bash
OPENCLAW_INSTANCES='[
  {"name": "worker-1", "url": "http://10.0.0.1:18789", "token": "sk-1", "default": true},
  {"name": "worker-2", "url": "http://10.0.0.2:18789", "token": "sk-2", "timeout": "60s"}
]' ./openclaw-mcp-server
```

Each tool accepts an optional `instance` parameter to target a specific worker. If omitted, the default instance is used.

## Development

### Run Tests

```bash
go test ./...
```

### Project Structure

```
cmd/server/         Entry point
internal/
  config/           Configuration loading
  openclaw/         HTTP client + instance registry
  tools/            MCP tool handlers
  server/           Server creation and wiring
docs/               Research and documentation
plan/               Implementation plans
```

## License

See [LICENSE](LICENSE).
