package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/spotify"
)

const debugEventLimit = 200

type DebugParticipant struct {
	UserID           string `json:"userId"`
	SpotifyUserID    string `json:"spotifyUserId"`
	DisplayName      string `json:"displayName"`
	Role             string `json:"role"`
	JoinedAt         string `json:"joinedAt"`
	LastSeenAt       string `json:"lastSeenAt"`
	SyncState        string `json:"syncState"`
	ConnectionStatus string `json:"connectionStatus"`
}

type DebugSpotifyState struct {
	UserID     string `json:"userId"`
	Accessible bool   `json:"accessible"`
	Error      string `json:"error,omitempty"`
	IsPlaying  bool   `json:"isPlaying"`
	TrackID    string `json:"trackId,omitempty"`
	TrackName  string `json:"trackName,omitempty"`
	Artist     string `json:"artist,omitempty"`
	PositionMs int64  `json:"positionMs"`
	DurationMs int64  `json:"durationMs"`
	ImageURL   string `json:"imageUrl,omitempty"`
}

type DebugEvent struct {
	EventSeq    int64                  `json:"eventSeq"`
	EventType   string                 `json:"eventType"`
	CreatedAt   string                 `json:"createdAt"`
	ActorUserID string                 `json:"actorUserId,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
}

type DebugSessionResponse struct {
	SessionID        string              `json:"sessionId"`
	GeneratedAt      string              `json:"generatedAt"`
	Participants     []DebugParticipant  `json:"participants"`
	SpotifyStates    []DebugSpotifyState `json:"spotifyStates"`
	ActionLog        []DebugEvent        `json:"actionLog"`
	SpotifyChangeLog []DebugEvent        `json:"spotifyChangeLog"`
}

// DebugSession returns a debug snapshot for a session
// GET /api/sessions/:sessionId/debug
func (h *SessionHandler) DebugSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	if sessionID == "" {
		RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "sessionId is required")
		return
	}

	var participants []models.SessionParticipant
	if err := h.db.Where("session_id = ?", sessionID).Order("joined_at ASC").Find(&participants).Error; err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load participants")
		return
	}

	connectedUserIDs := map[string]bool{}
	if h.realtimeHub != nil {
		for _, client := range h.realtimeHub.GetSessionClients(sessionID) {
			connectedUserIDs[client.UserID] = true
		}
	}

	debugParticipants := make([]DebugParticipant, 0, len(participants))
	for _, participant := range participants {
		status := "offline"
		if connectedUserIDs[participant.UserID] {
			status = "online"
		}

		debugParticipants = append(debugParticipants, DebugParticipant{
			UserID:           participant.UserID,
			SpotifyUserID:    participant.UserID,
			DisplayName:      participant.DisplayName,
			Role:             participant.Role,
			JoinedAt:         participant.JoinedAt.UTC().Format(time.RFC3339),
			LastSeenAt:       time.Unix(participant.LastSeenAt, 0).UTC().Format(time.RFC3339),
			SyncState:        participant.SyncState,
			ConnectionStatus: status,
		})
	}

	spotifyStates := make([]DebugSpotifyState, 0, len(participants))
	for _, participant := range participants {
		state := DebugSpotifyState{UserID: participant.UserID}

		if h.spotifyClient == nil {
			state.Accessible = false
			state.Error = "SPOTIFY_CLIENT_NOT_CONFIGURED"
			spotifyStates = append(spotifyStates, state)
			continue
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		playerState, err := h.spotifyClient.GetPlayerState(ctx, participant.UserID)
		cancel()

		if err != nil {
			state.Accessible = false
			state.Error = mapSpotifyError(err)
			spotifyStates = append(spotifyStates, state)
			continue
		}

		state.Accessible = true
		state.IsPlaying = playerState.IsPlaying
		state.PositionMs = playerState.PositionMs
		state.DurationMs = playerState.DurationMs
		state.TrackID = playerState.TrackID
		state.TrackName = playerState.TrackName
		state.Artist = playerState.Artist
		state.ImageURL = playerState.ImageURL
		spotifyStates = append(spotifyStates, state)
	}

	var events []models.Event
	if err := h.db.Where("session_id = ?", sessionID).Order("event_seq DESC").Limit(debugEventLimit).Find(&events).Error; err != nil {
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load events")
		return
	}

	actionLog := make([]DebugEvent, 0)
	spotifyChangeLog := make([]DebugEvent, 0)
	for _, event := range events {
		payload := map[string]interface{}{}
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			log.Printf("DEBUG: Failed to parse event payload: %v", err)
		}

		source := ""
		if src, ok := payload["source"].(string); ok {
			source = src
		}

		debugEvent := DebugEvent{
			EventSeq:  event.EventSeq,
			EventType: event.EventType,
			CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
			Payload:   payload,
		}

		if userID, ok := payload["userId"].(string); ok {
			debugEvent.ActorUserID = userID
		}

		if isActionEvent(event.EventType) && source != "poller" {
			actionLog = append(actionLog, debugEvent)
		}

		if isSpotifyChangeEvent(event.EventType) && source == "poller" {
			spotifyChangeLog = append(spotifyChangeLog, debugEvent)
		}
	}

	RespondJSON(w, http.StatusOK, DebugSessionResponse{
		SessionID:        sessionID,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Participants:     debugParticipants,
		SpotifyStates:    spotifyStates,
		ActionLog:        actionLog,
		SpotifyChangeLog: spotifyChangeLog,
	})
}

func isActionEvent(eventType string) bool {
	switch eventType {
	case "PLAYER_PAUSED", "PLAYER_RESUMED", "PLAYER_SEEKED", "TRACK_CHANGED":
		return true
	default:
		return false
	}
}

func isSpotifyChangeEvent(eventType string) bool {
	switch eventType {
	case "PLAYER_STATE_UPDATE", "TRACK_CHANGED":
		return true
	default:
		return false
	}
}

func mapSpotifyError(err error) string {
	var rateLimited *spotify.RateLimitedError
	switch {
	case errors.As(err, &rateLimited):
		return "SPOTIFY_RATE_LIMITED"
	case errors.Is(err, spotify.ErrSpotifyNotConnected):
		return "SPOTIFY_NOT_CONNECTED"
	default:
		return "SPOTIFY_UNAVAILABLE"
	}
}
