package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// CronInput is the input schema for the openclaw_cron tool.
type CronInput struct {
	Action   string         `json:"action" jsonschema:"Cron action: status, list, add, update, remove, run"`
	Job      map[string]any `json:"job,omitempty" jsonschema:"Job definition for the add action"`
	JobID    string         `json:"job_id,omitempty" jsonschema:"Job ID for update, remove, or run actions"`
	Patch    map[string]any `json:"patch,omitempty" jsonschema:"Patch fields for the update action"`
	Instance string         `json:"instance,omitempty" jsonschema:"Target worker instance name (uses default if omitted)"`
}

// CronOutput is the output from the openclaw_cron tool.
type CronOutput struct {
	OK       bool   `json:"ok"`
	Instance string `json:"instance"`
	Result   any    `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

// CronHandler returns the MCP tool handler for openclaw_cron.
func CronHandler(registry *openclaw.Registry) func(ctx context.Context, req *mcp.CallToolRequest, input CronInput) (*mcp.CallToolResult, CronOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CronInput) (*mcp.CallToolResult, CronOutput, error) {
		if input.Action == "" {
			return nil, CronOutput{}, fmt.Errorf("action is required")
		}

		inst, err := registry.Resolve(input.Instance)
		if err != nil {
			return nil, CronOutput{}, err
		}

		// Build tool invoke args for the cron agent tool.
		args := map[string]any{
			"action": input.Action,
		}
		if input.Job != nil {
			args["job"] = input.Job
		}
		if input.JobID != "" {
			args["jobId"] = input.JobID
		}
		if input.Patch != nil {
			args["patch"] = input.Patch
		}

		toolReq := openclaw.ToolInvokeRequest{
			Tool: "cron",
			Args: args,
		}

		resp, err := inst.Client.ToolInvoke(ctx, toolReq)
		if err != nil {
			return errorResult(fmt.Sprintf("cron error on %s: %v", inst.Name, err)), CronOutput{}, nil
		}

		output := CronOutput{
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
