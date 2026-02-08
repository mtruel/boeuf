# Story 1.1: Démarrer le squelette exécutable (frontend/backend/deploy)

Status: ready-for-dev

## Story

As a développeur,
I want un squelette exécutable (frontend, backend, proxy),
so that je peux itérer rapidement et valider l’intégration bout-en-bout.

## Acceptance Criteria

1. **Frontend scaffold**
   - **Given** le dépôt sur une machine de dev
   - **When** j’initialise le frontend via `pnpm create vue@latest` (TS + Router + Pinia + Vitest + ESLint + Prettier) dans `frontend/`
   - **Then** l’app frontend démarre en dev (`pnpm dev`) et compile sans erreur
   - **And** les frontières `frontend/` / `backend/` / `deploy/` sont respectées

2. **Backend /health**
   - **Given** un backend Go initial dans `backend/`
   - **When** je lance le serveur (ex: `go run ./cmd/boeuf-server`)
   - **Then** il expose un endpoint `/health` qui répond 200 avec JSON
   - **And** il peut répondre à un appel depuis le frontend (CORS/dev proxy ou même origin via reverse proxy)

3. **Deploy compose + reverse proxy**
   - **Given** une configuration de déploiement MVP
   - **When** je lance `docker compose up`
   - **Then** le reverse proxy (Caddy) sert le frontend et reverse-proxy le backend
   - **And** les communications en prod passent par HTTPS/WSS

4. **Secrets côté serveur uniquement**
   - **Given** une instance self-hosted
   - **When** je configure les secrets serveur via `.env`/variables d’environnement
   - **Then** le backend démarre avec `SPOTIFY_CLIENT_ID` et `SPOTIFY_CLIENT_SECRET` disponibles côté serveur
   - **And** aucun secret Spotify n’est stocké de manière persistante côté navigateur (LocalStorage/IndexedDB)

## Tasks / Subtasks

- [ ] Créer l’arborescence projet (AC: 1)
  - [ ] Créer les dossiers `frontend/`, `backend/`, `deploy/`
  - [ ] Ajouter un `README.md` minimal au root qui documente dev + compose

- [ ] Scaffold frontend Vue 3 (AC: 1)
  - [ ] Exécuter `pnpm create vue@latest` dans `frontend/` avec options : TS + Router + Pinia + Vitest + ESLint + Prettier
  - [ ] Ajouter Tailwind CSS v4 via `tailwindcss` + `@tailwindcss/vite` et configurer `vite.config.ts`
  - [ ] Importer Tailwind dans `frontend/src/style.css` via `@import "tailwindcss";`
  - [ ] Initialiser shadcn-vue (`pnpm dlx shadcn-vue@latest init`) et ajouter 1 composant de test (ex: Button)
  - [ ] Ajouter une page simple qui affiche : état du backend (GET `/health`) + version

- [ ] Scaffold backend Go (AC: 2)
  - [ ] Initialiser module Go dans `backend/` (choisir un module cohérent avec le repo)
  - [ ] Créer un binaire `./cmd/boeuf-server` (ex: `main.go`) et démarrage HTTP
  - [ ] Implémenter `GET /health` qui renvoie `200` avec JSON stable (ex: `{ "status": "ok" }`)
  - [ ] Configurer cookies/sessions en vue des stories suivantes (sans implémentation complète)

- [ ] Intégration FE↔BE en dev (AC: 2)
  - [ ] Choisir l’approche : Vite dev proxy (`/api`) OU CORS en dev
  - [ ] Vérifier qu’un appel frontend -> backend fonctionne en local

- [ ] Déploiement Docker Compose + Caddy (AC: 3)
  - [ ] Créer `deploy/compose.yaml` (ou `docker-compose.yml`) avec services : `caddy`, `backend`, `frontend` (build)
  - [ ] Créer `deploy/Caddyfile` : servir le build frontend statique + `reverse_proxy` vers backend
  - [ ] Vérifier `docker compose up` lance l’ensemble
  - [ ] Prévoir la prod : HTTPS/WSS géré par Caddy (domain requis); en local, documenter le mode HTTP

- [ ] Variables d’environnement + secrets (AC: 4)
  - [ ] Créer un `deploy/.env.example` documentant `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET` (et futures clés : chiffrement tokens)
  - [ ] Vérifier qu’aucun secret n’est nécessaire côté frontend

## Dev Notes

### Guardrails (à ne pas violer)

- Respecter strictement les frontières : `frontend/` / `backend/` / `deploy/`.
- En prod, toutes les communications passent via HTTPS/WSS (Caddy).
- Aucun secret persistant côté navigateur (pas de LocalStorage/IndexedDB pour secrets).
- Convention projet : JSON (REST/WS) en `camelCase`, DB en `snake_case`.

### Structure recommandée (MVP)

- `frontend/` : Vue SPA (Vite)
- `backend/`
  - `cmd/boeuf-server/main.go`
  - `internal/` (packages domaines futurs : auth/session/spotify/...)
- `deploy/`
  - `compose.yaml`
  - `Caddyfile`
  - `.env.example`

### Notes techniques

- Backend SQLite driver : `mattn/go-sqlite3` implique CGO (valider build Docker tôt).
- Préparer l’API à retourner des erreurs stables (format standard défini dans le projet context).

## Testing Requirements

- Frontend : au moins un test Vitest smoke (montage du layout / rendu page).
- Backend : au moins un test Go `httptest` sur `GET /health`.

## References

- Source: `_bmad-output/planning-artifacts/epics.md` (Story 1.1)
- Source: `_bmad-output/planning-artifacts/architecture.md` (Starter Vue + deploy Compose/Caddy)
- Source: `_bmad-output/project-context.md` (règles structure + conventions)
- Source: `_bmad-output/implementation-artifacts/sprint-1-plan.md` (ordre sprint)
- Web: Tailwind “Using Vite” (plugin `@tailwindcss/vite` + `@import "tailwindcss";`)

## Dev Agent Record

### Agent Model Used

GPT-5.2

### Debug Log References

- N/A (story prep)

### Completion Notes List

- Story préparée en respectant la structure `frontend/`/`backend/`/`deploy/` et les contraintes “no secrets in frontend”.
- Validation checklist automatisée indisponible (tâche `validate-workflow.xml` absente) → revue manuelle intégrée dans Dev Notes.

### File List

- `_bmad-output/implementation-artifacts/1-1-demarrer-le-squelette-executable-frontend-backend-deploy.md`
