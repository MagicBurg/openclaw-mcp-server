package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OPENCLAW_URL", "OPENCLAW_TOKEN", "OPENCLAW_TIMEOUT",
		"OPENCLAW_INSTANCES", "MCP_TRANSPORT", "MCP_HOST",
		"MCP_PORT", "MCP_AUTH_TOKEN",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	clearEnv(t)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Transport != "stdio" {
		t.Errorf("transport = %q, want %q", cfg.Transport, "stdio")
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.Port != 8080 {
		t.Errorf("port = %d, want %d", cfg.Port, 8080)
	}
	if len(cfg.Instances) != 1 {
		t.Fatalf("instances count = %d, want 1", len(cfg.Instances))
	}
	inst := cfg.Instances[0]
	if inst.Name != "default" {
		t.Errorf("instance name = %q, want %q", inst.Name, "default")
	}
	if inst.URL != "http://127.0.0.1:18789" {
		t.Errorf("instance URL = %q, want default", inst.URL)
	}
	if !inst.Default {
		t.Error("instance should be default")
	}
}

func TestLoadFromEnv_SingleInstance(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENCLAW_URL", "http://10.0.0.1:18789")
	t.Setenv("OPENCLAW_TOKEN", "sk-test")
	t.Setenv("OPENCLAW_TIMEOUT", "30s")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inst := cfg.Instances[0]
	if inst.URL != "http://10.0.0.1:18789" {
		t.Errorf("URL = %q", inst.URL)
	}
	if inst.Token != "sk-test" {
		t.Errorf("token = %q", inst.Token)
	}
	if inst.Timeout != 30*time.Second {
		t.Errorf("timeout = %v", inst.Timeout)
	}
}

func TestLoadFromEnv_MultiInstance(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENCLAW_INSTANCES", `[
		{"name": "prod", "url": "http://10.0.0.1:18789", "token": "sk-1", "default": true},
		{"name": "staging", "url": "http://10.0.0.2:18789", "token": "sk-2", "timeout": "60s"}
	]`)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Instances) != 2 {
		t.Fatalf("instances = %d, want 2", len(cfg.Instances))
	}
	if cfg.Instances[0].Name != "prod" || !cfg.Instances[0].Default {
		t.Errorf("first instance: %+v", cfg.Instances[0])
	}
	if cfg.Instances[1].Timeout != 60*time.Second {
		t.Errorf("second instance timeout = %v", cfg.Instances[1].Timeout)
	}
}

func TestLoadFromEnv_MultiInstanceOverridesSingle(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENCLAW_URL", "http://should-be-ignored:18789")
	t.Setenv("OPENCLAW_INSTANCES", `[{"name": "worker", "url": "http://10.0.0.1:18789"}]`)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Instances) != 1 || cfg.Instances[0].Name != "worker" {
		t.Errorf("multi-instance should override single: %+v", cfg.Instances)
	}
}

func TestLoadFromEnv_InvalidTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENCLAW_TIMEOUT", "notaduration")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestLoadFromEnv_InvalidInstancesJSON(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENCLAW_INSTANCES", "not json")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateInstances_DuplicateNames(t *testing.T) {
	err := validateInstances([]InstanceConfig{
		{Name: "worker", URL: "http://a:1"},
		{Name: "worker", URL: "http://b:2"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate names")
	}
}

func TestValidateInstances_InvalidName(t *testing.T) {
	err := validateInstances([]InstanceConfig{
		{Name: "-bad", URL: "http://a:1"},
	})
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestValidateInstances_InvalidURL(t *testing.T) {
	err := validateInstances([]InstanceConfig{
		{Name: "worker", URL: "ftp://bad:1"},
	})
	if err == nil {
		t.Fatal("expected error for non-http URL")
	}
}

func TestValidateInstances_MultipleDefaults(t *testing.T) {
	err := validateInstances([]InstanceConfig{
		{Name: "a", URL: "http://a:1", Default: true},
		{Name: "b", URL: "http://b:1", Default: true},
	})
	if err == nil {
		t.Fatal("expected error for multiple defaults")
	}
}

func TestValidateInstances_Empty(t *testing.T) {
	err := validateInstances(nil)
	if err == nil {
		t.Fatal("expected error for empty instances")
	}
}

func TestInstanceConfig_Redact(t *testing.T) {
	ic := InstanceConfig{Name: "prod", URL: "http://a:1", Token: "secret"}
	r := ic.Redact()
	if r.Token != "" {
		t.Error("redacted config should not contain token")
	}
	if r.Name != "prod" {
		t.Error("redacted config should preserve name")
	}
}

func TestInstanceConfig_HasToken(t *testing.T) {
	if (InstanceConfig{Token: ""}).HasToken() {
		t.Error("empty token should return false")
	}
	if (InstanceConfig{Token: "  "}).HasToken() {
		t.Error("whitespace token should return false")
	}
	if !(InstanceConfig{Token: "sk-test"}).HasToken() {
		t.Error("non-empty token should return true")
	}
}

func TestLoadFromEnv_ServerConfig(t *testing.T) {
	clearEnv(t)
	t.Setenv("MCP_TRANSPORT", "http")
	t.Setenv("MCP_HOST", "127.0.0.1")
	t.Setenv("MCP_PORT", "9090")
	t.Setenv("MCP_AUTH_TOKEN", "my-token")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Transport != "http" {
		t.Errorf("transport = %q", cfg.Transport)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("host = %q", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("port = %d", cfg.Port)
	}
	if cfg.AuthToken != "my-token" {
		t.Errorf("auth token = %q", cfg.AuthToken)
	}
}
