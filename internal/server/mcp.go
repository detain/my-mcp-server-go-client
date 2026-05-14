// Package server provides MCP server setup and configuration.
package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PaginationLimit is the maximum number of items per page.
const PaginationLimit = 1000

// Config holds server configuration.
type Config struct {
	Name    string
	Version string
}

// NewServer creates a new MCP server instance.
func NewServer(cfg Config, logger *slog.Logger) (*MCPServer, error) {
	// Guard: Validate config
	if cfg.Name == "" {
		return nil, errors.New("server name is required")
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}

	logger.Debug("Creating MCP server",
		slog.String("name", cfg.Name),
		slog.String("version", cfg.Version),
		slog.Int("paginationLimit", PaginationLimit),
	)

	// Create server implementation info
	impl := &mcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}

	// Create server options
	options := &mcp.ServerOptions{
		Logger:   logger,
		PageSize: PaginationLimit,
	}

	server := mcp.NewServer(impl, options)

	return &MCPServer{
		config:    cfg,
		logger:    logger,
		mcpServer: server,
	}, nil
}

// MCPServer represents the MCP server.
type MCPServer struct {
	config    Config
	logger    *slog.Logger
	mcpServer *mcp.Server
}

// Server returns the underlying MCP server instance.
func (s *MCPServer) Server() *mcp.Server {
	return s.mcpServer
}

// Start starts the MCP server.
func (s *MCPServer) Start(ctx context.Context) error {
	s.logger.Info("MCP server starting",
		slog.String("name", s.config.Name),
		slog.String("version", s.config.Version),
	)
	// Server is started via transport (stdio or HTTP) - no additional start needed
	return nil
}
