// Package server provides MCP server setup and configuration.
package server

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewServer_CreatesServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() returned error: %v", err)
	}
	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}
}

func TestNewServer_SetsConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name:    "my-test-server",
		Version: "2.0.0",
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() returned error: %v", err)
	}
	if server.config.Name != cfg.Name {
		t.Errorf("Server config.Name = %s, want %s", server.config.Name, cfg.Name)
	}
	if server.config.Version != cfg.Version {
		t.Errorf("Server config.Version = %s, want %s", server.config.Version, cfg.Version)
	}
}

func TestNewServer_ReturnsServerWithMCPImplementation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() returned error: %v", err)
	}
	if server.Server() == nil {
		t.Error("Server.Server() returned nil MCP server")
	}
}

func TestNewServer_DefaultVersion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name: "test-server",
		// Version not set - should default to "1.0.0"
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() returned error: %v", err)
	}
	if server.config.Version != "1.0.0" {
		t.Errorf("Server config.Version = %s, want '1.0.0' (default)", server.config.Version)
	}
}

func TestNewServer_RequiresName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name: "", // Empty name should fail
	}

	_, err := NewServer(cfg, logger)
	if err == nil {
		t.Error("NewServer() should return error for empty name")
	}
}

func TestMCPServer_Start(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() returned error: %v", err)
	}

	// Start should not error
	err = server.Start(nil)
	if err != nil {
		t.Errorf("Start() returned error: %v", err)
	}
}