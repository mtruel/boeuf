package handlers

import (
	"sync"
	"time"
)

// IdempotenceCache provides deduplication for client commands
// TTL-based cache prevents double-execution of same command (network retries)
type IdempotenceCache struct {
	mu      sync.RWMutex
	cache   map[string]map[string]*cacheEntry // sessionID -> clientMsgID -> entry
	ttl     time.Duration
	cleanup *time.Ticker
}

type cacheEntry struct {
	eventSeq  int64
	timestamp time.Time
}

// NewIdempotenceCache creates a new IdempotenceCache with TTL-based cleanup
func NewIdempotenceCache(ttl time.Duration) *IdempotenceCache {
	cache := &IdempotenceCache{
		cache:   make(map[string]map[string]*cacheEntry),
		ttl:     ttl,
		cleanup: time.NewTicker(ttl / 2), // Cleanup every ttl/2
	}

	// Start background cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves cached eventSeq for a clientMsgID (if exists and not expired)
func (c *IdempotenceCache) Get(sessionID, clientMsgID string) (int64, bool) {
	c.mu.RLock()

	sessionCache, ok := c.cache[sessionID]
	if !ok {
		c.mu.RUnlock()
		return 0, false
	}

	entry, ok := sessionCache[clientMsgID]
	if !ok {
		c.mu.RUnlock()
		return 0, false
	}

	// Check if entry expired
	if time.Since(entry.timestamp) > c.ttl {
		c.mu.RUnlock()
		// Lazy delete expired entry
		c.mu.Lock()
		delete(sessionCache, clientMsgID)
		if len(sessionCache) == 0 {
			delete(c.cache, sessionID)
		}
		c.mu.Unlock()
		return 0, false
	}

	eventSeq := entry.eventSeq
	c.mu.RUnlock()
	return eventSeq, true
}

// Set stores eventSeq for a clientMsgID with current timestamp
func (c *IdempotenceCache) Set(sessionID, clientMsgID string, eventSeq int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache[sessionID] == nil {
		c.cache[sessionID] = make(map[string]*cacheEntry)
	}

	c.cache[sessionID][clientMsgID] = &cacheEntry{
		eventSeq:  eventSeq,
		timestamp: time.Now(),
	}

	// Lazy cleanup: remove expired entries from this session only
	c.lazyCleanupSession(sessionID)
}

// lazyCleanupSession removes expired entries for a single session (caller must hold lock)
func (c *IdempotenceCache) lazyCleanupSession(sessionID string) {
	sessionCache := c.cache[sessionID]
	now := time.Now()

	for clientMsgID, entry := range sessionCache {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(sessionCache, clientMsgID)
		}
	}

	// Remove empty session cache
	if len(sessionCache) == 0 {
		delete(c.cache, sessionID)
	}
}

// cleanupLoop removes expired entries periodically
func (c *IdempotenceCache) cleanupLoop() {
	for range c.cleanup.C {
		c.removeExpired()
	}
}

// removeExpired deletes all expired entries from cache
func (c *IdempotenceCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for sessionID, sessionCache := range c.cache {
		for clientMsgID, entry := range sessionCache {
			if now.Sub(entry.timestamp) > c.ttl {
				delete(sessionCache, clientMsgID)
			}
		}

		// Remove empty session caches
		if len(sessionCache) == 0 {
			delete(c.cache, sessionID)
		}
	}
}

// Stop stops the cleanup ticker (for graceful shutdown)
func (c *IdempotenceCache) Stop() {
	c.cleanup.Stop()
}
