# Story 1.7: Synchronisation play/pause (événements ordonnés + idempotence)

Status: ready-for-dev

## Story

As a participant,
I want que play/pause se propage à tous,
So that l'écoute reste synchronisée.

## Acceptance Criteria

1. **Propagation événement Pause**
   - **Given** deux participants connectés à la même session en état "synced"
   - **When** l'un met pause via l'UI
   - **Then** tous les clients reçoivent un événement WS `PLAYER_PAUSED` avec `eventSeq` monotone
   - **And** la lecture est en pause sur tous les appareils Spotify dans un délai ≤ 3s (NFR1)
   - **And** l'UI de tous les participants reflète l'état "paused"

2. **Propagation événement Play/Resume**
   - **Given** deux participants connectés à la même session avec lecture en pause
   - **When** l'un clique play/reprise
   - **Then** tous les clients reçoivent un événement WS `PLAYER_RESUMED`
   - **And** la lecture reprend sur tous les appareils Spotify dans un délai ≤ 3s (NFR1)
   - **And** tous les participants voient l'état "playing" et la progression se mettre à jour

3. **Propagation Skip/Next Track**
   - **Given** deux participants connectés à la même session
   - **When** l'un déclenche "piste suivante" (skip)
   - **Then** le changement est propagé à tous via événement WS `TRACK_CHANGED`
   - **And** l'UI "Now Playing" se met à jour pour tous les participants
   - **And** le nouveau track démarre chez tous dans un délai ≤ 3s

4. **Propagation Seek (position dans le morceau)**
   - **Given** deux participants connectés à la même session
   - **When** l'un effectue un seek (avance/recul dans le morceau)
   - **Then** la position converge chez tous via événement WS `PLAYER_SEEKED`
   - **And** l'écart de position entre participants reste ≤ 3s (NFR1)
   - **And** le serveur reste l'arbitre de l'état partagé (pas de dépendance aux horloges client)

5. **Idempotence des commandes (clientMsgId)**
   - **Given** un client envoie une commande (pause, play, skip, seek)
   - **When** il renvoie le même message avec le même `clientMsgId` (retry réseau)
   - **Then** le serveur traite l'action de manière idempotente (pas de double-exécution)
   - **And** le serveur répond avec le même `eventSeq` que lors de la première exécution
   - **And** aucun événement WS dupliqué n'est broadcasté

6. **Événements ordonnés (eventSeq monotone)**
   - **Given** plusieurs actions se produisent en séquence (pause, play, skip)
   - **When** les événements WS sont broadcastés
   - **Then** chaque événement a un `eventSeq` unique et strictement croissant
   - **And** les clients peuvent détecter des messages manqués via gaps dans la séquence
   - **And** l'ordre d'application côté client respecte l'ordre `eventSeq` serveur

7. **Trust but Verify (resync avec état Spotify réel)**
   - **Given** une commande de lecture (pause/play/skip/seek) est envoyée à Spotify
   - **When** le serveur reçoit la réponse de l'API Spotify
   - **Then** il relit l'état réel du player Spotify (GET `/me/player`)
   - **And** il broadcaste l'état vérifié (pas seulement l'intention) dans le payload de l'événement
   - **And** en cas de divergence (ex: pause refusée par Spotify), l'état réel prime

## Tasks / Subtasks

- [ ] Backend: Endpoints commandes player (AC: 1, 2, 3, 4, 7)
  - [ ] `POST /api/sessions/:sessionId/player/pause` - mettre en pause
  - [ ] `POST /api/sessions/:sessionId/player/resume` - reprendre lecture
  - [ ] `POST /api/sessions/:sessionId/player/next` - piste suivante (skip)
  - [ ] `POST /api/sessions/:sessionId/player/seek` - seek position (body: `{ positionMs }`)
  - [ ] Tous endpoints requièrent auth (cookie-session) + participant de la session
  - [ ] Valider que participant est en état "synced" avant commande

- [ ] Backend: Orchestration commandes Spotify + Trust but Verify (AC: 7)
  - [ ] Envoyer commande à Spotify Player API (PUT `/me/player/pause`, `/play`, `/next`, `/seek`)
  - [ ] Gérer tokens Spotify (refresh si nécessaire)
  - [ ] Après commande, relire état réel: GET `/me/player` (Trust but Verify)
  - [ ] Extraire état vérifié: `isPlaying`, `positionMs`, `trackId`, `trackName`, `artist`, etc.
  - [ ] Gestion erreurs Spotify: 429 rate limit, 403 no active device, 404 no context, timeouts
  - [ ] Retourner erreurs stables: `SPOTIFY_RATE_LIMITED`, `SPOTIFY_NO_DEVICE`, `SPOTIFY_UNAVAILABLE`

- [ ] Backend: Gestion idempotence (AC: 5)
  - [ ] Accepter `clientMsgId` dans body de chaque commande (optionnel mais recommandé)
  - [ ] Maintenir cache/map par session: `clientMsgId → eventSeq` pour dédoublonnage
  - [ ] Si `clientMsgId` déjà traité: retourner cached `eventSeq` sans réexécuter commande
  - [ ] Si `clientMsgId` absent: traiter normalement (pas d'erreur, juste pas d'idempotence)
  - [ ] TTL du cache idempotence: 1 minute (suffisant pour retry réseau)

- [ ] Backend: Broadcast événements WebSocket (AC: 1, 2, 3, 4, 6)
  - [ ] Événement `PLAYER_PAUSED` après succès pause
  - [ ] Événement `PLAYER_RESUMED` après succès play/resume
  - [ ] Événement `TRACK_CHANGED` après succès skip/next
  - [ ] Événement `PLAYER_SEEKED` après succès seek
  - [ ] Tous événements incluent payload complet d'état player (isPlaying, positionMs, track info)
  - [ ] Générer `eventSeq` monotone par session (incrémenter compteur Hub)
  - [ ] Broadcaster à tous les participants connectés de la session

- [ ] Backend: Gestion eventSeq (AC: 6)
  - [ ] Maintenir compteur `eventSeq` par session dans Hub (mémoire MVP)
  - [ ] Incrémenter atomiquement à chaque broadcast
  - [ ] Persister dans event log DB (append-only) pour debug/audit
  - [ ] Exposer `eventSeq` actuel dans snapshot initial WS

- [ ] Backend: Event Log (append-only, audit + debug) (AC: 6, 7)
  - [ ] Table `events` avec colonnes: `id`, `session_id`, `event_seq`, `event_type`, `payload_json`, `created_at`
  - [ ] Insérer chaque événement broadcasté (async, ne pas bloquer broadcast)
  - [ ] Indexes: `(session_id, event_seq)` pour requêtes resync futures

- [ ] Frontend: Actions player dans store (AC: 1, 2, 3, 4)
  - [ ] Action `pausePlayer()` dans `usePlayerStore` ou `useSessionStore`
  - [ ] Action `resumePlayer()` pour reprendre lecture
  - [ ] Action `nextTrack()` pour skip
  - [ ] Action `seekTo(positionMs)` pour avancer/reculer
  - [ ] Chaque action génère un `clientMsgId` unique (UUID v4)
  - [ ] Appeler endpoint backend correspondant avec `clientMsgId` dans body
  - [ ] Gérer états: loading/success/error par action

- [ ] Frontend: Réception événements WebSocket (AC: 1, 2, 3, 4, 6)
  - [ ] Écouter événements `PLAYER_PAUSED`, `PLAYER_RESUMED`, `TRACK_CHANGED`, `PLAYER_SEEKED`
  - [ ] Parser payload et mettre à jour store player (isPlaying, positionMs, track)
  - [ ] Vérifier `eventSeq` pour détecter gaps (messages manqués)
  - [ ] Si gap détecté: logger warning + déclencher resync (future story 4.3)
  - [ ] Appliquer événements dans l'ordre `eventSeq` croissant (buffer si besoin)

- [ ] Frontend: UI contrôles player (AC: 1, 2, 3, 4)
  - [ ] Bouton Pause visible quand `isPlaying === true`
  - [ ] Bouton Play visible quand `isPlaying === false`
  - [ ] Bouton Next/Skip toujours disponible
  - [ ] Slider/scrubber pour seek (optionnel MVP, sinon boutons ±10s)
  - [ ] États de loading sur boutons pendant action en cours
  - [ ] Feedback visuel immédiat (optimistic update) puis correction si erreur

- [ ] Frontend: Gestion erreurs Spotify (AC: 7)
  - [ ] Afficher toast/notification si erreur Spotify (rate limited, no device, etc.)
  - [ ] Messages spécifiques par code erreur:
    - `SPOTIFY_RATE_LIMITED`: "Spotify is busy. Please try again in a moment."
    - `SPOTIFY_NO_DEVICE`: "No active Spotify device found. Please start Spotify."
    - `SPOTIFY_UNAVAILABLE`: "Unable to control playback. Please check your connection."
  - [ ] Bouton retry disponible après erreur

- [ ] Tests Backend (AC: tous)
  - [ ] Test `POST /player/pause` avec participant valide → commande Spotify + broadcast `PLAYER_PAUSED`
  - [ ] Test idempotence: même `clientMsgId` deux fois → une seule exécution, même `eventSeq`
  - [ ] Test `eventSeq` strictement croissant sur plusieurs commandes
  - [ ] Test Trust but Verify: mock réponse Spotify divergente → broadcast état réel
  - [ ] Test rate limit Spotify (429) → erreur stable `SPOTIFY_RATE_LIMITED`
  - [ ] Test permissions: non-participant ne peut pas envoyer commandes
  - [ ] Test état sync: participant "ready" (pas "synced") ne peut pas envoyer commandes

- [ ] Tests Frontend (AC: tous)
  - [ ] Test action `pausePlayer()` appelle endpoint avec `clientMsgId`
  - [ ] Test réception événement `PLAYER_PAUSED` met à jour store
  - [ ] Test UI boutons Play/Pause toggle selon état `isPlaying`
  - [ ] Test détection gap `eventSeq` (mock événements manquants)
  - [ ] Test affichage erreur si commande échoue (mock fetch error)

## Dev Notes

### Architecture Play/Pause Sync

**Flow complet (exemple: Pause):**

1. **Frontend:** User clique bouton Pause
2. **Frontend:** Store génère `clientMsgId` (UUID v4), appelle `POST /player/pause { clientMsgId }`
3. **Backend:** Reçoit commande, vérifie auth/participation/sync_state
4. **Backend:** Check idempotence cache (`clientMsgId` déjà traité?)
5. **Backend:** Si nouveau: appelle Spotify `PUT /me/player/pause`
6. **Backend:** Spotify répond (success ou error)
7. **Backend:** Trust but Verify: `GET /me/player` pour lire état réel
8. **Backend:** Extrait état vérifié (isPlaying=false, positionMs, track, etc.)
9. **Backend:** Génère `eventSeq` (incrémente compteur session)
10. **Backend:** Broadcaster événement WS `PLAYER_PAUSED` avec état vérifié à tous participants
11. **Backend:** Persiste événement dans event log DB (async)
12. **Backend:** Répond au client initiateur avec `eventSeq`
13. **Frontend (tous):** Reçoivent événement WS, mettent à jour store player, UI se rafraîchit

**Délai total cible:** ≤ 3 secondes (NFR1)

### Idempotence avec clientMsgId

**Problème:** Retry réseau peut envoyer la même commande plusieurs fois, causant double-exécution (ex: pause → play → pause involontaire).

**Solution:**

- Client génère `clientMsgId` unique par intention utilisateur (pas par requête HTTP)
- Serveur maintient cache: `map[clientMsgId]eventSeq` par session
- Si `clientMsgId` déjà traité: retourner cached `eventSeq` sans réexécuter
- TTL cache: 1 minute (suffisant pour retries réseau typiques)

**Implémentation suggérée (backend):**

```go
type IdempotenceCache struct {
  mu    sync.RWMutex
  cache map[string]map[string]int64 // sessionId -> clientMsgId -> eventSeq
  ttl   time.Duration
}

func (c *IdempotenceCache) Get(sessionID, clientMsgID string) (int64, bool) {
  c.mu.RLock()
  defer c.mu.RUnlock()
  if sessionCache, ok := c.cache[sessionID]; ok {
    if eventSeq, ok := sessionCache[clientMsgID]; ok {
      return eventSeq, true
    }
  }
  return 0, false
}

func (c *IdempotenceCache) Set(sessionID, clientMsgID string, eventSeq int64) {
  c.mu.Lock()
  defer c.mu.Unlock()
  if c.cache[sessionID] == nil {
    c.cache[sessionID] = make(map[string]int64)
  }
  c.cache[sessionID][clientMsgID] = eventSeq
  // TODO: implement TTL cleanup (goroutine or lazy deletion)
}
```

### Trust but Verify Pattern

**Contexte:** L'API Spotify peut être ambiguë:

- Commande acceptée (200) mais état réel différent (ex: device pas actif)
- Commande refusée (403) mais état réel déjà correct
- Latence/timing: commande envoyée, état changé par autre source avant lecture

**Solution:**
Après chaque commande Spotify, toujours relire l'état réel via `GET /me/player`:

```
1. PUT /me/player/pause → 200 OK
2. GET /me/player → { is_playing: false, ... }
3. Broadcast état vérifié (is_playing=false) pas juste l'intention
```

**Bénéfices:**

- Convergence état: tous les clients voient la même réalité Spotify
- Résilience: détecte automatiquement divergences (device disconnect, changements externes)
- Debug: event log contient état réel observé, pas juste intentions

**Coût:**

- +1 requête API Spotify par commande (acceptable pour MVP)
- Latence légèrement augmentée (~100-200ms supplémentaires)

### EventSeq Monotone

**Garantie:** `eventSeq` strictement croissant par session (source d'autorité serveur).

**Implémentation Hub:**

```go
type SessionHub struct {
  mu       sync.RWMutex
  sessions map[string]*SessionState
}

type SessionState struct {
  mu         sync.Mutex
  eventSeq   int64
  clients    map[string]*Client
}

func (s *SessionState) NextEventSeq() int64 {
  s.mu.Lock()
  defer s.mu.Unlock()
  s.eventSeq++
  return s.eventSeq
}
```

**Détection gaps (client):**

```typescript
let lastEventSeq = 0

function onWebSocketMessage(msg: WebSocketMessage) {
  const { eventSeq } = msg
  if (eventSeq !== lastEventSeq + 1) {
    console.warn(`Gap detected: expected ${lastEventSeq + 1}, got ${eventSeq}`)
    // Future (Story 4.3): trigger resync
  }
  lastEventSeq = eventSeq
  // Process message...
}
```

### Gestion Rate Limit Spotify (429)

**Pattern établi (Story 1.2):**

- Lire header `Retry-After` (secondes)
- Backoff + jitter
- Retourner erreur stable `SPOTIFY_RATE_LIMITED` au client
- Client affiche message + suggère retry

**Amélioration (cette story):**

- Marquer session comme "rate-limited" temporairement
- Broadcaster événement WS `SESSION_RATE_LIMITED` avec `retryAfter` timestamp
- Tous les clients affichent notification + désactivent contrôles temporairement
- Auto-réactiver après expiration `retryAfter`

### Événements WebSocket

**Format standard (établi Story 1.5):**

```json
{
  "type": "PLAYER_PAUSED",
  "sessionId": "sess_abc123",
  "eventSeq": 45,
  "sentAt": "2026-01-28T15:30:45.123Z",
  "payload": {
    "userId": "user_xyz",
    "isPlaying": false,
    "positionMs": 67500,
    "track": {
      "id": "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp",
      "name": "Mr. Brightside",
      "artist": "The Killers",
      "album": "Hot Fuss",
      "durationMs": 222000,
      "imageUrl": "https://i.scdn.co/image/..."
    }
  }
}
```

**Types d'événements (cette story):**

- `PLAYER_PAUSED`: lecture mise en pause
- `PLAYER_RESUMED`: lecture reprise
- `TRACK_CHANGED`: changement de track (skip/next/prev)
- `PLAYER_SEEKED`: changement de position dans le morceau
- `SESSION_RATE_LIMITED`: session temporairement rate-limitée par Spotify (futur optionnel)

**Payload commun:**

- `userId`: qui a déclenché l'action
- `isPlaying`: état lecture actuel (vérifié)
- `positionMs`: position actuelle dans le morceau
- `track`: infos track actuel (id, name, artist, album, duration, imageUrl)
- `timestamp`: quand l'action s'est produite (ISO UTC)

### Event Log DB (append-only)

**Table `events`:**

```sql
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  event_seq INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(session_id, event_seq)
);
CREATE INDEX idx_events_session_seq ON events(session_id, event_seq);
```

**Usage:**

- Audit trail complet de toutes les actions dans une session
- Debug: rejouer l'historique d'événements
- Future (Story 4.3): resync clients avec `events since eventSeq`
- Pas de DELETE/UPDATE: append-only pour garantir intégrité

**Insertion async:**
Écriture non-bloquante pour ne pas ralentir broadcast WS:

```go
go func() {
  err := db.Create(&Event{
    SessionID:   sessionID,
    EventSeq:    eventSeq,
    EventType:   "PLAYER_PAUSED",
    PayloadJSON: payloadJSON,
  }).Error
  if err != nil {
    log.Error("Failed to persist event", "error", err)
  }
}()
```

### Frontend: Store Player

**État suggéré:**

```typescript
interface PlayerState {
  isPlaying: boolean
  positionMs: number
  track: {
    id: string
    name: string
    artist: string
    album: string
    durationMs: number
    imageUrl: string
  } | null
  loading: {
    pause: boolean
    resume: boolean
    next: boolean
    seek: boolean
  }
  error: string | null
  lastEventSeq: number
}
```

**Actions:**

```typescript
const playerStore = defineStore('player', {
  state: (): PlayerState => ({ ... }),
  actions: {
    async pausePlayer() {
      this.loading.pause = true
      try {
        const clientMsgId = crypto.randomUUID()
        const response = await fetch(`/api/sessions/${sessionId}/player/pause`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ clientMsgId })
        })
        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.code)
        }
        // Success: attendre événement WS pour update
      } catch (err) {
        this.error = err.message
      } finally {
        this.loading.pause = false
      }
    },
    
    handlePlayerPaused(event: WebSocketMessage) {
      // Vérifier eventSeq
      if (event.eventSeq !== this.lastEventSeq + 1) {
        console.warn('EventSeq gap detected')
      }
      this.lastEventSeq = event.eventSeq
      
      // Mettre à jour état
      this.isPlaying = event.payload.isPlaying
      this.positionMs = event.payload.positionMs
      this.track = event.payload.track
    }
  }
})
```

### Frontend: UI Contrôles Player

**Composant suggéré: `PlayerControls.vue`**

```vue
<template>
  <div class="player-controls">
    <!-- Play/Pause Toggle -->
    <button
      @click="togglePlayPause"
      :disabled="loading"
      class="btn-primary"
    >
      <Icon :name="isPlaying ? 'pause' : 'play'" />
    </button>
    
    <!-- Next/Skip -->
    <button
      @click="nextTrack"
      :disabled="loading"
      class="btn-secondary"
    >
      <Icon name="skip-forward" />
    </button>
    
    <!-- Position Slider (optionnel MVP) -->
    <input
      v-if="track"
      type="range"
      :min="0"
      :max="track.durationMs"
      :value="positionMs"
      @change="seekTo($event.target.value)"
      :disabled="loading"
    />
  </div>
</template>

<script setup lang="ts">
const playerStore = usePlayerStore()
const { isPlaying, positionMs, track } = storeToRefs(playerStore)

const loading = computed(() =>
  playerStore.loading.pause ||
  playerStore.loading.resume ||
  playerStore.loading.next ||
  playerStore.loading.seek
)

async function togglePlayPause() {
  if (isPlaying.value) {
    await playerStore.pausePlayer()
  } else {
    await playerStore.resumePlayer()
  }
}

async function nextTrack() {
  await playerStore.nextTrack()
}

async function seekTo(positionMs: number) {
  await playerStore.seekTo(positionMs)
}
</script>
```

**UX Optimistic Update (optionnel):**
Pour réactivité perçue, mettre à jour UI immédiatement puis corriger si événement WS diverge:

```typescript
async pausePlayer() {
  // Optimistic update
  const previousState = this.isPlaying
  this.isPlaying = false
  
  try {
    // Send command...
  } catch (err) {
    // Rollback on error
    this.isPlaying = previousState
  }
  // Final state will be set by WS event
}
```

### Learnings des Stories Précédentes

**Story 1.5 (WebSocket):**

- Hub WebSocket fonctionnel avec broadcast par session
- Enveloppe message standard établie
- EventSeq infrastructure en place (compteur par session)
- Client détection déconnexion (≤ 5s)

**Story 1.6 (Airlock/Sas):**

- État `sync_state` sur `session_participants` ("ready" vs "synced")
- Validation backend: commandes player nécessitent `sync_state = 'synced'`
- Pattern fetch avec `clientMsgId` déjà utilisé pour startListening (réutilisable ici)

**Story 1.2 (Auth Spotify):**

- `SpotifyClient` abstraction pour appels API
- Gestion refresh tokens automatique
- Pattern rate limiting avec `Retry-After`

**Story 1.4 (Contrôle d'accès):**

- Middleware `RequireParticipant` pour valider participation
- Pattern erreurs stables (codes + messages)

### Code Patterns Établis

**Backend: Handler avec idempotence**

```go
func (h *SessionHandler) PausePlayer(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context()
  userID := GetUserIDFromSession(r)
  sessionID := chi.URLParam(r, "sessionId")
  
  // Parse body
  var req struct {
    ClientMsgID string `json:"clientMsgId"`
  }
  json.NewDecoder(r.Body).Decode(&req)
  
  // Check idempotence
  if req.ClientMsgID != "" {
    if eventSeq, found := h.idempotenceCache.Get(sessionID, req.ClientMsgID); found {
      respondJSON(w, http.StatusOK, map[string]interface{}{
        "eventSeq": eventSeq,
        "cached": true,
      })
      return
    }
  }
  
  // Validate sync state
  participant, err := h.repo.GetParticipant(ctx, sessionID, userID)
  if participant.SyncState != "synced" {
    respondError(w, http.StatusForbidden, "PARTICIPANT_NOT_SYNCED", "Must be in synced state")
    return
  }
  
  // Call Spotify
  err = h.spotifyClient.Pause(ctx, userID)
  if err != nil {
    // Handle Spotify errors (rate limit, no device, etc.)
    respondError(w, http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE", err.Error())
    return
  }
  
  // Trust but Verify: read actual state
  playerState, err := h.spotifyClient.GetPlayerState(ctx, userID)
  
  // Broadcast event
  eventSeq := h.hub.BroadcastPlayerPaused(sessionID, userID, playerState)
  
  // Cache for idempotence
  if req.ClientMsgID != "" {
    h.idempotenceCache.Set(sessionID, req.ClientMsgID, eventSeq)
  }
  
  // Persist event (async)
  go h.persistEvent(sessionID, eventSeq, "PLAYER_PAUSED", playerState)
  
  respondJSON(w, http.StatusOK, map[string]interface{}{
    "eventSeq": eventSeq,
  })
}
```

**Frontend: Action avec clientMsgId**

```typescript
async pausePlayer() {
  this.loading.pause = true
  this.error = null
  
  const clientMsgId = crypto.randomUUID()
  
  try {
    const response = await fetch(`/api/sessions/${this.sessionId}/player/pause`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ clientMsgId })
    })
    
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.code || 'PAUSE_FAILED')
    }
    
    const data = await response.json()
    console.log('Pause command sent, eventSeq:', data.eventSeq)
    
    // State update will come via WS event
  } catch (err) {
    this.error = err.message
    // Show user-friendly error
    useNotifications().show({
      type: 'error',
      message: this.getFriendlyErrorMessage(err.message)
    })
  } finally {
    this.loading.pause = false
  }
}

getFriendlyErrorMessage(code: string): string {
  const messages = {
    'SPOTIFY_RATE_LIMITED': 'Spotify is busy. Please try again in a moment.',
    'SPOTIFY_NO_DEVICE': 'No active Spotify device found. Please start Spotify.',
    'SPOTIFY_UNAVAILABLE': 'Unable to control playback. Please check your connection.',
    'PARTICIPANT_NOT_SYNCED': 'Please click "Start Listening" first.',
  }
  return messages[code] || 'An error occurred. Please try again.'
}
```

### Testing Strategy

**Backend Tests Prioritaires:**

1. **Test commandes basiques:**
   - Pause réussie → broadcast `PLAYER_PAUSED` avec état vérifié
   - Resume réussie → broadcast `PLAYER_RESUMED`
   - Next réussie → broadcast `TRACK_CHANGED`
   - Seek réussie → broadcast `PLAYER_SEEKED`

2. **Test idempotence:**
   - Envoyer même commande avec même `clientMsgId` deux fois
   - Vérifier: une seule exécution Spotify, même `eventSeq` retourné
   - Vérifier: un seul événement WS broadcasté

3. **Test eventSeq monotone:**
   - Envoyer plusieurs commandes en séquence
   - Vérifier: `eventSeq` strictement croissant (1, 2, 3, ...)

4. **Test Trust but Verify:**
   - Mock Spotify: commande "pause" acceptée mais GET player retourne `is_playing: true`
   - Vérifier: événement broadcasté reflète état réel (isPlaying: true)

5. **Test permissions:**
   - Non-participant tente commande → `FORBIDDEN`
   - Participant "ready" (pas "synced") tente commande → `PARTICIPANT_NOT_SYNCED`

6. **Test erreurs Spotify:**
   - Mock 429 rate limit → erreur `SPOTIFY_RATE_LIMITED`
   - Mock 404 no device → erreur `SPOTIFY_NO_DEVICE`

**Frontend Tests Prioritaires:**

1. **Test actions store:**
   - `pausePlayer()` génère `clientMsgId` et appelle endpoint
   - `resumePlayer()` idem
   - États loading mis à jour correctement

2. **Test réception événements WS:**
   - Recevoir `PLAYER_PAUSED` → store mis à jour (isPlaying: false)
   - Recevoir `TRACK_CHANGED` → track info mise à jour

3. **Test détection gaps eventSeq:**
   - Mock événements avec `eventSeq` non-consécutifs (ex: 1, 2, 4)
   - Vérifier: warning loggé

4. **Test UI contrôles:**
   - Clic bouton Pause → action déclenchée
   - Bouton Play/Pause toggle selon état `isPlaying`
   - Boutons disabled pendant loading

5. **Test affichage erreurs:**
   - Mock fetch error → message d'erreur affiché
   - Message user-friendly selon code erreur

### Performance Considerations

**Latence cible: ≤ 3s (NFR1)**

Breakdown estimé:

- Client → Backend: ~50ms
- Backend validation + DB: ~50ms
- Backend → Spotify PUT: ~200ms
- Spotify → Backend GET (verify): ~200ms
- Backend broadcast WS: ~50ms
- WS → All clients: ~50ms
- Client processing + UI: ~50ms
**Total: ~650ms** ✅ Well under 3s target

**Optimisations possibles:**

- Paralléliser Spotify GET avec broadcast (sacrifice Trust but Verify légèrement)
- Cache player state côté backend (1s TTL) pour réduire GET requests
- Batch multiple commandes rapides (ex: seek repeated) avec debounce

**Throttling UI:**

- Seek slider: debounce 300ms avant envoi commande
- Multi-clic boutons: disable pendant action en cours
- Rate limit client-side: max 1 commande/seconde par type

### Security Considerations

**Validation stricte:**

- Toutes les commandes nécessitent auth + participation + sync_state="synced"
- `clientMsgId` doit être UUID valide (si fourni)
- `positionMs` dans seek doit être ≥ 0 et ≤ track.durationMs

**Pas de secrets exposés:**

- Tokens Spotify jamais dans payloads WS
- Event log ne contient pas de credentials

**Rate limiting suggéré:**

- Par user: max 10 commandes/minute (prévenir spam/abuse)
- Par session: max 50 commandes/minute global

**Audit trail:**

- Event log contient `userId` de l'initiateur
- Logs backend incluent `userID` et `sessionID` pour chaque commande

### Monitoring & Observability

**Métriques suggérées:**

- Latence commande player (P50, P95, P99)
- Taux erreurs Spotify par type (429, 404, 403, timeouts)
- Nombre de commandes par type (pause, play, skip, seek)
- Cache hit rate idempotence
- Gaps eventSeq détectés côté client

**Logs structurés:**

```json
{
  "level": "info",
  "message": "Player command executed",
  "sessionId": "sess_abc123",
  "userId": "user_xyz",
  "command": "pause",
  "clientMsgId": "uuid-...",
  "eventSeq": 45,
  "latencyMs": 650,
  "spotifyResult": "success"
}
```

**Timeline UI (Story 5.3 future):**
Afficher dans l'UI une timeline des derniers événements:

- 15:30:45 - User1 paused
- 15:30:50 - User2 resumed
- 15:31:00 - User1 skipped to next track

Utile pour debug et "Trust but Verify" transparency.

## Project Structure Notes

### Backend Files à Créer

```
backend/
  internal/
    handlers/
      player.go           # Nouveau: PausePlayer, ResumePlayer, NextTrack, SeekPlayer
      player_test.go      # Tests commandes player
      idempotence.go      # Nouveau: IdempotenceCache
      idempotence_test.go # Tests cache idempotence
    models/
      event.go            # Nouveau: Event model pour event log
    realtime/
      events.go           # Ajouter: types PLAYER_PAUSED, PLAYER_RESUMED, TRACK_CHANGED, PLAYER_SEEKED
  migrations/
    YYYYMMDDHHMMSS_create_events_table.sql  # Nouveau: table events
```

### Backend Files à Modifier

```
backend/
  cmd/boeuf-server/main.go    # Enregistrer routes /player/*
  internal/realtime/hub.go    # Ajouter méthodes broadcast pour événements player
  internal/spotify/client.go  # Ajouter méthodes Pause, Resume, Next, Seek, GetPlayerState
```

### Frontend Files à Créer

```
frontend/
  src/
    stores/
      player.ts           # Nouveau: usePlayerStore avec actions + état player
    components/
      PlayerControls.vue  # Nouveau: UI contrôles play/pause/skip/seek
      NowPlaying.vue      # Nouveau ou enrichir: affichage track actuel + progression
```

### Frontend Files à Modifier

```
frontend/
  src/
    stores/
      realtime.ts         # Ajouter dispatch événements PLAYER_* vers playerStore
    views/
      SessionView.vue     # Intégrer PlayerControls et NowPlaying
```

### Database Migration

**Migration: Create Events Table**

```sql
-- migrations/YYYYMMDDHHMMSS_create_events_table.sql
-- +goose Up
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  event_seq INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_events_session_seq ON events(session_id, event_seq);
CREATE INDEX idx_events_created_at ON events(created_at);

-- +goose Down
DROP TABLE events;
```

## References

- **Source:** [_bmad-output/planning-artifacts/epics.md](../planning-artifacts/epics.md#story-17-synchronisation-playPause-événements-ordonnés--idempotence) (Story 1.7)
- **Source:** [_bmad-output/planning-artifacts/architecture.md](../planning-artifacts/architecture.md#api--communication-patterns) (WebSocket patterns, idempotence, Trust but Verify)
- **Source:** [_bmad-output/project-context.md](../project-context.md) (Conventions naming, formats, erreurs, event_seq, client_msg_id)
- **Learnings:** [1-5-websocket-session-presence-participants-etat-connexion.md](./1-5-websocket-session-presence-participants-etat-connexion.md) (WebSocket Hub, broadcast, eventSeq monotone)
- **Learnings:** [1-6-sas-start-listening-anti-autoplay-et-etat-ready-synced.md](./1-6-sas-start-listening-anti-autoplay-et-etat-ready-synced.md) (sync_state validation, pattern clientMsgId)
- **Learnings:** [1-2-auth-spotify-oauth-authorization-code-pkce-avec-session-cookie.md](./1-2-auth-spotify-oauth-authorization-code-pkce-avec-session-cookie.md) (SpotifyClient, rate limiting, refresh tokens)
- **Spec:** [Spotify Web API - Player Endpoints](https://developer.spotify.com/documentation/web-api/reference/start-a-users-playback) (PUT /me/player/play, /pause, /next, /seek, GET /me/player)
- **Spec:** [RFC 6455 - WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- **Pattern:** [Idempotency in Distributed Systems](https://www.microsoft.com/en-us/research/publication/idempotency-is-not-a-medical-condition/) (Academic reference)

## Dev Agent Record

### Agent Model Used

_À remplir lors de l'implémentation_

### Debug Log References

_À remplir lors de l'implémentation_

### Completion Notes List

_À remplir lors de l'implémentation_

### File List

_À remplir lors de l'implémentation_
