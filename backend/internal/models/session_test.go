package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	
	// Auto migrate the tables for testing
	err = db.AutoMigrate(&Session{}, &SessionInvite{}, &SessionParticipant{})
	if err != nil {
		panic("failed to migrate database")
	}
	
	return db
}

func TestSessionModel(t *testing.T) {
	db := setupTestDB()
	
	t.Run("should create session with correct fields", func(t *testing.T) {
		session := Session{
			ID:        "session123",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Active:    true,
		}
		
		result := db.Create(&session)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)
		
		var found Session
		db.First(&found, "id = ?", "session123")
		assert.Equal(t, "session123", found.ID)
		assert.True(t, found.Active)
	})
	
	t.Run("should use snake_case table name", func(t *testing.T) {
		session := Session{}
		assert.Equal(t, "sessions", session.TableName())
	})
}

func TestSessionInviteModel(t *testing.T) {
	db := setupTestDB()
	
	t.Run("should create session invite with correct fields", func(t *testing.T) {
		invite := SessionInvite{
			ID:        "invite123",
			SessionID: "session123",
			TokenHash: "hashed_token",
			Code:      "ABC123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}
		
		result := db.Create(&invite)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)
		
		var found SessionInvite
		db.First(&found, "id = ?", "invite123")
		assert.Equal(t, "invite123", found.ID)
		assert.Equal(t, "session123", found.SessionID)
		assert.Equal(t, "hashed_token", found.TokenHash)
		assert.Equal(t, "ABC123", found.Code)
	})
	
	t.Run("should use snake_case table name", func(t *testing.T) {
		invite := SessionInvite{}
		assert.Equal(t, "session_invites", invite.TableName())
	})
}

func TestSessionParticipantModel(t *testing.T) {
	db := setupTestDB()
	
	t.Run("should create session participant with correct fields", func(t *testing.T) {
		participant := SessionParticipant{
			SessionID:  "session123",
			UserID:     "user456",
			JoinedAt:   time.Now(),
			Role:       "host",
			LastSeenAt: time.Now(),
		}
		
		result := db.Create(&participant)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)
		
		var found SessionParticipant
		db.First(&found, "session_id = ? AND user_id = ?", "session123", "user456")
		assert.Equal(t, "session123", found.SessionID)
		assert.Equal(t, "user456", found.UserID)
		assert.Equal(t, "host", found.Role)
	})
	
	t.Run("should use snake_case table name", func(t *testing.T) {
		participant := SessionParticipant{}
		assert.Equal(t, "session_participants", participant.TableName())
	})
}

func TestSessionModelsIntegration(t *testing.T) {
	db := setupTestDB()
	
	t.Run("should create complete session with invite and participant", func(t *testing.T) {
		now := time.Now()
		expiresAt := now.Add(24 * time.Hour)
		
		// Create session
		session := Session{
			ID:        "session123",
			CreatedAt: now,
			ExpiresAt: expiresAt,
			Active:    true,
		}
		result := db.Create(&session)
		assert.NoError(t, result.Error)
		
		// Create invite for session
		invite := SessionInvite{
			ID:        "invite123",
			SessionID: session.ID,
			TokenHash: "hashed_secure_token",
			Code:      "ABC123",
			ExpiresAt: expiresAt,
			CreatedAt: now,
		}
		result = db.Create(&invite)
		assert.NoError(t, result.Error)
		
		// Create participant (session creator)
		participant := SessionParticipant{
			SessionID:  session.ID,
			UserID:     "creator123",
			JoinedAt:   now,
			Role:       "host",
			LastSeenAt: now,
		}
		result = db.Create(&participant)
		assert.NoError(t, result.Error)
		
		// Verify all components exist and relate to the session
		var foundSession Session
		err := db.First(&foundSession, "id = ?", "session123").Error
		assert.NoError(t, err)
		
		var foundInvite SessionInvite
		err = db.First(&foundInvite, "session_id = ?", "session123").Error
		assert.NoError(t, err)
		assert.Equal(t, "hashed_secure_token", foundInvite.TokenHash)
		
		var foundParticipant SessionParticipant
		err = db.First(&foundParticipant, "session_id = ? AND user_id = ?", "session123", "creator123").Error
		assert.NoError(t, err)
		assert.Equal(t, "host", foundParticipant.Role)
	})
}