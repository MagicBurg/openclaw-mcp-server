package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// InstanceConfig defines a single OpenClaw gateway instance (worker).
type InstanceConfig struct {
	Name    string        `json:"name" toml:"name"`
	URL     string        `json:"url" toml:"url"`
	Token   string        `json:"token,omitempty" toml:"token,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty" toml:"-"`
	Default bool          `json:"default,omitempty" toml:"default,omitempty"`

	// TimeoutStr is used for TOML/JSON parsing, then converted to Timeout.
	TimeoutStr string `json:"-" toml:"timeout,omitempty"`
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

// fileConfig mirrors the TOML file structure.
type fileConfig struct {
	Server struct {
		Transport string `toml:"transport"`
		Host      string `toml:"host"`
		Port      int    `toml:"port"`
		AuthToken string `toml:"auth_token"`
	} `toml:"server"`

	Instances []InstanceConfig `toml:"instances"`
}

// configSearchPaths returns paths to search for config.toml, in priority order.
func configSearchPaths() []string {
	paths := []string{"config.toml"}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "openclaw-mcp-server", "config.toml"))
	}

	return paths
}

// Load builds a ServerConfig from config file, environment variables, and defaults.
// Precedence: CLI flags (applied after) > env vars > config file > defaults.
func Load(configPath string) (*ServerConfig, error) {
	cfg := &ServerConfig{
		Transport: "stdio",
		Host:      "0.0.0.0",
		Port:      8080,
	}

	// Step 1: Load from config file.
	if err := loadFile(cfg, configPath); err != nil {
		return nil, err
	}

	// Step 2: Override with environment variables.
	if err := applyEnv(cfg); err != nil {
		return nil, err
	}

	// Step 3: Validate.
	if len(cfg.Instances) == 0 {
		// No instances from file or OPENCLAW_INSTANCES env — fall back to single-instance env vars.
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

// LoadFromEnv builds a ServerConfig from environment variables only (no config file).
// Kept for backward compatibility.
func LoadFromEnv() (*ServerConfig, error) {
	return Load("")
}

func loadFile(cfg *ServerConfig, path string) error {
	// Find config file.
	if path == "" {
		for _, p := range configSearchPaths() {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return nil // No config file found — that's fine.
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}

	// Apply server settings.
	if fc.Server.Transport != "" {
		cfg.Transport = fc.Server.Transport
	}
	if fc.Server.Host != "" {
		cfg.Host = fc.Server.Host
	}
	if fc.Server.Port != 0 {
		cfg.Port = fc.Server.Port
	}
	if fc.Server.AuthToken != "" {
		cfg.AuthToken = fc.Server.AuthToken
	}

	// Apply instances — parse timeout strings.
	for i := range fc.Instances {
		inst := &fc.Instances[i]
		if inst.TimeoutStr != "" {
			d, err := time.ParseDuration(inst.TimeoutStr)
			if err != nil {
				return fmt.Errorf("config file: instance %q: invalid timeout %q: %w", inst.Name, inst.TimeoutStr, err)
			}
			inst.Timeout = d
		}
	}
	if len(fc.Instances) > 0 {
		cfg.Instances = fc.Instances
	}

	return nil
}

func applyEnv(cfg *ServerConfig) error {
	if v := os.Getenv("MCP_TRANSPORT"); v != "" {
		cfg.Transport = v
	}
	if v := os.Getenv("MCP_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("MCP_PORT"); v != "" {
		cfg.Port = envIntOr("MCP_PORT", cfg.Port)
	}
	if v := os.Getenv("MCP_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}

	// Multi-instance env var overrides file instances.
	if raw := os.Getenv("OPENCLAW_INSTANCES"); raw != "" {
		instances, err := parseInstances(raw)
		if err != nil {
			return fmt.Errorf("OPENCLAW_INSTANCES: %w", err)
		}
		cfg.Instances = instances
	}

	return nil
}

func parseInstances(raw string) ([]InstanceConfig, error) {
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

// Redact returns the config with token removed (for logging).
func (ic InstanceConfig) Redact() InstanceConfig {
	return InstanceConfig{
		Name:    ic.Name,
		URL:     ic.URL,
		Default: ic.Default,
		Timeout: ic.Timeout,
	}
}

// HasToken reports whether a non-empty token is set, without exposing the value.
func (ic InstanceConfig) HasToken() bool {
	return strings.TrimSpace(ic.Token) != ""
}
