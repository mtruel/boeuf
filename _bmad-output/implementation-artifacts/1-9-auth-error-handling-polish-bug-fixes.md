# Story 1.9: Auth & Error Handling Polish (Bug Fixes)

Status: ready-for-dev

## Story

As a utilisateur,
I want clear feedback when authentication expires or device issues occur,
So that je comprends pourquoi les commandes échouent et je sais comment corriger.

## Context

Cette story regroupe les bug fixes identifiés lors de la session de test du 2026-01-30. Tous ces bugs concernent l'authentification, les erreurs de device Spotify, et la gestion des états invalides découverts post-implémentation des Stories 1-1 à 1-7.

**Bugs Fixes:**
- BUG #1: Commandes Player 503 - SPOTIFY_NO_DEVICE
- BUG #2: Session Data Visible Après Logout
- BUG #3: Session Page Accessible Sans Auth
- BUG #4: 401 Errors Sans Feedback UX Après Logout Multi-Tab

## Acceptance Criteria

### AC 1: Pre-check Device Avant Sync Start (BUG #1 - CRITICAL)

- **Given** un utilisateur clique "Démarrer l'écoute" sur la session page
- **When** backend reçoit la requête POST `/api/sessions/{id}/sync/start`
- **Then** le backend DOIT vérifier qu'au moins un device Spotify est disponible AVANT d'accepter le sync
- **And** si aucun device n'est disponible:
  - Retourner 503 avec code `SPOTIFY_NO_DEVICE`
  - Response body DOIT inclure `requiresActiveDevice: true` flag
- **And** si device disponible:
  - Procéder normalement avec sync start
  - Session passe en mode "Live"

### AC 2: UX Toast avec Instructions Device (BUG #1 - CRITICAL)

- **Given** frontend reçoit 503 `SPOTIFY_NO_DEVICE` error
- **When** la response contient `requiresActiveDevice: true`
- **Then** afficher un toast spécifique avec:
  - **Titre:** "No Active Spotify Device"
  - **Message:** "Please start playback in Spotify (Web Player, Desktop, or Mobile) and try again."
  - **Action button:** "Open Spotify Web Player" → lien vers `https://open.spotify.com`
  - **Secondary button:** "Retry"
- **And** le bouton "Démarrer l'écoute" reste enabled pour retry
- **And** la session reste en mode "Airlock" (pas de transition vers Live)
- **Validation:** User peut ouvrir Spotify Web Player, lancer une track, puis retry avec succès

### AC 3: Cache Session Data Après Logout (BUG #2 - MEDIUM)

- **Given** un utilisateur est connecté et a une session active
- **When** l'utilisateur clique "Se déconnecter" (POST `/api/auth/logout`)
- **Then** frontend DOIT immédiatement:
  - Clear le state auth (`isAuthenticated = false`)
  - Clear le state session (sessionId, inviteCode, participants, etc.)
  - Cacher la section "Session" de la home page
- **And** seul le bouton "Connecter Spotify" reste visible
- **And** aucune donnée de session précédente ne doit rester affichée
- **Validation:** Après logout, recharger la page → aucune info session visible

### AC 4: Navigation Guard Session Routes (BUG #3 - MEDIUM)

- **Given** un utilisateur non authentifié (pas de cookie valide)
- **When** il tente de naviguer vers `/session/{sessionId}` directement (URL ou bookmark)
- **Then** Vue Router DOIT intercepter la navigation via `beforeEach` guard
- **And** rediriger automatiquement vers home (`/`)
- **And** afficher un toast: "Please sign in to access this session"
- **And** préserver l'URL de destination dans query param: `/?redirect=/session/{sessionId}`
- **And** après login réussi, rediriger automatiquement vers l'URL préservée
- **Validation:** User non-auth → visite `/session/abc` → redirect home → login → redirect `/session/abc`

### AC 5: Navigation Guard Join Routes (BUG #3 - Extension)

- **Given** un utilisateur non authentifié
- **When** il tente de naviguer vers `/join/{inviteCode}`
- **Then** Vue Router DOIT:
  - Rediriger vers home avec toast "Sign in to join this session"
  - Préserver inviteCode dans query: `/?invite={inviteCode}`
  - Après login, auto-join avec le code préservé
- **Validation:** User clique invite link sans auth → login → auto-join session

### AC 6: Global 401 Interceptor (BUG #4 - MEDIUM)

- **Given** un utilisateur a 2 onglets ouverts (Tab 1 et Tab 2)
- **When** user logout sur Tab 1
- **And** user fait une action sur Tab 2 (ex: click Play)
- **Then** l'axios request échoue avec 401 UNAUTHENTICATED
- **And** l'axios response interceptor DOIT:
  - Logger l'utilisateur out localement (`authStore.logout()`)
  - Rediriger vers home (`router.push('/')`)
  - Afficher toast: "Your session expired. Please sign in again."
  - Clear tous les stores (auth, session, player, presence)
- **Validation:** Multi-tab logout → Tab 2 action → 401 → auto-redirect home avec toast

### AC 7: Consistent 401 Handling Across All Endpoints (BUG #4 - Extension)

- **Given** n'importe quelle requête API échoue avec 401
- **When** l'interceptor détecte le 401
- **Then** le comportement DOIT être identique pour tous les endpoints:
  - `/api/sessions/*` → logout + redirect
  - `/api/auth/*` → logout + redirect
  - `/api/sessions/{id}/player/*` → logout + redirect
- **And** éviter les "toast storms" (max 1 toast "session expired" même si multiple 401)
- **Validation:** Force expire cookie → faire plusieurs actions → voir 1 seul toast

### AC 8: Retry Logic pour Device Check (BUG #1 - Enhancement)

- **Given** user clique "Retry" après avoir activé Spotify
- **When** frontend re-tente POST `/api/sessions/{id}/sync/start`
- **Then** si device maintenant disponible:
  - Requête réussit (200 OK)
  - Session passe en mode "Live"
  - Toast success: "Synced! You're now listening together."
- **And** si toujours pas de device:
  - Afficher à nouveau toast avec instructions
  - Limiter retries automatiques (max 3x avec backoff)

## Tasks / Subtasks

### Backend Tasks

- [ ] Backend: Pre-check Spotify device availability (AC 1)
  - [ ] Dans handler `/sync/start`, appeler `spotifyClient.GetAvailableDevices()` AVANT sync
  - [ ] Si `len(devices) == 0`, retourner 503 avec body:
    ```json
    {
      "code": "SPOTIFY_NO_DEVICE",
      "message": "No active Spotify device found. Please start playback in Spotify.",
      "requiresActiveDevice": true
    }
    ```
  - [ ] Si `devices` disponible, procéder normalement
  - [ ] Log device info pour debug: `device.Name`, `device.Type`, `device.IsActive`

- [ ] Backend: Améliorer error response structure (AC 1)
  - [ ] Ajouter field `requiresActiveDevice` dans error responses
  - [ ] Standardiser structure pour toutes les erreurs device-related
  - [ ] Ajouter `suggestedAction` field (ex: "OPEN_SPOTIFY_WEB_PLAYER")

- [ ] Backend: Validation seek avec metadata sync (extension BUG #6)
  - [ ] Dans handler `/player/seek`, valider `positionMs <= track.DurationMs`
  - [ ] Si invalide, retourner 400 avec metadata correctes dans response
  - [ ] Log warning si validation fail (indicates stale metadata)

### Frontend Tasks

- [ ] Frontend: Toast spécifique Device Error (AC 2)
  - [ ] Détecter response `code === "SPOTIFY_NO_DEVICE"` ET `requiresActiveDevice === true`
  - [ ] Afficher toast custom avec:
    - Title: "No Active Spotify Device"
    - Description: "Please start playback in Spotify (Web Player, Desktop, or Mobile) and try again."
    - Action primary: `<Button href="https://open.spotify.com" target="_blank">Open Spotify Web Player</Button>`
    - Action secondary: `<Button variant="outline" @click="retrySync">Retry</Button>`
  - [ ] Toast duration: `Infinity` (ne pas auto-dismiss)
  - [ ] User doit cliquer "X" ou "Retry" pour fermer

- [ ] Frontend: Retry logic sync start (AC 2, 8)
  - [ ] Fonction `retrySync()` qui:
    - Ferme toast actuel
    - Re-tente POST `/sync/start`
    - Limite retries (max 3x avec backoff 2s, 4s, 8s)
    - Si success → passe en Live mode + toast success
    - Si fail → re-affiche toast avec instructions

- [ ] Frontend: Clear session state on logout (AC 3)
  - [ ] Dans `authStore.logout()`, appeler:
    - `sessionStore.$reset()`
    - `playerStore.$reset()`
    - `presenceStore.$reset()`
    - `realtimeStore.disconnect()`
  - [ ] Ajouter `isAuthenticated` computed dans composants affichant session data
  - [ ] Wrapper section Session home page: `v-if="authStore.isAuthenticated"`

- [ ] Frontend: Navigation guard auth check (AC 4, 5)
  - [ ] Créer `router/guards/auth.guard.ts`:
    ```typescript
    export const authGuard: NavigationGuard = (to, from, next) => {
      const authStore = useAuthStore()
      
      if (to.path.startsWith('/session/') && !authStore.isAuthenticated) {
        next({ path: '/', query: { redirect: to.fullPath } })
        toast.warning('Please sign in to access this session')
        return
      }
      
      if (to.path.startsWith('/join/') && !authStore.isAuthenticated) {
        const inviteCode = to.params.inviteCode
        next({ path: '/', query: { invite: inviteCode } })
        toast.warning('Sign in to join this session')
        return
      }
      
      next()
    }
    ```
  - [ ] Enregistrer dans `router/index.ts`: `router.beforeEach(authGuard)`
  - [ ] Implémenter auto-redirect après login (lire `route.query.redirect` ou `route.query.invite`)

- [ ] Frontend: Global 401 interceptor (AC 6, 7)
  - [ ] Créer `api/interceptors/auth.interceptor.ts`:
    ```typescript
    let sessionExpiredToastShown = false
    
    axios.interceptors.response.use(
      response => response,
      error => {
        if (error.response?.status === 401) {
          const authStore = useAuthStore()
          const router = useRouter()
          
          // Logout localement
          authStore.logout() // déjà appelle $reset() des autres stores
          
          // Toast unique (éviter spam)
          if (!sessionExpiredToastShown) {
            sessionExpiredToastShown = true
            toast.error('Your session expired. Please sign in again.')
            setTimeout(() => { sessionExpiredToastShown = false }, 5000)
          }
          
          // Redirect home
          router.push('/')
        }
        
        return Promise.reject(error)
      }
    )
    ```
  - [ ] Importer et activer dans `main.ts` ou `api/client.ts`

- [ ] Frontend: Debounce multiple 401 toasts (AC 7)
  - [ ] Flag global `sessionExpiredToastShown` avec timeout
  - [ ] Si déjà affiché, skip nouveaux toasts pendant 5s
  - [ ] Éviter "toast storm" si plusieurs requêtes fail simultanément

### Tests

- [ ] Tests Backend (AC 1, 8)
  - [ ] Test `/sync/start` avec no devices → 503 `SPOTIFY_NO_DEVICE`
  - [ ] Test response contient `requiresActiveDevice: true`
  - [ ] Test `/sync/start` avec devices disponibles → 200 OK
  - [ ] Test validation seek: `positionMs > durationMs` → 400
  - [ ] Test error response structure standardisée

- [ ] Tests Frontend (AC 2, 3, 4, 5, 6, 7, 8)
  - [ ] Test toast spécifique device error affiché avec buttons
  - [ ] Test retry sync après activation Spotify → success
  - [ ] Test logout clear tous les stores session
  - [ ] Test section Session cachée après logout
  - [ ] Test navigation `/session/abc` sans auth → redirect home
  - [ ] Test navigation `/join/code` sans auth → redirect home avec invite preserved
  - [ ] Test auto-redirect après login avec query param
  - [ ] Test 401 interceptor → logout + redirect + toast
  - [ ] Test multi-tab 401 → 1 seul toast (pas de spam)

- [ ] Tests E2E (scénarios complets)
  - [ ] Scenario 1: User sans Spotify actif → click "Démarrer l'écoute" → toast device error → ouvre Spotify → retry → success
  - [ ] Scenario 2: User logout Tab 1 → Tab 2 click Play → 401 → redirect home + toast
  - [ ] Scenario 3: User non-auth visite `/session/abc` → redirect home → login → auto-redirect session
  - [ ] Scenario 4: User non-auth clique invite link → login → auto-join

## Dev Notes

### Error Response Standard

Standardiser toutes les error responses API avec structure:

```typescript
interface ApiError {
  code: string // Machine-readable code (ex: "SPOTIFY_NO_DEVICE")
  message: string // Human-readable message
  requiresActiveDevice?: boolean // Flag pour UX spécifique
  suggestedAction?: string // Action recommandée (ex: "OPEN_SPOTIFY_WEB_PLAYER")
  metadata?: Record<string, any> // Infos additionnelles contextuelles
}
```

### Toast UX Patterns

**Device Error Toast (Special):**
- Persistent (pas d'auto-dismiss)
- Dual actions (Open Spotify + Retry)
- Styling distinct (warning variant)
- Icon: 🎵 ou speaker icon

**Session Expired Toast (Standard):**
- Auto-dismiss après 5s
- Single action implicite (redirect déjà effectué)
- Error variant
- Icon: 🔒

### Navigation Guard Flow

```
User visite URL protégée
  ↓
Guard check: isAuthenticated?
  ↓
  NO → Redirect home + preserve URL
  ↓
User login
  ↓
Auth callback check query params
  ↓
  redirect? → Navigate to preserved URL
  invite? → Auto-join session
```

### Multi-Tab Sync Consideration

**Limitation actuelle:** Chaque tab a son propre store Pinia (pas de shared state).
- Tab 1 logout → Tab 2 state reste "authenticated" jusqu'à prochaine requête API
- Solution: 401 interceptor détecte et sync automatiquement

**Future enhancement (pas MVP):** 
- Utiliser `BroadcastChannel` API pour sync logout entre tabs instantanément
- Ou `localStorage` event listener cross-tab

### Spotify Device Detection

**Endpoint Spotify:**
```
GET https://api.spotify.com/v1/me/player/devices
```

**Response sample:**
```json
{
  "devices": [
    {
      "id": "device_id",
      "name": "Chrome (Web Player)",
      "type": "Computer",
      "is_active": true,
      "volume_percent": 100
    }
  ]
}
```

**Empty devices:**
```json
{
  "devices": []
}
```

### Retry Backoff Strategy

```typescript
const RETRY_DELAYS = [2000, 4000, 8000] // 2s, 4s, 8s
let retryCount = 0

const retrySync = async () => {
  if (retryCount >= 3) {
    toast.error('Unable to sync. Please ensure Spotify is playing.')
    return
  }
  
  await delay(RETRY_DELAYS[retryCount])
  retryCount++
  
  try {
    await sessionStore.startSync()
    toast.success('Synced! You're now listening together.')
    retryCount = 0
  } catch (error) {
    if (error.code === 'SPOTIFY_NO_DEVICE') {
      // Re-show toast
    } else {
      toast.error('Sync failed. Please try again.')
    }
  }
}
```

## References

- **Bug Report:** [test-session-bugs-2026-01-30.md](../action-items/test-session-bugs-2026-01-30.md)
- **Related Stories:** 
  - Story 1-2: Auth Spotify OAuth (login/logout base)
  - Story 1-6: Sas "Start Listening" (sync start endpoint)
  - Story 1-7: Synchronisation Play/Pause (player commands)
- **Spotify API Docs:** [Web API - Get Available Devices](https://developer.spotify.com/documentation/web-api/reference/get-a-users-available-devices)

## NFRs (Non-Functional Requirements)

- **Error Messages:** Langue anglaise (document_output_language: English) mais considérer i18n future
- **Performance:** Device check ne doit pas ajouter >500ms latency à sync start
- **Accessibility:** Toasts lisibles par screen readers (ARIA live regions)
- **UX:** "Open Spotify Web Player" doit ouvrir nouvel onglet (target="_blank")
- **Security:** 401 interceptor ne doit PAS logger sensitive data (tokens, user IDs)

## Acceptance Testing Checklist

- [ ] ✅ BUG #1 Fixed: Pre-check device avant sync + toast avec instructions
- [ ] ✅ BUG #2 Fixed: Session data cachée après logout
- [ ] ✅ BUG #3 Fixed: Navigation guard bloque accès session sans auth
- [ ] ✅ BUG #4 Fixed: 401 multi-tab → logout + redirect + toast
- [ ] ✅ All 4 bugs validés par QA avec scénarios E2E
- [ ] ✅ No regressions sur Stories 1-1 à 1-8

---

**Story Owner:** Bob (Scrum Master)  
**Created:** 2026-01-31  
**Epic:** Epic 1 - MVP Core Listening Session  
**Dependencies:** Stories 1-2, 1-6, 1-7 (déjà done)  
**Blocks:** None (polish/bug fixes)
