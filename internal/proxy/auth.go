// Package proxy provides authentication header extraction and handling.
package proxy

import (
	"log/slog"
	"net/http"
)

// Authenticator handles auth header extraction.
type Authenticator struct {
	logger *slog.Logger
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(logger *slog.Logger) *Authenticator {
	return &Authenticator{logger: logger}
}

// ExtractBearerToken extracts Bearer token from Authorization header.
func (a *Authenticator) ExtractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	a.logger.Debug("Extracting auth header", slog.String("header", authHeader))

	// TODO: Implement token extraction
	return ""
}

// AddAuthHeader adds authentication header to a request.
func (a *Authenticator) AddAuthHeader(r *http.Request, token string) {
	r.Header.Set("Authorization", "Bearer "+token)
}
