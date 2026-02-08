package spotify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupClientTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	db.AutoMigrate(&models.SpotifyToken{})
	return db
}

func TestClientAutoRefresh(t *testing.T) {
	// ARRANGE
	db := setupClientTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"

	// Mock Spotify refresh endpoint
	refreshCallCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token" {
			refreshCallCount++
			r.ParseForm()
			if r.FormValue("grant_type") != "refresh_token" {
				t.Error("Expected grant_type=refresh_token")
			}

			// Return new access token
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "new_access_token",
				"expires_in":   3600,
				"scope":        "user-read-playback-state",
				"token_type":   "Bearer",
			})
		}
	}))
	defer mockServer.Close()

	// Prepare expired token in DB
	encryptedRefresh, _ := spotify.EncryptForTest("refresh_token_123", encryptionKey)
	expiredToken := &models.SpotifyToken{
		SpotifyUserID:         "user_456",
		AccessToken:           "expired_access_token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(-10 * time.Minute), // Expired
		Scope:                 "user-read-playback-state",
	}
	repo.Save(expiredToken)

	client := spotify.NewClient(repo, encryptionKey, "client_id")
	client.SetTokenURL(mockServer.URL + "/api/token")

	// ACT - Get token (should trigger refresh)
	token, err := client.GetValidToken(context.Background(), "user_456")

	// ASSERT
	if err != nil {
		t.Fatalf("GetValidToken failed: %v", err)
	}

	if token != "new_access_token" {
		t.Errorf("Expected new_access_token, got %s", token)
	}

	if refreshCallCount != 1 {
		t.Errorf("Expected 1 refresh call, got %d", refreshCallCount)
	}

	// Verify DB was updated
	updated, _ := repo.GetBySpotifyUserID("user_456")
	if updated.AccessToken != "new_access_token" {
		t.Error("Expected DB to be updated with new access token")
	}
}

func TestClientNoRefreshWhenValid(t *testing.T) {
	// ARRANGE
	db := setupClientTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"

	// Mock server (should not be called)
	refreshCallCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCallCount++
	}))
	defer mockServer.Close()

	// Prepare valid token in DB
	encryptedRefresh, _ := spotify.EncryptForTest("refresh_token_789", encryptionKey)
	validToken := &models.SpotifyToken{
		SpotifyUserID:         "user_789",
		AccessToken:           "valid_access_token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(30 * time.Minute), // Still valid
		Scope:                 "user-read-playback-state",
	}
	repo.Save(validToken)

	client := spotify.NewClient(repo, encryptionKey, "client_id")
	client.SetTokenURL(mockServer.URL + "/api/token")

	// ACT
	token, err := client.GetValidToken(context.Background(), "user_789")

	// ASSERT
	if err != nil {
		t.Fatalf("GetValidToken failed: %v", err)
	}

	if token != "valid_access_token" {
		t.Errorf("Expected valid_access_token, got %s", token)
	}

	if refreshCallCount != 0 {
		t.Errorf("Expected no refresh calls, got %d", refreshCallCount)
	}
}

func TestClientRefreshFailure(t *testing.T) {
	// ARRANGE
	db := setupClientTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"

	// Mock server returns error (e.g., revoked token)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "invalid_grant",
			"error_description": "Refresh token has been revoked",
		})
	}))
	defer mockServer.Close()

	// Prepare expired token
	encryptedRefresh, _ := spotify.EncryptForTest("revoked_refresh_token", encryptionKey)
	expiredToken := &models.SpotifyToken{
		SpotifyUserID:         "user_revoked",
		AccessToken:           "expired_access_token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(-10 * time.Minute),
		Scope:                 "user-read-playback-state",
	}
	repo.Save(expiredToken)

	client := spotify.NewClient(repo, encryptionKey, "client_id")
	client.SetTokenURL(mockServer.URL + "/api/token")

	// ACT
	_, err := client.GetValidToken(context.Background(), "user_revoked")

	// ASSERT
	if err == nil {
		t.Error("Expected error when refresh fails")
	}

	// Should return SPOTIFY_NOT_CONNECTED error as per AC3
	if err.Error() != "SPOTIFY_NOT_CONNECTED" {
		t.Errorf("Expected SPOTIFY_NOT_CONNECTED error, got: %v", err)
	}
}
