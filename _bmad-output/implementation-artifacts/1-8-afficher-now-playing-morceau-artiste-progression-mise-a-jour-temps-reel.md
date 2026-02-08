# Story 1.8: Afficher "Now Playing" (morceau, artiste, progression) + mise à jour temps réel

Status: ready-for-dev

## Story

As a utilisateur,
I want voir le morceau en cours (et une progression indicative),
So that je sache ce que le groupe écoute.

## Acceptance Criteria

1. **Affichage Now Playing complet**
   - **Given** une session active avec un morceau en cours de lecture
   - **When** un participant consulte l'UI
   - **Then** il voit les informations suivantes affichées:
     - Pochette de l'album (image, taille appropriée)
     - Titre du morceau (Fraunces font selon UX spec)
     - Nom de l'artiste (Inter font, style muted)
     - Durée totale du morceau (format mm:ss)
   - **And** la mise en page respecte le design "Center Stage" (pochette + infos centrées)

2. **Barre de progression temps réel**
   - **Given** une session active avec lecture en cours
   - **When** le morceau progresse
   - **Then** une barre de progression s'affiche montrant:
     - Position actuelle dans le morceau (format mm:ss)
     - Barre visuelle de progression (0-100%)
     - Durée totale du morceau
   - **And** la progression se met à jour de manière fluide (pas de saccades)
   - **And** la mise à jour se fait localement (interpolation côté client) entre les sync serveur

3. **Mise à jour lors de changement de track**
   - **Given** plusieurs participants connectés à une session
   - **When** le morceau change (skip, fin naturelle, ou ajout queue)
   - **Then** tous les participants reçoivent un événement WebSocket `TRACK_CHANGED`
   - **And** l'UI de tous les participants se met à jour immédiatement avec les nouvelles infos track
   - **And** la transition visuelle est fluide (cross-fade pochette sur 300ms selon UX spec)
   - **And** la barre de progression reset à 0 pour le nouveau morceau

4. **Mise à jour lors de play/pause**
   - **Given** un participant change l'état de lecture (play/pause)
   - **When** l'événement WebSocket correspondant arrive (`PLAYER_PAUSED` ou `PLAYER_RESUMED`)
   - **Then** l'UI Now Playing reflète le nouvel état:
     - État "playing": Progression continue, pochette pleine opacité
     - État "paused": Progression figée, pochette dimmed (opacity 0.7, grayscale 50% selon UX spec)
   - **And** le changement d'état visuel est immédiat

5. **Sync position avec état serveur**
   - **Given** le serveur broadcaste régulièrement l'état player (via polling Spotify)
   - **When** un événement WebSocket contient une mise à jour de position (`positionMs`)
   - **Then** le client recalibre sa progression locale si l'écart dépasse un seuil (ex: >2s)
   - **And** la recalibration est fluide (pas de saut brutal)
   - **And** les timestamps échangés sont en RFC3339 UTC (NFR requirement)

6. **États visuels spéciaux**
   - **Given** une session sans morceau en cours
   - **When** l'UI charge
   - **Then** un état "Empty" s'affiche avec:
     - Placeholder art (faint vinyl groove pattern selon UX spec)
     - Message CTA: "The silence is loud. Add a track."

   - **Given** le player charge un nouveau morceau
   - **When** les métadonnées ne sont pas encore disponibles
   - **Then** un état "Loading" s'affiche avec:
     - Skeleton loader pour pochette/texte (Shadcn Skeleton)
     - Animation de chargement subtile

7. **Feedback tab en arrière-plan**
   - **Given** l'utilisateur a l'onglet boeuf en arrière-plan
   - **When** le morceau change ou l'état lecture change
   - **Then** le titre de l'onglet se met à jour dynamiquement:
     - Format playing: `▶ {Titre} - {Artiste}`
     - Format paused: `⏸ Paused`
   - **And** le favicon affiche un badge vert quand la session est "Live/Synced"

8. **Fix: Progression temps réel automatique (BUG #5)**
   - **Given** une session active avec lecture en cours (`isPlaying === true`)
   - **When** l'utilisateur observe le player pendant plusieurs secondes
   - **Then** la position actuelle (`currentTime`) DOIT s'incrémenter automatiquement chaque seconde
   - **And** l'interpolation locale DOIT être implémentée côté client (pas uniquement polling serveur)
   - **And** le slider de progression DOIT refléter visuellement l'avancement en temps réel
   - **And** le formatage du temps affiché (mm:ss) DOIT se mettre à jour continuellement
   - **Validation**: Observer la progression pendant 10 secondes → `currentTime` doit passer de 4:11 à 4:21

9. **Fix: Sync métadonnées lors de changements de track (BUG #6)**
   - **Given** un utilisateur change de track directement dans Spotify Web Player
   - **When** le backend détecte le changement via polling Spotify API
   - **Then** un événement WebSocket `TRACK_CHANGED` DOIT être envoyé avec les nouvelles métadonnées complètes:
     - `trackId`, `trackName`, `artist`, `albumArt`, `durationMs`
   - **And** le frontend DOIT mettre à jour immédiatement toutes les informations affichées:
     - Titre du morceau
     - Nom de l'artiste
     - Pochette de l'album
     - Durée totale du morceau
     - Position réinitialisée à 0
   - **And** les métadonnées du slider DOIT être mises à jour (`valuemax` = nouvelle `durationMs`)
   - **And** toute tentative de seek DOIT utiliser les métadonnées à jour
   - **Validation**: Backend DOIT rejeter toute requête seek où `positionMs > durationMs` actuelle
   - **Validation**: Changer de track dans Spotify → observer dans Boeuf sous 5-10s (délai polling)

## Tasks / Subtasks

- [ ] Backend: Polling régulier état Spotify player (AC: 2, 5, 8, 9)
  - [ ] Job/goroutine périodique par session active (ex: toutes les 5-10s)
  - [ ] Appeler Spotify GET `/me/player` pour tous les participants "synced"
  - [ ] Détecter changements d'état: track, position, isPlaying, device
  - [ ] **CRITICAL (AC 9):** Comparer `trackId` pour détecter changement de track
  - [ ] Broadcaster événements WS uniquement si changement détecté (éviter spam)
  - [ ] Gestion rate limiting Spotify (backoff si 429)
  - [ ] Stop polling si tous les participants sont déconnectés ou session inactive

- [ ] Backend: Événements WebSocket état player (AC: 1, 2, 3, 4, 5, 8, 9)
  - [ ] Événement `PLAYER_STATE_UPDATE` avec payload complet:
    - `trackId`, `trackName`, `artist`, `albumArt` (URL)
    - `isPlaying` (boolean)
    - `positionMs` (number)
    - `durationMs` (number)
    - `timestamp` (RFC3339 UTC)
  - [ ] Broadcast régulier (via polling) pour sync position (AC 8)
  - [ ] **CRITICAL (AC 9):** Événement `TRACK_CHANGED` enrichi avec métadonnées complètes lors de changement de `trackId`
  - [ ] Réutiliser événements `PLAYER_PAUSED`/`PLAYER_RESUMED` de Story 1.7

- [ ] Backend: Enrichissement snapshot initial (AC: 1)
  - [ ] Ajouter état "now playing" complet dans `SESSION_SNAPSHOT`
  - [ ] Inclure même structure que `PLAYER_STATE_UPDATE`
  - [ ] Gérer cas où aucun morceau n'est actif (null/empty)

- [ ] Frontend: Store player state (AC: 1, 2, 3, 4, 5, 8, 9)
  - [ ] Étendre `usePlayerStore` (ou créer si absent) avec:
    - `currentTrack`: `{ id, name, artist, albumArt, durationMs }`
    - `isPlaying`: boolean
    - `positionMs`: number (position serveur)
    - `lastUpdateAt`: timestamp (pour interpolation locale)
  - [ ] Action `updatePlayerState(payload)` appelée lors des événements WS
  - [ ] **CRITICAL (AC 9):** Action `updateTrackMetadata(track)` pour changements de track
  - [ ] Computed `progressPercent`: calculé depuis position/duration
  - [ ] Computed `positionFormatted` et `durationFormatted` (mm:ss)

- [ ] Frontend: Interpolation locale progression (AC: 2, 5, 8)
  - [ ] **CRITICAL (AC 8):** Composable `usePlayerProgress()` qui:
    - Lit `positionMs` et `lastUpdateAt` du store
    - Calcule position locale interpolée via `requestAnimationFrame` ou interval
    - **Incrémente automatiquement position chaque seconde si `isPlaying === true`**
    - Recalibre si écart serveur > seuil (2s)
  - [ ] Retourne `currentPositionMs` réactif pour l'UI
  - [ ] Gère pause: fige l'interpolation
  - [ ] **Test validation:** Observer progression pendant 10s → temps avance automatiquement
  - [ ] Cleanup lors du unmount

- [ ] Frontend: Composant `MusicPlayerDisplay.vue` (AC: 1, 2, 6)
  - [ ] Affichage pochette album (image `albumArt` URL)
  - [ ] Shadow sur pochette (selon UX spec)
  - [ ] Titre morceau (Fraunces font, animated scroll si trop long)
  - [ ] Artiste (Inter font, text-muted)
  - [ ] Barre de progression avec position actuelle/totale
  - [ ] États visuels:
    - `Empty`: placeholder art + message CTA
    - `Loading`: Skeleton (Shadcn)
    - `Playing`: full opacity, shadow active
    - `Paused`: dimmed (opacity 0.7, grayscale filter 50%)
  - [ ] Respect layout "Center Stage" (centré, espacements généreux)

- [ ] Frontend: Composant `ProgressBar.vue` (AC: 2, 5)
  - [ ] Barre visuelle de progression (0-100%)
  - [ ] Affichage temps: `{current} / {total}` (mm:ss)
  - [ ] Update fluide via interpolation locale
  - [ ] Interactive on hover (pour Host, future enhancement - pas MVP)
  - [ ] Utiliser Shadcn `Progress` comme base
  - [ ] Animations fluides (CSS transitions)

- [ ] Frontend: Gestion événements WebSocket (AC: 3, 4, 5, 9)
  - [ ] Écouter `PLAYER_STATE_UPDATE` → `updatePlayerState()`
  - [ ] **CRITICAL (AC 9):** Écouter `TRACK_CHANGED` → `updateTrackMetadata()` + trigger transition
  - [ ] Écouter `PLAYER_PAUSED` → set `isPlaying = false`
  - [ ] Écouter `PLAYER_RESUMED` → set `isPlaying = true`
  - [ ] **CRITICAL (AC 9):** Détecter changement de track (différent `trackId`) → déclencher cross-fade + reset position
  - [ ] **Validation (AC 9):** Update slider `valuemax` avec nouvelle `durationMs`

- [ ] Frontend: Transition visuelle changement track (AC: 3)
  - [ ] Cross-fade pochette (300ms selon UX spec)
  - [ ] Fade-out ancienne image, fade-in nouvelle
  - [ ] Update métadonnées texte pendant la transition
  - [ ] Respect `prefers-reduced-motion` (pas d'animation si désactivé)

- [ ] Frontend: Feedback tab arrière-plan (AC: 7)
  - [ ] Composable `useTabTitle()` qui watch l'état player
  - [ ] Update `document.title` dynamiquement:
    - `▶ {trackName} - {artist}` si playing
    - `⏸ Paused` si paused
    - `boeuf - {sessionName}` si idle
  - [ ] Update favicon dynamiquement (badge vert si synced)
  - [ ] Utiliser bibliothèque `@vueuse/core` (`useTitle`, `useFavicon`) si disponible

- [ ] Frontend: Gestion état Empty (AC: 6)
  - [ ] Détecter `currentTrack === null` ou `trackId === ""`
  - [ ] Afficher placeholder art (vinyl groove pattern SVG)
  - [ ] Message CTA cliquable qui ouvre panneau social/search (future)
  - [ ] Style selon UX spec (subtle, dark)

- [ ] Frontend: Gestion état Loading (AC: 6)
  - [ ] Détecter transition track (ancien track → nouveau track loading)
  - [ ] Afficher Skeleton loader (Shadcn) pour pochette + textes
  - [ ] Animation pulse subtile pour album art
  - [ ] Timeout: si loading > 5s, afficher erreur

- [ ] Tests Backend (AC: tous)
  - [ ] Test job polling: appelle Spotify GET `/me/player` régulièrement
  - [ ] Test détection changement: broadcast si track/position/état change
  - [ ] **CRITICAL (AC 9):** Test détection changement de `trackId` → broadcast `TRACK_CHANGED` avec metadata complètes
  - [ ] Test pas de broadcast si aucun changement (éviter spam WS)
  - [ ] Test format événement `PLAYER_STATE_UPDATE` (tous champs requis)
  - [ ] Test timestamps en RFC3339 UTC
  - [ ] Test snapshot initial contient état now playing
  - [ ] Test gestion 429 rate limit Spotify (backoff)
  - [ ] Test stop polling si session inactive
  - [ ] **CRITICAL (AC 9):** Test validation seek: reject si `positionMs > durationMs`

- [ ] Tests Frontend (AC: tous)
  - [ ] Test affichage Now Playing avec données complètes
  - [ ] **CRITICAL (AC 8):** Test interpolation locale progression automatique (observer 10s → temps avance)
  - [ ] **CRITICAL (AC 8):** Test position s'incrémente chaque seconde quand `isPlaying === true`
  - [ ] Test recalibration si écart serveur > seuil
  - [ ] **CRITICAL (AC 9):** Test changement track → métadonnées mises à jour (titre, artiste, artwork, duration)
  - [ ] **CRITICAL (AC 9):** Test slider `valuemax` updated avec nouvelle `durationMs`
  - [ ] Test transition track: cross-fade + update métadonnées
  - [ ] Test états visuels Empty/Loading/Playing/Paused
  - [ ] Test update titre tab dynamiquement
  - [ ] Test respect `prefers-reduced-motion` pour animations

## Dev Notes

### Architecture Now Playing

**Backend - Polling Strategy:**

- Un job périodique par session active (goroutine dédiée)
- Fréquence: 5-10s (balance entre fraîcheur et rate limiting)
- Appelle Spotify Player API pour chaque participant "synced" (ou juste le host pour MVP)
- Compare état reçu avec état précédent (cache en mémoire)
- Broadcast WebSocket uniquement si changement détecté
- Inclure `timestamp` serveur (RFC3339 UTC) pour calcul drift côté client

**Backend - Event Structure:**

```json
{
  "type": "PLAYER_STATE_UPDATE",
  "sessionId": "sess_abc123",
  "eventSeq": 42,
  "sentAt": "2026-01-28T14:30:00Z",
  "payload": {
    "trackId": "spotify:track:xyz",
    "trackName": "So What",
    "artist": "Miles Davis",
    "albumArt": "https://i.scdn.co/image/...",
    "isPlaying": true,
    "positionMs": 125000,
    "durationMs": 320000,
    "timestamp": "2026-01-28T14:30:00Z"
  }
}
```

**Frontend - Interpolation Locale:**

Le serveur envoie la position toutes les 5-10s, mais l'UI doit montrer une progression fluide (60fps idéalement). Solution: interpolation locale.

```typescript
// Pseudo-code composable usePlayerProgress()
const interpolatedPosition = ref(0)
let animationFrame: number

const startInterpolation = () => {
  const startPosition = store.positionMs
  const startTime = Date.now()
  
  const update = () => {
    if (!store.isPlaying) return
    
    const elapsed = Date.now() - startTime
    interpolatedPosition.value = startPosition + elapsed
    
    // Vérifier si on dépasse la durée totale
    if (interpolatedPosition.value >= store.currentTrack.durationMs) {
      interpolatedPosition.value = store.currentTrack.durationMs
      return // Arrêter, en attente du prochain track
    }
    
    animationFrame = requestAnimationFrame(update)
  }
  
  update()
}

// Recalibrer si écart serveur > 2s
watch(() => store.positionMs, (newPos) => {
  const drift = Math.abs(newPos - interpolatedPosition.value)
  if (drift > 2000) {
    interpolatedPosition.value = newPos
  }
})
```

**Frontend - Component Hierarchy:**

```
SessionView.vue
└─ MusicPlayerDisplay.vue (Hero component)
   ├─ AlbumArt.vue (pochette avec états visuels)
   ├─ TrackInfo.vue (titre + artiste avec animations)
   └─ ProgressBar.vue (barre + timestamps)
```

### UX Implementation Details

**Center Stage Layout:**

```css
.music-player-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2rem; /* 32px spacing généreux */
  padding: 4rem; /* 64px selon UX spec */
}

.album-art {
  width: 320px;
  height: 320px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  transition: opacity 300ms, filter 300ms;
}

.album-art.paused {
  opacity: 0.7;
  filter: grayscale(50%);
}

.track-title {
  font-family: 'Fraunces', serif; /* Display font */
  font-size: 2rem;
  font-weight: 600;
  text-align: center;
}

.track-artist {
  font-family: 'Inter', sans-serif; /* Body font */
  font-size: 1.25rem;
  color: var(--muted); /* text-muted */
  text-align: center;
}
```

**Cross-fade Transition:**

```typescript
// Animation pochette lors de changement track
const transitionTrack = async (oldTrack, newTrack) => {
  // Fade out
  albumArtRef.value.style.opacity = '0'
  await wait(150) // Half of 300ms
  
  // Swap image
  albumArtSrc.value = newTrack.albumArt
  
  // Fade in
  await nextTick()
  albumArtRef.value.style.opacity = '1'
}
```

**Respect prefers-reduced-motion:**

```css
@media (prefers-reduced-motion: reduce) {
  .album-art {
    transition: none; /* Pas d'animation */
  }
  
  .progress-bar {
    animation: none; /* Pas de pulse si buffering */
  }
}
```

### Backend: Job Polling Implementation

```go
// internal/session/polling.go

type PlayerPoller struct {
    hub         *realtime.Hub
    spotify     spotify.Client
    sessionRepo repository.SessionRepository
    interval    time.Duration
}

func (p *PlayerPoller) StartPolling(ctx context.Context, sessionID string) {
    ticker := time.NewTicker(p.interval) // 5-10s
    defer ticker.Stop()
    
    var lastState *PlayerState
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Récupérer host ou premier participant synced
            participant := p.getActiveSyncedParticipant(sessionID)
            if participant == nil {
                continue // Pas de participant actif, skip
            }
            
            // Appeler Spotify
            currentState, err := p.spotify.GetPlayerState(participant.UserID)
            if err != nil {
                // Log erreur, continue polling
                continue
            }
            
            // Comparer avec état précédent
            if hasChanged(lastState, currentState) {
                // Broadcaster événement
                event := buildPlayerStateUpdateEvent(sessionID, currentState)
                p.hub.BroadcastToSession(sessionID, event)
                
                lastState = currentState
            }
        }
    }
}
```

### Frontend: Tab Title Management

```typescript
// composables/useTabTitle.ts
import { watch } from 'vue'
import { usePlayerStore } from '@/stores/player'
import { useTitle } from '@vueuse/core'

export function useTabTitle() {
  const playerStore = usePlayerStore()
  const title = useTitle('boeuf')
  
  watch(
    () => [playerStore.currentTrack, playerStore.isPlaying],
    ([track, isPlaying]) => {
      if (!track) {
        title.value = 'boeuf'
        return
      }
      
      if (isPlaying) {
        title.value = `▶ ${track.name} - ${track.artist}`
      } else {
        title.value = `⏸ Paused`
      }
    }
  )
}
```

### Testing Strategy

**Backend Tests:**

- Mock Spotify API responses (track metadata)
- Simulate polling cycle avec états changeants
- Vérifier broadcast WebSocket uniquement si changement
- Tester rate limiting (mock 429)

**Frontend Tests:**

- Mock WebSocket events avec différents payloads
- Vérifier interpolation locale via `vi.useFakeTimers()`
- Tester transitions visuels avec `@vue/test-utils`
- Snapshot tests pour états Empty/Loading/Playing/Paused

### Project Structure Notes

**Backend Files:**

```
backend/internal/
├── session/
│   ├── polling.go           # NEW: Job polling Spotify
│   └── polling_test.go
├── realtime/
│   ├── events.go            # EXTEND: Add PLAYER_STATE_UPDATE
│   └── hub.go               # EXTEND: Broadcast methods
└── spotify/
    ├── player.go            # EXTEND: GetPlayerState()
    └── player_test.go
```

**Frontend Files:**

```
frontend/src/
├── components/
│   ├── MusicPlayerDisplay.vue     # NEW: Hero component
│   ├── AlbumArt.vue               # NEW: Pochette avec états
│   ├── TrackInfo.vue              # NEW: Titre + artiste
│   └── ProgressBar.vue            # NEW: Barre progression
├── composables/
│   ├── usePlayerProgress.ts       # NEW: Interpolation locale
│   └── useTabTitle.ts             # NEW: Tab title management
├── stores/
│   └── player.ts                  # EXTEND: Add Now Playing state
└── views/
    └── SessionView.vue            # EXTEND: Integrate MusicPlayerDisplay
```

### References

- [Source: [_bmad-output/planning-artifacts/epics.md](/_bmad-output/planning-artifacts/epics.md#L332)] - Story 1.8 definition et AC
- [Source: [_bmad-output/planning-artifacts/architecture.md](/_bmad-output/planning-artifacts/architecture.md#L42-82)] - Architecture WebSocket + événements ordonnés
- [Source: [_bmad-output/planning-artifacts/ux-design-specification.md](/_bmad-output/planning-artifacts/ux-design-specification.md#L245-280)] - Component `MusicPlayerDisplay` specs
- [Source: [_bmad-output/planning-artifacts/ux-design-specification.md](/_bmad-output/planning-artifacts/ux-design-specification.md#L355-390)] - Feedback patterns et loading states
- [Source: [_bmad-output/implementation-artifacts/1-5-websocket-session-presence-participants-etat-connexion.md](/_bmad-output/implementation-artifacts/1-5-websocket-session-presence-participants-etat-connexion.md)] - Architecture WebSocket existante
- [Source: [_bmad-output/implementation-artifacts/1-7-synchronisation-play-pause-evenements-ordonnes-idempotence.md](/_bmad-output/implementation-artifacts/1-7-synchronisation-play-pause-evenements-ordonnes-idempotence.md)] - Événements player existants (PAUSED/RESUMED/TRACK_CHANGED)
- [Source: NFR1] - Propagation ≤3s, NFR3 - Updates ≤2s
- [Source: Additional Requirements] - Timestamps RFC3339 UTC, polling Spotify (no webhooks), Trust but Verify

### Integration with Previous Stories

**Story 1.5 (WebSocket):**

- Réutilise l'infrastructure Hub pour broadcaster événements
- Suit la même enveloppe message standard (type/sessionId/eventSeq/sentAt/payload)
- Étend les types d'événements avec `PLAYER_STATE_UPDATE`

**Story 1.6 (Airlock):**

- Le Now Playing s'affiche après transition "Ready → Synced"
- En mode Ready (Sas), pochette est floutée (cf. UX spec)
- En mode Synced, pochette devient nette et progression active

**Story 1.7 (Play/Pause Sync):**

- Réutilise événements `PLAYER_PAUSED` et `PLAYER_RESUMED`
- Complète avec affichage visuel de l'état (pochette dimmed si paused)
- Étend avec événement `TRACK_CHANGED` déjà défini
- Ajoute polling régulier pour sync continu (complément aux commandes manuelles)

### Performance Considerations

**Polling Frequency:**

- 5s: Bon équilibre fraîcheur/rate limiting pour MVP
- À ajuster selon retours utilisateurs et monitoring rate limits Spotify

**WebSocket Bandwidth:**

- Broadcaster uniquement si changement détecté (éviter spam)
- Position update: ~100 bytes/message, ~0.2 KB/s par participant si 5s interval
- Acceptable pour session 2-5 personnes (< 1 KB/s total)

**Frontend Performance:**

- `requestAnimationFrame` pour interpolation (60fps) optimisé par navigateur
- Limiter DOM updates: utiliser `v-memo` sur composants stables
- Lazy load images pochette (native `loading="lazy"`)

### Trust but Verify Application

Cette story incarne le principe "Trust but Verify":

- Backend poll régulièrement l'état réel Spotify (verify)
- Frontend fait confiance aux événements serveur mais interpole localement (smooth UX)
- Recalibration automatique si drift détecté (verify wins over interpolation)
- L'UI affiche toujours l'état vérifié, pas juste l'intention

### Fallback & Error States

**Spotify API indisponible:**

- Continuer à afficher dernier état connu
- Indicateur "Syncing..." dans status badge
- Retry polling avec backoff

**Pochette album non disponible:**

- Fallback sur placeholder générique (vinyl groove pattern)
- Pas d'erreur bloquante

**Track metadata incomplète:**

- Afficher champs disponibles
- Placeholder pour champs manquants (ex: "Unknown Artist")

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.5 (via GitHub Copilot)

### Story Generation Context

Generated in YOLO mode (fully automated) as part of Epic 1 completion, following comprehensive analysis of:

- Epic 1 stories 1-5, 1-6, 1-7 (WebSocket, Airlock, Sync foundation)
- Architecture decisions (WebSocket events, Spotify polling, Trust but Verify)
- UX Design Specification (MusicPlayerDisplay component, Center Stage layout, visual states)
- PRD requirements (FR27, NFR1, NFR3, timestamps RFC3339)

### Completion Notes List

- ✅ Story structure follows template with comprehensive AC
- ✅ All tasks mapped to specific ACs with source references
- ✅ Integration points with stories 1-5, 1-6, 1-7 clearly documented
- ✅ Dev notes include implementation details, code samples, and architecture
- ✅ UX specifications fully integrated (fonts, spacing, transitions, states)
- ✅ Performance and error handling considerations included
- ✅ Testing strategy covers backend polling and frontend interpolation
- ✅ Ready for immediate development with no ambiguity

### File List

Story document: [_bmad-output/implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md](_bmad-output/implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md)
