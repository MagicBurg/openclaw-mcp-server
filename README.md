# OpenClaw MCP Server

A Go [MCP](https://modelcontextprotocol.io/) (Model Context Protocol) server that exposes [OpenClaw](https://github.com/nicepkg/openclaw)'s skills, tools, chat, and cron capabilities. Designed to orchestrate multiple OpenClaw instances as workers for an upstream agentic platform.

## Overview

OpenClaw is a personal AI assistant framework with 50+ skills, 86+ extensions, and 20+ built-in agent tools. This MCP server acts as a bridge layer, letting any MCP-compatible client (Claude Desktop, custom agents, etc.) invoke OpenClaw capabilities across one or more gateway instances.

**Key features:**
- 5 MCP tools covering chat, direct tool invocation, cron management, health checks, and instance listing
- Multi-instance support — treat multiple OpenClaw gateways as workers
- Two transports: stdio (local) and Streamable HTTP (remote)
- Simple bearer token auth for HTTP mode

See the [Architecture](docs/architecture.md) doc for details, or jump to [Installation](docs/installation.md) to get started.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/MagicBurg/openclaw-mcp-server/main/install.sh | sh
```

Or clone and install manually:

```bash
git clone https://github.com/MagicBurg/openclaw-mcp-server.git && cd openclaw-mcp-server && ./install.sh
```

Installs to `~/.local/bin` by default. See [Installation](docs/installation.md) for options.

## Quick Start

```bash
# Run (stdio, single instance)
OPENCLAW_URL=http://127.0.0.1:18789 \
OPENCLAW_TOKEN=your-token \
openclaw-mcp-server

# Run (HTTP, multi-instance)
OPENCLAW_INSTANCES='[{"name":"w1","url":"http://10.0.0.1:18789","token":"sk-1","default":true}]' \
MCP_AUTH_TOKEN=my-secret \
openclaw-mcp-server --transport http --port 21789
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `openclaw_chat` | Send a message to an OpenClaw instance and get a response |
| `openclaw_tool_invoke` | Invoke any OpenClaw tool directly (browser, memory, web_fetch, etc.) |
| `openclaw_cron` | Manage cron jobs (list, add, update, remove, run, status) |
| `openclaw_status` | Check health status of a worker instance |
| `openclaw_instances` | List all configured worker instances |

See [Tools Reference](docs/tools.md) for full input/output schemas and examples.

## CLI Chat Client

An interactive chat-first CLI is included for testing and demos:

```bash
openclaw-mcp-cli

> Hello, what can you do?
I can help you with browsing, scheduling, memory search...

> /status
{ "status": "ok", "instance": "default" }

> /invoke browser --action snapshot --args '{"url":"https://example.com"}'
{ "ok": true, ... }

> /help
```

Type messages to chat, use `/commands` for tools. See `/help` for all commands.

## Documentation

| Doc | Description |
|-----|-------------|
| [Installation](docs/installation.md) | Prerequisites, build, and setup |
| [Configuration](docs/configuration.md) | Environment variables, CLI flags, multi-instance |
| [Architecture](docs/architecture.md) | System design, components, data flow |
| [Tools Reference](docs/tools.md) | Detailed MCP tool schemas and usage examples |
| [Development](docs/development.md) | Contributing, testing, project structure |

## License

See [LICENSE](LICENSE).
