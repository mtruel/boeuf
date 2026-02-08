# Story 1.4: Rejoindre une session via lien/code (avec contrôle d’accès)

Status: ready-for-dev

## Story

As a invité,
I want rejoindre une session via un lien/code,
so that je puisse écouter avec le groupe sans friction.

## Acceptance Criteria

1. **Join via invite**
   - **Given** un lien/code valide
   - **When** l’invité l’ouvre
   - **Then** il rejoint la session après OAuth (si nécessaire)
   - **And** un lien/code invalide retourne une erreur stable (`SESSION_NOT_FOUND` ou équivalent)

2. **Contrôle d’accès session (NFR6)**
   - **Given** un utilisateur authentifié qui n’est pas participant d’une session
   - **When** il tente d’accéder à ses ressources (REST/WS)
   - **Then** le serveur refuse l’accès avec une erreur stable (ex: `FORBIDDEN`)
   - **And** l’utilisateur ne peut accéder qu’aux sessions où il est participant

## Tasks / Subtasks

- [ ] Routing “magic link” côté frontend (AC: 1)
  - [ ] Route `/join/:token` (ou query `?code=`) qui déclenche le flow join
  - [ ] Si utilisateur non connecté Spotify : CTA “Connecter Spotify” puis reprendre join

- [ ] Endpoint join backend (AC: 1)
  - [ ] `POST /api/sessions/join` avec payload `{ inviteToken }` (ou `{ code }`)
  - [ ] Valider invite : existante + non expirée
  - [ ] Ajouter (ou upsert) participant dans `session_participants`
  - [ ] Répondre `{ sessionId }` (et éventuellement détails minimal session)

- [ ] Erreurs stables join (AC: 1)
  - [ ] Invite invalide/expirée → `SESSION_NOT_FOUND` (ou `INVITE_INVALID`), message actionnable
  - [ ] Non authentifié → `UNAUTHENTICATED`

- [ ] Contrôle d’accès sur endpoints session (AC: 2)
  - [ ] Middleware : vérifier que `user_id` est participant de `session_id` avant accès
  - [ ] Retourner `FORBIDDEN` (stable) si non participant

- [ ] Préparer l’intégration WS (sans implémenter WS complet) (AC: 2)
  - [ ] S’assurer que le modèle de session/participants supporte l’auth WS via cookie-session

## Dev Notes

### Join UX (zéro friction)

- Objectif “link-to-music” : un clic sur le lien doit amener au bon écran.
- Si OAuth requis : après callback, reprendre automatiquement l’opération join (state interne côté serveur, ou paramètre de retour).

### Contrôle d’accès

- Toutes les routes session (REST et WS plus tard) doivent vérifier la participation.
- Éviter les fuites d’information : une session inconnue ou non autorisée doit répondre avec des codes stables et des messages neutres.

## Testing Requirements

- Tests Go :
  - join avec invite valide → participant créé
  - join avec invite expirée → `SESSION_NOT_FOUND`
  - accès à une ressource session sans être participant → `FORBIDDEN`

## References

- Source: `_bmad-output/planning-artifacts/epics.md` (Story 1.4)
- Source: `_bmad-output/planning-artifacts/architecture.md` (contrôle d’accès par session, cookie-session)
- Source: `_bmad-output/project-context.md` (format erreurs, conventions)
- Source: `_bmad-output/implementation-artifacts/sprint-1-plan.md` (story dans scope)

## Dev Agent Record

### Agent Model Used

GPT-5.2

### Debug Log References

- N/A (story prep)

### Completion Notes List

- Story préparée en priorisant contrôle d’accès NFR6 + erreurs stables pour éviter les fuites et faciliter le frontend.

### File List

- `_bmad-output/implementation-artifacts/1-4-rejoindre-une-session-via-lien-code-avec-controle-dacces.md`
