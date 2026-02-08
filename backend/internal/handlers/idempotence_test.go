package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdempotenceCache_SetAndGet(t *testing.T) {
	cache := NewIdempotenceCache(1 * time.Minute)
	defer cache.Stop()

	sessionID := "sess_123"
	clientMsgID := "msg_abc"
	eventSeq := int64(42)

	// Initially not present
	_, found := cache.Get(sessionID, clientMsgID)
	assert.False(t, found, "Entry should not exist initially")

	// Set entry
	cache.Set(sessionID, clientMsgID, eventSeq)

	// Now should be present
	retrievedSeq, found := cache.Get(sessionID, clientMsgID)
	assert.True(t, found, "Entry should exist after Set")
	assert.Equal(t, eventSeq, retrievedSeq, "Retrieved eventSeq should match")
}

func TestIdempotenceCache_DifferentSessions(t *testing.T) {
	cache := NewIdempotenceCache(1 * time.Minute)
	defer cache.Stop()

	session1 := "sess_1"
	session2 := "sess_2"
	clientMsgID := "msg_abc" // Same clientMsgID in different sessions

	cache.Set(session1, clientMsgID, 10)
	cache.Set(session2, clientMsgID, 20)

	// Should retrieve correct eventSeq per session
	seq1, _ := cache.Get(session1, clientMsgID)
	seq2, _ := cache.Get(session2, clientMsgID)

	assert.Equal(t, int64(10), seq1)
	assert.Equal(t, int64(20), seq2)
}

func TestIdempotenceCache_TTLExpiration(t *testing.T) {
	shortTTL := 100 * time.Millisecond
	cache := NewIdempotenceCache(shortTTL)
	defer cache.Stop()

	sessionID := "sess_123"
	clientMsgID := "msg_abc"
	eventSeq := int64(42)

	// Set entry
	cache.Set(sessionID, clientMsgID, eventSeq)

	// Should be present immediately
	_, found := cache.Get(sessionID, clientMsgID)
	assert.True(t, found, "Entry should exist immediately")

	// Wait for TTL to expire
	time.Sleep(shortTTL + 50*time.Millisecond)

	// Should now be considered expired
	_, found = cache.Get(sessionID, clientMsgID)
	assert.False(t, found, "Entry should be expired after TTL")
}

func TestIdempotenceCache_CleanupRemovesExpired(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	cache := NewIdempotenceCache(shortTTL)
	defer cache.Stop()

	sessionID := "sess_123"
	clientMsgID1 := "msg_1"
	clientMsgID2 := "msg_2"

	// Add entries
	cache.Set(sessionID, clientMsgID1, 10)
	time.Sleep(30 * time.Millisecond)
	cache.Set(sessionID, clientMsgID2, 20)

	// Wait for first entry to expire and cleanup to run
	time.Sleep(shortTTL)

	// First entry should be expired (not returned by Get)
	_, found := cache.Get(sessionID, clientMsgID1)
	assert.False(t, found, "First entry should be expired")

	// Second entry might still be valid or just expired depending on timing
	// We don't assert on it here to avoid flaky test

	// Wait for cleanup cycle
	time.Sleep(shortTTL / 2)

	// Verify cache is cleaned up (check internals)
	cache.mu.RLock()
	sessionCache := cache.cache[sessionID]
	cache.mu.RUnlock()

	// First entry should be removed by cleanup
	if sessionCache != nil {
		_, exists := sessionCache[clientMsgID1]
		assert.False(t, exists, "Expired entry should be removed by cleanup")
	}
}

func TestIdempotenceCache_ConcurrentAccess(t *testing.T) {
	cache := NewIdempotenceCache(1 * time.Minute)
	defer cache.Stop()

	sessionID := "sess_123"
	const goroutines = 10
	const iterations = 100

	done := make(chan bool, goroutines)

	// Concurrent writes
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				clientMsgID := string(rune('a' + id))
				cache.Set(sessionID, clientMsgID, int64(i))
			}
			done <- true
		}(g)
	}

	// Concurrent reads
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				clientMsgID := string(rune('a' + id))
				cache.Get(sessionID, clientMsgID)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines*2; i++ {
		<-done
	}

	// No assertion needed - test passes if no race detector errors
}
