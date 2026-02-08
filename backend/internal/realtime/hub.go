package realtime

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket configuration
	writeWait      = 10 * time.Second
	pongWait       = 5 * time.Second // Must detect disconnection in ≤ 5 seconds (NFR8)
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB max message size
)

// Client represents a WebSocket client connection
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	Send      chan []byte
	SessionID string
	UserID    string
	Role      string // "host" or "participant"
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, sessionID, userID, role string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    userID,
		Role:      role,
	}
}

// readPump pumps messages from the WebSocket connection to the hub
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error (session=%s, user=%s): %v", c.SessionID, c.UserID, err)
			}
			break
		}
		// For MVP, we don't process client messages (future: client actions)
		// Just keep connection alive and detect disconnections
	}
}

// writePump pumps messages from the hub to the WebSocket connection
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients per session
	// Map: sessionID -> map[userID]Client
	sessions map[string]map[string]*Client

	// Event sequence counters per session
	// Map: sessionID -> eventSeq
	eventSeqs map[string]int64

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Broadcast messages to all clients in a session
	Broadcast chan *BroadcastMessage

	// Callback when client disconnects (for PARTICIPANT_LEFT broadcast)
	onClientDisconnect func(sessionID, userID string)

	// Mutex for thread-safe access
	mu sync.RWMutex
}

// BroadcastMessage represents a message to broadcast to a session
type BroadcastMessage struct {
	SessionID   string
	Message     []byte
	ExcludeUser string // Optional: exclude this user from broadcast
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		sessions:   make(map[string]map[string]*Client),
		eventSeqs:  make(map[string]int64),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *BroadcastMessage),
	}
}

// SetOnClientDisconnect sets the callback function for client disconnections
func (h *Hub) SetOnClientDisconnect(callback func(sessionID, userID string)) {
	h.onClientDisconnect = callback
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastToSession(message)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Initialize session map if it doesn't exist
	if h.sessions[client.SessionID] == nil {
		h.sessions[client.SessionID] = make(map[string]*Client)
	}

	// Add client to session
	h.sessions[client.SessionID][client.UserID] = client

	// Log registration (reduced verbosity for production)
	log.Printf("WebSocket client registered: session=%s, user=%s", client.SessionID, client.UserID)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.sessions[client.SessionID]; ok {
		if _, ok := clients[client.UserID]; ok {
			delete(clients, client.UserID)
			close(client.Send)

			sessionID := client.SessionID
			userID := client.UserID

			// Clean up empty session
			if len(clients) == 0 {
				delete(h.sessions, client.SessionID)
				delete(h.eventSeqs, client.SessionID)
			}

			log.Printf("Client unregistered: session=%s, user=%s, remaining=%d",
				client.SessionID, client.UserID, len(clients))

			// Call disconnect callback if registered (for PARTICIPANT_LEFT broadcast)
			if h.onClientDisconnect != nil {
				go h.onClientDisconnect(sessionID, userID)
			}
		}
	}
}

// broadcastToSession sends a message to all clients in a session
func (h *Hub) broadcastToSession(msg *BroadcastMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.sessions[msg.SessionID]
	if !ok {
		return
	}

	for userID, client := range clients {
		// Skip excluded user (if specified)
		if msg.ExcludeUser != "" && userID == msg.ExcludeUser {
			continue
		}

		select {
		case client.Send <- msg.Message:
		default:
			// Client buffer is full; drop it from the hub.
			close(client.Send)
			delete(clients, userID)
		}
	}
}

// GetNextEventSeq returns and increments the event sequence for a session
func (h *Hub) GetNextEventSeq(sessionID string) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	seq := h.eventSeqs[sessionID]
	h.eventSeqs[sessionID] = seq + 1
	return seq
}

// GetSessionClients returns the list of connected clients for a session
func (h *Hub) GetSessionClients(sessionID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.sessions[sessionID]
	if !ok {
		return nil
	}

	result := make([]*Client, 0, len(clients))
	for _, client := range clients {
		result = append(result, client)
	}
	return result
}
