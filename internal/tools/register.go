package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// RegisterAll registers all OpenClaw MCP tools on the server.
func RegisterAll(server *mcp.Server, registry *openclaw.Registry) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_chat",
		Description: "Send a message to an OpenClaw instance and get a response. Supports session continuity via session_id.",
	}, ChatHandler(registry))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_tool_invoke",
		Description: "Invoke any OpenClaw tool directly on a worker instance (e.g. browser, memory_search, web_fetch, canvas, cron). Pass the tool name, optional action, and arguments.",
	}, ToolInvokeHandler(registry))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_cron",
		Description: "Manage cron jobs on an OpenClaw instance. Actions: status, list, add, update, remove, run.",
	}, CronHandler(registry))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_status",
		Description: "Check the health status of an OpenClaw worker instance.",
	}, StatusHandler(registry))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_instances",
		Description: "List all configured OpenClaw worker instances with their names, URLs, and default status. Never exposes authentication tokens.",
	}, InstancesHandler(registry))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "openclaw_discover",
		Description: "Discover which OpenClaw tools are available on a worker instance by probing the gateway.",
	}, DiscoverHandler(registry))
}

// errorResult creates an MCP error result with the given message.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
