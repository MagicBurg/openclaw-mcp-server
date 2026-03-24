# Installation

## Prerequisites

- **Go 1.24+** — the project uses `go 1.24.0` in `go.mod`. If your system Go is older, Go's toolchain auto-download will fetch the right version (set `GOTOOLCHAIN=auto` if needed).
- **One or more running OpenClaw gateway instances** — the MCP server connects to OpenClaw via HTTP. See the [OpenClaw documentation](https://github.com/nicepkg/openclaw) for setup instructions.

## Quick Install

```bash
git clone https://github.com/weiboz/openclaw-mcp-server.git
cd openclaw-mcp-server
./install.sh
```

The install script will:
1. Check for Go 1.24+ (and install it if missing)
2. Download dependencies
3. Run tests
4. Build the binary
5. Install to `/usr/local/bin`
6. Create `.env` from `.env.example`

Options:
```bash
./install.sh --prefix ~/.local     # Install to ~/.local/bin instead
./install.sh --skip-tests          # Skip test step
./install.sh --help                # Show all options
```

## Manual Build

```bash
git clone https://github.com/weiboz/openclaw-mcp-server.git
cd openclaw-mcp-server
go build -o openclaw-mcp-server ./cmd/server/
```

This produces a single static binary `openclaw-mcp-server`.

## Verify

```bash
./openclaw-mcp-server --help
```

## Run

### Stdio Mode (Local / Claude Desktop)

Stdio is the default transport. The server reads MCP messages from stdin and writes to stdout.

```bash
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-gateway-token \
./openclaw-mcp-server
```

### HTTP Mode (Remote / Agentic Platform)

For remote access, use Streamable HTTP transport:

```bash
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-gateway-token \
MCP_AUTH_TOKEN=your-mcp-auth-token \
./openclaw-mcp-server --transport http --port 8080
```

The MCP endpoint is served at `POST /mcp`. A health check is available at `GET /health`.

### Multi-Instance (Workers)

To connect to multiple OpenClaw gateways as workers:

```bash
OPENCLAW_INSTANCES='[
  {"name": "worker-1", "url": "http://10.0.0.1:18789", "token": "sk-1", "default": true},
  {"name": "worker-2", "url": "http://10.0.0.2:18789", "token": "sk-2", "timeout": "60s"}
]' ./openclaw-mcp-server
```

See [Configuration](configuration.md) for the full reference.

## OpenClaw Gateway Setup

The MCP server communicates with OpenClaw via two HTTP endpoints:

1. **`POST /v1/chat/completions`** — for chat (OpenAI-compatible). Requires the HTTP API to be enabled in `openclaw.json`:
   ```json
   {
     "gateway": {
       "http": {
         "endpoints": {
           "chatCompletions": { "enabled": true }
         }
       }
     }
   }
   ```

2. **`POST /tools/invoke`** — for direct tool invocation (tool invoke, cron). Requires gateway auth token.

3. **`GET /health`** — for health checks. Always available.

## Using with Claude Desktop

Add the server to your Claude Desktop MCP configuration:

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
