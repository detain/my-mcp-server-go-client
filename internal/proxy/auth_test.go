package proxy

import (
	"net/http"
	"testing"

	"log/slog"
)

func TestNewAuthenticator(t *testing.T) {
	logger := slog.Default()
	auth := NewAuthenticator(logger)

	if auth == nil {
		t.Fatal("NewAuthenticator returned nil")
	}
}

func TestAuthenticator_ExtractBearerToken(t *testing.T) {
	logger := slog.Default()
	auth := NewAuthenticator(logger)

	tests := []struct {
		name        string
		authHeader  string
		expectToken string
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer abc123xyz",
			expectToken: "abc123xyz",
		},
		{
			name:        "empty header",
			authHeader:  "",
			expectToken: "",
		},
		{
			name:        "missing Bearer prefix",
			authHeader:  "abc123xyz",
			expectToken: "",
		},
		{
			name:        "only Bearer prefix",
			authHeader:  "Bearer ",
			expectToken: "",
		},
		{
			name:        "Basic auth (not bearer)",
			authHeader:  "Basic dXNlcjpwYXNz",
			expectToken: "",
		},
		{
			name:        "bearer with extra spaces",
			authHeader:  "Bearer   token123  ",
			expectToken: "  token123  ", // preserves internal spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: http.Header{}}
			req.Header.Set("Authorization", tt.authHeader)

			token := auth.ExtractBearerToken(req)
			if token != tt.expectToken {
				t.Errorf("ExtractBearerToken() = %q, want %q", token, tt.expectToken)
			}
		})
	}
}

func TestAuthenticator_ExtractAuth(t *testing.T) {
	logger := slog.Default()
	auth := NewAuthenticator(logger)

	tests := []struct {
		name              string
		setupHeader       func(http.Header)
		expectBearer      string
		expectAPIKey      string
		expectSessionID   string
		expectHasAuth     bool
	}{
		{
			name: "bearer token only",
			setupHeader: func(h http.Header) {
				h.Set("Authorization", "Bearer mytoken123")
			},
			expectBearer:  "mytoken123",
			expectAPIKey:  "",
			expectSessionID: "",
			expectHasAuth: true,
		},
		{
			name: "api key only",
			setupHeader: func(h http.Header) {
				h.Set("X-API-KEY", "apikey456")
			},
			expectBearer:    "",
			expectAPIKey:    "apikey456",
			expectSessionID: "",
			expectHasAuth:   true,
		},
		{
			name: "session id only",
			setupHeader: func(h http.Header) {
				h.Set("sessionid", "session789")
			},
			expectBearer:    "",
			expectAPIKey:    "",
			expectSessionID: "session789",
			expectHasAuth:   true,
		},
		{
			name: "all auth headers",
			setupHeader: func(h http.Header) {
				h.Set("Authorization", "Bearer mytoken123")
				h.Set("X-API-KEY", "apikey456")
				h.Set("sessionid", "session789")
			},
			expectBearer:    "mytoken123",
			expectAPIKey:    "apikey456",
			expectSessionID: "session789",
			expectHasAuth:   true,
		},
		{
			name: "no auth headers",
			setupHeader: func(h http.Header) {
				h.Set("Content-Type", "application/json")
			},
			expectBearer:    "",
			expectAPIKey:    "",
			expectSessionID: "",
			expectHasAuth:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: http.Header{}}
			tt.setupHeader(req.Header)

			result := auth.ExtractAuth(req)

			if result.BearerToken != tt.expectBearer {
				t.Errorf("BearerToken = %q, want %q", result.BearerToken, tt.expectBearer)
			}
			if result.APIKey != tt.expectAPIKey {
				t.Errorf("APIKey = %q, want %q", result.APIKey, tt.expectAPIKey)
			}
			if result.SessionID != tt.expectSessionID {
				t.Errorf("SessionID = %q, want %q", result.SessionID, tt.expectSessionID)
			}
			if result.HasAuth != tt.expectHasAuth {
				t.Errorf("HasAuth = %v, want %v", result.HasAuth, tt.expectHasAuth)
			}
		})
	}
}

func TestAuthenticator_AddAuthHeaders(t *testing.T) {
	logger := slog.Default()
	auth := NewAuthenticator(logger)

	tests := []struct {
		name        string
		authResult  *AuthResult
		expectAuth  string
		expectKey   string
		expectSess  string
	}{
		{
			name: "bearer token only",
			authResult: &AuthResult{
				BearerToken: "mytoken",
				HasAuth:     true,
			},
			expectAuth: "Bearer mytoken",
			expectKey:  "",
			expectSess: "",
		},
		{
			name: "api key only",
			authResult: &AuthResult{
				APIKey:  "apikey",
				HasAuth: true,
			},
			expectAuth: "",
			expectKey:  "apikey",
			expectSess: "",
		},
		{
			name: "all headers",
			authResult: &AuthResult{
				BearerToken: "mytoken",
				APIKey:      "apikey",
				SessionID:   "sess123",
				HasAuth:     true,
			},
			expectAuth: "Bearer mytoken",
			expectKey:  "apikey",
			expectSess: "sess123",
		},
		{
			name:        "nil auth result",
			authResult:  nil,
			expectAuth:  "",
			expectKey:   "",
			expectSess:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: http.Header{}}
			auth.AddAuthHeaders(req, tt.authResult)

			authHeader := req.Header.Get("Authorization")
			if authHeader != tt.expectAuth {
				t.Errorf("Authorization = %q, want %q", authHeader, tt.expectAuth)
			}

			apiKey := req.Header.Get("X-API-KEY")
			if apiKey != tt.expectKey {
				t.Errorf("X-API-KEY = %q, want %q", apiKey, tt.expectKey)
			}

			sessionID := req.Header.Get("sessionid")
			if sessionID != tt.expectSess {
				t.Errorf("sessionid = %q, want %q", sessionID, tt.expectSess)
			}
		})
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short string", "abc", "***"},
		{"exactly 6 chars", "abcdef", "***"},
		{"longer than 6", "longtokenvalue", "lon...lue"},
		{"empty string", "", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskString(tt.input)
			if result != tt.expected {
				t.Errorf("maskString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
