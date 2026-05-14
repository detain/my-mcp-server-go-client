// Package server provides transport layer for MCP proxy server.
package server

import (
	"log/slog"
	"net/http"
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
