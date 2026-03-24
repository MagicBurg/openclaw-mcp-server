# Tools Reference

All tools return JSON-formatted text content. Each tool that communicates with a gateway accepts an optional `instance` parameter to target a specific worker.

---

## openclaw_chat

Send a message to an OpenClaw instance and get a response. Supports session continuity.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | yes | The message to send |
| `session_id` | string | no | Session ID for conversation continuity across calls |
| `instance` | string | no | Target worker instance name |

### Output

```json
{
  "response": "The assistant's reply text",
  "model": "openclaw",
  "instance": "worker-1",
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 42,
    "total_tokens": 57
  }
}
```

### Example

```json
{
  "message": "What is the weather in Tokyo?",
  "session_id": "conv-123",
  "instance": "worker-1"
}
```

### Gateway Endpoint

`POST /v1/chat/completions` (OpenAI-compatible format).

---

## openclaw_tool_invoke

Invoke any OpenClaw tool directly on a worker instance. This is the most powerful tool — it provides access to all 20+ built-in OpenClaw tools (browser, canvas, memory, web_fetch, etc.).

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tool` | string | yes | Tool name (e.g., `browser`, `memory_search`, `web_fetch`, `canvas`) |
| `action` | string | no | Action for the tool (e.g., `snapshot`, `search`) |
| `args` | object | no | Tool arguments as key-value pairs |
| `session_key` | string | no | Session context for the tool |
| `instance` | string | no | Target worker instance name |

### Output

```json
{
  "ok": true,
  "instance": "worker-1",
  "result": { "content": [{ "type": "text", "text": "..." }] }
}
```

On error:

```json
{
  "ok": false,
  "instance": "worker-1",
  "error": "tool not found"
}
```

### Examples

**Browser snapshot:**
```json
{
  "tool": "browser",
  "action": "snapshot",
  "args": { "url": "https://example.com" }
}
```

**Memory search:**
```json
{
  "tool": "memory_search",
  "args": { "query": "meeting notes from last week", "limit": 5 }
}
```

**Web fetch:**
```json
{
  "tool": "web_fetch",
  "args": { "url": "https://api.example.com/data" }
}
```

### Available Tools

The tools available depend on the OpenClaw instance's configuration and installed extensions. Common built-in tools include:

| Tool | Description |
|------|-------------|
| `browser` | Browser automation (snapshot, navigate, click, type) |
| `canvas` | Visual workspace (present, hide, navigate, eval) |
| `memory_search` | Search long-term memory |
| `memory_get` | Retrieve specific memories |
| `web_fetch` | Fetch/scrape URLs |
| `web_search` | Web search |
| `image` | Image analysis |
| `image_generate` | Text-to-image generation |
| `pdf` | PDF reading/analysis |
| `cron` | Cron job management |
| `message` | Send messages to channels |
| `tts` | Text-to-speech |
| `sessions_list` | List agent sessions |
| `agents_list` | List available agents |
| `gateway` | Call remote gateways |

### Gateway Endpoint

`POST /tools/invoke`

---

## openclaw_cron

Manage cron jobs on an OpenClaw instance. This is a convenience wrapper around `openclaw_tool_invoke` with `tool: "cron"`.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | string | yes | One of: `status`, `list`, `add`, `update`, `remove`, `run` |
| `job` | object | no | Job definition (for `add` action) |
| `job_id` | string | no | Job ID (for `update`, `remove`, `run` actions) |
| `patch` | object | no | Patch fields (for `update` action) |
| `instance` | string | no | Target worker instance name |

### Output

```json
{
  "ok": true,
  "instance": "worker-1",
  "result": { "jobs": [...], "total": 3 }
}
```

### Examples

**List jobs:**
```json
{ "action": "list" }
```

**Check cron system status:**
```json
{ "action": "status" }
```

**Add a cron job:**
```json
{
  "action": "add",
  "job": {
    "name": "Daily summary",
    "schedule": { "kind": "cron", "expr": "0 9 * * *", "tz": "America/New_York" },
    "payload": { "kind": "agentTurn", "message": "Summarize today's events" },
    "sessionTarget": "isolated",
    "wakeMode": "next-heartbeat"
  }
}
```

**Run a job immediately:**
```json
{
  "action": "run",
  "job_id": "abc123"
}
```

**Update a job:**
```json
{
  "action": "update",
  "job_id": "abc123",
  "patch": { "enabled": false }
}
```

**Remove a job:**
```json
{
  "action": "remove",
  "job_id": "abc123"
}
```

### Cron Schedule Types

| Kind | Description | Example |
|------|-------------|---------|
| `at` | One-shot at a specific time | `{"kind": "at", "at": "2026-04-01T09:00:00Z"}` |
| `every` | Repeating interval | `{"kind": "every", "everyMs": 3600000}` |
| `cron` | Standard cron expression | `{"kind": "cron", "expr": "0 9 * * *", "tz": "UTC"}` |

### Gateway Endpoint

`POST /tools/invoke` with `tool: "cron"`

---

## openclaw_status

Check the health status of a worker instance.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `instance` | string | no | Target worker instance name |

### Output

```json
{
  "status": "ok",
  "instance": "worker-1"
}
```

On error:

```json
{
  "status": "error",
  "instance": "worker-1",
  "message": "request to /health: dial tcp 10.0.0.1:18789: connect: connection refused"
}
```

### Gateway Endpoint

`GET /health`

---

## openclaw_instances

List all configured worker instances. This tool reads from the local registry — it does not make any gateway calls.

### Input

_(no parameters)_

### Output

```json
{
  "instances": [
    { "name": "worker-1", "url": "http://10.0.0.1:18789", "is_default": true },
    { "name": "worker-2", "url": "http://10.0.0.2:18789", "is_default": false }
  ],
  "total": 2
}
```

Authentication tokens are never included in the output.
