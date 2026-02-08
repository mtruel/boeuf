# Story 1.10: Persistent Room Model - Session Lifecycle Management

Status: ready

## Story

As a utilisateur,
I want créer des "rooms" persistantes que je peux rejoindre/quitter à volonté,
So that je n'ai pas besoin de recréer des sessions à chaque utilisation et je peux gérer mes espaces d'écoute comme des salles permanentes.

## Context

**Paradigm Shift:** Les sessions passent d'un modèle jetable (create → use once → auto-cleanup) à un modèle de **rooms persistantes** (create once → join/leave repeatedly → manual archive/delete only).

Ce changement transforme le produit de "sessions éphémères style Spotify Group Session" vers "rooms permanentes style Discord channels", offrant une expérience plus fluide et moins de friction pour les utilisateurs récurrents.

**Architecture Clarification: Room Owner vs Sync Leader**

Cette story introduit deux concepts distincts:

1. **Room Owner (`created_by`)** - User-facing concept
   - L'utilisateur qui a créé la room
   - Droits: supprimer la room, gérer les paramètres
   - Permanent (ne change jamais)
   - Visible dans l'UI: "Created by {userName}"

2. **Sync Leader (`sync_leader_user_id`)** - Internal technical mechanism
   - La source de vérité pour la synchronisation playback
   - Le serveur poll le Spotify de CET utilisateur
   - Change automatiquement (l'utilisateur qui agit devient sync leader)
   - **TRANSPARENT pour l'utilisateur** - jamais affiché dans l'UI
   - Null si room STALE (0 participants actifs)
   - Implementé dans Epic 3 (gouvernance), pas dans cette story

**Important:** Story 1.10 implémente uniquement `created_by` (room owner). Le champ `sync_leader_user_id` sera ajouté plus tard dans Epic 3 quand la logique de synchronisation sera finalisée.

## Acceptance Criteria

### AC1: Create Room with Required Name

- **Given** un utilisateur authentifié veut créer une room
- **When** il accède au formulaire de création
- **Then** un champ "Room Name" est requis avec:
  - Placeholder/default: "{UserName}'s Room" (ex: "Mathias's Room")
  - Validation: 3-100 caractères
  - Validation: pas uniquement des espaces
- **When** l'utilisateur soumet la création
- **Then** le backend:
  - Crée la room avec `name`, `created_by` (user ID)
  - Génère un persistent invite link (pas d'expiration)
  - Join automatiquement le créateur dans la room
  - Retourne: room object + invite link
- **And** le frontend affiche:
  - Confirmation "Room created!"
  - Room details (nom, invite link)
  - Bouton "Copy Link" + "Regenerate Link"
  - Transition automatique vers la room (SessionLive view)

### AC2: Single Active Participation Enforcement

- **Given** un utilisateur est déjà actif dans Room A
- **When** il tente de joindre Room B
- **Then** le système:
  - **Auto-leave** Room A (set `is_active=false`, `left_at=now()` dans `session_participants`)
  - **Join** Room B (insert/update `is_active=true` dans `session_participants`)
  - Broadcast `USER_LEFT` à Room A
  - Broadcast `USER_JOINED` à Room B
- **And** l'UI du user reflète instantanément le changement
- **And** les autres participants de Room A voient le départ
- **Validation:** Un user ne peut JAMAIS être `is_active=true` dans 2+ rooms simultanément

### AC3: Join/Leave Endpoints

- **Endpoint:** `POST /api/sessions/:id/join`
  - Body: `{ inviteCode: string }` (optionnel si déjà participant)
  - Auth: required (cookie session)
  - Logic:
    1. Valider que la room existe et `active=true`
    2. Valider invite code si pas déjà participant
    3. Auto-leave current active room (AC2)
    4. Join target room (upsert `session_participants` avec `is_active=true`, `left_at=null`)
    5. Update room `last_activity_at`
    6. Broadcast `USER_JOINED` event
  - Response: `200 + { session, participants[] }`
  - Errors:
    - `404` si room n'existe pas
    - `403` si invite code invalide
    - `410` si room archivée (`active=false`)

- **Endpoint:** `POST /api/sessions/:id/leave`
  - Auth: required
  - Logic:
    1. Valider que user est participant actif
    2. Update `session_participants`: `is_active=false`, `left_at=now()`
    3. Update room `last_activity_at`
    4. Check si room devient stale (0 participants actifs)
    5. Broadcast `USER_LEFT` event
  - Response: `200 + { session, remainingParticipants }`
  - Errors:
    - `404` si room n'existe pas
    - `403` si user pas participant

### AC4: Persistent Invite Links with Regeneration

- **Endpoint:** `GET /api/sessions/:id/invite`
  - Auth: required (participant only)
  - Response: `200 + { inviteLink, code, createdAt }`
  - Errors: `403` si non-participant, `404` si room n'existe pas

- **Endpoint:** `POST /api/sessions/:id/invite/regenerate`
  - Auth: required (creator only? ou any participant?)
  - Logic:
    1. Désactiver ancien invite: `UPDATE session_invites SET is_active=false WHERE session_id=:id`
    2. Créer nouveau invite avec nouveau code
    3. Update room `last_activity_at`
  - Response: `200 + { newInviteLink, newCode }`
  - **Security:** Ancien code retourne `403 Invite link disabled` lors de join attempt

- **Schema:** `SessionInvite` fields:
  - `is_active BOOL DEFAULT true` (NEW)
  - **REMOVE** `expires_at` (plus d'expiration temporelle)
  - Cleanup: Only delete when session deleted

### AC5: Room State Model (LIVE/STALE/ARCHIVED)

**States:**

- **LIVE:** `active=true` AND has participants with `is_active=true`
- **STALE:** `active=true` AND zero participants with `is_active=true`
- **ARCHIVED:** `active=false` (hidden from main list, manual archive only)

**State Transitions:**

```
Create Room → LIVE (creator auto-joins)
  ↓
All users leave → STALE
  ↓
User rejoins → LIVE
  ↓
Manual archive → ARCHIVED
  ↓
Manual delete → DELETED (physical removal)
```

**Computed State Logic (backend):**

```go
func (s *Session) GetState(db *gorm.DB) string {
    if !s.Active {
        return "ARCHIVED"
    }
    var count int64
    db.Model(&SessionParticipant{}).
        Where("session_id = ? AND is_active = ?", s.ID, true).
        Count(&count)
    if count > 0 {
        return "LIVE"
    }
    return "STALE"
}
```

### AC6: List User Rooms with Filtering

- **Endpoint:** `GET /api/sessions?state={LIVE|STALE|ARCHIVED}&limit=50`
  - Auth: required
  - Query params:
    - `state` (optional): filter by state
    - `limit` (optional): default 50, max 100
  - Logic:
    1. Fetch rooms where user is in `session_participants`
    2. Join avec `sessions` table
    3. Compute state for each room (LIVE/STALE/ARCHIVED)
    4. Order by `last_activity_at DESC`
  - Response: `200 + { rooms: [{ id, name, state, createdBy, lastActivityAt, participantCount, currentTrack?, isUserActive }] }`
  - **Frontend:** Afficher liste groupée par state (LIVE en haut, STALE ensuite, ARCHIVED en bas ou onglet séparé)

### AC7: Manual Archive and Delete

- **Endpoint:** `POST /api/sessions/:id/archive`
  - Auth: required (creator only? ou any participant?)
  - Logic:
    1. Set `active=false`
    2. Auto-leave tous les participants actifs (`is_active=false`, `left_at=now()`)
    3. Broadcast `SESSION_ARCHIVED` event
  - Response: `200 + { session }`
  - **UX:** Move room to "Archived" section, can't be rejoined

- **Endpoint:** `DELETE /api/sessions/:id`
  - Auth: required (creator only)
  - Logic:
    1. Hard delete session + cascade (participants, invites, events)
    2. Broadcast `SESSION_DELETED` à tous les participants connectés (force disconnect)
  - Response: `204 No Content`
  - **UX:** Permanent removal, show confirmation modal ("This action cannot be undone")

### AC8: Activity Tracking (LastActivityAt Updates)

- **Trigger events** qui update `last_activity_at`:
  - User join/leave room
  - Play/pause/seek actions
  - Track changes
  - Invite regeneration
  - Any WebSocket broadcast from room
- **Backend:** Helper function `UpdateRoomActivity(sessionID)` appelée par tous les handlers concernés
- **Purpose:** Permet de trier les rooms par activité récente dans l'UI

### AC9: Schema Migration

**Sessions table changes:**

```sql
-- Add new fields
ALTER TABLE sessions ADD COLUMN name TEXT NOT NULL DEFAULT 'Unnamed Room';
ALTER TABLE sessions ADD COLUMN created_by TEXT NOT NULL DEFAULT '';

-- Rename field (GORM will handle via AutoMigrate)
-- ExpiresAt → LastActivityAt
ALTER TABLE sessions RENAME COLUMN expires_at TO last_activity_at;

-- Note: sync_leader_user_id will be added in Epic 3 (governance/host logic)
-- This story only implements room ownership (created_by)

-- Data migration for existing sessions
UPDATE sessions 
SET 
  name = 'Legacy Session ' || substr(id, 1, 8),
  created_by = (
    SELECT user_id FROM session_participants 
    WHERE session_id = sessions.id 
    ORDER BY joined_at LIMIT 1
  ),
  last_activity_at = COALESCE(expires_at, created_at);
```

**SessionParticipant table changes:**

```sql
-- Add new fields
ALTER TABLE session_participants ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE session_participants ADD COLUMN left_at TIMESTAMP NULL;

-- Data migration: marquer tous les participants existants comme actifs
UPDATE session_participants SET is_active = true WHERE is_active IS NULL;
```

**SessionInvite table changes:**

```sql
-- Add new field
ALTER TABLE session_invites ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;

-- Remove expires_at (no longer used)
ALTER TABLE session_invites DROP COLUMN expires_at;

-- Data migration: tous les invites existants deviennent persistants
UPDATE session_invites SET is_active = true;
```

**Cleanup handler changes:**

```go
// REMOVE auto-delete logic
❌ Delete sessions after 7 days inactive

// KEEP deactivate logic? → NO, archiving is manual only
❌ Auto-archive stale sessions after X time

// Backend cleanup.go becomes minimal:
// - Delete orphaned events older than 30 days
// - Delete session_participants entries for deleted sessions (cascade)
// - NO automatic session deletion
```

## Tasks / Subtasks

### Backend: Schema & Models

- [ ] Backend: Migration fichier SQL (AC9)
  - [ ] Créer migration `006_persistent_rooms.sql`
  - [ ] ALTER sessions: add name, created_by, rename expires_at → last_activity_at
  - [ ] ALTER session_participants: add is_active, left_at
  - [ ] ALTER session_invites: add is_active, drop expires_at
  - [ ] Data migration queries pour existing data
  - [ ] Test migration sur DB fresh + avec data existante

- [ ] Backend: Update models (AC9)
  - [ ] `models/session.go`:
    - Add `Name string` (room name)
    - Add `CreatedBy string` (room owner user_id)
    - Rename `ExpiresAt` → `LastActivityAt`
    - Add method `GetState(db) string` (LIVE/STALE/ARCHIVED)
    - Add method `GetActiveParticipantCount(db) int64`
    - **Note:** `SyncLeaderUserID` field sera ajouté dans Epic 3 (pas dans cette story)
  - [ ] `models/session.go` (`SessionParticipant`):
    - Add `IsActive bool`
    - Add `LeftAt *time.Time`
  - [ ] `models/session.go` (`SessionInvite`):
    - Add `IsActive bool`
    - Remove `ExpiresAt` field
  - [ ] Valider que GORM tags sont corrects pour nouvelles colonnes
  - [ ] Tests unitaires: `models/session_test.go` pour nouvelles méthodes

### Backend: Room Lifecycle Endpoints

- [ ] Backend: Helper `UpdateRoomActivity()` (AC8)
  - [ ] Fonction `UpdateRoomActivity(db *gorm.DB, sessionID string) error`
  - [ ] Update `last_activity_at = NOW()`
  - [ ] Appelée par tous les handlers room-related
  - [ ] Test unitaire

- [ ] Backend: Create room endpoint refactor (AC1)
  - [ ] POST `/api/sessions` - accept `name` in body (required, 3-100 chars)
  - [ ] Validation: trim whitespace, reject empty/too long
  - [ ] Set default name: `{userName}'s Room` si non fourni (bien que requis par AC, fallback utile)
  - [ ] Set `created_by = userId` from session cookie
  - [ ] Set `last_activity_at = NOW()`
  - [ ] Generate persistent invite (is_active=true, no expires_at)
  - [ ] Auto-join creator (insert session_participants avec is_active=true)
  - [ ] Call `UpdateRoomActivity()`
  - [ ] Response: include invite link in response
  - [ ] Tests: creation avec name, validation errors, auto-join creator

- [ ] Backend: Join room endpoint (AC3)
  - [ ] POST `/api/sessions/:id/join` - body: `{ inviteCode?: string }`
  - [ ] Validate room exists, active=true
  - [ ] If user not participant: validate invite code + is_active=true
  - [ ] Auto-leave current active room (AC2):
    - Query `session_participants WHERE user_id=X AND is_active=true`
    - Update old room: `is_active=false, left_at=NOW()`
    - Broadcast `USER_LEFT` to old room
  - [ ] Join target room: upsert `session_participants` (is_active=true, left_at=null, joined_at=NOW() if new)
  - [ ] Call `UpdateRoomActivity(sessionID)`
  - [ ] Broadcast `USER_JOINED` to target room
  - [ ] Response: session details + participants list
  - [ ] Tests: join with code, rejoin without code, auto-leave previous, errors (404, 403, 410)

- [ ] Backend: Leave room endpoint (AC3)
  - [ ] POST `/api/sessions/:id/leave`
  - [ ] Validate user is active participant
  - [ ] Update `session_participants`: `is_active=false, left_at=NOW()`
  - [ ] Call `UpdateRoomActivity(sessionID)`
  - [ ] Broadcast `USER_LEFT` event
  - [ ] Response: session + remaining participants count
  - [ ] Tests: leave active room, check stale state after last user leaves, errors

- [ ] Backend: List user rooms (AC6)
  - [ ] GET `/api/sessions?state={LIVE|STALE|ARCHIVED}&limit=50`
  - [ ] Query rooms where user in `session_participants`
  - [ ] Join with `sessions` table
  - [ ] For each room: compute state (LIVE/STALE/ARCHIVED) via `GetState()`
  - [ ] Filter by state param if provided
  - [ ] Order by `last_activity_at DESC`
  - [ ] Limit results (default 50, max 100)
  - [ ] Response: array of room objects with computed fields
  - [ ] Tests: list all, filter by state, pagination, empty list

- [ ] Backend: Get/Regenerate invite endpoints (AC4)
  - [ ] GET `/api/sessions/:id/invite`
    - Validate user is participant
    - Return current active invite link + code
    - Tests: success, 403 non-participant
  - [ ] POST `/api/sessions/:id/invite/regenerate`
    - Validate user is participant (or creator only? → decide)
    - Disable old invite: `UPDATE session_invites SET is_active=false WHERE session_id=X`
    - Generate new invite code + token
    - Insert new invite with is_active=true
    - Call `UpdateRoomActivity()`
    - Return new invite link
    - Tests: regenerate, old code becomes invalid, 403 errors

- [ ] Backend: Archive room endpoint (AC7)
  - [ ] POST `/api/sessions/:id/archive`
  - [ ] Validate user is creator (or any participant? → decide)
  - [ ] Set `active=false`
  - [ ] Auto-leave all active participants: `UPDATE session_participants SET is_active=false, left_at=NOW() WHERE session_id=X AND is_active=true`
  - [ ] Broadcast `SESSION_ARCHIVED` event to all connected clients
  - [ ] Response: archived session object
  - [ ] Tests: archive room, check participants kicked, archived state

- [ ] Backend: Delete room endpoint (AC7)
  - [ ] DELETE `/api/sessions/:id`
  - [ ] Validate user is creator
  - [ ] Hard delete session (cascade: participants, invites, events via GORM)
  - [ ] Broadcast `SESSION_DELETED` to all connected clients (force disconnect)
  - [ ] Response: 204 No Content
  - [ ] Tests: delete room, cascade check, creator-only enforcement

### Backend: Room State & Activity Logic

- [ ] Backend: Enforce single active participation (AC2)
  - [ ] Constraint check helper: `EnsureSingleActiveParticipation(db, userID, newSessionID)`
  - [ ] Called in join endpoint BEFORE inserting new participation
  - [ ] Auto-leave logic extracted into reusable function
  - [ ] Tests: cannot be active in 2 rooms, auto-leave works

- [ ] Backend: WebSocket event handlers updates (AC2, AC7)
  - [ ] Add event types: `USER_JOINED`, `USER_LEFT`, `SESSION_ARCHIVED`, `SESSION_DELETED`
  - [ ] Broadcast `USER_LEFT` lors de auto-leave
  - [ ] Broadcast `SESSION_ARCHIVED` lors de archive (kick tous les users)
  - [ ] Broadcast `SESSION_DELETED` lors de delete (force disconnect)
  - [ ] Tests: verify events broadcasted correctly

- [ ] Backend: Cleanup handler simplification (AC9)
  - [ ] **REMOVE** auto-delete sessions after 7 days
  - [ ] **REMOVE** auto-deactivate expired sessions
  - [ ] KEEP: cleanup orphaned events older than 30 days
  - [ ] Update `cleanup.go` logic
  - [ ] Update cleanup tests

- [ ] Backend: Activity tracking integration (AC8)
  - [ ] Call `UpdateRoomActivity()` dans:
    - Join/leave handlers
    - Play/pause/seek handlers (story 1.7)
    - Track change polling (story 1.8)
    - Invite regenerate
    - Any WebSocket broadcast
  - [ ] Tests: verify last_activity_at updates correctly

### Frontend: Room Dashboard UI

- [ ] Frontend: Create room modal/form (AC1)
  - [ ] Component `CreateRoomModal.vue` ou inline form
  - [ ] Input "Room Name" (required, 3-100 chars validation)
  - [ ] Placeholder/default: "{userName}'s Room"
  - [ ] Submit → POST `/api/sessions` with name
  - [ ] On success:
    - Show success message + invite link
    - Copy link button
    - "Regenerate Link" button
    - Navigate to room (SessionLive)
  - [ ] Error handling: validation errors, network errors
  - [ ] Tests: form submission, validation, success flow

- [ ] Frontend: Room list/dashboard view (AC6)
  - [ ] New route: `/rooms` ou integrate into HomeView
  - [ ] Component `RoomList.vue`
  - [ ] Fetch GET `/api/sessions` on mount
  - [ ] Group rooms by state:
    - Section "Active Rooms" (LIVE rooms)
    - Section "Empty Rooms" (STALE rooms)
    - Section "Archived" (ARCHIVED rooms) - collapsible ou onglet séparé
  - [ ] Room card display:
    - 🟢 LIVE: nom, participant count, "Leave" button if user active, "Join" otherwise
    - 🟡 STALE: nom, "Empty • Xh ago", "Join" button
    - 📦 ARCHIVED: nom, "Archived", "Delete" button (creator only)
  - [ ] Sort by `lastActivityAt` DESC
  - [ ] Tests: render rooms, grouping logic, empty state

- [ ] Frontend: Room card actions (AC3, AC7)
  - [ ] Component `RoomCard.vue`
  - [ ] Button "Join" → POST `/api/sessions/:id/join`
  - [ ] Button "Leave" → POST `/api/sessions/:id/leave`
  - [ ] Button "Settings" → open settings modal
  - [ ] Success: refresh room list + navigate if joining
  - [ ] Error handling: display toast notifications
  - [ ] Tests: join, leave actions, UI state updates

- [ ] Frontend: Room settings modal (AC4, AC7)
  - [ ] Component `RoomSettingsModal.vue`
  - [ ] Tabs/sections:
    - "General": Rename room (future), view invite link
    - "Invite Link": Display current link, "Copy" button, "Regenerate" button
    - "Danger Zone": "Archive Room", "Delete Room" (with confirmation)
  - [ ] Regenerate link → POST `/api/sessions/:id/invite/regenerate`
  - [ ] Archive → POST `/api/sessions/:id/archive` + confirmation modal
  - [ ] Delete → DELETE `/api/sessions/:id` + confirmation modal ("cannot be undone")
  - [ ] Tests: regenerate, archive, delete flows

- [ ] Frontend: Auto-leave previous room on join (AC2)
  - [ ] Store: `useRoomStore()` track current active room ID
  - [ ] When joining new room via invite link or dashboard
  - [ ] Backend handles auto-leave (AC2)
  - [ ] Frontend: listen for `USER_LEFT` event for own user
  - [ ] Update local store: clear old active room, set new active room
  - [ ] Show notification: "Left {OldRoomName}, joined {NewRoomName}"
  - [ ] Tests: verify store state updates, notification shown

### Frontend: WebSocket Events & State Management

- [ ] Frontend: Handle new WS events (AC2, AC7)
  - [ ] Listen `USER_JOINED`: update participant list, show notification
  - [ ] Listen `USER_LEFT`: update participant list, show notification
  - [ ] Listen `SESSION_ARCHIVED`: show modal "Room archived", redirect to dashboard
  - [ ] Listen `SESSION_DELETED`: show modal "Room deleted", force disconnect, redirect
  - [ ] Tests: event handlers, UI updates, redirects

- [ ] Frontend: Room store state (AC5, AC6)
  - [ ] Store `useRoomStore()` ou extend `useSessionStore()`
  - [ ] State fields:
    - `currentRoom: { id, name, state, participants, ... }`
    - `userRooms: []` (from GET /api/sessions)
    - `activeRoomId: string | null` (user's current active room)
  - [ ] Actions:
    - `fetchUserRooms()`
    - `joinRoom(id, inviteCode?)`
    - `leaveRoom(id)`
    - `archiveRoom(id)`
    - `deleteRoom(id)`
    - `regenerateInvite(id)`
  - [ ] Computed: `isUserInRoom(id)`, `getUserActiveRoom()`
  - [ ] Tests: store actions, state mutations

### Integration & Testing

- [ ] Backend: Integration tests suite
  - [ ] Test full room lifecycle: create → join → leave → archive → delete
  - [ ] Test auto-leave on join new room (AC2)
  - [ ] Test persistent invite links (AC4)
  - [ ] Test invite regeneration disables old code
  - [ ] Test room state transitions (LIVE → STALE → ARCHIVED)
  - [ ] Test activity tracking updates
  - [ ] Test single active participation enforcement

- [ ] Frontend: E2E tests (Playwright)
  - [ ] Test: Create room with name
  - [ ] Test: Join room via invite link
  - [ ] Test: Leave room and rejoin
  - [ ] Test: Auto-leave previous room when joining new
  - [ ] Test: Regenerate invite link
  - [ ] Test: Archive room (creator)
  - [ ] Test: Delete room (creator)
  - [ ] Test: Dashboard displays rooms by state

- [ ] Documentation updates
  - [ ] Update API docs with new endpoints
  - [ ] Update architecture.md: room model vs session model
  - [ ] Add migration guide for existing users
  - [ ] Update README: features section

## Non-Functional Requirements

- **Performance:** Room list endpoint doit répondre < 500ms pour 50 rooms
- **Security:**
  - Creator-only actions enforced (delete)
  - Invite code validation remains secure (token hash)
  - Old invite codes must be truly disabled (not just soft-deleted)
- **UX:**
  - Auto-leave transition should be smooth, no confusing states
  - Clear notifications for all state changes
  - Confirmation modals for destructive actions
- **Data Integrity:**
  - Migration MUST preserve all existing session data
  - Cascade deletes properly configured (no orphaned records)

## Out of Scope (Future Stories)

- [ ] **Sync Leader logic (`sync_leader_user_id`)** - Epic 3 (gouvernance temps réel)
  - Automatically set when user takes playback action
  - Server polls sync leader's Spotify as source of truth
  - Transparent to users (internal technical mechanism)
- [ ] Room renaming after creation (Epic 2?)
- [ ] Room transfer ownership (Epic 2?)
- [ ] Room avatars/images (Epic 5 - polish)
- [ ] Room discovery/public rooms (Epic 3 - social)
- [ ] Room categories/tags (Epic 3 - social)
- [ ] Room search (Epic 3 - social)
- [ ] Notification preferences per room (Epic 5)
- [ ] Auto-expire stale rooms after X days (maybe never - fully manual)

## Dependencies

- **Depends on:** Story 1.9 (auth error handling) - should be complete first
- **Blocks:** Story 1.11 (room management UI enhancements), Epic 2 planning

## Definition of Done

- [ ] All AC validés (tests pass)
- [ ] Backend endpoints implémentés + tests unitaires + integration tests
- [ ] Frontend UI implémentée + tests unitaires + E2E tests
- [ ] Schema migration testée (fresh DB + existing data)
- [ ] Code review passed
- [ ] Documentation updated (API docs, architecture.md)
- [ ] No breaking changes to existing stories 1.1-1.9
- [ ] All existing tests still pass (regression check)
- [ ] Manual QA: create/join/leave/archive/delete flows validated

---

## Dev Agent Record

### Session 1 - 2026-02-01 - Story Creation

**Participants:** Party Mode agents (Bob, Winston, John, Mary, Amelia, Sally, Dr. Quinn, Victor)

**Created:** Story 1-10 specification following architectural discussion with Mathias

**Key Decisions:**

- Paradigm shift from ephemeral sessions to persistent rooms
- Single active participation model (auto-leave on join new room)
- Persistent invite links with regeneration capability
- Manual-only archive/delete (no auto-cleanup after 7 days)
- ExpiresAt → LastActivityAt repurposing
- Room naming required on creation
- **Owner vs Sync Leader clarification:**
  - `created_by` = room owner (user-facing, permanent, manages room)
  - `sync_leader_user_id` = sync source (internal, transparent, Epic 3 scope)
  - Sync leader logic deferred to Epic 3 (gouvernance temps réel)
  - User never sees "sync leader" concept in UI

**Next Steps:**

- Implement schema migration (task 1)
- Update backend models (task 2)
- Begin endpoint implementation

**Files:** Story file created at `_bmad-output/implementation-artifacts/1-10-persistent-room-model-session-lifecycle-management.md`
