package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// StatusInput is the input schema for the openclaw_status tool.
type StatusInput struct {
	Instance string `json:"instance,omitempty" jsonschema:"Target worker instance name (uses default if omitted)"`
}

// StatusOutput is the output from the openclaw_status tool.
type StatusOutput struct {
	Status   string `json:"status"`
	Instance string `json:"instance"`
	Message  string `json:"message,omitempty"`
}

// StatusHandler returns the MCP tool handler for openclaw_status.
func StatusHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
		inst, err := registry.Resolve(input.Instance)
		if err != nil {
			return nil, StatusOutput{}, err
		}

		status, healthErr := inst.Client.Health(ctx)
		output := StatusOutput{
			Status:   status,
			Instance: inst.Name,
		}
		if healthErr != nil {
			output.Message = healthErr.Error()
		}

		text, _ := json.Marshal(output)
		callResult := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}
		if status != "ok" {
			callResult.IsError = true
		}
		return callResult, output, nil
	}
}
