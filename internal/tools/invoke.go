package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// ToolInvokeInput is the input schema for the openclaw_tool_invoke tool.
type ToolInvokeInput struct {
	Tool       string         `json:"tool" jsonschema:"Tool name (e.g. browser, memory_search, web_fetch)"`
	Action     string         `json:"action,omitempty" jsonschema:"Action for the tool (e.g. snapshot, search)"`
	Args       map[string]any `json:"args,omitempty" jsonschema:"Tool arguments as key-value pairs"`
	SessionKey string         `json:"session_key,omitempty" jsonschema:"Optional session context for the tool"`
	Instance   string         `json:"instance,omitempty" jsonschema:"Target worker instance name (uses default if omitted)"`
}

// ToolInvokeOutput is the output from the openclaw_tool_invoke tool.
type ToolInvokeOutput struct {
	OK       bool   `json:"ok"`
	Instance string `json:"instance"`
	Result   any    `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ToolInvokeHandler returns the MCP tool handler for openclaw_tool_invoke.
func ToolInvokeHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input ToolInvokeInput) (*mcp.CallToolResult, ToolInvokeOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ToolInvokeInput) (*mcp.CallToolResult, ToolInvokeOutput, error) {
		if input.Tool == "" {
			return nil, ToolInvokeOutput{}, fmt.Errorf("tool is required")
		}

		inst, err := registry.Resolve(input.Instance)
		if err != nil {
			return nil, ToolInvokeOutput{}, err
		}

		toolReq := openclaw.ToolInvokeRequest{
			Tool:       input.Tool,
			Action:     input.Action,
			Args:       input.Args,
			SessionKey: input.SessionKey,
		}

		resp, err := inst.Client.ToolInvoke(ctx, toolReq)
		if err != nil {
			return errorResult(fmt.Sprintf("tool invoke error on %s: %v", inst.Name, err)), ToolInvokeOutput{}, nil
		}

		output := ToolInvokeOutput{
			OK:       resp.OK,
			Instance: inst.Name,
			Result:   resp.Result,
		}
		if resp.Error != nil {
			output.Error = resp.Error.Message
		}

		text, _ := json.Marshal(output)
		callResult := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}
		if !resp.OK {
			callResult.IsError = true
		}
		return callResult, output, nil
	}
}
