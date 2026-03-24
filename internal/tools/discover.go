package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// DiscoverInput is the input schema for the openclaw_discover tool.
type DiscoverInput struct {
	Instance string `json:"instance,omitempty" jsonschema:"Target worker instance name (uses default if omitted)"`
}

// DiscoverOutput is the output from the openclaw_discover tool.
type DiscoverOutput struct {
	Instance  string                    `json:"instance"`
	Available []openclaw.ToolAvailability `json:"available"`
	Total     int                        `json:"total"`
}

// DiscoverHandler returns the MCP tool handler for openclaw_discover.
func DiscoverHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input DiscoverInput) (*mcp.CallToolResult, DiscoverOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input DiscoverInput) (*mcp.CallToolResult, DiscoverOutput, error) {
		inst, err := registry.Resolve(input.Instance)
		if err != nil {
			return nil, DiscoverOutput{}, err
		}

		tools := inst.Client.DiscoverTools(ctx)

		// Filter to only available tools.
		var available []openclaw.ToolAvailability
		for _, t := range tools {
			if t.Status == "available" {
				available = append(available, t)
			}
		}

		output := DiscoverOutput{
			Instance:  inst.Name,
			Available: available,
			Total:     len(available),
		}

		text, _ := json.Marshal(output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, output, nil
	}
}
