package handlers

import (
	"testing"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCleanupExpiredSessions(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})

	now := time.Now()
	cleaner := NewSessionCleaner(db)

	// Create an active non-expired session
	activeSession := models.Session{
		ID:        "active-session",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&activeSession)

	// Create an expired but still active session
	expiredSession := models.Session{
		ID:        "expired-session",
		CreatedAt: now.Add(-48 * time.Hour),
		ExpiresAt: now.Add(-24 * time.Hour),
		Active:    true,
	}
	db.Create(&expiredSession)

	// Create an already inactive session (should be deleted after 7 days grace period)
	inactiveSession := models.Session{
		ID:        "inactive-session",
		CreatedAt: now.Add(-10 * 24 * time.Hour),
		ExpiresAt: now.Add(-8 * 24 * time.Hour), // Expired more than 7 days ago
		Active:    false,
	}
	db.Create(&inactiveSession)

	// Run cleanup
	deleted, err := cleaner.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Should have deactivated the expired session
	var updatedExpiredSession models.Session
	db.First(&updatedExpiredSession, "id = ?", "expired-session")
	if updatedExpiredSession.Active {
		t.Error("Expired session should have been deactivated")
	}

	// Active session should still be active
	var stillActiveSession models.Session
	db.First(&stillActiveSession, "id = ?", "active-session")
	if !stillActiveSession.Active {
		t.Error("Active non-expired session should remain active")
	}

	// Inactive session should be deleted
	var inactiveCount int64
	db.Model(&models.Session{}).Where("id = ?", "inactive-session").Count(&inactiveCount)
	if inactiveCount != 0 {
		t.Error("Inactive session should have been deleted")
	}

	if deleted == 0 {
		t.Error("Expected at least one session to be cleaned up")
	}
}

func TestCleanupExpiredInvites(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})

	now := time.Now()
	cleaner := NewSessionCleaner(db)

	// Create a valid invite
	validInvite := models.SessionInvite{
		ID:        "valid-invite",
		SessionID: "test-session",
		TokenHash: "valid-hash",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	db.Create(&validInvite)

	// Create an expired invite
	expiredInvite := models.SessionInvite{
		ID:        "expired-invite",
		SessionID: "test-session",
		TokenHash: "expired-hash",
		ExpiresAt: now.Add(-24 * time.Hour),
		CreatedAt: now.Add(-48 * time.Hour),
	}
	db.Create(&expiredInvite)

	// Run cleanup
	deleted, err := cleaner.CleanupExpiredInvites()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Expired invite should be deleted
	var expiredCount int64
	db.Model(&models.SessionInvite{}).Where("id = ?", "expired-invite").Count(&expiredCount)
	if expiredCount != 0 {
		t.Error("Expired invite should have been deleted")
	}

	// Valid invite should still exist
	var validCount int64
	db.Model(&models.SessionInvite{}).Where("id = ?", "valid-invite").Count(&validCount)
	if validCount != 1 {
		t.Error("Valid invite should still exist")
	}

	if deleted != 1 {
		t.Errorf("Expected 1 invite deleted, got %d", deleted)
	}
}
