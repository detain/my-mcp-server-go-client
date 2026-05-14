// Package openapi provides caching for OpenAPI specifications and tools.
package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached specs.
	DefaultCacheTTL = 1 * time.Hour
	// CacheDirName is the directory name for the cache.
	CacheDirName = "openapi_cache"
)

// Cache provides caching for OpenAPI specs and generated tools.
type Cache struct {
	logger    *slog.Logger
	mu        sync.RWMutex
	items     map[string]*CacheItem
	cacheDir  string
	defaultTTL time.Duration
}

// CacheItem represents a cached item with expiration.
type CacheItem struct {
	Spec   *Spec    // Parsed OpenAPI spec (optional, for direct caching)
	Tools  []Tool   // Generated tools
	SpecURL string  // URL the spec was fetched from
	FetchedAt time.Time
	ExpiresAt time.Time
}

// CachedTools represents the cached tools structure in JSON.
type CachedTools struct {
	Tools      []Tool  `json:"tools"`
	SpecTitle  string  `json:"specTitle"`
	SpecVersion string `json:"specVersion"`
	CachedAt   int64   `json:"cachedAt"`
}

// NewCache creates a new Cache instance.
func NewCache(logger *slog.Logger) *Cache {
	cacheDir := filepath.Join(os.TempDir(), CacheDirName)
	return NewCacheWithDir(logger, cacheDir, DefaultCacheTTL)
}

// NewCacheWithDir creates a new Cache with a specific cache directory and TTL.
func NewCacheWithDir(logger *slog.Logger, cacheDir string, ttl time.Duration) *Cache {
	return &Cache{
		logger:    logger,
		items:     make(map[string]*CacheItem),
		cacheDir:  cacheDir,
		defaultTTL: ttl,
	}
}

// SetCacheDir sets the cache directory path.
func (c *Cache) SetCacheDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheDir = dir
}

// Get retrieves cached tools by URL if valid.
func (c *Cache) Get(ctx context.Context, specURL string) ([]Tool, bool) {
	c.mu.RLock()
	item, ok := c.items[specURL]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		c.logger.Debug("Cache expired", slog.String("url", specURL))
		c.invalidate(specURL)
		return nil, false
	}

	c.logger.Debug("Cache hit", slog.String("url", specURL))
	return item.Tools, true
}

// Set stores tools in the cache.
func (c *Cache) Set(specURL string, tools []Tool, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[specURL] = &CacheItem{
		Tools:      tools,
		SpecURL:    specURL,
		FetchedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
	}

	c.logger.Debug("Cached tools",
		slog.String("url", specURL),
		slog.Int("toolCount", len(tools)),
		slog.Duration("ttl", ttl),
	)
}

// GetOrFetch retrieves tools from cache, or fetches and parses the spec if not cached.
func (c *Cache) GetOrFetch(
	ctx context.Context,
	specURL string,
	parser *Parser,
	generator *Generator,
	ttl time.Duration,
) ([]Tool, error) {
	// Check cache first
	if tools, ok := c.Get(ctx, specURL); ok {
		return tools, nil
	}

	// Ensure cache directory exists
	if err := c.ensureCacheDir(); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if we have a file cache
	if cachedTools, ok := c.loadFromFileCache(specURL); ok {
		// Validate against remote spec age
		remoteAge, err := parser.GetRemoteSpecAge(ctx, specURL)
		if err != nil {
			c.logger.Warn("Failed to get remote spec age, using cached", slog.String("error", err.Error()))
		}

		cacheFile := c.cacheFilePath(specURL)
		cacheModTime := getFileModTime(cacheFile)

		if remoteAge != nil && cacheModTime.After(*remoteAge) {
			// Cache is still valid
			c.Set(specURL, cachedTools, ttl)
			return cachedTools, nil
		}
	}

	// Fetch and parse spec
	spec, err := parser.Parse(ctx, specURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	// Generate tools
	tools := generator.ExtractTools(spec.SpecV3)
	if tools == nil {
		tools = []Tool{}
	}

	// Cache the tools
	c.Set(specURL, tools, ttl)

	// Save to file cache
	if err := c.saveToFileCache(specURL, tools, spec); err != nil {
		c.logger.Warn("Failed to save to file cache", slog.String("error", err.Error()))
	}

	return tools, nil
}

// saveToFileCache saves tools to a JSON file in the cache directory.
func (c *Cache) saveToFileCache(specURL string, tools []Tool, spec *Spec) error {
	cacheFile := c.cacheFilePath(specURL)

	cachedTools := CachedTools{
		Tools:       tools,
		SpecTitle:   spec.Title,
		SpecVersion: spec.Version,
		CachedAt:    time.Now().Unix(),
	}

	data, err := json.Marshal(cachedTools)
	if err != nil {
		return fmt.Errorf("failed to marshal cached tools: %w", err)
	}

	// Write atomically by writing to temp file first
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	c.logger.Debug("Saved tools to file cache",
		slog.String("file", cacheFile),
		slog.Int("toolCount", len(tools)),
	)

	return nil
}

// loadFromFileCache loads tools from the file cache.
func (c *Cache) loadFromFileCache(specURL string) ([]Tool, bool) {
	cacheFile := c.cacheFilePath(specURL)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var cachedTools CachedTools
	if err := json.Unmarshal(data, &cachedTools); err != nil {
		c.logger.Warn("Failed to parse cache file, removing", slog.String("file", cacheFile))
		_ = os.Remove(cacheFile)
		return nil, false
	}

	return cachedTools.Tools, true
}

// cacheFilePath returns the path to the cache file for a given URL.
func (c *Cache) cacheFilePath(specURL string) string {
	// Use URL hash as filename
	hash := hashURL(specURL)
	return filepath.Join(c.cacheDir, fmt.Sprintf("tools_%s.json", hash))
}

// hashURL creates a simple hash of a URL for use as filename.
func hashURL(specURL string) string {
	// Simple hash using string conversion - good enough for cache filenames
	hash := 0
	for _, c := range specURL {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("%x", hash)
}

// getFileModTime returns the modification time of a file.
func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// ensureCacheDir ensures the cache directory exists.
func (c *Cache) ensureCacheDir() error {
	// Re-check lock
	c.mu.RLock()
	cacheDir := c.cacheDir
	c.mu.RUnlock()

	if _, err := os.Stat(cacheDir); err == nil {
		return nil
	}

	return os.MkdirAll(cacheDir, 0700)
}

// invalidate removes an item from the cache.
func (c *Cache) invalidate(specURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, specURL)

	// Also remove file cache
	cacheFile := c.cacheFilePath(specURL)
	_ = os.Remove(cacheFile)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear in-memory cache
	c.items = make(map[string]*CacheItem)

	// Clear file cache
	_ = os.RemoveAll(c.cacheDir)
	_ = os.MkdirAll(c.cacheDir, 0700)

	c.logger.Info("Cache cleared")
}

// ClearURL removes cache for a specific URL.
func (c *Cache) ClearURL(specURL string) {
	c.invalidate(specURL)
	c.logger.Info("Cache cleared for URL", slog.String("url", specURL))
}

// Stats returns cache statistics.
func (c *Cache) Stats() (itemCount int, cacheDir string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items), c.cacheDir
}

// FileCacheExists checks if a file cache exists for the given URL.
func (c *Cache) FileCacheExists(specURL string) bool {
	cacheFile := c.cacheFilePath(specURL)
	_, err := os.Stat(cacheFile)
	return err == nil
}

// GetCachedToolsInfo returns info about cached tools without loading them.
func (c *Cache) GetCachedToolsInfo(specURL string) (title, version string, cachedAt time.Time, ok bool) {
	cacheFile := c.cacheFilePath(specURL)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", "", time.Time{}, false
	}

	var cachedTools CachedTools
	if err := json.Unmarshal(data, &cachedTools); err != nil {
		return "", "", time.Time{}, false
	}

	return cachedTools.SpecTitle, cachedTools.SpecVersion,
		time.Unix(cachedTools.CachedAt, 0), true
}

// PurgeExpired removes all expired items from the cache and file cache.
func (c *Cache) PurgeExpired() {
	c.mu.Lock()
	now := time.Now()

	for specURL, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, specURL)
		}
	}
	c.mu.Unlock()

	// Purge expired file caches
	c.purgeExpiredFileCache()
}

// purgeExpiredFileCache removes expired cache files.
func (c *Cache) purgeExpiredFileCache() {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-c.defaultTTL)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "tools_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(c.cacheDir, entry.Name())
			_ = os.Remove(filePath)
		}
	}
}

// OpenAPI Spec caching functions

// GetSpec retrieves a cached spec by URL (not tools).
func (c *Cache) GetSpec(url string) (*Spec, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[url]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Spec, true
}

// SetSpec stores a spec in the cache.
func (c *Cache) SetSpec(url string, spec *Spec, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[url] = &CacheItem{
		Spec:       spec,
		SpecURL:    url,
		FetchedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
	}
}

// LoadSpecFromCache loads a cached OpenAPI spec (uses kin-openapi parsing).
func (c *Cache) LoadSpecFromCache(specURL string) (*openapi3.T, error) {
	cacheFile := c.cacheFilePath(specURL)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	// Re-parse the spec from the raw data stored in cache
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	return loader.LoadFromData(data)
}
