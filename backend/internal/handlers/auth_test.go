package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mathias/boeuf/internal/handlers"
	"github.com/mathias/boeuf/internal/session"
)

func TestSpotifyAuthStart(t *testing.T) {
	// ARRANGE
	os.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
	os.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost/callback")
	defer os.Unsetenv("SPOTIFY_CLIENT_ID")
	defer os.Unsetenv("SPOTIFY_REDIRECT_URI")

	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store)

	req := httptest.NewRequest("GET", "/auth/spotify/start", nil)
	w := httptest.NewRecorder()

	// ACT
	handler.Start(w, req)

	// ASSERT
	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "https://accounts.spotify.com/authorize") {
		t.Errorf("Expected redirect to Spotify, got: %s", location)
	}

	if !strings.Contains(location, "code_challenge_method=S256") {
		t.Error("Expected PKCE parameters in URL")
	}

	if !strings.Contains(location, "client_id=test_client_id") {
		t.Error("Expected client_id in URL")
	}

	// Check session cookie was set
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected session cookie to be set")
	}
}

func TestSpotifyAuthStartMissingConfig(t *testing.T) {
	// ARRANGE - No env vars set
	os.Unsetenv("SPOTIFY_CLIENT_ID")
	os.Unsetenv("SPOTIFY_REDIRECT_URI")

	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store)

	req := httptest.NewRequest("GET", "/auth/spotify/start", nil)
	w := httptest.NewRecorder()

	// ACT
	handler.Start(w, req)

	// ASSERT
	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestSpotifyAuthLogout(t *testing.T) {
	// ARRANGE
	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()

	// Create session with spotify_user_id
	sess, _ := store.Get(req, session.SessionName)
	sess.Values["spotify_user_id"] = "test-user-123"
	sess.Save(req, w)

	// Get the session cookie from first response
	cookies := w.Result().Cookies()
	if len(cookies) > 0 {
		req.AddCookie(cookies[0])
	}

	// Reset recorder for actual logout test
	w = httptest.NewRecorder()

	// ACT
	handler.Logout(w, req)

	// ASSERT
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify session was cleared by getting session again
	req2 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	sess2, _ := store.Get(req2, session.SessionName)
	if sess2.Values["spotify_user_id"] != nil {
		t.Error("Expected spotify_user_id to be cleared from session")
	}
}

func TestSpotifyAuthLogoutRequiresPost(t *testing.T) {
	// ARRANGE
	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store)

	req := httptest.NewRequest("GET", "/api/auth/logout", nil)
	w := httptest.NewRecorder()

	// ACT
	handler.Logout(w, req)

	// ASSERT
	resp := w.Result()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
