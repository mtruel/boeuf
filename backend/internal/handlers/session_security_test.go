package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mathias/boeuf/internal/models"
)

func TestSecurityConventions(t *testing.T) {
	db, handler, store := setupSessionTest()

	t.Run("should require authentication", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for unauthenticated request, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["code"] != "UNAUTHENTICATED" {
			t.Errorf("Expected UNAUTHENTICATED error code, got %v", response["code"])
		}
	})

	t.Run("should use standardized error format", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// Check error format has required fields
		if _, ok := response["code"]; !ok {
			t.Error("Error response should have 'code' field")
		}
		if _, ok := response["message"]; !ok {
			t.Error("Error response should have 'message' field")
		}

		// Check Content-Type
		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Error response should have application/json content type")
		}
	})

	t.Run("should not expose spotify tokens in response", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		// Mock authentication
		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		responseBody := w.Body.String()

		// Ensure no sensitive Spotify tokens are exposed
		sensitiveFields := []string{
			"access_token",
			"refresh_token",
			"spotify_access_token",
			"spotify_refresh_token",
			"bearer",
		}

		for _, field := range sensitiveFields {
			if strings.Contains(strings.ToLower(responseBody), field) {
				t.Errorf("Response should not contain sensitive field '%s'", field)
			}
		}
	})

	t.Run("should use secure token storage (hashed)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		// Mock authentication
		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		// Check that token is hashed in DB (not stored as plaintext)
		var invite models.SessionInvite
		db.First(&invite)

		// TokenHash should be different from the token sent in URL
		var response CreateSessionResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		// Extract token from URL (after the last slash)
		urlParts := strings.Split(response.InviteURL, "/")
		token := urlParts[len(urlParts)-1]

		// Hash should NOT equal the plaintext token
		if invite.TokenHash == token {
			t.Error("Token should be hashed in database, not stored as plaintext")
		}

		// Hash should be base64-encoded (typical hash format)
		if len(invite.TokenHash) == 0 {
			t.Error("Token hash should not be empty")
		}
	})

	t.Run("should enforce session-based authorization", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		// Mock authentication with session cookie
		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for authenticated user, got %d", w.Code)
		}

		// Verify user can only create sessions for themselves
		// (implicit in the fact that we use session-stored spotify_user_id)
		var participant models.SessionParticipant
		db.First(&participant)

		if participant.UserID != "test_user_123" {
			t.Errorf("Expected participant user_id to match session user, got %s", participant.UserID)
		}
	})

	t.Run("should use camelCase JSON response format", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		// Mock authentication
		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// Check camelCase fields exist
		expectedFields := []string{"sessionId", "inviteUrl", "expiresAt"}
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Response should have camelCase field '%s'", field)
			}
		}

		// Check no snake_case equivalents
		forbiddenFields := []string{"session_id", "invite_url", "expires_at"}
		for _, field := range forbiddenFields {
			if _, ok := response[field]; ok {
				t.Errorf("Response should not have snake_case field '%s' (use camelCase)", field)
			}
		}
	})
}
