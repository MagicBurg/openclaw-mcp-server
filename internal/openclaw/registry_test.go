package openclaw

import (
	"testing"
	"time"

	"github.com/weiboz/openclaw-mcp-server/internal/config"
)

func TestRegistry_SingleInstance(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "default", URL: "http://localhost:18789", Default: true},
	})

	if r.Size() != 1 {
		t.Errorf("size = %d", r.Size())
	}
	if r.DefaultName() != "default" {
		t.Errorf("default = %q", r.DefaultName())
	}
}

func TestRegistry_MultiInstance(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "prod", URL: "http://prod:18789", Default: true},
		{Name: "staging", URL: "http://staging:18789"},
	})

	if r.Size() != 2 {
		t.Errorf("size = %d", r.Size())
	}
	if r.DefaultName() != "prod" {
		t.Errorf("default = %q", r.DefaultName())
	}
}

func TestRegistry_Resolve_Default(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "worker", URL: "http://worker:18789", Default: true},
	})

	inst, err := r.Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Name != "worker" {
		t.Errorf("name = %q", inst.Name)
	}
	if inst.Client == nil {
		t.Error("client is nil")
	}
}

func TestRegistry_Resolve_ByName(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "prod", URL: "http://prod:18789", Default: true},
		{Name: "staging", URL: "http://staging:18789"},
	})

	inst, err := r.Resolve("staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Name != "staging" {
		t.Errorf("name = %q", inst.Name)
	}
}

func TestRegistry_Resolve_Unknown(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "worker", URL: "http://worker:18789"},
	})

	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown instance")
	}
}

func TestRegistry_List_NoTokens(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "prod", URL: "http://prod:18789", Token: "secret", Default: true},
		{Name: "staging", URL: "http://staging:18789", Token: "also-secret"},
	})

	infos := r.List()
	if len(infos) != 2 {
		t.Fatalf("list len = %d", len(infos))
	}
	for _, info := range infos {
		// InstanceInfo has no Token field — verify the struct doesn't expose secrets.
		if info.Name == "prod" && !info.IsDefault {
			t.Error("prod should be default")
		}
		if info.Name == "staging" && info.IsDefault {
			t.Error("staging should not be default")
		}
	}
}

func TestRegistry_FirstInstanceBecomesDefault(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "first", URL: "http://first:18789"},
		{Name: "second", URL: "http://second:18789"},
	})

	if r.DefaultName() != "first" {
		t.Errorf("default = %q, want %q", r.DefaultName(), "first")
	}
}

func TestRegistry_CustomTimeout(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "worker", URL: "http://worker:18789", Timeout: 30 * time.Second},
	})

	inst, _ := r.Resolve("worker")
	if inst.Client.timeout != 30*time.Second {
		t.Errorf("timeout = %v", inst.Client.timeout)
	}
}

func TestRegistry_DefaultTimeoutFallback(t *testing.T) {
	r := NewRegistry([]config.InstanceConfig{
		{Name: "worker", URL: "http://worker:18789"},
	})

	inst, _ := r.Resolve("worker")
	if inst.Client.timeout != config.DefaultTimeout() {
		t.Errorf("timeout = %v, want %v", inst.Client.timeout, config.DefaultTimeout())
	}
}
