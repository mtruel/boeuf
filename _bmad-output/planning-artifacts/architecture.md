---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/planning-artifacts/brainstorming-session-2026-01-14.md
  - _bmad-output/implementation-artifacts/spotify-api-spike.md
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-01-22'
project_name: 'boeuf'
user_name: 'Mathias'
date: '2026-01-22'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements (architectural implications):**

- **Session management (FR1–FR5)**: création/join via lien/code, présence participants, cycle de vie session.
- **Spotify integration (FR6–FR11)**: OAuth2, lecture état player/track/position, contrôle play/pause/skip/seek, lecture/écriture queue.
- **Playback sync (FR12–FR17)**: propagation temps réel des actions, cohérence “groupe”, tolérance de dérive ≤ 3s.
- **Host management (FR18–FR20)**: host tournant “qui agit devient host”, failover transparent.
- **Resilience (FR21–FR24)**: détection déconnexion rapide, reconnexion + resynchronisation, continuité pour les autres.
- **Conflict resolution (FR25–FR26)**: arbitrage simple (first/last wins), pas de verrouillage lourd.
- **UI observability (FR27–FR29)**: now playing, queue partagée, état participants.

**Non-Functional Requirements (drivers):**

- **Performance**: action → propagation ≤ 3s, WebSocket RTT < 500ms, UI interactive < 3s.
- **Reliability**: détection déconnexion ≤ 5s; reconnexion + resync ≤ 10s; continuité de session.
- **Security**: tokens Spotify stockés de manière sécurisée; HTTPS/WSS; contrôle d’accès par session; expiration sessions 24h.
- **Integration**: gestion rate limits Spotify (429), refresh tokens, tolérance aux réponses ambiguës (ex: pause 403 mais effet réel).
- **Accessibility**: base clavier + contraste, labels, “reduced motion”.

**Scale & Complexity:**

- Domaine primaire: **web app temps réel + orchestration API externe (Spotify)**
- Complexité: **medium** (temps réel + multi-utilisateurs + OAuth + résilience + intégration tierce)
- Indicateurs majeurs: WebSockets, synchronisation multi-clients, polling Spotify, gestion conflits/failover, UX “airlock”.
- Composants architecturaux estimés: **~8–12** (SPA, hub temps réel, services session/présence, intégration Spotify, stockage tokens, persistance session+événements, job/polling, observabilité/monitoring).

### Technical Constraints & Dependencies

- Spotify Web API (Premium requis pour Player API), OAuth2 Authorization Code + refresh tokens.
- Pas de notifications push Spotify (pas de webhooks) → détection de changements via **polling**.
- Rate limiting Spotify non documenté précisément → gestion robuste des `429` + backoff.
- UX “Trust but Verify”: boeuf complète Spotify (états, failover, secours), ne remplace pas le player natif.

### Cross-Cutting Concerns Identified

- Temps réel: diffusion d’événements, ordre, idempotence, reconnexion WebSocket.
- Cohérence: arbitrage des actions concurrentes + convergence d’état (player/queue).
- Résilience: host failover, resync, tolérance aux erreurs intermittentes Spotify.
- Sécurité: gestion tokens, séparation des permissions, isolation par session.
- Observabilité: logs d’événements (au moins pour debug), métriques latence, erreurs Spotify, churn de connexions.

## Starter Template Evaluation

### Primary Technology Domain

Web application (SPA) with real-time features (WebSockets) + external API orchestration (Spotify), based on PRD + UX spec.

### Starter Options Considered

1) **create-vue (official Vue scaffolding, Vite-based)**

- Pros: setup complet et “best practices” (TS/router/tests/lint) via prompts, bon fit pour une SPA.
- Cons: choix interactifs à trancher dès le début.

1) **create-vite (Vue + TS minimal)**

- Pros: ultra minimal.
- Cons: il faut ajouter manuellement router/tests/lint/format, donc plus de décisions “diffuses”.

1) **Nuxt (Nuxt 4)**

- Pros: framework très complet.
- Cons: SSR/fullstack et conventions plus lourdes que nécessaire pour un MVP “SPA + Go backend”.

### Selected Starter: create-vue (Vue SPA, Vite-based)

**Rationale for Selection:**

- Le PRD cible une SPA (pas de SSR/SEO) + temps réel.
- On veut réduire le nombre de micro-décisions et standardiser (tests/lint/router) dès le départ.
- Compatible avec Tailwind v4 et shadcn-vue.

**Initialization Command:**

```bash
pnpm create vue@latest
# sélections recommandées:
# - TypeScript: Yes
# - Vue Router: Yes
# - Pinia: Yes
# - Vitest: Yes
# - ESLint: Yes
# - Prettier: Yes
```

- Tailwind CSS v4 (Vite plugin): installer `tailwindcss` + `@tailwindcss/vite`, configurer `vite.config.ts`, importer `@import "tailwindcss";`
- shadcn-vue: exécuter `pnpm dlx shadcn-vue@latest init` puis ajouter des composants via `pnpm dlx shadcn-vue@latest add ...`

**Architectural Decisions Provided by Starter:**

- **Language & Runtime:** Vue + TypeScript, build Vite.
- **Code Organization:** structure de projet Vue standard + routing.
- **State management:** Pinia.
- **Testing:** Vitest.
- **Lint/Format:** ESLint + Prettier.
- **DX:** HMR, scripts dev/build, config TS cohérente.

**Note:** ce starter couvre le frontend SPA. Le backend Go (OAuth Spotify + WebSocket hub) sera scaffoldé séparément.

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**

- **SQLite (MVP) + `mattn/go-sqlite3` (CGO)** pour la persistance backend.
- **ORM:** GORM (v1.31.1).
- **Migrations:** goose (v3.26.0).
- **Event log append-only** par session (base de debug + resync).
- **Auth Spotify:** Authorization Code + **PKCE**.
- **Sécurité tokens:** refresh tokens **chiffrés en DB (AES-256-GCM)**.
- **Auth app:** **cookie-session** (HTTP-only/Secure/SameSite), réutilisée côté WebSocket.
- **Accès session:** invite opaque (code/lien) stockée côté serveur, expirable.
- **API:** REST JSON + WebSocket JSON.
- **Temps réel:** ordre serveur via `event_seq` monotone par session + idempotence `client_msg_id`.
- **Rate limits Spotify:** respect `Retry-After` + backoff + jitter (pilotage “global” par session).

**Important Decisions (Shape Architecture):**

- **Frontend data fetching:** `fetch` natif + wrapper léger (pas de lib cache lourde MVP).
- **Validation client:** Zod.
- **WebSocket client (MVP):** géré dans Pinia.
- **Stores Pinia:** découpage par domaine (session/player/presence/health/ui).
- **Observabilité UI:** timeline d’événements + indicateurs (drift / état “trust but verify”).
- **Déploiement MVP:** Docker Compose + Caddy + volume persistant.
- **Observabilité serveur:** logs structurés + métriques minimales.

**Deferred Decisions (Post-MVP):**

- Migration Postgres, cache avancé, CI/CD automatisé, monitoring full-stack (Prometheus/Grafana), event sourcing “strict”, multi-région.

### Data Architecture

- **Database:** SQLite (MVP), fichier sur volume Docker persistant.
- **Driver:** `mattn/go-sqlite3` (CGO) pour compatibilité/performances.
- **ORM:** GORM v1.31.1.
- **Migrations:** goose v3.26.0 (migrations SQL).
- **Event log:** table append-only, partitionnée logiquement par `session_id`, avec `event_seq` monotone (source d’audit + resync).
- **Affects:** session lifecycle, présence, sync, debug “trust but verify”.

### Authentication & Security

- **Spotify OAuth:** Authorization Code + PKCE.
- **Token storage:** refresh token chiffré en DB (AES-256-GCM), clé maître via variable d’environnement (secret) injectée dans Docker Compose (`.env`).
- **App auth:** cookie-session (HTTP-only, Secure, SameSite) — utilisée pour HTTP + WS.
- **Session access:** invite opaque serveur (expirable), contrôle d’accès par session.
- **Affects:** endpoints auth, WS handshake, rotation tokens, sécurité.

### API & Communication Patterns

- **HTTP API:** REST JSON.
- **WebSocket:** messages JSON, serveur “source d’autorité”.
- **Ordering / idempotence:** `event_seq` serveur + `client_msg_id` pour dédoublonnage.
- **Resync:** snapshot + events since `event_seq` (reconnexion).
- **Error format standard (HTTP + WS):** `{ code, message, details?, trace_id? }`.
- **Spotify 429:** lecture `Retry-After` + backoff + jitter (global par session), et ré-lecture d’état (“Trust but Verify”) après commandes.

### Frontend Architecture

- **Data fetching:** wrapper `fetch` (headers, cookies, retry léger si besoin, mapping erreurs standard).
- **Validation:** Zod (côté client) sur les payloads critiques (create/join session, commandes).
- **WebSocket client (MVP):** géré dans Pinia (un store “realtime” + dispatch vers stores domaines).
- **State:** Pinia par domaine (session/player/presence/health/ui).
- **UI observability:** timeline des derniers événements (action → ack → état Spotify), plus indicateurs drift/latence.

### Infrastructure & Deployment

- **Hosting:** 1 VPS ou serveur “maison”.
- **Packaging:** Docker (backend + frontend) orchestré via Docker Compose.
- **Reverse proxy:** Caddy v2.10.2.
- **Persistence:** volume Docker pour SQLite (+ éventuels fichiers d’app).
- **Config & secrets:** variables d’environnement, fournies via fichier `.env` consommé par Docker Compose.
- **CI/CD:** déploiement manuel au début.
- **Monitoring/logs:** logs structurés + métriques minimales (latence WS, erreurs Spotify/429, drift sync, connexions).

### Decision Impact Analysis

**Implementation Sequence:**

1) Schéma DB + migrations (sessions, participants, invites, tokens chiffrés, events).
2) OAuth Spotify + PKCE + stockage tokens chiffrés.
3) WebSocket hub + protocole (event_seq, client_msg_id, resync snapshot/events).
4) Polling Spotify + rate limiting (Retry-After/backoff/jitter) + “verify state”.
5) Frontend: stores Pinia + WS + fetch wrapper + Zod + timeline observabilité.
6) Docker Compose + Caddy + volumes + `.env` + logs/métriques.

**Cross-Component Dependencies:**

- `event_seq` + event log ↔ resync WS ↔ UI timeline ↔ debugging sync.
- cookie-session ↔ WS auth ↔ REST endpoints ↔ invite opaque.
- backoff Spotify ↔ UX “trust but verify” ↔ failover/résilience.

## Implementation Patterns & Consistency Rules

### Pattern Categories Defined

**Critical Conflict Points Identified:**

- Nommage DB vs JSON (snake_case vs camelCase)
- Convention des types d’événements (WS) et codes d’erreur
- Formats de dates/temps (API + UI)
- Organisation des packages Go (par domaine)
- Organisation des composants Vue (ui vs métier)
- Emplacement des tests (co-localisés)

### Naming Patterns

**Database Naming Conventions:**

- Tables: **snake_case**, **pluriel**
  - Exemples: `sessions`, `session_participants`, `session_invites`, `spotify_tokens`, `events`
- Colonnes: **snake_case**
  - Exemples: `session_id`, `user_id`, `created_at`, `event_seq`

**API Naming Conventions:**

- JSON (REST + WS): **camelCase**
  - Exemples: `sessionId`, `userId`, `eventSeq`, `clientMsgId`, `createdAt`
- Règle: toute exposition JSON convertit depuis/vers DB (snake_case ↔ camelCase) dans la couche d’API.

**Event & Error Naming Conventions:**

- Types d’événements (WS) en **UPPER_SNAKE**
  - Exemples: `SESSION_CREATED`, `SESSION_JOINED`, `PLAYER_PAUSED`, `QUEUE_UPDATED`, `HOST_CHANGED`
- Codes d’erreurs en **SCREAMING_SNAKE**
  - Exemples: `UNAUTHORIZED`, `FORBIDDEN`, `SESSION_NOT_FOUND`, `SPOTIFY_RATE_LIMITED`, `VALIDATION_FAILED`

### Structure Patterns

**Go Project Organization:**

- Packages Go **par domaine** (séparation stricte)
  - Exemples de domaines: `session`, `realtime`, `spotify`, `storage`, `auth`, `observability`
- Le code HTTP (REST) et WS vit dans une couche d’API dédiée qui dépend des domaines, pas l’inverse.

**Vue Project Organization:**

- Composants UI (shadcn-vue) séparés des composants métier
  - `src/components/ui/` : composants de base (button, dialog, etc.)
  - `src/components/` (ou `src/components/features/`) : composants métier boeuf

**Test Organization:**

- Tests **co-localisés** avec le code
  - Go: `*_test.go` dans les mêmes dossiers
  - Vue: `*.test.ts` proches des modules testés

### Format Patterns

**API Response Formats (REST):**

- Réponses “directes” en succès (pas de wrapper systématique `data:`)
- Erreurs via status codes HTTP + payload standard:
  - `{ code, message, details?, trace_id? }`

**Data Exchange Formats:**

- Dates/temps en **ISO-8601 UTC** (RFC3339)
  - Exemples: `2026-01-22T12:34:56Z`

### Communication Patterns

**WebSocket Message Envelope (JSON):**

- Champs minimum recommandés: `type`, `sessionId`, `eventSeq`, `clientMsgId`, `sentAt`, `payload`
- `type`: UPPER_SNAKE (voir conventions)
- `eventSeq`: entier monotone serveur par session
- `clientMsgId`: identifiant idempotence côté client
- `sentAt`: ISO UTC

**State Management Patterns (Pinia):**

- Les stores restent découplés par domaine (session/player/presence/health/ui)
- Le WS (MVP) est géré dans Pinia et redistribue les événements vers les stores concernés

### Process Patterns

**Error Handling Patterns:**

- Toute erreur REST et WS doit inclure un `code` stable (SCREAMING_SNAKE) et un `message` humain
- `details` est réservé au debug/validation (structure stable)

**Loading State Patterns:**

- États de chargement explicités par domaine (ex: `playerLoading`, `presenceLoading`) plutôt qu’un bool global unique

### Enforcement Guidelines

**All AI Agents MUST:**

- Respecter DB snake_case/pluriel et JSON camelCase (avec mapping explicite)
- Utiliser UPPER_SNAKE pour `type` (WS) et SCREAMING_SNAKE pour `code` (erreurs)
- Émettre des timestamps ISO UTC partout
- Co-localiser tests et code

**Pattern Enforcement:**

- Toute nouvelle API/WS doit fournir un exemple JSON complet (success + error) dans la PR/artefact
- Toute divergence de convention est bloquante tant qu’elle n’est pas documentée et alignée

## Project Structure & Boundaries

### Structure Overview

Le dépôt est organisé en **deux racines** applicatives dans le même repo :

- `frontend/` : SPA Vue (Vite) buildée en assets statiques
- `backend/` : serveur Go (REST + WebSocket)

Un dossier `deploy/` contient les artefacts d’infrastructure (Docker Compose, Caddy, docs de déploiement).

### Complete Project Directory Structure

```
boeuf/
├── deploy/
│   ├── docker-compose.yml
│   ├── Caddyfile
│   └── README.md
├── frontend/
│   ├── package.json
│   ├── pnpm-lock.yaml
│   ├── index.html
│   ├── vite.config.ts
│   ├── public/
│   └── src/
│       ├── api/
│       ├── ws/
│       ├── stores/
│       ├── components/
│       └── pages/
└── backend/
  ├── go.mod
  ├── cmd/
  │   └── boeuf-server/
  ├── internal/
  │   ├── httpapi/
  │   ├── realtime/
  │   ├── spotify/
  │   └── storage/
  └── migrations/
```

### Responsibility Boundaries

**Frontend (`frontend/`)**

- Responsabilités : UI, orchestration client (navigation, états Pinia), gestion WS côté client, appels REST.
- Interdits : logique de “source of truth” serveur (pas de calcul d’`eventSeq`, pas de droits implicites), stockage durable de secrets.
- Communication : REST JSON + WS JSON. Validation stricte des payloads (Zod) au bord.

**Backend (`backend/`)**

- Responsabilités : autorité sur l’état partagé, gestion des sessions/invites, polling Spotify + rate limit, event log append-only, diffusion WS.
- “Source of truth” : `eventSeq` monotone serveur par session, snapshot + resync.

**Deploy (`deploy/`)**

- Responsabilités : reverse-proxy (Caddy), TLS, routage, containerisation, volumes persistants (SQLite).
- Le frontend est servi en **statique** (build Vite) via Caddy; le backend reste derrière Caddy (reverse proxy).

### Build & Run (High-Level)

- Frontend : build Vite → assets statiques
- Backend : binaire Go (CGO activé pour SQLite), exposé en HTTP (REST + WS)
- Infra : Docker Compose orchestre Caddy + backend (+ optionnel: jobs/cron si nécessaire)

## Architecture Validation Results

### Coherence Check

- Architecture cohérente : SPA Vue buildée en statique servie via Caddy + backend Go (REST + WebSocket) + SQLite (GORM/goose) + OAuth Spotify PKCE.
- Les règles de cohérence (naming, formats, enveloppes WS, erreurs) sont alignées avec les besoins de multi-agents.

### Requirements Coverage Check

- FR (sessions/invites/présence) : couvert par les décisions (invites opaques, cookie-session) et le plan REST/WS.
- FR (sync playback + realtime) : couvert via WS + `eventSeq` monotone + resync (snapshot + events since).
- FR (Spotify) : couvert via polling, refresh tokens chiffrés, gestion 429 (`Retry-After` + backoff + jitter), “Trust but Verify”.
- NFR : sécurité (HTTPS/WSS, stockage tokens chiffré), fiabilité (reconnect/resync), performance (propagation WS) et observabilité minimale (logs/metrics) sont adressées au niveau architecture.

### Implementation Readiness Check

- Prêt pour démarrer un MVP si le **contrat API/WS minimal** est figé tôt (endpoints, événements, payloads, codes d’erreur).
- Les frontières `frontend/` / `backend/` / `deploy/` sont suffisamment nettes pour paralléliser l’implémentation.

### Gaps / Risks (to Track)

- Formaliser le catalogue des endpoints REST + événements WS (noms, payloads, erreurs) pour éviter la dérive inter-agents.
- Détail des règles d’arbitrage “host” et des edge cases (actions concurrentes, split-brain, reconnexions).
- SQLite : activer WAL + gérer `busy_timeout`/retry, et confirmer les patterns de transaction pour l’event log.
- Sécurité des secrets : rotation/backup de la clé AES-GCM, politique cookies (SameSite, domain/path) et mitigation CSRF si nécessaire.

## Architecture Completion Summary

### Workflow Completion

**Architecture Decision Workflow:** COMPLETED
**Total Steps Completed:** 8
**Date Completed:** 2026-01-22
**Document Location:** `_bmad-output/planning-artifacts/architecture.md`

### Final Architecture Deliverables

- Décisions d’architecture, patterns de cohérence, structure de repo et frontières documentés.
- Conventions prêtes pour implémentation multi-agents (formats, erreurs, WS, naming).
- Validation de cohérence + liste de risques/gaps à suivre pendant l’implémentation.

### Implementation Handoff

**Pour les agents d’implémentation :** ce document est la source de vérité. Toute divergence doit être discutée et mise à jour ici.

**First Implementation Priority:** initialiser le projet frontend via `pnpm create vue@latest`, puis scaffold backend Go et la base `deploy/` (Compose + Caddy) en respectant la structure.

**Development Sequence:**

1. Initialiser `frontend/` (Vite + Router + Pinia + Vitest + ESLint/Prettier, Tailwind v4)
2. Scaffold `backend/` (serveur HTTP + WS, config, logging)
3. Mettre en place SQLite + migrations goose + repositories
4. Implémenter auth Spotify PKCE + stockage tokens chiffrés
5. Définir contrat REST/WS minimal (endpoints + events + codes d’erreur) puis itérer features

### Quality Assurance Checklist

- [x] Décisions actionnables (versions/patterns)
- [x] Règles de cohérence explicites pour éviter les conflits inter-agents
- [x] Structure `frontend/` / `backend/` / `deploy/` claire
- [x] Validation : cohérence + coverage FR/NFR + readiness + risques

### Project Success Factors

- **Clear Decision Framework** : choix faits avec rationale et contraintes explicites.
- **Consistency Guarantee** : conventions et patterns conçus pour la production de code homogène.
- **Complete Coverage** : besoins fonctionnels + non-fonctionnels supportés au niveau architecture.
- **Solid Foundation** : starter + infra compatibles avec le MVP.

---

**Architecture Status:** READY FOR IMPLEMENTATION

**Document Maintenance:** mettre à jour ce document si une décision technique majeure change pendant l’implémentation.
