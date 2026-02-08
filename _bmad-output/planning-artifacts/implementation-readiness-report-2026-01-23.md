---
date: 2026-01-23
project: boeuf
stepsCompleted:
	- step-01-document-discovery
	- step-02-prd-analysis
	- step-03-epic-coverage-validation
	- step-04-ux-alignment
	- step-05-epic-quality-review
	- step-06-final-assessment
documents:
	prd: _bmad-output/planning-artifacts/prd.md
	architecture: _bmad-output/planning-artifacts/architecture.md
	epics: _bmad-output/planning-artifacts/epics.md
	ux_spec: _bmad-output/planning-artifacts/ux-design-specification.md
	ux_directions: _bmad-output/planning-artifacts/ux-design-directions.html
---

# Implementation Readiness Assessment Report

**Date:** 2026-01-23
**Project:** boeuf

## Step 1 — Document Discovery

### PRD Files Found

**Whole Documents:**

- prd.md (20121 bytes, modified 2026-01-22 20:19:37 +0100)

**Sharded Documents:**

- None found

### Architecture Files Found

**Whole Documents:**

- architecture.md (20435 bytes, modified 2026-01-22 23:02:12 +0100)

**Sharded Documents:**

- None found

### Epics & Stories Files Found

**Whole Documents:**

- epics.md (26961 bytes, modified 2026-01-22 23:21:30 +0100)

**Sharded Documents:**

- None found

### UX Design Files Found

**Whole Documents:**

- ux-design-specification.md (25620 bytes, modified 2026-01-22 21:40:24 +0100)
- ux-design-directions.html (30421 bytes, modified 2026-01-22 21:31:34 +0100)

**Sharded Documents:**

- None found

### Additional Planning Artifacts (Informational)

- brainstorming-session-2026-01-14.md (32302 bytes)
- bmm-workflow-status.yaml (1494 bytes)

## Issues Found

- No duplicate “whole vs sharded” documents detected.
- No missing documents detected for PRD/Architecture/Epics/UX (at least one per category).

## PRD Analysis

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

Total FRs: 29

### Non-Functional Requirements

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

Total NFRs: 16

### Additional Requirements

- MVP: web app (Vue.js + Go) avec Spotify uniquement, communication temps réel via WebSockets.
- Backend: Go; DB: SQLite (migration Postgres possible); OAuth 2.0 Spotify Web API.
- Cibles perf (MVP): FCP < 2s, TTI < 3s, WebSocket latency < 500ms, sync delay ≤ 3s.
- Support navigateurs: Chrome/Firefox/Safari (Latest-2) en haute priorité; mobile responsive.
- Accessibilité: basique MVP (clavier, contraste, labels principaux) ; WCAG AA post-MVP.
- RGPD applicable; chiffrement HTTPS/WSS; sessions expirent après 24h.
- Risque bloquant: capacités API Spotify à valider avant d'implémenter (spike technique).

### PRD Completeness Assessment

- Forces: exigences FR/NFR explicites et numérotées; cibles de performance claires; edge-cases (déconnexion, conflits) identifiés.
- Zones à clarifier avant implémentation: règles exactes de “host tournant” (définition d’host, autorité d’écriture), stratégie de résolution de conflits (first vs last) et traçabilité, modèle de session (code, permissions), gestion des erreurs OAuth/Spotify (UX), et critères d’expiration/cleanup des sessions.

## Epic Coverage Validation

### Coverage Matrix

| FR Number | PRD Requirement | Epic Coverage | Status |
| --------- | --------------- | ------------ | ------ |
| FR1 | Un utilisateur peut créer une nouvelle session d'écoute | Epic 1 | ✓ Covered |
| FR2 | Un utilisateur peut générer un lien de partage pour sa session | Epic 1 | ✓ Covered |
| FR3 | Un utilisateur peut rejoindre une session existante via un lien ou code | Epic 1 | ✓ Covered |
| FR4 | Un utilisateur peut quitter une session | Epic 4 | ✓ Covered |
| FR5 | Un utilisateur peut voir les participants actuels de la session | Epic 1 | ✓ Covered |
| FR6 | Un utilisateur peut connecter son compte Spotify via OAuth | Epic 1 | ✓ Covered |
| FR7 | Un utilisateur peut déconnecter son compte Spotify | Epic 4 | ✓ Covered |
| FR8 | Le système peut lire l'état de lecture actuel d'un utilisateur (morceau, position, état play/pause) | Epic 1 | ✓ Covered |
| FR9 | Le système peut contrôler la lecture d'un utilisateur (play, pause, skip) | Epic 1 | ✓ Covered |
| FR10 | Le système peut lire la file d'attente d'un utilisateur | Epic 2 | ✓ Covered |
| FR11 | Le système peut modifier la file d'attente d'un utilisateur | Epic 2 | ✓ Covered |
| FR12 | Quand un participant lance un morceau, il se lance chez tous les participants | Epic 1 | ✓ Covered |
| FR13 | Quand un participant met pause, la lecture se met en pause chez tous les participants | Epic 1 | ✓ Covered |
| FR14 | Quand un participant reprend la lecture, elle reprend chez tous les participants | Epic 1 | ✓ Covered |
| FR15 | Quand un participant ajoute un morceau à la file d'attente, il apparaît chez tous les participants | Epic 2 | ✓ Covered |
| FR16 | Quand un participant réorganise la file d'attente, elle se réorganise chez tous les participants | Epic 2 | ✓ Covered |
| FR17 | La synchronisation respecte une tolérance de ≤3 secondes de décalage | Epic 1 | ✓ Covered |
| FR18 | Le participant qui effectue une action devient automatiquement le host | Epic 3 | ✓ Covered |
| FR19 | Le système sélectionne automatiquement un nouveau host si le host actuel se déconnecte | Epic 3 | ✓ Covered |
| FR20 | Le changement de host est transparent pour les participants | Epic 3 | ✓ Covered |
| FR21 | Le système détecte quand un participant perd sa connexion | Epic 4 | ✓ Covered |
| FR22 | Un participant déconnecté peut se reconnecter à sa session | Epic 4 | ✓ Covered |
| FR23 | Un participant reconnecté est automatiquement resynchronisé avec l'état actuel de la session | Epic 4 | ✓ Covered |
| FR24 | La musique continue de jouer chez les autres participants pendant une déconnexion | Epic 4 | ✓ Covered |
| FR25 | Quand deux participants agissent simultanément, le système résout le conflit (first wins ou last wins) | Epic 3 | ✓ Covered |
| FR26 | Le système ne bloque pas les actions en cas de conflit | Epic 3 | ✓ Covered |
| FR27 | Un utilisateur peut voir le morceau actuellement en cours de lecture | Epic 1 | ✓ Covered |
| FR28 | Un utilisateur peut voir la file d'attente partagée | Epic 2 | ✓ Covered |
| FR29 | Un utilisateur peut voir l'état de connexion des participants | Epic 1 | ✓ Covered |

### Missing Requirements

- Aucun FR manquant: la “FR Coverage Map” dans epics.md couvre FR1–FR29.
- Aucun FR “extra” détecté côté epics (la liste FR inventoriée correspond à celle du PRD).

### Coverage Statistics

- Total PRD FRs: 29
- FRs covered in epics: 29
- Coverage percentage: 100%

## UX Alignment Assessment

### UX Document Status

- Found: ux-design-specification.md
- Supporting: ux-design-directions.html

### UX ↔ PRD Alignment

- Aligné sur le cœur produit: “Trust but Verify”, usage Spotify natif + tableau de bord boeuf.
- Le pattern “Sas / Airlock” (bouton explicite “Start Listening / Sync Now”) correspond au besoin d’éviter l’autoplay et clarifie le modèle mental.
- Les flows d’erreur (autoplay blocked, désync >3s, token expiré) sont cohérents avec NFRs (perf/fiabilité) et exigences de résilience.

### UX ↔ Architecture Alignment

- Fort alignement sur: WebSockets temps réel, resync (snapshot + events since eventSeq), idempotence (clientMsgId), et “polling + verify” côté Spotify.
- Les composants UI prévus (toasts, banner, side panel social, indicateurs état) sont soutenus par l’architecture (event log + protocole WS + format erreurs).

### Alignment Issues / Gaps

1) **Conflit UX vs sécurité sur le stockage des clés Spotify**
 - UX (Journey 2) propose “Save Keys to LocalStorage” après input “Client ID & Secret”.
 - Architecture + epics imposent “aucun secret durable côté frontend” (tokens/clé maître côté serveur).
 - Action: ajuster l’UX flow pour stocker ces valeurs côté serveur (env/DB chiffrée) et ne garder côté client qu’un état “configured”.

2) **Onboarding Admin vs MVP scope**
 - L’UX mentionne un onboarding “Admin” pour configurer une app Spotify + whitelisting.
 - Le PRD présente plutôt un produit perso (Mathias + amis) et ne clarifie pas si cette configuration est “self-hosted” uniquement.
 - Action: préciser dans PRD/architecture si boeuf est uniquement self-hosted (probable) et si l’onboarding admin doit exister dans l’app (ou documenté hors-app).

### Warnings

- Aucun document UX manquant.
- Le point “LocalStorage secrets” est un **bloqueur sécurité** avant implémentation UI.

## Epic Quality Review

### Overall Assessment

- Structure globale solide: epics orientées valeur utilisateur, FR coverage map explicite, et stories avec AC testables pour la plupart.
- Les epics sont séquencées de manière logique (Epic 1 → 2 → 3 → 4), avec Epic 5 en “cross-cutting” qualité.

### 🔴 Critical Violations

1) **Secrets côté frontend (déjà identifié) doit être résolu AVANT implémentation**
 - Impact: fuite potentielle de secrets, impossibilité de respecter NFR4/NFR5.
 - Remediation: supprimer toute persistance de “Client Secret” côté browser; stocker côté serveur (env/DB chiffrée) et ajuster flux UX.

### 🟠 Major Issues

1) **Epic 5 contient des stories “tech-only” (ex: Story 5.1 “As a développeur”)**
 - Risque: dérive vers milestones techniques, moins traçable au user value.
 - Remediation: soit (a) rephraser en valeur utilisateur (“messages d’erreur actionnables et cohérents”), soit (b) intégrer ces exigences dans les stories des epics 1–4 comme critères d’acceptation transverses.

2) **Politique de conflit pas totalement tranchée** (Story 3.4)
 - Aujourd’hui: “policy documentée (ex: first-wins)”.
 - Remediation: décider explicitement MVP (recommandation: first-wins par ordre de réception serveur) + définir comment l’UI reflète l’arbitrage (timeline + toast non bloquant).

3) **Queue reordering (Story 2.3) est un gros morceau et potentiellement risqué**
 - Spotify ne supportant pas un réordonnancement direct, la story implique un modèle “desired queue” + convergence.
 - Remediation: scinder en 2–3 stories: (a) modèle “desired queue” + rendu UI, (b) stratégie de convergence minimale (best-effort), (c) gestion des divergences/erreurs + “Trust but Verify”.

4) **Pré-requis Spotify Premium / device actif** (implicite)
 - Architecture mentionne Player API (Premium requis) et contrôle playback dépend d’un device actif.
 - Remediation: ajouter une story/ACs sur détection + UX guidance (“Open Spotify on a device”, erreurs `SPOTIFY_NO_ACTIVE_DEVICE`, etc.) pour éviter blocage utilisateur.

### 🟡 Minor Concerns

- Duplication de header “FR Coverage Map” (cosmétique) dans epics.md.
- Certaines ACs gagneraient à inclure un cas d’erreur systématique (ex: WS auth/session invalide, token expiré) même si couvert partiellement ailleurs.

### Dependency / Independence Notes

- Epic 2/3/4 dépendent naturellement d’Epic 1 (session + auth + WS), ce qui est acceptable.
- Pas de dépendances “forward” détectées à l’intérieur des epics (les stories sont séquencées sans référence explicite à des stories futures).

## Summary and Recommendations

**Assessor:** Winston (Architect)
**Assessment Date:** 2026-01-23

### Overall Readiness Status

NEEDS WORK

### Critical Issues Requiring Immediate Action

1) **Sécurité: aucun secret persistant côté frontend**
 - Statut: **corrigé dans les artefacts** (UX + epics) — credentials Spotify configurés côté serveur (MVP self-hosted).
 - Reste à faire: s’assurer que l’implémentation respecte strictement ce principe.

### Recommended Next Steps

1. **Finaliser le modèle “keys/config Spotify” côté implémentation**: self-hosted via `SPOTIFY_CLIENT_ID/SECRET` (env/.env) et supprimer toute UI qui demande/stockerait des secrets côté navigateur.
2. **Trancher la policy de conflits** (first-wins recommandé) et documenter le comportement UI (timeline + feedback) pour réduire l’ambiguïté d’implémentation.
3. **Scinder Story 2.3 (queue reorder)** et définir une convergence MVP réaliste (fonctionne même si Spotify ne permet pas tout).
4. **Ajouter une story/ACs sur prérequis Spotify** (Premium + device actif) avec messages et erreurs stables.
5. **Revoir Epic 5**: rephraser en valeur utilisateur ou intégrer comme critères transverses dans Epics 1–4.

### Final Note

Cette revue n’a pas détecté de gaps FR (29/29 couverts) ni de documents manquants. Les principaux risques sont de nature **sécurité** (secrets côté client) et **ambigüité d’implémentation** (policy conflits, queue reorder). Corriger les points critiques avant de démarrer l’implémentation de l’UI et des flows d’onboarding.
