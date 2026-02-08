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
	"github.com/gorilla/websocket"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/session"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestWebSocket creates a test WebSocket handler with in-memory database
func setupTestWebSocket(t *testing.T) (*WebSocketHandler, *gorm.DB, *sessions.CookieStore, *realtime.Hub) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	db.AutoMigrate(&models.SpotifyToken{}, &models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{})

	// Create session store
	store := session.NewStore("test-session-key-32-bytes-long!", false)

	// Create hub
	hub := realtime.NewHub()
	go hub.Run()

	// Create handler
	handler := NewWebSocketHandler(hub, store, db)

	// Insert test Spotify tokens for common test users
	testTokens := []models.SpotifyToken{
		{
			SpotifyUserID:         "user_test_1",
			DisplayName:           "Test User 1",
			AccessToken:           "test-access-token",
			RefreshTokenEncrypted: "test-refresh-token",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
		{
			SpotifyUserID:         "user_test_2",
			DisplayName:           "Test User 2",
			AccessToken:           "test-access-token-2",
			RefreshTokenEncrypted: "test-refresh-token-2",
			ExpiresAt:             time.Now().Add(1 * time.Hour),
			Scope:                 "user-read-playback-state user-modify-playback-state",
		},
	}
	for _, token := range testTokens {
		db.Create(&token)
	}

	return handler, db, store, hub
}

// Test WebSocket upgrade with valid authentication
func TestWebSocketUpgradeWithAuth(t *testing.T) {
	handler, db, store, _ := setupTestWebSocket(t)

	// Create test session and participant
	sessionID := "sess_test_123"
	userID := "user_test_1"

	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	participant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "host",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	}
	db.Create(&participant)

	// Create HTTP server for WebSocket test
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// Create session cookie
	req := httptest.NewRequest("GET", "/ws/"+sessionID, nil)
	w := httptest.NewRecorder()

	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	// Get cookie from response
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No session cookie created")
	}

	// Convert test server URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Create WebSocket connection with cookie
	header := http.Header{}
	// Only send name=value in Cookie header, not attributes like Path/HttpOnly
	header.Add("Cookie", cookies[0].Name+"="+cookies[0].Value)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v (status: %v)", err, resp)
	}
	defer conn.Close()

	// Should receive SESSION_SNAPSHOT on connection
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read initial message: %v", err)
	}

	var msg realtime.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if msg.Type != realtime.TypeSessionSnapshot {
		t.Errorf("Expected SESSION_SNAPSHOT, got %s", msg.Type)
	}

	if msg.SessionID != sessionID {
		t.Errorf("Expected sessionID %s, got %s", sessionID, msg.SessionID)
	}

	if msg.EventSeq != 0 {
		t.Errorf("First message should have eventSeq 0, got %d", msg.EventSeq)
	}

	// Verify payload contains participants
	var snapshot realtime.SessionSnapshotPayload
	if err := json.Unmarshal(msg.Payload, &snapshot); err != nil {
		t.Fatalf("Failed to unmarshal snapshot payload: %v", err)
	}

	if len(snapshot.Participants) == 0 {
		t.Error("Snapshot should contain at least one participant")
	}

	// Find our participant
	found := false
	for _, p := range snapshot.Participants {
		if p.UserID == userID {
			found = true
			if p.Role != "host" {
				t.Errorf("Expected role host, got %s", p.Role)
			}
			if p.ConnectionStatus != "online" {
				t.Errorf("Expected status online, got %s", p.ConnectionStatus)
			}
		}
	}

	if !found {
		t.Error("Participant not found in snapshot")
	}
}

// Test WebSocket refuses connection without authentication
func TestWebSocketRefuseNoAuth(t *testing.T) {
	handler, db, _, _ := setupTestWebSocket(t)

	// Create test session but NO participant
	sessionID := "sess_test_123"

	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	// Create HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Try to connect WITHOUT cookie
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		conn.Close()
		t.Error("WebSocket should have refused connection without auth")
	}

	if resp != nil && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 401 or 400, got %d", resp.StatusCode)
	}
}

// Test WebSocket refuses connection for non-participant
func TestWebSocketRefuseNonParticipant(t *testing.T) {
	handler, db, store, _ := setupTestWebSocket(t)

	// Create session
	sessionID := "sess_test_123"
	ownerID := "user_owner"
	nonParticipantID := "user_intruder"

	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	// Create participant for owner only
	participant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     ownerID,
		Role:       "host",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	}
	db.Create(&participant)

	// Create HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// Create session cookie for NON-participant
	req := httptest.NewRequest("GET", "/ws/"+sessionID, nil)
	w := httptest.NewRecorder()

	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = nonParticipantID // Not a participant!
	sess.Save(req, w)

	cookies := w.Result().Cookies()
	header := http.Header{}
	header.Add("Cookie", cookies[0].Name+"="+cookies[0].Value)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Connect as non-participant: server should accept upgrade, send WS_FORBIDDEN, then close.
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v (status: %v)", err, resp)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WS_FORBIDDEN message: %v", err)
	}

	var msg realtime.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if msg.Type != realtime.TypeWSForbidden {
		t.Fatalf("Expected WS_FORBIDDEN, got %s", msg.Type)
	}

	if msg.SessionID != sessionID {
		t.Fatalf("Expected sessionId %s, got %s", sessionID, msg.SessionID)
	}

	var errPayload realtime.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &errPayload); err != nil {
		t.Fatalf("Failed to unmarshal error payload: %v", err)
	}
	if errPayload.Code != "WS_FORBIDDEN" {
		t.Fatalf("Expected code WS_FORBIDDEN, got %s", errPayload.Code)
	}

	// Next read should observe a policy violation close (1008)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("Expected WS close after WS_FORBIDDEN")
	}
	if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
		t.Fatalf("Expected close code 1008 (policy violation), got %v", err)
	}
}

// Test PARTICIPANT_JOINED broadcast
func TestParticipantJoinedBroadcast(t *testing.T) {
	handler, db, store, _ := setupTestWebSocket(t)

	sessionID := "sess_test_123"
	user1ID := "user_1"
	user2ID := "user_2"

	// Create session
	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	// Create participants
	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     user1ID,
		Role:       "host",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	})

	db.Create(&models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     user2ID,
		Role:       "participant",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	})

	// Create server
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Connect user1
	req1 := httptest.NewRequest("GET", "/ws/"+sessionID, nil)
	w1 := httptest.NewRecorder()
	sess1, _ := store.Get(req1, "boeuf-session")
	sess1.Values["spotify_user_id"] = user1ID
	sess1.Save(req1, w1)

	header1 := http.Header{}
	cookies1 := w1.Result().Cookies()
	header1.Add("Cookie", cookies1[0].Name+"="+cookies1[0].Value)

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
	if err != nil {
		t.Fatalf("User1 connection failed: %v", err)
	}
	defer conn1.Close()

	// Read snapshot for user1
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn1.ReadMessage() // Consume snapshot

	// Connect user2 (should trigger PARTICIPANT_JOINED for user1)
	req2 := httptest.NewRequest("GET", "/ws/"+sessionID, nil)
	w2 := httptest.NewRecorder()
	sess2, _ := store.Get(req2, "boeuf-session")
	sess2.Values["spotify_user_id"] = user2ID
	sess2.Save(req2, w2)

	header2 := http.Header{}
	cookies2 := w2.Result().Cookies()
	header2.Add("Cookie", cookies2[0].Name+"="+cookies2[0].Value)

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, header2)
	if err != nil {
		t.Fatalf("User2 connection failed: %v", err)
	}
	defer conn2.Close()

	// User1 should receive PARTICIPANT_JOINED event
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read PARTICIPANT_JOINED: %v", err)
	}

	var msg realtime.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if msg.Type != realtime.TypeParticipantJoined {
		t.Errorf("Expected PARTICIPANT_JOINED, got %s", msg.Type)
	}

	var payload realtime.ParticipantJoinedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload.UserID != user2ID {
		t.Errorf("Expected joined user %s, got %s", user2ID, payload.UserID)
	}

	if payload.Role != "participant" {
		t.Errorf("Expected role participant, got %s", payload.Role)
	}
}

// Test eventSeq is monotonic
func TestEventSeqMonotonic(t *testing.T) {
	handler, db, store, _ := setupTestWebSocket(t)

	sessionID := "sess_test_123"
	userID := "user_test_1"

	// Setup session and participant
	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	participant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "host",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	}
	db.Create(&participant)

	// Create server
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Connect
	req := httptest.NewRequest("GET", "/ws/"+sessionID, nil)
	w := httptest.NewRecorder()
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	header := http.Header{}
	cookies := w.Result().Cookies()
	header.Add("Cookie", cookies[0].Name+"="+cookies[0].Value)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	// Read snapshot (eventSeq should be 0)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var msg1 realtime.Message
	json.Unmarshal(msgBytes, &msg1)

	if msg1.EventSeq != 0 {
		t.Errorf("First message should have eventSeq 0, got %d", msg1.EventSeq)
	}

	// Future messages would have eventSeq 1, 2, 3... (tested in Hub tests)
}

// Test WebSocket refuses empty session cookie
func TestWebSocketRefuseEmptySession(t *testing.T) {
	handler, db, _, _ := setupTestWebSocket(t)

	// Create test session and participant (but don't create session cookie)
	sessionID := "sess_test_empty"
	userID := "user_test_empty"

	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	participant := models.SessionParticipant{
		SessionID:  sessionID,
		UserID:     userID,
		Role:       "host",
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now().Unix(),
	}
	db.Create(&participant)

	// Create HTTP server for WebSocket test
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert test server URL to WebSocket URL (no cookie set)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Try to connect WITHOUT any session cookie
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		conn.Close()
		t.Error("WebSocket should have refused connection without session cookie")
	}

	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// Test end-to-end WebSocket integration with multiple participants
func TestWebSocketEndToEndIntegration(t *testing.T) {
	handler, db, store, hub := setupTestWebSocket(t)

	sessionID := "sess_integration_test"
	user1ID := "user_1"
	user2ID := "user_2"

	// Set up disconnect callback for PARTICIPANT_LEFT broadcasts
	hub.SetOnClientDisconnect(handler.BroadcastParticipantLeft)

	// Create test session
	session := models.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}
	db.Create(&session)

	// Create participants
	participants := []models.SessionParticipant{
		{SessionID: sessionID, UserID: user1ID, Role: "host", JoinedAt: time.Now(), LastSeenAt: time.Now().Unix()},
		{SessionID: sessionID, UserID: user2ID, Role: "participant", JoinedAt: time.Now(), LastSeenAt: time.Now().Unix()},
	}
	for _, p := range participants {
		db.Create(&p)
	}

	// Create HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/ws/{sessionId}", handler.HandleConnection)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID

	// Connect first user
	conn1 := connectWithAuth(t, wsURL, user1ID, store)
	defer conn1.Close()

	// Read SESSION_SNAPSHOT from first user
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg1, _ := conn1.ReadMessage()
	var snapshot realtime.Message
	json.Unmarshal(msg1, &snapshot)

	// Verify snapshot contains both participants (user1 online, user2 offline)
	var snapshotPayload realtime.SessionSnapshotPayload
	json.Unmarshal(snapshot.Payload, &snapshotPayload)

	if len(snapshotPayload.Participants) != 2 {
		t.Errorf("Expected 2 participants in snapshot, got %d", len(snapshotPayload.Participants))
	}

	// Connect second user
	conn2 := connectWithAuth(t, wsURL, user2ID, store)
	defer conn2.Close()

	// User1 should receive PARTICIPANT_JOINED for user2
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg2, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("User1 didn't receive PARTICIPANT_JOINED: %v", err)
	}

	var joinedMsg realtime.Message
	json.Unmarshal(msg2, &joinedMsg)

	if joinedMsg.Type != realtime.TypeParticipantJoined {
		t.Errorf("Expected PARTICIPANT_JOINED, got %v", joinedMsg.Type)
	}

	var joinedPayload realtime.ParticipantJoinedPayload
	json.Unmarshal(joinedMsg.Payload, &joinedPayload)

	if joinedPayload.UserID != user2ID {
		t.Errorf("Expected joined userID %s, got %s", user2ID, joinedPayload.UserID)
	}

	// Disconnect user2
	conn2.Close()

	// User1 should receive PARTICIPANT_LEFT for user2
	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg3, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("User1 didn't receive PARTICIPANT_LEFT: %v", err)
	}

	var leftMsg realtime.Message
	json.Unmarshal(msg3, &leftMsg)

	if leftMsg.Type != realtime.TypeParticipantLeft {
		t.Errorf("Expected PARTICIPANT_LEFT, got %v", leftMsg.Type)
	}

	var leftPayload realtime.ParticipantLeftPayload
	json.Unmarshal(leftMsg.Payload, &leftPayload)

	if leftPayload.UserID != user2ID {
		t.Errorf("Expected left userID %s, got %s", user2ID, leftPayload.UserID)
	}

	t.Logf("End-to-end integration test passed: participant join/leave cycle completed")
}

// Helper function to connect with authentication
func connectWithAuth(t *testing.T, wsURL, userID string, store *sessions.CookieStore) *websocket.Conn {
	// Create session cookie
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["spotify_user_id"] = userID
	sess.Save(req, w)

	cookies := w.Result().Cookies()
	header := http.Header{}
	header.Add("Cookie", cookies[0].Name+"="+cookies[0].Value)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket for user %s: %v", userID, err)
	}

	return conn
}
