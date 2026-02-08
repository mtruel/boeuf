# Matrice de traçabilité & Quality Gate — Story 1.1

**Story :** 1.1 — Démarrer le squelette exécutable (frontend/backend/deploy)
**Date :** 2026-01-22
**Évaluateur :** Mathias (TEA/Murat)

> Note : ce workflow ne génère pas de tests. En cas de gaps, exécuter `AT` (*atdd*) ou `TA` (*automate*) après avoir initialisé le framework (`TF`).

## PHASE 1 — TRAÇABILITÉ EXIGENCES → TESTS

### Résumé couverture

> Hypothèses : à date, aucun code `frontend/` / `backend/` / `deploy/` n’est présent dans le repo, et aucun répertoire `tests/` n’est détecté. La couverture « réelle » est donc **nulle** ; les recommandations ci-dessous décrivent la **cible de tests** à créer en même temps que l’implémentation de la story.

| Priorité | Total critères | Couverture FULL | Couverture % | Statut |
| --- | ---: | ---: | ---: | --- |
| P0 | 2 | 0 | 0% | ❌ FAIL |
| P1 | 1 | 0 | 0% | ❌ FAIL |
| P2 | 0 | 0 | — | — |
| P3 | 0 | 0 | — | — |
| **Total** | **3** | **0** | **0%** | **❌ FAIL** |

Seuils (référence workflow) : P0 = 100% obligatoire, P1 ≥ 90% recommandé, total ≥ 80% recommandé.

---

### Mapping détaillé

#### AC-1 (P0) — Frontend scaffoldé et exécutable

- **Critère** :
  - Given le dépôt sur une machine de dev
  - When j’initialise le frontend via `pnpm create vue@latest` (TS + Router + Pinia + Vitest + ESLint + Prettier) dans `frontend/`
  - Then l’app démarre en dev (`pnpm dev`) et compile sans erreur
  - And les frontières `frontend/` / `backend/` / `deploy/` sont respectées
- **Couverture** : NONE ❌
- **Tests existants** : aucun détecté
- **Recommandation (tests à ajouter)** :
  - `1.1-CI-001` (smoke) — build frontend
    - **Given** un checkout propre
    - **When** `pnpm install && pnpm -C frontend lint && pnpm -C frontend test && pnpm -C frontend build`
    - **Then** tous les jobs passent
  - `1.1-UNIT-001` (frontend, Vitest) — smoke de boot
    - **Given** l’app Vue
    - **When** on monte le root component
    - **Then** rendu sans erreur (pas de crash)

#### AC-2 (P0) — Backend `/health` + appel frontend

- **Critère** :
  - Given un backend Go initial dans `backend/`
  - When je lance le serveur (ex: `go run ./cmd/boeuf-server`)
  - Then il expose un endpoint `/health` qui répond 200 avec JSON
  - And il peut répondre à un appel depuis le frontend (CORS/dev proxy ou via reverse proxy)
- **Couverture** : NONE ❌
- **Tests existants** : aucun détecté
- **Recommandation (tests à ajouter)** :
  - `1.1-API-001` (Go, integration) — `/health` répond 200 + JSON stable
    - **Given** le serveur démarré via `httptest` (ou handler pur)
    - **When** GET `/health`
    - **Then** status 200 et body JSON `{ "status": "ok" }` (ou schéma équivalent documenté)
  - `1.1-API-002` (Go) — politique CORS / proxy
    - **Given** une requête cross-origin simulée
    - **When** OPTIONS/GET avec headers CORS
    - **Then** en-têtes attendus présents (ou proxy dev configuré et validé)

#### AC-3 (P1) — Docker Compose + reverse proxy (HTTPS/WSS)

- **Critère** :
  - Given une configuration de déploiement MVP
  - When je lance `docker compose up`
  - Then Caddy sert le frontend et reverse-proxy le backend
  - And les communications en prod passent par HTTPS/WSS
- **Couverture** : NONE ❌
- **Tests existants** : aucun détecté
- **Recommandation (tests à ajouter)** :
  - `1.1-E2E-001` (smoke deploy) — compose smoke test
    - **Given** `docker compose up -d`
    - **When** on interroge `https://localhost/health` (ou endpoint backend via proxy)
    - **Then** 200 OK + JSON
    - **And** un check WSS minimal (handshake) est possible

---

### Analyse des gaps

#### Gaps critiques (BLOCKER) ❌

2 gaps trouvés. **Ne pas considérer la story “done” sans ces validations minimales.**

1. **AC-1 — Frontend scaffoldé et exécutable (P0)**
   - Couverture actuelle : NONE
   - Tests manquants : CI smoke (lint/test/build) + boot test Vitest
   - Reco : `1.1-CI-001`, `1.1-UNIT-001`
   - Impact : risque TECH élevé (pipeline non stable), risque OPS (build cassé)

2. **AC-2 — Backend `/health` + appel frontend (P0)**
   - Couverture actuelle : NONE
   - Tests manquants : test handler `/health` + validation CORS/proxy
   - Reco : `1.1-API-001`, `1.1-API-002`
   - Impact : risque OPS (monitoring), TECH (contrat instable), BUS (frontend bloqué)

#### Gaps haute priorité (PR blocker) ⚠️

1 gap trouvé.

1. **AC-3 — Docker Compose + reverse proxy (P1)**
   - Couverture actuelle : NONE
   - Reco : `1.1-E2E-001`
   - Impact : risque OPS (déploiement non testable), PERF/SEC selon config TLS

---

### Qualité des tests

À date : **aucun test détecté**, donc aucun audit DoD possible. Rappels DoD (à appliquer dès la création) :

- Déterministes (pas de `sleep`, pas de waits arbitraires)
- Assertions explicites dans le corps du test
- Isolation + cleanup
- < 300 lignes / fichier, < 90s par test

---

### Couverture par niveau de test

| Niveau | Tests | Critères couverts | Couverture % |
| --- | ---: | ---: | ---: |
| E2E | 0 | 0 | 0% |
| API | 0 | 0 | 0% |
| Composant | 0 | 0 | 0% |
| Unit | 0 | 0 | 0% |
| **Total** | **0** | **0** | **0%** |

---

### Recommandations

#### Actions immédiates (avant merge)

1. **Initialiser l’ossature et ajouter les smoke tests** : `1.1-CI-001`, `1.1-API-001`
2. **Verrouiller les conventions de contrat** : JSON camelCase, erreurs `{ code, message, details?, trace_id? }`, WS `type` en `UPPER_SNAKE` (référence : contexte projet)

#### Court terme (dans le sprint)

1. **Valider le chemin de déploiement** : `1.1-E2E-001` (compose smoke)
2. **Formaliser la stratégie de sélection** : tags `@p0/@p1` et jobs CI différenciés (smoke vs regression)

---

## PHASE 2 — DÉCISION QUALITY GATE

Phase 2 **non exécutée** : aucun résultat d’exécution de tests (CI/JUnit/rapport) fourni/détecté.

### Décision (préliminaire)

- **Décision** : FAIL (préliminaire)
- **Raison** : Couverture P0 = 0% (seuil requis = 100%) et absence d’évidence d’exécution

### Gate YAML (snippet)

```yaml
traceability:
  gate_type: story
  story_id: '1.1'
  date: '2026-01-22'
  coverage:
    overall: 0
    p0: 0
    p1: 0
  gaps:
    critical: 2
    high: 1
    medium: 0
    low: 0
  decision:
    status: 'FAIL'
    mode: 'deterministic'
    rationale: 'P0 coverage < 100% and no test execution evidence'
```
