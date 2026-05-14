package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/myadmin/go-mcp-proxy-client/internal/openapi"
	"github.com/myadmin/go-mcp-proxy-client/internal/proxy"
)

// integrationHandler provides a handler with a real HTTP test server for integration testing
type integrationHandler struct {
	*Handler
	server *httptest.Server
}

func newIntegrationHandler(t *testing.T) *integrationHandler {
	// Create a test server that captures requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","path":"` + r.URL.Path + `"}`))
	}))

	// Create proxy client pointing to test server
	proxyClient := proxy.NewClient(server.URL, slog.Default())

	h := &integrationHandler{
		Handler: NewHandler(slog.Default(), proxyClient),
		server:  server,
	}

	t.Cleanup(func() {
		server.Close()
	})

	return h
}

func TestNewHandler(t *testing.T) {
	logger := slog.Default()
	proxyClient := &proxy.Client{}
	h := NewHandler(logger, proxyClient)

	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestToolHandler_Success(t *testing.T) {
	ih := newIntegrationHandler(t)

	tool := openapi.Tool{
		Name:        "testTool",
		Description: "Test tool",
		HTTPMethod:  "GET",
		Path:        "/api/test",
		PathParams:  []string{"id"},
		QueryParams: []string{"format"},
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":     map[string]interface{}{"type": "string"},
				"format": map[string]interface{}{"type": "string"},
			},
			"required": []string{"id"},
		},
	}

	handler := ih.ToolHandler(tool)

	// Create a request with auth headers in Extra
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "testTool",
			Arguments: json.RawMessage(`{"id":"123","format":"json"}`),
		},
		Extra: &mcp.RequestExtra{
			Header: http.Header{},
		},
	}
	req.Extra.Header.Set("Authorization", "Bearer testtoken")
	req.Extra.Header.Set("X-API-KEY", "apikey123")
	req.Extra.Header.Set("sessionid", "sess456")

	// Execute handler
	result, err := handler(context.Background(), req)

	// Verify no error
	if err != nil {
		t.Fatalf("ToolHandler returned error: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("ToolHandler returned nil result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	// Verify response contains success
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if !strings.Contains(textContent.Text, "success") {
		t.Errorf("Expected 'success' in response, got: %s", textContent.Text)
	}
}

func TestToolHandler_ErrorResponse(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	proxyClient := proxy.NewClient(server.URL, slog.Default())
	h := NewHandler(slog.Default(), proxyClient)

	tool := openapi.Tool{
		Name:       "errorTool",
		HTTPMethod: "GET",
		Path:       "/api/error",
	}

	handler := h.ToolHandler(tool)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "errorTool",
			Arguments: json.RawMessage(`{}`),
		},
		Extra: &mcp.RequestExtra{
			Header: http.Header{},
		},
	}

	result, err := handler(context.Background(), req)

	// Error should be returned in result, not as err
	if err != nil {
		t.Fatalf("ToolHandler returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("ToolHandler returned nil result")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true for non-2xx response")
	}
}

func TestToMCPTool(t *testing.T) {
	logger := slog.Default()
	proxyClient := &proxy.Client{}
	h := NewHandler(logger, proxyClient)

	tests := []struct {
		name           string
		tool           openapi.Tool
		expectName     string
		expectDesc     string
		expectReadOnly bool
		expectDest     bool
	}{
		{
			name: "basic tool",
			tool: openapi.Tool{
				Name:        "getUsers",
				Description: "Get all users",
				HTTPMethod:  "GET",
				Path:        "/users",
				Annotations: openapi.ToolAnnotations{
					ReadOnlyHint: true,
				},
			},
			expectName:     "getUsers",
			expectDesc:     "Get all users",
			expectReadOnly: true,
		},
		{
			name: "destructive tool",
			tool: openapi.Tool{
				Name:        "deleteUser",
				Description: "Delete a user",
				HTTPMethod:  "DELETE",
				Path:        "/users/{id}",
				Annotations: openapi.ToolAnnotations{
					ReadOnlyHint:    false,
					DestructiveHint: true,
				},
			},
			expectName: "deleteUser",
			expectDesc: "Delete a user",
			expectDest: true,
		},
		{
			name: "tool without annotations",
			tool: openapi.Tool{
				Name:        "simpleTool",
				Description: "A simple tool",
				HTTPMethod:  "POST",
				Path:        "/simple",
			},
			expectName: "simpleTool",
			expectDesc: "A simple tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpTool := h.toMCPTool(tt.tool)

			if mcpTool.Name != tt.expectName {
				t.Errorf("Name = %q, want %q", mcpTool.Name, tt.expectName)
			}

			if mcpTool.Description != tt.expectDesc {
				t.Errorf("Description = %q, want %q", mcpTool.Description, tt.expectDesc)
			}

			if tt.expectReadOnly && mcpTool.Annotations != nil && !mcpTool.Annotations.ReadOnlyHint {
				t.Error("Expected ReadOnlyHint to be true")
			}

			if tt.expectDest && mcpTool.Annotations != nil && (mcpTool.Annotations.DestructiveHint == nil || !*mcpTool.Annotations.DestructiveHint) {
				t.Error("Expected DestructiveHint to be true")
			}
		})
	}
}

func TestToolHandler_AuthExtraction(t *testing.T) {
	var capturedAuth string
	var capturedAPIKey string
	var capturedSessionID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAPIKey = r.Header.Get("X-API-KEY")
		capturedSessionID = r.Header.Get("sessionid")

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	proxyClient := proxy.NewClient(server.URL, slog.Default())
	h := NewHandler(slog.Default(), proxyClient)

	tool := openapi.Tool{
		Name:       "authTest",
		HTTPMethod: "GET",
		Path:       "/api/test",
	}

	handler := h.ToolHandler(tool)

	tests := []struct {
		name           string
		headerKey      string
		headerVal      string
		capturedValPtr *string
		capturedVal    string
	}{
		{"Bearer token", "Authorization", "Bearer mytoken", &capturedAuth, "Bearer mytoken"},
		{"API Key", "X-API-KEY", "apikey123", &capturedAPIKey, "apikey123"},
		{"Session ID", "sessionid", "sess456", &capturedSessionID, "sess456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset captured values
			capturedAuth = ""
			capturedAPIKey = ""
			capturedSessionID = ""

			req := &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name:      "authTest",
					Arguments: json.RawMessage(`{}`),
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{},
				},
			}
			req.Extra.Header.Set(tt.headerKey, tt.headerVal)

			result, err := handler(context.Background(), req)

			if err != nil {
				t.Fatalf("ToolHandler returned error: %v", err)
			}

			if result == nil {
				t.Fatal("ToolHandler returned nil result")
			}
		})
	}
}

func TestRegisterTools(t *testing.T) {
	logger := slog.Default()
	proxyClient := &proxy.Client{}
	h := NewHandler(logger, proxyClient)

	tools := []openapi.Tool{
		{
			Name:        "tool1",
			Description: "First tool",
			HTTPMethod:  "GET",
			Path:        "/api/tool1",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			HTTPMethod:  "POST",
			Path:        "/api/tool2",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	// Create a minimal MCP server for testing
	cfg := Config{Name: "test", Version: "1.0.0"}
	mcpServer, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	count := h.RegisterTools(mcpServer.Server(), tools)

	if count != len(tools) {
		t.Errorf("RegisterTools returned %d, want %d", count, len(tools))
	}
}

// Test that tools can be registered and retrieved
func TestToolRegistration_Integration(t *testing.T) {
	logger := slog.Default()

	// Create handler with test server
	ih := newIntegrationHandler(t)

	// Create MCP server
	cfg := Config{Name: "test", Version: "1.0.0"}
	mcpServer, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create tools
	tools := []openapi.Tool{
		{
			Name:        "getUsers",
			Description: "Get all users",
			HTTPMethod:  "GET",
			Path:        "/users",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{"type": "integer"},
				},
			},
			PathParams:  []string{},
			QueryParams: []string{"limit"},
			Annotations: openapi.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: true,
			},
		},
	}

	// Register tools
	count := ih.RegisterTools(mcpServer.Server(), tools)
	if count != 1 {
		t.Errorf("Registered %d tools, expected 1", count)
	}
}

// Test path parameter substitution via DoRequest
func TestPathParamSubstitution(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := proxy.NewClient(server.URL, slog.Default())

	resp, err := client.DoRequest(context.Background(), proxy.RequestOptions{
		Method:     "GET",
		Path:       "/users/{id}/posts/{postId}",
		PathParams: map[string]string{"id": "123", "postId": "456"},
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer resp.Body.Close()

	expectedPath := "/users/123/posts/456"
	if capturedPath != expectedPath {
		t.Errorf("Path param substitution failed: got %q, want %q", capturedPath, expectedPath)
	}
}

// Test that query params are properly added to URL
func TestQueryParamAddition(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := proxy.NewClient(server.URL, slog.Default())

	resp, err := client.DoRequest(context.Background(), proxy.RequestOptions{
		Method:      "GET",
		Path:        "/users",
		QueryParams: map[string]string{"limit": "10", "offset": "20"},
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(capturedURL, "limit=10") {
		t.Errorf("Query param 'limit' not found in URL: %s", capturedURL)
	}
	if !strings.Contains(capturedURL, "offset=20") {
		t.Errorf("Query param 'offset' not found in URL: %s", capturedURL)
	}
}

// Test that X-API-APP header is set correctly
func TestXAPIAPPHeader(t *testing.T) {
	var capturedXAPIAPP string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedXAPIAPP = r.Header.Get("X-API-APP")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := proxy.NewClient(server.URL, slog.Default())

	resp, err := client.DoRequest(context.Background(), proxy.RequestOptions{
		Method: "GET",
		Path:   "/test",
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if capturedXAPIAPP != "1" {
		t.Errorf("X-API-APP header = %q, want %q", capturedXAPIAPP, "1")
	}
}

// Test that X-Request-Id header is set correctly
func TestXRequestIdHeader(t *testing.T) {
	var capturedRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Header.Get("X-Request-Id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := proxy.NewClient(server.URL, slog.Default())

	resp, err := client.DoRequest(context.Background(), proxy.RequestOptions{
		Method: "GET",
		Path:   "/test",
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if capturedRequestID == "" {
		t.Error("X-Request-Id header was not set")
	}
}

// Test POST request with body
func TestPOSTWithBody(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		capturedBody = string(body[:n])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer server.Close()

	client := proxy.NewClient(server.URL, slog.Default())

	resp, err := client.DoRequest(context.Background(), proxy.RequestOptions{
		Method: "POST",
		Path:   "/users",
		Body:   map[string]string{"name": "John", "email": "john@example.com"},
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	if !strings.Contains(capturedBody, "John") {
		t.Errorf("Expected 'John' in body, got: %s", capturedBody)
	}
}