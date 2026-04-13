package server

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// OAuthStateCache stores temporary state parameters for OAuth flows
type OAuthStateCache struct {
	cache map[string]oauthStateEntry
	mu    sync.RWMutex
}

type oauthStateEntry struct {
	WeatherClientID string
	RedirectURI     string
	ExpiresAt       time.Time
}

// NewOAuthStateCache creates a new OAuth state cache
func NewOAuthStateCache() *OAuthStateCache {
	cache := &OAuthStateCache{
		cache: make(map[string]oauthStateEntry),
	}
	// Start cleanup goroutine
	go cache.cleanupLoop()
	return cache
}

// Store generates a random state, stores it with the weather client ID and redirect URI, and returns the state
func (c *OAuthStateCache) Store(weatherClientID string, redirectURI string, ttl time.Duration) string {
	state := generateRandomState()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[state] = oauthStateEntry{
		WeatherClientID: weatherClientID,
		RedirectURI:     redirectURI,
		ExpiresAt:       time.Now().Add(ttl),
	}

	return state
}

// Validate checks if the state exists and is not expired, returns the weather client ID and redirect URI if valid
func (c *OAuthStateCache) Validate(state string) (string, string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.cache[state]
	if !exists {
		return "", "", false
	}

	// Delete after use (one-time)
	delete(c.cache, state)

	if time.Now().After(entry.ExpiresAt) {
		return "", "", false
	}

	return entry.WeatherClientID, entry.RedirectURI, true
}

// cleanupLoop periodically removes expired entries
func (c *OAuthStateCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.Cleanup()
	}
}

// Cleanup removes expired entries
func (c *OAuthStateCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for state, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, state)
		}
	}
}

// generateRandomState generates a random 32-byte hex string
func generateRandomState() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based state if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}
