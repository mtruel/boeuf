# Story 1.6: Sas "Start Listening" (anti-autoplay) et état "Ready → Synced"

Status: ready-for-dev

## Story

As a invité,
I want un écran "Sas" avec un bouton Start Listening,
So that je garde le contrôle et évite l'autoplay bloqué/surprenant.

## Acceptance Criteria

1. **Écran Sas (état Ready)**
   - **Given** l'utilisateur est dans une session mais n'a pas encore "démarré l'écoute"
   - **When** il ouvre la page session
   - **Then** la musique ne démarre pas automatiquement
   - **And** l'UI affiche un bouton principal "Start Listening / Sync Now"

2. **Activation Sync (Ready → Synced)**
   - **Given** l'utilisateur clique sur "Start Listening"
   - **When** le backend initialise la synchronisation
   - **Then** l'UI passe en état "Synced/Live" (feedback visuel clair)

3. **État persiste (refresh sans restart)**
   - **Given** un utilisateur en état "Synced"
   - **When** il rafraîchit la page
   - **Then** il revient directement à l'état "Synced" sans passer par le Sas
   - **And** aucune musique ne redémarre automatiquement (juste reconnexion état)

4. **Feedback visuel transition d'état**
   - **Given** l'utilisateur clique sur "Start Listening"
   - **When** la transition Ready → Synced s'effectue
   - **Then** l'UI affiche un feedback intermédiaire (ex: "Connecting...", "Syncing...")
   - **And** la transition se complète en ≤ 3 secondes
   - **And** l'état final affiche clairement "Live" ou "Synced" avec indicateur visuel stable

5. **Erreurs de synchronisation**
   - **Given** l'utilisateur clique sur "Start Listening" mais son compte Spotify n'est pas accessible
   - **When** l'initialisation échoue
   - **Then** l'UI affiche un message d'erreur actionnable (ex: "Unable to sync. Please check your Spotify connection.")
   - **And** l'utilisateur reste en état "Ready" avec possibilité de réessayer

## Tasks / Subtasks

- [ ] Backend: État session participant (AC: 1, 2, 3)
  - [ ] Ajouter colonne `sync_state` à table `session_participants` (valeurs: "ready", "synced")
  - [ ] Endpoint `POST /api/sessions/:sessionId/sync/start` pour marquer participant comme "synced"
  - [ ] Endpoint `GET /api/sessions/:sessionId/me` pour récupérer l'état actuel du participant
  - [ ] Valider que seuls les participants authentifiés peuvent démarrer sync
  - [ ] Broadcaster événement WS `PARTICIPANT_SYNC_STATE_CHANGED` lors de changement d'état

- [ ] Backend: Logique d'initialisation sync (AC: 2, 5)
  - [ ] Valider que le participant a un token Spotify valide/rafraîchi
  - [ ] Retourner erreur stable `SPOTIFY_NOT_CONNECTED` si tokens absents/invalides
  - [ ] Lecture état Spotify actuel (track/position/isPlaying) pour baseline
  - [ ] Enregistrer baseline dans session (pour nouveaux arrivants)
  - [ ] Gestion erreur: rate limit Spotify, token expiré, player inactif

- [ ] Frontend: Store état sync participant (AC: 1, 2, 3)
  - [ ] Ajouter `syncState` dans `useSessionStore` ou dédié `useSyncStore`
  - [ ] États: "ready", "syncing", "synced", "error"
  - [ ] Fonction `startListening()` qui appelle backend et gère transitions
  - [ ] Persistance `syncState` en sessionStorage pour survivre aux refreshes
  - [ ] Récupération état serveur au mount (GET `/api/sessions/:sessionId/me`)

- [ ] Frontend: UI Sas (écran Ready) (AC: 1, 4)
  - [ ] Composant `SessionAirlock.vue` affiché quand `syncState === 'ready'`
  - [ ] Afficher infos contextuelles: titre session, participants présents, morceau en cours (si disponible)
  - [ ] Bouton principal "Start Listening" / "Sync Now" (CTA primaire, couleur accent Amber)
  - [ ] Design selon spec UX: pochette floutée, typographie Fraunces pour titre, espacements généreux
  - [ ] Feedback loading pendant `syncState === 'syncing'` (spinner + texte "Connecting...")

- [ ] Frontend: UI Live (écran Synced) (AC: 2, 4)
  - [ ] Basculement UI vers mode "Live" après succès
  - [ ] Indicateur visuel d'état "Synced" (badge vert, icône, ou barre de statut)
  - [ ] Afficher Now Playing actif (pochette nette/colorée, infos track, progression)
  - [ ] Liste participants avec statut sync de chacun (Ready vs Synced)
  - [ ] Transition animée Sas → Live (fade, scale, ou slide) avec respect `prefers-reduced-motion`

- [ ] Frontend: Gestion erreurs sync (AC: 5)
  - [ ] Afficher message d'erreur si startListening échoue
  - [ ] Bouton "Retry" disponible après erreur
  - [ ] Log erreur pour debug (console + optionnel: Sentry)
  - [ ] Messages spécifiques selon type erreur (`SPOTIFY_NOT_CONNECTED`, `RATE_LIMITED`, etc.)

- [ ] Tests Backend (AC: tous)
  - [ ] Test `POST /sync/start` avec participant valide → `sync_state = 'synced'`
  - [ ] Test `POST /sync/start` avec participant sans tokens Spotify → `SPOTIFY_NOT_CONNECTED`
  - [ ] Test `GET /me` retourne `sync_state` correct
  - [ ] Test broadcast événement `PARTICIPANT_SYNC_STATE_CHANGED` lors de transition
  - [ ] Test permissions: seul le participant peut changer son propre état

- [ ] Tests Frontend (AC: tous)
  - [ ] Test affichage Sas quand `syncState === 'ready'`
  - [ ] Test clic "Start Listening" déclenche API et change état → "synced"
  - [ ] Test affichage écran Live quand `syncState === 'synced'`
  - [ ] Test persistance état après refresh (sessionStorage)
  - [ ] Test affichage erreur si sync échoue

## Dev Notes

### UX "Airlock/Sas" Pattern

**Contexte:**
L'autoplay audio est bloqué par les navigateurs modernes (Chrome, Firefox, Safari) sans interaction utilisateur préalable. De plus, rejoindre une session et se retrouver immédiatement avec du son peut être surprenant/intrusif (surtout si volume élevé).

**Solution:**
Un écran intermédiaire ("Sas" ou "Airlock") qui:

1. Donne le contrôle à l'utilisateur (pas de surprise sonore)
2. Contourne le blocage autoplay (clic = interaction explicite)
3. Valide le contexte audio (utilisateur conscient qu'il va écouter)
4. Permet inspection pré-sync (voir qui est là, quel morceau)

**Design Pattern (selon UX spec):**

- **Mode "Zen" initial**: Pochette floutée, infos session en overlay, gros bouton central CTA
- **Métaphore**: Comme "Join Voice" dans Discord - acte volontaire d'entrer dans l'espace audio
- **Transition**: De floue/N&B/statique → nette/colorée/animée après activation

### États Sync Participant

**États possibles:**

- `ready`: Participant dans la session, pas encore synchronisé audio
- `syncing`: Transition en cours (appel backend, init Spotify)
- `synced`: Synchronisation active, participant écoute en temps réel
- `error`: Erreur lors de l'initialisation (tokens Spotify, rate limit, etc.)

**Persistance:**

- Backend: colonne `sync_state` dans `session_participants` (source de vérité)
- Frontend: `sessionStorage` pour UX (éviter retour au Sas après F5)
- Réconciliation: au mount, GET état serveur pour sync client/serveur

### Backend: Table Migration

Ajouter colonne à `session_participants`:

```sql
-- migrations/YYYYMMDDHHMMSS_add_sync_state.sql
-- +goose Up
ALTER TABLE session_participants ADD COLUMN sync_state VARCHAR(20) NOT NULL DEFAULT 'ready';
CREATE INDEX idx_session_participants_sync_state ON session_participants(sync_state);

-- +goose Down
ALTER TABLE session_participants DROP COLUMN sync_state;
```

**Valeurs autorisées:** `ready`, `synced`
**Transition:** Uniquement `ready` → `synced` pour MVP (pas de retour arrière sauf leave/rejoin)

### Backend: Endpoints

**POST /api/sessions/:sessionId/sync/start**

- **Auth:** Cookie-session (require participant)
- **Contrôle accès:** Participant de la session uniquement
- **Logic:**
  1. Vérifier token Spotify valide/rafraîchir si nécessaire
  2. Si tokens absents/invalides → erreur `SPOTIFY_NOT_CONNECTED`
  3. Lire état player Spotify actuel (GET `/me/player`) pour baseline
  4. Mettre à jour `session_participants.sync_state = 'synced'`
  5. Broadcaster événement WS `PARTICIPANT_SYNC_STATE_CHANGED`
- **Response Success (200):**

  ```json
  {
    "syncState": "synced",
    "nowPlaying": {
      "trackId": "spotify:track:...",
      "trackName": "Song Title",
      "artist": "Artist Name",
      "isPlaying": true,
      "positionMs": 45000,
      "durationMs": 240000
    }
  }
  ```

- **Response Errors:**
  - `401 UNAUTHENTICATED`: pas de cookie-session
  - `403 FORBIDDEN`: pas participant de la session
  - `409 SPOTIFY_NOT_CONNECTED`: tokens Spotify absents/invalides
  - `503 SPOTIFY_UNAVAILABLE`: erreur API Spotify (rate limit, player inactif)

**GET /api/sessions/:sessionId/me**

- **Auth:** Cookie-session (require participant)
- **Logic:** Retourner info participant actuel (userId, role, syncState, lastSeenAt)
- **Response Success (200):**

  ```json
  {
    "userId": "user_abc123",
    "role": "participant",
    "syncState": "ready",
    "lastSeenAt": "2026-01-28T12:34:56Z"
  }
  ```

### Backend: Événement WebSocket

**PARTICIPANT_SYNC_STATE_CHANGED:**
Broadcasté à tous les participants de la session quand un participant change d'état sync.

```json
{
  "type": "PARTICIPANT_SYNC_STATE_CHANGED",
  "sessionId": "sess_abc123",
  "eventSeq": 43,
  "sentAt": "2026-01-28T12:35:00.123Z",
  "payload": {
    "userId": "user_xyz",
    "syncState": "synced",
    "timestamp": "2026-01-28T12:35:00Z"
  }
}
```

### Frontend: Store Pinia

**useSessionStore (ou dédié useSyncStore):**

```typescript
interface SessionSyncState {
  syncState: 'ready' | 'syncing' | 'synced' | 'error'
  error: string | null
}

const store = defineStore('session', {
  state: (): SessionSyncState => ({
    syncState: 'ready',
    error: null
  }),
  actions: {
    async startListening() {
      this.syncState = 'syncing'
      this.error = null
      try {
        const response = await fetch(`/api/sessions/${sessionId}/sync/start`, {
          method: 'POST',
          credentials: 'include'
        })
        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.code || 'SYNC_FAILED')
        }
        const data = await response.json()
        this.syncState = 'synced'
        sessionStorage.setItem('syncState', 'synced')
        // Update nowPlaying avec data.nowPlaying
      } catch (err) {
        this.syncState = 'error'
        this.error = err.message
      }
    },
    async loadSyncState() {
      // Au mount: récupérer état depuis sessionStorage + confirmer avec serveur
      const cached = sessionStorage.getItem('syncState')
      if (cached === 'synced') {
        this.syncState = 'synced'
      }
      // Confirmer avec serveur
      const response = await fetch(`/api/sessions/${sessionId}/me`, {
        credentials: 'include'
      })
      const data = await response.json()
      this.syncState = data.syncState
    }
  }
})
```

### Frontend: Composants

**SessionAirlock.vue (Écran Sas):**

- Affiché quand `syncState === 'ready'`
- Design:
  - Fond: Charcoal `#1A1816`
  - Pochette album (si disponible): floutée 50%, grayscale 100%
  - Titre session: Fraunces, taille XXL
  - Participants présents: avatars ronds, petits
  - Bouton CTA: "Start Listening", Amber `#D97706`, taille L, centré
  - Texte secondaire: "Click to join the listening session"
- États:
  - Normal: bouton enabled
  - Syncing: spinner + texte "Connecting..." + bouton disabled
  - Error: message erreur + bouton "Retry"

**SessionLive.vue (Écran Live):**

- Affiché quand `syncState === 'synced'`
- Design:
  - Pochette album: nette, colorée, grande
  - Indicateur "Live": badge vert ou barre de statut
  - Now Playing: titre, artiste, progression
  - Participants: liste avec état sync de chacun (badge "Synced" vs "Ready")
  - Transition depuis Airlock: fade-in + scale (0.95 → 1.0) sur 300ms

### Intégration avec Story 1.5 (WebSocket)

- Réutiliser le Hub WebSocket existant pour broadcaster `PARTICIPANT_SYNC_STATE_CHANGED`
- Le store realtime doit dispatcher cet événement vers le store session/presence
- Mise à jour UI participantsList pour afficher état sync de chaque participant

### Gestion Erreurs Spotify

**Scénarios d'erreur:**

1. **Tokens absents/invalides:** User n'a pas connecté Spotify ou tokens révoqués
   - Code: `SPOTIFY_NOT_CONNECTED`
   - Message: "Please connect your Spotify account to start listening."
   - Action: Rediriger vers flow OAuth

2. **Rate limit Spotify (429):**
   - Code: `SPOTIFY_RATE_LIMITED`
   - Message: "Spotify is temporarily unavailable. Please try again in a moment."
   - Action: Suggérer retry dans X secondes

3. **Player Spotify inactif/indisponible:**
   - Code: `SPOTIFY_PLAYER_UNAVAILABLE`
   - Message: "No active Spotify device found. Please start Spotify on any device."
   - Action: Expliquer comment activer un device

4. **Erreur réseau/timeout:**
   - Code: `NETWORK_ERROR`
   - Message: "Unable to connect. Please check your connection and try again."
   - Action: Bouton retry

### Performance & UX

**Délai cible sync:** ≤ 3 secondes (NFR2)

- Appel API backend
- Validation tokens Spotify
- Lecture état player
- Broadcast WS aux autres participants
- Update UI

**Optimisations:**

- Précharger infos session (participants, nowPlaying) pendant affichage Sas
- Utiliser loading optimiste (assume success, rollback si erreur)
- Cache tokens Spotify en mémoire backend (éviter DB round-trip à chaque sync)

### Accessibilité

**Conformité WCAG AA:**

- Bouton "Start Listening": ratio contraste ≥ 4.5:1 (Amber sur Charcoal vérifié)
- Focus keyboard: ring visible blanc/gris clair
- Texte état: taille minimum 16px, lisible
- Loading spinner: `aria-label="Connecting to session"`
- Error messages: `role="alert"` pour lecture screen reader

**Reduced Motion:**

- Transitions animées désactivées si `prefers-reduced-motion: reduce`
- Remplacement par fade simple sans scale/slide

### Security Considerations

**Contrôle d'accès:**

- Seul le participant peut changer son propre `sync_state`
- Validation backend: userId depuis cookie-session doit matcher participant
- Pas de possibilité pour un participant de forcer l'état sync d'un autre

**Rate Limiting:**

- Limiter appels `/sync/start` par participant (ex: 1 req/5s) pour éviter spam
- Gérer idempotence: appeler `/sync/start` alors que déjà `synced` → 200 OK (noop)

### Testing Strategy

**Backend Tests:**

- Unit tests: transitions état sync, validation tokens
- Integration tests: flow complet OAuth → join → sync/start
- Mock Spotify API pour tester rate limit, player unavailable, etc.

**Frontend Tests:**

- Composant SessionAirlock: rendering, clic bouton, états loading/error
- Composant SessionLive: rendering après transition
- Store sync: actions startListening, loadSyncState
- Persistance sessionStorage
- Error handling (mock fetch errors)

**E2E Tests (Playwright):**

- User flow complet: join session → voir Sas → clic Start → voir Live
- Refresh page en état Synced → pas de retour au Sas
- Erreur Spotify → message actionnable + retry

## Learnings des Stories Précédentes

**Story 1.1 - Infrastructure:**

- Docker Compose et Caddy configurés, HTTPS/WSS fonctionnel
- Variables d'environnement pour secrets (Spotify client ID/secret)
- Pattern de migrations goose établi

**Story 1.2 - Auth Spotify:**

- OAuth PKCE implémenté, tokens chiffrés en DB (AES-256-GCM)
- Cookie-session fonctionnelle (gorilla/sessions)
- Pattern refresh tokens automatique
- Abstraction `SpotifyClient` pour appels API

**Story 1.3 - Créer session:**

- Modèles `Session`, `SessionParticipant`, `SessionInvite` existants
- `SessionHandler` pour logique métier session
- Pattern création ressource + validation

**Story 1.4 - Rejoindre session:**

- Endpoint `POST /api/sessions/join` avec validation invite
- Middleware `RequireParticipant` pour contrôle d'accès
- Pattern erreurs stables (`SESSION_NOT_FOUND`, `FORBIDDEN`)
- Tests montrent comment mocker DB et sessions

**Story 1.5 - WebSocket:**

- Hub WebSocket fonctionnel (package `internal/realtime`)
- Enveloppe message standard avec `type`, `sessionId`, `eventSeq`, `sentAt`, `payload`
- Pattern broadcast événements à tous participants
- Store Pinia `useRealtimeStore` pour gérer connexion WS
- Détection déconnexion via ping/pong (≤ 5s)

### Code Patterns Établis

**Backend:**

```go
// Handler pattern
func (h *SessionHandler) StartSync(w http.ResponseWriter, r *http.Request) {
  // 1. Extract context (userID from session, sessionID from URL)
  // 2. Validate access (RequireParticipant middleware)
  // 3. Business logic (refresh tokens, call Spotify, update DB)
  // 4. Broadcast WS event (if applicable)
  // 5. Return JSON response
}

// Error response pattern
type ErrorResponse struct {
  Code    string `json:"code"`
  Message string `json:"message"`
}
respondError(w, http.StatusConflict, "SPOTIFY_NOT_CONNECTED", "Spotify not connected")
```

**Frontend:**

```typescript
// Fetch pattern (established in stores)
async apiCall() {
  try {
    const response = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.code)
    }
    return await response.json()
  } catch (err) {
    // Handle error (store state + UI)
  }
}
```

### Files to Create/Modify

**Backend files à créer:**

- `backend/migrations/YYYYMMDDHHMMSS_add_sync_state.sql` (migration)

**Backend files à modifier:**

- `backend/internal/handlers/session.go`: ajouter `StartSync` et `GetParticipantMe`
- `backend/internal/models/session.go`: ajouter champ `SyncState` à `SessionParticipant`
- `backend/internal/realtime/message.go`: ajouter type `PARTICIPANT_SYNC_STATE_CHANGED`
- `backend/cmd/boeuf-server/main.go`: enregistrer routes `/sync/start` et `/me`

**Frontend files à créer:**

- `frontend/src/components/SessionAirlock.vue` (composant Sas)
- `frontend/src/components/SessionLive.vue` (composant Live)
- `frontend/src/components/SyncStateIndicator.vue` (badge état sync, réutilisable)

**Frontend files à modifier:**

- `frontend/src/stores/session.ts`: ajouter `syncState`, `startListening()`, `loadSyncState()`
- `frontend/src/views/SessionView.vue`: afficher SessionAirlock ou SessionLive selon état
- `frontend/src/stores/realtime.ts`: dispatcher événement `PARTICIPANT_SYNC_STATE_CHANGED`

## Project Structure Notes

### Backend Structure (nouveau)

```
backend/
  internal/
    handlers/
      session.go          # Ajouter StartSync, GetParticipantMe
    models/
      session.go          # Ajouter SyncState à SessionParticipant
    realtime/
      message.go          # Ajouter PARTICIPANT_SYNC_STATE_CHANGED
  migrations/
    YYYYMMDDHHMMSS_add_sync_state.sql  # Migration
```

### Frontend Structure (nouveau)

```
frontend/
  src/
    components/
      SessionAirlock.vue      # Écran Sas (nouveau)
      SessionLive.vue         # Écran Live (nouveau)
      SyncStateIndicator.vue  # Badge état sync (nouveau)
    stores/
      session.ts              # Ajouter syncState logic
    views/
      SessionView.vue         # Router Airlock/Live
```

### Intégration avec Architecture Existante

- **Backend:** Réutilise middleware `RequireParticipant`, handlers session, hub WebSocket
- **Frontend:** Intègre avec stores existants (`session`, `realtime`, `presence`)
- **Database:** Extension table existante `session_participants`
- **Communication:** Nouvel endpoint REST + nouvel événement WS

## References

- **Source:** [_bmad-output/planning-artifacts/epics.md](../planning-artifacts/epics.md#story-16-sas-start-listening-anti-autoplay-et-etat-ready--synced) (Story 1.6)
- **Source:** [_bmad-output/planning-artifacts/architecture.md](../planning-artifacts/architecture.md) (API patterns, WebSocket, conventions)
- **Source:** [_bmad-output/planning-artifacts/ux-design-specification.md](../planning-artifacts/ux-design-specification.md#21-defining-experience) (Airlock Pattern, Visual Design)
- **Source:** [_bmad-output/project-context.md](../project-context.md) (Conventions naming, formats, erreurs)
- **Learnings:** [1-4-rejoindre-une-session-via-lien-code-avec-controle-dacces.md](./1-4-rejoindre-une-session-via-lien-code-avec-controle-dacces.md) (Contrôle d'accès, flow join)
- **Learnings:** [1-5-websocket-session-presence-participants-etat-connexion.md](./1-5-websocket-session-presence-participants-etat-connexion.md) (WebSocket Hub, broadcast événements)
- **Spec:** [Web Audio Autoplay Policy](https://developer.chrome.com/blog/autoplay/) (Chrome, autres browsers)
- **Pattern:** [Discord Voice Channels UX](https://support.discord.com/hc/en-us/articles/360045138571) (Inspiration)

## Dev Agent Record

### Agent Model Used

_À remplir lors de l'implémentation_

### Debug Log References

_À remplir lors de l'implémentation_

### Completion Notes List

_À remplir lors de l'implémentation_

### File List

_À remplir lors de l'implémentation_
