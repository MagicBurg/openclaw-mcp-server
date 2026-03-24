package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
	"github.com/weiboz/openclaw-mcp-server/internal/tools"
)

const (
	ServerName    = "openclaw-mcp-server"
	ServerVersion = "0.1.0"
)

// New creates and configures the MCP server with all tools registered.
func New(registry *openclaw.Registry) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: ServerName, Version: ServerVersion},
		nil,
	)
	tools.RegisterAll(server, registry)
	return server
}
