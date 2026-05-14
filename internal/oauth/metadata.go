// Package oauth provides OAuth 2.1 protected resource handling.
package oauth

import (
	"log/slog"
	"net/http"
)

// ProtectedResource handles OAuth 2.1 protected resource metadata.
type ProtectedResource struct {
	logger *slog.Logger
}

// ProtectedResourceMetadata represents the OAuth 2.1 Protected Resource metadata.
type ProtectedResourceMetadata struct {
	Resource string `json:"resource"`
	Scopes   []string `json:"scopes_supported"`
}

// NewProtectedResource creates a new ProtectedResource handler.
func NewProtectedResource(logger *slog.Logger) *ProtectedResource {
	return &ProtectedResource{logger: logger}
}

// GetMetadata returns the protected resource metadata.
func (p *ProtectedResource) GetMetadata(w http.ResponseWriter, r *http.Request) {
	p.logger.Debug("GetMetadata called")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// TODO: Return proper metadata
}
