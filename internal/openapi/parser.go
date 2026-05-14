// Package openapi provides OpenAPI specification parsing and tool generation.
package openapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Parser handles OpenAPI spec fetching and parsing.
type Parser struct {
	logger     *slog.Logger
	httpClient *http.Client
}

// NewParser creates a new Parser instance.
func NewParser(logger *slog.Logger) *Parser {
	return &Parser{
		logger: logger,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: http.DefaultTransport,
		},
	}
}

// NewParserWithClient creates a new Parser with a custom HTTP client.
func NewParserWithClient(logger *slog.Logger, client *http.Client) *Parser {
	return &Parser{
		logger:     logger,
		httpClient: client,
	}
}

// Spec represents a parsed OpenAPI specification.
type Spec struct {
	Title       string
	Version     string
	SpecV3      *openapi3.T
	RawSpec     []byte
	Description string
}

// Parse fetches and parses an OpenAPI specification from a URL.
func (p *Parser) Parse(ctx context.Context, specURL string) (*Spec, error) {
	p.logger.Debug("Parsing OpenAPI spec", slog.String("url", specURL))

	// Validate URL early
	parsedURL, err := url.Parse(specURL)
	if err != nil {
		return nil, fmt.Errorf("invalid spec URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme %q: must be http or https", parsedURL.Scheme)
	}

	// Fetch the spec content
	content, err := p.fetchSpec(ctx, specURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec: %w", err)
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("empty response from OpenAPI spec URL")
	}

	// Parse the spec
	specV3, err := p.parseSpecContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec content: %w", err)
	}

	// Build Spec struct
	spec := &Spec{
		Title:       specV3.Info.Title,
		Version:     specV3.Info.Version,
		Description: specV3.Info.Description,
		SpecV3:      specV3,
		RawSpec:     content,
	}

	p.logger.Info("Fetched OpenAPI spec",
		slog.String("title", spec.Title),
		slog.String("version", spec.Version),
		slog.String("host", parsedURL.Host),
		slog.Int("pathCount", specV3.Paths.Len()),
	)

	return spec, nil
}

// fetchSpec retrieves the OpenAPI spec content from a URL.
func (p *Parser) fetchSpec(ctx context.Context, specURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, specURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Accept JSON and YAML
	req.Header.Set("Accept", "application/json, application/x-yaml, text/yaml, text/x-yaml, */*")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d response from spec URL", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// parseSpecContent parses JSON or YAML content into an OpenAPI 3.x spec.
func (p *Parser) parseSpecContent(content []byte) (*openapi3.T, error) {
	// Try parsing as YAML first (many specs are YAML)
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromData(content)
	if err != nil {
		// Try to detect if it's JSON and give a better error
		trimmed := strings.TrimSpace(string(content))
		if strings.HasPrefix(trimmed, "{") {
			return nil, fmt.Errorf("failed to parse JSON OpenAPI spec: %w", err)
		}
		return nil, fmt.Errorf("failed to parse OpenAPI spec (not valid JSON or YAML): %w", err)
	}

	// Validate the spec
	if err := spec.Validate(loader.Context); err != nil {
		p.logger.Warn("Spec validation warning", slog.String("error", err.Error()))
		// Continue anyway - some specs have minor validation issues
	}

	return spec, nil
}

// GetRemoteSpecAge returns the Last-Modified timestamp of a remote spec via HEAD request.
// Returns nil if unable to determine.
func (p *Parser) GetRemoteSpecAge(ctx context.Context, specURL string) (*time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, specURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // Not an error, just can't determine age
	}

	lastModified := resp.Header.Get("Last-Modified")
	if lastModified == "" {
		return nil, nil
	}

	timestamp, err := http.ParseTime(lastModified)
	if err != nil {
		return nil, nil
	}

	return &timestamp, nil
}
