package models

import (
	"time"
)

// SpotifyToken stores Spotify OAuth tokens for a user
type SpotifyToken struct {
	ID                    uint      `gorm:"primaryKey"`
	SpotifyUserID         string    `gorm:"uniqueIndex;not null"`
	DisplayName           string    `gorm:"not null"` // Spotify user display name
	AccessToken           string    `gorm:"not null"`
	RefreshTokenEncrypted string    `gorm:"not null"`
	ExpiresAt             time.Time `gorm:"not null"`
	Scope                 string    `gorm:"not null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TableName overrides the table name used by GORM
func (SpotifyToken) TableName() string {
	return "spotify_tokens"
}
