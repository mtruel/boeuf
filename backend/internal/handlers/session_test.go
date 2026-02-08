package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testBaseURL = "http://test-server"

func setupSessionTest() (*gorm.DB, *SessionHandler, *sessions.CookieStore) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})

	store := sessions.NewCookieStore([]byte("test-session-key-1234567890123456"))
	handler := NewSessionHandler(store, db, testBaseURL)

	return db, handler, store
}

func TestCreateSession_Success(t *testing.T) {
	db, handler, store := setupSessionTest()

	// Create a request with authenticated session
	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	// Mock authentication in session
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test_user_123"
	session.Save(req, w)

	handler.Create(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	if response.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
	if response.InviteURL == "" {
		t.Error("InviteURL should not be empty")
	}
	if response.ExpiresAt == "" {
		t.Error("ExpiresAt should not be empty")
	}

	// Verify session was created in DB
	var sessionCount int64
	db.Model(&models.Session{}).Count(&sessionCount)
	if sessionCount != 1 {
		t.Errorf("Expected 1 session in DB, got %d", sessionCount)
	}

	// Verify participant was created
	var participantCount int64
	db.Model(&models.SessionParticipant{}).Count(&participantCount)
	if participantCount != 1 {
		t.Errorf("Expected 1 participant in DB, got %d", participantCount)
	}

	// Verify invite was created
	var inviteCount int64
	db.Model(&models.SessionInvite{}).Count(&inviteCount)
	if inviteCount != 1 {
		t.Errorf("Expected 1 invite in DB, got %d", inviteCount)
	}
}

func TestCreateSession_Unauthenticated(t *testing.T) {
	_, handler, _ := setupSessionTest()

	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCreateSession_InviteExpiration(t *testing.T) {
	db, handler, store := setupSessionTest()

	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test_user_123"
	session.Save(req, w)

	handler.Create(w, req)

	var response CreateSessionResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	// Parse expiration time
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Errorf("Failed to parse expiresAt: %v", err)
	}

	// Verify expiration is within 24 hours
	now := time.Now()
	maxExpiry := now.Add(SessionDuration)
	minExpiry := now.Add(SessionDuration - time.Minute) // Allow some margin

	if expiresAt.After(maxExpiry) || expiresAt.Before(minExpiry) {
		t.Errorf("Expiration time %v should be within 24 hours from now", expiresAt)
	}

	// Verify invite in DB has correct expiration
	var invite models.SessionInvite
	db.First(&invite)
	if invite.ExpiresAt.Before(minExpiry) || invite.ExpiresAt.After(maxExpiry) {
		t.Errorf("DB invite expiration %v should be within 24 hours", invite.ExpiresAt)
	}
}

func TestCreateSession_NoSpotifyTokensInResponse(t *testing.T) {
	_, handler, store := setupSessionTest()

	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test_user_123"
	session.Save(req, w)

	handler.Create(w, req)

	responseBody := w.Body.String()

	// Ensure no sensitive Spotify tokens are exposed
	if bytes.Contains([]byte(responseBody), []byte("access_token")) ||
		bytes.Contains([]byte(responseBody), []byte("refresh_token")) {
		t.Error("Response should not contain Spotify tokens")
	}
}

func TestCreateSession_CustomSessionDuration(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})
	store := sessions.NewCookieStore([]byte("test-session-key-1234567890123456"))
	
	// Create handler with custom 2-hour session duration
	customDuration := 2 * time.Hour
	handler := NewSessionHandlerWithDuration(store, db, testBaseURL, customDuration)

	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test_user_123"
	session.Save(req, w)

	handler.Create(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response CreateSessionResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	// Parse expiration time
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Fatalf("Failed to parse expiresAt: %v", err)
	}

	// Verify expiration matches custom duration (2 hours)
	now := time.Now()
	expectedExpiry := now.Add(customDuration)
	margin := time.Minute

	if expiresAt.Before(expectedExpiry.Add(-margin)) || expiresAt.After(expectedExpiry.Add(margin)) {
		t.Errorf("Expiration time %v should be within 2 hours from now (expected ~%v)", expiresAt, expectedExpiry)
	}
}

func TestCreateSession_RateLimitExceeded(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})
	store := sessions.NewCookieStore([]byte("test-session-key-1234567890123456"))

	// Create handler with max 2 active sessions per user
	handler := NewSessionHandlerWithDuration(store, db, testBaseURL, 24*time.Hour)
	handler.SetMaxActiveSessionsPerUser(2)

	// Create 2 sessions (should succeed)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
		w := httptest.NewRecorder()

		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for session %d, got %d", i+1, w.Code)
		}
	}

	// Try to create a 3rd session (should fail with 429)
	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()

	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test_user_123"
	session.Save(req, w)

	handler.Create(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"] != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("Expected error code RATE_LIMIT_EXCEEDED, got %v", response["code"])
	}
}

func TestCreateSession_MethodNotAllowed(t *testing.T) {
	_, handler, store := setupSessionTest()

	tests := []struct {
		method string
	}{
		{"GET"},
		{"PUT"},
		{"DELETE"},
		{"PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/sessions", nil)
			w := httptest.NewRecorder()

			// Mock authentication
			session, _ := store.Get(req, "boeuf-session")
			session.Values["spotify_user_id"] = "test_user_123"
			session.Save(req, w)

			handler.Create(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Method %s: Expected status 405, got %d", tt.method, w.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if response["code"] != "METHOD_NOT_ALLOWED" {
				t.Errorf("Method %s: Expected error code METHOD_NOT_ALLOWED, got %v", tt.method, response["code"])
			}
		})
	}
}

func TestCreateSession_MultipleSessionsSameUser(t *testing.T) {
	db, handler, store := setupSessionTest()

	// Create 3 sessions for the same user (no rate limit by default)
	createdSessionIDs := make([]string, 3)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer([]byte("{}")))
		w := httptest.NewRecorder()

		session, _ := store.Get(req, "boeuf-session")
		session.Values["spotify_user_id"] = "test_user_123"
		session.Save(req, w)

		handler.Create(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Session %d: Expected status 200, got %d", i+1, w.Code)
		}

		var response CreateSessionResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		createdSessionIDs[i] = response.SessionID

		// Verify each session has unique ID
		for j := 0; j < i; j++ {
			if createdSessionIDs[i] == createdSessionIDs[j] {
				t.Errorf("Sessions %d and %d have duplicate IDs: %s", i+1, j+1, createdSessionIDs[i])
			}
		}
	}

	// Verify all 3 sessions exist in database
	var sessionCount int64
	db.Model(&models.Session{}).Where("active = ?", true).Count(&sessionCount)
	if sessionCount != 3 {
		t.Errorf("Expected 3 active sessions in DB, got %d", sessionCount)
	}

	// Verify all sessions belong to same user
	var participantCount int64
	db.Model(&models.SessionParticipant{}).Where("user_id = ?", "test_user_123").Count(&participantCount)
	if participantCount != 3 {
		t.Errorf("Expected 3 participants for user, got %d", participantCount)
	}
}
