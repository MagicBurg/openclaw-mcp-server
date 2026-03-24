package openclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChat_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type")
		}

		json.NewEncoder(w).Encode(ChatCompletionResponse{
			ID:    "chatcmpl-123",
			Model: "openclaw",
			Choices: []ChatChoice{
				{Index: 0, Message: ChatMessage{Role: "assistant", Content: "Hello!"}, FinishReason: "stop"},
			},
			Usage: &ChatUsage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	result, err := c.Chat(context.Background(), "Hi", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Response != "Hello!" {
		t.Errorf("response = %q", result.Response)
	}
	if result.Usage.TotalTokens != 8 {
		t.Errorf("total tokens = %d", result.Usage.TotalTokens)
	}
}

func TestChat_WithSessionID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-openclaw-session-key") != "my-session" {
			t.Errorf("session header = %q", r.Header.Get("x-openclaw-session-key"))
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []ChatChoice{{Message: ChatMessage{Content: "ok"}}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	_, err := c.Chat(context.Background(), "Hi", "my-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChat_WithToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []ChatChoice{{Message: ChatMessage{Content: "ok"}}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sk-test", 5*time.Second)
	_, err := c.Chat(context.Background(), "Hi", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChat_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	_, err := c.Chat(context.Background(), "Hi", "")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestChat_ClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	_, err := c.Chat(context.Background(), "Hi", "")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestChat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: []ChatChoice{}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	_, err := c.Chat(context.Background(), "Hi", "")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestToolInvoke_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tools/invoke" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req ToolInvokeRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tool != "browser" || req.Action != "snapshot" {
			t.Errorf("req = %+v", req)
		}

		json.NewEncoder(w).Encode(ToolInvokeResponse{
			OK:     true,
			Result: map[string]any{"text": "done"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	resp, err := c.ToolInvoke(context.Background(), ToolInvokeRequest{
		Tool:   "browser",
		Action: "snapshot",
		Args:   map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.OK {
		t.Error("expected ok=true")
	}
}

func TestToolInvoke_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ToolInvokeResponse{
			OK:    false,
			Error: &ToolInvokeError{Type: "validation", Message: "bad input"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	resp, err := c.ToolInvoke(context.Background(), ToolInvokeRequest{Tool: "bad"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OK {
		t.Error("expected ok=false")
	}
	if resp.Error.Message != "bad input" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
}

func TestHealth_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(HealthResponse{OK: true, Status: "ok"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	status, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "ok" {
		t.Errorf("status = %q", status)
	}
}

func TestHealth_ServerDown(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "", 1*time.Second)
	status, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if status != "error" {
		t.Errorf("status = %q, want %q", status, "error")
	}
}

func TestNewClient_TrailingSlash(t *testing.T) {
	c := NewClient("http://example.com/", "", 0)
	if c.baseURL != "http://example.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	c := NewClient("http://example.com", "", 0)
	if c.timeout != 120*time.Second {
		t.Errorf("timeout = %v", c.timeout)
	}
}
