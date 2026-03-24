package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/config"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
	"github.com/weiboz/openclaw-mcp-server/internal/server"
)

func main() {
	// CLI flags.
	configFile := flag.String("config", "", "Path to config.toml (default: ./config.toml or ~/.config/openclaw-mcp-server/config.toml)")
	transport := flag.String("transport", "", "Transport mode: stdio, http (env: MCP_TRANSPORT)")
	port := flag.Int("port", 0, "HTTP port (env: MCP_PORT)")
	host := flag.String("host", "", "HTTP bind address (env: MCP_HOST)")
	openclawURL := flag.String("openclaw-url", "", "Single instance URL (env: OPENCLAW_URL)")
	openclawToken := flag.String("openclaw-token", "", "Single instance token (env: OPENCLAW_TOKEN)")
	authToken := flag.String("auth-token", "", "Bearer token for MCP client auth (env: MCP_AUTH_TOKEN)")
	flag.Parse()

	// Apply CLI flag overrides to env before loading config.
	setEnvIfFlag("MCP_TRANSPORT", *transport)
	setEnvIfFlag("MCP_HOST", *host)
	if *port != 0 {
		os.Setenv("MCP_PORT", fmt.Sprintf("%d", *port))
	}
	setEnvIfFlag("OPENCLAW_URL", *openclawURL)
	setEnvIfFlag("OPENCLAW_TOKEN", *openclawToken)
	setEnvIfFlag("MCP_AUTH_TOKEN", *authToken)

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	registry := openclaw.NewRegistry(cfg.Instances)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Probe all instances at startup.
	log.Printf("probing %d instance(s)...", len(cfg.Instances))
	registry.ProbeAll(ctx)

	switch cfg.Transport {
	case "stdio":
		runStdio(ctx, registry)
	case "http":
		runHTTP(ctx, cfg, registry)
	default:
		log.Fatalf("unknown transport: %q", cfg.Transport)
	}
}

func runStdio(ctx context.Context, registry *openclaw.Registry) {
	srv := server.New(registry)
	log.Printf("starting %s v%s (stdio)", server.ServerName, server.ServerVersion)
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runHTTP(ctx context.Context, cfg *config.ServerConfig, registry *openclaw.Registry) {
	srv := server.New(registry)

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return srv },
		nil,
	)

	var finalHandler http.Handler = handler
	if cfg.AuthToken != "" {
		finalHandler = bearerAuthMiddleware(cfg.AuthToken, handler)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/mcp", finalHandler)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		log.Println("shutting down HTTP server...")
		httpServer.Close()
	}()

	log.Printf("starting %s v%s (http) on %s", server.ServerName, server.ServerVersion, addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func bearerAuthMiddleware(expectedToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+expectedToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setEnvIfFlag(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	}
}
