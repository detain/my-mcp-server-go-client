// Package main is the entry point for the MCP Proxy Client server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/myadmin/go-mcp-proxy-client/internal/openapi"
	"github.com/myadmin/go-mcp-proxy-client/internal/oauth"
	"github.com/myadmin/go-mcp-proxy-client/internal/proxy"
	"github.com/myadmin/go-mcp-proxy-client/internal/server"
)

const (
	version     = "1.0.0"
	defaultName = "myadmin-client-mcp"
)

func main() {
	// Command line flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Load .env file if present
	_ = godotenv.Load()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Handle version flag
	if *showVersion {
		fmt.Printf("MCP Proxy Client %s\n", version)
		os.Exit(0)
	}

	// Get configuration from environment
	serverName := getEnv("SERVER_NAME", defaultName)
	serverVersion := getEnv("SERVER_VERSION", version)

	logger.Info("MCP Proxy Client starting...",
		slog.String("name", serverName),
		slog.String("version", serverVersion),
	)

	// Initialize server components
	cfg := server.Config{
		Name:    serverName,
		Version: serverVersion,
	}

	mcpServer, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Error("Failed to create MCP server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize supporting components (imported for go mod tidy)
	_ = openapi.NewParser(logger)
	_ = openapi.NewGenerator(logger)
	_ = openapi.NewCache(logger)
	_ = proxy.NewClient("", logger)
	_ = proxy.NewAuthenticator(logger)
	_ = oauth.NewProtectedResource(logger)

	// Start server
	ctx := context.Background()
	if err := mcpServer.Start(ctx); err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("MCP Proxy Client started successfully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
