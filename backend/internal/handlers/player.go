package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/gorm"
)

// PlayerHandler handles player control commands (pause, resume, next, seek)
type PlayerHandler struct {
	sessionStore     *sessions.CookieStore
	db               *gorm.DB
	spotifyClient    *spotify.Client
	hub              *realtime.Hub
	idempotenceCache *IdempotenceCache
}

// NewPlayerHandler creates a new PlayerHandler
func NewPlayerHandler(sessionStore *sessions.CookieStore, db *gorm.DB, spotifyClient *spotify.Client, hub *realtime.Hub) *PlayerHandler {
	return &PlayerHandler{
		sessionStore:     sessionStore,
		db:               db,
		spotifyClient:    spotifyClient,
		hub:              hub,
		idempotenceCache: NewIdempotenceCache(1 * time.Minute),
	}
}

// PlayerCommandRequest represents the body for player commands
type PlayerCommandRequest struct {
	ClientMsgID string `json:"clientMsgId"` // For idempotence
	PositionMs  *int64 `json:"positionMs"`  // For seek command only
}

// PlayerCommandResponse represents the response from player commands
type PlayerCommandResponse struct {
	EventSeq int64 `json:"eventSeq"`
	Cached   bool  `json:"cached,omitempty"`
}

// PausePlayer handles POST /api/sessions/:sessionId/player/pause
func (h *PlayerHandler) PausePlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := mux.Vars(r)["sessionId"]
	userID := h.getUserIDFromSession(r)

	// Parse request body
	var req PlayerCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is acceptable (clientMsgId optional)
		req = PlayerCommandRequest{}
	}

	// Check idempotence cache
	if req.ClientMsgID != "" {
		if eventSeq, found := h.idempotenceCache.Get(sessionID, req.ClientMsgID); found {
			RespondJSON(w, http.StatusOK, PlayerCommandResponse{
				EventSeq: eventSeq,
				Cached:   true,
			})
			return
		}
	}

	// Validate participant is synced
	if err := h.validateSyncState(sessionID, userID); err != nil {
		RespondError(w, http.StatusForbidden, "PARTICIPANT_NOT_SYNCED", err.Error())
		return
	}

	// Get Spotify user ID for this participant
	spotifyUserID, err := h.getSpotifyUserID(userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get Spotify user")
		return
	}

	// Execute Spotify pause command
	if err := h.spotifyClient.Pause(ctx, spotifyUserID); err != nil {
		h.handleSpotifyError(w, err)
		return
	}

	// Trust but Verify: Read actual player state
	playerState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil {
		log.Printf("Warning: Failed to verify player state after pause: %v", err)
		// Continue anyway - command was sent
		playerState = &spotify.PlayerState{IsPlaying: false}
	}

	// Convert to payload format
	statePayload := h.convertPlayerState(playerState)

	// Broadcast event
	eventSeq := h.hub.BroadcastPlayerPaused(sessionID, userID, statePayload)

	// Persist event (synchronous - must succeed for data integrity AC#6)
	if err := h.persistEvent(sessionID, eventSeq, "PLAYER_PAUSED", statePayload); err != nil {
		log.Printf("ERROR: Failed to persist PLAYER_PAUSED event: %v", err)
		RespondError(w, http.StatusInternalServerError, "EVENT_PERSISTENCE_FAILED", "Failed to log event")
		return
	}

	// Cache for idempotence
	if req.ClientMsgID != "" {
		h.idempotenceCache.Set(sessionID, req.ClientMsgID, eventSeq)
	}

	RespondJSON(w, http.StatusOK, PlayerCommandResponse{
		EventSeq: eventSeq,
	})
}

// ResumePlayer handles POST /api/sessions/:sessionId/player/resume
func (h *PlayerHandler) ResumePlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := mux.Vars(r)["sessionId"]
	userID := h.getUserIDFromSession(r)

	var req PlayerCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = PlayerCommandRequest{}
	}

	// Check idempotence cache
	if req.ClientMsgID != "" {
		if eventSeq, found := h.idempotenceCache.Get(sessionID, req.ClientMsgID); found {
			RespondJSON(w, http.StatusOK, PlayerCommandResponse{
				EventSeq: eventSeq,
				Cached:   true,
			})
			return
		}
	}

	// Validate participant is synced
	if err := h.validateSyncState(sessionID, userID); err != nil {
		RespondError(w, http.StatusForbidden, "PARTICIPANT_NOT_SYNCED", err.Error())
		return
	}

	// Get Spotify user ID
	spotifyUserID, err := h.getSpotifyUserID(userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get Spotify user")
		return
	}

	// Execute Spotify resume command
	if err := h.spotifyClient.Resume(ctx, spotifyUserID); err != nil {
		h.handleSpotifyError(w, err)
		return
	}

	// Trust but Verify: Read actual player state
	playerState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil {
		log.Printf("Warning: Failed to verify player state after resume: %v", err)
		playerState = &spotify.PlayerState{IsPlaying: true}
	}

	// Convert to payload format
	statePayload := h.convertPlayerState(playerState)

	// Broadcast event
	eventSeq := h.hub.BroadcastPlayerResumed(sessionID, userID, statePayload)

	// Persist event (synchronous - must succeed for data integrity AC#6)
	if err := h.persistEvent(sessionID, eventSeq, "PLAYER_RESUMED", statePayload); err != nil {
		log.Printf("ERROR: Failed to persist PLAYER_RESUMED event: %v", err)
		RespondError(w, http.StatusInternalServerError, "EVENT_PERSISTENCE_FAILED", "Failed to log event")
		return
	}

	// Cache for idempotence
	if req.ClientMsgID != "" {
		h.idempotenceCache.Set(sessionID, req.ClientMsgID, eventSeq)
	}

	RespondJSON(w, http.StatusOK, PlayerCommandResponse{
		EventSeq: eventSeq,
	})
}

// NextTrack handles POST /api/sessions/:sessionId/player/next
func (h *PlayerHandler) NextTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := mux.Vars(r)["sessionId"]
	userID := h.getUserIDFromSession(r)

	var req PlayerCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = PlayerCommandRequest{}
	}

	// Check idempotence cache
	if req.ClientMsgID != "" {
		if eventSeq, found := h.idempotenceCache.Get(sessionID, req.ClientMsgID); found {
			RespondJSON(w, http.StatusOK, PlayerCommandResponse{
				EventSeq: eventSeq,
				Cached:   true,
			})
			return
		}
	}

	// Validate participant is synced
	if err := h.validateSyncState(sessionID, userID); err != nil {
		RespondError(w, http.StatusForbidden, "PARTICIPANT_NOT_SYNCED", err.Error())
		return
	}

	// Get Spotify user ID
	spotifyUserID, err := h.getSpotifyUserID(userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get Spotify user")
		return
	}

	// Execute Spotify next command
	if err := h.spotifyClient.Next(ctx, spotifyUserID); err != nil {
		h.handleSpotifyError(w, err)
		return
	}

	// Trust but Verify: Read actual player state
	playerState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil {
		log.Printf("Warning: Failed to verify player state after next: %v", err)
		playerState = &spotify.PlayerState{}
	}

	// Convert to payload format
	statePayload := h.convertPlayerState(playerState)

	// Broadcast event
	eventSeq := h.hub.BroadcastTrackChanged(sessionID, userID, statePayload)

	// Persist event (synchronous - must succeed for data integrity AC#6)
	if err := h.persistEvent(sessionID, eventSeq, "TRACK_CHANGED", statePayload); err != nil {
		log.Printf("ERROR: Failed to persist TRACK_CHANGED event: %v", err)
		RespondError(w, http.StatusInternalServerError, "EVENT_PERSISTENCE_FAILED", "Failed to log event")
		return
	}

	// Cache for idempotence
	if req.ClientMsgID != "" {
		h.idempotenceCache.Set(sessionID, req.ClientMsgID, eventSeq)
	}

	RespondJSON(w, http.StatusOK, PlayerCommandResponse{
		EventSeq: eventSeq,
	})
}

// SeekPlayer handles POST /api/sessions/:sessionId/player/seek
func (h *PlayerHandler) SeekPlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := mux.Vars(r)["sessionId"]
	userID := h.getUserIDFromSession(r)

	var req PlayerCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate positionMs provided
	if req.PositionMs == nil {
		RespondError(w, http.StatusBadRequest, "MISSING_POSITION", "positionMs is required")
		return
	}

	if *req.PositionMs < 0 {
		RespondError(w, http.StatusBadRequest, "INVALID_POSITION", "positionMs must be >= 0")
		return
	}

	// Check idempotence cache
	if req.ClientMsgID != "" {
		if eventSeq, found := h.idempotenceCache.Get(sessionID, req.ClientMsgID); found {
			RespondJSON(w, http.StatusOK, PlayerCommandResponse{
				EventSeq: eventSeq,
				Cached:   true,
			})
			return
		}
	}

	// Validate participant is synced
	if err := h.validateSyncState(sessionID, userID); err != nil {
		RespondError(w, http.StatusForbidden, "PARTICIPANT_NOT_SYNCED", err.Error())
		return
	}

	// Get Spotify user ID
	spotifyUserID, err := h.getSpotifyUserID(userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get Spotify user")
		return
	}

	// Get current player state to validate seek position against track duration
	currentState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil || currentState == nil || currentState.DurationMs <= 0 {
		// Fail fast if we can't get valid duration for validation
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Track duration unavailable for seek validation")
		return
	}

	// Validate position is within track duration (BUG #6 - AC extension)
	if *req.PositionMs > currentState.DurationMs {
		log.Printf("WARNING: Seek validation failed - positionMs (%d) exceeds track duration (%d) for track %s (indicates stale metadata)",
			*req.PositionMs, currentState.DurationMs, currentState.TrackID)

		// Return 400 with current track metadata to help client sync
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "INVALID_POSITION",
			"message": fmt.Sprintf("positionMs (%d) exceeds track duration (%d)", *req.PositionMs, currentState.DurationMs),
			"metadata": map[string]interface{}{
				"trackId":    currentState.TrackID,
				"trackName":  currentState.TrackName,
				"durationMs": currentState.DurationMs,
				"artist":     currentState.Artist,
			},
		})
		return
	}

	// Execute Spotify seek command
	if err := h.spotifyClient.Seek(ctx, spotifyUserID, *req.PositionMs); err != nil {
		h.handleSpotifyError(w, err)
		return
	}

	// Trust but Verify: Read actual player state
	playerState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil {
		log.Printf("Warning: Failed to verify player state after seek: %v", err)
		playerState = &spotify.PlayerState{PositionMs: *req.PositionMs}
	}

	// Convert to payload format
	statePayload := h.convertPlayerState(playerState)

	// Broadcast event
	eventSeq := h.hub.BroadcastPlayerSeeked(sessionID, userID, statePayload)

	// Persist event (synchronous - must succeed for data integrity AC#6)
	if err := h.persistEvent(sessionID, eventSeq, "PLAYER_SEEKED", statePayload); err != nil {
		log.Printf("ERROR: Failed to persist PLAYER_SEEKED event: %v", err)
		RespondError(w, http.StatusInternalServerError, "EVENT_PERSISTENCE_FAILED", "Failed to log event")
		return
	}

	// Cache for idempotence
	if req.ClientMsgID != "" {
		h.idempotenceCache.Set(sessionID, req.ClientMsgID, eventSeq)
	}

	RespondJSON(w, http.StatusOK, PlayerCommandResponse{
		EventSeq: eventSeq,
	})
}

// GetPlayerState handles GET /api/sessions/:sessionId/player/state
// Returns current player state without requiring any action
func (h *PlayerHandler) GetPlayerState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromSession(r)

	// Note: RequireParticipant middleware already validated that user is a participant
	// No need to re-validate sync state here - it's a read-only operation
	// The participant will only be able to execute commands (pause/resume) if synced

	// Get Spotify user ID
	spotifyUserID, err := h.getSpotifyUserID(userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get Spotify user")
		return
	}

	// Get current player state from Spotify
	playerState, err := h.spotifyClient.GetPlayerState(ctx, spotifyUserID)
	if err != nil {
		h.handleSpotifyError(w, err)
		return
	}

	// Convert to response format
	statePayload := h.convertPlayerState(playerState)

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"state": statePayload,
	})
}

// Helper methods

func (h *PlayerHandler) getUserIDFromSession(r *http.Request) string {
	sess, _ := h.sessionStore.Get(r, "boeuf-session")
	if userID, ok := sess.Values["spotify_user_id"].(string); ok {
		return userID
	}
	return ""
}

func (h *PlayerHandler) validateSyncState(sessionID, userID string) error {
	var participant models.SessionParticipant
	if err := h.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error; err != nil {
		return err
	}

	if participant.SyncState != "synced" {
		return fmt.Errorf("participant must be in synced state")
	}

	// Update last_seen_at to keep participant active (AC 1.10)
	now := time.Now()
	participant.LastSeenAt = now.Unix()
	if err := h.db.Save(&participant).Error; err != nil {
		log.Printf("WARNING: Failed to update participant last_seen_at: %v", err)
		// Don't fail - this is just a timestamp update
	}

	return nil
}

func (h *PlayerHandler) getSpotifyUserID(userID string) (string, error) {
	// In MVP, userID IS spotifyUserID (email from Spotify)
	// For future: might need to look up mapping table
	return userID, nil
}

func (h *PlayerHandler) handleSpotifyError(w http.ResponseWriter, err error) {
	if rateLimitErr, ok := err.(*spotify.RateLimitedError); ok {
		RespondError(w, http.StatusTooManyRequests, "SPOTIFY_RATE_LIMITED",
			"Spotify rate limited. Try again later.")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", rateLimitErr.RetryAfterSeconds))
		return
	}

	errMsg := err.Error()
	switch errMsg {
	case "SPOTIFY_NO_DEVICE":
		RespondError(w, http.StatusServiceUnavailable, "SPOTIFY_NO_DEVICE",
			"No active Spotify device found")
	case "SPOTIFY_NO_ACTIVE_DEVICE":
		RespondError(w, http.StatusServiceUnavailable, "SPOTIFY_NO_ACTIVE_DEVICE",
			"No active Spotify device. Please start playback in Spotify and try again.")
	case "SPOTIFY_FORBIDDEN":
		RespondError(w, http.StatusForbidden, "SPOTIFY_FORBIDDEN",
			"Spotify command forbidden")
	default:
		RespondError(w, http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE",
			"Unable to control playback")
	}
}

func (h *PlayerHandler) convertPlayerState(state *spotify.PlayerState) map[string]interface{} {
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

	return map[string]interface{}{
		"isPlaying":  state.IsPlaying,
		"positionMs": state.PositionMs,
		"track":      track,
	}
}

func (h *PlayerHandler) persistEvent(sessionID string, eventSeq int64, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal event payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	event := &models.Event{
		SessionID:   sessionID,
		EventSeq:    eventSeq,
		EventType:   eventType,
		PayloadJSON: string(payloadJSON),
	}

	// Retry with exponential backoff
	maxRetries := 3
	backoff := 50 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := h.db.Create(event).Error; err != nil {
			// Check if it's a unique constraint violation (duplicate eventSeq)
			if attempt < maxRetries-1 {
				log.Printf("WARNING: Failed to persist event (attempt %d/%d): %v", attempt+1, maxRetries, err)
				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
				continue
			}
			// Final attempt failed - return error to caller
			return fmt.Errorf("failed to persist event after %d attempts: %w", maxRetries, err)
		}
		// Success
		return nil
	}
	return nil
}
