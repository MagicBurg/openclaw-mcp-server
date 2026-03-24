# TODO

## Completed
- [x] Study `../openclaw/` — understand core domain, extensions, skills, and key APIs
- [x] Study `../openclaw-mcp/` — understand existing MCP tool definitions, plugin structure, and protocol usage
- [x] Research Go MCP SDK — chose official `modelcontextprotocol/go-sdk` v1.3.1
- [x] Write implementation plan (plan/001)
- [x] Implement MCP server foundation with 5 tools

## Future Enhancements
- [ ] Streaming chat support (`stream: true` in chat completions)
- [ ] WebSocket client for gateway RPC (tool discovery, sessions, real-time events)
- [ ] Async task management (queue long-running operations)
- [ ] OAuth 2.1 auth for HTTP transport (replace simple bearer token)
- [ ] Worker routing strategies (round-robin, capability-based, load-aware)
- [ ] Structured result types per tool (instead of generic `any`)
- [ ] Upgrade to MCP SDK v1.4+ when Go 1.25 is stable
- [ ] Additional MCP tools: sessions, agents, config, memory
