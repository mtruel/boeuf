package repository

import (
	"errors"

	"github.com/mathias/boeuf/internal/models"
	"gorm.io/gorm"
)

type SpotifyTokenRepository struct {
	db *gorm.DB
}

func NewSpotifyTokenRepository(db *gorm.DB) *SpotifyTokenRepository {
	return &SpotifyTokenRepository{db: db}
}

// Save creates or updates a token record for a Spotify user
func (r *SpotifyTokenRepository) Save(token *models.SpotifyToken) error {
	// Use GORM's Upsert behavior with conflict handling
	result := r.db.Where("spotify_user_id = ?", token.SpotifyUserID).
		Assign(map[string]interface{}{
			"access_token":            token.AccessToken,
			"refresh_token_encrypted": token.RefreshTokenEncrypted,
			"expires_at":              token.ExpiresAt,
			"scope":                   token.Scope,
		}).
		FirstOrCreate(token)

	return result.Error
}

// GetBySpotifyUserID retrieves a token by Spotify user ID
func (r *SpotifyTokenRepository) GetBySpotifyUserID(spotifyUserID string) (*models.SpotifyToken, error) {
	var token models.SpotifyToken
	result := r.db.Where("spotify_user_id = ?", spotifyUserID).First(&token)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &token, nil
}
