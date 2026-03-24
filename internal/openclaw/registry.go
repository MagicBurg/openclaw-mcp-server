package openclaw

import (
	"fmt"

	"github.com/weiboz/openclaw-mcp-server/internal/config"
)

// ResolvedInstance pairs a name with its client.
type ResolvedInstance struct {
	Name   string
	Client *Client
}

// InstanceInfo is the public metadata for an instance (no secrets).
type InstanceInfo struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	IsDefault bool   `json:"is_default"`
}

// Registry manages multiple OpenClaw instances.
type Registry struct {
	clients     map[string]*Client
	defaultName string
	configs     []config.InstanceConfig
}

// NewRegistry creates a registry from validated instance configs.
func NewRegistry(instances []config.InstanceConfig) *Registry {
	r := &Registry{
		clients: make(map[string]*Client, len(instances)),
		configs: instances,
	}

	for _, inst := range instances {
		timeout := inst.Timeout
		if timeout <= 0 {
			timeout = config.DefaultTimeout()
		}
		r.clients[inst.Name] = NewClient(inst.URL, inst.Token, timeout)

		if inst.Default || r.defaultName == "" {
			r.defaultName = inst.Name
		}
	}

	return r
}

// Resolve returns the client for the given instance name.
// If name is empty, the default instance is returned.
func (r *Registry) Resolve(name string) (ResolvedInstance, error) {
	if name == "" {
		name = r.defaultName
	}
	client, ok := r.clients[name]
	if !ok {
		return ResolvedInstance{}, fmt.Errorf("unknown instance: %q", name)
	}
	return ResolvedInstance{Name: name, Client: client}, nil
}

// List returns public metadata for all instances.
func (r *Registry) List() []InstanceInfo {
	infos := make([]InstanceInfo, len(r.configs))
	for i, inst := range r.configs {
		infos[i] = InstanceInfo{
			Name:      inst.Name,
			URL:       inst.URL,
			IsDefault: inst.Name == r.defaultName,
		}
	}
	return infos
}

// DefaultName returns the name of the default instance.
func (r *Registry) DefaultName() string {
	return r.defaultName
}

// Size returns the number of registered instances.
func (r *Registry) Size() int {
	return len(r.clients)
}
