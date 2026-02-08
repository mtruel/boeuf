package spotify

import "fmt"

// RateLimitedError indicates Spotify has rate limited the request (HTTP 429).
// RetryAfterSeconds is best-effort parsed from the Retry-After header.
type RateLimitedError struct {
	RetryAfterSeconds int
}

func (e *RateLimitedError) Error() string {
	if e == nil {
		return "spotify rate limited"
	}
	if e.RetryAfterSeconds > 0 {
		return fmt.Sprintf("spotify rate limited, retry after %d seconds", e.RetryAfterSeconds)
	}
	return "spotify rate limited"
}
