package models

import (
	"time"
)

// Session represents a synchronization session
type Session struct {
	ID        string    `gorm:"primaryKey;type:text"`
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Active    bool      `gorm:"not null;default:true"`

	// Baseline playback state (optional). Used to show context to newcomers.
	BaselineTrackID    string     `gorm:"type:text"`
	BaselineTrackName  string     `gorm:"type:text"`
	BaselineArtist     string     `gorm:"type:text"`
	BaselineIsPlaying  bool       `gorm:"not null;default:false"`
	BaselinePositionMs int64      `gorm:"not null;default:0"`
	BaselineDurationMs int64      `gorm:"not null;default:0"`
	BaselineImageURL   string     `gorm:"type:text"` // Album art URL
	BaselineCapturedAt *time.Time `gorm:""`
}

// TableName overrides the table name used by GORM
func (Session) TableName() string {
	return "sessions"
}

// SessionInvite represents an invitation to join a session
type SessionInvite struct {
	ID        string    `gorm:"primaryKey;type:text"`
	SessionID string    `gorm:"not null;type:text;index"`
	TokenHash string    `gorm:"not null"`
	Code      string    `gorm:""`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`

	// GORM relation (optional, for future use)
	Session Session `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName overrides the table name used by GORM
func (SessionInvite) TableName() string {
	return "session_invites"
}

// SessionParticipant represents a participant in a session
type SessionParticipant struct {
	SessionID  string    `gorm:"primaryKey;type:text"`
	UserID     string    `gorm:"primaryKey;type:text"`
	JoinedAt   time.Time `gorm:"not null"`
	Role       string    `gorm:"not null"`
	LastSeenAt time.Time `gorm:"not null"`
	SyncState  string    `gorm:"not null;default:'ready'"` // "ready" or "synced"

	// GORM relation (optional, for future use)
	Session Session `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName overrides the table name used by GORM
func (SessionParticipant) TableName() string {
	return "session_participants"
}
