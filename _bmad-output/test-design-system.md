# Test Design Système (Phase 3) — boeuf

**Date :** 2026-01-22  
**Auteur :** Mathias  
**Statut :** Draft

---

## Executive Summary

Ce document définit une **stratégie de tests système** (pyramide de tests + NFR) et une **revue de testabilité** avant implémentation.

Contexte produit : web app temps réel (Vue SPA + Go backend) orchestrant Spotify (OAuth + Web API), avec synchronisation multi-clients via WebSocket, ordre serveur (`eventSeq`) et résilience réseau (reconnect `snapshot + events since`).

---

## Testability Assessment

### Controllability — **CONCERNS**

#### Points forts

- Serveur “source of truth” + `eventSeq` monotone : on peut piloter la cohérence côté serveur.
- Event log append-only : bon support pour rejouer/valider des scénarios.

#### Risques / manques probables

- **Spotify comme dépendance externe** (pas de webhooks, Premium requis pour Player API) rend les scénarios de lecture **difficiles à rendre déterministes**.
- Besoin explicite d’un **Spotify Adapter** derrière interface + **Fake/Stub** (et/ou enregistrements) pour tests d’intégration fiables.

#### Décision testabilité (Controllability)

- PASS si : on a un mode `SPOTIFY_MODE=mock|record|live`, avec un fake player contrôlable.
- CONCERNS tant que : les tests d’intégration dépendent d’un état Spotify réel non contrôlable.
**[CR 1-7 Finding]** Problème détecté : tests d'intégration ne couvrent pas l'**initialisation du player state** sur charge page. Les contrôles sont désactivés faute de données initiales (GET endpoint manquant). Cela a échappé aux tests car seuls les POST endpoints (pause/resume/next/seek) étaient testés.

### Observability — **CONCERNS → PASS si instrumentation minimale**

#### Attendus minimum (Observability)

- Logs structurés (avec `trace_id`, `sessionId`, `eventSeq`, `clientMsgId`).
- Métriques : latence propagation WS, temps de resync, erreurs Spotify (429/5xx), taux de retry, drift de synchro.
- Contrat d’erreurs stable `{ code, message, details?, trace_id? }` (REST + WS).

#### Décision testabilité (Observability)

- PASS si : `/health` + métriques + logs corrélables existent dès Sprint 0.
- CONCERNS si : pas de moyens automatiques pour attribuer un bug “Spotify vs WS vs UI”.
**[CR 1-7 Finding]** Le problème d'initialisation n'a pas été détecté car l'observabilité sur le **frontend store state** est faible. Manque : logs de l'initialisation de `playerStore`, trace de l'appel GET endpoint, validation que `track !== null` après init.

### Reliability (tests isolation / reproductibilité) — **CONCERNS**

**Risque principal** : flakiness liée au temps réel (WS), à la reconnexion, au polling Spotify, et aux races multi-clients.

#### Attendus minimum (Reliability)

- Un “clock” serveur (ou time abstraction) pour tests (timeouts, backoff, jitter) sans `sleep` réel.
- Fixtures DB + reset (SQLite) pour tests parallèles.
- Politique anti-flakiness (pas de hard-waits, time budgets, retries maîtrisés).

---

## Architecturally Significant Requirements (ASRs)

ASR = exigences (souvent NFR) qui structurent l’architecture et doivent être testées avec une preuve mesurable.

### ASR list (avec scoring)

| ASR | Type | Exigence | Mesure / preuve | Risque (P×I) |
| --- | --- | --- | --- | --- |
| ASR-01 | PERF | Propagation play/pause/skip ≤ 3s | mesure latence WS + durée resync | 3×3=9 |
| ASR-02 | REL | Détection déconnexion ≤ 5s | tests réseau + métriques | 2×3=6 |
| ASR-03 | REL | Reconnect + resync ≤ 10s | scénarios reconnect + drift | 2×3=6 |
| ASR-04 | REL | Continuité session si 1 client tombe | scénarios multi-clients | 2×3=6 |
| ASR-05 | SEC | Tokens Spotify non exposés côté client | inspection réponses + stockage | 2×3=6 |
| ASR-06 | SEC | Accès session strictement participants | tests authz REST + WS | 2×3=6 |
| ASR-07 | DATA | Ordre serveur (`eventSeq`) + idempotence (`clientMsgId`) | tests protocole WS + replay | 3×3=9 |
| ASR-08 | OPS | Sessions inactives expirent à 24h | tests DB + jobs/cleanup | 2×2=4 |
| ASR-09 | PERF/OPS | Rate-limit Spotify (429) respect Retry-After + backoff/jitter | tests retry/backoff + logs | 3×2=6 |

---

## Risk Register (système)

Échelle : Probability (1-3) × Impact (1-3). Scores ≥6 = mitigation obligatoire. Score 9 = blocker gate.

| Risk ID | Catégorie | Description | Prob. | Impact | Score | Mitigation (résumé) | Owner |
| --- | --- | --- | ---: | ---: | ---: | --- | --- |
| R-001 | TECH | Bug ordre `eventSeq` / concurrence (host, conflits) → divergence d’état | 3 | 3 | 9 | Tests protocole (multi-clients), invariants `eventSeq` monotone + idempotence | Dev |
| R-002 | TECH | Dédoublonnage `clientMsgId` incomplet → actions dupliquées | 3 | 3 | 9 | Tests d’idempotence (replay, retry, reconnect) + table de dédup côté serveur | Dev |
| R-003 | PERF | Latence de propagation >3s sous jitter réseau | 3 | 3 | 9 | Mesures WS (p95) + budgets + optimisations + tests k6 WS | Dev/QA |
| R-004 | REL | Reconnect/resync flaky (snapshot/events since) | 2 | 3 | 6 | Harness de tests WS + simulateur réseau + tests de resync | QA |
| R-005 | SEC | Fuite token (localStorage, logs, payload) | 2 | 3 | 6 | Tests sécurité: inspection stockage/headers, scanning logs, review cookies | Dev |
| R-006 | SEC | Authz session cassée (join via code, accès cross-session) | 2 | 3 | 6 | Tests d’isolement sessions REST/WS + fuzz sur sessionId/invite | QA |
| R-007 | OPS | Dépendance Spotify non déterministe (état player) → tests instables | 3 | 2 | 6 | Spotify adapter + mock/record, tests live séparés (nightly) | Dev |
| R-008 | PERF | Rate-limit Spotify mal géré → cascade de 429 + user impact | 3 | 2 | 6 | Retry-After respecté + backoff/jitter + circuit breaker | Dev |
| R-009 | DATA | Chiffrement AES-256-GCM mal utilisé (clé, nonce) | 2 | 3 | 6 | Tests crypto (roundtrip), rotation clé, erreurs contrôlées | Dev |
| R-010 | BUS | UX “Airlock” mal appliqué → autoplay bloqué / confusion | 2 | 2 | 4 | Tests E2E sur flow “Start Listening” + états audio | QA |
| R-011 | OPS | Expiration sessions 24h non appliquée → DB gonfle | 2 | 2 | 4 | Job cleanup + tests d’expiration + métriques | Dev/Ops |
| R-012 | REL | Détection déconnexion >5s → host failover tardif | 2 | 3 | 6 | Tests heartbeat/timeout + métriques de déconnexion | Dev/QA |
| R-013 | ARCH | **[CR 1-7] Player state init missing** → controls non-functional on load | 3 | 3 | 9 | **BLOCKER** : Ajouter GET endpoint `/player/state` + appeler depuis `playerStore.init()` | Dev |
| R-014 | ARCH | Seek validation incomplet → AC#4 violation | 2 | 3 | 6 | Valider `positionMs ≤ track.durationMs` avant appel Spotify | Dev |
| R-015 | PERF | **[CR 1-7] IdempotenceCache lock contention** → latence spikes | 3 | 2 | 6 | Remplacer cleanup périodique par lazy deletion (risk ≤3s SLA) | Dev |

---

## Test Levels Strategy (pyramide)

### Recommandation de split

- **Unit : 55%** — logique pure (event ordering, idempotence, backoff/jitter, règles host/conflits, chiffrement)
  - **[CR 1-7]** Ajouter tests : seek position validation (`positionMs ≤ durationMs`), GetPlayerState avec Item==nil (edge case), IdempotenceCache cleanup performance.
- **Integration/API : 35%** — Go handlers + DB (SQLite) + hub WS (multi-clients), sans navigateur
  - **Gap trouvé [CR 1-7]** : Tests couvrent les actions (pause/resume/next/seek POST), mais pas l'initialisation (GET /player/state absent). Ajouter : test du flow "charge page → appel GET → playerStore init avec state".
- **E2E : 10%** — flux critiques UI (create/join session, airlock, présence, reconnect basique)
  - **[CR 1-7]** Ajouter test : "join session → PlayerControls buttons enabled" (dépend de GET /player/state)

**Rationale** : le cœur de boeuf est protocolaire/temps réel. Maximiser les tests hors navigateur limite la flakiness et accélère le feedback.

---

## NFR Testing Approach

### Security

Objectifs : pas de secret côté client, contrôle d’accès session, cookies sécurisés.

Automatisation (exemples)

- REST/WS : non-auth → 401/403 et erreur standard `{code,message,...}`.
- Authz : un utilisateur ne peut ni `join` ni lire events d’une session non autorisée.
- Stockage : vérif absence de refresh token dans responses, dans storage navigateur, dans logs.
- Crypto : tests AES-256-GCM (roundtrip + échec sur mauvais key/nonce) + gestion d’erreur stable.

### Performance

Objectifs : propagation ≤3s, UI interactive <3s, queue update ≤2s.

Automatisation (exemples)

- **k6** (ou équivalent) pour charge REST + WS (latence p95/p99, taux d’erreur).
- Mesure “action → ack WS → converged state” (p95 < 3s).
- Budget de polling Spotify (fréquence, backoff, jitter) + protections 429.

### Reliability

Objectifs : déconnexion ≤5s, resync ≤10s, continuité pour autres.

Automatisation (exemples)

- Simulateur réseau (drop WS, latency, packet loss) → tests reconnect.
- Scénarios multi-clients : host down → failover transparent.
- “Trust but Verify” : après commande Spotify, relecture état; assert convergence.

### Maintainability

Objectifs : suite rapide et fiable, signal fort.

Automatisation (exemples)

- CI : lint + unit/integration + (plus tard) e2e.
- Standards : pas de `sleep`, tests parallèles, cleanup DB.
- Artifacts : traces/logs/metrics sur échec.

---

## Test Environment Requirements

### Environnements

- **Local dev** : Docker Compose (Caddy + backend + frontend + SQLite volume).
- **CI** : backend tests sans Spotify live, via mock.
- **Nightly (optionnel)** : “live Spotify smoke” avec comptes de test (Premium), isolé et non-bloquant.

### Données / fixtures

- Reset DB SQLite par test (ou transaction + rollback).
- Factories pour sessions/participants/events.
- Harness WS multi-clients (N connexions, scripts actions, assertions sur ordre/convergence).

---

## Testability Concerns (Blockers / Concerns)

### Blockers potentiels (à éviter)

- Pas d’abstraction Spotify → tests d’intégration instables (dépendance live).
- Pas de mécanisme de reset DB/hub → tests non parallélisables.
- Pas de corrélation (`trace_id`) → debug lent.- **[CR 1-7]** Pas de couverture d'initialisation frontend → features non-functional on first load passent les tests.

### Concerns actuels (à résoudre Sprint 0)

- Définir un mode **mock/record/live** pour Spotify.
- Définir un **contrat WS minimal** (types, enveloppe, codes d’erreur) et le centraliser.
- Décider l’outil E2E (Playwright recommandé pour TS; Cypress aussi viable) et poser la structure.- **[CR 1-7 - URGENT]** : Couvrir les flux d'initialisation frontend (GET endpoints pour state init, store setup, UI readiness). Gap critique : tests unitaires/intégration sur backend POST controllers seul, sans vérifier que frontend peut utiliser le player.

---

## Recommendations for Sprint 0

1. **Tester le protocole avant l’UI** : harness WS + tests d’order/idempotence/resync.
2. **Spotify Adapter** (interface + impl live + impl mock) + tests sur mock.
3. **Observabilité minimale** : `/health`, logs structurés, `trace_id` partout.
4. **Quality gates** : P0 (protocole) doit être vert à 100%.
5. Lancer ensuite les workflows : `TF` (framework), puis `AT` (ATDD) une fois les P0 listées.

---

## Annexes

### Références

- NFR criteria : `_bmad/bmm/testarch/knowledge/nfr-criteria.md`
- Test levels : `_bmad/bmm/testarch/knowledge/test-levels-framework.md`
- Risk governance : `_bmad/bmm/testarch/knowledge/risk-governance.md`
- Test quality DoD : `_bmad/bmm/testarch/knowledge/test-quality.md`

### Documents projet

- PRD : `_bmad-output/planning-artifacts/prd.md`
- Architecture : `_bmad-output/planning-artifacts/architecture.md`
- Epics : `_bmad-output/planning-artifacts/epics.md`
- Project context : `_bmad-output/project-context.md`
