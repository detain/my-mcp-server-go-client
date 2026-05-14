// Package openapi provides caching for OpenAPI specifications.
package openapi

import (
	"log/slog"
	"sync"
	"time"
)

// Cache provides caching for OpenAPI specs.
type Cache struct {
	logger *slog.Logger
	mu     sync.RWMutex
	items  map[string]*CacheItem
}

// CacheItem represents a cached item with expiration.
type CacheItem struct {
	Spec      *Spec
	ExpiresAt time.Time
}

// NewCache creates a new Cache instance.
func NewCache(logger *slog.Logger) *Cache {
	return &Cache{
		logger: logger,
		items:  make(map[string]*CacheItem),
	}
}

// Get retrieves a cached spec by URL.
func (c *Cache) Get(url string) (*Spec, bool) {
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

// Set stores a spec in the cache.
func (c *Cache) Set(url string, spec *Spec, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[url] = &CacheItem{
		Spec:      spec,
		ExpiresAt: time.Now().Add(ttl),
	}
}
