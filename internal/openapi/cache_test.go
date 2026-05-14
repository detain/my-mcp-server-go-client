package openapi

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewCache(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	if cache.cacheDir == "" {
		t.Error("cacheDir should not be empty")
	}

	if cache.defaultTTL != DefaultCacheTTL {
		t.Errorf("Expected default TTL %v, got %v", DefaultCacheTTL, cache.defaultTTL)
	}
}

func TestNewCacheWithDir(t *testing.T) {
	logger := slog.Default()
	customDir := "/tmp/custom/cache/dir"
	customTTL := 30 * time.Minute

	cache := NewCacheWithDir(logger, customDir, customTTL)

	if cache.cacheDir != customDir {
		t.Errorf("Expected cacheDir %q, got %q", customDir, cache.cacheDir)
	}

	if cache.defaultTTL != customTTL {
		t.Errorf("Expected TTL %v, got %v", customTTL, cache.defaultTTL)
	}
}

func TestCache_SetCacheDir(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	newDir := "/tmp/new/cache/dir"
	cache.SetCacheDir(newDir)

	if cache.cacheDir != newDir {
		t.Errorf("Expected cacheDir %q, got %q", newDir, cache.cacheDir)
	}
}

func TestCache_Get_Set(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL := "https://example.com/spec.json"
	testTools := []Tool{
		{Name: "tool1", Description: "Test tool 1"},
		{Name: "tool2", Description: "Test tool 2"},
	}

	// Get should return false initially
	if tools, ok := cache.Get(context.Background(), testURL); ok {
		t.Errorf("Expected false for new URL, got tools: %v", tools)
	}

	// Set should store the tools
	cache.Set(testURL, testTools, time.Hour)

	// Get should now return the tools
	if tools, ok := cache.Get(context.Background(), testURL); !ok {
		t.Error("Expected true after Set, got false")
	} else if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestCache_Get_Expired(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL := "https://example.com/spec.json"
	testTools := []Tool{{Name: "tool1"}}

	// Set with very short TTL
	cache.Set(testURL, testTools, 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get should return false (expired)
	if _, ok := cache.Get(context.Background(), testURL); ok {
		t.Error("Expected false for expired item, got true")
	}
}

func TestCache_Clear(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL := "https://example.com/spec.json"
	testTools := []Tool{{Name: "tool1"}}

	cache.Set(testURL, testTools, time.Hour)

	// Clear should remove all items
	cache.Clear()

	if _, ok := cache.Get(context.Background(), testURL); ok {
		t.Error("Expected false after Clear, got true")
	}
}

func TestCache_ClearURL(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL1 := "https://example.com/spec1.json"
	testURL2 := "https://example.com/spec2.json"
	testTools := []Tool{{Name: "tool1"}}

	cache.Set(testURL1, testTools, time.Hour)
	cache.Set(testURL2, testTools, time.Hour)

	// Clear URL1 only
	cache.ClearURL(testURL1)

	if _, ok := cache.Get(context.Background(), testURL1); ok {
		t.Error("Expected false after ClearURL for URL1")
	}

	// URL2 should still be present
	if _, ok := cache.Get(context.Background(), testURL2); !ok {
		t.Error("Expected true for URL2 after clearing URL1")
	}
}

func TestCache_Stats(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	if count, _ := cache.Stats(); count != 0 {
		t.Errorf("Expected 0 items initially, got %d", count)
	}

	cache.Set("https://example.com/spec1.json", []Tool{{Name: "t1"}}, time.Hour)
	cache.Set("https://example.com/spec2.json", []Tool{{Name: "t2"}}, time.Hour)

	if count, _ := cache.Stats(); count != 2 {
		t.Errorf("Expected 2 items, got %d", count)
	}
}

func TestCache_FileCacheOperations(t *testing.T) {
	// Skip if we can't create temp directory
	tmpDir, err := os.MkdirTemp("", "openapi_cache_test")
	if err != nil {
		t.Skip("Cannot create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.Default()
	cache := NewCacheWithDir(logger, tmpDir, time.Hour)

	testURL := "https://example.com/spec.json"
	testTools := []Tool{
		{
			Name:        "testTool",
			Description: "A test tool",
			HTTPMethod:  "GET",
			Path:        "/test",
		},
	}

	// Create a minimal spec for file caching
	spec := &Spec{
		Title:   "Test API",
		Version: "1.0.0",
		SpecV3: &openapi3.T{
			Info: &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		},
	}

	// Save to file cache
	if err := cache.saveToFileCache(testURL, testTools, spec); err != nil {
		t.Fatalf("Failed to save to file cache: %v", err)
	}

	// Check file exists
	if !cache.FileCacheExists(testURL) {
		t.Error("Expected file cache to exist")
	}

	// Load from file cache
	loadedTools, ok := cache.loadFromFileCache(testURL)
	if !ok {
		t.Fatal("Failed to load from file cache")
	}

	if len(loadedTools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(loadedTools))
	}

	if loadedTools[0].Name != "testTool" {
		t.Errorf("Expected tool name 'testTool', got %q", loadedTools[0].Name)
	}

	// Check cached tools info
	title, version, cachedAt, ok := cache.GetCachedToolsInfo(testURL)
	if !ok {
		t.Fatal("Failed to get cached tools info")
	}

	if title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", title)
	}

	if version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", version)
	}

	if cachedAt.IsZero() {
		t.Error("Expected non-zero cachedAt")
	}
}

func TestCache_cacheFilePath(t *testing.T) {
	logger := slog.Default()
	cache := NewCacheWithDir(logger, "/tmp/cache", time.Hour)

	testURL := "https://example.com/spec.json"
	path := cache.cacheFilePath(testURL)

	// Should be in cache dir
	if !filepath.IsAbs(path) {
		t.Error("Expected absolute path")
	}

	if filepath.Dir(path) != "/tmp/cache" {
		t.Errorf("Expected directory /tmp/cache, got %s", filepath.Dir(path))
	}

	// Should start with tools_
	if filepath.Base(path)[:6] != "tools_" {
		t.Errorf("Expected filename to start with 'tools_', got %s", filepath.Base(path))
	}
}

func TestCache_hashURL(t *testing.T) {
	// Test that hashURL is deterministic
	url := "https://example.com/spec.json"
	hash1 := hashURL(url)
	hash2 := hashURL(url)

	if hash1 != hash2 {
		t.Error("hashURL should be deterministic")
	}

	// Test that different URLs produce different hashes
	differentURL := "https://example.com/different.json"
	hash3 := hashURL(differentURL)

	if hash1 == hash3 {
		t.Error("Different URLs should produce different hashes")
	}
}

func TestCache_ensureCacheDir(t *testing.T) {
	logger := slog.Default()
	tmpDir, err := os.MkdirTemp("", "openapi_cache_test")
	if err != nil {
		t.Skip("Cannot create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	cache := NewCacheWithDir(logger, filepath.Join(tmpDir, "nested", "cache"), time.Hour)

	// Should create nested directories
	if err := cache.ensureCacheDir(); err != nil {
		t.Fatalf("ensureCacheDir failed: %v", err)
	}

	// Directory should exist now
	if _, err := os.Stat(cache.cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory should exist after ensureCacheDir")
	}
}

func TestCache_PurgeExpired(t *testing.T) {
	logger := slog.Default()
	tmpDir, err := os.MkdirTemp("", "openapi_cache_test")
	if err != nil {
		t.Skip("Cannot create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	cache := NewCacheWithDir(logger, tmpDir, time.Hour)

	// Add items with very short TTL
	cache.Set("https://example.com/spec1.json", []Tool{{Name: "t1"}}, 1*time.Millisecond)
	cache.Set("https://example.com/spec2.json", []Tool{{Name: "t2"}}, time.Hour)

	// Wait for first item to expire
	time.Sleep(10 * time.Millisecond)

	// Purge should remove only expired items
	cache.PurgeExpired()

	// spec1 should be gone
	if _, ok := cache.Get(context.Background(), "https://example.com/spec1.json"); ok {
		t.Error("Expected spec1 to be purged")
	}

	// spec2 should still be present
	if _, ok := cache.Get(context.Background(), "https://example.com/spec2.json"); !ok {
		t.Error("Expected spec2 to still be present")
	}
}

func TestCache_GetSpec_SetSpec(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL := "https://example.com/spec.json"
	spec := &Spec{
		Title:   "Test API",
		Version: "1.0.0",
	}

	// GetSpec should return false initially
	if _, ok := cache.GetSpec(testURL); ok {
		t.Error("Expected false for new URL")
	}

	// SetSpec should store
	cache.SetSpec(testURL, spec, time.Hour)

	// GetSpec should return the spec
	if storedSpec, ok := cache.GetSpec(testURL); !ok {
		t.Error("Expected true after SetSpec")
	} else if storedSpec.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", storedSpec.Title)
	}
}

func TestCache_Set_ZeroTTL(t *testing.T) {
	logger := slog.Default()
	cache := NewCache(logger)

	testURL := "https://example.com/spec.json"
	testTools := []Tool{{Name: "tool1"}}

	// Set with zero TTL should use default
	cache.Set(testURL, testTools, 0)

	// Get should succeed (using default TTL)
	if _, ok := cache.Get(context.Background(), testURL); !ok {
		t.Error("Expected true with default TTL")
	}
}
