package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// ChatInput is the input schema for the openclaw_chat tool.
type ChatInput struct {
	Message   string `json:"message" jsonschema:"The message to send to OpenClaw"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Optional session ID for conversation continuity"`
	Instance  string `json:"instance,omitempty" jsonschema:"Target worker instance name (uses default if omitted)"`
}

// ChatOutput is the output from the openclaw_chat tool.
type ChatOutput struct {
	Response string           `json:"response"`
	Model    string           `json:"model,omitempty"`
	Instance string           `json:"instance"`
	Usage    *openclaw.ChatUsage `json:"usage,omitempty"`
}

// ChatHandler returns the MCP tool handler for openclaw_chat.
func ChatHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input ChatInput) (*mcp.CallToolResult, ChatOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ChatInput) (*mcp.CallToolResult, ChatOutput, error) {
		if input.Message == "" {
			return nil, ChatOutput{}, fmt.Errorf("message is required")
		}

		inst, err := registry.Resolve(input.Instance)
		if err != nil {
			return nil, ChatOutput{}, err
		}

		result, err := inst.Client.Chat(ctx, input.Message, input.SessionID)
		if err != nil {
			return errorResult(fmt.Sprintf("chat error on %s: %v", inst.Name, err)), ChatOutput{}, nil
		}

		output := ChatOutput{
			Response: result.Response,
			Model:    result.Model,
			Instance: inst.Name,
			Usage:    result.Usage,
		}

		text, _ := json.Marshal(output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, output, nil
	}
}
