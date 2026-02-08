package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"gorm.io/gorm"
)

type SessionHandler struct {
	store                    *sessions.CookieStore
	db                       *gorm.DB
	baseURL                  string
	sessionDuration          time.Duration
	maxActiveSessionsPerUser int
}

func NewSessionHandler(store *sessions.CookieStore, db *gorm.DB, baseURL string) *SessionHandler {
	return &SessionHandler{
		store:                    store,
		db:                       db,
		baseURL:                  baseURL,
		sessionDuration:          24 * time.Hour, // Default 24 hours
		maxActiveSessionsPerUser: 0,              // 0 = unlimited (default behavior)
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
	}
}

// SetMaxActiveSessionsPerUser configures rate limiting for session creation
func (h *SessionHandler) SetMaxActiveSessionsPerUser(max int) {
	h.maxActiveSessionsPerUser = max
}

// SessionDuration defines the standard validity period for a session (deprecated constant - use handler.sessionDuration)
const SessionDuration = 24 * time.Hour

type CreateSessionResponse struct {
	SessionID  string `json:"sessionId"`  // Format: "sess_" + base64url (~22 chars after prefix)
	InviteURL  string `json:"inviteUrl"`  // Full URL to join session
	InviteCode string `json:"inviteCode,omitempty"` // Optional short code (not implemented in MVP)
	ExpiresAt  string `json:"expiresAt"`  // ISO 8601 / RFC 3339 format (e.g., "2026-01-28T21:00:00Z")
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

	// Add creator as participant
	participant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		JoinedAt:   now,
		Role:       "host", // Creator is the host
		LastSeenAt: now,
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
