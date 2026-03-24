package openclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
	defaultMaxTokens = 4096
)

// Client communicates with an OpenClaw gateway over HTTP.
type Client struct {
	baseURL    string
	token      string
	timeout    time.Duration
	httpClient *http.Client
}

// NewClient creates a new OpenClaw client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Chat sends a message via the OpenAI-compatible chat completions endpoint.
func (c *Client) Chat(ctx context.Context, message, sessionID string) (*ChatResult, error) {
	req := ChatRequest{
		Model:     "openclaw",
		Messages:  []ChatMessage{{Role: "user", Content: message}},
		MaxTokens: defaultMaxTokens,
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/v1/chat/completions", req, sessionID)
	if err != nil {
		return nil, err
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse chat response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in chat response")
	}

	return &ChatResult{
		Response: resp.Choices[0].Message.Content,
		Model:    resp.Model,
		Usage:    resp.Usage,
	}, nil
}

// ToolInvoke calls POST /tools/invoke on the gateway.
func (c *Client) ToolInvoke(ctx context.Context, toolReq ToolInvokeRequest) (*ToolInvokeResponse, error) {
	body, err := c.doJSON(ctx, http.MethodPost, "/tools/invoke", toolReq, "")
	if err != nil {
		return nil, err
	}

	var resp ToolInvokeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse tool invoke response: %w", err)
	}

	return &resp, nil
}

// Health checks the gateway's health endpoint.
func (c *Client) Health(ctx context.Context) (string, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/health", nil, "")
	if err != nil {
		return "error", err
	}

	var resp HealthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// If we got a response but can't parse it, gateway is alive.
		return "ok", nil
	}

	if resp.Status != "" {
		return resp.Status, nil
	}
	if resp.OK {
		return "ok", nil
	}
	return "ok", nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, sessionID string) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return c.doRequest(ctx, method, path, bytes.NewReader(data), sessionID)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, sessionID string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if sessionID != "" {
		req.Header.Set("x-openclaw-session-key", sessionID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", path, err)
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("gateway error %d from %s: %s", resp.StatusCode, path, truncate(string(respBody), 200))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("client error %d from %s: %s", resp.StatusCode, path, truncate(string(respBody), 200))
	}

	return respBody, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
