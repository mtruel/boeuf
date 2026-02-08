package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/session"
)

type AuthStatusHandler struct {
	store session.Store
	repo  *repository.SpotifyTokenRepository
}

func NewAuthStatusHandler(store session.Store, repo *repository.SpotifyTokenRepository) *AuthStatusHandler {
	return &AuthStatusHandler{
		store: store,
		repo:  repo,
	}
}

// Status returns the authentication status for the current user
func (h *AuthStatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess, err := h.store.Get(r, session.SessionName)
	if err != nil {
		// Session error, return not authenticated
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Check if user has spotify_user_id in session
	spotifyUserID, ok := sess.Values["spotify_user_id"].(string)
	if !ok || spotifyUserID == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Verify token exists in DB
	token, err := h.repo.GetBySpotifyUserID(spotifyUserID)
	if err != nil || token == nil {
		// No token found, not authenticated
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	if token.ExpiresAt.IsZero() || token.ExpiresAt.Before(time.Now()) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// User is authenticated
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"spotifyUserId": spotifyUserID,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
