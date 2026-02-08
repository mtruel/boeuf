# Story 1.3: Créer une session et générer un lien/code d’invitation

Status: ready-for-dev

## Story

As a host (initiateur),
I want créer une session et obtenir un lien/code,
so that je puisse inviter quelqu’un rapidement.

## Acceptance Criteria

1. **Création session + invite**
   - **Given** un utilisateur authentifié Spotify
   - **When** il crée une session
   - **Then** le serveur crée une session avec une invite opaque (code/lien) expirable
   - **And** le frontend affiche un lien partageable et un code court (si applicable)

## Tasks / Subtasks

- [ ] Modéliser la session et l’invite (AC: 1)
  - [ ] Table `sessions` (snake_case) : `id`, `created_at`, `expires_at`, `active`...
  - [ ] Table `session_invites` : `id`, `session_id`, `token_hash` (ou token opaque), `code` (optionnel), `expires_at`, `created_at`
  - [ ] Table `session_participants` : `session_id`, `user_id`, `joined_at`, `role` (MVP), `last_seen_at` (préparation présence)

- [ ] Endpoint création session (AC: 1)
  - [ ] `POST /api/sessions` (auth requise)
  - [ ] Crée la session + ajoute le créateur comme participant
  - [ ] Crée une invite (token opaque) avec expiration (24h max, aligné NFR7)
  - [ ] Retourne JSON `camelCase` : `{ sessionId, inviteUrl, inviteCode?, expiresAt }`

- [ ] Frontend : CTA créer session + affichage lien/code (AC: 1)
  - [ ] Bouton “Créer une session”
  - [ ] UI : afficher lien copiables + code (si implémenté)
  - [ ] UI : feedback copie (toast)

- [ ] Conventions & sécurité (AC: 1)
  - [ ] Contrôle d’accès via cookie-session : un utilisateur ne peut créer que pour lui-même
  - [ ] Ne jamais exposer de secrets Spotify ni tokens dans la réponse

## Dev Notes

### Token d’invitation

- L’invite doit être **opaque** (imprévisible). Recommandation : 128 bits min de random, encodé base64url.
- Stockage recommandé : hash en DB (pour éviter fuite DB) + comparaison constante.

### Expiration

- L’invite doit être expirable; aligner sur la règle sessions inactives ≤ 24h.

### Format JSON / erreurs

- JSON en `camelCase`.
- Erreurs stables (ex: `UNAUTHENTICATED`, `FORBIDDEN`, `VALIDATION_ERROR`).

## Testing Requirements

- Tests Go :
  - création session crée 1 session + 1 participant + 1 invite
  - invite expiration correctement calculée
  - réponse n’expose pas de token Spotify

## References

- Source: `_bmad-output/planning-artifacts/epics.md` (Story 1.3)
- Source: `_bmad-output/planning-artifacts/architecture.md` (invite opaque expirable, contrôle d’accès)
- Source: `_bmad-output/project-context.md` (DB snake_case, JSON camelCase)
- Source: `_bmad-output/implementation-artifacts/sprint-1-plan.md` (story dans scope)

## Dev Agent Record

### Agent Model Used

GPT-5.2

### Debug Log References

- N/A (story prep)

### Completion Notes List

- Story préparée avec DB et API minimales, en gardant la compatibilité avec la gouvernance (host) et présence des stories suivantes.

### File List

- `_bmad-output/implementation-artifacts/1-3-creer-une-session-et-generer-un-lien-code-dinvitation.md`
