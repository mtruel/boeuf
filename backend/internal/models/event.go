package models

import (
	"time"
)

// Event represents an event in the session event log
// Append-only table for audit trail and future resync capability
type Event struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	SessionID   string    `gorm:"not null;type:text;index:idx_events_session_seq"`
	EventSeq    int64     `gorm:"not null;index:idx_events_session_seq"`
	EventType   string    `gorm:"not null;type:text"` // PLAYER_PAUSED, PLAYER_RESUMED, etc.
	PayloadJSON string    `gorm:"not null;type:text"` // JSON payload
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName overrides the table name used by GORM
func (Event) TableName() string {
	return "events"
}
