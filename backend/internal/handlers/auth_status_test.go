package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mathias/boeuf/internal/handlers"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/session"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForAuthStatus(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	db.AutoMigrate(&models.SpotifyToken{})
	return db
}

func TestAuthStatusNotAuthenticated(t *testing.T) {
	// ARRANGE
	db := setupTestDBForAuthStatus(t)
	repo := repository.NewSpotifyTokenRepository(db)
	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)

	handler := handlers.NewAuthStatusHandler(store, repo)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()

	// ACT
	handler.Status(w, req)

	// ASSERT
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["authenticated"] != false {
		t.Error("Expected authenticated=false for unauthenticated user")
	}
}

func TestAuthStatusAuthenticated(t *testing.T) {
	// ARRANGE
	db := setupTestDBForAuthStatus(t)
	repo := repository.NewSpotifyTokenRepository(db)

	// Save a token in DB
	token := &models.SpotifyToken{
		SpotifyUserID:         "user_123",
		AccessToken:           "access_token",
		RefreshTokenEncrypted: "encrypted_refresh",
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		Scope:                 "user-read-playback-state",
	}
	repo.Save(token)

	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)

	handler := handlers.NewAuthStatusHandler(store, repo)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()

	// Set spotify_user_id in session
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = "user_123"
	sess.Save(req, w)

	// Use cookies from response in new request
	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest("GET", "/api/auth/status", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()

	// ACT
	handler.Status(w2, req2)

	// ASSERT
	resp := w2.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["authenticated"] != true {
		t.Error("Expected authenticated=true for authenticated user")
	}

	if result["spotify_user_id"] != "user_123" {
		t.Errorf("Expected spotify_user_id=user_123, got %v", result["spotify_user_id"])
	}
}
