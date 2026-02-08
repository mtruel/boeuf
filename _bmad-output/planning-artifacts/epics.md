---
stepsCompleted: ['step-01-validate-prerequisites', 'step-02-design-epics', 'step-03-create-stories', 'step-04-final-validation']
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/project-context.md
---

# boeuf - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for boeuf, decomposing the requirements from the PRD, UX Design, Architecture requirements, and Project Context into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: Un utilisateur peut créer une nouvelle session d'écoute
FR2: Un utilisateur peut générer un lien de partage pour sa session
FR3: Un utilisateur peut rejoindre une session existante via un lien ou code
FR4: Un utilisateur peut quitter une session
FR5: Un utilisateur peut voir les participants actuels de la session

FR6: Un utilisateur peut connecter son compte Spotify via OAuth
FR7: Un utilisateur peut déconnecter son compte Spotify
FR8: Le système peut lire l'état de lecture actuel d'un utilisateur (morceau, position, état play/pause)
FR9: Le système peut contrôler la lecture d'un utilisateur (play, pause, skip)
FR10: Le système peut lire la file d'attente d'un utilisateur
FR11: Le système peut modifier la file d'attente d'un utilisateur

FR12: Quand un participant lance un morceau, il se lance chez tous les participants
FR13: Quand un participant met pause, la lecture se met en pause chez tous les participants
FR14: Quand un participant reprend la lecture, elle reprend chez tous les participants
FR15: Quand un participant ajoute un morceau à la file d'attente, il apparaît chez tous les participants
FR16: Quand un participant réorganise la file d'attente, elle se réorganise chez tous les participants
FR17: La synchronisation respecte une tolérance de ≤3 secondes de décalage

FR18: Le participant qui effectue une action devient automatiquement le host
FR19: Le système sélectionne automatiquement un nouveau host si le host actuel se déconnecte
FR20: Le changement de host est transparent pour les participants

FR21: Le système détecte quand un participant perd sa connexion
FR22: Un participant déconnecté peut se reconnecter à sa session
FR23: Un participant reconnecté est automatiquement resynchronisé avec l'état actuel de la session
FR24: La musique continue de jouer chez les autres participants pendant une déconnexion

FR25: Quand deux participants agissent simultanément, le système résout le conflit (first wins ou last wins)
FR26: Le système ne bloque pas les actions en cas de conflit

FR27: Un utilisateur peut voir le morceau actuellement en cours de lecture
FR28: Un utilisateur peut voir la file d'attente partagée
FR29: Un utilisateur peut voir l'état de connexion des participants

### NonFunctional Requirements

NFR1: Une action de synchronisation (play/pause/skip) doit se propager à tous les participants en ≤3 secondes
NFR2: L'interface doit être interactive en moins de 3 secondes après chargement
NFR3: Les mises à jour de la file d'attente doivent apparaître chez tous les participants en ≤2 secondes

NFR4: Les tokens OAuth Spotify sont stockés de manière sécurisée (pas en clair côté client)
NFR5: Les communications entre client et serveur sont chiffrées (HTTPS/WSS)
NFR6: Un utilisateur ne peut accéder qu'aux sessions où il est participant
NFR7: Les sessions inactives expirent après 24h

NFR8: Le système détecte une déconnexion utilisateur en ≤5 secondes
NFR9: Un utilisateur déconnecté peut se reconnecter et resynchroniser en ≤10 secondes
NFR10: La perte de connexion d'un participant n'interrompt pas la session pour les autres
NFR11: Le système gère gracieusement les erreurs API Spotify (retry, feedback utilisateur)

NFR12: Le système respecte les rate limits de l'API Spotify
NFR13: Les tokens OAuth sont rafraîchis automatiquement avant expiration
NFR14: L'architecture permet d'ajouter d'autres services de streaming (abstraction)

NFR15: Les éléments interactifs sont accessibles au clavier
NFR16: Les contrastes de couleurs respectent un ratio minimum de 4.5:1

### Additional Requirements

- Le serveur est la source de vérité (ordre serveur), via `eventSeq` monotone par session.
- Les clients doivent inclure `clientMsgId` pour idempotence et dédoublonnage.
- Reconnexion WebSocket: stratégie `snapshot + events since eventSeq`.
- Spotify n'ayant pas de webhooks: polling + approche “Trust but Verify” (relire l’état après commande).
- Gestion rate limit Spotify: respect `Retry-After` + backoff + jitter.
- Stockage des refresh tokens Spotify: chiffrés en DB (AES-256-GCM), clé maître via variable d’environnement (secrets côté serveur uniquement).
- Auth applicative: cookie-session HTTP-only/Secure/SameSite, réutilisée pour REST + WebSocket.
- Accès à une session via invite opaque serveur (lien/code), expirable; contrôle d’accès par session.
- Contrat WS: enveloppe JSON avec champs minimum recommandés `type`, `sessionId`, `eventSeq`, `clientMsgId`, `sentAt`, `payload`.
- Conventions de nommage/format:
  - DB tables/colonnes en `snake_case` (tables au pluriel)
  - JSON (REST+WS) en `camelCase`
  - WS `type` en `UPPER_SNAKE`
  - Erreurs (REST+WS) `code` en `SCREAMING_SNAKE` et payload `{ code, message, details?, trace_id? }`
  - Dates/temps en ISO-8601 UTC (RFC3339)
- Organisation repo à respecter: frontières `frontend/` (Vue SPA) / `backend/` (Go REST+WS) / `deploy/` (Compose+Caddy).
- Starter frontend imposé (MVP): `pnpm create vue@latest` (TS + Router + Pinia + Vitest + ESLint + Prettier) + Tailwind v4 + shadcn-vue.
- Backend MVP: Go + SQLite (CGO avec `mattn/go-sqlite3`) + GORM + goose.
- Tests:
  - Frontend: Vitest, tests co-localisés quand possible
  - Backend: tests ciblés sur protocole (ordre/idempotence/resync) et rate limiting
- UX “Airlock/Sas”: un écran intermédiaire post-login avec un bouton explicite “Start Listening / Sync Now” (éviter l’autoplay bloqué et valider le contexte audio).
- Feedback en arrière-plan: notifier via UI + (optionnel) canal navigateur (title/favicon/notifications) sans bloquer l’écoute.
- Responsive: desktop-first, mais mobile fonctionnel; le Social Panel devient un `Sheet/Drawer` sur mobile.
- Accessibilité: clavier, contrastes, `aria-live` pour changements d’état, respect `prefers-reduced-motion`.
- Décision (cohérence sécurité): **aucun secret persistant côté frontend**. Les credentials Spotify (client id/secret) sont configurés **côté serveur** (MVP self-hosted: variables d’environnement / `.env` Docker Compose). Le frontend peut afficher un statut “instance configurée” et des instructions, mais ne stocke pas de secrets.

### FR Coverage Map

### FR Coverage Map

FR1: Epic 1 - Créer une session
FR2: Epic 1 - Générer un lien/code de partage
FR3: Epic 1 - Rejoindre via lien/code
FR4: Epic 4 - Quitter une session proprement
FR5: Epic 1 - Voir les participants

FR6: Epic 1 - Connecter Spotify (OAuth)
FR7: Epic 4 - Déconnecter Spotify
FR8: Epic 1 - Lire l’état de lecture (track/position/play-pause)
FR9: Epic 1 - Contrôler la lecture (play/pause/skip/seek)
FR10: Epic 2 - Lire la file d’attente
FR11: Epic 2 - Modifier la file d’attente

FR12: Epic 1 - Propager “play” à tous
FR13: Epic 1 - Propager “pause” à tous
FR14: Epic 1 - Propager “resume” à tous
FR15: Epic 2 - Propager ajout à la queue
FR16: Epic 2 - Propager réorganisation de la queue
FR17: Epic 1 - Tolérance de synchro ≤ 3s

FR18: Epic 3 - “Qui agit devient host”
FR19: Epic 3 - Failover host automatique
FR20: Epic 3 - Changement de host transparent

FR21: Epic 4 - Détecter déconnexion
FR22: Epic 4 - Rejoindre à nouveau après déconnexion
FR23: Epic 4 - Resynchronisation automatique
FR24: Epic 4 - Continuité pour les autres pendant une déconnexion

FR25: Epic 3 - Résolution de conflit (first/last wins)
FR26: Epic 3 - Ne pas bloquer les actions

FR27: Epic 1 - Afficher “now playing”
FR28: Epic 2 - Afficher la queue partagée
FR29: Epic 1 - Afficher l’état de connexion des participants

## Epic List

### Epic 1: Démarrer une session synchronisée (Link-to-Music)

Objectif: Créer/rejoindre une session, s’authentifier Spotify, passer par le “Sas” (Start Listening) et obtenir une écoute synchronisée (play/pause) + dashboard minimal (now playing, présence, statut).
**FRs covered:** FR1, FR2, FR3, FR5, FR6, FR8, FR9, FR12, FR13, FR14, FR17, FR27, FR29

### Epic 2: Queue collaborative (contribuer ensemble)

Objectif: Voir et modifier la file d’attente de manière partagée (lecture, ajout, réorganisation), avec propagation temps réel.
**FRs covered:** FR10, FR11, FR15, FR16, FR28

### Epic 3: Gouvernance temps réel (host tournant + conflits)

Objectif: Rendre les actions multi-utilisateurs prévisibles: host tournant (“qui agit devient host”), failover, et arbitrage simple en cas d’actions simultanées.
**FRs covered:** FR18, FR19, FR20, FR25, FR26

### Epic 4: Résilience session (déconnexion, reconnexion, resync)

Objectif: Assurer la continuité quand le réseau bouge: détecter la déconnexion, permettre la reconnexion, resynchroniser automatiquement, et garder la session fonctionnelle pour les autres.
**FRs covered:** FR4, FR7, FR21, FR22, FR23, FR24

### Epic 5: Confiance & qualité MVP (observabilité, erreurs, accessibilité)

Objectif: Renforcer la confiance “Trust but Verify” (statuts, feedback), traiter les erreurs Spotify/rate-limit proprement, et couvrir responsive + accessibilité.
**FRs covered:** (principalement NFR/UX; aucun FR unique restant)

## Epic 1: Démarrer une session synchronisée (Link-to-Music)

Permettre à un utilisateur de créer une session, inviter un ami, se connecter à Spotify, passer par le “Sas” (Start Listening) et écouter le même contenu avec play/pause synchronisés, visibilité “now playing” et présence.

### Story 1.1: Démarrer le squelette exécutable (frontend/backend/deploy)

As a développeur,
I want un squelette exécutable (frontend, backend, proxy),
So that je peux itérer rapidement et valider l’intégration bout-en-bout.

**Acceptance Criteria:**

**Given** le dépôt sur une machine de dev
**When** j’initialise le frontend via `pnpm create vue@latest` (TS + Router + Pinia + Vitest + ESLint + Prettier) dans `frontend/`
**Then** l’app frontend démarre en dev (`pnpm dev`) et compile sans erreur
**And** les frontières `frontend/` / `backend/` / `deploy/` sont respectées

**Given** un backend Go initial dans `backend/`
**When** je lance le serveur (ex: `go run ./cmd/boeuf-server`)
**Then** il expose un endpoint `/health` qui répond 200 avec JSON
**And** il peut répondre à un appel depuis le frontend (CORS/dev proxy ou même origin via reverse proxy)

**Given** une configuration de déploiement MVP
**When** je lance `docker compose up`
**Then** le reverse proxy (Caddy) sert le frontend et reverse-proxy le backend
**And** les communications en prod passent par HTTPS/WSS

**Given** une instance self-hosted
**When** je configure les secrets serveur via `.env`/variables d’environnement
**Then** le backend démarre avec `SPOTIFY_CLIENT_ID` et `SPOTIFY_CLIENT_SECRET` disponibles côté serveur
**And** aucun secret Spotify n’est stocké de manière persistante côté navigateur (LocalStorage/IndexedDB)

### Story 1.2: Auth Spotify (OAuth Authorization Code + PKCE) avec session cookie

As a utilisateur,
I want connecter mon compte Spotify via OAuth,
So that boeuf puisse orchestrer la lecture sur mon appareil.

**Acceptance Criteria:**

**Given** un utilisateur non authentifié
**When** il clique sur “Connecter Spotify”
**Then** il est redirigé vers Spotify et revient sur boeuf avec une session active
**And** le backend stocke les tokens côté serveur (aucun secret durable côté frontend)

**Given** un refresh token à stocker
**When** le backend le persiste
**Then** il est chiffré en base (AES-256-GCM) et la clé n’est jamais exposée au client

**Given** un access token expiré ou proche de l’expiration
**When** le backend doit appeler Spotify
**Then** il rafraîchit automatiquement le token (sans intervention utilisateur)
**And** aucune action Spotify n’échoue “silencieusement” à cause d’un token expiré

**Given** l’intégration Spotify côté backend
**When** on implémente la logique d’orchestration
**Then** elle est encapsulée derrière une abstraction (ex: interface `StreamingProvider` / `SpotifyClient`)
**And** la logique “session/sync” ne dépend pas directement de détails Spotify (facilite NFR14)

### Story 1.3: Créer une session et générer un lien/code d’invitation

As a host (initiateur),
I want créer une session et obtenir un lien/code,
So that je puisse inviter quelqu’un rapidement.

**Acceptance Criteria:**

**Given** un utilisateur authentifié Spotify
**When** il crée une session
**Then** le serveur crée une session avec une invite opaque (code/lien) expirable
**And** le frontend affiche un lien partageable et un code court (si applicable)

### Story 1.4: Rejoindre une session via lien/code (avec contrôle d’accès)

As a invité,
I want rejoindre une session via un lien/code,
So that je puisse écouter avec le groupe sans friction.

**Acceptance Criteria:**

**Given** un lien/code valide
**When** l’invité l’ouvre
**Then** il rejoint la session après OAuth (si nécessaire)
**And** un lien/code invalide retourne une erreur stable (`SESSION_NOT_FOUND` ou équivalent)

**Given** un utilisateur authentifié qui n’est pas participant d’une session
**When** il tente d’accéder à ses ressources (REST/WS)
**Then** le serveur refuse l’accès avec une erreur stable (ex: `FORBIDDEN`)
**And** l’utilisateur ne peut accéder qu’aux sessions où il est participant (NFR6)

### Story 1.5: WebSocket session + présence (participants + état connexion)

As a utilisateur,
I want voir qui est présent et leur état de connexion,
So that je sache si on écoute vraiment “ensemble”.

**Acceptance Criteria:**

**Given** un utilisateur a rejoint une session
**When** il se connecte au WebSocket
**Then** il reçoit un snapshot de session (participants, now playing minimal)
**And** les événements de join/leave sont diffusés à tous les participants en temps réel

**Given** un message WebSocket envoyé par le serveur
**When** il est reçu par le client
**Then** il respecte l’enveloppe JSON (au minimum: `type`, `sessionId`, `eventSeq`, `sentAt`, `payload`)
**And** `type` est en `UPPER_SNAKE` et `sentAt` est en RFC3339 UTC

### Story 1.6: Sas “Start Listening” (anti-autoplay) et état “Ready → Synced”

As a invité,
I want un écran “Sas” avec un bouton Start Listening,
So that je garde le contrôle et évite l’autoplay bloqué/surprenant.

**Acceptance Criteria:**

**Given** l’utilisateur est dans une session mais n’a pas encore “démarré l’écoute”
**When** il ouvre la page session
**Then** la musique ne démarre pas automatiquement
**And** l’UI affiche un bouton principal “Start Listening / Sync Now”

**Given** l’utilisateur clique sur “Start Listening”
**When** le backend initialise la synchronisation
**Then** l’UI passe en état “Synced/Live” (feedback visuel clair)

### Story 1.7: Synchronisation play/pause (événements ordonnés + idempotence)

As a participant,
I want que play/pause se propage à tous,
So that l’écoute reste synchronisée.

**Acceptance Criteria:**

**Given** deux participants connectés à la même session
**When** l’un met pause
**Then** tous les clients reçoivent un événement WS ordonné (`eventSeq` monotone)
**And** la lecture est en pause sur tous les appareils dans un délai ≤ 3s

**Given** deux participants connectés à la même session
**When** l’un clique play/reprise
**Then** tous les clients reçoivent l’événement correspondant
**And** la lecture reprend sur tous les appareils dans un délai ≤ 3s

**Given** deux participants connectés à la même session
**When** l’un déclenche “piste suivante” (skip)
**Then** le changement est propagé à tous via WS
**And** l’UI “Now Playing” se met à jour pour tous

**Given** deux participants connectés à la même session
**When** l’un effectue un seek (avance/recul)
**Then** la position converge chez tous dans un délai ≤ 3s
**And** le serveur reste l’arbitre de l’état partagé (pas de dépendance aux horloges client)

**Given** un client renvoie le même message (retry réseau)
**When** il réutilise le même `clientMsgId`
**Then** le serveur traite l’action de manière idempotente (pas de double-exécution)

### Story 1.8: Afficher “Now Playing” (morceau, artiste, progression) + mise à jour temps réel

As a utilisateur,
I want voir le morceau en cours (et une progression indicative),
So that je sache ce que le groupe écoute.

**Acceptance Criteria:**

**Given** une session active
**When** le track courant change
**Then** le dashboard met à jour “Now Playing” pour tous les participants
**And** les timestamps échangés sont en RFC3339 UTC

## Epic 2: Queue collaborative (contribuer ensemble)

Permettre de consulter et manipuler la file d’attente de manière partagée, avec propagation temps réel.

**Note (Story 1.10 Alignment):** Avec le modèle de rooms persistantes, la queue survit aux départs/rejoin des participants. La queue persiste tant que la room est active (LIVE ou STALE).

### Story 2.1: Afficher la file d'attente partagée

As a utilisateur,
I want voir la queue partagée,
So that je comprenne ce qui va passer ensuite.

**Acceptance Criteria:**

**Given** un utilisateur dans une room (LIVE ou STALE)
**When** il ouvre la vue queue
**Then** l'UI affiche une liste d'items (au minimum: titre, artiste, durée)
**And** les données échangées respectent JSON camelCase
**And** la queue reste visible même si la room est STALE (0 participants actifs)

**Note:** API endpoint reste `/api/sessions/:id/queue` pour backward compatibility, mais conceptuellement il s'agit de la queue de la room.

### Story 2.2: Ajouter un titre à la queue (propagation temps réel)

As a participant,
I want ajouter un titre à la queue,
So that je puisse contribuer à l’écoute.

**Acceptance Criteria:**

**Given** un participant actif (`is_active=true`) dans une room LIVE
**When** il sélectionne un titre et demande l'ajout (via UI ou identifiant Spotify)
**Then** le serveur orchestre l'ajout côté Spotify (best-effort)
**And** tous les participants actifs voient l'item apparaître en ≤ 2s (NFR3)

**Given** une room STALE (0 participants actifs) et un utilisateur rejoint
**When** le premier participant ajoute un titre à la queue
**Then** la room passe automatiquement de STALE → LIVE
**And** l'ajout est orchestré normalement

**Given** Spotify répond 429
**When** l'ajout est tenté
**Then** le serveur respecte `Retry-After` et expose un état "rate-limited" compréhensible au client

**Note:** Queue persiste avec la room - un utilisateur peut quitter et rejoindre sans perdre la queue existante.

### Story 2.3: Réordonner la queue (stratégie MVP compatible Spotify)

As a participant,
I want réordonner la queue,
So that on puisse décider collectivement de l’ordre de lecture.

**Acceptance Criteria:**

**Given** un participant actif dans une room LIVE
**And** la plateforme Spotify ne supporte pas nativement le réordonnancement de queue
**When** l'utilisateur réordonne dans boeuf
**Then** le serveur maintient un ordre "desired queue" partagé et converge au mieux avec Spotify
**And** l'UI reflète l'ordre "boeuf" comme source de vérité pour la suite des morceaux

**Given** une room STALE avec une queue existante
**When** un utilisateur rejoint et visualise la queue
**Then** l'ordre précédent est préservé (queue persiste avec la room)

**Note:** La persistence des rooms (Story 1.10) améliore l'UX - les participants peuvent quitter/rejoindre sans perdre l'ordre de la queue.

### Story 2.4: Mettre à jour la queue à partir de la réalité Spotify (“Trust but Verify”)

As a utilisateur,
I want que boeuf vérifie l’état réel Spotify régulièrement,
So that l’UI ne dérive pas silencieusement.

**Acceptance Criteria:**

**Given** une room LIVE (has active participants avec `is_active=true`)
**When** le serveur poll Spotify et détecte un écart entre "desired queue" et l'état Spotify réel
**Then** il publie un événement de resync et met à jour l'UI pour tous les participants actifs
**And** les resync n'interrompent pas la lecture (pas de modals bloquantes)

**Given** une room STALE (0 participants actifs)
**When** le polling timer s'exécute
**Then** le serveur NE poll PAS Spotify (pas de participants pour synchroniser)
**And** le polling reprend automatiquement quand la room redevient LIVE (un participant rejoint)

**Note:** Le polling Spotify est désactivé pour les rooms STALE pour économiser les API calls. L'état de la queue est préservé en DB et reste consultable.

## Epic 3: Sync Leadership & Playback Orchestration

Établir un mécanisme de "sync leader" transparent qui détermine quelle source Spotify le serveur poll pour orchestrer la synchronisation playback de la room.

**Key Principles:**

- **Sync Leader = Technical Mechanism:** L'utilisateur dont le compte Spotify sert de source de vérité pour le polling serveur
- **Transparent Assignment:** Changement automatique basé sur les actions, sans friction
- **Discrete Visibility:** Badge visuel discret (pastille ●) dans la liste participants, SANS notifications/toasts
- **Room Owner ≠ Sync Leader:** Concepts distincts (owner = permanent/manages room, sync leader = temporary/polling source)

**Context (Story 1.10 Integration):**

- Story 1.10 ajoute `created_by` (room owner) - permanent, visible
- Epic 3 ajoute `sync_leader_user_id` - dynamique, badge discret seulement
- Optimise Story 1.8 polling: au lieu de poller tous les participants, poll uniquement sync leader

**Schema Addition:**

```sql
ALTER TABLE sessions ADD COLUMN sync_leader_user_id TEXT NULL;
ALTER TABLE session_participants ADD COLUMN last_action_at TIMESTAMP NULL;
```

### Story 3.0: Initial Sync Leader Assignment

As a backend system,
I want automatically assign a sync leader when a room becomes LIVE,
So that polling can begin immediately without ambiguity.

**Acceptance Criteria:**

**Given** une room STALE (0 participants actifs, `sync_leader_user_id = NULL`)
**When** le premier participant rejoint la room
**Then** ce participant devient automatiquement sync leader
**And** `sync_leader_user_id` est set au `user_id` du participant
**And** le serveur commence le polling Spotify sur ce user's account
**And** room state passe STALE → LIVE

**Given** un utilisateur crée une nouvelle room (Story 1.10)
**When** le backend auto-join le créateur après création
**Then** le créateur devient sync leader par défaut (premier participant actif)

**Given** un participant a `user_id == sync_leader_user_id`
**When** la liste des participants est affichée dans l'UI
**Then** une pastille ronde discrète (●) apparaît à côté du nom du sync leader
**And** AUCUNE notification/toast n'explique le concept
**And** la pastille est le SEUL indicateur visuel (pas de highlight, pas de section dédiée)

### Story 3.1: Automatic Sync Leader Switch on Playback Action

As a backend system,
I want automatically switch sync leader to the user who takes playback actions,
So that the polling source reflects the most active/relevant Spotify account.

**Acceptance Criteria:**

**Given** User A est le sync leader actuel (`sync_leader_user_id = userA`)
**And** User B prend une action playback (play, pause, seek, skip)
**When** le backend traite l'action de User B avec succès
**Then** `sync_leader_user_id` est automatiquement changé vers `userB`
**And** `session_participants.last_action_at` est mis à jour pour User B
**And** le polling Spotify switch immédiatement de User A account → User B account
**And** AUCUNE notification n'est envoyée aux clients

**Given** le sync leader change de User A → User B
**When** les clients reçoivent l'événement de state update
**Then** la pastille ● disparaît de User A et apparaît à côté de User B dans la participant list
**And** le changement est visuel seulement (pas de toast/modal)

**Given** User A est sync leader
**When** User B ajoute un titre à la queue (Epic 2)
**Then** `sync_leader_user_id` reste User A (AUCUN changement)
**Note:** Queue actions n'impactent pas playback state source → pas de raison de changer polling target

### Story 3.2: Sync Leader Failover on Disconnect

As a backend system,
I want select a new sync leader when the current sync leader disconnects,
So that polling can continue seamlessly without interruption.

**Acceptance Criteria:**

**Given** User A est sync leader (`sync_leader_user_id = userA`)
**And** il y a d'autres participants actifs dans la room (User B, User C)
**When** User A se déconnecte (WebSocket close) OU fait POST `/api/sessions/:id/leave`
**Then** le serveur détecte la perte en ≤ 5s (NFR8)
**And** sélectionne automatiquement le participant avec le `last_action_at` le plus récent comme nouveau sync leader
**And** `sync_leader_user_id` est mis à jour vers le nouvel élu
**And** polling reprend immédiatement sur le nouveau sync leader

**Given** le sync leader actuel se déconnecte
**When** le serveur doit choisir un remplaçant
**Then** la stratégie de sélection est: **dernier participant qui a agi** (via `last_action_at` DESC)
**And** si aucun participant n'a `last_action_at` (tous nouveaux), prendre le premier par ordre alphabétique de `user_id`

**Given** User A est sync leader ET le seul participant actif
**When** User A se déconnecte ou leave
**Then** room passe LIVE → STALE
**And** `sync_leader_user_id = NULL`
**And** polling Spotify est arrêté (pas de participants à synchroniser)

**Given** une room STALE avec `sync_leader_user_id = NULL`
**When** un nouveau participant rejoint (Story 3.0)
**Then** ce participant devient sync leader automatiquement
**And** polling reprend immédiatement

**Given** le sync leader change due à déconnexion (User A → User B)
**When** les participants restants reçoivent l'update
**Then** la pastille ● se déplace vers User B dans la participant list
**And** AUCUNE notification n'est affichée

### Story 3.3: Polling Optimization - Single Source of Truth

As a backend system,
I want poll only the sync leader's Spotify account for playback state,
So that API calls are optimized and state propagation is unambiguous.

**Acceptance Criteria:**

**Given** une room LIVE avec sync leader défini (`sync_leader_user_id = userA`)
**When** le polling timer s'exécute (Story 1.8 - toutes les 5-10s)
**Then** le serveur poll UNIQUEMENT le Spotify account de User A
**And** appel API: `GET /me/player` avec access token de User A
**And** les autres participants ne sont PAS pollés (réduction ~80% des API calls)

**Given** le serveur reçoit playback state du sync leader
**When** le state diffère de l'état précédent (track changed, position drift, etc.)
**Then** broadcast événement `PLAYER_STATE_UPDATE` à TOUS les participants actifs
**And** tous les clients se synchronisent sur cet état (Story 1.8 logic)

**Migration from Story 1.8:**

- Before: Poll tous les participants synced
- After: Poll uniquement sync leader
- Breaking change: Oui, mais amélioration (moins d'API calls, moins de risque rate limit)

### Story 3.4: Résolution de conflits concurrentiels (policy simple)

As a backend system,
I want resolve concurrent actions with a simple documented policy,
So that the system remains stable without heavy locking.

**Acceptance Criteria:**

**Given** deux actions concurrentes arrivent au serveur quasi simultanément
**When** elles sont reçues (ex: User A pause, User B play)
**Then** le serveur applique une policy **first-wins** par ordre de réception
**And** première action reçue est traitée, seconde est appliquée après (pas rejetée)

**Given** chaque action génère un événement
**When** les événements sont diffusés
**Then** chaque événement a un `eventSeq` monotone croissant par room
**And** les clients peuvent auditer l'ordre exact des actions

**Given** deux users agissent simultanément
**When** le serveur arbitre avec first-wins
**Then** AUCUNE action n'est bloquée/rejetée
**And** les deux actions sont appliquées séquentiellement

**Note:** Story 3.4 est orthogonale au concept de sync leader. Le sync leader détermine la SOURCE de polling, pas l'arbitrage des actions concurrentes

## Epic 4: Résilience Room (déconnexion, reconnexion, resync)

Assurer que la room survit aux coupures réseau et que les clients se resynchronisent automatiquement.

**Context (Story 1.10 Integration):**

- Story 1.10 introduces `session_participants.is_active` tracking
- Disconnection updates `is_active = false` + sets `left_at` timestamp
- Reconnection sets `is_active = true` again (same participant record)
- Room lifecycle (LIVE/STALE/ARCHIVED) already handled in Story 1.10
- Epic 4 focuses on **client-side resilience** and **state convergence** after network failures

### Story 4.1: Détection de déconnexion et état présence (is_active tracking)

As a utilisateur,
I want que la présence reflète rapidement les déconnexions,
So that je sache si quelqu'un est "vraiment là".

**Acceptance Criteria:**

**Given** un participant perd le réseau
**When** sa connexion WS est interrompue
**Then** le serveur détecte la perte en ≤ 5s (NFR8)
**And** met à jour `session_participants.is_active = false` + `left_at = NOW()`
**And** diffuse un événement `PARTICIPANT_LEFT` à tous les participants actifs restants

**Given** le participant déconnecté était sync leader (Story 3.2)
**When** `is_active = false` est appliqué
**Then** le serveur déclenche sync leader failover automatiquement
**And** sélectionne nouveau sync leader parmi participants où `is_active = true`

**Given** un participant se déconnecte pendant une lecture en cours
**When** l'événement est traité
**Then** la lecture continue pour les autres participants actifs (aucun arrêt forcé)
**And** l'UI indique que le participant est "offline" dans la participant list
**And** la room reste LIVE tant qu'il y a ≥ 1 participant avec `is_active = true`

**Given** le dernier participant actif se déconnecte
**When** `is_active = false` pour tous les participants
**Then** room passe LIVE → STALE (Story 1.10 AC5)
**And** `sync_leader_user_id = NULL` (Story 3.2)
**And** polling Spotify est arrêté

### Story 4.2: Reconnexion automatique côté client (WS) avec backoff

As a utilisateur,
I want que l’app se reconnecte automatiquement,
So that je ne doive pas rafraîchir la page.

**Acceptance Criteria:**

**Given** une perte WS temporaire (< 24h depuis `left_at`)
**When** le client tente de se reconnecter
**Then** il applique un backoff progressif: 1s, 2s, 4s, 8s (max 30s) avec jitter
**And** l'UI indique "Reconnecting…" sans bloquer le reste de l'interface
**And** le bouton "Leave Room" reste accessible pendant reconnexion

**Given** reconnexion WS réussie dans les 24h
**When** le client envoie son `user_id` + `session_id` au serveur
**Then** le serveur met à jour `session_participants.is_active = true` + `left_at = NULL`
**And** diffuse événement `PARTICIPANT_JOINED` aux autres participants actifs
**And** si la room était STALE, elle repasse LIVE (Story 1.10 AC4)

**Given** reconnexion après > 24h depuis `left_at`
**When** le client tente de rejoindre
**Then** le serveur refuse la reconnexion automatique (participant record peut être archivé)
**And** l'utilisateur doit rejoindre manuellement via invite link (Story 1.4)

### Story 4.3: Resync “snapshot + events since seq”

As a participant reconnecté,
I want récupérer l'état de room et rejouer les événements manquants,
So que je converge vers l'état actuel sans ambiguïté.

**Acceptance Criteria:**

**Given** un client se reconnecte avec un `lastEventSeq` mémorisé
**When** il rejoint le WS après reconnexion
**Then** le serveur renvoie:

- Room snapshot: `{ room_state, participants[], current_track, queue[], sync_leader_user_id }`
- Events depuis `lastEventSeq`: `[{ eventSeq, type, payload, timestamp }]`
**And** le client applique ces événements dans l'ordre pour retrouver l'état courant

**Given** `lastEventSeq` est trop ancien (events purgés/archivés)
**When** le serveur ne peut pas fournir tous les events
**Then** il renvoie un snapshot complet + flag `full_resync = true`
**And** le client reset son état local et applique uniquement le snapshot

**Given** une reconnexion après coupure réseau temporaire (< 5min)
**When** le client retrouve la connectivité
**Then** la resynchronisation complète (snapshot + events + convergence) se fait en ≤ 10s (NFR9)
**And** l'UI repasse de "Reconnecting…" → état normal avec données à jour

### Story 4.4: Déconnecter Spotify (révoquer tokens côté serveur)

As a utilisateur,
I want déconnecter mon compte Spotify,
So that je puisse arrêter de donner accès à boeuf.

**Acceptance Criteria:**

**Given** un utilisateur connecté à Spotify
**When** il clique “Déconnecter Spotify”
**Then** le backend invalide/supprime les tokens stockés côté serveur
**And** le frontend revient à un état “non connecté” avec un CTA “Connecter Spotify”

**Given** un utilisateur déconnecté Spotify
**When** il tente une action de lecture/queue
**Then** l’API répond avec un code d’erreur stable (ex: `SPOTIFY_NOT_CONNECTED`)
**And** l’UI affiche un message actionnable (ex: “Reconnecter Spotify”)
**Note:** Story 4.4 est orthogonale au room lifecycle (Story 1.10). Un user peut disconnect Spotify tout en restant participant actif d'une room (mais ne peut plus agir sur playback/queue).

**Note:** Story "Quitter la room et expiration" est maintenant couverte par Story 1.10 AC3 (leave), AC5 (STALE state), AC7 (manual archive/delete). Epic 4 se concentre sur **client resilience** uniquement.

## Epic 5: Confiance & qualité MVP (observabilité, erreurs, accessibilité)

Renforcer la confiance utilisateur et la robustesse: erreurs standardisées, rate limiting, UX d’état, accessibilité et responsive.

### Story 5.1: Contrat d’erreurs standard (REST + WS)

As a développeur,
I want un format d’erreur stable et uniforme,
So that le frontend puisse afficher des messages fiables et débugger vite.

**Acceptance Criteria:**

**Given** une erreur REST ou WS
**When** elle est renvoyée au client
**Then** elle suit le format `{ code, message, details?, trace_id? }`
**And** `code` est en `SCREAMING_SNAKE` et reste stable dans le temps

### Story 5.2: Gestion rate-limit Spotify (429) + “Retry-After”

As a utilisateur,
I want que boeuf gère les limites Spotify proprement,
So that l’app reste fiable même quand Spotify ralentit.

**Acceptance Criteria:**

**Given** Spotify renvoie 429 avec `Retry-After`
**When** le backend doit relancer une requête
**Then** il respecte `Retry-After` et applique un backoff avec jitter
**And** l’UI reçoit un état clair (ex: “Syncing… / Rate limited”) sans spam

### Story 5.3: Timeline d’événements “Trust but Verify” (debug UX)

As a utilisateur,
I want voir un petit historique des actions récentes,
So that je comprenne “qui a fait quoi” quand ça paraît désynchronisé.

**Acceptance Criteria:**

**Given** des actions (play/pause/add)
**When** elles sont arbitrées par le serveur
**Then** l’UI affiche une timeline des N derniers événements (N petit, ex: 20)
**And** les items incluent auteur + type + timestamp

### Story 5.4: Accessibilité & motion (MVP)

As a utilisateur,
I want une UI utilisable au clavier et qui respecte “reduced motion”,
So that je puisse utiliser boeuf dans des contextes variés.

**Acceptance Criteria:**

**Given** un utilisateur navigue au clavier
**When** il parcourt l’interface session
**Then** les contrôles principaux sont focusables et utilisables
**And** les animations importantes respectent `prefers-reduced-motion`

**Given** un thème sombre avec accents (ex: amber sur charcoal)
**When** l’UI est utilisée dans des conditions courantes
**Then** les contrastes des éléments clés respectent un ratio minimum 4.5:1 (NFR16)

### Story 5.5: Budget perf MVP (TTI) + mesures de base

As a utilisateur,
I want que l’interface soit rapidement utilisable,
So that rejoindre une session ne soit pas frustrant.

**Acceptance Criteria:**

**Given** un desktop moderne et une connexion standard
**When** je charge l’app
**Then** l’interface devient utilisable en < 3s (NFR2)
**And** une mesure simple est documentée (ex: procédure Lighthouse/devtools) pour vérifier ce budget
