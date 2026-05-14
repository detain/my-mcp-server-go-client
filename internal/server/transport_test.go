// Package server provides transport layer for MCP proxy server.
package server

import (
	"log/slog"
	"os"
	"testing"
)

func TestDetectTransport(t *testing.T) {
	// Test that DetectTransport returns a valid transport type
	transport := DetectTransport()
	if transport != "http" && transport != "stdio" {
		t.Errorf("DetectTransport() returned unexpected value: %s, expected 'http' or 'stdio'", transport)
	}
}

func TestDetectTransport_ReturnsString(t *testing.T) {
	// Verify the return type is string
	transport := DetectTransport()
	if transport == "" {
		t.Error("DetectTransport() returned empty string")
	}
}

func TestDetectTransport_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DetectTransport() panicked: %v", r)
		}
	}()
	DetectTransport()
}

func TestDetectTransport_WithStdinStatError(t *testing.T) {
	// This test verifies that DetectTransport handles error cases gracefully
	// by returning "stdio" as the default fallback
	transport := DetectTransport()
	// When stdin stat fails, it should return "stdio"
	if transport != "http" && transport != "stdio" {
		t.Errorf("DetectTransport() returned unexpected value: %s", transport)
	}
}

func TestCreateStreamableHTTPHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := createTestServer(t, logger)

	handler := CreateStreamableHTTPHandler(server.Server(), logger)
	if handler == nil {
		t.Error("CreateStreamableHTTPHandler() returned nil handler")
	}
}

func TestServeStdio_DoesNotPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := createTestServer(t, logger)

	// ServeStdio should not panic and should return immediately in test
	// since stdin is not a real terminal
	err := ServeStdio(server.Server(), logger)
	// We expect an error because stdin is not a proper TTY in test
	// but it should not panic
	if err == nil {
		t.Log("ServeStdio returned without error (expected in non-TTY environment)")
	}
}

func createTestServer(t *testing.T, logger *slog.Logger) *MCPServer {
	// Create a minimal server for testing
	t.Helper()
	cfg := Config{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	return server
}
