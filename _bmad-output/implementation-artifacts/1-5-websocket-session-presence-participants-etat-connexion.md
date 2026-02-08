# Story 1.5: WebSocket session + présence (participants + état connexion)

Status: ready-for-dev

## Story

As a utilisateur,
I want voir qui est présent et leur état de connexion,
So that je sache si on écoute vraiment "ensemble".

## Acceptance Criteria

1. **WebSocket connection avec authentification session**
   - **Given** un utilisateur a rejoint une session (participant valide)
   - **When** il se connecte au WebSocket
   - **Then** l'authentification se fait via le cookie-session existant (réutilisation de l'auth REST)
   - **And** la connexion est établie en WSS (sécurisé)

2. **Snapshot initial de session**
   - **Given** un utilisateur se connecte au WebSocket d'une session
   - **When** la connexion est établie
   - **Then** il reçoit un snapshot initial incluant:
     - Liste des participants (userId, role, lastSeenAt, connectionStatus)
     - État "now playing" minimal (trackId, trackName, artist, isPlaying, position)
     - sessionId et eventSeq actuel
   - **And** le message respecte l'enveloppe JSON standard

3. **Événements de présence temps réel**
   - **Given** plusieurs participants connectés à la même session
   - **When** un nouveau participant rejoint (WS connect)
   - **Then** tous les participants reçoivent un événement `PARTICIPANT_JOINED`
   - **And** l'événement contient userId, role, timestamp

   - **When** un participant se déconnecte (WS disconnect)
   - **Then** tous les autres participants reçoivent un événement `PARTICIPANT_LEFT`
   - **And** le serveur détecte la déconnexion en ≤ 5 secondes (via ping/pong ou timeout)

4. **Enveloppe WebSocket standard**
   - **Given** tout message WebSocket envoyé par le serveur
   - **When** il est reçu par le client
   - **Then** il respecte l'enveloppe JSON avec au minimum:
     - `type`: string en UPPER_SNAKE (ex: "SESSION_SNAPSHOT", "PARTICIPANT_JOINED")
     - `sessionId`: string
     - `eventSeq`: number (monotone par session, géré serveur)
     - `sentAt`: string RFC3339 UTC (ex: "2026-01-28T12:34:56Z")
     - `payload`: object (contenu spécifique à l'événement)
   - **And** les timestamps sont toujours en ISO-8601 UTC

5. **Gestion des erreurs WebSocket**
   - **Given** un utilisateur tente de se connecter sans être participant de la session
   - **When** il ouvre la connexion WS
   - **Then** le serveur ferme la connexion avec un code d'erreur stable
   - **And** un message d'erreur est envoyé avant fermeture (ex: `WS_FORBIDDEN`)

## Tasks / Subtasks

- [ ] Backend: WebSocket hub infrastructure (AC: 1, 2, 3, 4)
  - [ ] Endpoint `/ws` qui accepte connexions WebSocket
  - [ ] Upgrade HTTP → WebSocket avec validation cookie-session
  - [ ] Manager de connexions par session (map sessionId → [clients])
  - [ ] Gestion lifecycle: connect, disconnect, cleanup
  - [ ] Mécanisme de détection déconnexion (ping/pong ou read timeout ≤ 5s)

- [ ] Backend: Messages et événements WebSocket (AC: 2, 3, 4)
  - [ ] Enveloppe message standard avec type/sessionId/eventSeq/sentAt/payload
  - [ ] Message `SESSION_SNAPSHOT` envoyé à la connexion
  - [ ] Événement `PARTICIPANT_JOINED` broadcasté lors de nouvelles connexions
  - [ ] Événement `PARTICIPANT_LEFT` broadcasté lors de déconnexions
  - [ ] Générateur d'eventSeq monotone par session (stocké en DB ou en mémoire MVP)

- [ ] Backend: Contrôle d'accès WebSocket (AC: 1, 5)
  - [ ] Vérifier que userId (depuis cookie-session) est participant de la session demandée
  - [ ] Refuser connexion avec erreur stable si non-participant
  - [ ] Logger tentatives d'accès non autorisées

- [ ] Backend: Modèle de données présence (AC: 2, 3)
  - [ ] Enrichir `SessionParticipant` avec champ `ConnectionStatus` (online/offline)
  - [ ] Mettre à jour `LastSeenAt` à chaque activité
  - [ ] Gérer état de connexion en mémoire (+ sync DB périodique ou événement-driven)

- [ ] Frontend: Client WebSocket Pinia store (AC: 1, 2, 3, 4)
  - [ ] Store `useRealtimeStore` pour gérer connexion WebSocket
  - [ ] Établir connexion WS vers `/ws?sessionId=XXX` (ou via path)
  - [ ] Parser messages JSON et dispatcher vers stores domaines
  - [ ] Gérer états: connecting, connected, disconnected, error
  - [ ] Recevoir et traiter `SESSION_SNAPSHOT` pour initialiser état local

- [ ] Frontend: Store présence participants (AC: 2, 3)
  - [ ] Store `usePresenceStore` (ou intégré dans `useSessionStore`)
  - [ ] Maintenir liste de participants avec statuts connexion
  - [ ] Réagir aux événements `PARTICIPANT_JOINED` et `PARTICIPANT_LEFT`
  - [ ] Exposer computed pour UI (nombre participants, liste avec statuts)

- [ ] Frontend: UI présence (AC: 3)
  - [ ] Composant affichant liste des participants
  - [ ] Indicateurs visuels: online (vert), offline (gris)
  - [ ] Affichage rôle (host/participant)
  - [ ] Mise à jour temps réel lors des événements

- [ ] Tests Backend (AC: tous)
  - [ ] Test connexion WS avec cookie-session valide → snapshot reçu
  - [ ] Test connexion sans authentification → refusée
  - [ ] Test connexion utilisateur non-participant → refusée
  - [ ] Test broadcast `PARTICIPANT_JOINED` lors de nouvelle connexion
  - [ ] Test broadcast `PARTICIPANT_LEFT` lors de déconnexion
  - [ ] Test format enveloppe message (tous champs requis)
  - [ ] Test eventSeq monotone

- [ ] Tests Frontend (AC: tous)
  - [ ] Test connexion réussie et réception snapshot
  - [ ] Test parsing événements PARTICIPANT_JOINED/LEFT
  - [ ] Test mise à jour store présence
  - [ ] Test affichage UI participants (mock WS messages)

## Dev Notes

### Architecture WebSocket

**Backend:**

- Utiliser la bibliothèque standard `gorilla/websocket` (déjà un standard Go)
- Structure suggérée:
  - `internal/realtime/hub.go`: Hub central gérant toutes les connexions
  - `internal/realtime/client.go`: Représente une connexion client WebSocket
  - `internal/realtime/message.go`: Types de messages et enveloppe standard
  - `internal/handlers/websocket.go`: Handler HTTP → WS upgrade

**Cycle de vie connexion:**

1. Client HTTP GET `/ws?sessionId=XXX` avec cookie-session
2. Backend valide authentification et participation
3. Upgrade HTTP → WebSocket (gorilla/websocket.Upgrader)
4. Enregistrer client dans Hub (map par sessionId)
5. Envoyer SESSION_SNAPSHOT au nouveau client
6. Broadcaster PARTICIPANT_JOINED aux autres
7. Boucle read: attendre messages client (pour future, actuellement juste keepalive)
8. Boucle write: envoyer événements depuis Hub vers client
9. Détection déconnexion: timeout ou erreur read
10. Cleanup: retirer du Hub, broadcaster PARTICIPANT_LEFT

**Frontend:**

- Store Pinia centralisé pour gérer la connexion WebSocket unique par session
- Pattern dispatcher: réception message → dispatch vers store approprié selon `type`
- Gestion reconnexion automatique sera Story 4.2 (pas dans cette story)

### Enveloppe WebSocket Standard

Tous les messages serveur → client respectent ce format:

```json
{
  "type": "SESSION_SNAPSHOT",
  "sessionId": "sess_abc123",
  "eventSeq": 42,
  "sentAt": "2026-01-28T12:34:56.789Z",
  "payload": {
    "participants": [
      {
        "userId": "user_xyz",
        "role": "host",
        "lastSeenAt": "2026-01-28T12:34:55Z",
        "connectionStatus": "online"
      }
    ],
    "nowPlaying": {
      "trackId": "spotify:track:...",
      "trackName": "Example Song",
      "artist": "Example Artist",
      "isPlaying": true,
      "positionMs": 45000
    }
  }
}
```

Types d'événements (cette story):

- `SESSION_SNAPSHOT`: envoyé à la connexion (snapshot complet)
- `PARTICIPANT_JOINED`: nouveau participant connecté
- `PARTICIPANT_LEFT`: participant déconnecté

Future (autres stories): `PLAYER_PAUSED`, `PLAYER_RESUMED`, `TRACK_CHANGED`, `QUEUE_UPDATED`, `HOST_CHANGED`, etc.

### EventSeq Monotone

- Le serveur maintient un compteur `eventSeq` par session
- Incrémenté à chaque événement broadcasté
- Permet au client de détecter des messages manqués (future: resync)
- MVP: peut être en mémoire (Hub), future: persister dans event log DB

### Détection Déconnexion

Impératif de détecter en ≤ 5 secondes (NFR8):

- Option 1: WebSocket ping/pong avec timeout 5s
- Option 2: Read timeout sur socket (si aucun message en 5s, considérer déconnecté)
- Recommandation MVP: combo des deux (ping/pong actif + read deadline)

### Contrôle d'Accès

Réutiliser la logique existante `RequireParticipant` de Story 1.4:

- Extraire userId du cookie-session
- Extraire sessionId de la requête WS (query param ou path)
- Vérifier existence dans `session_participants`
- Si échec: ne pas upgrader, retourner 403 ou close WS immédiatement

### Learnings des Stories Précédentes

**Story 1.1** - Infrastructure:

- Docker Compose et Caddy déjà configurés
- HTTPS/WSS fonctionnel en production
- Variables d'environnement pour secrets

**Story 1.2** - Auth Spotify:

- Cookie-session fonctionnelle (gorilla/sessions)
- Pattern d'extraction userId établi dans handlers

**Story 1.3** - Créer session:

- Modèles `Session`, `SessionParticipant`, `SessionInvite` existants
- `SessionHandler` pour logique métier session

**Story 1.4** - Contrôle d'accès:

- Middleware `RequireParticipant` déjà implémenté
- Pattern de validation session/participant établi
- Tests montrent comment mocker DB et sessions

### Intégration avec Code Existant

**Backend files à créer:**

- `backend/internal/realtime/` (nouveau package)
- `backend/internal/handlers/websocket.go` (nouveau handler)

**Backend files à modifier:**

- `backend/cmd/boeuf-server/main.go`: enregistrer route `/ws`
- `backend/internal/models/session.go`: ajouter `ConnectionStatus` à `SessionParticipant` (ou gérer en mémoire pour MVP)

**Frontend files à créer:**

- `frontend/src/stores/realtime.ts`: store WebSocket central
- `frontend/src/stores/presence.ts`: store participants et présence (ou intégrer dans session store)
- `frontend/src/components/ParticipantsList.vue`: UI liste participants

**Frontend files à modifier:**

- `frontend/src/views/SessionView.vue`: afficher composant participants (à créer dans story future si pas existant)

### Testing Strategy

**Backend:**

- Tests unitaires: enveloppe message, eventSeq monotone
- Tests intégration: upgrade WS avec auth, broadcast événements
- Utiliser `gorilla/websocket.NewClient()` ou équivalent pour simuler clients
- Tester refus accès (non-participant, non-authentifié)

**Frontend:**

- Mock WebSocket dans tests via `vi.mock()` ou classe mock
- Tester parsing messages et dispatch vers stores
- Tester updates réactifs de l'UI

### Library Versions

**Backend:**

```go
// go.mod
require (
    github.com/gorilla/websocket v1.5.3  // Latest stable
)
```

**Frontend:**

- WebSocket natif du navigateur (API standard)
- Pas de bibliothèque tierce nécessaire pour MVP

### Performance Considerations

- Broadcast événements de manière efficace (éviter N messages individuels)
- Limiter taille snapshot (pas de données inutiles)
- Buffer messages sortants pour éviter blocking
- Gérer goroutines proprement (pas de leak)

### Security Considerations

- Toujours WSS en production (via Caddy TLS)
- Valider sessionId avant upgrade (injection)
- Ne jamais broadcaster des données privées à des non-participants
- Logger tentatives d'accès non autorisées

### UX Considerations

- Indicateur de connexion WebSocket dans UI (connecté/déconnecté)
- Afficher état "connecting..." pendant établissement connexion
- Messages d'erreur clairs si connexion échoue
- Éviter surprises UI: annoncer présence dès connexion

### Monitoring & Observability

- Logger événements WS: connexions, déconnexions, broadcasts
- Métriques suggérées:
  - Nombre de connexions actives par session
  - Nombre d'événements broadcastés par type
  - Latence broadcast (temps entre événement et envoi)
  - Erreurs WS (upgrades refusés, timeouts, panics)

## Testing Requirements

**Backend Tests Minimum:**

- Connexion WS réussie avec authentification valide
- Réception snapshot correct à la connexion
- Refus connexion si non-participant
- Broadcast PARTICIPANT_JOINED lors de nouvelle connexion
- Broadcast PARTICIPANT_LEFT lors de déconnexion
- Validation format enveloppe message
- EventSeq monotone et incrémenté correctement

**Frontend Tests Minimum:**

- Connexion réussie et transition états (connecting → connected)
- Parsing snapshot et mise à jour store
- Parsing événements présence et mise à jour liste participants
- Affichage UI reflète état présence correctement

## Project Structure Notes

### Backend Structure (à créer)

```
backend/
  internal/
    realtime/
      hub.go          # Hub central (connexions, broadcast)
      client.go       # Client WebSocket individuel
      message.go      # Types messages et enveloppe
      hub_test.go     # Tests Hub
      message_test.go # Tests enveloppe
    handlers/
      websocket.go    # Handler upgrade HTTP → WS
      websocket_test.go
```

### Frontend Structure (à créer)

```
frontend/
  src/
    stores/
      realtime.ts       # Store WebSocket central
      presence.ts       # Store participants/présence
    components/
      ParticipantsList.vue      # UI liste participants
      ConnectionIndicator.vue   # Indicateur connexion WS (optionnel)
```

### Integration avec Architecture Existante

- **Backend:** Nouveau package `realtime` séparé, appelé depuis handlers
- **Frontend:** Nouveaux stores Pinia s'intègrent avec stores existants
- **Communication:** WebSocket complète l'architecture REST existante
- **Sécurité:** Réutilise cookie-session et middleware existants

## References

- **Source:** [_bmad-output/planning-artifacts/epics.md](../_bmad-output/planning-artifacts/epics.md) (Story 1.5)
- **Source:** [_bmad-output/planning-artifacts/architecture.md](../_bmad-output/planning-artifacts/architecture.md#api--communication-patterns) (WebSocket patterns, enveloppe message)
- **Source:** [_bmad-output/project-context.md](../_bmad-output/project-context.md) (Conventions naming, formats)
- **Learnings:** [1-4-rejoindre-une-session-via-lien-code-avec-controle-dacces.md](./1-4-rejoindre-une-session-via-lien-code-avec-controle-dacces.md) (Contrôle d'accès, cookie-session)
- **Library:** [gorilla/websocket](https://github.com/gorilla/websocket) (Go WebSocket library standard)
- **Spec:** [RFC 6455 - WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- **Spec:** [RFC 3339 - Date/Time Format](https://tools.ietf.org/html/rfc3339)

## Dev Agent Record

### Agent Model Used

_À remplir lors de l'implémentation_

### Debug Log References

_À remplir lors de l'implémentation_

### Completion Notes List

_À remplir lors de l'implémentation_

### File List

_À remplir lors de l'implémentation_
