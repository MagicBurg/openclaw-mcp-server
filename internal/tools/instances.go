package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// InstancesInput is the input schema for the openclaw_instances tool (no params).
type InstancesInput struct{}

// InstancesOutput is the output from the openclaw_instances tool.
type InstancesOutput struct {
	Instances []openclaw.InstanceInfo `json:"instances"`
	Total     int                     `json:"total"`
}

// InstancesHandler returns the MCP tool handler for openclaw_instances.
func InstancesHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input InstancesInput) (*mcp.CallToolResult, InstancesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input InstancesInput) (*mcp.CallToolResult, InstancesOutput, error) {
		infos := registry.List()
		output := InstancesOutput{
			Instances: infos,
			Total:     len(infos),
		}

		text, _ := json.Marshal(output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, output, nil
	}
}
