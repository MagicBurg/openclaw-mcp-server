package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/config"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

// newTestRegistry creates a registry pointing at the test server.
func newTestRegistry(t *testing.T, handler http.HandlerFunc) *openclaw.Registry {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return openclaw.NewRegistry([]config.InstanceConfig{
		{Name: "test", URL: srv.URL, Default: true},
		{Name: "other", URL: srv.URL},
	})
}

// --- Chat tests ---

func TestChatHandler_HappyPath(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openclaw.ChatCompletionResponse{
			Model:   "openclaw",
			Choices: []openclaw.ChatChoice{{Message: openclaw.ChatMessage{Content: "Hello!"}}},
			Usage:   &openclaw.ChatUsage{TotalTokens: 10},
		})
	})

	handler := ChatHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, ChatInput{Message: "Hi"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result")
	}
	if output.Response != "Hello!" {
		t.Errorf("response = %q", output.Response)
	}
	if output.Instance != "test" {
		t.Errorf("instance = %q", output.Instance)
	}
}

func TestChatHandler_EmptyMessage(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})
	handler := ChatHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ChatInput{Message: ""})
	if err == nil {
		t.Fatal("expected error for empty message")
	}
}

func TestChatHandler_UnknownInstance(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})
	handler := ChatHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ChatInput{Message: "Hi", Instance: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown instance")
	}
}

func TestChatHandler_GatewayError(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	handler := ChatHandler(registry)
	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ChatInput{Message: "Hi"})
	if err != nil {
		t.Fatalf("handler should not return error, got: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for gateway failure")
	}
}

func TestChatHandler_SpecificInstance(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openclaw.ChatCompletionResponse{
			Choices: []openclaw.ChatChoice{{Message: openclaw.ChatMessage{Content: "ok"}}},
		})
	})

	handler := ChatHandler(registry)
	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, ChatInput{Message: "Hi", Instance: "other"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if output.Instance != "other" {
		t.Errorf("instance = %q, want %q", output.Instance, "other")
	}
}

// --- ToolInvoke tests ---

func TestToolInvokeHandler_HappyPath(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openclaw.ToolInvokeResponse{
			OK:     true,
			Result: map[string]any{"data": "done"},
		})
	})

	handler := ToolInvokeHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, ToolInvokeInput{
		Tool:   "browser",
		Action: "snapshot",
		Args:   map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result")
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestToolInvokeHandler_EmptyTool(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})
	handler := ToolInvokeHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ToolInvokeInput{Tool: ""})
	if err == nil {
		t.Fatal("expected error for empty tool")
	}
}

func TestToolInvokeHandler_ToolError(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openclaw.ToolInvokeResponse{
			OK:    false,
			Error: &openclaw.ToolInvokeError{Message: "not found"},
		})
	})

	handler := ToolInvokeHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, ToolInvokeInput{Tool: "bad"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
	if output.Error != "not found" {
		t.Errorf("error = %q", output.Error)
	}
}

// --- Cron tests ---

func TestCronHandler_HappyPath(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		var req openclaw.ToolInvokeRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tool != "cron" {
			t.Errorf("tool = %q, want cron", req.Tool)
		}
		json.NewEncoder(w).Encode(openclaw.ToolInvokeResponse{OK: true, Result: "listed"})
	})

	handler := CronHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, CronInput{Action: "list"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result")
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestCronHandler_EmptyAction(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})
	handler := CronHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, CronInput{Action: ""})
	if err == nil {
		t.Fatal("expected error for empty action")
	}
}

func TestCronHandler_AddWithJob(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		var req openclaw.ToolInvokeRequest
		json.NewDecoder(r.Body).Decode(&req)
		args := req.Args
		if args["action"] != "add" {
			t.Errorf("action = %v", args["action"])
		}
		if args["job"] == nil {
			t.Error("job should not be nil")
		}
		json.NewEncoder(w).Encode(openclaw.ToolInvokeResponse{OK: true})
	})

	handler := CronHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, CronInput{
		Action: "add",
		Job:    map[string]any{"name": "test-job", "schedule": map[string]any{"kind": "cron", "expr": "0 9 * * *"}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

// --- Status tests ---

func TestStatusHandler_OK(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openclaw.HealthResponse{OK: true, Status: "ok"})
	})

	handler := StatusHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, StatusInput{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result")
	}
	if output.Status != "ok" {
		t.Errorf("status = %q", output.Status)
	}
	if output.Instance != "test" {
		t.Errorf("instance = %q", output.Instance)
	}
}

func TestStatusHandler_UnknownInstance(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})
	handler := StatusHandler(registry)
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, StatusInput{Instance: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown instance")
	}
}

// --- Instances tests ---

func TestInstancesHandler(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})

	handler := InstancesHandler(registry)
	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, InstancesInput{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result")
	}
	if output.Total != 2 {
		t.Errorf("total = %d, want 2", output.Total)
	}
	// Verify no tokens in output.
	for _, inst := range output.Instances {
		if inst.Name == "test" && !inst.IsDefault {
			t.Error("test should be default")
		}
	}
}

// --- Register tests ---

func TestRegisterAll(t *testing.T) {
	registry := newTestRegistry(t, func(w http.ResponseWriter, r *http.Request) {})

	server := mcp.NewServer(
		&mcp.Implementation{Name: "test", Version: "0.0.1"},
		nil,
	)
	RegisterAll(server, registry)
	// If we got here without panicking, registration succeeded.
}
