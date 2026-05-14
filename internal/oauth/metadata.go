// Package oauth provides OAuth 2.1 protected resource handling.
package oauth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ProtectedResource handles OAuth 2.1 protected resource metadata.
type ProtectedResource struct {
	logger        *slog.Logger
	resource      string
	authServerURL string
}

// ProtectedResourceMetadata represents the OAuth 2.1 Protected Resource metadata.
// See: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-resource-metadata
type ProtectedResourceMetadata struct {
	Resource                  string   `json:"resource"`
	AuthorizationServers      []string `json:"authorization_servers"`
	ScopesSupported           []string `json:"scopes_supported"`
	BearerMethodsSupported    []string `json:"bearer_methods_supported"`
	ResourceSigningAlgorithms []string `json:"resource_signing_alg_values_supported,omitempty"`
}

// NewProtectedResource creates a new ProtectedResource handler.
func NewProtectedResource(logger *slog.Logger) *ProtectedResource {
	return &ProtectedResource{
		logger: logger,
	}
}

// Configure sets the resource and authorization server URLs.
func (p *ProtectedResource) Configure(resource, authServerURL string) {
	p.resource = resource
	p.authServerURL = authServerURL
}

// GetMetadata returns the protected resource metadata.
// This implements the OAuth 2.1 Protected Resource Metadata endpoint.
// See: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-resource-metadata
func (p *ProtectedResource) GetMetadata(w http.ResponseWriter, r *http.Request) {
	p.logger.Debug("GetMetadata called",
		slog.String("resource", p.resource),
		slog.String("authServer", p.authServerURL),
	)

	metadata := ProtectedResourceMetadata{
		Resource:               p.resource,
		AuthorizationServers:   []string{p.authServerURL},
		ScopesSupported:        []string{"read", "write"},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		p.logger.Error("Failed to encode metadata", slog.String("error", err.Error()))
	}
}
