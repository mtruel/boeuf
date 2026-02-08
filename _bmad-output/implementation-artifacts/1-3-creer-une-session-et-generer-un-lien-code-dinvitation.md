# Story 1.3: Créer une session et générer un lien/code d’invitation

Status: done

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

- [x] Modéliser la session et l'invite (AC: 1)
  - [x] Table `sessions` (snake_case) : `id`, `created_at`, `expires_at`, `active`...
  - [x] Table `session_invites` : `id`, `session_id`, `token_hash` (ou token opaque), `code` (optionnel), `expires_at`, `created_at`
  - [x] Table `session_participants` : `session_id`, `user_id`, `joined_at`, `role` (MVP), `last_seen_at` (préparation présence)

- [x] Endpoint création session (AC: 1)
  - [x] `POST /api/sessions` (auth requise)
  - [x] Crée la session + ajoute le créateur comme participant
  - [x] Crée une invite (token opaque) avec expiration (24h max, aligné NFR7)
  - [x] Retourne JSON `camelCase` : `{ sessionId, inviteUrl, inviteCode?, expiresAt }`

- [x] Frontend : CTA créer session + affichage lien/code (AC: 1)
  - [x] Bouton "Créer une session"
  - [x] UI : afficher lien copiables + code (si implémenté)
  - [x] UI : feedback copie (toast)

- [x] Conventions & sécurité (AC: 1)
  - [x] Contrôle d'accès via cookie-session : un utilisateur ne peut créer que pour lui-même
  - [x] Ne jamais exposer de secrets Spotify ni tokens dans la réponse

### Review Follow-ups (AI)

- [x] [AI-Review][HIGH] Externaliser SessionDuration (24h hardcodé) via env var ou config injectable [backend/internal/handlers/session.go:30]
- [x] [AI-Review][HIGH] Implémenter rate limiting sur création de sessions (max N sessions actives/user ou rate limit IP) [backend/internal/handlers/session.go:73]
- [x] [AI-Review][HIGH] Ajouter job de nettoyage périodique pour sessions/invites expirées (croissance DB non contrôlée) [backend/internal/models/session.go]
- [x] [AI-Review][HIGH] Clarifier commentaire base64url encoding (code correct mais commentaire ambigu) [backend/internal/handlers/session.go:63]
- [x] [AI-Review][MEDIUM] Valider méthode HTTP (rejeter non-POST sur endpoint) [backend/internal/handlers/session.go:73]
- [x] [AI-Review][MEDIUM] Renommer ou clarifier generateSessionID pour inviteID (préfixe sess_ inapproprié) [backend/internal/handlers/session.go:136]
- [x] [AI-Review][MEDIUM] Ajouter test pour création de sessions multiples par même utilisateur [backend/internal/handlers/session_test.go]
- [x] [AI-Review][MEDIUM] Ajouter gestion d'erreur pour formatExpirationDate (Invalid Date) [frontend/src/components/CreateSessionComponent.vue:148]
- [x] [AI-Review][MEDIUM] Documenter ou supprimer backend/Dockerfile.backup [backend/Dockerfile.backup]
- [x] [AI-Review][MEDIUM] Externaliser baseURL dans tests (hardcodé "<http://test-server>") [backend/internal/handlers/session_test.go:22]
- [x] [AI-Review][LOW] Ajouter test du fallback clipboard execCommand [frontend/src/components/CreateSessionComponent.spec.ts:107]
- [x] [AI-Review][LOW] Standardiser langue des commentaires (français vs anglais) [backend/internal/models/session.go:8]
- [x] [AI-Review][LOW] Documenter format token dans API spec (base64url, ~22 chars)

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

- ✅ **Backend endpoint `POST /api/sessions` implémenté** avec authentification via cookie-session, création de session/participant/invite, tokens sécurisés (hachés), expiration 24h, format JSON camelCase
- ✅ **Frontend composant CreateSessionComponent** avec bouton de création, affichage lien d'invitation, fonctionnalité copie avec feedback, gestion d'erreurs, UI responsive
- ✅ **Conventions de sécurité respectées** : pas d'exposition des tokens Spotify, contrôle d'accès par session, format d'erreur standardisé, tokens d'invitation hachés en DB
- ✅ **Tests complets ajoutés** : tests unitaires backend (création/sécurité), tests frontend (composant/interactions), couverture complète des AC
- ✅ **Intégration complète** : route ajoutée dans main.go, composant intégré dans HomeView, pas de régressions
- 🔄 **Refactoring (Review)** : externalisation de `PUBLIC_URL` dans `main.go`, nettoyage des magic strings/numbers dans `session.go`, mise à jour de la documentation.
- 📋 **Code Review 2026-01-27** : 13 issues identifiées (4 HIGH, 6 MEDIUM, 3 LOW) - action items créés dans Review Follow-ups section
- ✅ **Review Follow-ups Completed 2026-01-27** : Tous les 13 items de review résolus
  - [HIGH] Externalisation de SessionDuration via env var `SESSION_DURATION_HOURS`
  - [HIGH] Rate limiting implémenté avec `MAX_ACTIVE_SESSIONS_PER_USER` (default: 10)
  - [HIGH] Job de nettoyage périodique des sessions expirées (toutes les heures)
  - [HIGH] Commentaires clarifiés pour base64url encoding
  - [MEDIUM] Validation méthode HTTP (405 pour non-POST)
  - [MEDIUM] Fonction generateUniqueID créée pour sessions et invites
  - [MEDIUM] Tests ajoutés pour sessions multiples par utilisateur
  - [MEDIUM] Gestion d'erreur "Date invalide" dans formatExpirationDate
  - [MEDIUM] Suppression de Dockerfile.backup obsolète
  - [MEDIUM] Externalisation de baseURL dans tests (constante testBaseURL)
  - [LOW] Test ajouté pour fallback clipboard execCommand
  - [LOW] Standardisation des commentaires en anglais
  - [LOW] Documentation complète du format token dans API spec
- ✅ **Final Testing 2026-01-28** : Tests complets validés
  - Backend: 49 tests passing (session, security, cleanup, models)
  - Frontend: 28 tests passing (CreateSessionComponent, AuthStatus)
  - Makefile optimisé: tests sans rebuild (test-quick), pas d'intervention manuelle
  - Total: 77 tests passing

### File List

- `_bmad-output/implementation-artifacts/1-3-creer-une-session-et-generer-un-lien-code-dinvitation.md`
- `backend/internal/handlers/session.go`
- `backend/internal/handlers/session_test.go`
- `backend/internal/handlers/session_security_test.go`
- `backend/internal/handlers/cleanup.go` (NEW)
- `backend/internal/handlers/cleanup_test.go` (NEW)
- `backend/internal/models/session.go`
- `backend/internal/models/session_test.go`
- `backend/cmd/boeuf-server/main.go`
- `backend/Dockerfile.backup` (DELETED)
- `.env.example` (UPDATED)
- `frontend/src/components/CreateSessionComponent.vue`
- `frontend/src/components/CreateSessionComponent.spec.ts`
- `frontend/src/views/HomeView.vue`
