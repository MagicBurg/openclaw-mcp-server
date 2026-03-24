package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CLI is an interactive chat-first REPL that talks to the MCP server.
type CLI struct {
	session  *mcp.ClientSession
	instance string // active instance (empty = default)
	sessionID string // active chat session
	in       io.Reader
	out      io.Writer
}

// New creates a new CLI with the given MCP client session.
func New(session *mcp.ClientSession, in io.Reader, out io.Writer) *CLI {
	return &CLI{
		session: session,
		in:      in,
		out:     out,
	}
}

// Run starts the interactive REPL. Blocks until quit or EOF.
func (c *CLI) Run(ctx context.Context) error {
	c.printf("OpenClaw MCP CLI\n")
	c.printf("Type a message to chat, or /help for commands.\n\n")

	scanner := bufio.NewScanner(c.in)
	for {
		c.printf("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if err := c.handleCommand(ctx, line); err != nil {
				if err == errQuit {
					return nil
				}
				c.printf("Error: %v\n", err)
			}
		} else {
			c.handleChat(ctx, line)
		}
	}

	return scanner.Err()
}

var errQuit = fmt.Errorf("quit")

func (c *CLI) handleCommand(ctx context.Context, line string) error {
	parts := splitArgs(line)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help", "/h":
		c.printHelp()
	case "/quit", "/exit", "/q":
		return errQuit
	case "/tools":
		return c.cmdTools(ctx)
	case "/status":
		return c.cmdStatus(ctx, args)
	case "/instances":
		return c.cmdInstances(ctx)
	case "/instance":
		c.cmdSetInstance(args)
	case "/session":
		c.cmdSetSession(args)
	case "/discover":
		return c.cmdDiscover(ctx, args)
	case "/invoke":
		return c.cmdInvoke(ctx, args)
	case "/cron":
		return c.cmdCron(ctx, args)
	default:
		c.printf("Unknown command: %s (type /help)\n", cmd)
	}
	return nil
}

func (c *CLI) handleChat(ctx context.Context, message string) {
	args := map[string]any{
		"message": message,
	}
	if c.instance != "" {
		args["instance"] = c.instance
	}
	if c.sessionID != "" {
		args["session_id"] = c.sessionID
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "openclaw_chat",
		Arguments: args,
	})
	if err != nil {
		c.printf("Error: %v\n", err)
		return
	}

	c.printResult(result)
}

func (c *CLI) cmdTools(ctx context.Context) error {
	result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return err
	}
	c.printf("\nAvailable tools:\n")
	for _, tool := range result.Tools {
		c.printf("  %-25s %s\n", tool.Name, tool.Description)
	}
	c.printf("\n")
	return nil
}

func (c *CLI) cmdDiscover(ctx context.Context, args []string) error {
	toolArgs := map[string]any{}
	inst := flagValue(args, "--instance", "-i")
	if inst != "" {
		toolArgs["instance"] = inst
	} else if c.instance != "" {
		toolArgs["instance"] = c.instance
	}

	c.printf("Probing gateway for available tools...\n")
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "openclaw_discover",
		Arguments: toolArgs,
	})
	if err != nil {
		return err
	}

	// Pretty-print discovery results.
	text := extractText(result)
	var data struct {
		Instance  string `json:"instance"`
		Available []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"available"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(text), &data); err == nil {
		c.printf("\nAvailable tools on %s (%d):\n", data.Instance, data.Total)
		for _, t := range data.Available {
			c.printf("  %s\n", t.Name)
		}
		c.printf("\nUse: /invoke <tool> [--action ACTION] [--args JSON]\n\n")
	} else {
		c.printf("%s\n", text)
	}
	return nil
}

func (c *CLI) cmdStatus(ctx context.Context, args []string) error {
	toolArgs := map[string]any{}
	inst := flagValue(args, "--instance", "-i")
	if inst != "" {
		toolArgs["instance"] = inst
	} else if c.instance != "" {
		toolArgs["instance"] = c.instance
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "openclaw_status",
		Arguments: toolArgs,
	})
	if err != nil {
		return err
	}
	c.printResult(result)
	return nil
}

func (c *CLI) cmdInstances(ctx context.Context) error {
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "openclaw_instances",
	})
	if err != nil {
		return err
	}

	// Pretty-print instances.
	text := extractText(result)
	var data struct {
		Instances []struct {
			Name      string `json:"name"`
			URL       string `json:"url"`
			IsDefault bool   `json:"is_default"`
		} `json:"instances"`
	}
	if err := json.Unmarshal([]byte(text), &data); err == nil && len(data.Instances) > 0 {
		c.printf("\n")
		for _, inst := range data.Instances {
			marker := "  "
			if inst.IsDefault {
				marker = "* "
			}
			c.printf("  %s%-15s %s\n", marker, inst.Name, inst.URL)
		}
		active := c.instance
		if active == "" {
			active = "(default)"
		}
		c.printf("\n  Active: %s\n\n", active)
	} else {
		c.printf("%s\n", text)
	}
	return nil
}

func (c *CLI) cmdSetInstance(args []string) {
	if len(args) == 0 {
		c.instance = ""
		c.printf("Instance reset to default\n")
		return
	}
	c.instance = args[0]
	c.printf("Instance set to: %s\n", c.instance)
}

func (c *CLI) cmdSetSession(args []string) {
	if len(args) == 0 {
		if c.sessionID == "" {
			c.printf("No session set\n")
		} else {
			c.printf("Session: %s\n", c.sessionID)
		}
		return
	}
	c.sessionID = args[0]
	c.printf("Session set to: %s\n", c.sessionID)
}

func (c *CLI) cmdInvoke(ctx context.Context, args []string) error {
	if len(args) == 0 {
		c.printf("Usage: /invoke <tool> [--action ACTION] [--args JSON] [--session-key KEY] [--instance NAME]\n")
		return nil
	}

	toolArgs := map[string]any{
		"tool": args[0],
	}
	if v := flagValue(args, "--action", "-a"); v != "" {
		toolArgs["action"] = v
	}
	if v := flagValue(args, "--args"); v != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return fmt.Errorf("invalid --args JSON: %w", err)
		}
		toolArgs["args"] = parsed
	}
	if v := flagValue(args, "--session-key", "-s"); v != "" {
		toolArgs["session_key"] = v
	}
	inst := flagValue(args, "--instance", "-i")
	if inst != "" {
		toolArgs["instance"] = inst
	} else if c.instance != "" {
		toolArgs["instance"] = c.instance
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "openclaw_tool_invoke",
		Arguments: toolArgs,
	})
	if err != nil {
		return err
	}
	c.printResult(result)
	return nil
}

func (c *CLI) cmdCron(ctx context.Context, args []string) error {
	if len(args) == 0 {
		c.printf("Usage: /cron <action> [--job JSON] [--job-id ID] [--patch JSON] [--instance NAME]\n")
		c.printf("Actions: status, list, add, update, remove, run\n")
		return nil
	}

	toolArgs := map[string]any{
		"action": args[0],
	}
	if v := flagValue(args, "--job"); v != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return fmt.Errorf("invalid --job JSON: %w", err)
		}
		toolArgs["job"] = parsed
	}
	if v := flagValue(args, "--job-id"); v != "" {
		toolArgs["job_id"] = v
	}
	if v := flagValue(args, "--patch"); v != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return fmt.Errorf("invalid --patch JSON: %w", err)
		}
		toolArgs["patch"] = parsed
	}
	inst := flagValue(args, "--instance", "-i")
	if inst != "" {
		toolArgs["instance"] = inst
	} else if c.instance != "" {
		toolArgs["instance"] = c.instance
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "openclaw_cron",
		Arguments: toolArgs,
	})
	if err != nil {
		return err
	}
	c.printResult(result)
	return nil
}

func (c *CLI) printHelp() {
	c.printf(`
Commands:
  /help, /h                Show this help
  /tools                   List available MCP tools
  /discover [--instance X] Discover available OpenClaw tools on a worker
  /status [--instance X]   Health check
  /instances               List worker instances
  /instance [name]         Set active worker (empty = reset to default)
  /session [id]            Set/show chat session ID
  /invoke <tool> [flags]   Invoke any OpenClaw tool
    --action, -a ACTION      Tool action
    --args JSON              Tool arguments as JSON
    --session-key, -s KEY    Session context
    --instance, -i NAME      Target instance
  /cron <action> [flags]   Manage cron jobs (status, list, add, update, remove, run)
    --job JSON               Job definition (for add)
    --job-id ID              Job ID (for update, remove, run)
    --patch JSON             Patch fields (for update)
    --instance, -i NAME      Target instance
  /quit, /exit, /q         Exit

Everything else is sent as a chat message.
`)
}

func (c *CLI) printResult(result *mcp.CallToolResult) {
	text := extractText(result)
	if text == "" {
		c.printf("(empty response)\n")
		return
	}

	// Try to pretty-print if it's JSON with a "response" field (chat output).
	var chatOutput struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(text), &chatOutput); err == nil && chatOutput.Response != "" {
		c.printf("%s\n", chatOutput.Response)
		return
	}

	// Try to pretty-print other JSON.
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err == nil {
		pretty, err := json.MarshalIndent(raw, "", "  ")
		if err == nil {
			c.printf("%s\n", string(pretty))
			return
		}
	}

	c.printf("%s\n", text)
}

func (c *CLI) printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

// extractText pulls the first text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, content := range result.Content {
		if tc, ok := content.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	// Fallback: marshal content to JSON.
	if len(result.Content) > 0 {
		data, err := json.Marshal(result.Content)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// splitArgs splits a command line respecting single-quoted strings.
func splitArgs(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\'' && !inQuote:
			inQuote = true
		case ch == '\'' && inQuote:
			inQuote = false
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// flagValue extracts the value following a flag name from args.
// Supports multiple flag names (e.g., "--instance", "-i").
func flagValue(args []string, names ...string) string {
	for i, arg := range args {
		for _, name := range names {
			if arg == name && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}
