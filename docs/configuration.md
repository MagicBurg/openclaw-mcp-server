# Configuration

Configuration is loaded from a TOML config file, environment variables, and CLI flags.

**Precedence:** CLI flags > environment variables > config file > defaults.

## Config File

The server searches for `config.toml` in the following locations (first found wins):

1. `./config.toml` (current directory)
2. `~/.config/openclaw-mcp-server/config.toml`

Or specify explicitly: `--config /path/to/config.toml`

### Example

```toml
[server]
transport = "http"
host = "127.0.0.1"
port = 8080
auth_token = "my-mcp-token"

[[instances]]
name = "worker-1"
url = "http://10.0.0.1:18789"
token = "sk-token-1"
default = true
timeout = "30s"

[[instances]]
name = "worker-2"
url = "http://10.0.0.2:18789"
token = "sk-token-2"
```

See `config.example.toml` in the project root for a fully commented template.

## Environment Variables

### OpenClaw Gateway

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENCLAW_URL` | Single instance gateway URL | `http://127.0.0.1:18789` |
| `OPENCLAW_TOKEN` | Bearer token for gateway authentication | _(none)_ |
| `OPENCLAW_TIMEOUT` | Request timeout as a Go duration (e.g., `30s`, `2m`) | `120s` |
| `OPENCLAW_INSTANCES` | JSON array of instance configs (see below). Overrides single-instance vars when set. | _(none)_ |

### MCP Server

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_TRANSPORT` | Transport mode: `stdio` or `http` | `stdio` |
| `MCP_PORT` | HTTP server port | `8080` |
| `MCP_HOST` | HTTP server bind address | `0.0.0.0` |
| `MCP_AUTH_TOKEN` | Bearer token required from MCP clients in HTTP mode | _(none)_ |

## CLI Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to config.toml |
| `--transport` | Transport mode: `stdio` or `http` |
| `--port` | HTTP server port |
| `--host` | HTTP server bind address |
| `--openclaw-url` | Single instance gateway URL |
| `--openclaw-token` | Single instance gateway token |
| `--auth-token` | Bearer token for MCP client auth (HTTP mode) |

## Single Instance

The simplest setup — one OpenClaw gateway:

```bash
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-gateway-token \
./openclaw-mcp-server
```

Or with CLI flags:

```bash
./openclaw-mcp-server \
  --openclaw-url http://127.0.0.1:18789 \
  --openclaw-token your-gateway-token
```

## Multi-Instance (Workers)

For multiple OpenClaw gateways, set `OPENCLAW_INSTANCES` as a JSON array:

```bash
OPENCLAW_INSTANCES='[
  {
    "name": "worker-1",
    "url": "http://10.0.0.1:18789",
    "token": "sk-token-1",
    "default": true
  },
  {
    "name": "worker-2",
    "url": "http://10.0.0.2:18789",
    "token": "sk-token-2",
    "timeout": "60s"
  },
  {
    "name": "worker-3",
    "url": "https://remote.example.com:18789",
    "token": "sk-token-3"
  }
]' ./openclaw-mcp-server
```

### Instance Config Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Unique name (1-64 chars, alphanumeric + dashes/underscores, starts with alphanumeric) |
| `url` | string | yes | Gateway URL (must be `http://` or `https://`) |
| `token` | string | no | Bearer token for gateway authentication |
| `timeout` | string | no | Request timeout as Go duration (e.g., `30s`). Falls back to `120s` |
| `default` | boolean | no | Mark as the default instance. At most one instance can be default. If none is marked, the first instance becomes the default |

### Instance Selection

Every MCP tool accepts an optional `instance` parameter:

- If `instance` is provided, the request goes to that specific worker
- If `instance` is omitted, the request goes to the default worker

When `OPENCLAW_INSTANCES` is set, it completely overrides `OPENCLAW_URL` and `OPENCLAW_TOKEN`.

## Validation

The server validates all configuration at startup and exits with an error if:

- No instances are configured
- An instance name is invalid (wrong format or duplicate)
- An instance URL is not `http://` or `https://`
- More than one instance is marked as default
- A timeout string can't be parsed as a Go duration

## HTTP Transport

When running in HTTP mode (`--transport http`), the server exposes:

| Endpoint | Auth | Description |
|----------|------|-------------|
| `POST /mcp` | Bearer token (if `MCP_AUTH_TOKEN` set) | MCP Streamable HTTP endpoint |
| `GET /health` | None | Health check, returns `{"status":"ok"}` |

### Authentication

If `MCP_AUTH_TOKEN` is set, all requests to `/mcp` must include:

```
Authorization: Bearer <MCP_AUTH_TOKEN>
```

Requests without a valid token receive `401 Unauthorized`.

The `/health` endpoint is always unauthenticated.

## Examples

### Development (stdio, local gateway)

```bash
OPENCLAW_URL=http://127.0.0.1:18789 ./openclaw-mcp-server
```

### Production (HTTP, multiple workers, auth)

```bash
OPENCLAW_INSTANCES='[
  {"name":"prod-1","url":"http://10.0.0.1:18789","token":"sk-1","default":true},
  {"name":"prod-2","url":"http://10.0.0.2:18789","token":"sk-2"}
]' \
MCP_AUTH_TOKEN=secure-random-token \
./openclaw-mcp-server --transport http --port 8080
```

### Claude Desktop

```json
{
  "mcpServers": {
    "openclaw": {
      "command": "/path/to/openclaw-mcp-server",
      "env": {
        "OPENCLAW_URL": "http://127.0.0.1:18789",
        "OPENCLAW_TOKEN": "your-gateway-token"
      }
    }
  }
}
```
