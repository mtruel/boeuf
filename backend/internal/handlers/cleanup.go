package handlers

import (
	"log"
	"time"

	"github.com/mathias/boeuf/internal/models"
	"gorm.io/gorm"
)

// SessionCleaner handles periodic cleanup of expired sessions and invites
type SessionCleaner struct {
	db *gorm.DB
}

// NewSessionCleaner creates a new SessionCleaner instance
func NewSessionCleaner(db *gorm.DB) *SessionCleaner {
	return &SessionCleaner{db: db}
}

// CleanupExpiredSessions deactivates expired sessions and deletes old inactive sessions
// Returns the number of sessions processed
func (c *SessionCleaner) CleanupExpiredSessions() (int64, error) {
	now := time.Now()
	
	// Deactivate expired sessions that are still active
	result := c.db.Model(&models.Session{}).
		Where("active = ? AND expires_at < ?", true, now).
		Update("active", false)
	
	if result.Error != nil {
		log.Printf("ERROR: Failed to deactivate expired sessions: %v", result.Error)
		return 0, result.Error
	}
	
	deactivatedCount := result.RowsAffected
	
	// Delete inactive sessions that expired more than 7 days ago
	// This gives a grace period for potential auditing/debugging
	deleteThreshold := now.Add(-7 * 24 * time.Hour)
	result = c.db.Where("active = ? AND expires_at < ?", false, deleteThreshold).
		Delete(&models.Session{})
	
	if result.Error != nil {
		log.Printf("ERROR: Failed to delete old inactive sessions: %v", result.Error)
		return deactivatedCount, result.Error
	}
	
	deletedCount := result.RowsAffected
	totalProcessed := deactivatedCount + deletedCount
	
	if totalProcessed > 0 {
		log.Printf("INFO: Cleaned up %d sessions (%d deactivated, %d deleted)", 
			totalProcessed, deactivatedCount, deletedCount)
	}
	
	return totalProcessed, nil
}

// CleanupExpiredInvites deletes expired invites
// Returns the number of invites deleted
func (c *SessionCleaner) CleanupExpiredInvites() (int64, error) {
	now := time.Now()
	
	result := c.db.Where("expires_at < ?", now).Delete(&models.SessionInvite{})
	
	if result.Error != nil {
		log.Printf("ERROR: Failed to delete expired invites: %v", result.Error)
		return 0, result.Error
	}
	
	if result.RowsAffected > 0 {
		log.Printf("INFO: Deleted %d expired invites", result.RowsAffected)
	}
	
	return result.RowsAffected, nil
}

// RunPeriodicCleanup runs both session and invite cleanup and should be called periodically
func (c *SessionCleaner) RunPeriodicCleanup() error {
	log.Println("INFO: Starting periodic cleanup of expired sessions and invites")
	
	_, err := c.CleanupExpiredSessions()
	if err != nil {
		return err
	}
	
	_, err = c.CleanupExpiredInvites()
	if err != nil {
		return err
	}
	
	return nil
}
