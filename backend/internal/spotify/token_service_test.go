package spotify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	db.AutoMigrate(&models.SpotifyToken{})
	return db
}

func TestExchangeCodeForTokens(t *testing.T) {
	// ARRANGE
	db := setupTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)

	encryptionKey := "12345678901234567890123456789012"
	os.Setenv("ENCRYPTION_KEY", encryptionKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	// Mock Spotify token endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify PKCE parameters
		r.ParseForm()
		if r.FormValue("grant_type") != "authorization_code" {
			t.Error("Expected grant_type=authorization_code")
		}
		if r.FormValue("code_verifier") == "" {
			t.Error("Expected code_verifier")
		}

		// Return mock tokens
		response := map[string]interface{}{
			"access_token":  "mock_access_token",
			"refresh_token": "mock_refresh_token",
			"expires_in":    3600,
			"scope":         "user-read-playback-state user-modify-playback-state",
			"token_type":    "Bearer",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Mock user info endpoint
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "spotify_user_123",
		})
	}))
	defer userInfoServer.Close()

	service := spotify.NewTokenService(repo, encryptionKey)
	service.SetTokenURL(mockServer.URL + "/api/token")
	service.SetUserInfoURL(userInfoServer.URL)

	// ACT
	result, err := service.ExchangeCodeForTokens(context.Background(), "test_code", "test_verifier", "client_id", "redirect_uri")

	// ASSERT
	if err != nil {
		t.Fatalf("ExchangeCodeForTokens failed: %v", err)
	}

	if result.SpotifyUserID != "spotify_user_123" {
		t.Errorf("Expected spotify_user_123, got %s", result.SpotifyUserID)
	}

	if result.AccessToken != "mock_access_token" {
		t.Errorf("Expected mock_access_token, got %s", result.AccessToken)
	}

	// Verify token was saved to DB
	saved, err := repo.GetBySpotifyUserID("spotify_user_123")
	if err != nil {
		t.Fatalf("Failed to retrieve saved token: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected token to be saved in DB")
	}

	// Refresh token should be encrypted in DB
	if saved.RefreshTokenEncrypted == "mock_refresh_token" {
		t.Error("Refresh token should be encrypted in DB, not plaintext")
	}
}

func TestExchangeCodeForTokensSpotifyError(t *testing.T) {
	// ARRANGE
	db := setupTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"

	// Mock Spotify error response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "invalid_grant",
			"error_description": "Invalid authorization code",
		})
	}))
	defer mockServer.Close()

	service := spotify.NewTokenService(repo, encryptionKey)
	service.SetTokenURL(mockServer.URL)

	// ACT
	_, err := service.ExchangeCodeForTokens(context.Background(), "bad_code", "verifier", "client_id", "redirect_uri")

	// ASSERT
	if err == nil {
		t.Error("Expected error from Spotify API")
	}

	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("Expected error to mention 'invalid_grant', got: %v", err)
	}
}
