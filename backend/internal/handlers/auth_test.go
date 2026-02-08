package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mathias/boeuf/internal/handlers"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/session"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestTokenService(t *testing.T) (*spotify.TokenService, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&models.SpotifyToken{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := repository.NewSpotifyTokenRepository(db)
	tokenService := spotify.NewTokenService(repo, "test-encryption-key-32-bytes-lng")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"expires_in":    3600,
				"scope":         "user-read-playback-state",
				"token_type":    "Bearer",
			})
		case "/me":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "test-user-id",
				"display_name": "Test User",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	tokenService.SetTokenURL(server.URL + "/token")
	tokenService.SetUserInfoURL(server.URL + "/me")

	return tokenService, server.Close
}

func setupFailingTokenService(t *testing.T) (*spotify.TokenService, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&models.SpotifyToken{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := repository.NewSpotifyTokenRepository(db)
	tokenService := spotify.NewTokenService(repo, "test-encryption-key-32-bytes-lng")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	tokenService.SetTokenURL(server.URL + "/token")

	return tokenService, server.Close
}

func setSessionValues(t *testing.T, store session.Store, req *http.Request, values map[string]interface{}) *http.Cookie {
	t.Helper()

	w := httptest.NewRecorder()
	sess, err := store.Get(req, session.SessionName)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	for key, value := range values {
		sess.Values[key] = value
	}
	if err := sess.Save(req, w); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == session.SessionName {
			return cookie
		}
	}

	t.Fatal("session cookie not set")
	return nil
}

func TestSpotifyAuthStart(t *testing.T) {
	// ARRANGE
	t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
	t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost/callback")

	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

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
	t.Setenv("SPOTIFY_CLIENT_ID", "")
	t.Setenv("SPOTIFY_REDIRECT_URI", "")

	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

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
	handler := handlers.NewSpotifyAuthHandler(store, nil)

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
	handler := handlers.NewSpotifyAuthHandler(store, nil)

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

func TestSpotifyAuthCallbackMissingState(t *testing.T) {
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	w := httptest.NewRecorder()

	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSpotifyAuthCallbackStateMismatch(t *testing.T) {
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=wrong_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state":         "expected_state",
		"code_verifier": "test_verifier",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=wrong_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", callbackW.Code)
	}
}

func TestSpotifyAuthCallbackMissingCodeVerifier(t *testing.T) {
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state": "test_state",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", callbackW.Code)
	}
}

func TestSpotifyAuthCallbackWithErrorQuery(t *testing.T) {
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?error=access_denied&state=test_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state": "test_state",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?error=access_denied&state=test_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", callbackW.Code)
	}
}

func TestSpotifyAuthCallbackNilTokenService(t *testing.T) {
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state":         "test_state",
		"code_verifier": "test_verifier",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", callbackW.Code)
	}
}

func TestSpotifyAuthCallbackTokenExchangeError(t *testing.T) {
	tokenService, cleanup := setupFailingTokenService(t)
	defer cleanup()
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, tokenService)

	t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
	t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state":         "test_state",
		"code_verifier": "test_verifier",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", callbackW.Code)
	}
}

func TestSpotifyAuthCallbackInvalidReturnToFallback(t *testing.T) {
	tokenService, cleanup := setupTestTokenService(t)
	defer cleanup()
	store := session.NewStore("test-session-key-32-bytes-long!!", false)
	handler := handlers.NewSpotifyAuthHandler(store, tokenService)

	t.Setenv("SPOTIFY_CLIENT_ID", "test_client_id")
	t.Setenv("SPOTIFY_REDIRECT_URI", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	sessionCookie := setSessionValues(t, store, req, map[string]interface{}{
		"state":         "test_state",
		"code_verifier": "test_verifier",
		"return_to":     "https://evil.example.com/phish",
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/spotify/callback?code=test_code&state=test_state", nil)
	callbackReq.AddCookie(sessionCookie)
	callbackW := httptest.NewRecorder()

	handler.Callback(callbackW, callbackReq)

	if callbackW.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", callbackW.Code)
	}
	if location := callbackW.Header().Get("Location"); location != "/" {
		t.Errorf("Expected redirect to '/', got %s", location)
	}
}
