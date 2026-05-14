package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewParser(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	if parser == nil {
		t.Fatal("NewParser returned nil")
	}

	if parser.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewParserWithClient(t *testing.T) {
	logger := slog.Default()
	client := &http.Client{Timeout: 10}
	parser := NewParserWithClient(logger, client)

	if parser == nil {
		t.Fatal("NewParserWithClient returned nil")
	}

	if parser.httpClient != client {
		t.Error("httpClient does not match provided client")
	}
}

func TestParser_Parse_InvalidURL(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	tests := []struct {
		name    string
		specURL string
	}{
		{"invalid scheme", "ftp://example.com/spec.yaml"},
		{"missing host", "http:///spec.yaml"},
		{"parse error", "not-a-url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tt.specURL)
			if err == nil {
				t.Error("Expected error for invalid URL, got nil")
			}
		})
	}
}

func TestParser_Parse_EmptyResponse(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
	}))
	defer server.Close()

	_, err := parser.Parse(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected error for empty response, got nil")
	}
}

func TestParser_Parse_JSONSpec(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	spec := createTestOpenAPISpec()
	specJSON, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal test spec: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(specJSON)
	}))
	defer server.Close()

	result, err := parser.Parse(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	if result.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", result.Title)
	}

	if result.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", result.Version)
	}

	if result.SpecV3 == nil {
		t.Error("SpecV3 should not be nil")
	}

	if result.SpecV3.Paths.Len() != 1 {
		t.Errorf("Expected 1 path, got %d", result.SpecV3.Paths.Len())
	}
}

func TestParser_Parse_YAMLSpec(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	// YAML spec as a simple string
	yamlSpec := `
openapi: "3.0.0"
info:
  title: "YAML Test API"
  version: "2.0.0"
paths:
  /test:
    get:
      summary: "Test endpoint"
      operationId: "testGet"
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(yamlSpec))
	}))
	defer server.Close()

	result, err := parser.Parse(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Failed to parse YAML spec: %v", err)
	}

	if result.Title != "YAML Test API" {
		t.Errorf("Expected title 'YAML Test API', got %q", result.Title)
	}

	if result.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %q", result.Version)
	}
}

func TestParser_Parse_HTTPError(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := parser.Parse(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected error for HTTP 404, got nil")
	}
}

func TestParser_Parse_InvalidJSON(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json or yaml"))
	}))
	defer server.Close()

	_, err := parser.Parse(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected error for invalid content, got nil")
	}
}

func TestParser_GetRemoteSpecAge(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2025 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	age, err := parser.GetRemoteSpecAge(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetRemoteSpecAge failed: %v", err)
	}

	if age == nil {
		t.Fatal("Expected non-nil age")
	}

	// Last-Modified parsed above should be 2025-10-21
	if age.Year() != 2025 || age.Month() != 10 || age.Day() != 21 {
		t.Errorf("Unexpected date: %v", age)
	}
}

func TestParser_GetRemoteSpecAge_NoLastModified(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No Last-Modified header
	}))
	defer server.Close()

	age, err := parser.GetRemoteSpecAge(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetRemoteSpecAge failed: %v", err)
	}

	if age != nil {
		t.Error("Expected nil age when Last-Modified is missing")
	}
}

func TestParser_GetRemoteSpecAge_HeadError(t *testing.T) {
	logger := slog.Default()
	parser := NewParser(logger)

	// Server that returns error on HEAD
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	age, err := parser.GetRemoteSpecAge(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetRemoteSpecAge should not return error: %v", err)
	}

	if age != nil {
		t.Error("Expected nil age when HEAD fails")
	}
}

// Helper to create a minimal OpenAPI spec for testing
func createTestOpenAPISpec() *openapi3.T {
	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
			Tags:        []string{"users"},
		},
		Post: &openapi3.Operation{
			OperationID: "createUser",
			Summary:     "Create a user",
			Tags:        []string{"users"},
		},
	})

	return &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}
}
