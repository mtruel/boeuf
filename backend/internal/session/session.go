package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "boeuf-session"
	// MaxAge = 7 days in seconds
	MaxAge = 7 * 24 * 60 * 60
)

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
