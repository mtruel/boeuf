package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mathias/boeuf/internal/crypto"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
)

const (
	DefaultTokenURL    = "https://accounts.spotify.com/api/token"
	DefaultUserInfoURL = "https://api.spotify.com/v1/me"
)

type TokenService struct {
	repo          *repository.SpotifyTokenRepository
	encryptionKey string
	tokenURL      string
	userInfoURL   string
}

func NewTokenService(repo *repository.SpotifyTokenRepository, encryptionKey string) *TokenService {
	return &TokenService{
		repo:          repo,
		encryptionKey: encryptionKey,
		tokenURL:      DefaultTokenURL,
		userInfoURL:   DefaultUserInfoURL,
	}
}

// SetTokenURL allows overriding the token URL for testing
func (s *TokenService) SetTokenURL(url string) {
	s.tokenURL = url
}

// SetUserInfoURL allows overriding the user info URL for testing
func (s *TokenService) SetUserInfoURL(url string) {
	s.userInfoURL = url
}

// TokenResponse represents Spotify's token endpoint response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// UserInfoResponse represents Spotify's user info response
type UserInfoResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// ExchangeResult contains the result of token exchange
type ExchangeResult struct {
	SpotifyUserID string
	AccessToken   string
	ExpiresAt     time.Time
}

// ExchangeCodeForTokens exchanges authorization code for access and refresh tokens
func (s *TokenService) ExchangeCodeForTokens(ctx context.Context, code, codeVerifier, clientID, redirectURI string) (*ExchangeResult, error) {
	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", s.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp map[string]interface{}
		json.Unmarshal(body, &errorResp)
		return nil, fmt.Errorf("spotify token error (status %d): %v", resp.StatusCode, errorResp)
	}

	// Parse successful response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Get user info (ID and display name)
	spotifyUserID, displayName, err := s.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Encrypt refresh token
	encryptedRefreshToken, err := crypto.Encrypt(tokenResp.RefreshToken, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Save to database
	token := &models.SpotifyToken{
		SpotifyUserID:         spotifyUserID,
		DisplayName:           displayName,
		AccessToken:           tokenResp.AccessToken,
		RefreshTokenEncrypted: encryptedRefreshToken,
		ExpiresAt:             expiresAt,
		Scope:                 tokenResp.Scope,
	}

	if err := s.repo.Save(token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return &ExchangeResult{
		SpotifyUserID: spotifyUserID,
		AccessToken:   tokenResp.AccessToken,
		ExpiresAt:     expiresAt,
	}, nil
}

// getUserInfo fetches the Spotify user ID and display name using the access token
func (s *TokenService) getUserInfo(ctx context.Context, accessToken string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.userInfoURL, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New("failed to fetch user info from Spotify")
	}

	var userInfo UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", "", err
	}

	return userInfo.ID, userInfo.DisplayName, nil
}
