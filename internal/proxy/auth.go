// Package proxy provides authentication header extraction and handling.
package proxy

import (
	"log/slog"
	"net/http"
	"strings"
)

// Authenticator handles auth header extraction.
type Authenticator struct {
	logger *slog.Logger
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(logger *slog.Logger) *Authenticator {
	return &Authenticator{logger: logger}
}

// AuthResult holds extracted authentication data from a request.
type AuthResult struct {
	BearerToken string
	APIKey      string
	SessionID   string
	HasAuth     bool
}

// ExtractAuth extracts all supported auth headers from the incoming request.
// Checks for Authorization: Bearer <token>, X-API-KEY: <key>, and sessionid: <id>.
func (a *Authenticator) ExtractAuth(r *http.Request) *AuthResult {
	result := &AuthResult{}

	// Check Authorization: Bearer <token>
	bearerToken := a.ExtractBearerToken(r)
	if bearerToken != "" {
		result.BearerToken = bearerToken
		result.HasAuth = true
	}

	// Check X-API-KEY: <key>
	apiKey := r.Header.Get("X-API-KEY")
	if apiKey != "" {
		result.APIKey = apiKey
		result.HasAuth = true
	}

	// Check sessionid: <id>
	sessionID := r.Header.Get("sessionid")
	if sessionID != "" {
		result.SessionID = sessionID
		result.HasAuth = true
	}

	a.logger.Debug("Extracted auth",
		slog.String("bearer", maskString(result.BearerToken)),
		slog.String("apiKey", maskString(result.APIKey)),
		slog.String("sessionId", maskString(result.SessionID)),
		slog.Bool("hasAuth", result.HasAuth),
	)

	return result
}

// ExtractBearerToken extracts Bearer token from Authorization header.
func (a *Authenticator) ExtractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Must start with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return ""
	}

	a.logger.Debug("Extracted bearer token", slog.String("token", maskString(token)))
	return token
}

// AddAuthHeaders adds all extracted auth headers to an outgoing request.
func (a *Authenticator) AddAuthHeaders(out *http.Request, auth *AuthResult) {
	if auth == nil {
		return
	}

	if auth.BearerToken != "" {
		out.Header.Set("Authorization", "Bearer "+auth.BearerToken)
	}

	if auth.APIKey != "" {
		out.Header.Set("X-API-KEY", auth.APIKey)
	}

	if auth.SessionID != "" {
		out.Header.Set("sessionid", auth.SessionID)
	}
}

// AddAuthHeader adds a single authentication header to a request.
func (a *Authenticator) AddAuthHeader(r *http.Request, token string) {
	r.Header.Set("Authorization", "Bearer "+token)
}

// maskString masks a sensitive string for logging, showing first and last 3 chars.
func maskString(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "..." + s[len(s)-3:]
}
