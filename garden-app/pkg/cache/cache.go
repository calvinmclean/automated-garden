// Package cache provides a simple in-memory cache with TTL support.
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached value with its expiration time.
type Entry[T any] struct {
	Value      T
	Expiration time.Time
}

// Cache is a generic in-memory cache with TTL support.
// T is the type of values stored in the cache.
type Cache[T any] struct {
	mu      sync.RWMutex
	entries map[string]Entry[T]
	ttl     time.Duration
}

// New creates a new Cache with the specified TTL for all entries.
func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		entries: make(map[string]Entry[T]),
		ttl:     ttl,
	}
}

// Get retrieves a value from the cache by key.
// Returns the value and true if found and not expired, otherwise returns zero value and false.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || time.Now().After(entry.Expiration) {
		var zero T
		return zero, false
	}
	return entry.Value, true
}

// Set stores a value in the cache with the configured TTL.
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry[T]{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache by key.
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]Entry[T])
}

// Size returns the number of entries in the cache (including expired ones).
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
