// Package config provides configuration management tests.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequiredFields(t *testing.T) {
	// Test that missing required fields returns error
	_, err := Load("/nonexistent/.env")
	if err == nil {
		t.Error("Expected error for missing OPENAPI_SPEC_URL, got nil")
	}
	if !isMissingRequired(err) {
		t.Errorf("Expected missing required error, got: %v", err)
	}
}

func TestLoadWithValidEnvVars(t *testing.T) {
	// Clear any existing env vars and set test values
	unsetEnvVars()

	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.OpenAPISpecURL != "https://api.example.com/openapi.json" {
		t.Errorf("Expected OpenAPISpecURL to be 'https://api.example.com/openapi.json', got: %s", cfg.OpenAPISpecURL)
	}
	if cfg.APIBaseURL != "https://api.example.com" {
		t.Errorf("Expected APIBaseURL to be 'https://api.example.com', got: %s", cfg.APIBaseURL)
	}
}

func TestLoadDefaultValues(t *testing.T) {
	unsetEnvVars()

	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check defaults
	if cfg.SessionDir != DefaultSessionDir {
		t.Errorf("Expected SessionDir to be '%s', got: %s", DefaultSessionDir, cfg.SessionDir)
	}
	if cfg.CacheDir != DefaultCacheDir {
		t.Errorf("Expected CacheDir to be '%s', got: %s", DefaultCacheDir, cfg.CacheDir)
	}
	if cfg.ServerName != DefaultServerName {
		t.Errorf("Expected ServerName to be '%s', got: %s", DefaultServerName, cfg.ServerName)
	}
	if cfg.ServerVersion != DefaultServerVersion {
		t.Errorf("Expected ServerVersion to be '%s', got: %s", DefaultServerVersion, cfg.ServerVersion)
	}
}

func TestLoadEnvVarOverride(t *testing.T) {
	unsetEnvVars()

	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com")
	os.Setenv("SESSION_DIR", "/custom/sessions")
	os.Setenv("CACHE_DIR", "/custom/cache")
	os.Setenv("SERVER_NAME", "custom-server")
	os.Setenv("SERVER_VERSION", "2.0.0")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.SessionDir != "/custom/sessions" {
		t.Errorf("Expected SessionDir to be '/custom/sessions', got: %s", cfg.SessionDir)
	}
	if cfg.CacheDir != "/custom/cache" {
		t.Errorf("Expected CacheDir to be '/custom/cache', got: %s", cfg.CacheDir)
	}
	if cfg.ServerName != "custom-server" {
		t.Errorf("Expected ServerName to be 'custom-server', got: %s", cfg.ServerName)
	}
	if cfg.ServerVersion != "2.0.0" {
		t.Errorf("Expected ServerVersion to be '2.0.0', got: %s", cfg.ServerVersion)
	}
}

func TestLoadOAuthDerivation(t *testing.T) {
	unsetEnvVars()

	// OAuthAuthorizationServer should be derived from APIBaseURL
	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com/v1")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should derive OAuth server from APIBaseURL (scheme + host)
	if cfg.OAuthAuthorizationServer != "https://api.example.com" {
		t.Errorf("Expected OAuthAuthorizationServer to be 'https://api.example.com', got: %s", cfg.OAuthAuthorizationServer)
	}
}

func TestLoadOAuthExplicitOverride(t *testing.T) {
	unsetEnvVars()

	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com/v1")
	os.Setenv("OAUTH_AUTHORIZATION_SERVER", "https://auth.example.com")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use explicit OAuth server, not derived
	if cfg.OAuthAuthorizationServer != "https://auth.example.com" {
		t.Errorf("Expected OAuthAuthorizationServer to be 'https://auth.example.com', got: %s", cfg.OAuthAuthorizationServer)
	}
}

func TestLoadInvalidURLs(t *testing.T) {
	testCases := []struct {
		name          string
		openapiURL    string
		apiBaseURL    string
		expectedError string
	}{
		{
			name:          "Invalid OpenAPI URL",
			openapiURL:    "not-a-valid-url",
			apiBaseURL:    "https://api.example.com",
			expectedError: "OPENAPI_SPEC_URL",
		},
		{
			name:          "Invalid API Base URL",
			openapiURL:    "https://api.example.com/openapi.json",
			apiBaseURL:    "ftp://api.example.com",
			expectedError: "API_BASE_URL",
		},
		{
			name:          "Empty OpenAPI URL",
			openapiURL:    "",
			apiBaseURL:    "https://api.example.com",
			expectedError: "OPENAPI_SPEC_URL",
		},
		{
			name:          "Empty API Base URL",
			openapiURL:    "https://api.example.com/openapi.json",
			apiBaseURL:    "",
			expectedError: "API_BASE_URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unsetEnvVars()
			if tc.openapiURL != "" {
				os.Setenv("OPENAPI_SPEC_URL", tc.openapiURL)
			}
			if tc.apiBaseURL != "" {
				os.Setenv("API_BASE_URL", tc.apiBaseURL)
			}
			defer unsetEnvVars()

			_, err := Load("/nonexistent/.env")
			if err == nil {
				t.Fatalf("Expected error containing '%s', got nil", tc.expectedError)
			}
			if !contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	unsetEnvVars()

	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `OPENAPI_SPEC_URL=https://file.example.com/openapi.json
API_BASE_URL=https://file.example.com
SESSION_DIR=/file/sessions
SERVER_NAME=from-file
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.OpenAPISpecURL != "https://file.example.com/openapi.json" {
		t.Errorf("Expected OpenAPISpecURL to be 'https://file.example.com/openapi.json', got: %s", cfg.OpenAPISpecURL)
	}
	if cfg.SessionDir != "/file/sessions" {
		t.Errorf("Expected SessionDir to be '/file/sessions', got: %s", cfg.SessionDir)
	}
	if cfg.ServerName != "from-file" {
		t.Errorf("Expected ServerName to be 'from-file', got: %s", cfg.ServerName)
	}
}

func TestLoadEnvFileEnvOverride(t *testing.T) {
	unsetEnvVars()

	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `OPENAPI_SPEC_URL=https://file.example.com/openapi.json
API_BASE_URL=https://file.example.com
SERVER_NAME=from-file
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Set environment variable to override .env value
	os.Setenv("SERVER_NAME", "from-env")

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Environment variable should take precedence
	if cfg.ServerName != "from-env" {
		t.Errorf("Expected ServerName to be 'from-env' (env override), got: %s", cfg.ServerName)
	}
}

func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"https://api.example.com", true},
		{"http://api.example.com", true},
		{"https://api.example.com/openapi.json", true},
		{"ftp://api.example.com", false},
		{"not-a-url", false},
		{"", false},
		{"https://", false},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			result := isValidURL(tc.url)
			if result != tc.expected {
				t.Errorf("isValidURL(%q) = %v, expected %v", tc.url, result, tc.expected)
			}
		})
	}
}

func TestConfigGetters(t *testing.T) {
	unsetEnvVars()
	os.Setenv("OPENAPI_SPEC_URL", "https://api.example.com/openapi.json")
	os.Setenv("API_BASE_URL", "https://api.example.com/v1")
	os.Setenv("OAUTH_AUTHORIZATION_SERVER", "https://auth.example.com")
	defer unsetEnvVars()

	cfg, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test GetOAuthAuthorizationServer
	if cfg.GetOAuthAuthorizationServer() != "https://auth.example.com" {
		t.Errorf("GetOAuthAuthorizationServer() = %s, expected https://auth.example.com", cfg.GetOAuthAuthorizationServer())
	}

	// Test HasStdioCredentials
	if cfg.HasStdioCredentials() {
		t.Error("HasStdioCredentials() = true, expected false (no stdio creds set)")
	}

	// Set stdio credentials
	os.Setenv("BEARER_TOKEN", "test-token")
	cfg2, err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !cfg2.HasStdioCredentials() {
		t.Error("HasStdioCredentials() = false, expected true (bearer token set)")
	}

	// Test GetServerInfo
	if cfg.GetServerInfo() != "myadmin-client-mcp/1.0.0" {
		t.Errorf("GetServerInfo() = %s, expected 'myadmin-client-mcp/1.0.0'", cfg.GetServerInfo())
	}

	// Test NormalizeAPIBaseURL
	if cfg.NormalizeAPIBaseURL() != "https://api.example.com/v1" {
		t.Errorf("NormalizeAPIBaseURL() = %s, expected 'https://api.example.com/v1'", cfg.NormalizeAPIBaseURL())
	}
}

// Helper functions

func unsetEnvVars() {
	envVars := []string{
		"OPENAPI_SPEC_URL",
		"API_BASE_URL",
		"SESSION_DIR",
		"CACHE_DIR",
		"SERVER_NAME",
		"SERVER_VERSION",
		"BEARER_TOKEN",
		"API_KEY",
		"SESSION_ID",
		"OAUTH_AUTHORIZATION_SERVER",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func isMissingRequired(err error) bool {
	return err != nil && (contains(err.Error(), "missing required") || err == ErrMissingRequired)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
