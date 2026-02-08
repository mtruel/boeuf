// Package session provides session management and player state polling
package session

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/gorm"
)

// PlayerPoller manages periodic polling of Spotify player state for active sessions
type PlayerPoller struct {
	db            *gorm.DB
	spotifyClient *spotify.Client
	hub           *realtime.Hub
	interval      time.Duration

	// Active polling goroutines: sessionID -> cancel function
	activePollers map[string]context.CancelFunc
	mu            sync.RWMutex

	// Last known state for change detection: sessionID -> state hash
	lastStates map[string]string
	stateMu    sync.RWMutex
}

// NewPlayerPoller creates a new PlayerPoller instance
func NewPlayerPoller(db *gorm.DB, spotifyClient *spotify.Client, hub *realtime.Hub, interval time.Duration) *PlayerPoller {
	if interval == 0 {
		interval = 5 * time.Second // Default: 5 seconds
	}

	return &PlayerPoller{
		db:            db,
		spotifyClient: spotifyClient,
		hub:           hub,
		interval:      interval,
		activePollers: make(map[string]context.CancelFunc),
		lastStates:    make(map[string]string),
	}
}

// StartSessionPolling begins polling for a specific session
func (p *PlayerPoller) StartSessionPolling(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Don't start if already polling
	if _, exists := p.activePollers[sessionID]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.activePollers[sessionID] = cancel

	go p.pollLoop(ctx, sessionID)
	log.Printf("[Poller] Started polling for session %s", sessionID)
}

// StopSessionPolling stops polling for a specific session
func (p *PlayerPoller) StopSessionPolling(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cancel, exists := p.activePollers[sessionID]; exists {
		cancel()
		delete(p.activePollers, sessionID)
		log.Printf("[Poller] Stopped polling for session %s", sessionID)
	}

	// Clean up last state
	p.stateMu.Lock()
	delete(p.lastStates, sessionID)
	p.stateMu.Unlock()
}

// StopAll stops all active polling goroutines
func (p *PlayerPoller) StopAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for sessionID, cancel := range p.activePollers {
		cancel()
		delete(p.activePollers, sessionID)
		log.Printf("[Poller] Stopped polling for session %s", sessionID)
	}

	p.stateMu.Lock()
	p.lastStates = make(map[string]string)
	p.stateMu.Unlock()
}

// IsPolling returns true if the session is currently being polled
func (p *PlayerPoller) IsPolling(sessionID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.activePollers[sessionID]
	return exists
}

// pollLoop is the main polling loop for a session
func (p *PlayerPoller) pollLoop(ctx context.Context, sessionID string) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Initial poll immediately
	p.pollOnce(sessionID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if session still has active synced participants
			if !p.hasActiveSyncedParticipants(sessionID) {
				log.Printf("[Poller] No active synced participants for session %s, stopping polling", sessionID)
				p.StopSessionPolling(sessionID)
				return
			}
			p.pollOnce(sessionID)
		}
	}
}

// pollOnce performs a single polling cycle
func (p *PlayerPoller) pollOnce(sessionID string) {
	// Check if spotify client is available
	if p.spotifyClient == nil {
		log.Printf("[Poller] No Spotify client available for session %s", sessionID)
		return
	}

	// Get an active synced participant to query Spotify
	participant := p.getActiveSyncedParticipant(sessionID)
	if participant == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query Spotify for player state
	state, err := p.spotifyClient.GetPlayerState(ctx, participant.UserID)
	if err != nil {
		if _, isRateLimit := err.(*spotify.RateLimitedError); isRateLimit {
			log.Printf("[Poller] Rate limited for session %s, will retry with backoff", sessionID)
		}
		// Log error but don't stop polling - transient errors happen
		log.Printf("[Poller] Failed to get player state for session %s: %v", sessionID, err)
		return
	}

	// Check if state changed
	if !p.hasStateChanged(sessionID, state) {
		return // No change, don't broadcast
	}

	// Determine event type based on what changed
	eventType := p.determineEventType(sessionID, state)

	// Update last known state
	p.updateLastState(sessionID, state)

	// Broadcast event
	p.broadcastEvent(sessionID, participant.UserID, eventType, state)

	log.Printf("[Poller] State change detected for session %s, broadcasted %s (track: %s, playing: %v, position: %dms)",
		sessionID, eventType, state.TrackName, state.IsPlaying, state.PositionMs)
}

// hasActiveSyncedParticipants checks if session has any active synced participants
func (p *PlayerPoller) hasActiveSyncedParticipants(sessionID string) bool {
	var count int64
	err := p.db.Model(&models.SessionParticipant{}).
		Where("session_id = ? AND sync_state = ? AND last_seen_at > ?",
			sessionID, "synced", time.Now().Add(-30*time.Second)).
		Count(&count).Error

	if err != nil {
		log.Printf("[Poller] Failed to count active participants for session %s: %v", sessionID, err)
		return false
	}

	return count > 0
}

// getActiveSyncedParticipant returns an active synced participant for the session
func (p *PlayerPoller) getActiveSyncedParticipant(sessionID string) *models.SessionParticipant {
	var participant models.SessionParticipant
	err := p.db.Where("session_id = ? AND sync_state = ? AND last_seen_at > ?",
		sessionID, "synced", time.Now().Add(-30*time.Second)).
		Order("last_seen_at DESC").
		First(&participant).Error

	if err != nil {
		return nil
	}

	return &participant
}

// stateHash creates a hash-like string for change detection
func (p *PlayerPoller) stateHash(state *spotify.PlayerState) string {
	return fmt.Sprintf("%s|%v|%d", state.TrackID, state.IsPlaying, state.PositionMs/1000)
}

// hasStateChanged checks if player state has meaningfully changed
func (p *PlayerPoller) hasStateChanged(sessionID string, state *spotify.PlayerState) bool {
	p.stateMu.RLock()
	lastHash, exists := p.lastStates[sessionID]
	p.stateMu.RUnlock()

	currentHash := p.stateHash(state)

	if !exists {
		return true // First time seeing this session
	}

	return lastHash != currentHash
}

// updateLastState stores the current state for future change detection
func (p *PlayerPoller) updateLastState(sessionID string, state *spotify.PlayerState) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	p.lastStates[sessionID] = p.stateHash(state)
}

// determineEventType decides which event type to broadcast based on changes
func (p *PlayerPoller) determineEventType(sessionID string, state *spotify.PlayerState) string {
	// Check if track ID changed
	p.stateMu.RLock()
	lastHash, exists := p.lastStates[sessionID]
	p.stateMu.RUnlock()

	if exists {
		// Parse last hash to extract trackId (format: "trackId|isPlaying|position")
		parts := strings.Split(lastHash, "|")
		if len(parts) > 0 {
			lastTrackID := parts[0]
			// Compare track IDs - both must be non-empty to detect a real change
			if lastTrackID != state.TrackID && lastTrackID != "" && state.TrackID != "" {
				return "TRACK_CHANGED"
			}
		}
	}

	// Default to PLAYER_STATE_UPDATE for position/playing state changes
	return "PLAYER_STATE_UPDATE"
}

// broadcastEvent sends the appropriate WebSocket event
func (p *PlayerPoller) broadcastEvent(sessionID, userID, eventType string, state *spotify.PlayerState) {
	// Convert to NowPlayingInfo
	track := &realtime.NowPlayingInfo{
		TrackID:    state.TrackID,
		TrackName:  state.TrackName,
		Artist:     state.Artist,
		Album:      state.Album,
		DurationMs: state.DurationMs,
		ImageURL:   state.ImageURL,
		IsPlaying:  state.IsPlaying,
		PositionMs: state.PositionMs,
	}

	// Build player state payload
	playerState := map[string]interface{}{
		"isPlaying":  state.IsPlaying,
		"positionMs": state.PositionMs,
		"track":      track,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	switch eventType {
	case "TRACK_CHANGED":
		p.hub.BroadcastTrackChanged(sessionID, userID, playerState)
	default:
		// PLAYER_STATE_UPDATE - polling broadcast uses canonical schema (no userId)
		p.hub.BroadcastPlayerStateUpdate(sessionID, playerState)
	}
}
