package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // No origin header (e.g., native clients)
		}

		// Development: allow localhost
		if os.Getenv("ENV") != "production" {
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				return true
			}
		}

		// Production: validate against PUBLIC_URL
		publicURL := os.Getenv("PUBLIC_URL")
		if publicURL == "" {
			log.Printf("[WebSocket] WARNING: PUBLIC_URL not set, allowing all origins")
			return true
		}

		parsedPublic, err := url.Parse(publicURL)
		if err != nil {
			log.Printf("[WebSocket] ERROR: Invalid PUBLIC_URL: %v", err)
			return false
		}

		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			log.Printf("[WebSocket] ERROR: Invalid Origin: %v", err)
			return false
		}

		// Compare hostnames
		allowed := parsedOrigin.Hostname() == parsedPublic.Hostname()
		if !allowed {
			log.Printf("[WebSocket] REJECTED: Origin %s does not match PUBLIC_URL %s", origin, publicURL)
		}
		return allowed
	},
}

// WebSocketHandler handles WebSocket connection upgrades
type WebSocketHandler struct {
	hub   *realtime.Hub
	store *sessions.CookieStore
	db    *gorm.DB
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *realtime.Hub, store *sessions.CookieStore, db *gorm.DB) *WebSocketHandler {
	return &WebSocketHandler{
		hub:   hub,
		store: store,
		db:    db,
	}
}

// HandleConnection upgrades HTTP to WebSocket and manages the connection lifecycle
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract sessionId from URL path
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	if sessionID == "" {
		log.Printf("WebSocket upgrade failed: missing sessionId")
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Authenticate via cookie-session (reuse REST auth)
	session, err := h.store.Get(r, "boeuf-session")
	if err != nil {
		log.Printf("WebSocket auth failed: invalid session cookie: %v", err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID, ok := session.Values["spotify_user_id"].(string)
	if !ok || userID == "" {
		log.Printf("WebSocket auth failed: no spotify_user_id in session")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify user is participant of this session
	var participant models.SessionParticipant
	result := h.db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("WebSocket forbidden: user %s not participant of session %s", userID, sessionID)

			// AC5: For authenticated users who are not participants, upgrade to WebSocket
			// so we can send a stable WS_FORBIDDEN message before closing.
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Printf("WebSocket upgrade error (forbidden): %v", err)
				return
			}
			h.sendForbiddenAndClose(conn, sessionID)
			return
		}
		log.Printf("WebSocket error checking participant: %v", result.Error)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client and register with hub
	client := realtime.NewClient(h.hub, conn, sessionID, userID, participant.Role)

	// Register client (will trigger PARTICIPANT_JOINED broadcast)
	h.hub.Register <- client

	// Send initial snapshot to this client
	if err := h.sendInitialSnapshot(client); err != nil {
		log.Printf("Failed to send initial snapshot: %v", err)
		conn.Close()
		return
	}

	// Broadcast PARTICIPANT_JOINED to other clients (exclude the new client)
	h.broadcastParticipantJoined(sessionID, userID, participant.Role, client.UserID)

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()

	log.Printf("WebSocket connection established: session=%s, user=%s, role=%s", sessionID, userID, participant.Role)
}

// sendInitialSnapshot sends the SESSION_SNAPSHOT message to a newly connected client
func (h *WebSocketHandler) sendInitialSnapshot(client *realtime.Client) error {
	// Get all participants in the session
	var participants []models.SessionParticipant
	if err := h.db.Where("session_id = ?", client.SessionID).Find(&participants).Error; err != nil {
		return err
	}

	// Build participants list with connection status
	connectedClients := h.hub.GetSessionClients(client.SessionID)
	connectedUserIDs := make(map[string]bool)
	for _, c := range connectedClients {
		connectedUserIDs[c.UserID] = true
	}
	// Ensure the newly connected client is marked online in its own snapshot.
	// This avoids relying on timing between hub registration and snapshot sending.
	connectedUserIDs[client.UserID] = true

	participantInfos := make([]realtime.ParticipantInfo, len(participants))
	for i, p := range participants {
		status := "offline"
		if connectedUserIDs[p.UserID] {
			status = "online"
		}

		participantInfos[i] = realtime.ParticipantInfo{
			UserID:           p.UserID,
			DisplayName:      p.DisplayName,
			Role:             p.Role,
			SyncState:        p.SyncState,
			LastSeenAt:       time.Unix(p.LastSeenAt, 0).UTC().Format(time.RFC3339),
			ConnectionStatus: status,
		}
	}

	// Best-effort: load baseline now playing from session.
	var nowPlaying *realtime.NowPlayingInfo
	var sess models.Session
	if err := h.db.Where("id = ?", client.SessionID).First(&sess).Error; err == nil {
		if sess.BaselineCapturedAt != nil && sess.BaselineTrackID != "" {
			nowPlaying = &realtime.NowPlayingInfo{
				TrackID:    sess.BaselineTrackID,
				TrackName:  sess.BaselineTrackName,
				Artist:     sess.BaselineArtist,
				Album:      "", // Album not stored separately in baseline
				DurationMs: sess.BaselineDurationMs,
				ImageURL:   sess.BaselineImageURL,
				IsPlaying:  sess.BaselineIsPlaying,
				PositionMs: sess.BaselinePositionMs,
			}
		}
	}

	// Build snapshot payload (nowPlaying is optional)
	payload := realtime.SessionSnapshotPayload{
		Participants: participantInfos,
		NowPlaying:   nowPlaying,
	}

	// Create message with next event sequence
	eventSeq := h.hub.GetNextEventSeq(client.SessionID)
	msg, err := realtime.NewMessage(realtime.TypeSessionSnapshot, client.SessionID, eventSeq, payload)
	if err != nil {
		return err
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		return err
	}

	// Send directly to this client
	select {
	case client.Send <- msgBytes:
		return nil
	default:
		return errors.New("websocket send buffer full while sending initial snapshot")
	}
}

// broadcastParticipantJoined sends PARTICIPANT_JOINED event to all clients in session
func (h *WebSocketHandler) broadcastParticipantJoined(sessionID, userID, role, excludeUserID string) {
	payload := realtime.ParticipantJoinedPayload{
		UserID:    userID,
		Role:      role,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	eventSeq := h.hub.GetNextEventSeq(sessionID)
	msg, err := realtime.NewMessage(realtime.TypeParticipantJoined, sessionID, eventSeq, payload)
	if err != nil {
		log.Printf("Failed to create PARTICIPANT_JOINED message: %v", err)
		return
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		log.Printf("Failed to marshal PARTICIPANT_JOINED message: %v", err)
		return
	}

	h.hub.Broadcast <- &realtime.BroadcastMessage{
		SessionID:   sessionID,
		Message:     msgBytes,
		ExcludeUser: excludeUserID,
	}
}

// BroadcastParticipantLeft sends PARTICIPANT_LEFT event (called when client disconnects)
// This will be called by Hub when unregister happens
func (h *WebSocketHandler) BroadcastParticipantLeft(sessionID, userID string) {
	payload := realtime.ParticipantLeftPayload{
		UserID:    userID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	eventSeq := h.hub.GetNextEventSeq(sessionID)
	msg, err := realtime.NewMessage(realtime.TypeParticipantLeft, sessionID, eventSeq, payload)
	if err != nil {
		log.Printf("Failed to create PARTICIPANT_LEFT message: %v", err)
		return
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		log.Printf("Failed to marshal PARTICIPANT_LEFT message: %v", err)
		return
	}

	h.hub.Broadcast <- &realtime.BroadcastMessage{
		SessionID: sessionID,
		Message:   msgBytes,
	}
}

// ErrorResponse sends an error message before closing connection
func (h *WebSocketHandler) sendForbiddenAndClose(conn *websocket.Conn, sessionID string) {
	payload := realtime.ErrorPayload{
		Code:    "WS_FORBIDDEN",
		Message: "Not a participant",
	}

	msg, err := realtime.NewMessage(realtime.TypeWSForbidden, sessionID, 0, payload)
	if err == nil {
		if msgBytes, err2 := msg.Marshal(); err2 == nil {
			_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			_ = conn.WriteMessage(websocket.TextMessage, msgBytes)
		}
	}

	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "WS_FORBIDDEN"),
		time.Now().Add(2*time.Second),
	)
	_ = conn.Close()
}
