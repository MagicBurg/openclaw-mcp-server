# Research Notes

## Round 1 — Initial Research (2026-03-23)

### Overview

We studied two sibling projects to inform the design of this Go MCP server:

- **`../openclaw/`** — The main OpenClaw application (TypeScript monorepo)
- **`../openclaw-mcp/`** — An existing MCP implementation (TypeScript/Node)

---

## OpenClaw Gateway API

The gateway is the primary interface for interacting with OpenClaw. It exposes both WebSocket and HTTP APIs.

### HTTP Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health`, `/healthz` | GET | Liveness check |
| `/ready`, `/readyz` | GET | Readiness check |
| `/v1/chat/completions` | POST | OpenAI-compatible chat (main integration point) |
| `/v1/responses` | POST | Alternative chat format |
| `/tools/invoke` | POST | **Direct tool invocation** |
| `/sessions/kill` | POST | Kill/stop a session |
| `/sessions/history` | GET | Get session transcript |
| `/hooks/{action}` | POST | Webhook dispatcher |

### Authentication

- Bearer token: `Authorization: Bearer <token>`
- Loopback connections (127.0.0.1) skip auth by default
- Configurable in `openclaw.json` under `gateway.auth`

### Chat Completions (`POST /v1/chat/completions`)

The existing `openclaw-mcp` uses this as its sole integration point.

**Request:**
```json
{
  "model": "claude-opus-4-5",
  "messages": [{ "role": "user", "content": "..." }],
  "max_tokens": 4096
}
```
**Headers:** `x-openclaw-session-key` for session context.

**Response:** Standard OpenAI chat completion format.

### Direct Tool Invocation (`POST /tools/invoke`)

This is the key endpoint for our MCP server — it allows invoking OpenClaw tools directly without going through a chat/agent flow.

**Request:**
```json
{
  "tool": "browser",
  "action": "snapshot",
  "args": { "url": "https://example.com" },
  "sessionKey": "agent1:main",
  "dryRun": false
}
```
**Headers:**
- `Authorization: Bearer <token>`
- `X-OpenClaw-Message-Channel: <channel>` (optional)
- `X-OpenClaw-Account-Id: <account-id>` (optional)

**Response:**
```json
{
  "ok": true,
  "result": { "content": [{ "type": "text", "text": "..." }] }
}
```

### WebSocket RPC Methods (80+)

Key methods relevant to our MCP server:

| Method | Purpose |
|--------|---------|
| `agent` | Execute agent with message |
| `agent.wait` | Wait for agent run completion |
| `chat.send` | Send message to session |
| `chat.history` | Get conversation history |
| `chat.abort` | Cancel running chat |
| `sessions.list` | List sessions |
| `sessions.create` | Create session |
| `sessions.send` | Send to session |
| `sessions.patch` | Update session config |
| `sessions.delete` | Delete session |
| `models.list` | List available models |
| `tools.catalog` | Get available tools/skills |
| `skills.status` | Get skills status |
| `skills.install` | Install skill |
| `health` | Gateway health |
| `config.get` | Get config |
| `node.list` | List connected nodes |
| `cron.list/add/remove/run` | Cron management |

### Agent Invocation Parameters

```typescript
{
  message: string,           // required
  agentId?: string,
  model?: string,
  provider?: string,
  thinking?: "enabled"|"disabled"|"auto",
  sessionKey?: string,
  deliver?: boolean,         // route to channels
  timeout?: number,
  attachments?: unknown[],
  idempotencyKey: string
}
```

---

## OpenClaw Skills & Tools

### Built-in Agent Tools (~20)

Created via `createOpenClawTools()` factory. Each tool follows:

```typescript
interface AgentTool {
  name: string;
  label: string;
  description: string;
  parameters: TypeBoxSchema;  // JSON Schema via TypeBox
  execute: (toolCallId, args) => Promise<AgentToolResult>;
}
```

**Tool list:**
- `browser` — sandbox/host browser automation (snapshot, navigate, click, type, etc.)
- `canvas` — visual workspace (present, hide, navigate, eval, snapshot)
- `nodes` — device commands (camera, screen, location, notifications)
- `cron` — schedule tasks
- `message` — send/route messages to channels
- `tts` — text-to-speech
- `image_generate` — text-to-image (DALL-E etc.)
- `image` — vision/image analysis
- `pdf` — PDF reading/analysis
- `web_search` — web search
- `web_fetch` — URL fetching/scraping
- `gateway` — call remote gateways
- `agents_list` — list agents
- `sessions_list/spawn/send/yield/history` — session management
- `subagents` — subagent management
- `memory_search/get` — long-term memory

### Tool Schema Pattern

Tools use a discriminator pattern with an `action` enum field:
```typescript
Type.Object({
  action: stringEnum(["status", "start", "stop", "open", "snapshot", ...]),
  target: Type.Optional(Type.String()),
  url: Type.Optional(Type.String()),
  // ... action-specific optional fields
})
```

### Skills (50+ in `skills/` directory)

Skills are user-facing tool collections defined by `SKILL.md` manifests:

```yaml
---
name: github
description: GitHub operations
metadata:
  openclaw:
    emoji: "🐙"
    requires:
      bins: ["gh"]
    install:
      - id: gh-cli
        kind: brew
        formula: gh
---
```

Skills are wrappers around CLI tools/external services. They're made available to agents as system prompt context, not as direct tool invocations.

### Direct Tool Invocation

Tools **can** be invoked directly via `POST /tools/invoke` HTTP endpoint — this is our primary integration path. The gateway handles tool instantiation, schema validation, policy checks, and execution.

### MCP/ACP Bridge

OpenClaw already has MCP integration via:
- **mcporter** skill — CLI for calling MCP servers
- **ACPX** extension — bridges MCP tools into agent environment

---

## Existing `openclaw-mcp` Architecture

### Components

```
src/
├── index.ts                    # Entry point
├── cli.ts                      # yargs argument parsing
├── openclaw/
│   ├── client.ts               # HTTP client (POST /v1/chat/completions)
│   ├── registry.ts             # Multi-instance management
│   └── types.ts                # Type definitions
├── server/
│   ├── tools-registration.ts   # MCP server factory + tool handlers
│   └── sse.ts                  # Express HTTP server (SSE + Streamable HTTP)
├── auth/
│   └── provider.ts             # OAuth 2.1 (PKCE, token lifecycle)
├── mcp/
│   ├── tools/
│   │   ├── chat.ts             # openclaw_chat
│   │   ├── status.ts           # openclaw_status
│   │   ├── instances.ts        # openclaw_instances
│   │   └── tasks.ts            # async task tools (4 tools)
│   └── tasks/
│       └── manager.ts          # In-memory task queue
└── utils/
    ├── logger.ts               # Logging with sensitive data redaction
    ├── errors.ts               # Custom error types
    ├── validation.ts           # Input validation
    └── response-helpers.ts     # Tool response formatting
```

### 7 MCP Tools Exposed

**Sync:**
- `openclaw_chat` — Send message, get response (via `/v1/chat/completions`)
- `openclaw_status` — Health check
- `openclaw_instances` — List instances (never exposes tokens)

**Async:**
- `openclaw_chat_async` — Queue message, get task_id
- `openclaw_task_status` — Check task progress
- `openclaw_task_list` — List tasks with filtering
- `openclaw_task_cancel` — Cancel pending task

### Instance Registry

- Multi-instance support via `OPENCLAW_INSTANCES` JSON env var
- Single-instance fallback via `OPENCLAW_URL` + `OPENCLAW_GATEWAY_TOKEN`
- Instance names: 1-64 chars, alphanumeric + dashes + underscores
- URLs validated (http/https only, SSRF prevention)
- Tokens never exposed in API responses

### Transport Modes

- **stdio** — local Claude Desktop (default)
- **SSE** — remote Claude.ai, with OAuth 2.1 + CORS

### Key Limitations

The existing implementation **only uses `/v1/chat/completions`** — it doesn't leverage the gateway's direct tool invocation (`/tools/invoke`) or WebSocket RPC methods. This is the main gap our Go server will fill.

---

## Key Insights for Our Go MCP Server

1. **Primary integration: `POST /tools/invoke`** — invoke OpenClaw tools directly, bypassing the chat/agent layer. This gives us direct access to all 20+ built-in tools.

2. **Secondary integration: WebSocket RPC** — for real-time operations (sessions, agent invocation, event streaming). May be needed for tools.catalog, skills.status, etc.

3. **Multi-instance as workers** — each OpenClaw instance is a worker. The MCP server routes tool calls to appropriate workers based on availability, capability, or explicit targeting.

4. **Extensible tool surface** — start with core tools (chat, tool invoke, status), add more gateway functions as needed. The architecture should make adding new MCP tools trivial.

5. **Skill discovery** — use `tools.catalog` and `skills.status` WebSocket methods to dynamically discover what tools/skills each worker has available.

6. **Go MCP SDK** — use the official SDK (`github.com/modelcontextprotocol/go-sdk`).

---

## Round 2 — Go MCP SDK Research (2026-03-23)

### SDK Choice: Official `modelcontextprotocol/go-sdk`

Two viable options exist. We chose the official SDK over `mark3labs/mcp-go`.

| | **Official SDK** | **mcp-go** |
|---|---|---|
| Import | `github.com/modelcontextprotocol/go-sdk/mcp` | `github.com/mark3labs/mcp-go` |
| Stars | 4,214 | 8,411 (older, was de facto before official) |
| API | Generics + struct-based (type-safe) | Builder pattern (manual extraction) |
| Auth | Built-in `auth` package (OAuth/bearer) | DIY middleware |
| Backing | Anthropic + Google | Community |

### Key API Patterns

**Tool definition via Go structs:**
```go
type GreetInput struct {
    Name string `json:"name" jsonschema:"the name of the person to greet"`
}

func Greet(ctx context.Context, req *mcp.CallToolRequest, input GreetInput) (
    *mcp.CallToolResult, GreetOutput, error,
) {
    return nil, GreetOutput{Greeting: "Hello, " + input.Name}, nil
}

mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hello"}, Greet)
```

**Transports:**
```go
// Stdio
server.Run(ctx, &mcp.StdioTransport{})

// Streamable HTTP
handler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server { return server },
    nil,
)
http.ListenAndServe(":8080", handler)

// SSE (legacy)
handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return server }, nil)
```

**Auth middleware:**
```go
verifier := func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
    // validate token
    return &auth.TokenInfo{UserID: "user-123", Scopes: []string{"read"}}, nil
}
authMiddleware := auth.RequireBearerToken(verifier, nil)
http.ListenAndServe(":8080", authMiddleware(mcpHandler))
```

### Features Available
- Auto schema inference from Go structs via `jsonschema` tags
- Built-in pagination (configurable page size)
- Resources and resource templates (URI-based)
- Prompts with arguments
- `slog` integration for MCP logging
- `InMemoryTransport` for testing
- Session timeout and stream resumption
- Session hijacking protection (tied to auth user)

---

## Round 3 — Chat & Cron Endpoints (2026-03-23)

### Chat Endpoints

Two ways to interact with OpenClaw for chat:

#### HTTP: `POST /v1/chat/completions` (OpenAI-compatible)

**Request:**
```json
{
  "model": "openclaw",
  "messages": [
    { "role": "user", "content": "Hello" }
  ],
  "stream": false,
  "user": "user-id"
}
```
- Supports multi-part content (text + image_url)
- Session derived from `model` or `user` param
- Streaming via SSE (`stream: true`)

**Response (non-streaming):**
```json
{
  "id": "chatcmpl_<uuid>",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "openclaw",
  "choices": [{ "index": 0, "message": { "role": "assistant", "content": "..." }, "finish_reason": "stop" }],
  "usage": { "prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0 }
}
```

#### WebSocket RPC: `chat.send` vs `agent`

| Aspect | `chat.send` | `agent` |
|--------|-------------|---------|
| Session | Requires existing sessionKey | Optional, spawns ephemeral |
| Model override | Not allowed | Auth-gated |
| Delivery | Single session | Multi-channel, groups, threads |
| Use case | Chat within session | Ad-hoc tasks, automation |

**`agent` method params (key fields):**
```
message, agentId, model, provider, thinking, sessionKey, deliver, channel, to,
timeout, attachments, idempotencyKey, extraSystemPrompt, lane, label
```

### Cron System

#### WebSocket RPC Methods

| Method | Purpose |
|--------|---------|
| `cron.list` | List jobs (filterable, paginated, sortable) |
| `cron.add` | Create a new cron job |
| `cron.update` | Patch an existing job |
| `cron.remove` | Delete a job |
| `cron.run` | Execute a job immediately (force or due-only) |
| `cron.runs` | Get execution history (filterable, paginated) |
| `cron.status` | Get cron system status (enabled, job count, next wake) |
| `wake` | Trigger a wake event |

#### Cron Job Data Model

```typescript
{
  id: string,
  name: string,
  description?: string,
  enabled: boolean,
  deleteAfterRun?: boolean,
  agentId?: string,
  sessionKey?: string,

  // Schedule (one of three kinds)
  schedule:
    | { kind: "at", at: "ISO-8601" }                    // one-shot
    | { kind: "every", everyMs: number, anchorMs?: number }  // interval
    | { kind: "cron", expr: "0 9 * * *", tz?: "America/New_York", staggerMs?: number },  // cron expr

  // What to execute
  payload:
    | { kind: "systemEvent", text: string }
    | { kind: "agentTurn", message: string, model?: string, thinking?: string, deliver?: boolean, ... },

  // Where to run
  sessionTarget: "main" | "isolated" | "current" | "session:<id>",
  wakeMode: "next-heartbeat" | "now",

  // Optional delivery
  delivery?: { mode: "none"|"announce"|"webhook", channel?: string, to?: string, ... },

  // Runtime state
  state: {
    nextRunAtMs?: number,
    lastRunAtMs?: number,
    lastRunStatus?: "ok"|"error"|"skipped",
    consecutiveErrors?: number,
    ...
  }
}
```

#### Cron Agent Tool

The `cron` tool is available as a built-in agent tool with actions:
`status`, `list`, `add`, `update`, `remove`, `run`, `runs`, `wake`

It uses a flat params schema with an `action` discriminator — same pattern as other OpenClaw tools.

### No HTTP Endpoints for Cron

Cron is **WebSocket-only**. To manage cron jobs from our MCP server, we'll need a WebSocket client to the gateway, or use `POST /tools/invoke` with `tool: "cron"` to invoke the cron agent tool via HTTP.

### Integration Strategy

For our Go MCP server, we have two paths for cron:

1. **Via `POST /tools/invoke`** — invoke the `cron` agent tool directly (HTTP, simpler)
2. **Via WebSocket RPC** — call `cron.list`, `cron.add`, etc. directly (richer, real-time)

Option 1 is simpler and consistent with our tools strategy. Option 2 gives more control but requires a WebSocket client. We can start with option 1 and add WebSocket later if needed.
