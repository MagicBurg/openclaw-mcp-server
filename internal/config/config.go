package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// InstanceConfig defines a single OpenClaw gateway instance (worker).
type InstanceConfig struct {
	Name    string        `json:"name"`
	URL     string        `json:"url"`
	Token   string        `json:"token,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
	Default bool          `json:"default,omitempty"`
}

// ServerConfig holds the full server configuration.
type ServerConfig struct {
	Instances []InstanceConfig
	Transport string // "stdio" or "http"
	Host      string
	Port      int
	AuthToken string // bearer token for MCP client auth (HTTP mode)
}

var instanceNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// LoadFromEnv builds a ServerConfig from environment variables.
// CLI flags should override the returned config after this call.
func LoadFromEnv() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Transport: envOr("MCP_TRANSPORT", "stdio"),
		Host:      envOr("MCP_HOST", "0.0.0.0"),
		Port:      envIntOr("MCP_PORT", 8080),
		AuthToken: os.Getenv("MCP_AUTH_TOKEN"),
	}

	// Multi-instance takes precedence.
	if raw := os.Getenv("OPENCLAW_INSTANCES"); raw != "" {
		instances, err := parseInstances(raw)
		if err != nil {
			return nil, fmt.Errorf("OPENCLAW_INSTANCES: %w", err)
		}
		cfg.Instances = instances
	} else {
		// Single-instance fallback.
		inst := InstanceConfig{
			Name:    "default",
			URL:     envOr("OPENCLAW_URL", "http://127.0.0.1:18789"),
			Token:   os.Getenv("OPENCLAW_TOKEN"),
			Default: true,
		}
		if t := os.Getenv("OPENCLAW_TIMEOUT"); t != "" {
			d, err := time.ParseDuration(t)
			if err != nil {
				return nil, fmt.Errorf("OPENCLAW_TIMEOUT: %w", err)
			}
			inst.Timeout = d
		}
		cfg.Instances = []InstanceConfig{inst}
	}

	if err := validateInstances(cfg.Instances); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseInstances(raw string) ([]InstanceConfig, error) {
	// Parse JSON with duration as string or milliseconds.
	var rawInstances []struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Token   string `json:"token,omitempty"`
		Timeout string `json:"timeout,omitempty"`
		Default bool   `json:"default,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &rawInstances); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	instances := make([]InstanceConfig, len(rawInstances))
	for i, r := range rawInstances {
		inst := InstanceConfig{
			Name:    r.Name,
			URL:     r.URL,
			Token:   r.Token,
			Default: r.Default,
		}
		if r.Timeout != "" {
			d, err := time.ParseDuration(r.Timeout)
			if err != nil {
				return nil, fmt.Errorf("instance %q: invalid timeout %q: %w", r.Name, r.Timeout, err)
			}
			inst.Timeout = d
		}
		instances[i] = inst
	}
	return instances, nil
}

func validateInstances(instances []InstanceConfig) error {
	if len(instances) == 0 {
		return fmt.Errorf("at least one instance is required")
	}

	names := make(map[string]bool)
	defaultCount := 0

	for _, inst := range instances {
		if !instanceNameRe.MatchString(inst.Name) {
			return fmt.Errorf("instance name %q is invalid: must be 1-64 alphanumeric chars, dashes, or underscores, starting with alphanumeric", inst.Name)
		}
		if names[inst.Name] {
			return fmt.Errorf("duplicate instance name: %q", inst.Name)
		}
		names[inst.Name] = true

		if err := validateURL(inst.URL); err != nil {
			return fmt.Errorf("instance %q: %w", inst.Name, err)
		}
		if inst.Default {
			defaultCount++
		}
	}

	if defaultCount > 1 {
		return fmt.Errorf("only one instance can be marked as default")
	}

	return nil
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https scheme", rawURL)
	}
	if u.Host == "" {
		return fmt.Errorf("URL %q must have a host", rawURL)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

// DefaultTimeout returns the default request timeout.
func DefaultTimeout() time.Duration {
	return 120 * time.Second
}

// Redact returns the URL with any userinfo removed (for logging).
func (ic InstanceConfig) Redact() InstanceConfig {
	return InstanceConfig{
		Name:    ic.Name,
		URL:     ic.URL,
		Default: ic.Default,
		Timeout: ic.Timeout,
		// Token intentionally omitted.
	}
}

// HasToken reports whether a non-empty token is set, without exposing the value.
func (ic InstanceConfig) HasToken() bool {
	return strings.TrimSpace(ic.Token) != ""
}
