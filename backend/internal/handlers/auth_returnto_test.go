package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/handlers"
	"github.com/mathias/boeuf/internal/session"
	"github.com/stretchr/testify/assert"
)

func TestOAuthReturnTo(t *testing.T) {
	// Create a cookie store for sessions
	store := sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!"))

	t.Run("Start with return_to query param stores it in session", func(t *testing.T) {
		handler := handlers.NewSpotifyAuthHandler(store, nil)

		// Set required env vars
		t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
		t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost:8080/auth/spotify/callback")

		req := httptest.NewRequest(http.MethodGet, "/auth/spotify/start?return_to=/join/abc123", nil)
		w := httptest.NewRecorder()

		handler.Start(w, req)

		// Should redirect to Spotify (302)
		assert.Equal(t, http.StatusFound, w.Code)

		// Extract session cookie to verify return_to was stored
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == session.SessionName {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "Session cookie should be set")

		// Create a new request with the session cookie to read it
		verifyReq := httptest.NewRequest(http.MethodGet, "/", nil)
		verifyReq.AddCookie(sessionCookie)
		sess, err := store.Get(verifyReq, session.SessionName)
		assert.NoError(t, err)

		returnTo, ok := sess.Values["return_to"].(string)
		assert.True(t, ok, "return_to should be in session")
		assert.Equal(t, "/join/abc123", returnTo)
	})

	t.Run("Callback with return_to in session redirects to it", func(t *testing.T) {
		tokenService, cleanup := setupTestTokenService(t)
		defer cleanup()
		handler := handlers.NewSpotifyAuthHandler(store, tokenService)
		t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
		t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost:8080/auth/spotify/callback")

		// Create a request with session containing return_to
		req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
		w := httptest.NewRecorder()

		// Pre-populate session with OAuth state and return_to
		sess, err := store.Get(req, session.SessionName)
		assert.NoError(t, err)
		sess.Values["state"] = "test_state"
		sess.Values["code_verifier"] = "test_verifier"
		sess.Values["return_to"] = "/join/xyz789"
		assert.NoError(t, sess.Save(req, w))

		// Get the session cookie from the recorder
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == session.SessionName {
				sessionCookie = cookie
				break
			}
		}

		assert.NotNil(t, sessionCookie, "Session cookie should be set")

		// Create new request with the session cookie
		callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
		callbackReq.AddCookie(sessionCookie)
		callbackW := httptest.NewRecorder()

		handler.Callback(callbackW, callbackReq)

		assert.Equal(t, http.StatusFound, callbackW.Code)
		assert.Equal(t, "/join/xyz789", callbackW.Header().Get("Location"))

		callbackCookies := callbackW.Result().Cookies()
		var callbackSessionCookie *http.Cookie
		for _, cookie := range callbackCookies {
			if cookie.Name == session.SessionName {
				callbackSessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, callbackSessionCookie, "Session cookie should be set on callback")

		verifyReq := httptest.NewRequest(http.MethodGet, "/", nil)
		verifyReq.AddCookie(callbackSessionCookie)
		updatedSession, err := store.Get(verifyReq, session.SessionName)
		assert.NoError(t, err)
		assert.Equal(t, "test-user-id", updatedSession.Values["spotify_user_id"])
		assert.Nil(t, updatedSession.Values["return_to"])
	})

	t.Run("Callback without return_to defaults to /", func(t *testing.T) {
		tokenService, cleanup := setupTestTokenService(t)
		defer cleanup()
		handler := handlers.NewSpotifyAuthHandler(store, tokenService)
		t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
		t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost:8080/auth/spotify/callback")

		req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
		w := httptest.NewRecorder()

		// Pre-populate session with OAuth state but NO return_to
		sess, err := store.Get(req, session.SessionName)
		assert.NoError(t, err)
		sess.Values["state"] = "test_state"
		sess.Values["code_verifier"] = "test_verifier"
		// Note: no return_to set
		assert.NoError(t, sess.Save(req, w))

		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == session.SessionName {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "Session cookie should be set")

		callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
		callbackReq.AddCookie(sessionCookie)
		callbackW := httptest.NewRecorder()

		handler.Callback(callbackW, callbackReq)

		assert.Equal(t, http.StatusFound, callbackW.Code)
		assert.Equal(t, "/", callbackW.Header().Get("Location"))

		callbackCookies := callbackW.Result().Cookies()
		var callbackSessionCookie *http.Cookie
		for _, cookie := range callbackCookies {
			if cookie.Name == session.SessionName {
				callbackSessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, callbackSessionCookie, "Session cookie should be set on callback")

		verifyReq := httptest.NewRequest(http.MethodGet, "/", nil)
		verifyReq.AddCookie(callbackSessionCookie)
		updatedSession, err := store.Get(verifyReq, session.SessionName)
		assert.NoError(t, err)
		assert.Equal(t, "test-user-id", updatedSession.Values["spotify_user_id"])
	})
}
