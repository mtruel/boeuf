package repository_test

import (
	"os"
	"testing"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silence GORM logs in tests
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migration
	db.AutoMigrate(&models.SpotifyToken{})

	return db
}

func TestSaveAndGetToken(t *testing.T) {
	// ARRANGE
	db := setupTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"
	os.Setenv("ENCRYPTION_KEY", encryptionKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	token := &models.SpotifyToken{
		SpotifyUserID:         "test_user_123",
		AccessToken:           "access_token_abc",
		RefreshTokenEncrypted: "refresh_token_xyz",
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		Scope:                 "user-read-playback-state",
	}

	// ACT
	err := repo.Save(token)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// ASSERT
	retrieved, err := repo.GetBySpotifyUserID("test_user_123")
	if err != nil {
		t.Fatalf("GetBySpotifyUserID failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected non-nil token")
	}

	if retrieved.SpotifyUserID != token.SpotifyUserID {
		t.Errorf("Expected SpotifyUserID %s, got %s", token.SpotifyUserID, retrieved.SpotifyUserID)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("Expected AccessToken %s, got %s", token.AccessToken, retrieved.AccessToken)
	}
}

func TestUpdateExistingToken(t *testing.T) {
	// ARRANGE
	db := setupTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)

	token := &models.SpotifyToken{
		SpotifyUserID:         "test_user_456",
		AccessToken:           "old_access_token",
		RefreshTokenEncrypted: "old_refresh_token",
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		Scope:                 "user-read-playback-state",
	}

	repo.Save(token)

	// ACT - Update with new token
	newToken := &models.SpotifyToken{
		SpotifyUserID:         "test_user_456",
		AccessToken:           "new_access_token",
		RefreshTokenEncrypted: "new_refresh_token",
		ExpiresAt:             time.Now().Add(2 * time.Hour),
		Scope:                 "user-read-playback-state user-modify-playback-state",
	}

	err := repo.Save(newToken)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// ASSERT
	retrieved, _ := repo.GetBySpotifyUserID("test_user_456")

	if retrieved.AccessToken != "new_access_token" {
		t.Errorf("Expected updated AccessToken, got %s", retrieved.AccessToken)
	}

	// Should only have one record (updated, not duplicated)
	var count int64
	db.Model(&models.SpotifyToken{}).Where("spotify_user_id = ?", "test_user_456").Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 token record, got %d", count)
	}
}

func TestGetBySpotifyUserIDNotFound(t *testing.T) {
	// ARRANGE
	db := setupTestDB(t)
	repo := repository.NewSpotifyTokenRepository(db)

	// ACT
	retrieved, err := repo.GetBySpotifyUserID("nonexistent_user")

	// ASSERT
	if err != nil {
		t.Errorf("Expected no error for not found, got: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil token for nonexistent user")
	}
}
