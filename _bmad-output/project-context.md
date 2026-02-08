---
project_name: 'boeuf'
user_name: 'Mathias'
date: '2026-01-22'
sections_completed:
  ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'quality_rules', 'workflow_rules', 'anti_patterns']
existing_patterns_found: 12
source_documents:
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/implementation-artifacts/spotify-api-spike.md
status: 'complete'
rule_count: 18
optimized_for_llm: true
---

# Project Context for AI Agents

_Ce fichier est un guide concis et “LLM-friendly” des règles à respecter. Il complète l’architecture et sert de garde-fou contre les oublis et divergences inter-agents._

---

## Technology Stack & Versions

### Backend

- Language: Go
- Database: SQLite (MVP)
- ORM: GORM (v1.31.1)
- Migrations: goose (v3.26.0)
- OAuth: Spotify Authorization Code + PKCE
- Realtime: WebSocket (JSON)
- Spike (référence): `spike-spotify-api/go.mod` utilise Go 1.21 + `golang.org/x/oauth2`

### Frontend

- Framework: Vue 3 (SPA)
- Build: Vite (via `pnpm create vue@latest`)
- State: Pinia
- Router: Vue Router
- Validation: Zod
- UI: Tailwind CSS v4 + shadcn-vue
- Tests: Vitest
- Lint/Format: ESLint + Prettier

### Infrastructure

- Reverse proxy: Caddy (v2.10.2)
- Deploy: Docker Compose
- Persistence: volume Docker pour SQLite
- Config: `.env` consommé par compose

## Critical Implementation Rules

### Language-Specific Rules (TypeScript / Go)

- Ne pas faire dépendre la cohérence temps réel d’horloges client (le serveur arbitre).
- Go (SQLite): prévoir l’exécution avec CGO activé (driver `mattn/go-sqlite3`).

### Framework-Specific Rules (Vue / Backend)

- Le serveur est **source of truth**; `eventSeq` monotone par session.
- Les clients incluent `clientMsgId` pour idempotence/dédoublonnage.
- Reconnexion: **snapshot + events since seq** (pas de “full refresh” implicite sans raison).
- Spotify: pas de webhooks → polling + “Trust but Verify” (relire l’état après commande).
**Database Migrations:**
- **Stratégie MVP:** GORM AutoMigrate (simplifie dev/deploy)
- Migration goose créée pour référence mais non utilisée: [backend/migrations/20260125000001_create_spotify_tokens.sql](backend/migrations/20260125000001_create_spotify_tokens.sql)
- **Pourquoi AutoMigrate?** MVP rapide, moins de setup, migrations gérées par GORM
- **Future:** Migrer vers goose si contrôle versioning SQL devient nécessaire (production multi-env)

### Testing Rules

- Frontend: Vitest; tests co-localisés quand possible.
- **Important:** Toujours lancer les tests frontend en mode `--run` (non-watch) : `npm run test:unit -- --run`
  - Éviter le mode watch qui bloque le terminal et nécessite intervention manuelle (appuyer sur 'q')
  - Pour les agents IA : utiliser systématiquement `--run` pour éviter les processus bloquants
- Backend: ajouter des tests ciblés sur le protocole (ordre/idempotence/resync) et sur la logique rate-limit.
- Backend: `go test ./...` est déjà en mode one-shot (pas de watch par défaut)

### Code Quality & Style Rules

- DB: tables/colonnes en `snake_case` (tables au pluriel).
- JSON: `camelCase`.
- WS: `type` en `UPPER_SNAKE`.
- Erreurs (REST + WS): `code` en `SCREAMING_SNAKE`, payload `{ code, message, details?, trace_id? }`.
- Dates/temps: ISO-8601 UTC (RFC3339).

### Development Workflow Rules

- Respecter les frontières `frontend/` / `backend/` / `deploy/` (domaines de responsabilité).
  - Note : fichiers d'infrastructure (`docker-compose.yml`, `Caddyfile`, `.env.example`) sont à la racine pour ergonomie
  - Le dossier `deploy/` contient la documentation de déploiement
- Garder le contrat REST/WS minimal dans un endroit unique (éviter les duplications divergentes).

### Critical Don't-Miss Rules

- 429 Spotify: respecter `Retry-After` + backoff + jitter.
- `details` d’erreur doit rester stable (utile debug/validation), pas une dump arbitraire.
- Secrets: refresh tokens Spotify chiffrés (AES-256-GCM) en DB; aucun secret persistant côté frontend.

### OAuth & CORS Strategy

**CORS Configuration (Dev vs Prod):**

- **Dev (Docker Compose):** Backend CORS middleware permet `http://localhost:3000` (Caddy proxy)
  - Dev workflow: `docker compose up -d --build` - stack complet (frontend build + backend + Caddy)
  - Alternative: Vite dev server avec proxy (`pnpm dev`) - tests unitaires/composants uniquement
  - **Important:** OAuth flow requiert stack Docker (Caddy :3000) - redirect URI Spotify ne peut pas pointer vers Vite dev
- **Prod:** CORS géré par Caddy reverse proxy headers (à configurer selon domaine)
- Frontend DOIT passer par Caddy (`:3000`) - accès direct backend (`:8080`) bloqué par CORS

**Spotify OAuth Redirect URI:**

- MUST match exactement `SPOTIFY_REDIRECT_URI` env var: `http://localhost:3000/auth/spotify/callback`
- Caddy route `/auth/*` vers backend - pas de réécriture path
- Spotify Dashboard: ajouter redirect URI exacte (protocole + domaine + port + path)
- **Pourquoi pas :8080 direct?** OAuth exige redirect URI publique; backend interne au réseau Docker

**Spotify Scopes Strategy:**

- Scopes minimaux requis: `user-read-playback-state`, `user-read-currently-playing`, `user-modify-playback-state`
- **Identité stable:** `user-read-email` (choisi) - fournit email unique et stable pour lier compte
  - Alternative `user-read-private` donnerait country/subscription mais email suffit pour MVP
  - Email utilisé comme `spotify_user_id` unique dans la DB
- **Rationale:** MVP minimise scopes; email = identifiant stable sans données sensibles supplémentaires

**Testing:**

- E2E tests Playwright pointent vers `localhost:3000` (stack Docker complet)
- Vite dev proxy `/auth` et `/api` vers backend pour dev local (si besoin)

## Defaults & Anti-Patterns

- Ne pas introduire une lib data-fetching/cache lourde (MVP) sans décision explicite.
- Ne pas casser les conventions (DB/JSON/WS) “juste pour aller vite” : si divergence, elle doit être documentée et alignée.
- Ne pas faire dépendre la cohérence temps réel d’horloges client (le serveur arbitre).

## Quick Start (Implementation Phase)

- Initialiser le frontend: `pnpm create vue@latest` (TS + Router + Pinia + Vitest + ESLint + Prettier)
- Ajouter Tailwind v4 + shadcn-vue
- Scaffold backend Go + SQLite (CGO) + goose + GORM
- Définir tôt le contrat REST/WS minimal (endpoints + events + codes d’erreur)

---

## Usage Guidelines

**Pour les agents :**

- Lire ce fichier AVANT d’implémenter une story.
- En cas de doute, préférer l’option la plus restrictive (contrat, sécurité, cohérence).
- Si une décision change, mettre à jour ce fichier ET l’architecture.

**Pour les humains :**

- Garder ce fichier court (règles non-obvies uniquement).
- Réviser périodiquement et supprimer les règles devenues évidentes.

Last Updated: 2026-01-25

## Developer Workflow (Standardized)

Pour réduire la friction, utilisez le `Makefile` à la racine pour toutes les opérations courantes.

### Commandes Principales

| Commande | Description | Contexte |
|---|---|---|
| `make watch` | **Humain Uniquement**. Lance l'environnement avec **Hot Reload** (Front+Back). **Bloquant**. | Développement |
| `make dev-restart` | **Agent Friendly**. Rebuild et démarre les conteneurs en background. Utile pour appliquer des changements. | Développement |
| `make test` | Lance tous les tests (Back + Front) dans Docker. | CI / Check |
| `make test-backend` | Tests Go uniquement (`go test ./...`). | Backend |
| `make test-frontend` | Tests Vue uniquement (`npm run test:unit -- --run`). | Frontend |
| `make prod` | Lance l'environnement en mode production (build optimisé). | Staging |
| `make logs` | Affiche un snapshot des logs (non bloquant). | Debug Agent |
| `make watch-logs` | Affiche les logs en continu (bloquant). | Debug Humain |

### Architecture de Developpement

- **Frontend** : En mode dev, tourne sur une image Node avec Vite en mode HMR. Les changements dans `frontend/src` sont synchronisés instantanément.
- **Backend** : En mode dev, le conteneur peut redémarrer (rebuild) à chaque changement de fichier Go (via `watch` ou `dev-restart`).
- **Proxy** : `Caddyfile.dev` est utilisé pour router les requêtes vers le serveur de dev Vite (port 5173).

### Règles pour l'IA

1. **Ne pas deviner** les commandes npm ou go. Utiliser `make`.
2. **Ne jamais utiliser `make watch`** (c'est une commande bloquante). Utiliser `make dev-restart`.
3. Pour appliquer un changement de code : éditer les fichiers, puis `make dev-restart` (ou juste `make test` si TDD).
4. Si un test échoue, utiliser `make test-backend` ou `make test-frontend` pour isoler.
5. **Frontend Debugging** : Vous pouvez utiliser les outils Chrome DevTools (`mcp_chrome-devtoo_*`) pour inspecter le DOM, la console ou le réseau du frontend accessible sur `http://localhost:3000`.
