// Package server provides HTTP handlers and tool management for the MCP proxy server.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/myadmin/go-mcp-proxy-client/internal/openapi"
	"github.com/myadmin/go-mcp-proxy-client/internal/proxy"
)

// Handler holds HTTP handler dependencies.
type Handler struct {
	logger       *slog.Logger
	proxyClient  *proxy.Client
	auth         *proxy.Authenticator
	openapiGen   *openapi.Generator
}

// NewHandler creates a new Handler instance.
func NewHandler(logger *slog.Logger, proxyClient *proxy.Client) *Handler {
	return &Handler{
		logger:      logger,
		proxyClient: proxyClient,
		auth:        proxy.NewAuthenticator(logger),
		openapiGen:  openapi.NewGenerator(logger),
	}
}

// HandleTools handles tool listing and execution.
func (h *Handler) HandleTools(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handleTools called", slog.String("method", r.Method))

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Method string `json:"method"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ToolHandler creates an MCP tool handler for the given tool.
// It returns an mcp.ToolHandler callback that proxies requests to the upstream API.
func (h *Handler) ToolHandler(tool openapi.Tool) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		h.logger.Debug("ToolHandler invoked",
			slog.String("tool", tool.Name),
			slog.String("method", tool.HTTPMethod),
			slog.String("path", tool.Path),
		)

		// Extract auth from incoming request headers
		var authResult *proxy.AuthResult
		if request.Extra != nil && request.Extra.Header != nil {
			// Create a mock http.Request to use existing auth extraction
			mockReq := &http.Request{Header: request.Extra.Header}
			authResult = h.auth.ExtractAuth(mockReq)
		}

		// Parse arguments from request
		args := make(map[string]string)
		if request.Params != nil && len(request.Params.Arguments) > 0 {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				h.logger.Warn("Failed to parse tool arguments",
					slog.String("error", err.Error()),
					slog.String("arguments", string(request.Params.Arguments)),
				)
			}
		}

		// Separate path params and query params from args
		pathParams := make(map[string]string)
		queryParams := make(map[string]string)

		for _, paramName := range tool.PathParams {
			if value, ok := args[paramName]; ok {
				pathParams[paramName] = fmt.Sprintf("%v", value)
				delete(args, paramName)
			}
		}

		for _, paramName := range tool.QueryParams {
			if value, ok := args[paramName]; ok {
				queryParams[paramName] = fmt.Sprintf("%v", value)
				delete(args, paramName)
			}
		}

		// Build request options
		opts := proxy.RequestOptions{
			Method:      tool.HTTPMethod,
			Path:        tool.Path,
			PathParams:  pathParams,
			QueryParams: queryParams,
			Auth:        authResult,
		}

		// Add body for POST/PUT/PATCH if hasBody is true and we have remaining args
		if tool.HasBody && len(args) > 0 {
			opts.Body = args
		}

		// Make request to upstream API
		resp, err := h.proxyClient.DoRequest(ctx, opts)
		if err != nil {
			h.logger.Error("Upstream API request failed",
				slog.String("tool", tool.Name),
				slog.String("error", err.Error()),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Error calling upstream API: %s", err.Error()),
					},
				},
				IsError: true,
			}, nil
		}
		defer resp.Body.Close()

		// Read response body
		body, err := h.proxyClient.ReadResponseBody(resp)
		if err != nil {
			h.logger.Error("Failed to read upstream response",
				slog.String("tool", tool.Name),
				slog.String("error", err.Error()),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Error reading upstream response: %s", err.Error()),
					},
				},
				IsError: true,
			}, nil
		}

		// Check for non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			h.logger.Warn("Upstream API returned non-success status",
				slog.String("tool", tool.Name),
				slog.Int("status", resp.StatusCode),
				slog.String("body", string(body)),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Upstream API error (HTTP %d): %s", resp.StatusCode, string(body)),
					},
				},
				IsError: true,
			}, nil
		}

		// Wrap top-level array responses for MCP compliance.
		// MCP requires structuredContent to be a JSON object, not a list.
		// ~89 OpenAPI list endpoints return top-level arrays; wrap them
		// so the SDK emits a valid object.
		responseBody := wrapArrayResponse(body)

		// Return successful response
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: responseBody,
				},
			},
		}, nil
	}
}

// toMCPTool converts an openapi.Tool to an mcp.Tool.
func (h *Handler) toMCPTool(tool openapi.Tool) *mcp.Tool {
	mcpTool := &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: tool.InputSchema,
	}

	// Set annotations if any are present
	if tool.Annotations.Title != "" || tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint || tool.Annotations.IdempotentHint || tool.Annotations.OpenWorldHint {
		mcpTool.Annotations = &mcp.ToolAnnotations{
			Title:          tool.Annotations.Title,
			ReadOnlyHint:   tool.Annotations.ReadOnlyHint,
			IdempotentHint: tool.Annotations.IdempotentHint,
		}

		// These fields are pointers
		if tool.Annotations.DestructiveHint {
			val := true
			mcpTool.Annotations.DestructiveHint = &val
		}
		if tool.Annotations.OpenWorldHint {
			val := true
			mcpTool.Annotations.OpenWorldHint = &val
		}
	}

	return mcpTool
}

// RegisterToolsFromSpec fetches the OpenAPI spec and registers tools with the MCP server.
// Returns the number of tools registered.
func (h *Handler) RegisterToolsFromSpec(ctx context.Context, server *mcp.Server, specURL string) (int, error) {
	h.logger.Info("Registering tools from OpenAPI spec", slog.String("url", specURL))

	// Parse OpenAPI spec
	parser := openapi.NewParser(h.logger)
	spec, err := parser.Parse(ctx, specURL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Extract tools from spec
	tools := h.openapiGen.ExtractTools(spec.SpecV3)
	if len(tools) == 0 {
		h.logger.Warn("No tools found in OpenAPI spec")
		return 0, nil
	}

	// Register each tool with the server
	for _, tool := range tools {
		mcpTool := h.toMCPTool(tool)
		handler := h.ToolHandler(tool)

		server.AddTool(mcpTool, handler)

		h.logger.Debug("Registered tool",
			slog.String("name", tool.Name),
			slog.String("method", tool.HTTPMethod),
			slog.String("path", tool.Path),
		)
	}

	h.logger.Info("Registered tools from OpenAPI spec", slog.Int("count", len(tools)))
	return len(tools), nil
}

// RegisterTools registers the given tools with the MCP server.
func (h *Handler) RegisterTools(server *mcp.Server, tools []openapi.Tool) int {
	count := 0
	for _, tool := range tools {
		mcpTool := h.toMCPTool(tool)
		handler := h.ToolHandler(tool)

		server.AddTool(mcpTool, handler)
		count++
	}

	h.logger.Info("Registered tools", slog.Int("count", count))
	return count
}

// wrapArrayResponse checks if the body is a JSON array and wraps it as {"items": ...}.
// MCP requires structuredContent to be a JSON object, not a list.
// This ensures list responses from OpenAPI endpoints are properly wrapped.
func wrapArrayResponse(body []byte) string {
	// Try to parse the body as JSON
	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		// Not valid JSON, return as-is
		return string(body)
	}

	// Check if it's an array (slice in Go)
	if arr, ok := decoded.([]interface{}); ok {
		// Wrap the array as {"items": [...]}
		wrapped := map[string]interface{}{
			"items": arr,
		}
		wrappedJSON, err := json.Marshal(wrapped)
		if err != nil {
			// If marshaling fails, return original
			return string(body)
		}
		return string(wrappedJSON)
	}

	// It's already an object or other type, return as-is
	return string(body)
}