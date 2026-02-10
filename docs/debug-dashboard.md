# Debug Dashboard

## Overview
The debug dashboard provides a per-session live snapshot that combines participant presence, Spotify playback state, and a timeline of detected actions and polling changes.

Route (frontend): `/session/:sessionId/debug`

API (backend): `GET /api/sessions/{sessionId}/debug`

Access: any authenticated participant (no host restriction).

## What the dashboard shows

### Participants table
For each session participant:

- Display name and Spotify user id
- Role (host or participant)
- Sync state (ready or synced)
- Connection status (online/offline from realtime hub)
- Spotify state (accessible/playing/paused)
- Track name and current position
- Joined at and last seen timestamp

### Action log (boeuf detected)
Chronological events triggered by user actions through the app:

- `PLAYER_PAUSED`
- `PLAYER_RESUMED`
- `PLAYER_SEEKED`
- `TRACK_CHANGED`

Events come from the player handlers and are persisted in the `events` table.

### Spotify change log (poller detected)
Playback changes detected by the Spotify polling loop:

- `PLAYER_STATE_UPDATE`
- `TRACK_CHANGED`

Poller events are tagged with `source: "poller"` and `polledUserId` in the payload.

## Backend data sources

### Session participants
- Table: `session_participants`
- Fields used: `user_id`, `display_name`, `role`, `sync_state`, `joined_at`, `last_seen_at`

### Connection status
- Realtime hub clients: `realtime.Hub.GetSessionClients(sessionID)`
- If user id is present in the hub, status is `online`, otherwise `offline`

### Spotify state per user
- Each participant is queried via `spotify.Client.GetPlayerState`
- The response is mapped into `DebugSpotifyState`
- If Spotify is unreachable or not connected, `accessible` is false and `error` is set

### Event logs
- Table: `events`
- Action log = action events without `source: poller`
- Spotify change log = polling events with `source: poller`
- Limit: 200 events (most recent by `event_seq`)

## Frontend behavior

### Refresh
- Automatic refresh every 5 seconds
- Manual refresh button

### Mapping
- Spotify state is matched by `userId`
- Event lists are rendered as simple chronological cards

## Operational notes

- `GET /api/sessions/{sessionId}/debug` is protected by `RequireParticipant`
- The poller only tracks sessions with active synced participants
- Poller events are persisted on every state change
- The debug endpoint queries Spotify for every participant; this can trigger rate limiting with large sessions

## Files

Backend:
- `backend/internal/handlers/debug.go`
- `backend/internal/session/polling.go`
- `backend/internal/handlers/player.go`

Frontend:
- `frontend/src/views/SessionDebugView.vue`
- `frontend/src/router/index.ts`
