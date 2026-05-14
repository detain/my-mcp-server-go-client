// Package main is the entry point for the MCP Proxy Client server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/myadmin/go-mcp-proxy-client/internal/oauth"
	"github.com/myadmin/go-mcp-proxy-client/internal/server"
)

const (
	version           = "1.0.0"
	defaultName       = "myadmin-client-mcp"
	defaultAuthServer = "https://auth.example.com"
	defaultServerURL  = "http://localhost:8080"
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
	authServerURL := getEnv("AUTH_SERVER_URL", defaultAuthServer)
	serverURL := getEnv("SERVER_URL", defaultServerURL)

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

	// Detect transport mode
	transport := server.DetectTransport()
	logger.Info("Transport mode detected", slog.String("transport", transport))

	// Configure OAuth protected resource metadata
	protectedResource := oauth.NewProtectedResource(logger)
	protectedResource.Configure(serverURL, authServerURL)

	// Start server based on transport mode
	ctx := context.Background()

	if transport == "stdio" {
		// STDIO mode - serve MCP over stdin/stdout
		if err := server.ServeStdio(mcpServer.Server(), logger); err != nil {
			logger.Error("STDIO server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	} else {
		// HTTP mode - use Gin router
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.Use(gin.Recovery())

		// OAuth 2.1 Protected Resource Metadata endpoint
		router.GET("/.well-known/oauth-protected-resource", gin.WrapH(http.HandlerFunc(protectedResource.GetMetadata)))

		// MCP Streamable HTTP handler
		mcpHandler := server.CreateStreamableHTTPHandler(mcpServer.Server(), logger)
		router.Any("/mcp", gin.WrapH(mcpHandler))

		logger.Info("Starting HTTP server on :8080")

		if err := router.Run(":8080"); err != nil {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	<-ctx.Done()
	logger.Info("MCP Proxy Client shutting down")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}