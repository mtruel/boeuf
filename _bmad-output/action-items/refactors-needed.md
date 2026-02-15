# Refactors Needed

**Date:** 2026-02-11

- Scinder la logique des sessions (handlers/services/domain)
- Decoupler la logique session de Spotify (interfaces/adapters)
- Backend source de verite pour etats/droits + validation seek (ne pas decider cote UI)
- Remplacer la logique front de retry/backoff + polling player (playerStore) par des mecanismes backend/resync WS
