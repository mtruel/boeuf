package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	apiBaseURL    string
}

func NewClient(repo *repository.SpotifyTokenRepository, encryptionKey, clientID string) *Client {
	return &Client{
		repo:          repo,
		encryptionKey: encryptionKey,
		clientID:      clientID,
		tokenURL:      DefaultTokenURL,
		apiBaseURL:    spotifyAPIBase,
	}
}

func (c *Client) SetTokenURL(url string) {
	c.tokenURL = url
}

func (c *Client) SetAPIBaseURL(url string) {
	c.apiBaseURL = url
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

// PlayerState represents the current state of the Spotify player
type PlayerState struct {
	IsPlaying  bool   `json:"isPlaying"`
	PositionMs int64  `json:"positionMs"`
	TrackID    string `json:"trackId"`
	TrackName  string `json:"trackName"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int64  `json:"durationMs"`
	ImageURL   string `json:"imageUrl"`
}

// SpotifyPlayerResponse represents the response from GET /me/player
type SpotifyPlayerResponse struct {
	IsPlaying bool  `json:"is_playing"`
	Progress  int64 `json:"progress_ms"`
	Item      *struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Duration int64  `json:"duration_ms"`
		Artists  []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Album struct {
			Name   string `json:"name"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"album"`
	} `json:"item"`
}

const spotifyAPIBase = "https://api.spotify.com/v1"

// GetPlayerState retrieves the current player state from Spotify (Trust but Verify)
func (c *Client) GetPlayerState(ctx context.Context, spotifyUserID string) (*PlayerState, error) {
	accessToken, err := c.GetValidToken(ctx, spotifyUserID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.apiBaseURL+"/me/player", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create player state request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("player state request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 1
		if h := resp.Header.Get("Retry-After"); h != "" {
			if n, _ := strconv.Atoi(h); n > 0 {
				retryAfter = n
			}
		}
		return nil, &RateLimitedError{RetryAfterSeconds: retryAfter}
	}

	// No active device or no content playing
	if resp.StatusCode == http.StatusNoContent {
		return &PlayerState{
			IsPlaying: false,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify player state error %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var playerResp SpotifyPlayerResponse
	if err := json.Unmarshal(body, &playerResp); err != nil {
		return nil, fmt.Errorf("failed to parse player state: %w", err)
	}

	// Extract player state
	state := &PlayerState{
		IsPlaying:  playerResp.IsPlaying,
		PositionMs: playerResp.Progress,
	}

	// Extract track info if available
	if playerResp.Item != nil {
		// Validate required fields exist
		if playerResp.Item.ID == "" {
			return nil, fmt.Errorf("missing track ID in Spotify response")
		}
		if playerResp.Item.Name == "" {
			return nil, fmt.Errorf("missing track name in Spotify response (track: %s)", playerResp.Item.ID)
		}

		state.TrackID = "spotify:track:" + playerResp.Item.ID
		state.TrackName = playerResp.Item.Name
		state.DurationMs = playerResp.Item.Duration

		// Validate duration - critical for seek validation
		if state.DurationMs <= 0 {
			return nil, fmt.Errorf("invalid track duration: %d (track: %s)", state.DurationMs, state.TrackID)
		}

		if len(playerResp.Item.Artists) > 0 {
			state.Artist = playerResp.Item.Artists[0].Name
		}

		// Album may be nil for podcasts/local files
		if playerResp.Item.Album.Name != "" {
			state.Album = playerResp.Item.Album.Name
		}
		if len(playerResp.Item.Album.Images) > 0 {
			state.ImageURL = playerResp.Item.Album.Images[0].URL
		}
	}

	return state, nil
}

// Device represents a Spotify playback device
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
}

// GetAvailableDevices retrieves the list of available Spotify devices
func (c *Client) GetAvailableDevices(ctx context.Context, spotifyUserID string) ([]Device, error) {
	accessToken, err := c.GetValidToken(ctx, spotifyUserID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.apiBaseURL+"/me/player/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create devices request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("devices request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 1
		if h := resp.Header.Get("Retry-After"); h != "" {
			if n, _ := strconv.Atoi(h); n > 0 {
				retryAfter = n
			}
		}
		return nil, &RateLimitedError{RetryAfterSeconds: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify devices error %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var devicesResp struct {
		Devices []Device `json:"devices"`
	}
	if err := json.Unmarshal(body, &devicesResp); err != nil {
		return nil, fmt.Errorf("failed to parse devices response: %w", err)
	}

	return devicesResp.Devices, nil
}

// Pause pauses playback on the user's active device
func (c *Client) Pause(ctx context.Context, spotifyUserID string) error {
	return c.playerCommand(ctx, spotifyUserID, "PUT", "/me/player/pause", nil)
}

// Resume resumes playback on the user's active device
func (c *Client) Resume(ctx context.Context, spotifyUserID string) error {
	return c.playerCommand(ctx, spotifyUserID, "PUT", "/me/player/play", nil)
}

// Next skips to the next track
func (c *Client) Next(ctx context.Context, spotifyUserID string) error {
	return c.playerCommand(ctx, spotifyUserID, "POST", "/me/player/next", nil)
}

// Seek seeks to a position in the current track
func (c *Client) Seek(ctx context.Context, spotifyUserID string, positionMs int64) error {
	endpoint := fmt.Sprintf("/me/player/seek?position_ms=%d", positionMs)
	return c.playerCommand(ctx, spotifyUserID, "PUT", endpoint, nil)
}

// playerCommand executes a player control command (Pause, Resume, Next, Seek)
func (c *Client) playerCommand(ctx context.Context, spotifyUserID, method, endpoint string, body io.Reader) error {
	accessToken, err := c.GetValidToken(ctx, spotifyUserID)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiBaseURL+endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create player command request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("player command request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 1
		if h := resp.Header.Get("Retry-After"); h != "" {
			if n, _ := strconv.Atoi(h); n > 0 {
				retryAfter = n
			}
		}
		log.Printf("Spotify player command rate limited endpoint=%s retryAfter=%d", endpoint, retryAfter)
		return &RateLimitedError{RetryAfterSeconds: retryAfter}
	}

	// Success cases: accept any 2xx response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		respBody, _ := io.ReadAll(resp.Body)
		respBodyStr := strings.TrimSpace(string(respBody))
		if respBodyStr != "" {
			log.Printf("Spotify player command returned status=%d endpoint=%s body=%s", resp.StatusCode, endpoint, respBodyStr)
		}
		return nil
	}

	// Handle specific errors
	respBody, _ := io.ReadAll(resp.Body)
	respBodyStr := string(respBody)
	log.Printf("Spotify player command failed endpoint=%s status=%d body=%s", endpoint, resp.StatusCode, respBodyStr)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return errors.New("SPOTIFY_NO_DEVICE")
	case http.StatusForbidden:
		if isRestrictionViolation(respBody) || strings.Contains(strings.ToLower(respBodyStr), "restriction violated") {
			return errors.New("SPOTIFY_RESTRICTION")
		}
		return errors.New("SPOTIFY_FORBIDDEN")
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		// Check if it's a "No active device" error
		if strings.Contains(strings.ToLower(respBodyStr), "no active device") {
			return errors.New("SPOTIFY_NO_ACTIVE_DEVICE")
		}
		return fmt.Errorf("SPOTIFY_UNAVAILABLE: %d %s", resp.StatusCode, respBodyStr)
	default:
		return fmt.Errorf("SPOTIFY_UNAVAILABLE: %d %s", resp.StatusCode, respBodyStr)
	}
}

func isRestrictionViolation(respBody []byte) bool {
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &payload); err != nil {
		return false
	}

	message := strings.ToLower(payload.Error.Message)
	if strings.Contains(message, "restriction violated") {
		return true
	}

	reason := strings.ToLower(payload.Error.Reason)
	return strings.Contains(reason, "restriction")
}
