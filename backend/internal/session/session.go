package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "boeuf-session"
	// MaxAge sets the session TTL (7 days, MVP default).
	// Update this value if session retention policy changes.
	MaxAge = 7 * 24 * 60 * 60
)

// Store abstracts session storage to make handlers testable.
type Store interface {
	Get(r *http.Request, name string) (*sessions.Session, error)
	New(r *http.Request, name string) (*sessions.Session, error)
}

// NewStore creates a new session store with appropriate security settings
func NewStore(sessionKey string, secure bool) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(sessionKey))

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   MaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}

	return store
}
