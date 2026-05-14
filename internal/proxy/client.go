// Package proxy provides upstream API client functionality.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// Client makes requests to the upstream API.
type Client struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
	auth    *Authenticator
}

// NewClient creates a new API client.
func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
		logger:  logger,
		auth:    NewAuthenticator(logger),
	}
}

// RequestOptions holds options for making a request.
type RequestOptions struct {
	Method      string
	Path        string
	PathParams  map[string]string // path parameter substitutions
	QueryParams map[string]string // query parameters to add
	Body        interface{}       // JSON body for POST/PUT/PATCH
	Auth        *AuthResult       // extracted auth from incoming request
}

// DoRequest performs an authenticated request to the upstream API.
func (c *Client) DoRequest(ctx context.Context, opts RequestOptions) (*http.Response, error) {
	// Guard: Validate method
	if opts.Method == "" {
		return nil, fmt.Errorf("HTTP method is required")
	}

	// Guard: Validate path
	if opts.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Build URL with path parameters substituted
	urlPath := c.substitutePathParams(opts.Path, opts.PathParams)

	// Add query parameters
	fullURL := c.buildURL(urlPath, opts.QueryParams)

	c.logger.Debug("DoRequest",
		slog.String("method", opts.Method),
		slog.String("url", fullURL),
		slog.Any("pathParams", opts.PathParams),
	)

	// Create request
	var bodyReader io.Reader
	if opts.Body != nil && opts.hasBody() {
		jsonBody, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, opts.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setHeaders(req, opts.Auth)

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// substitutePathParams replaces {param} placeholders in the path with values from pathParams.
func (c *Client) substitutePathParams(path string, pathParams map[string]string) string {
	if pathParams == nil || len(pathParams) == 0 {
		return path
	}

	result := path
	for param, value := range pathParams {
		// Replace {param} with the actual value
		placeholder := "{" + param + "}"
		result = strings.ReplaceAll(result, placeholder, url.PathEscape(value))
	}

	return result
}

// buildURL constructs the full URL with query parameters.
func (c *Client) buildURL(path string, queryParams map[string]string) string {
	baseURL := strings.TrimSuffix(c.baseURL, "/")
	path = strings.TrimPrefix(path, "/")

	fullURL := baseURL + "/" + path

	if len(queryParams) > 0 {
		query := url.Values{}
		for key, value := range queryParams {
			query.Add(key, value)
		}
		fullURL += "?" + query.Encode()
	}

	return fullURL
}

// setHeaders sets required headers on the outgoing request.
func (c *Client) setHeaders(req *http.Request, auth *AuthResult) {
	// Set X-API-APP: 1 - Short-circuits rate limiting for MCP callers
	req.Header.Set("X-API-APP", "1")

	// Set X-Request-Id: <uuid> - For tracing (generate UUID if not provided)
	requestID := req.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	req.Header.Set("X-Request-Id", requestID)

	// Set auth headers if provided
	if auth != nil {
		c.auth.AddAuthHeaders(req, auth)
	}

	// Set content type for requests with body
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

// hasBody returns true if the HTTP method typically has a body.
func (c *Client) hasBody() bool {
	switch c.client.Transport.(type) {
	default:
		return true
	}
}

// DoRequestSimple performs a simple authenticated request without body.
func (c *Client) DoRequestSimple(ctx context.Context, method, path string, auth *AuthResult) (*http.Response, error) {
	return c.DoRequest(ctx, RequestOptions{
		Method: method,
		Path:   path,
		Auth:   auth,
	})
}

// DoRequestWithBody performs an authenticated request with JSON body.
func (c *Client) DoRequestWithBody(ctx context.Context, method, path string, body interface{}, auth *AuthResult) (*http.Response, error) {
	return c.DoRequest(ctx, RequestOptions{
		Method: method,
		Path:   path,
		Body:   body,
		Auth:   auth,
	})
}

// ReadResponseBody reads and returns the body from an HTTP response.
func (c *Client) ReadResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// hasBody returns true if the HTTP method typically has a body.
func (o RequestOptions) hasBody() bool {
	switch strings.ToUpper(o.Method) {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}
