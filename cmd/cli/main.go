package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/weiboz/openclaw-mcp-server/internal/cli"
	"github.com/weiboz/openclaw-mcp-server/internal/config"
	"github.com/weiboz/openclaw-mcp-server/internal/openclaw"
	"github.com/weiboz/openclaw-mcp-server/internal/server"
)

func main() {
	configFile := flag.String("config", "", "Path to config.toml")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	registry := openclaw.NewRegistry(cfg.Instances)

	// Probe instances at startup.
	log.Printf("probing %d instance(s)...", len(cfg.Instances))
	registry.ProbeAll(context.Background())

	srv := server.New(registry)

	// Create in-memory transport pair.
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start server on its transport.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Run(ctx, serverTransport)
	}()

	// Connect client.
	client := mcp.NewClient(
		&mcp.Implementation{Name: "openclaw-mcp-cli", Version: "0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatalf("failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Run the interactive CLI.
	repl := cli.New(session, registry, os.Stdin, os.Stdout)
	if err := repl.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	cancel()
	<-serverDone
}
