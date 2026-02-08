package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/gorm"
)

type SessionHandler struct {
	store                    *sessions.CookieStore
	db                       *gorm.DB
	baseURL                  string
	sessionDuration          time.Duration
	maxActiveSessionsPerUser int
	spotifyClient            *spotify.Client
	spotifyAPIBaseURL        string
	spotifyHTTPClient        *http.Client
	realtimeHub              *realtime.Hub
}

func NewSessionHandler(store *sessions.CookieStore, db *gorm.DB, baseURL string) *SessionHandler {
	return &SessionHandler{
		store:                    store,
		db:                       db,
		baseURL:                  baseURL,
		sessionDuration:          24 * time.Hour, // Default 24 hours
		maxActiveSessionsPerUser: 0,              // 0 = unlimited (default behavior)
		spotifyClient:            nil,
		spotifyAPIBaseURL:        "https://api.spotify.com",
		spotifyHTTPClient:        &http.Client{Timeout: 10 * time.Second},
		realtimeHub:              nil,
	}
}

// NewSessionHandlerWithDuration creates a SessionHandler with custom session duration
func NewSessionHandlerWithDuration(store *sessions.CookieStore, db *gorm.DB, baseURL string, duration time.Duration) *SessionHandler {
	return &SessionHandler{
		store:                    store,
		db:                       db,
		baseURL:                  baseURL,
		sessionDuration:          duration,
		maxActiveSessionsPerUser: 0, // 0 = unlimited (default behavior)
		spotifyClient:            nil,
		spotifyAPIBaseURL:        "https://api.spotify.com",
		spotifyHTTPClient:        &http.Client{Timeout: 10 * time.Second},
		realtimeHub:              nil,
	}
}

// SetSpotifyClient sets the Spotify client for API calls
func (h *SessionHandler) SetSpotifyClient(client *spotify.Client) {
	h.spotifyClient = client
}

// SetSpotifyAPIBaseURL overrides the Spotify API base URL (useful for tests).
func (h *SessionHandler) SetSpotifyAPIBaseURL(url string) {
	h.spotifyAPIBaseURL = url
}

// SetSpotifyHTTPClient overrides the HTTP client used for Spotify API calls (useful for tests).
func (h *SessionHandler) SetSpotifyHTTPClient(client *http.Client) {
	h.spotifyHTTPClient = client
}

// SetRealtimeHub sets the realtime hub for broadcasting events
func (h *SessionHandler) SetRealtimeHub(hub *realtime.Hub) {
	h.realtimeHub = hub
}

// SetMaxActiveSessionsPerUser configures rate limiting for session creation
func (h *SessionHandler) SetMaxActiveSessionsPerUser(max int) {
	h.maxActiveSessionsPerUser = max
}

// SessionDuration defines the standard validity period for a session (deprecated constant - use handler.sessionDuration)
const SessionDuration = 24 * time.Hour

type CreateSessionResponse struct {
	SessionID  string `json:"sessionId"`            // Format: "sess_" + base64url (~22 chars after prefix)
	InviteURL  string `json:"inviteUrl"`            // Full URL to join session
	InviteCode string `json:"inviteCode,omitempty"` // Optional short code (not implemented in MVP)
	ExpiresAt  string `json:"expiresAt"`            // ISO 8601 / RFC 3339 format (e.g., "2026-01-28T21:00:00Z")
}

// generateUniqueID creates a unique identifier with an optional prefix
// Used for sessions (prefix "sess_") and invites (prefix "inv_")
func generateUniqueID(prefix string) (string, error) {
	// Generate 128 bits of random data
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate unique ID: %w", err)
	}

	encodedID := base64.URLEncoding.EncodeToString(randomBytes)
	if prefix != "" {
		return fmt.Sprintf("%s%s", prefix, encodedID), nil
	}
	return encodedID, nil
}

// generateInviteToken creates a secure opaque token for invitations
// Returns:
//   - token: Base64url-encoded random bytes (128 bits = 16 bytes → ~22 chars)
//   - tokenHash: SHA-256 hash of token, also base64url-encoded (~43 chars)
//   - error: Any error during generation
func generateInviteToken() (string, string, error) {
	// Generate 128 bits (16 bytes) of random data
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode as base64url (RFC 4648 §5) for URL safety - uses '-' and '_' instead of '+' and '/'
	// This encoding is safe to use in URLs without percent-encoding
	token := base64.URLEncoding.EncodeToString(randomBytes)

	// Hash token with SHA-256 for secure storage - only hash is persisted to database
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	return token, tokenHash, nil
}

// Create creates a new session and returns invite details
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	// Validate authentication
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		h.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}

	// Check rate limit: max active sessions per user
	if h.maxActiveSessionsPerUser > 0 {
		var activeSessionCount int64
		err := h.db.Model(&models.SessionParticipant{}).
			Joins("JOIN sessions ON sessions.id = session_participants.session_id").
			Where("session_participants.user_id = ? AND sessions.active = ? AND sessions.expires_at > ?",
				userID, true, time.Now()).
			Count(&activeSessionCount).Error

		if err != nil {
			log.Printf("ERROR: Failed to count active sessions: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check rate limit")
			return
		}

		if activeSessionCount >= int64(h.maxActiveSessionsPerUser) {
			h.sendErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				fmt.Sprintf("Maximum %d active sessions per user exceeded", h.maxActiveSessionsPerUser))
			return
		}
	}

	// Generate session ID
	sessionID, err := generateUniqueID("sess_")
	if err != nil {
		log.Printf("ERROR: Failed to generate session ID: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate session ID")
		return
	}
	now := time.Now()
	expiresAt := now.Add(h.sessionDuration)

	// Create session
	newSession := models.Session{
		ID:        sessionID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		Active:    true,
	}

	if err := h.db.Create(&newSession).Error; err != nil {
		log.Printf("ERROR: Failed to create session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
		return
	}

	// Fetch display name from spotify_tokens table
	var spotifyToken models.SpotifyToken
	if err := h.db.Where("spotify_user_id = ?", userID).First(&spotifyToken).Error; err != nil {
		log.Printf("ERROR: Failed to fetch spotify token for display name: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user info")
		return
	}

	// Add creator as participant
	participant := models.SessionParticipant{
		SessionID:   sessionID,
		UserID:      userID,
		DisplayName: spotifyToken.DisplayName,
		JoinedAt:    now,
		Role:        "host", // Creator is the host
		LastSeenAt:  now.Unix(),
	}

	if err := h.db.Create(&participant).Error; err != nil {
		log.Printf("ERROR: Failed to create participant: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add session participant")
		return
	}

	// Generate invite token
	token, tokenHash, err := generateInviteToken()
	if err != nil {
		log.Printf("ERROR: Failed to generate invite token: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate invite")
		return
	}

	// Create invite
	inviteID, err := generateUniqueID("inv_")
	if err != nil {
		log.Printf("ERROR: Failed to generate invite ID: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate invite ID")
		return
	}

	invite := models.SessionInvite{
		ID:        inviteID,
		SessionID: sessionID,
		TokenHash: tokenHash,
		Code:      "", // Optional short code not implemented in MVP
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := h.db.Create(&invite).Error; err != nil {
		log.Printf("ERROR: Failed to create invite: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create invite")
		return
	}

	// Build invite URL
	// Clean potentially trailing slash from base URL before appending path
	inviteURL := fmt.Sprintf("%s/join/%s", h.baseURL, token)

	// Prepare response
	response := CreateSessionResponse{
		SessionID:  sessionID,
		InviteURL:  inviteURL,
		InviteCode: "", // Optional in MVP
		ExpiresAt:  expiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}

// JoinSessionRequest represents the request body for joining a session
type JoinSessionRequest struct {
	InviteToken string `json:"inviteToken"`
}

// JoinSessionResponse represents the response for successfully joining a session
type JoinSessionResponse struct {
	SessionID string `json:"sessionId"`
}

// Join allows a user to join a session using an invite token
func (h *SessionHandler) Join(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	// Validate authentication
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		h.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}

	// Limit request body size to 1KB (invite tokens are small)
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	// Parse request body
	var req JoinSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.InviteToken == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "inviteToken is required")
		return
	}

	// Hash the provided token to match against stored hash
	hash := sha256.Sum256([]byte(req.InviteToken))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// Find the invite by token hash
	var invite models.SessionInvite
	err = h.db.Where("token_hash = ?", tokenHash).First(&invite).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Invalid or expired invitation")
		} else {
			log.Printf("ERROR: Failed to lookup invite: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to lookup invitation")
		}
		return
	}

	// Check if invite is expired
	if time.Now().After(invite.ExpiresAt) {
		h.sendErrorResponse(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Invalid or expired invitation")
		return
	}

	// Verify session exists and is active
	var sessionModel models.Session
	err = h.db.Where("id = ?", invite.SessionID).First(&sessionModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found")
		} else {
			log.Printf("ERROR: Failed to lookup session: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to lookup session")
		}
		return
	}

	// Check if session is expired or inactive
	if !sessionModel.Active || time.Now().After(sessionModel.ExpiresAt) {
		h.sendErrorResponse(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session is no longer active")
		return
	}

	// Check if user is already a participant (idempotent operation)
	var existingParticipant models.SessionParticipant
	err = h.db.Where("session_id = ? AND user_id = ?", invite.SessionID, userID).First(&existingParticipant).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// Fetch display name from spotify_tokens table
		var spotifyToken models.SpotifyToken
		if err := h.db.Where("spotify_user_id = ?", userID).First(&spotifyToken).Error; err != nil {
			log.Printf("ERROR: Failed to fetch spotify token for display name: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user info")
			return
		}

		// Create new participant
		participant := models.SessionParticipant{
			SessionID:   invite.SessionID,
			UserID:      userID,
			DisplayName: spotifyToken.DisplayName,
			JoinedAt:    now,
			Role:        "participant",
			LastSeenAt:  now.Unix(),
		}

		if err := h.db.Create(&participant).Error; err != nil {
			log.Printf("ERROR: Failed to create participant: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to join session")
			return
		}
	} else if err != nil {
		log.Printf("ERROR: Failed to check existing participant: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check participation status")
		return
	} else {
		// Update last_seen_at for existing participant (idempotent)
		existingParticipant.LastSeenAt = now.Unix()
		if err := h.db.Save(&existingParticipant).Error; err != nil {
			log.Printf("WARNING: Failed to update participant last_seen_at: %v", err)
			// Don't fail the request - this is just a timestamp update
		}
	}

	// Return success response
	response := JoinSessionResponse{
		SessionID: invite.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}

// sendErrorResponse sends a standardized error response
func (h *SessionHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	response := map[string]interface{}{
		"code":    code,
		"message": message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// GetParticipantMeResponse represents the response for /api/sessions/:sessionId/me
type GetParticipantMeResponse struct {
	UserID     string `json:"userId"`
	Role       string `json:"role"`
	SyncState  string `json:"syncState"`
	LastSeenAt string `json:"lastSeenAt"` // RFC3339
}

// GetParticipantMe returns information about the current participant
// GET /api/sessions/:sessionId/me
func (h *SessionHandler) GetParticipantMe(w http.ResponseWriter, r *http.Request) {
	// Validate authentication
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		h.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}

	// Extract sessionId from URL
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]
	if sessionID == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "sessionId is required")
		return
	}

	// Find participant
	var participant models.SessionParticipant
	err = h.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Not a participant of this session")
		} else {
			log.Printf("ERROR: Failed to query participant: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get participant info")
		}
		return
	}

	// Prepare response
	response := GetParticipantMeResponse{
		UserID:     participant.UserID,
		Role:       participant.Role,
		SyncState:  participant.SyncState,
		LastSeenAt: time.Unix(participant.LastSeenAt, 0).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}

// NowPlayingInfo represents current playback information
type NowPlayingInfo struct {
	TrackID    string `json:"trackId"`
	TrackName  string `json:"trackName"`
	Artist     string `json:"artist"`
	IsPlaying  bool   `json:"isPlaying"`
	PositionMs int64  `json:"positionMs"`
	DurationMs int64  `json:"durationMs"`
	ImageURL   string `json:"imageUrl,omitempty"` // Album art URL (largest available)
}

// StartSyncResponse represents the response for POST /sync/start
type StartSyncResponse struct {
	SyncState  string          `json:"syncState"`
	NowPlaying *NowPlayingInfo `json:"nowPlaying,omitempty"`
}

var errSpotifyPlayerUnavailable = errors.New("SPOTIFY_PLAYER_UNAVAILABLE")
var errSpotifyNoDevice = errors.New("SPOTIFY_NO_DEVICE")

// StartSync marks the participant as synced and initializes synchronization
// POST /api/sessions/:sessionId/sync/start
func (h *SessionHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	// Validate authentication
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		h.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}

	// Extract sessionId from URL
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]
	if sessionID == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "sessionId is required")
		return
	}

	// Find participant
	var participant models.SessionParticipant
	err = h.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Not a participant of this session")
		} else {
			log.Printf("ERROR: Failed to query participant: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get participant info")
		}
		return
	}

	// Check if already synced (idempotent)
	if participant.SyncState == "synced" {
		response := StartSyncResponse{
			SyncState: "synced",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate Spotify token
	if h.spotifyClient == nil {
		log.Printf("ERROR: Spotify client not configured")
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Spotify client not configured")
		return
	}

	ctx := r.Context()
	accessToken, err := h.spotifyClient.GetValidToken(ctx, userID)
	if err != nil {
		var rateLimited *spotify.RateLimitedError
		if errors.As(err, &rateLimited) {
			msg := "Spotify is rate limiting requests. Please retry in a moment."
			if rateLimited.RetryAfterSeconds > 0 {
				msg = fmt.Sprintf("Spotify is rate limiting requests. Please retry after %d seconds.", rateLimited.RetryAfterSeconds)
			}
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_RATE_LIMITED", msg)
			return
		}

		if err == spotify.ErrSpotifyNotConnected {
			h.sendErrorResponse(w, http.StatusConflict, "SPOTIFY_NOT_CONNECTED", "Please connect your Spotify account to start listening")
		} else {
			log.Printf("ERROR: Failed to get Spotify token: %v", err)
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE", "Failed to connect to Spotify")
		}
		return
	}

	// Pre-check: Verify at least one device is available BEFORE starting sync (AC 1)
	devices, err := h.spotifyClient.GetAvailableDevices(ctx, userID)
	if err != nil {
		var rateLimited *spotify.RateLimitedError
		if errors.As(err, &rateLimited) {
			msg := "Spotify is rate limiting requests. Please retry in a moment."
			if rateLimited.RetryAfterSeconds > 0 {
				msg = fmt.Sprintf("Spotify is rate limiting requests. Please retry after %d seconds.", rateLimited.RetryAfterSeconds)
			}
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_RATE_LIMITED", msg)
		} else {
			log.Printf("ERROR: Failed to get Spotify devices: %v", err)
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE", "Failed to check Spotify devices")
		}
		return
	}

	// AC 1: No device available → return 503 with requiresActiveDevice flag
	if len(devices) == 0 {
		log.Printf("INFO: No active Spotify device found for user %s", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":                 "SPOTIFY_NO_DEVICE",
			"message":              "No active Spotify device found. Please start playback in Spotify.",
			"requiresActiveDevice": true,
		})
		return
	}

	// Log device info for debugging
	for _, device := range devices {
		log.Printf("DEBUG: Device found - Name: %s, Type: %s, Active: %v", device.Name, device.Type, device.IsActive)
	}

	// Get current playback state from Spotify
	nowPlaying, err := h.getCurrentPlayback(ctx, accessToken)
	if err != nil {
		var rateLimited *spotify.RateLimitedError
		switch {
		case errors.As(err, &rateLimited):
			msg := "Spotify is rate limiting requests. Please retry in a moment."
			if rateLimited.RetryAfterSeconds > 0 {
				msg = fmt.Sprintf("Spotify is rate limiting requests. Please retry after %d seconds.", rateLimited.RetryAfterSeconds)
			}
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_RATE_LIMITED", msg)
		case errors.Is(err, spotify.ErrSpotifyNotConnected):
			h.sendErrorResponse(w, http.StatusConflict, "SPOTIFY_NOT_CONNECTED", "Please connect your Spotify account to start listening")
		case errors.Is(err, errSpotifyPlayerUnavailable):
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_PLAYER_UNAVAILABLE", "No active Spotify device found. Start Spotify on any device and retry.")
		default:
			log.Printf("ERROR: Failed to get Spotify playback: %v", err)
			h.sendErrorResponse(w, http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE", "Failed to fetch Spotify playback state")
		}
		return
	}

	// Persist baseline playback state in session (for newcomers).
	// Best-effort: if session row is missing, return a stable internal error.
	var sess models.Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		log.Printf("ERROR: Failed to load session for baseline update: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to persist sync baseline")
		return
	}
	baselineCapturedAt := time.Now().UTC()
	sess.BaselineTrackID = nowPlaying.TrackID
	sess.BaselineTrackName = nowPlaying.TrackName
	sess.BaselineArtist = nowPlaying.Artist
	sess.BaselineIsPlaying = nowPlaying.IsPlaying
	sess.BaselinePositionMs = nowPlaying.PositionMs
	sess.BaselineDurationMs = nowPlaying.DurationMs
	sess.BaselineImageURL = nowPlaying.ImageURL
	sess.BaselineCapturedAt = &baselineCapturedAt
	if err := h.db.Save(&sess).Error; err != nil {
		log.Printf("ERROR: Failed to save session baseline: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to persist sync baseline")
		return
	}

	// Update participant sync state
	participant.SyncState = "synced"
	participant.LastSeenAt = time.Now().Unix()
	if err := h.db.Save(&participant).Error; err != nil {
		log.Printf("ERROR: Failed to update participant sync state: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update sync state")
		return
	}

	// Broadcast sync state change to all participants
	if h.realtimeHub != nil {
		payload := realtime.ParticipantSyncStateChangedPayload{
			UserID:    userID,
			SyncState: "synced",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		h.realtimeHub.BroadcastToSession(sessionID, realtime.TypeParticipantSyncStateChanged, payload)
	}

	// Return response
	response := StartSyncResponse{
		SyncState:  "synced",
		NowPlaying: nowPlaying,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}

// GetSession returns information about a session
// GET /api/sessions/:sessionId
type GetSessionResponse struct {
	SessionID  string          `json:"sessionId"`
	CreatedAt  string          `json:"createdAt"`
	ExpiresAt  string          `json:"expiresAt"`
	Active     bool            `json:"active"`
	NowPlaying *NowPlayingInfo `json:"nowPlaying,omitempty"`
}

func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	// Validate authentication
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		h.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}

	// Extract sessionId from URL
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]
	if sessionID == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "sessionId is required")
		return
	}

	// Verify user is a participant
	var participant models.SessionParticipant
	err = h.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Not a participant of this session")
		} else {
			log.Printf("ERROR: Failed to query participant: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify participation")
		}
		return
	}

	// Find session
	var sess models.Session
	err = h.db.Where("id = ?", sessionID).First(&sess).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sendErrorResponse(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found")
		} else {
			log.Printf("ERROR: Failed to query session: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get session info")
		}
		return
	}

	// Prepare response
	response := GetSessionResponse{
		SessionID: sess.ID,
		CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		ExpiresAt: sess.ExpiresAt.Format(time.RFC3339),
		Active:    sess.Active,
	}

	// Include baseline NowPlaying if available
	if sess.BaselineTrackID != "" && sess.BaselineCapturedAt != nil {
		response.NowPlaying = &NowPlayingInfo{
			TrackID:    sess.BaselineTrackID,
			TrackName:  sess.BaselineTrackName,
			Artist:     sess.BaselineArtist,
			IsPlaying:  sess.BaselineIsPlaying,
			PositionMs: sess.BaselinePositionMs,
			DurationMs: sess.BaselineDurationMs,
			ImageURL:   sess.BaselineImageURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}

// getCurrentPlayback fetches the current playback state from Spotify
func (h *SessionHandler) getCurrentPlayback(ctx context.Context, accessToken string) (*NowPlayingInfo, error) {
	baseURL := strings.TrimRight(h.spotifyAPIBaseURL, "/")
	url := baseURL + "/v1/me/player"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := h.spotifyHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content = no active playback
	if resp.StatusCode == http.StatusNoContent {
		return nil, errSpotifyPlayerUnavailable
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, spotify.ErrSpotifyNotConnected
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 1
		if h := resp.Header.Get("Retry-After"); h != "" {
			if n, _ := strconv.Atoi(h); n > 0 {
				retryAfter = n
			}
		}
		return nil, &spotify.RateLimitedError{RetryAfterSeconds: retryAfter}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errSpotifyPlayerUnavailable
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify api returned status %d", resp.StatusCode)
	}

	var playbackState struct {
		Item struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			DurationMs int64  `json:"duration_ms"`
			Artists    []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Images []struct {
					URL    string `json:"url"`
					Height int    `json:"height"`
					Width  int    `json:"width"`
				} `json:"images"`
			} `json:"album"`
		} `json:"item"`
		IsPlaying  bool  `json:"is_playing"`
		ProgressMs int64 `json:"progress_ms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&playbackState); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if playbackState.Item.ID == "" {
		return nil, errSpotifyPlayerUnavailable
	}

	// Extract artist names
	artistNames := make([]string, 0, len(playbackState.Item.Artists))
	for _, artist := range playbackState.Item.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	// Extract largest album image (first in list is usually largest)
	imageURL := ""
	if len(playbackState.Item.Album.Images) > 0 {
		imageURL = playbackState.Item.Album.Images[0].URL
	}

	return &NowPlayingInfo{
		TrackID:    "spotify:track:" + playbackState.Item.ID,
		TrackName:  playbackState.Item.Name,
		Artist:     strings.Join(artistNames, ", "),
		IsPlaying:  playbackState.IsPlaying,
		PositionMs: playbackState.ProgressMs,
		DurationMs: playbackState.Item.DurationMs,
		ImageURL:   imageURL,
	}, nil
}
