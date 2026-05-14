// Package proxy provides upstream API client functionality.
package proxy

import (
	"log/slog"
	"net/http"
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

// DoRequest performs an authenticated request to the upstream API.
func (c *Client) DoRequest(method, path string) (*http.Response, error) {
	c.logger.Debug("DoRequest",
		slog.String("method", method),
		slog.String("path", path),
	)

	// TODO: Implement actual request
	return nil, nil
}
