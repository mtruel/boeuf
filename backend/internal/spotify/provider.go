package spotify

import "context"

// Provider defines the interface for interacting with Spotify.
// It abstracts the implementation details of authentication and API calls.
type Provider interface {
	// GetValidToken returns a valid OAuth access token for the given user,
	// automatically refreshing it if necessary.
	GetValidToken(ctx context.Context, spotifyUserID string) (string, error)

	// Future methods for playback control will be added here
	// Play(ctx context.Context, userID string) error
	// Pause(ctx context.Context, userID string) error
}

// Ensure Client implements Provider
var _ Provider = (*Client)(nil)
