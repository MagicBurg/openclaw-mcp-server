# Installation

## Prerequisites

- **Go 1.24+** — the project uses `go 1.24.0` in `go.mod`. If your system Go is older, Go's toolchain auto-download will fetch the right version (set `GOTOOLCHAIN=auto` if needed).
- **One or more running OpenClaw gateway instances** — the MCP server connects to OpenClaw via HTTP. See [Step 1: OpenClaw Backend Setup](#step-1-openclaw-backend-setup) below.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/MagicBurg/openclaw-mcp-server/main/install.sh | sh
```

Or clone and run locally:

```bash
git clone https://github.com/MagicBurg/openclaw-mcp-server.git
cd openclaw-mcp-server
./install.sh
```

The install script will:
1. Check for Go 1.24+ (and install it if missing)
2. Download dependencies
3. Run tests
4. Build both `openclaw-mcp-server` and `openclaw-mcp-cli`
5. Install both binaries to `~/.local/bin`

Options:
```bash
./install.sh --prefix /usr/local   # Install to /usr/local/bin instead
./install.sh --skip-tests          # Skip test step
./install.sh --help                # Show all options
```

## Manual Build

```bash
git clone https://github.com/MagicBurg/openclaw-mcp-server.git
cd openclaw-mcp-server
go build -o openclaw-mcp-server ./cmd/server/
go build -o openclaw-mcp-cli ./cmd/cli/
```

## Verify

```bash
openclaw-mcp-server --help
openclaw-mcp-cli --help
```

---

## Step 1: OpenClaw Backend Setup

The MCP server treats OpenClaw gateways as backend worker instances. You need at least one running gateway before using the MCP server.

### 1.1 Install OpenClaw

Follow the [OpenClaw documentation](https://github.com/nicepkg/openclaw) to install and start a gateway. By default, it listens at `http://127.0.0.1:18789`.

### 1.2 Enable Required Endpoints

The MCP server uses three gateway endpoints. Configure your `openclaw.json` to enable them:

```json
{
  "gateway": {
    "auth": {
      "mode": "token",
      "token": "your-gateway-token"
    },
    "http": {
      "endpoints": {
        "chatCompletions": { "enabled": true }
      }
    }
  }
}
```

| Endpoint | Purpose | Required Config |
|----------|---------|-----------------|
| `GET /health` | Health checks | Always available |
| `POST /v1/chat/completions` | Chat (OpenAI-compatible) | `chatCompletions.enabled: true` |
| `POST /tools/invoke` | Direct tool invocation | Gateway auth token |

### 1.3 Verify the Gateway

```bash
# Health check
curl http://127.0.0.1:18789/health

# Chat (requires token)
curl -X POST http://127.0.0.1:18789/v1/chat/completions \
  -H "Authorization: Bearer your-gateway-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"openclaw","messages":[{"role":"user","content":"hello"}]}'
```

### 1.4 Multiple Workers (Optional)

For multi-instance setups, run additional OpenClaw gateways on different hosts/ports. Each gets its own URL and token. Example topology:

```
┌──────────────────────────────────────┐
│         MCP Server                   │
│    openclaw-mcp-server               │
└───┬──────────┬──────────┬────────────┘
    │          │          │
    ▼          ▼          ▼
 worker-1   worker-2   worker-3
 10.0.0.1   10.0.0.2   10.0.0.3
 :18789     :18789     :18789
```

Create a `config.toml` for this setup:

```toml
[server]
transport = "http"
port = 21789
auth_token = "my-mcp-secret"

[[instances]]
name = "worker-1"
url = "http://10.0.0.1:18789"
token = "gw-token-1"
default = true

[[instances]]
name = "worker-2"
url = "http://10.0.0.2:18789"
token = "gw-token-2"

[[instances]]
name = "worker-3"
url = "http://10.0.0.3:18789"
token = "gw-token-3"
timeout = "60s"
```

---

## Step 2: Run the MCP Server

### Single Worker (Simplest)

```bash
openclaw-mcp-server \
  --openclaw-url http://127.0.0.1:18789 \
  --openclaw-token your-gateway-token
```

### Multiple Workers (Config File)

```bash
openclaw-mcp-server --config config.toml
```

### Multiple Workers (Env Var)

```bash
OPENCLAW_INSTANCES='[
  {"name":"worker-1","url":"http://10.0.0.1:18789","token":"gw-token-1","default":true},
  {"name":"worker-2","url":"http://10.0.0.2:18789","token":"gw-token-2"}
]' openclaw-mcp-server
```

---

## Step 3: Connect Callers

### Using with OpenClaw (as MCP caller)

OpenClaw can consume MCP servers via its ACPX extension. To connect this MCP server as a tool source for another OpenClaw instance:

**Option A: stdio (local)**

Add to your calling OpenClaw's `openclaw.json`:

```json
{
  "mcpServers": {
    "openclaw-workers": {
      "command": "openclaw-mcp-server",
      "args": ["--config", "/path/to/config.toml"]
    }
  }
}
```

**Option B: HTTP (remote)**

Start the MCP server in HTTP mode:

```bash
openclaw-mcp-server --config config.toml --transport http --port 21789 --auth-token my-mcp-secret
```

Then configure the calling OpenClaw to connect via HTTP MCP:

```json
{
  "mcpServers": {
    "openclaw-workers": {
      "url": "http://mcp-server-host:21789/mcp",
      "headers": {
        "Authorization": "Bearer my-mcp-secret"
      }
    }
  }
}
```

Now the calling OpenClaw agent can use tools like `openclaw_chat`, `openclaw_tool_invoke`, and `openclaw_cron` to delegate work to the backend workers.

### Using with Claude Code

Add the MCP server to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "openclaw": {
      "command": "openclaw-mcp-server",
      "args": ["--config", "/path/to/config.toml"]
    }
  }
}
```

Or with environment variables:

```json
{
  "mcpServers": {
    "openclaw": {
      "command": "openclaw-mcp-server",
      "env": {
        "OPENCLAW_URL": "http://127.0.0.1:18789",
        "OPENCLAW_TOKEN": "your-gateway-token"
      }
    }
  }
}
```

Claude Code will automatically discover the 5 MCP tools (`openclaw_chat`, `openclaw_tool_invoke`, `openclaw_cron`, `openclaw_status`, `openclaw_instances`) and can use them during conversations.

### Using with Claude Desktop

Add to Claude Desktop's MCP configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "openclaw": {
      "command": "/home/you/.local/bin/openclaw-mcp-server",
      "env": {
        "OPENCLAW_URL": "http://127.0.0.1:18789",
        "OPENCLAW_TOKEN": "your-gateway-token"
      }
    }
  }
}
```

For multi-instance setups, use the config file:

```json
{
  "mcpServers": {
    "openclaw": {
      "command": "/home/you/.local/bin/openclaw-mcp-server",
      "args": ["--config", "/home/you/.config/openclaw-mcp-server/config.toml"]
    }
  }
}
```

---

## Step 4: Test with the CLI

The included CLI client is the fastest way to test your setup:

```bash
openclaw-mcp-cli --config config.toml
```

### Example Session

```
OpenClaw MCP CLI
Type a message to chat, or /help for commands.

> /instances
  * worker-1        http://10.0.0.1:18789
    worker-2        http://10.0.0.2:18789

  Active: (default)

> /status
{
  "status": "ok",
  "instance": "worker-1"
}

> /status --instance worker-2
{
  "status": "ok",
  "instance": "worker-2"
}

> Hello, what can you do?
I can help you with browsing, scheduling, memory search, and more.

> /session my-project
Session set to: my-project

> What did we discuss last time?
Based on our previous conversation...

> /instance worker-2
Instance set to: worker-2

> Summarize today's news
Here are today's top stories...

> /invoke web_fetch --args '{"url":"https://api.example.com/data"}'
{
  "ok": true,
  "instance": "worker-2",
  "result": { ... }
}

> /cron list
{
  "ok": true,
  "instance": "worker-2",
  "result": { "jobs": [], "total": 0 }
}

> /cron add --job '{"name":"daily-summary","schedule":{"kind":"cron","expr":"0 9 * * *"},"payload":{"kind":"agentTurn","message":"Summarize today"},"sessionTarget":"isolated","wakeMode":"next-heartbeat"}'
{
  "ok": true,
  ...
}

> /tools
Available tools:
  openclaw_chat             Send a message to an OpenClaw instance and get a response
  openclaw_tool_invoke      Invoke any OpenClaw tool directly
  openclaw_cron             Manage cron jobs
  openclaw_status           Check health status
  openclaw_instances        List all worker instances

> /quit
```

### CLI Quick Reference

| Command | Description |
|---------|-------------|
| _(any text)_ | Send as chat message to active worker |
| `/help` | Show all commands |
| `/tools` | List available MCP tools |
| `/status [--instance X]` | Health check |
| `/instances` | List all workers |
| `/instance [name]` | Switch active worker (empty = reset) |
| `/session [id]` | Set/show chat session ID |
| `/invoke <tool> [flags]` | Invoke any OpenClaw tool |
| `/cron <action> [flags]` | Manage cron jobs |
| `/quit` | Exit |
