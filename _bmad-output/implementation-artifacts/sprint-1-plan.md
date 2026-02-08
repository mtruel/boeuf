# Sprint 1 — Squelette + OAuth Spotify + Sessions

Date: 2026-01-23
Projet: boeuf

## Objectif du sprint

Avoir une verticale utilisable “de bout en bout” : un utilisateur peut démarrer l’app (frontend + backend + deploy), se connecter à Spotify via OAuth (PKCE) avec une session cookie serveur, créer une session boeuf et rejoindre via un lien/code avec contrôle d’accès.

## Périmètre (Sprint Backlog)

Stories sélectionnées (Epic 1) :

- 1.1 — Démarrer le squelette exécutable (frontend/backend/deploy)
- 1.2 — Auth Spotify (OAuth Authorization Code + PKCE) avec session cookie
- 1.3 — Créer une session et générer un lien/code d’invitation
- 1.4 — Rejoindre une session via lien/code (avec contrôle d’accès)

Hors périmètre explicite (report) :

- 1.5 WebSocket + présence
- 1.6 Sas “Start Listening”
- 1.7 Sync play/pause
- 1.8 Now Playing
- Epics 2–5

## Hypothèses / décisions

- Instance self-hosted : `SPOTIFY_CLIENT_ID` et `SPOTIFY_CLIENT_SECRET` sont configurés côté serveur (env / `.env` + Docker Compose). Aucun secret persistant côté navigateur.
- La session applicative (cookie HTTP-only/Secure/SameSite) est la base d’auth côté boeuf, réutilisée ensuite pour REST + WS.
- Le backend reste “source of truth” : même si WS n’est pas dans le sprint, on garde ces conventions dès maintenant (JSON camelCase, erreurs stables).

## Découpage recommandé (ordre d’exécution)

1) 1.1 — Squelette exécutable

   - Scaffold `frontend/` (Vue 3 + TS + Router + Pinia + Vitest + ESLint + Prettier) + Tailwind v4 + shadcn-vue
   - Scaffold `backend/` (Go + `/health`)
   - `deploy/` (Caddy + Docker Compose) avec reverse proxy

2) 1.2 — OAuth Spotify + session cookie

   - Endpoint start OAuth + callback
   - Stockage tokens côté serveur, refresh token chiffré (AES-256-GCM), clé via env
   - Abstraction Spotify (client/provider) pour éviter la dépendance directe

3) 1.3 — Créer session + invite

   - Modèle session + invite opaque expirable
   - API create + réponse lien/code

4) 1.4 — Rejoindre session + contrôle d’accès

   - Endpoint join, validation invite
   - Erreurs stables (`SESSION_NOT_FOUND`, `FORBIDDEN`…)

## Definition of Done (DoD) — Sprint

- Les 4 stories sont “done” (ACs respectés) et passent par tests/qualité minimaux.
- Build & run :
  - Dev : frontend démarre (`pnpm dev`), backend démarre (`go run ...`) et `/health` répond.
  - Compose : `docker compose up` lance reverse proxy + backend + frontend.
- Sécurité : aucun secret Spotify persistant côté frontend; refresh token chiffré en DB; variables d’environnement documentées.
- Contrats : conventions d’erreurs et JSON respectées; endpoints documentés minimalement.

## Risques & mitigations

- OAuth PKCE + cookies + proxy : risque de problèmes CORS/redirect → valider tôt un flow complet en dev, puis via Caddy.
- Chiffrement + stockage tokens : risque de mauvaise gestion de clés → wrapper simple + tests ciblés sur encrypt/decrypt.
- Docker + SQLite CGO : risque build/runtime → valider rapidement l’image backend avec CGO activé.
- Dépendances Spotify (rate limit, scopes) : risque d’échec d’appels → scopes minimaux + handling d’erreurs stable.

## Critères de succès

- Un utilisateur peut : lancer l’app, connecter Spotify, créer une session, partager un lien/code, et un invité peut rejoindre (avec refus propre si lien invalide).
