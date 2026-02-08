package realtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Test NewMessage creates a valid message envelope
func TestNewMessage(t *testing.T) {
	payload := map[string]string{"test": "data"}
	msg, err := NewMessage(TypeSessionSnapshot, "sess_123", 42, payload)

	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}

	if msg.Type != TypeSessionSnapshot {
		t.Errorf("Expected type %s, got %s", TypeSessionSnapshot, msg.Type)
	}

	if msg.SessionID != "sess_123" {
		t.Errorf("Expected sessionId sess_123, got %s", msg.SessionID)
	}

	if msg.EventSeq != 42 {
		t.Errorf("Expected eventSeq 42, got %d", msg.EventSeq)
	}

	if msg.SentAt == "" {
		t.Error("Expected sentAt to be set")
	}

	// Validate RFC3339 timestamp
	_, err = time.Parse(time.RFC3339, msg.SentAt)
	if err != nil {
		t.Errorf("SentAt is not valid RFC3339: %v", err)
	}

	// Verify payload can be unmarshaled
	var payloadData map[string]string
	if err := json.Unmarshal(msg.Payload, &payloadData); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payloadData["test"] != "data" {
		t.Errorf("Payload data mismatch: got %v", payloadData)
	}
}

// Test message marshaling
func TestMessageMarshal(t *testing.T) {
	payload := ParticipantJoinedPayload{
		UserID:    "user_123",
		Role:      "host",
		Timestamp: "2026-01-28T12:00:00Z",
	}

	msg, err := NewMessage(TypeParticipantJoined, "sess_123", 1, payload)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal and verify structure
	var unmarshaled Message
	if err := json.Unmarshal(msgBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if unmarshaled.Type != TypeParticipantJoined {
		t.Errorf("Type mismatch after marshal/unmarshal")
	}

	if unmarshaled.SessionID != "sess_123" {
		t.Errorf("SessionID mismatch after marshal/unmarshal")
	}

	if unmarshaled.EventSeq != 1 {
		t.Errorf("EventSeq mismatch after marshal/unmarshal")
	}
}

// Test all message types are in UPPER_SNAKE format
func TestMessageTypeFormat(t *testing.T) {
	types := []MessageType{
		TypeSessionSnapshot,
		TypeParticipantJoined,
		TypeParticipantLeft,
		TypeWSError,
		TypeWSForbidden,
	}

	for _, msgType := range types {
		typeStr := string(msgType)

		// Check it's uppercase
		if strings.ToUpper(typeStr) != typeStr {
			t.Errorf("Message type %s is not UPPER_SNAKE", msgType)
		}

		// Check it contains underscores (UPPER_SNAKE convention)
		hasUnderscore := false
		for _, char := range typeStr {
			if char == '_' {
				hasUnderscore = true
				break
			}
		}

		if !hasUnderscore {
			t.Errorf("Message type %s should contain underscores (UPPER_SNAKE)", msgType)
		}
	}
}

// Test SessionSnapshotPayload structure
func TestSessionSnapshotPayload(t *testing.T) {
	participants := []ParticipantInfo{
		{
			UserID:           "user_1",
			Role:             "host",
			LastSeenAt:       "2026-01-28T12:00:00Z",
			ConnectionStatus: "online",
		},
		{
			UserID:           "user_2",
			Role:             "participant",
			LastSeenAt:       "2026-01-28T11:55:00Z",
			ConnectionStatus: "offline",
		},
	}

	nowPlaying := &NowPlayingInfo{
		TrackID:    "spotify:track:123",
		TrackName:  "Test Song",
		Artist:     "Test Artist",
		IsPlaying:  true,
		PositionMs: 30000,
	}

	payload := SessionSnapshotPayload{
		Participants: participants,
		NowPlaying:   nowPlaying,
	}

	msg, err := NewMessage(TypeSessionSnapshot, "sess_123", 0, payload)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Unmarshal payload
	var parsedPayload SessionSnapshotPayload
	if err := json.Unmarshal(msg.Payload, &parsedPayload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if len(parsedPayload.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(parsedPayload.Participants))
	}

	if parsedPayload.NowPlaying.TrackName != "Test Song" {
		t.Errorf("NowPlaying mismatch")
	}
}

// Test error payload structure
func TestErrorPayload(t *testing.T) {
	errorPayload := ErrorPayload{
		Code:    "WS_FORBIDDEN",
		Message: "Not a participant",
		Details: "User is not part of this session",
	}

	msg, err := NewMessage(TypeWSError, "", 0, errorPayload)
	if err != nil {
		t.Fatalf("Failed to create error message: %v", err)
	}

	var parsed ErrorPayload
	if err := json.Unmarshal(msg.Payload, &parsed); err != nil {
		t.Fatalf("Failed to parse error payload: %v", err)
	}

	if parsed.Code != "WS_FORBIDDEN" {
		t.Errorf("Error code mismatch")
	}
}
