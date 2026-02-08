package realtime

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of WebSocket message (UPPER_SNAKE convention)
type MessageType string

const (
	TypeSessionSnapshot   MessageType = "SESSION_SNAPSHOT"
	TypeParticipantJoined MessageType = "PARTICIPANT_JOINED"
	TypeParticipantLeft   MessageType = "PARTICIPANT_LEFT"
	TypeWSError           MessageType = "WS_ERROR"
	TypeWSForbidden       MessageType = "WS_FORBIDDEN"
)

// Message represents the standard WebSocket message envelope
// All server->client messages use this format for consistency
type Message struct {
	Type      MessageType     `json:"type"`      // UPPER_SNAKE message type
	SessionID string          `json:"sessionId"` // Session this message belongs to
	EventSeq  int64           `json:"eventSeq"`  // Monotonic sequence number per session
	SentAt    string          `json:"sentAt"`    // RFC3339 UTC timestamp
	Payload   json.RawMessage `json:"payload"`   // Type-specific payload
}

// SessionSnapshotPayload contains the initial state sent on connection
type SessionSnapshotPayload struct {
	Participants []ParticipantInfo `json:"participants"`
	NowPlaying   *NowPlayingInfo   `json:"nowPlaying,omitempty"` // nil if nothing playing
}

// ParticipantInfo represents a session participant's status
type ParticipantInfo struct {
	UserID           string `json:"userId"`
	Role             string `json:"role"`             // "host" or "participant"
	LastSeenAt       string `json:"lastSeenAt"`       // RFC3339 UTC
	ConnectionStatus string `json:"connectionStatus"` // "online" or "offline"
}

// NowPlayingInfo represents the current playback state
type NowPlayingInfo struct {
	TrackID    string `json:"trackId"` // Spotify track URI
	TrackName  string `json:"trackName"`
	Artist     string `json:"artist"`
	IsPlaying  bool   `json:"isPlaying"`
	PositionMs int64  `json:"positionMs"` // Current position in milliseconds
}

// ParticipantJoinedPayload for PARTICIPANT_JOINED events
type ParticipantJoinedPayload struct {
	UserID    string `json:"userId"`
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"` // RFC3339 UTC
}

// ParticipantLeftPayload for PARTICIPANT_LEFT events
type ParticipantLeftPayload struct {
	UserID    string `json:"userId"`
	Timestamp string `json:"timestamp"` // RFC3339 UTC
}

// ErrorPayload for error messages
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewMessage creates a new WebSocket message with standard envelope
func NewMessage(msgType MessageType, sessionID string, eventSeq int64, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      msgType,
		SessionID: sessionID,
		EventSeq:  eventSeq,
		SentAt:    time.Now().UTC().Format(time.RFC3339),
		Payload:   payloadBytes,
	}, nil
}

// Marshal converts the message to JSON
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
