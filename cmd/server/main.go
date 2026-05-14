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
	"github.com/myadmin/go-mcp-proxy-client/internal/config"
	"github.com/myadmin/go-mcp-proxy-client/internal/oauth"
	"github.com/myadmin/go-mcp-proxy-client/internal/server"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "", "Path to .env configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Handle version flag
	if *showVersion {
		cfg, err := config.Load(*configPath)
		if err != nil {
			fmt.Printf("MCP Proxy Client %s\n", config.DefaultServerVersion)
		} else {
			fmt.Printf("MCP Proxy Client %s\n", cfg.ServerVersion)
		}
		os.Exit(0)
	}

	// Load configuration from .env file and environment variables
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("MCP Proxy Client starting...",
		slog.String("name", cfg.ServerName),
		slog.String("version", cfg.ServerVersion),
		slog.String("openapiSpecURL", maskURL(cfg.OpenAPISpecURL)),
		slog.String("apiBaseURL", maskURL(cfg.APIBaseURL)),
	)

	// Create server config from application config
	serverConfig := server.Config{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}

	mcpServer, err := server.NewServer(serverConfig, logger)
	if err != nil {
		logger.Error("Failed to create MCP server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Detect transport mode
	transport := server.DetectTransport()
	logger.Info("Transport mode detected", slog.String("transport", transport))

	// Configure OAuth protected resource metadata
	protectedResource := oauth.NewProtectedResource(logger)
	protectedResource.Configure(cfg.NormalizeAPIBaseURL(), cfg.GetOAuthAuthorizationServer())

	// Start server based on transport mode
	ctx := context.Background()

	if transport == "stdio" {
		// STDIO mode - serve MCP over stdin/stdout
		logger.Info("Starting in STDIO mode",
			slog.Bool("hasBearerToken", cfg.BearerToken != ""),
			slog.Bool("hasAPIKey", cfg.APIKey != ""),
			slog.Bool("hasSessionID", cfg.SessionID != ""),
		)
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

		logger.Info("Starting HTTP server on :8080",
			slog.String("apiBaseURL", cfg.APIBaseURL),
		)

		if err := router.Run(":8080"); err != nil {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	<-ctx.Done()
	logger.Info("MCP Proxy Client shutting down")
}

// maskURL returns a masked URL for logging (shows scheme and host only).
func maskURL(rawURL string) string {
	if rawURL == "" {
		return "(not set)"
	}
	// Simple masking: just show the scheme and host
	if len(rawURL) > 50 {
		return rawURL[:50] + "..."
	}
	return rawURL
}
