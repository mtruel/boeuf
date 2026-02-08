package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mathias/boeuf/internal/crypto"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
)

var (
	ErrSpotifyNotConnected = errors.New("SPOTIFY_NOT_CONNECTED")
)

const (
	// TokenRefreshBufferMinutes defines how many minutes before expiration we should refresh the token
	// This buffer accounts for network latency and ensures we never use an expired token
	TokenRefreshBufferMinutes = 5
)

// Client handles Spotify API calls with automatic token refresh
// It implements the Provider interface
type Client struct {
	repo          *repository.SpotifyTokenRepository
	encryptionKey string
	clientID      string
	tokenURL      string
}

func NewClient(repo *repository.SpotifyTokenRepository, encryptionKey, clientID string) *Client {
	return &Client{
		repo:          repo,
		encryptionKey: encryptionKey,
		clientID:      clientID,
		tokenURL:      DefaultTokenURL,
	}
}

func (c *Client) SetTokenURL(url string) {
	c.tokenURL = url
}

// GetValidToken returns a valid access token, refreshing if necessary
func (c *Client) GetValidToken(ctx context.Context, spotifyUserID string) (string, error) {
	// Get token from DB
	token, err := c.repo.GetBySpotifyUserID(spotifyUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	if token == nil {
		return "", ErrSpotifyNotConnected
	}

	// Check if token needs refresh (expired or expires within buffer minutes)
	if time.Now().After(token.ExpiresAt.Add(-TokenRefreshBufferMinutes * time.Minute)) {
		// Need to refresh
		newToken, err := c.refreshToken(ctx, token)
		if err != nil {
			return "", err
		}
		return newToken, nil
	}

	// Token still valid
	return token.AccessToken, nil
}

// refreshToken refreshes an expired access token
func (c *Client) refreshToken(ctx context.Context, token *models.SpotifyToken) (string, error) {
	// Decrypt refresh token
	refreshToken, err := crypto.Decrypt(token.RefreshTokenEncrypted, c.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	// Prepare refresh request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request with retry logic for rate limiting
	client := &http.Client{Timeout: 10 * time.Second}
	var resp *http.Response

	for attempt := 0; attempt < 3; attempt++ {
		resp, err = client.Do(req)
		if err != nil {
			return "", fmt.Errorf("refresh request failed: %w", err)
		}

		// Handle Rate Limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 1
			if h := resp.Header.Get("Retry-After"); h != "" {
				if n, _ := strconv.Atoi(h); n > 0 {
					retryAfter = n
				}
			}

			// If wait is too long (e.g. > 5s), fail fast for user experience.
			// Otherwise wait and retry.
			if retryAfter > 5 {
				resp.Body.Close()
				return "", &RateLimitedError{RetryAfterSeconds: retryAfter}
			}

			resp.Body.Close()
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		// Not a rate limit error, proceed
		break
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 1
		if h := resp.Header.Get("Retry-After"); h != "" {
			if n, _ := strconv.Atoi(h); n > 0 {
				retryAfter = n
			}
		}
		return "", &RateLimitedError{RetryAfterSeconds: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		// Refresh failed (likely revoked token)
		return "", ErrSpotifyNotConnected
	}

	// Parse response
	var refreshResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return "", fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Update token in DB
	token.AccessToken = refreshResp.AccessToken
	token.ExpiresAt = time.Now().Add(time.Duration(refreshResp.ExpiresIn) * time.Second)
	token.Scope = refreshResp.Scope

	if err := c.repo.Save(token); err != nil {
		return "", fmt.Errorf("failed to update token: %w", err)
	}

	return refreshResp.AccessToken, nil
}

// EncryptForTest is a helper for tests to encrypt refresh tokens
func EncryptForTest(plaintext, key string) (string, error) {
	return crypto.Encrypt(plaintext, key)
}
