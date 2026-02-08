package session

import (
	"testing"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Migrate tables
	err = db.AutoMigrate(&models.Session{}, &models.SessionParticipant{}, &models.SessionInvite{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// setupTestHub creates a test hub with buffered channels
func setupTestHub() *realtime.Hub {
	return realtime.NewHub()
}

func TestNewPlayerPoller(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()

	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	if poller == nil {
		t.Fatal("Expected poller to be created")
	}
	if poller.interval != 5*time.Second {
		t.Errorf("Expected interval 5s, got %v", poller.interval)
	}
	if len(poller.activePollers) != 0 {
		t.Errorf("Expected 0 active pollers, got %d", len(poller.activePollers))
	}
}

func TestNewPlayerPoller_DefaultInterval(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()

	poller := NewPlayerPoller(db, nil, hub, 0) // 0 should default to 5s

	if poller.interval != 5*time.Second {
		t.Errorf("Expected default interval 5s, got %v", poller.interval)
	}
}

func TestPlayerPoller_StartAndStopSessionPolling(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	go hub.Run() // Start hub in background

	// Create test session and participant
	session := &models.Session{
		ID:        "test-session",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Active:    true,
	}
	db.Create(session)

	participant := &models.SessionParticipant{
		SessionID:  "test-session",
		UserID:     "test-user",
		JoinedAt:   time.Now(),
		Role:       "host",
		LastSeenAt: time.Now(),
		SyncState:  "synced",
	}
	db.Create(participant)

	poller := NewPlayerPoller(db, nil, hub, 100*time.Millisecond)

	// Start polling
	poller.StartSessionPolling("test-session")

	if !poller.IsPolling("test-session") {
		t.Error("Expected session to be polling")
	}

	// Wait briefly
	time.Sleep(50 * time.Millisecond)

	// Stop polling
	poller.StopSessionPolling("test-session")

	if poller.IsPolling("test-session") {
		t.Error("Expected session to stop polling")
	}

	poller.StopAll()
}

func TestPlayerPoller_DoubleStart(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()

	poller := NewPlayerPoller(db, nil, hub, 100*time.Millisecond)

	// Start twice
	poller.StartSessionPolling("test-session")
	poller.StartSessionPolling("test-session") // Should not create duplicate

	// Should only have one active poller
	poller.mu.RLock()
	count := len(poller.activePollers)
	poller.mu.RUnlock()

	if count != 1 {
		t.Errorf("Expected 1 active poller, got %d", count)
	}

	poller.StopAll()
}

func TestPlayerPoller_HasActiveSyncedParticipants(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	// No participants - should return false
	if poller.hasActiveSyncedParticipants("nonexistent") {
		t.Error("Expected false for session with no participants")
	}

	// Create test session
	session := &models.Session{
		ID:        "test-session",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Active:    true,
	}
	db.Create(session)

	// Create inactive participant (old last_seen)
	inactiveParticipant := &models.SessionParticipant{
		SessionID:  "test-session",
		UserID:     "inactive-user",
		JoinedAt:   time.Now(),
		Role:       "participant",
		LastSeenAt: time.Now().Add(-1 * time.Hour), // Very old
		SyncState:  "synced",
	}
	db.Create(inactiveParticipant)

	if poller.hasActiveSyncedParticipants("test-session") {
		t.Error("Expected false for session with only inactive participants")
	}

	// Create active participant
	activeParticipant := &models.SessionParticipant{
		SessionID:  "test-session",
		UserID:     "active-user",
		JoinedAt:   time.Now(),
		Role:       "host",
		LastSeenAt: time.Now(), // Recent
		SyncState:  "synced",
	}
	db.Create(activeParticipant)

	if !poller.hasActiveSyncedParticipants("test-session") {
		t.Error("Expected true for session with active synced participant")
	}
}

func TestPlayerPoller_GetActiveSyncedParticipant(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	// No participants - should return nil
	if poller.getActiveSyncedParticipant("nonexistent") != nil {
		t.Error("Expected nil for session with no participants")
	}

	// Create test session
	session := &models.Session{
		ID:        "test-session",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Active:    true,
	}
	db.Create(session)

	// Create active participant
	participant := &models.SessionParticipant{
		SessionID:  "test-session",
		UserID:     "test-user",
		JoinedAt:   time.Now(),
		Role:       "host",
		LastSeenAt: time.Now(),
		SyncState:  "synced",
	}
	db.Create(participant)

	result := poller.getActiveSyncedParticipant("test-session")
	if result == nil {
		t.Fatal("Expected participant, got nil")
	}
	if result.UserID != "test-user" {
		t.Errorf("Expected user 'test-user', got '%s'", result.UserID)
	}
}

func TestPlayerPoller_StateChangeDetection(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	state1 := &spotify.PlayerState{
		TrackID:    "spotify:track:123",
		IsPlaying:  true,
		PositionMs: 1000,
	}

	state2 := &spotify.PlayerState{
		TrackID:    "spotify:track:123",
		IsPlaying:  true,
		PositionMs: 5000, // Different position
	}

	state3 := &spotify.PlayerState{
		TrackID:    "spotify:track:456", // Different track
		IsPlaying:  true,
		PositionMs: 1000,
	}

	// First state should always be considered changed
	if !poller.hasStateChanged("session-1", state1) {
		t.Error("Expected first state to be considered changed")
	}

	// Update last state
	poller.updateLastState("session-1", state1)

	// Same state (within 1 second position tolerance) should not be changed
	state1Same := &spotify.PlayerState{
		TrackID:    "spotify:track:123",
		IsPlaying:  true,
		PositionMs: 1500, // Within 1 second of 1000ms
	}
	if poller.hasStateChanged("session-1", state1Same) {
		t.Error("Expected same state to not be considered changed")
	}

	// Different position (beyond 1 second) should be changed
	if !poller.hasStateChanged("session-1", state2) {
		t.Error("Expected different position to be considered changed")
	}

	// Update to state2
	poller.updateLastState("session-1", state2)

	// Different track should be changed
	if !poller.hasStateChanged("session-1", state3) {
		t.Error("Expected different track to be considered changed")
	}
}

func TestPlayerPoller_StateHash(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	state := &spotify.PlayerState{
		TrackID:    "spotify:track:123",
		IsPlaying:  true,
		PositionMs: 5000, // Should be divided by 1000 for hashing
	}

	hash := poller.stateHash(state)
	expected := "spotify:track:123|true|5" // 5000/1000 = 5

	if hash != expected {
		t.Errorf("Expected hash '%s', got '%s'", expected, hash)
	}
}

func TestPlayerPoller_StopAll(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 100*time.Millisecond)

	// Start multiple sessions
	poller.StartSessionPolling("session-1")
	poller.StartSessionPolling("session-2")
	poller.StartSessionPolling("session-3")

	// Verify all are polling
	if !poller.IsPolling("session-1") || !poller.IsPolling("session-2") || !poller.IsPolling("session-3") {
		t.Error("Expected all sessions to be polling")
	}

	// Stop all
	poller.StopAll()

	// Verify none are polling
	if poller.IsPolling("session-1") || poller.IsPolling("session-2") || poller.IsPolling("session-3") {
		t.Error("Expected no sessions to be polling after StopAll")
	}

	// Verify lastStates cleared
	poller.stateMu.RLock()
	stateCount := len(poller.lastStates)
	poller.stateMu.RUnlock()

	if stateCount != 0 {
		t.Errorf("Expected 0 last states, got %d", stateCount)
	}
}

func TestPlayerPoller_DetermineEventType(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	state := &spotify.PlayerState{
		TrackID:   "spotify:track:123",
		IsPlaying: true,
	}

	// Currently always returns PLAYER_STATE_UPDATE
	eventType := poller.determineEventType("session-1", state)
	if eventType != "PLAYER_STATE_UPDATE" {
		t.Errorf("Expected PLAYER_STATE_UPDATE, got %s", eventType)
	}
}

func TestPlayerPoller_PollOnce_NoActiveParticipants(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	// Poll without any participants - should not panic or error
	poller.pollOnce("nonexistent-session")
}

func TestPlayerPoller_PollOnce_WithParticipant(t *testing.T) {
	db := setupTestDB(t)
	hub := setupTestHub()
	go hub.Run()

	// Create test session and participant
	session := &models.Session{
		ID:        "test-session",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Active:    true,
	}
	db.Create(session)

	participant := &models.SessionParticipant{
		SessionID:  "test-session",
		UserID:     "test-user",
		JoinedAt:   time.Now(),
		Role:       "host",
		LastSeenAt: time.Now(),
		SyncState:  "synced",
	}
	db.Create(participant)

	poller := NewPlayerPoller(db, nil, hub, 5*time.Second)

	// Poll - should not panic even with nil spotify client
	// (it will fail to get player state but shouldn't crash)
	poller.pollOnce("test-session")
}
