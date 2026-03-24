package openclaw

// ChatRequest is the OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model      string          `json:"model"`
	Messages   []ChatMessage   `json:"messages"`
	MaxTokens  int             `json:"max_tokens,omitempty"`
	Stream     bool            `json:"stream,omitempty"`
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse is the OpenAI-compatible response.
type ChatCompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []ChatChoice       `json:"choices"`
	Usage   *ChatUsage         `json:"usage,omitempty"`
}

// ChatChoice is a single choice in the completion response.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatUsage tracks token consumption.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResult is the simplified response returned to MCP tool handlers.
type ChatResult struct {
	Response string     `json:"response"`
	Model    string     `json:"model,omitempty"`
	Usage    *ChatUsage `json:"usage,omitempty"`
}

// ToolInvokeRequest is the body for POST /tools/invoke.
type ToolInvokeRequest struct {
	Tool       string         `json:"tool"`
	Action     string         `json:"action,omitempty"`
	Args       map[string]any `json:"args,omitempty"`
	SessionKey string         `json:"sessionKey,omitempty"`
	DryRun     bool           `json:"dryRun,omitempty"`
}

// ToolInvokeResponse is the response from POST /tools/invoke.
type ToolInvokeResponse struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  *ToolInvokeError `json:"error,omitempty"`
}

// ToolInvokeError carries error details from a failed tool invocation.
type ToolInvokeError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message"`
}

// HealthResponse from GET /health.
type HealthResponse struct {
	OK     bool   `json:"ok,omitempty"`
	Status string `json:"status,omitempty"`
}

// ToolAvailability reports whether a tool is available on a gateway.
type ToolAvailability struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "available", "not_found", "error"
}

// ToolsCatalogResult is the response from tools.catalog WebSocket RPC.
type ToolsCatalogResult struct {
	AgentID  string           `json:"agentId"`
	Profiles []ToolProfile    `json:"profiles"`
	Groups   []ToolGroup      `json:"groups"`
}

// ToolProfile is a tool profile (e.g., minimal, coding, full).
type ToolProfile struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ToolGroup is a group of related tools.
type ToolGroup struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Source   string     `json:"source"` // "core" or "plugin"
	PluginID string     `json:"pluginId,omitempty"`
	Tools    []ToolEntry `json:"tools"`
}

// ToolEntry is a single tool in a group.
type ToolEntry struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Description     string   `json:"description"`
	Source          string   `json:"source"`
	PluginID        string   `json:"pluginId,omitempty"`
	DefaultProfiles []string `json:"defaultProfiles"`
}

// SkillsStatusResult is the response from skills.status WebSocket RPC.
type SkillsStatusResult struct {
	WorkspaceDir    string        `json:"workspaceDir"`
	ManagedSkillsDir string       `json:"managedSkillsDir"`
	Skills          []SkillStatus `json:"skills"`
}

// SkillStatus is the status of a single skill.
type SkillStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Emoji       string `json:"emoji,omitempty"`
	Disabled    bool   `json:"disabled"`
	Eligible    bool   `json:"eligible"`
	PrimaryEnv  string `json:"primaryEnv,omitempty"`
}
