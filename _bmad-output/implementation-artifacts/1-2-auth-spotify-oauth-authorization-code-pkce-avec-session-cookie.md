# Story 1.2: Auth Spotify (OAuth Authorization Code + PKCE) avec session cookie

Status: ready-for-dev

## Story

As a utilisateur,
I want connecter mon compte Spotify via OAuth,
so that boeuf puisse orchestrer la lecture sur mon appareil.

## Acceptance Criteria

1. **Connexion OAuth**
   - **Given** un utilisateur non authentifié
   - **When** il clique sur “Connecter Spotify”
   - **Then** il est redirigé vers Spotify et revient sur boeuf avec une session active
   - **And** le backend stocke les tokens côté serveur (aucun secret durable côté frontend)

2. **Stockage sécurisé refresh token**
   - **Given** un refresh token à stocker
   - **When** le backend le persiste
   - **Then** il est chiffré en base (AES-256-GCM) et la clé n’est jamais exposée au client

3. **Refresh automatique**
   - **Given** un access token expiré ou proche de l’expiration
   - **When** le backend doit appeler Spotify
   - **Then** il rafraîchit automatiquement le token (sans intervention utilisateur)
   - **And** aucune action Spotify n’échoue “silencieusement” à cause d’un token expiré

4. **Abstraction Spotify**
   - **Given** l’intégration Spotify côté backend
   - **When** on implémente la logique d’orchestration
   - **Then** elle est encapsulée derrière une abstraction (ex: interface `StreamingProvider` / `SpotifyClient`)
   - **And** la logique “session/sync” ne dépend pas directement de détails Spotify

## Tasks / Subtasks

- [ ] Définir le contrat “auth app” (cookie-session) (AC: 1)
  - [ ] Ajouter une dépendance session côté backend (ex: cookie + store serveur)
  - [ ] Configurer cookies `HttpOnly`, `Secure` (en prod), `SameSite` raisonnable

- [ ] Implémenter OAuth Spotify Authorization Code + PKCE (AC: 1)
  - [ ] Endpoint `GET /auth/spotify/start` : génère `code_verifier`, `code_challenge=S256`, `state` ; stocke `state` + `code_verifier` côté serveur (session)
  - [ ] Redirige vers `https://accounts.spotify.com/authorize` avec `client_id`, `redirect_uri`, `response_type=code`, `code_challenge_method=S256`, `code_challenge`, `scope`, `state`
  - [ ] Endpoint callback `GET /auth/spotify/callback` : valide `state`, échange `code` via `POST https://accounts.spotify.com/api/token` (form-urlencoded) avec `code_verifier`
  - [ ] Redirige ensuite vers le frontend (ex: `/`) avec un état “connecté” récupérable via API

- [ ] Choisir les scopes minimaux Spotify (AC: 1)
  - [ ] Inclure au minimum : `user-read-playback-state`, `user-read-currently-playing`, `user-modify-playback-state`
  - [ ] Ajouter `user-read-email` ou `user-read-private` si nécessaire pour une identité stable / profil

- [ ] Persister les tokens côté serveur (DB) (AC: 2)
  - [ ] Ajouter SQLite + GORM + goose (si non fait)
  - [ ] Créer une table `spotify_tokens` (snake_case) avec : `spotify_user_id`, `access_token`, `refresh_token_encrypted`, `expires_at`, `scope`, timestamps
  - [ ] Implémenter un utilitaire AES-256-GCM (clé via variable d’environnement serveur), chiffrant le refresh token

- [ ] Refresh automatique (AC: 3)
  - [ ] Wrapper Spotify client : à chaque appel, vérifie expiration, rafraîchit si nécessaire
  - [ ] En cas d’échec refresh (token révoqué), renvoyer une erreur stable (ex: `SPOTIFY_NOT_CONNECTED`) et forcer reconnexion

- [ ] API “status Spotify” pour le frontend (AC: 1)
  - [ ] Endpoint `GET /api/me` ou `GET /api/auth/status` renvoyant l’état connecté + profil minimal
  - [ ] Frontend : bouton “Connecter Spotify” et affichage “Connecté” (sans fuite tokens)

## Dev Notes

### Guardrails sécurité

- PKCE : **ne pas** stocker `code_verifier` côté frontend (pas de LocalStorage). Le conserver côté serveur via la session cookie.
- Chiffrement : refresh token chiffré AES-256-GCM; clé maître serveur via env (32 bytes).
- OAuth : utiliser `state` (CSRF) et vérifier strictement au callback.

### Abstraction Spotify (anti-couplage)

- Créer une interface (ex: `SpotifyClient`) avec méthodes nécessaires (ex: `GetPlaybackState`, `Pause`, `Play`, `SkipNext`) pour éviter que la logique session/sync dépende des détails HTTP.

### Erreurs stables

- Définir un format d’erreur commun `{ code, message, details?, trace_id? }` (REST) dès maintenant.
- Exemples utiles pour ce sprint : `SPOTIFY_OAUTH_FAILED`, `SPOTIFY_NOT_CONNECTED`, `FORBIDDEN`.

## Testing Requirements

- Tests Go ciblés :
  - chiffrement AES-256-GCM (encrypt/decrypt roundtrip)
  - refresh token flow simulé (mock HTTP)
  - validation `state` (callback refuse mismatch)

## References

- Source: `_bmad-output/planning-artifacts/epics.md` (Story 1.2)
- Source: `_bmad-output/planning-artifacts/architecture.md` (PKCE, cookie-session, tokens chiffrés)
- Source: `_bmad-output/project-context.md` (no secrets frontend, conventions)
- Source: `_bmad-output/implementation-artifacts/sprint-1-plan.md` (périmètre sprint)
- Web: Spotify “Authorization Code with PKCE flow” (code_verifier 43–128, `code_challenge_method=S256`, `state`)
- Web: Spotify scopes: `user-read-playback-state`, `user-read-currently-playing`, `user-modify-playback-state`

## Dev Agent Record

### Agent Model Used

GPT-5.2

### Debug Log References

- N/A (story prep)

### Completion Notes List

- Story préparée en privilégiant un PKCE 100% côté serveur (session cookie) pour respecter “aucun secret persistant côté frontend”.
- Validation checklist automatisée indisponible (tâche `validate-workflow.xml` absente) → exigences de qualité intégrées dans Testing + Dev Notes.

### File List

- `_bmad-output/implementation-artifacts/1-2-auth-spotify-oauth-authorization-code-pkce-avec-session-cookie.md`
