---
date: 2026-01-30
type: bug-report
status: triaged
priority: medium
test_session: chrome-devtools-manual-testing
bugs_count: 6
resolution_plan: created
updated: 2026-01-31
---

# Bugs Identifiés - Session de Test 2026-01-30

**🎯 RESOLUTION PLAN (Updated 2026-01-31):**

- **BUG #5 & #6** → Added to [Story 1-8](../implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md) (AC 8 & 9)
- **BUG #1, #2, #3, #4** → New [Story 1-9: Auth & Error Handling Polish](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md)

---

## 📊 Test Summary

**Total Bugs:** 6  
**Critical:** 3 (BUG #1, BUG #5, BUG #6)  
**High:** 1 (BUG #2)  
**Medium:** 2 (BUG #3, BUG #4)

**Test Environment:**

- Browser: Chrome 144 (DevTools MCP)
- App: <http://127.0.0.1:3000>
- Backend: Docker Compose (Caddy + Go)
- Test Date: 2026-01-30 22:30-23:00 UTC

## BUG #1: Commandes Player 503 - SPOTIFY_NO_DEVICE

**Priorité:** 🔴 HIGH  
**Statut:** ✅ Triaged → [Story 1-9](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md) AC 1-2  
**Feature Impactée:** Story 1-7 (Synchronisation Play/Pause)

### Symptômes

- Requête POST `/api/sessions/{sessionId}/player/resume` → **503 Service Unavailable**
- Requête POST `/api/sessions/{sessionId}/player/next` → **503 Service Unavailable** (assumé, même pattern)
- Backend response body:

  ```json
  {
    "code": "SPOTIFY_NO_DEVICE",
    "message": "No active Spotify device found"
  }
  ```

### Comportement Observé

1. User lance session → POST `/sync/start` → ✅ 200 OK
2. GET `/player/state` → ✅ 200 OK (retourne track data complète)
3. UI affiche player avec track "Falling Grace" + artwork + contrôles
4. User clique Play → POST `/player/resume` → ❌ 503
5. Frontend retry 3x → tous échouent avec 503
6. UI affiche toast: "Unable to control playback. Please check your connection."
7. Boutons restent disabled

### Logs Backend

```
2026/01/30 22:31:02 POST /api/sessions/.../player/resume → 503 (duration: 0.066s)
Response: {"code":"SPOTIFY_NO_DEVICE","message":"No active Spotify device found"}
```

### Root Cause Hypothèses

1. **Race Condition:** Spotify Web Player était actif lors de `sync/start` mais s'est déconnecté (sleep/idle) avant la commande resume
2. **Device Detection Issue:** Backend n'identifie pas le device Spotify malgré track data disponible
3. **Spotify API Behavior:** L'endpoint `/player/state` cache les données mais `/player/resume` nécessite device actif en temps réel

### Contexte Technique

- **Fichier:** [backend/internal/handlers/player.go](../../backend/internal/handlers/player.go)
- **Spotify API:** Endpoints `PUT /me/player/play` nécessite device actif
- **Frontend Retry Logic:** 3 tentatives avec backoff (présent et fonctionnel)

### Recommendations Fix

#### Option A: Pre-check Device Avant Sync (UX Préventive)

```go
// Dans sync/start handler
devices, err := spotifyClient.GetAvailableDevices()
if err != nil || len(devices) == 0 {
    return 503, {"code": "SPOTIFY_NO_DEVICE", "message": "..."}
}
```

**Pros:** Bloque tôt, UX plus claire  
**Cons:** Call API supplémentaire

#### Option B: Auto-Transfer Playback

```go
// Dans player resume/pause handlers
if err == NoDeviceError {
    devices := spotifyClient.GetAvailableDevices()
    if len(devices) > 0 {
        spotifyClient.TransferPlayback(devices[0].ID)
        // Retry resume
    }
}
```

**Pros:** User-friendly, auto-récupère  
**Cons:** Complexité, peut transférer vers device non-voulu

#### Option C: Better UX Only

- Toast avec lien direct vers Spotify Web Player
- Instructions: "Ouvrez Spotify dans un autre onglet et lancez une track"
- Bouton "Réessayer" après correction

**Recommandation:** Option A + Option C pour MVP

### Steps to Reproduce

1. Navigate to `http://127.0.0.1:3000`
2. Authenticate with Spotify
3. Create session
4. Click "Démarrer l'écoute" (sans Spotify Web Player actif)
5. Click "Play" button
6. Observe: 503 errors, disabled buttons, generic toast

### Expected Behavior

- Soit bloquer `sync/start` avec message clair si pas de device
- Soit récupérer automatiquement en activant un device disponible
- Soit fournir instructions claires pour activer Spotify Web Player

### Actual Behavior

- UI passe en mode "Live" même sans device actif
- Commandes échouent silencieusement (sauf toast générique)
- User bloqué sans comprendre pourquoi

---

## BUG #2: Session Data Visible Après Logout

**Priorité:** 🟡 MEDIUM  
**Statut:** ✅ Triaged → [Story 1-9](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md) AC 3  
**Feature Impactée:** Auth UX / Home Page

### Symptômes

- User clique "Se déconnecter" sur home page
- POST `/api/auth/logout` → 200 OK
- Auth status passe à "Non connecté" + bouton "Connecter Spotify"
- **MAIS:** Section "Session" reste affichée avec données complètes:
  - Lien d'invitation visible
  - Session ID visible
  - Date d'expiration visible
  - Boutons "Ouvrir la session" et "Créer une nouvelle session" actifs

### Expected Behavior

Option A: Cacher complètement la section Session si non authentifié
Option B: Afficher section mais avec message "Connectez-vous pour créer une session"
Option C: Garder visible mais désactiver tous les boutons

### Actual Behavior

Données de session créée avant logout restent visibles et cliquables (mais non-fonctionnelles car pas de cookie valide).

### Recommendation

Option A recommandée pour MVP - clarté UX maximale.

---

## BUG #3: Session Page Accessible Sans Auth

**Priorité:** 🟡 MEDIUM  
**Statut:** ✅ Triaged → [Story 1-9](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md) AC 4-5  
**Feature Impactée:** Session View / Auth Guard

### Symptômes

- User déconnecté (ou cookie expiré)
- Navigate directement vers `/session/{sessionId}`
- Page charge et affiche:
  - "Session d'écoute" (session ID = null)
  - "0 participants"
  - Bouton "Démarrer l'écoute" (cliquable mais non-fonctionnel)

### Console Errors

```
[SessionView] No user ID available
[Session] Cannot start listening: no session ID
```

### Expected Behavior

- Soit redirect vers home avec toast "Connectez-vous d'abord"
- Soit afficher page d'erreur "Session nécessite authentification"
- Soit bloquer au niveau router (navigation guard)

### Actual Behavior

Page charge en état dégradé sans UX claire pour l'utilisateur.

### Recommendation

Implémenter navigation guard dans Vue Router:

```typescript
// router/index.ts
router.beforeEach((to, from, next) => {
  if (to.path.startsWith('/session/') && !authStore.isAuthenticated) {
    next({ path: '/', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})
```

---

## BUG #4: 401 Errors Sans Feedback UX Après Logout Multi-Tab

**Priorité:** 🟡 MEDIUM  
**Statut:** ✅ Triaged → [Story 1-9](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md) AC 6-7  
**Feature Impactée:** Multi-tab Behavior / Error Handling

### Symptômes

**Scenario:**

1. User a 2 onglets ouverts:
   - Tab 1: Home page (authentifié)
   - Tab 2: Session page active (Live mode)
2. User clique "Se déconnecter" sur Tab 1
3. Tab 2 reste sur session page
4. User clique Play/Pause sur Tab 2
5. Requêtes échouent:
   - 3x 503 SPOTIFY_NO_DEVICE
   - 3x **401 UNAUTHENTICATED**
6. Toast générique: "An error occurred. Please try again."

### Expected Behavior

- Après 401, redirect automatique vers home avec toast "Session expirée, reconnectez-vous"
- OU WebSocket détecte déconnexion et met à jour UI
- OU Global error handler intercepte 401 et trigger re-auth flow

### Actual Behavior

User reste sur page session avec état invalide, reçoit toast générique, aucune indication que l'auth a expiré.

### Network Evidence

```
POST /player/pause → 401
Response: {"code":"UNAUTHENTICATED","message":"Authentication required"}
```

### Recommendation

Implémenter global axios interceptor:

```typescript
// api/client.ts
axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      authStore.logout()
      router.push({ path: '/', query: { error: 'session_expired' } })
    }
    return Promise.reject(error)
  }
)
```

---

## Tests Réussis ✅

Pour référence, les features suivantes fonctionnent correctement:

### Auth & Backend

- ✅ GET `/api/auth/status` → User authentifié
- ✅ GET `/api/health` → Backend online
- ✅ Cookie session persisté

### Session Management

- ✅ POST `/api/sessions` → Création session
- ✅ UI affiche invite link + session ID + expiration
- ✅ Bouton "Ouvrir la session" → Navigation correcte

### Invitation Flow

- ✅ Route `/join/{inviteCode}` → Redirect automatique
- ✅ POST `/api/sessions/join` → 200 OK
- ✅ Navigation vers `/session/{sessionId}` après join
- ✅ Bouton "📋 Copier" fonctionne

### WebSocket

- ✅ Connection WS établie: `ws://127.0.0.1:3000/ws/{sessionId}`
- ✅ Stores Pinia initialisés (realtime, presence, session, player)
- ✅ Console logs: client registered, connection established

### Session Sync Start

- ✅ POST `/api/sessions/{id}/sync/start` → 200
- ✅ GET `/api/sessions/{id}/player/state` → 200
- ✅ UI transition "Airlock" → "Live" fonctionnelle
- ✅ Track data affiché: artwork, titre, artiste, album, duration, position
- ✅ Participants list affichée: count, user ID, role (Host), status (Synced), connection indicator

### Error Handling UX

- ✅ Toast "No active Spotify device found. Please start Spotify."
- ✅ Toast "Unable to control playback. Please check your connection."
- ✅ Boutons disabled pendant requêtes en cours
- ✅ Dismiss button sur toasts fonctionnel

---

## Prochains Tests à Effectuer

1. **Player Commands (avec device actif)**
   - [ ] Click Play → résumé lecture
   - [ ] Click Pause → pause lecture
   - [ ] Click Next → track suivante
   - [ ] Vérifier events WebSocket PLAYER_* reçus

2. **Real-time Sync**
   - [ ] Observer position progression automatique
   - [ ] Vérifier slider mise à jour temps réel
   - [ ] Tester commandes depuis Spotify Web Player → sync app

3. **Multi-participant**
   - [ ] Ouvrir 2nd client avec invite link
   - [ ] Vérifier presence updates (2 participants)
   - [ ] Tester commandes depuis host → propagation guest
   - [ ] Tester commandes depuis guest (si permissions)

4. **Déconnexion & Reconnexion**
   - [ ] Click "Se déconnecter"
   - [ ] Vérifier redirection + cookie cleared
   - [ ] Reconnecter avec même compte
   - [ ] Vérifier session state préservée

5. **WebSocket Resilience**
   - [ ] Kill connection (Dev Tools Network)
   - [ ] Vérifier reconnexion automatique
   - [ ] Vérifier snapshot resync après reconnexion

6. **Edge Cases**
   - [ ] Session expirée (après 24h - simuler avec DB edit?)
   - [ ] Invite code invalide
   - [ ] Session déjà full (max participants)
   - [ ] Backend down → retry logic

---

## Notes Techniques

### Environment de Test

- **Date:** 2026-01-30 ~22:30 UTC
- **Browser:** Chrome 144.0.0.0 (Linux x64)
- **URL:** `http://127.0.0.1:3000` (CRITICAL: utiliser 127.0.0.1, pas localhost)
- **Backend:** Docker Compose dev stack
- **Spotify Account:** 11160204402

### Outils Utilisés

- Chrome DevTools MCP Server
  - Network tab: monitoring requêtes
  - Console: stores Pinia logs
  - Snapshots: état DOM accessibility tree

---

## BUG #5: Current Time Gelé - Pas de Progression Automatique

**Priorité:** 🔴 CRITICAL  
**Statut:** ✅ Triaged → [Story 1-8](../implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md) AC 8  
**Feature Impactée:** Story 1-8 (Afficher Now Playing)

### Symptômes

- Player affiche track data: "Fingerprints" à **4:11/5:18**
- **Aucune progression automatique** du current time observée après 5+ secondes
- Time reste gelé à `4:11` malgré Spotify Web Player actif en arrière-plan
- Slider de progression (value=251408ms) ne se met pas à jour automatiquement

### Expected Behavior

- Current time devrait s'incrémenter automatiquement chaque seconde
- Slider devrait progresser visuellement en temps réel
- Frontend devrait poll `/player/state` périodiquement OU recevoir WebSocket updates

### Actual Behavior

```
Initial state: 4:11 / 5:18
After 5s:      4:11 / 5:18  ❌ (no change)
After 10s:     4:11 / 5:18  ❌ (still frozen)
```

### Context

- Session en mode "Live" (POST /sync/start → 200)
- WebSocket connecté: `ws://127.0.0.1:3000/ws/sess_...`
- GET `/player/state` réussit avec track data
- Spotify Web Player progresse normalement (vérifié sur onglet séparé: 4:11 → 4:20)

### Root Cause Hypothèses

1. **No Polling:** Frontend ne poll pas périodiquement `/player/state` pour updates
2. **No WebSocket Updates:** Backend n'envoie pas d'événements `PLAYER_POSITION_UPDATE` via WebSocket
3. **Client-Side Simulation Missing:** Frontend pourrait simuler progression localement entre les updates serveur

### Recommendations

**Option A: Client-Side Simulation (Optimal pour UX)**

```typescript
// player.ts store
let progressInterval: NodeJS.Timeout

function startProgressSimulation() {
  progressInterval = setInterval(() => {
    if (state.isPlaying && state.trackPosition < state.trackDuration) {
      state.trackPosition += 1000 // increment 1 seconde
    }
  }, 1000)
}

function stopProgressSimulation() {
  clearInterval(progressInterval)
}
```

**Option B: Server-Driven Polling**

```typescript
// Poll player state every 5s when playing
const pollInterval = setInterval(async () => {
  if (state.isPlaying) {
    await fetchPlayerState()
  }
}, 5000)
```

**Option C: WebSocket Position Events**

```go
// Backend broadcast position updates every 5s
ticker := time.NewTicker(5 * time.Second)
go func() {
  for range ticker.C {
    if session.IsPlaying {
      broadcastPlayerPosition(session)
    }
  }
}()
```

**Recommandation:** **Option A** (simulation client) + **Option B** (sync périodique) pour UX fluide + sync précise

### Impact

- ⚠️ **User Experience Brisée:** Impression que player est gelé/broken
- ⚠️ **Sync Accuracy:** Impossible de vérifier visuellement la sync entre participants
- ⚠️ **Story 1-8 Non Complète:** Spec demande "mise à jour temps réel" non implémentée

---

### Commandes Utiles

```bash
# Logs backend
docker logs boeuf-backend-1 --tail 100

# Logs Caddy
make logs | tail -50

# Restart stack
make dev-restart

# Tests
make test
```

---

## 🐛 BUG #6: Seek Rejected - Wrong Track Metadata (CRITICAL)

**Priority:** 🔴 **CRITICAL**  
**Status:** ✅ Triaged → [Story 1-8](../implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md) AC 9  
**Story Impact:** 1-7, 1-8  
**Discovered:** 2026-01-30 23:02 UTC

### Symptômes

1. **Seek fail avec 400 Bad Request:**

   ```
   POST /api/sessions/sess_.../player/seek
   → 400 Bad Request
   ```

2. **Error response:**

   ```json
   {
     "code": "INVALID_POSITION",
     "message": "positionMs (150000) exceeds track duration (130466)"
   }
   ```

3. **Frontend affiche fausses métadonnées:**
   - Affichage: "Fingerprints" (5:18 = 318026 ms)
   - Réalité backend: Track de 2:10 (130466 ms)
   - Slider max=318026 mais track max=130466

4. **User tente seek → toast error "An error occurred"**

### Network Trace

```
reqid=250 POST /api/sessions/.../player/seek
Status: 400 Bad Request

Request Body:
{"clientMsgId":"02e074cc-de4e-4cc0-9dbc-4ddacbe164b1","positionMs":150000}

Response Body:
{"code":"INVALID_POSITION","message":"positionMs (150000) exceeds track duration (130466)"}
```

### Console Logs

```
error> Failed to seek: {}
at player.ts:332:20
```

### Root Cause

**Désynchronisation métadonnées frontend/backend:**

1. Frontend affiche track metadata obsolètes ou incorrectes
2. Slider range basé sur les mauvaises valeurs (valuemax=318026 au lieu de 130466)
3. Backend valide avec les vraies métadonnées Spotify
4. User interaction (seek) fail car frontend envoie positionMs hors limites

**CONFIRMATION UTILISATEUR (2026-01-31):**
> "J'ai changé de morceau dans spotify et cela n'a pas été retranscris dans l'app"

**État constaté:**

- Spotify joue: "Paint The World" (3:58) - Chick Corea Elektric Band II
- Boeuf affiche: "Fingerprints" (5:18) - The Chick Corea New Trio
- **Les changements de track sur Spotify ne sont PAS propagés à Boeuf**

**Causes possibles:**

- GET /player/state retourne données périmées (cache?)
- WebSocket track_changed event PAS IMPLÉMENTÉ ou pas reçu/traité
- Aucun polling périodique des métadonnées Spotify
- Backend ne notifie pas le frontend lors des track changes
- Race condition: frontend charge avant track metadata update

### Impact

- **User Experience:** Impossible de seek dans la track (feature cassée)
- **Trust:** UI affiche infos incorrectes, erode user confidence  
- **Story 1-7:** Sync play/pause fail → position invalide
- **Story 1-8:** "Afficher morceau, artiste, progression" → **fausses données affichées**
- **Critical:**
  - Feature seek complètement inutilisable
  - **Changements de track sur Spotify INVISIBLES dans Boeuf**
  - Users doivent recharger page pour voir current track
  - Sync multi-participant impossible si metadata désynchronisées

### Reproduction Steps

1. Créer session et start listening (Mode Live)
2. Observer track metadata affichés (ex: "Fingerprints" 5:18)
3. **CHANGER de track directement dans Spotify Web Player**
4. **Observer: Boeuf continue d'afficher l'ancienne track**
5. Tenter de seek en déplaçant slider vers 2:30 (150000ms)
6. Backend rejette: duration réelle est 2:10 (130466ms)
7. Toast error apparaît

**OU:**

1. Start session, track "Fingerprints" s'affiche
2. User change de track dans Spotify → "Paint The World" joue
3. Boeuf affiche toujours "Fingerprints" (metadata jamais mises à jour)
4. Slider, duration, position basés sur mauvaise track

### Recommendations

**Option A - Fix Metadata Sync (prioritaire):**

```go
// Garantir GET /player/state retourne vraies données Spotify
func (h *PlayerHandler) GetPlayerState() {
    spotifyState := h.spotify.GetCurrentPlayback()
    // Store et return TRUE current track metadata
    return PlayerState{
        Track: spotifyState.Item,
        Duration: spotifyState.Item.DurationMs,
        Position: spotifyState.ProgressMs,
        IsPlaying: spotifyState.IsPlaying,
    }
}
```

**Option B - Frontend Validation:**

```ts
// Bloquer seek si metadata invalides ou duration suspecte
const handleSeek = (newPositionMs: number) => {
  if (newPositionMs > player.track.durationMs) {
    console.warn('Invalid seek position, fetching fresh state...')
    await playerStore.refreshState()
    return
  }
  await playerStore.seek(newPositionMs)
}
```

**Option C - Backend Error Response UX:**

```go
// Retourner metadata correctes dans error response
return c.JSON(400, ErrorWithMetadata{
    Code: "INVALID_POSITION",
    Message: "Position exceeds track duration",
    CurrentTrack: {
        Name: track.Name,
        DurationMs: track.DurationMs,
    },
})
```

**Option D - WebSocket Sync:**

```
// Envoyer track_metadata_updated event
{
  "type": "track_metadata_updated",
  "track": {
    "uri": "spotify:track:...",
    "name": "...",
    "artists": [...],
    "duration_ms": 130466
  }
}
```

### Notes

- BUG #5 (time frozen) pourrait masquer ce problème si time n'évolue jamais
- Suggests GET /player/state cache ou staleness issue
- Frontend slider constraints must match backend validation rules
- Critical car TOUTE interaction player (seek, progress display) impactée

---

## ✅ Action Items (Updated 2026-01-31)

### Stories Created

- ✅ **Story 1-8:** Enhanced with AC 8 (BUG #5) and AC 9 (BUG #6)
  - AC 8: Progression temps réel automatique avec interpolation client-side
  - AC 9: Sync métadonnées lors de changements de track + validation seek
  - [View Story 1-8](../implementation-artifacts/1-8-afficher-now-playing-morceau-artiste-progression-mise-a-jour-temps-reel.md)

- ✅ **Story 1-9:** Created for BUG #1, #2, #3, #4
  - AC 1-2: Pre-check device + UX toast with Spotify link (BUG #1)
  - AC 3: Clear session data after logout (BUG #2)
  - AC 4-5: Navigation guards for session routes (BUG #3)
  - AC 6-7: Global 401 interceptor multi-tab (BUG #4)
  - [View Story 1-9](../implementation-artifacts/1-9-auth-error-handling-polish-bug-fixes.md)

### Next Steps

- [ ] **DEV (Amelia):** Implement Story 1-8 (includes BUG #5, #6 fixes)
- [ ] **DEV (Amelia):** Implement Story 1-9 (includes BUG #1, #2, #3, #4 fixes)
- [ ] **QA (Quinn):** Validate all 6 bugs resolved after implementation
- [ ] **QA (Quinn):** Re-run full test session scenarios
- [ ] **SM (Bob):** Update sprint-status.yaml when stories move to in-progress/done

---

**Dernière mise à jour:** 2026-01-31 (Triaged + Stories Created)  
**Testeur:** Party Mode Team (Quinn, Amelia, Winston, Mary, Dr. Quinn)  
**Scrum Master:** Bob
