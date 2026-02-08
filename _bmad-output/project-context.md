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

### Testing Rules

- Frontend: Vitest; tests co-localisés quand possible.
- Backend: ajouter des tests ciblés sur le protocole (ordre/idempotence/resync) et sur la logique rate-limit.

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

Last Updated: 2026-01-22
