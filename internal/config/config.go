// Package config provides configuration management for the MCP proxy client.
// It loads configuration from .env files and environment variables with
// validation of required fields.
package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the MCP proxy client.
// Once created, Config is immutable.
type Config struct {
	OpenAPISpecURL           string
	APIBaseURL               string
	SessionDir               string
	CacheDir                 string
	ServerName               string
	ServerVersion            string
	BearerToken              string
	APIKey                   string
	SessionID                string
	OAuthAuthorizationServer string
}

// Default values for optional configuration.
const (
	DefaultSessionDir    = "/tmp/mcp_client_sessions"
	DefaultCacheDir      = "/tmp/mcp_client_cache"
	DefaultServerName    = "myadmin-client-mcp"
	DefaultServerVersion = "1.0.0"
)

// ErrMissingRequired indicates a required configuration field is missing.
var ErrMissingRequired = errors.New("missing required configuration")

// ErrInvalidURL indicates a configuration field contains an invalid URL.
var ErrInvalidURL = errors.New("invalid URL format")

// Load reads configuration from .env file and environment variables.
// Environment variables take precedence over .env file values.
// Required fields: OpenAPISpecURL, APIBaseURL.
func Load(configPath string) (*Config, error) {
	return LoadWithLogger(configPath, nil)
}

// LoadWithLogger reads configuration with optional logger for debugging.
func LoadWithLogger(configPath string, logger *slog.Logger) (*Config, error) {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	if err := godotenv.Load(configPath); err != nil && !os.IsNotExist(err) {
		if logger != nil {
			logger.Warn("Failed to load .env file", slog.String("error", err.Error()))
		}
	}

	cfg := &Config{
		OpenAPISpecURL:           getEnvOrEmpty("OPENAPI_SPEC_URL"),
		APIBaseURL:               getEnvOrEmpty("API_BASE_URL"),
		SessionDir:               getEnvOrDefault("SESSION_DIR", DefaultSessionDir),
		CacheDir:                 getEnvOrDefault("CACHE_DIR", DefaultCacheDir),
		ServerName:               getEnvOrDefault("SERVER_NAME", DefaultServerName),
		ServerVersion:            getEnvOrDefault("SERVER_VERSION", DefaultServerVersion),
		BearerToken:              getEnvOrEmpty("BEARER_TOKEN"),
		APIKey:                   getEnvOrEmpty("API_KEY"),
		SessionID:                getEnvOrEmpty("SESSION_ID"),
		OAuthAuthorizationServer: getEnvOrEmpty("OAUTH_AUTHORIZATION_SERVER"),
	}

	if logger != nil {
		logger.Debug("Configuration loaded from environment",
			slog.String("openapiSpecURL", maskEmpty(cfg.OpenAPISpecURL)),
			slog.String("apiBaseURL", maskEmpty(cfg.APIBaseURL)),
			slog.String("sessionDir", cfg.SessionDir),
			slog.String("cacheDir", cfg.CacheDir),
			slog.String("serverName", cfg.ServerName),
			slog.String("serverVersion", cfg.ServerVersion),
			slog.Bool("hasBearerToken", cfg.BearerToken != ""),
			slog.Bool("hasAPIKey", cfg.APIKey != ""),
			slog.Bool("hasSessionID", cfg.SessionID != ""),
			slog.String("oauthAuthServer", maskEmpty(cfg.OAuthAuthorizationServer)),
		)
	}

	// Derive OAuthAuthorizationServer from APIBaseURL if not set
	if cfg.OAuthAuthorizationServer == "" && cfg.APIBaseURL != "" {
		parsedURL, err := url.Parse(cfg.APIBaseURL)
		if err == nil {
			// Use the base URL (scheme + host) as the OAuth authorization server
			cfg.OAuthAuthorizationServer = parsedURL.Scheme + "://" + parsedURL.Host
		}
	}

	// Validate required fields (Fail Fast)
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required configuration fields are present and valid.
func validate(cfg *Config) error {
	// Early exit on missing required fields
	if cfg.OpenAPISpecURL == "" {
		return wrapRequiredError("OPENAPI_SPEC_URL")
	}
	if cfg.APIBaseURL == "" {
		return wrapRequiredError("API_BASE_URL")
	}

	// Validate URLs are well-formed
	if !isValidURL(cfg.OpenAPISpecURL) {
		return wrapInvalidURLError("OPENAPI_SPEC_URL", cfg.OpenAPISpecURL)
	}
	if !isValidURL(cfg.APIBaseURL) {
		return wrapInvalidURLError("API_BASE_URL", cfg.APIBaseURL)
	}

	return nil
}

// isValidURL checks if the given string is a valid HTTP or HTTPS URL with a host.
func isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	// Must be http or https scheme AND have a host
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// wrapRequiredError creates a descriptive error for missing required fields.
func wrapRequiredError(fieldName string) error {
	return errors.New(ErrMissingRequired.Error() + ": " + fieldName)
}

// wrapInvalidURLError creates a descriptive error for invalid URL fields.
func wrapInvalidURLError(fieldName, fieldValue string) error {
	return errors.New(ErrInvalidURL.Error() + ": " + fieldName + " value '" + maskEmpty(fieldValue) + "' is not a valid HTTP/HTTPS URL")
}

// getEnvOrEmpty returns the environment variable value or empty string if not set.
func getEnvOrEmpty(key string) string {
	return os.Getenv(key)
}

// getEnvOrDefault returns the environment variable value or the default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskEmpty returns "(empty)" for empty strings, otherwise the value.
// Used for logging to avoid exposing sensitive empty values.
func maskEmpty(s string) string {
	if s == "" {
		return "(empty)"
	}
	return s
}

// GetOAuthAuthorizationServer returns the OAuth authorization server URL.
// If not explicitly set, it returns the derived value from APIBaseURL.
func (c *Config) GetOAuthAuthorizationServer() string {
	return c.OAuthAuthorizationServer
}

// HasStdioCredentials checks if the configuration has stdio mode credentials.
func (c *Config) HasStdioCredentials() bool {
	return c.BearerToken != "" || c.APIKey != "" || c.SessionID != ""
}

// GetServerInfo returns the server name and version as a formatted string.
func (c *Config) GetServerInfo() string {
	return c.ServerName + "/" + c.ServerVersion
}

// NormalizeAPIBaseURL returns the API base URL without trailing slash.
func (c *Config) NormalizeAPIBaseURL() string {
	return strings.TrimRight(c.APIBaseURL, "/")
}
