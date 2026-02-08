package realtime

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test Hub creation
func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub.sessions == nil {
		t.Error("Hub sessions map not initialized")
	}

	if hub.eventSeqs == nil {
		t.Error("Hub eventSeqs map not initialized")
	}

	if hub.Register == nil {
		t.Error("Hub Register channel not initialized")
	}

	if hub.Unregister == nil {
		t.Error("Hub Unregister channel not initialized")
	}

	if hub.Broadcast == nil {
		t.Error("Hub Broadcast channel not initialized")
	}
}

// Test GetNextEventSeq is monotonic
func TestGetNextEventSeq(t *testing.T) {
	hub := NewHub()
	sessionID := "sess_test"

	// First call should return 0
	seq1 := hub.GetNextEventSeq(sessionID)
	if seq1 != 0 {
		t.Errorf("First eventSeq should be 0, got %d", seq1)
	}

	// Second call should return 1
	seq2 := hub.GetNextEventSeq(sessionID)
	if seq2 != 1 {
		t.Errorf("Second eventSeq should be 1, got %d", seq2)
	}

	// Third call should return 2
	seq3 := hub.GetNextEventSeq(sessionID)
	if seq3 != 2 {
		t.Errorf("Third eventSeq should be 2, got %d", seq3)
	}

	// Different session should start at 0
	seq4 := hub.GetNextEventSeq("sess_other")
	if seq4 != 0 {
		t.Errorf("New session eventSeq should be 0, got %d", seq4)
	}
}

// Test client registration
func TestClientRegistration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create mock client (without real websocket connection)
	client := &Client{
		hub:       hub,
		conn:      nil, // mock
		Send:      make(chan []byte, 256),
		SessionID: "sess_123",
		UserID:    "user_1",
		Role:      "host",
	}

	// Register client
	hub.Register <- client

	// Give hub time to process
	time.Sleep(10 * time.Millisecond)

	// Verify client is in hub
	clients := hub.GetSessionClients("sess_123")
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}

	if clients[0].UserID != "user_1" {
		t.Errorf("Expected user_1, got %s", clients[0].UserID)
	}
}

// Test multiple clients in same session
func TestMultipleClientsInSession(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sessionID := "sess_123"

	// Create multiple clients
	client1 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_1",
		Role:      "host",
	}

	client2 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_2",
		Role:      "participant",
	}

	// Register both clients
	hub.Register <- client1
	hub.Register <- client2

	time.Sleep(10 * time.Millisecond)

	// Verify both clients are registered
	clients := hub.GetSessionClients(sessionID)
	if len(clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(clients))
	}
}

// Test client unregistration
func TestClientUnregistration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: "sess_123",
		UserID:    "user_1",
		Role:      "host",
	}

	// Register then unregister
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Unregister <- client
	time.Sleep(10 * time.Millisecond)

	// Verify client is removed
	clients := hub.GetSessionClients("sess_123")
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients after unregister, got %d", len(clients))
	}
}

// Test broadcast to session
func TestBroadcastToSession(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sessionID := "sess_123"

	// Create two clients
	client1 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_1",
		Role:      "host",
	}

	client2 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_2",
		Role:      "participant",
	}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	testMessage := []byte(`{"type":"TEST"}`)
	hub.Broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   testMessage,
	}

	time.Sleep(10 * time.Millisecond)

	// Both clients should receive the message
	select {
	case msg := <-client1.Send:
		if string(msg) != string(testMessage) {
			t.Errorf("Client1 received wrong message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client1 did not receive message")
	}

	select {
	case msg := <-client2.Send:
		if string(msg) != string(testMessage) {
			t.Errorf("Client2 received wrong message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client2 did not receive message")
	}
}

// Test broadcast with exclude user
func TestBroadcastExcludeUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sessionID := "sess_123"

	client1 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_1",
		Role:      "host",
	}

	client2 := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_2",
		Role:      "participant",
	}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast excluding user_1
	testMessage := []byte(`{"type":"TEST"}`)
	hub.Broadcast <- &BroadcastMessage{
		SessionID:   sessionID,
		Message:     testMessage,
		ExcludeUser: "user_1",
	}

	time.Sleep(10 * time.Millisecond)

	// Client1 should NOT receive message
	select {
	case <-client1.Send:
		t.Error("Client1 should not have received message (was excluded)")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message for excluded user
	}

	// Client2 SHOULD receive message
	select {
	case msg := <-client2.Send:
		if string(msg) != string(testMessage) {
			t.Errorf("Client2 received wrong message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client2 did not receive message")
	}
}

// Test disconnect callback
func TestDisconnectCallback(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	callbackCalled := false
	var callbackSessionID, callbackUserID string

	hub.SetOnClientDisconnect(func(sessionID, userID string) {
		callbackCalled = true
		callbackSessionID = sessionID
		callbackUserID = userID
	})

	client := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: "sess_123",
		UserID:    "user_1",
		Role:      "host",
	}

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Unregister <- client
	time.Sleep(50 * time.Millisecond) // Callback is called in goroutine

	if !callbackCalled {
		t.Error("Disconnect callback was not called")
	}

	if callbackSessionID != "sess_123" {
		t.Errorf("Expected sessionID sess_123, got %s", callbackSessionID)
	}

	if callbackUserID != "user_1" {
		t.Errorf("Expected userID user_1, got %s", callbackUserID)
	}
}

// Test session cleanup when last client leaves
func TestSessionCleanup(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sessionID := "sess_123"

	client := &Client{
		hub:       hub,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    "user_1",
		Role:      "host",
	}

	// Register and unregister
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Event seq should exist
	seq1 := hub.GetNextEventSeq(sessionID)
	if seq1 != 0 {
		t.Errorf("Expected seq 0, got %d", seq1)
	}

	hub.Unregister <- client
	time.Sleep(10 * time.Millisecond)

	// After cleanup, session should start fresh
	seq2 := hub.GetNextEventSeq(sessionID)
	if seq2 != 0 {
		t.Errorf("After cleanup, seq should reset to 0, got %d", seq2)
	}
}

// Benchmark event sequence generation
func BenchmarkGetNextEventSeq(b *testing.B) {
	hub := NewHub()
	sessionID := "sess_bench"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.GetNextEventSeq(sessionID)
	}
}

// Test WebSocket client creation
func TestNewClient(t *testing.T) {
	hub := NewHub()

	// Create a mock websocket connection (not used in this test)
	var mockConn *websocket.Conn = nil

	client := NewClient(hub, mockConn, "sess_123", "user_1", "host")

	if client.SessionID != "sess_123" {
		t.Errorf("Expected SessionID sess_123, got %s", client.SessionID)
	}

	if client.UserID != "user_1" {
		t.Errorf("Expected UserID user_1, got %s", client.UserID)
	}

	if client.Role != "host" {
		t.Errorf("Expected Role host, got %s", client.Role)
	}

	if client.Send == nil {
		t.Error("Send channel should be initialized")
	}

	if client.hub != hub {
		t.Error("Client hub reference is incorrect")
	}
}

// Test timeout detection mechanism (NFR8: ≤ 5 seconds) - Simulation
func TestTimeoutConfiguration(t *testing.T) {
	// Verify timeout constants are set correctly for ≤ 5s detection
	if pongWait > 5*time.Second {
		t.Errorf("pongWait %v exceeds 5 seconds NFR8 requirement", pongWait)
	}

	if pingPeriod >= pongWait {
		t.Errorf("pingPeriod %v should be less than pongWait %v", pingPeriod, pongWait)
	}

	// Verify ping/pong intervals are reasonable
	expectedPingPeriod := (pongWait * 9) / 10
	if pingPeriod != expectedPingPeriod {
		t.Errorf("Expected pingPeriod %v, got %v", expectedPingPeriod, pingPeriod)
	}

	t.Logf("Timeout configuration valid: pongWait=%v, pingPeriod=%v", pongWait, pingPeriod)
}
