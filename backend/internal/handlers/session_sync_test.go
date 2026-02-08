package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSyncTest() (*gorm.DB, *SessionHandler, *sessions.CookieStore) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{}, &models.SpotifyToken{})

	store := sessions.NewCookieStore([]byte("test-session-key-1234567890123456"))
	handler := NewSessionHandler(store, db, testBaseURL)

	// Set up Spotify client and realtime hub
	tokenRepo := repository.NewSpotifyTokenRepository(db)
	encryptionKey := "12345678901234567890123456789012"
	spotifyClient := spotify.NewClient(tokenRepo, encryptionKey, "test-client-id")
	hub := realtime.NewHub()

	handler.SetSpotifyClient(spotifyClient)
	handler.SetRealtimeHub(hub)

	// Insert test Spotify tokens for common test users
	testTokens := []models.SpotifyToken{
		{
			SpotifyUserID:         "test-user-456",
			DisplayName:           "Test User",
			AccessToken:           "test-access-token",
			RefreshTokenEncrypted: "test-refresh-token",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
		{
			SpotifyUserID:         "user1",
			DisplayName:           "User One",
			AccessToken:           "token1",
			RefreshTokenEncrypted: "refresh1",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
		{
			SpotifyUserID:         "user2",
			DisplayName:           "User Two",
			AccessToken:           "token2",
			RefreshTokenEncrypted: "refresh2",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
	}
	for _, token := range testTokens {
		db.Create(&token)
	}

	return db, handler, store
}

// configureSpotifyMock configures the handler and client to use the mock server
func configureSpotifyMock(handler *SessionHandler, mockServer *httptest.Server) {
	handler.SetSpotifyAPIBaseURL(mockServer.URL)
	handler.SetSpotifyHTTPClient(mockServer.Client())
	handler.spotifyClient.SetAPIBaseURL(mockServer.URL + "/v1")
	handler.spotifyClient.SetTokenURL(mockServer.URL + "/api/token")
}

// setupSpotifyMock creates a mock Spotify server with devices and player endpoints
func setupSpotifyMock(t *testing.T, deviceCount int, playerResponse interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/me/player/devices":
			// Return devices list
			devices := make([]map[string]interface{}, deviceCount)
			for i := 0; i < deviceCount; i++ {
				devices[i] = map[string]interface{}{
					"id":        "device_" + string(rune('1'+i)),
					"name":      "Test Device " + string(rune('1'+i)),
					"type":      "Computer",
					"is_active": i == 0, // First device is active
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"devices": devices,
			})

		case r.URL.Path == "/v1/me/player":
			// Return player state
			if playerResponse == nil {
				w.WriteHeader(http.StatusNoContent)
			} else {
				json.NewEncoder(w).Encode(playerResponse)
			}

		case r.URL.Path == "/api/token":
			// Mock token refresh endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "refreshed-access-token",
				"expires_in":   3600,
				"scope":        "user-read-playback-state",
			})

		default:
			t.Logf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGetParticipantMe_Success(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session and participant
	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "participant",
		SyncState:  "ready",
		JoinedAt:   now,
		LastSeenAt: now.Unix(),
	})

	// Create request with auth
	req := httptest.NewRequest("GET", "/api/sessions/"+sessionID+"/me", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Save(req, w)

	handler.GetParticipantMe(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response GetParticipantMeResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, response.UserID)
	}
	if response.Role != "participant" {
		t.Errorf("Expected Role 'participant', got %s", response.Role)
	}
	if response.SyncState != "ready" {
		t.Errorf("Expected SyncState 'ready', got %s", response.SyncState)
	}
}

func TestGetParticipantMe_NotParticipant(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session but no participant
	sessionID := "test-session-123"
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})

	// Create request with auth
	req := httptest.NewRequest("GET", "/api/sessions/"+sessionID+"/me", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	// Mock authentication (user not in session)
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "different-user"
	session.Save(req, w)

	handler.GetParticipantMe(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["code"] != "FORBIDDEN" {
		t.Errorf("Expected error code FORBIDDEN, got %v", response["code"])
	}
}

func TestGetParticipantMe_Unauthenticated(t *testing.T) {
	_, handler, _ := setupSyncTest()

	req := httptest.NewRequest("GET", "/api/sessions/test-session/me", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": "test-session"})
	w := httptest.NewRecorder()

	handler.GetParticipantMe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestStartSync_AlreadySynced_Idempotent(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session and participant already synced
	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "participant",
		SyncState:  "synced", // Already synced
		JoinedAt:   now,
		LastSeenAt: now.Unix(),
	})

	// Create request with auth
	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (idempotent), got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response StartSyncResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.SyncState != "synced" {
		t.Errorf("Expected SyncState 'synced', got %s", response.SyncState)
	}
}

func TestStartSync_NotParticipant(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session but no participant
	sessionID := "test-session-123"
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})

	// Create request with auth
	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	// Mock authentication (user not in session)
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = "different-user"
	session.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}
}

func TestStartSync_NoSpotifyToken(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session and participant (with user that has NO Spotify token)
	sessionID := "test-session-123"
	userID := "user-without-token"  // Use user that doesn't have a token in setupSyncTest
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "participant",
		SyncState:  "ready",
		JoinedAt:   now,
		LastSeenAt: now.Unix(),
	})

	// Create request with auth
	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	// Mock authentication
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Save(req, w)

	handler.StartSync(w, req)

	// Should fail with SPOTIFY_NOT_CONNECTED
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["code"] != "SPOTIFY_NOT_CONNECTED" {
		t.Errorf("Expected error code SPOTIFY_NOT_CONNECTED, got %v", response["code"])
	}
}

func TestStartSync_Unauthenticated(t *testing.T) {
	_, handler, _ := setupSyncTest()

	req := httptest.NewRequest("POST", "/api/sessions/test-session/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": "test-session"})
	w := httptest.NewRecorder()

	handler.StartSync(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestStartSync_HappyPath_UpdatesSyncStateAndBaseline(t *testing.T) {
	db, handler, store := setupSyncTest()

	// Create session and participant (ready)
	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{
		ID:        sessionID,
		Active:    true,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "participant",
		SyncState:  "ready",
		JoinedAt:   now,
		LastSeenAt: now.Unix(),
	})

	// Add Spotify token
	encryptionKey := "12345678901234567890123456789012"
	encryptedRefresh, err := spotify.EncryptForTest("test-refresh-token", encryptionKey)
	if err != nil {
		t.Fatalf("Failed to encrypt refresh token: %v", err)
	}
	db.Create(&models.SpotifyToken{
		SpotifyUserID:         userID,
		AccessToken:           "test-access-token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(1 * time.Hour), // Far future to avoid refresh during test
		Scope:                 "user-read-playback-state",
	})

	// Mock Spotify with devices and player
	playerResp := map[string]interface{}{
		"item": map[string]interface{}{
			"id":          "track_123",
			"name":        "Test Song",
			"duration_ms": 200000,
			"artists": []map[string]interface{}{
				{"name": "Test Artist"},
			},
		},
		"is_playing":  true,
		"progress_ms": 1000,
	}
	spotifyServer := setupSpotifyMock(t, 1, playerResp) // 1 device available
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	// Drain broadcast to avoid blocking
	received := make(chan *realtime.BroadcastMessage, 1)
	go func() {
		received <- <-handler.realtimeHub.Broadcast
	}()

	// Create request with auth
	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response StartSyncResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if response.SyncState != "synced" {
		t.Fatalf("Expected SyncState 'synced', got %s", response.SyncState)
	}
	if response.NowPlaying == nil || response.NowPlaying.TrackName != "Test Song" {
		t.Fatalf("Expected NowPlaying with Test Song, got %#v", response.NowPlaying)
	}

	// Participant updated
	var p models.SessionParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&p).Error; err != nil {
		t.Fatalf("Failed to load participant: %v", err)
	}
	if p.SyncState != "synced" {
		t.Fatalf("Expected participant SyncState 'synced', got %s", p.SyncState)
	}

	// Baseline stored
	var s models.Session
	if err := db.Where("id = ?", sessionID).First(&s).Error; err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}
	if s.BaselineTrackID == "" || s.BaselineTrackName != "Test Song" {
		t.Fatalf("Expected baseline to be stored, got trackID=%q trackName=%q", s.BaselineTrackID, s.BaselineTrackName)
	}
	if s.BaselineCapturedAt == nil {
		t.Fatalf("Expected BaselineCapturedAt to be set")
	}

	// Ensure broadcast happened
	select {
	case <-received:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Expected a broadcast message")
	}
}

func TestStartSync_BroadcastsParticipantSyncStateChanged(t *testing.T) {
	_, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	handler.db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	handler.db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})
	handler.db.Create(&models.SpotifyToken{SpotifyUserID: userID, AccessToken: "test-access-token", RefreshTokenEncrypted: "dummy", ExpiresAt: time.Now().Add(1 * time.Hour), Scope: "user-read-playback-state"})

	playerResp := map[string]interface{}{
		"item": map[string]interface{}{
			"id":          "track_123",
			"name":        "Test Song",
			"duration_ms": 200000,
			"artists":     []map[string]interface{}{{"name": "Test Artist"}},
		},
		"is_playing":  true,
		"progress_ms": 1000,
	}
	spotifyServer := setupSpotifyMock(t, 1, playerResp)
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	broadcastCh := make(chan *realtime.BroadcastMessage, 1)
	go func() {
		broadcastCh <- <-handler.realtimeHub.Broadcast
	}()

	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	select {
	case msg := <-broadcastCh:
		var envelope realtime.Message
		if err := json.Unmarshal(msg.Message, &envelope); err != nil {
			t.Fatalf("Failed to unmarshal WS envelope: %v", err)
		}
		if envelope.Type != realtime.TypeParticipantSyncStateChanged {
			t.Fatalf("Expected message type %s, got %s", realtime.TypeParticipantSyncStateChanged, envelope.Type)
		}
		if envelope.SessionID != sessionID {
			t.Fatalf("Expected sessionId %s, got %s", sessionID, envelope.SessionID)
		}
		if envelope.EventSeq < 0 {
			t.Fatalf("Expected non-negative eventSeq, got %d", envelope.EventSeq)
		}
		if !strings.Contains(envelope.SentAt, "T") {
			t.Fatalf("Expected RFC3339 sentAt, got %q", envelope.SentAt)
		}
		var payload realtime.ParticipantSyncStateChangedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			t.Fatalf("Failed to unmarshal payload: %v", err)
		}
		if payload.UserID != userID || payload.SyncState != "synced" {
			t.Fatalf("Unexpected payload: %#v", payload)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Expected broadcast message")
	}
}

// TestStartSync_PlayerUnavailable_ReturnsStableError tests BUG #1 fix: no device returns SPOTIFY_NO_DEVICE
func TestStartSync_PlayerUnavailable_ReturnsStableError(t *testing.T) {
	db, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})
	db.Create(&models.SpotifyToken{SpotifyUserID: userID, AccessToken: "test-access-token", RefreshTokenEncrypted: "dummy", ExpiresAt: time.Now().Add(1 * time.Hour), Scope: "user-read-playback-state"})

	// Mock Spotify with NO devices - should return 503 SPOTIFY_NO_DEVICE
	spotifyServer := setupSpotifyMock(t, 0, nil)
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Expected status 503, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "SPOTIFY_NO_DEVICE" {
		t.Fatalf("Expected SPOTIFY_NO_DEVICE, got %v", resp["code"])
	}
	if resp["requiresActiveDevice"] != true {
		t.Fatalf("Expected requiresActiveDevice=true, got %v", resp["requiresActiveDevice"])
	}
	if !strings.Contains(resp["message"].(string), "No active Spotify device found") {
		t.Fatalf("Expected device error message, got %v", resp["message"])
	}
}

// TestStartSync_NoDevice_ReturnsRequiresActiveDeviceFlag validates AC 1: requiresActiveDevice flag
func TestStartSync_NoDevice_ReturnsRequiresActiveDeviceFlag(t *testing.T) {
	db, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})
	db.Create(&models.SpotifyToken{SpotifyUserID: userID, AccessToken: "test-access-token", RefreshTokenEncrypted: "dummy", ExpiresAt: time.Now().Add(1 * time.Hour), Scope: "user-read-playback-state"})

	// Mock Spotify with NO devices - should return 503 SPOTIFY_NO_DEVICE
	spotifyServer := setupSpotifyMock(t, 0, nil)
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Expected status 503, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "SPOTIFY_NO_DEVICE" {
		t.Fatalf("Expected SPOTIFY_NO_DEVICE, got %v", resp["code"])
	}
	if resp["requiresActiveDevice"] != true {
		t.Fatalf("Expected requiresActiveDevice=true, got %v", resp["requiresActiveDevice"])
	}
}

func TestStartSync_RateLimited_ReturnsStableError(t *testing.T) {
	db, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})
	db.Create(&models.SpotifyToken{SpotifyUserID: userID, AccessToken: "test-access-token", RefreshTokenEncrypted: "dummy", ExpiresAt: time.Now().Add(1 * time.Hour), Scope: "user-read-playback-state"})

	// Mock Spotify rate limited response
	spotifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	handler.StartSync(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Expected status 503, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "SPOTIFY_RATE_LIMITED" {
		t.Fatalf("Expected SPOTIFY_RATE_LIMITED, got %v", resp["code"])
	}
}

// TestStartSync_TimingUnder3Seconds validates NFR2: sync transition must complete within 3 seconds
func TestStartSync_TimingUnder3Seconds(t *testing.T) {
	db, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})

	encryptionKey := "12345678901234567890123456789012"
	encryptedRefresh, _ := spotify.EncryptForTest("test-refresh-token", encryptionKey)
	db.Create(&models.SpotifyToken{
		SpotifyUserID:         userID,
		AccessToken:           "test-access-token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		Scope:                 "user-read-playback-state",
	})

	playerResp := map[string]interface{}{
		"item": map[string]interface{}{
			"id":          "track_123",
			"name":        "Test Song",
			"duration_ms": 200000,
			"artists":     []map[string]interface{}{{"name": "Test Artist"}},
		},
		"is_playing":  true,
		"progress_ms": 1000,
	}
	spotifyServer := setupSpotifyMock(t, 1, playerResp)
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	// Drain broadcast to avoid blocking
	go func() {
		<-handler.realtimeHub.Broadcast
	}()

	req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	// Measure timing
	start := time.Now()
	handler.StartSync(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Validate NFR2: transition must complete within 3 seconds
	if elapsed > 3*time.Second {
		t.Fatalf("NFR2 violated: sync transition took %v, must be ≤ 3 seconds", elapsed)
	}

	t.Logf("✓ Sync transition completed in %v (within 3s limit)", elapsed)
}

// TestGetParticipantMe_AfterSyncStart_ReturnsSyncedState verifies that GetParticipantMe returns "synced" after successful StartSync
func TestGetParticipantMe_AfterSyncStart_ReturnsSyncedState(t *testing.T) {
	db, handler, store := setupSyncTest()

	sessionID := "test-session-123"
	userID := "test-user-456"
	now := time.Now()
	db.Create(&models.Session{ID: sessionID, Active: true, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	db.Create(&models.SessionParticipant{SessionID: sessionID, UserID: userID, Role: "participant", SyncState: "ready", JoinedAt: now, LastSeenAt: now.Unix()})

	encryptionKey := "12345678901234567890123456789012"
	encryptedRefresh, _ := spotify.EncryptForTest("test-refresh-token", encryptionKey)
	db.Create(&models.SpotifyToken{
		SpotifyUserID:         userID,
		AccessToken:           "test-access-token",
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		Scope:                 "user-read-playback-state",
	})

	playerResp := map[string]interface{}{
		"item": map[string]interface{}{
			"id":          "track_123",
			"name":        "Test Song",
			"duration_ms": 200000,
			"artists":     []map[string]interface{}{{"name": "Test Artist"}},
		},
		"is_playing":  true,
		"progress_ms": 1000,
	}
	spotifyServer := setupSpotifyMock(t, 1, playerResp)
	defer spotifyServer.Close()
	configureSpotifyMock(handler, spotifyServer)

	// Drain broadcast to avoid blocking
	go func() {
		<-handler.realtimeHub.Broadcast
	}()

	// Step 1: Verify initial state is "ready"
	req1 := httptest.NewRequest("GET", "/api/sessions/"+sessionID+"/me", nil)
	req1 = mux.SetURLVars(req1, map[string]string{"sessionId": sessionID})
	w1 := httptest.NewRecorder()
	sess1, _ := store.Get(req1, "boeuf-session")
	sess1.Values["spotify_user_id"] = userID
	sess1.Save(req1, w1)

	handler.GetParticipantMe(w1, req1)

	var initialResponse GetParticipantMeResponse
	json.Unmarshal(w1.Body.Bytes(), &initialResponse)
	if initialResponse.SyncState != "ready" {
		t.Fatalf("Expected initial SyncState 'ready', got %s", initialResponse.SyncState)
	}

	// Step 2: Call StartSync to transition to "synced"
	req2 := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/sync/start", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"sessionId": sessionID})
	w2 := httptest.NewRecorder()
	sess2, _ := store.Get(req2, "boeuf-session")
	sess2.Values["spotify_user_id"] = userID
	sess2.Save(req2, w2)

	handler.StartSync(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("StartSync failed: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Step 3: Verify GetParticipantMe now returns "synced"
	req3 := httptest.NewRequest("GET", "/api/sessions/"+sessionID+"/me", nil)
	req3 = mux.SetURLVars(req3, map[string]string{"sessionId": sessionID})
	w3 := httptest.NewRecorder()
	sess3, _ := store.Get(req3, "boeuf-session")
	sess3.Values["spotify_user_id"] = userID
	sess3.Save(req3, w3)

	handler.GetParticipantMe(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("GetParticipantMe failed: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var finalResponse GetParticipantMeResponse
	json.Unmarshal(w3.Body.Bytes(), &finalResponse)
	if finalResponse.SyncState != "synced" {
		t.Fatalf("Expected SyncState 'synced' after StartSync, got %s", finalResponse.SyncState)
	}
	if finalResponse.UserID != userID {
		t.Fatalf("Expected UserID %s, got %s", userID, finalResponse.UserID)
	}

	t.Logf("✓ GetParticipantMe correctly returns 'synced' state after StartSync transition")
}
