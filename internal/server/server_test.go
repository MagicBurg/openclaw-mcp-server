package server

import (
	"testing"

	"github.com/weiboz/openclaw-mcp-server/internal/config"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
)

func TestNew(t *testing.T) {
	registry := openclaw.NewRegistry([]config.InstanceConfig{
		{Name: "test", URL: "http://localhost:18789", Default: true},
	})

	srv := New(registry)
	if srv == nil {
		t.Fatal("server is nil")
	}
}
