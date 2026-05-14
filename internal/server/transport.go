// Package server provides transport layer for MCP proxy server.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Transport handles HTTP transport configuration.
type Transport struct {
	logger *slog.Logger
}

// NewTransport creates a new Transport instance.
func NewTransport(logger *slog.Logger) *Transport {
	return &Transport{logger: logger}
}

// ServeHTTP implements the http.Handler interface.
func (t *Transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.logger.Debug("ServeHTTP called",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)
}

// DetectTransport detects the transport mode based on stdin state.
// Returns "http" if TTY is connected (interactive mode), "stdio" otherwise.
func DetectTransport() string {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "stdio"
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "http"
	}
	return "stdio"
}

// CreateStreamableHTTPHandler creates an HTTP handler for the MCP server.
func CreateStreamableHTTPHandler(server *mcp.Server, logger *slog.Logger) *mcp.StreamableHTTPHandler {
	logger.Debug("Creating Streamable HTTP handler")
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Logger: logger,
	})
}

// ServeStdio serves the MCP server over stdio.
func ServeStdio(server *mcp.Server, logger *slog.Logger) error {
	logger.Info("Starting MCP server in STDIO mode")
	transport := &mcp.StdioTransport{}
	return server.Run(context.Background(), transport)
}
