package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"gorm.io/gorm"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const sessionIDKey contextKey = "sessionID"

// AccessControlMiddleware provides session-based access control
type AccessControlMiddleware struct {
	store *sessions.CookieStore
	db    *gorm.DB
}

// NewAccessControlMiddleware creates a new access control middleware
func NewAccessControlMiddleware(store *sessions.CookieStore, db *gorm.DB) *AccessControlMiddleware {
	return &AccessControlMiddleware{
		store: store,
		db:    db,
	}
}

// RequireParticipant is a middleware that ensures the authenticated user is a participant of the specified session
// It extracts sessionID from the URL path variable "{sessionId}" using gorilla/mux
// Returns UNAUTHENTICATED if user is not logged in
// Returns FORBIDDEN if user is not a participant of the session
func (m *AccessControlMiddleware) RequireParticipant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract sessionID from URL path variables (gorilla/mux)
		vars := mux.Vars(r)
		sessionID := vars["sessionId"]

		if sessionID == "" {
			log.Printf("ERROR: No sessionId in URL path")
			m.sendErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "Session ID required")
			return
		}

		// Check authentication
		session, err := m.store.Get(r, "boeuf-session")
		if err != nil {
			log.Printf("ERROR: Failed to get session: %v", err)
			m.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to access session")
			return
		}

		userID, ok := session.Values["spotify_user_id"].(string)
		if !ok || userID == "" {
			m.sendErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
			return
		}

		// Check if user is a participant of the session
		var participant models.SessionParticipant
		err = m.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// User is not a participant - return generic FORBIDDEN without leaking session existence
				m.sendErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			} else {
				log.Printf("ERROR: Failed to check participant: %v", err)
				m.sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify access")
			}
			return
		}

		// Store sessionID in context for downstream handlers
		ctx := context.WithValue(r.Context(), sessionIDKey, sessionID)

		// Access granted - call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SessionIDFromContext extracts the session ID from the request context
func SessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey).(string)
	return sessionID, ok
}

// sendErrorResponse sends a standardized error response
func (m *AccessControlMiddleware) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	response := map[string]interface{}{
		"code":    code,
		"message": message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
