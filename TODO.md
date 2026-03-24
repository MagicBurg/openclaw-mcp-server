# TODO

## Research
- [x] Study `../openclaw/` — understand core domain, extensions, skills, and key APIs
- [x] Study `../openclaw-mcp/` — understand existing MCP tool definitions, plugin structure, and protocol usage
- [x] Identify scope: expose OpenClaw skills/tools, multi-instance worker orchestration
- [x] Document research findings in `docs/research.md`
- [x] Research Go MCP SDK — chose official `modelcontextprotocol/go-sdk`
- [x] Research chat endpoints — HTTP (`/v1/chat/completions`) and WebSocket RPC (`chat.send`, `agent`)
- [x] Research cron system — WebSocket RPC methods, data model, agent tool
- [ ] Verify `POST /tools/invoke` endpoint behavior (test against a running OpenClaw instance)

## Key Decisions Made
- **Language:** Go
- **MCP SDK:** Official `github.com/modelcontextprotocol/go-sdk`
- **Primary integration:** `POST /tools/invoke` for direct tool invocation
- **Chat:** `POST /v1/chat/completions` (OpenAI-compatible)
- **Cron:** Via `POST /tools/invoke` with `tool: "cron"` (HTTP, simpler than WebSocket)
- **Architecture:** Multi-instance worker orchestration
- **Extensible:** Architecture must make adding new gateway functions trivial

## Key Decisions Remaining
- [ ] Transport: stdio + Streamable HTTP (recommended) or stdio + SSE?
- [ ] Worker routing strategy: round-robin, capability-based, explicit targeting?
- [ ] Auth: built-in bearer token middleware for MCP clients, gateway tokens per instance
- [ ] WebSocket client needed later for discovery (`tools.catalog`, `skills.status`)?

## Planning
- [ ] Write implementation plan for the MCP server

## Implementation
- [ ] (pending plan)
