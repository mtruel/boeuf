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

func setupJoinTestDB(t *testing.T) (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{}, &models.SpotifyToken{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Insert test Spotify tokens for common test users
	testTokens := []models.SpotifyToken{
		{
			SpotifyUserID:         "test-user-123",
			DisplayName:           "Test User",
			AccessToken:           "test-access-token",
			RefreshTokenEncrypted: "test-refresh-token",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
	}
	for _, token := range testTokens {
		db.Create(&token)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

func TestJoinSession_ValidInvite(t *testing.T) {
	db, cleanup := setupJoinTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))
	handler := NewSessionHandler(store, db, "http://localhost:3000")

	// Create a test session
	sessionID := "sess_test123"
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	testSession := models.Session{
		ID:        sessionID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		Active:    true,
	}
	db.Create(&testSession)

	// Create an invite
	token, tokenHash, _ := generateInviteToken()
	invite := models.SessionInvite{
		ID:        "inv_test123",
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	db.Create(&invite)

	// Create request payload
	payload := map[string]string{"inviteToken": token}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add authenticated session
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test-user-123"
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	
	// Save session to cookies
	session.Save(req, w)
	
	// Copy cookies to the request
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	
	// Reset the recorder
	w = httptest.NewRecorder()

	// Call the handler
	handler.Join(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["sessionId"] != sessionID {
		t.Errorf("Expected sessionId %s, got %v", sessionID, response["sessionId"])
	}

	// Verify participant was created in database
	var participant models.SessionParticipant
	err := db.Where("session_id = ? AND user_id = ?", sessionID, "test-user-123").First(&participant).Error
	if err != nil {
		t.Errorf("Expected participant to be created, got error: %v", err)
	}

	if participant.Role != "participant" {
		t.Errorf("Expected role 'participant', got %s", participant.Role)
	}
}

func TestJoinSession_InvalidToken(t *testing.T) {
	db, cleanup := setupJoinTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))
	handler := NewSessionHandler(store, db, "http://localhost:3000")

	// Create request with invalid token
	payload := map[string]string{"inviteToken": "invalid-token-xyz"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add authenticated session
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test-user-123"
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the handler
	handler.Join(w, req)

	// Verify error response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "SESSION_NOT_FOUND" {
		t.Errorf("Expected error code SESSION_NOT_FOUND, got %v", response["code"])
	}
}

func TestJoinSession_ExpiredInvite(t *testing.T) {
	db, cleanup := setupJoinTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))
	handler := NewSessionHandler(store, db, "http://localhost:3000")

	// Create a test session
	sessionID := "sess_expired"
	now := time.Now()

	testSession := models.Session{
		ID:        sessionID,
		CreatedAt: now.Add(-48 * time.Hour),
		ExpiresAt: now.Add(-24 * time.Hour), // Expired
		Active:    true,
	}
	db.Create(&testSession)

	// Create an expired invite
	token, tokenHash, _ := generateInviteToken()
	invite := models.SessionInvite{
		ID:        "inv_expired",
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(-24 * time.Hour), // Expired
		CreatedAt: now.Add(-48 * time.Hour),
	}
	db.Create(&invite)

	// Create request
	payload := map[string]string{"inviteToken": token}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add authenticated session
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "test-user-123"
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the handler
	handler.Join(w, req)

	// Verify error response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "SESSION_NOT_FOUND" {
		t.Errorf("Expected error code SESSION_NOT_FOUND, got %v", response["code"])
	}
}

func TestJoinSession_Unauthenticated(t *testing.T) {
	db, cleanup := setupJoinTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))
	handler := NewSessionHandler(store, db, "http://localhost:3000")

	// Create request without authentication
	payload := map[string]string{"inviteToken": "some-token"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Call the handler without setting up authentication
	handler.Join(w, req)

	// Verify error response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "UNAUTHENTICATED" {
		t.Errorf("Expected error code UNAUTHENTICATED, got %v", response["code"])
	}
}

func TestJoinSession_AlreadyParticipant(t *testing.T) {
	db, cleanup := setupJoinTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))
	handler := NewSessionHandler(store, db, "http://localhost:3000")

	// Create a test session
	sessionID := "sess_duplicate"
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	userID := "test-user-duplicate"

	testSession := models.Session{
		ID:        sessionID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		Active:    true,
	}
	db.Create(&testSession)

	// Create an invite
	token, tokenHash, _ := generateInviteToken()
	invite := models.SessionInvite{
		ID:        "inv_duplicate",
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	db.Create(&invite)

	// Pre-create participant
	existingParticipant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		JoinedAt:   now.Add(-1 * time.Hour),
		Role:       "participant",
		LastSeenAt: now.Add(-1 * time.Hour).Unix(),
	}
	db.Create(&existingParticipant)

	// Create request
	payload := map[string]string{"inviteToken": token}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add authenticated session
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the handler
	handler.Join(w, req)

	// Verify response - should succeed (idempotent operation)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["sessionId"] != sessionID {
		t.Errorf("Expected sessionId %s, got %v", sessionID, response["sessionId"])
	}

	// Verify LastSeenAt was updated
	var updatedParticipant models.SessionParticipant
	db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&updatedParticipant)

	if updatedParticipant.LastSeenAt < existingParticipant.LastSeenAt {
		t.Errorf("Expected LastSeenAt to be updated")
	}
}
