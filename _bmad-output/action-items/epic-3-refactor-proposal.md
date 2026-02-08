# Epic 3 Refactor Proposal - Sync Leader Model

**Date:** 2026-02-01  
**Context:** Alignment avec Story 1.10 (Room Owner vs Sync Leader clarification)  
**Status:** PROPOSAL - En attente validation Mathias

---

## Summary of Changes

**BEFORE (Epic 3 original):**

- Ambiguous "host" concept mixing ownership + sync
- "User wants to see who is host" → UI prominence
- Unclear relationship with polling mechanism

**AFTER (Epic 3 refactorisé):**

- Clear separation: Room Owner (permanent) vs Sync Leader (technical)
- **Sync Leader visible via discrete badge only** (⚡ or similar)
- No notifications/toasts when sync leader changes
- Explicit relationship with Spotify polling (Epic 3 enables Story 1.8 optimization)

---

## Proposed Epic 3 - Full Rewrite

### Epic 3: Sync Leadership & Playback Orchestration

**Goal:** Établir un mécanisme de "sync leader" transparent qui détermine quelle source Spotify le serveur poll pour orchestrer la synchronisation playback de la room.

**Key Principles:**

1. **Sync Leader = Technical Mechanism:** L'utilisateur dont le compte Spotify sert de source de vérité pour le polling serveur
2. **Transparent Assignment:** Changement automatique basé sur les actions, sans friction
3. **Discrete Visibility:** Badge visuel discret (⚡) dans la liste participants, SANS notifications/toasts
4. **Room Owner ≠ Sync Leader:** Concepts distincts (owner = permanent/manages room, sync leader = temporary/polling source)

**Context (Story 1.10 Integration):**

- Story 1.10 ajoute `created_by` (room owner) - permanent, visible
- Epic 3 ajoute `sync_leader_user_id` - dynamique, badge discret seulement
- Optimise Story 1.8 polling: au lieu de poller tous les participants, poll uniquement sync leader

---

### Story 3.0: Initial Sync Leader Assignment

**NEW STORY** - Couvre le cas non traité: qui devient sync leader quand room devient LIVE?

**As a backend system,**  
**I want** automatically assign a sync leader when a room becomes LIVE,  
**So that** polling can begin immediately without ambiguity.

#### Acceptance Criteria

**AC 3.0.1: First Participant Becomes Sync Leader**

- **Given** une room STALE (0 participants actifs, `sync_leader_user_id = NULL`)
- **When** le premier participant rejoint la room (via POST `/api/sessions/:id/join`)
- **Then** ce participant devient automatiquement sync leader
- **And** `sync_leader_user_id` est set au `user_id` du participant
- **And** le serveur commence le polling Spotify sur ce user's account
- **And** room state passe STALE → LIVE

**AC 3.0.2: Creator Auto-Join on Creation**

- **Given** un utilisateur crée une nouvelle room (Story 1.10 AC1)
- **When** le backend auto-join le créateur après création
- **Then** le créateur devient sync leader par défaut (premier participant actif)
- **And** `sync_leader_user_id = created_by` (dans ce cas spécifique)

**AC 3.0.3: Sync Leader Badge Visible**

- **Given** un participant a `user_id == sync_leader_user_id`
- **When** la liste des participants est affichée dans l'UI
- **Then** un badge discret (⚡ ou "🎵" ou "●") apparaît à côté du nom du sync leader
- **And** AUCUN tooltip/notification n'explique le concept (reste mystérieux/découvrable)
- **And** le badge est le SEUL indicateur visuel (pas de highlight, pas de section dédiée)

**AC 3.0.4: No Notification on Assignment**

- **Given** un participant devient sync leader (initial assignment)
- **When** le changement est effectué
- **Then** AUCUNE notification/toast n'est affichée
- **And** AUCUN événement WebSocket UI n'est envoyé
- **And** seul le badge dans la participant list change silencieusement

---

### Story 3.1: Automatic Sync Leader Switch on Action

**REFACTORED** - Ancien "Qui agit devient host" → Clarified as backend mechanism

**As a backend system,**  
**I want** automatically switch sync leader to the user who takes playback actions,  
**So that** the polling source reflects the most active/relevant Spotify account.

#### Acceptance Criteria

**AC 3.1.1: Playback Action Triggers Switch**

- **Given** User A est le sync leader actuel (`sync_leader_user_id = userA`)
- **And** User B prend une action playback (play, pause, seek, skip)
- **When** le backend traite l'action de User B avec succès
- **Then** `sync_leader_user_id` est automatiquement changé vers `userB`
- **And** le polling Spotify switch immédiatement de User A account → User B account
- **And** AUCUNE notification n'est envoyée aux clients

**AC 3.1.2: Badge Updates Silently**

- **Given** le sync leader change de User A → User B
- **When** les clients reçoivent l'événement de présence/state update (déjà existant)
- **Then** le badge ⚡ disparaît de User A et apparaît à côté de User B dans la participant list
- **And** le changement est visuel seulement (pas de toast/modal)
- **And** les users ne reçoivent PAS de message "User B is now the sync leader"

**AC 3.1.3: Queue Actions Do NOT Switch**

- **Given** User A est sync leader
- **When** User B ajoute un titre à la queue (Epic 2)
- **Then** `sync_leader_user_id` reste User A (AUCUN changement)
- **Rationale:** Queue actions n'impactent pas playback state source → pas de raison de changer polling target

**AC 3.1.4: Multiple Rapid Actions**

- **Given** User A et User B prennent des actions quasi-simultanément
- **When** les actions arrivent au serveur dans l'ordre: A → B → A
- **Then** sync leader change à chaque fois: A → B → A
- **And** dernier à agir reste sync leader (pas de debouncing)
- **Rationale:** Simplicité > optimisation prématurée

---

### Story 3.2: Sync Leader Failover on Disconnect

**REFACTORED** - Ancien "Failover host" → Clarified technical mechanism

**As a backend system,**  
**I want** select a new sync leader when the current sync leader disconnects,  
**So that** polling can continue seamlessly without interruption.

#### Acceptance Criteria

**AC 3.2.1: Failover on Disconnect**

- **Given** User A est sync leader (`sync_leader_user_id = userA`)
- **And** il y a d'autres participants actifs dans la room (User B, User C)
- **When** User A se déconnecte (WebSocket close) OU fait POST `/api/sessions/:id/leave`
- **Then** le serveur détecte la perte en ≤ 5s (NFR8)
- **And** sélectionne automatiquement un nouveau sync leader parmi les participants actifs restants
- **And** `sync_leader_user_id` est mis à jour vers le nouvel élu

**AC 3.2.2: Failover Selection Strategy**

- **Given** le sync leader actuel se déconnecte
- **When** le serveur doit choisir un remplaçant
- **Then** la stratégie de sélection est:
  1. **Option A (Recommandée):** Premier participant actif par ordre alphabétique de `user_id` (déterministe)
  2. **Option B:** Dernier participant qui a agi (track via `last_action_at`)
  3. **Option C:** Random parmi actifs (simple mais non-déterministe)
- **Decision Required:** Mathias choisit option A, B, ou C
- **And** la stratégie choisie est documentée dans `architecture.md`

**AC 3.2.3: Room Becomes STALE if Last User**

- **Given** User A est sync leader ET le seul participant actif
- **When** User A se déconnecte ou leave
- **Then** room passe LIVE → STALE
- **And** `sync_leader_user_id = NULL`
- **And** polling Spotify est arrêté (pas de participants à synchroniser)

**AC 3.2.4: Automatic Resumption when Room LIVE Again**

- **Given** une room STALE avec `sync_leader_user_id = NULL`
- **When** un nouveau participant rejoint (Story 3.0)
- **Then** ce participant devient sync leader automatiquement
- **And** polling reprend immédiatement

**AC 3.2.5: Silent Badge Update on Failover**

- **Given** le sync leader change due à déconnexion (User A → User B)
- **When** les participants restants reçoivent l'update
- **Then** le badge ⚡ se déplace vers User B dans la participant list
- **And** AUCUNE notification n'est affichée (type: "User B is now controlling playback")

---

### Story 3.3: Polling Optimization - Single Source of Truth

**NEW STORY** - Clarifie relation Epic 3 ↔ Story 1.8 polling

**As a backend system,**  
**I want** poll only the sync leader's Spotify account for playback state,  
**So that** API calls are optimized and state propagation is unambiguous.

#### Acceptance Criteria

**AC 3.3.1: Poll Sync Leader Only**

- **Given** une room LIVE avec sync leader défini (`sync_leader_user_id = userA`)
- **When** le polling timer s'exécute (Story 1.8 - toutes les 5-10s)
- **Then** le serveur poll UNIQUEMENT le Spotify account de User A
- **And** appel API: `GET /me/player` avec access token de User A
- **And** les autres participants ne sont PAS pollés (économie API calls)

**AC 3.3.2: Broadcast to All Participants**

- **Given** le serveur reçoit playback state du sync leader
- **When** le state diffère de l'état précédent (track changed, position drift, etc.)
- **Then** broadcast événement `PLAYER_STATE_UPDATE` à TOUS les participants actifs
- **And** tous les clients se synchronisent sur cet état (Story 1.8 logic)

**AC 3.3.3: Migration from Story 1.8 Current Logic**

- **Story 1.8 Current Behavior:** Poll "tous les participants synced"
- **After Epic 3:** Poll uniquement sync leader
- **Migration Path:**
  - Backend check: si `sync_leader_user_id != NULL` → poll ce user only
  - Si `sync_leader_user_id == NULL` (room STALE) → pas de polling
  - **Breaking Change:** Oui, mais amélioration (moins d'API calls)

**AC 3.3.4: Spotify API Rate Limit Benefits**

- **Given** une room avec 5 participants actifs
- **Before Epic 3:** 5 API calls par polling cycle (5 × `/me/player`)
- **After Epic 3:** 1 API call par polling cycle (sync leader only)
- **Then** réduction de 80% des calls Spotify API
- **And** moins de risque de hit rate limit 429

---

### Story 3.4: Conflict Resolution (Event Ordering)

**UNCHANGED** - Logique reste identique, indépendante du sync leader concept

**As a backend system,**  
**I want** resolve concurrent actions with a simple documented policy,  
**So that** the system remains stable without heavy locking.

#### Acceptance Criteria

**AC 3.4.1: First-Wins Policy**

- **Given** deux actions concurrentes arrivent au serveur quasi simultanément
- **When** elles sont reçues (ex: User A pause, User B play)
- **Then** le serveur applique une policy **first-wins** par ordre de réception
- **And** première action reçue est traitée, seconde est appliquée après (pas rejetée)

**AC 3.4.2: Event Ordering with eventSeq**

- **Given** chaque action génère un événement
- **When** les événements sont diffusés
- **Then** chaque événement a un `eventSeq` monotone croissant par room
- **And** les clients peuvent auditer l'ordre exact des actions

**AC 3.4.3: No Action Blocking**

- **Given** deux users agissent simultanément
- **When** le serveur arbitre avec first-wins
- **Then** AUCUNE action n'est bloquée/rejetée (pas de "you can't do that now")
- **And** les deux actions sont appliquées séquentiellement

**Note:** Story 3.4 est orthogonale au concept de sync leader. Le sync leader détermine la SOURCE de polling, pas l'arbitrage des actions concurrentes.

---

## Schema Changes

**Sessions table addition:**

```sql
-- Add sync_leader_user_id field (nullable)
ALTER TABLE sessions ADD COLUMN sync_leader_user_id TEXT NULL;

-- Index for fast lookup
CREATE INDEX idx_sessions_sync_leader ON sessions(sync_leader_user_id);

-- Constraints
-- NULL allowed (room STALE state)
-- Must reference a valid participant when non-NULL
```

**GORM Model Update:**

```go
type Session struct {
    ID        string
    Name      string
    CreatedBy string  // Room owner (Story 1.10)
    
    // Sync leadership (Epic 3)
    SyncLeaderUserID *string `gorm:"type:text;index"`
    
    Active    bool
    LastActivityAt time.Time
    // ... rest
}
```

---

## UI/UX Specifications

### Participant List Badge

**Design Options (Mathias to choose):**

**Option A: Lightning bolt (⚡)**

```
Participants:
🟢 Sophie ⚡
🟢 Alex
🟢 Mathias
```

**Option B: Musical note (🎵)**

```
Participants:
🟢 Sophie 🎵
🟢 Alex
🟢 Mathias
```

**Option C: Dot indicator (●)**

```
Participants:
🟢 Sophie ● 
🟢 Alex
🟢 Mathias
```

**Option D: No text, just color shift**

```
Participants:
🔵 Sophie    ← sync leader (blue vs green)
🟢 Alex
🟢 Mathias
```

**Recommended:** Option A (⚡) - universally understood as "active/power"

### Badge Behavior

- **Appears:** À côté du nom dans participant list uniquement
- **Updates:** Silently quand sync leader change (pas d'animation flashy)
- **Tooltip:** AUCUN (ou très minimal: "Sync source" si vraiment nécessaire)
- **Accessibility:** Badge has `aria-label="Sync source"` for screen readers

### What Users DON'T See

❌ Toast notification: "You are now the sync leader"  
❌ Modal: "Sophie took over playback control"  
❌ Highlight/border around sync leader name  
❌ Separate "Current DJ" section in UI  
❌ Setting to "transfer sync leader"  

---

## Integration with Existing Stories

### Story 1.8 (Now Playing / Polling)

**Current State:**

- Polling implémenté, mais poll "tous les participants synced"

**After Epic 3:**

- Poll uniquement `sync_leader_user_id`
- Modification requise dans `backend/internal/session/polling.go`
- Test: vérifier polling switch quand sync leader change

### Story 1.10 (Room Model)

**Integration:**

- `created_by` (owner) ≠ `sync_leader_user_id` (polling source)
- Lors de room creation: créateur devient premier sync leader (Story 3.0)
- Lors de leave: failover logic (Story 3.2)

### Epic 2 (Queue Collaborative)

**Integration:**

- Queue actions (add/reorder) ne changent PAS sync leader
- Seuls play/pause/seek/skip changent sync leader
- Queue persist indépendamment du sync leader

---

## Migration & Backward Compatibility

### Existing Rooms (Pre-Epic 3)

**Given** des rooms créées avant Epic 3 (pas de `sync_leader_user_id`)

**Migration Strategy:**

```sql
-- Set sync leader = first active participant (alphabetically)
UPDATE sessions
SET sync_leader_user_id = (
    SELECT user_id 
    FROM session_participants 
    WHERE session_id = sessions.id 
      AND is_active = true
    ORDER BY user_id
    LIMIT 1
)
WHERE active = true;  -- Only LIVE/STALE rooms

-- ARCHIVED rooms: leave NULL (no polling needed)
```

### Story 1.8 Polling Code

**Before:**

```go
// Poll all participants marked "synced"
for _, participant := range participants {
    state := pollSpotify(participant.UserID)
    broadcast(state)
}
```

**After:**

```go
// Poll only sync leader
if session.SyncLeaderUserID != nil {
    state := pollSpotify(*session.SyncLeaderUserID)
    broadcast(state)
} else {
    // Room STALE, skip polling
}
```

---

## Testing Strategy

### Unit Tests

- [ ] Story 3.0: Initial assignment on first join
- [ ] Story 3.1: Switch on play/pause/seek action
- [ ] Story 3.1: Queue actions don't trigger switch
- [ ] Story 3.2: Failover when sync leader disconnects
- [ ] Story 3.2: Room STALE when last user leaves
- [ ] Story 3.3: Poll only sync leader (mock Spotify API)

### Integration Tests

- [ ] Full flow: create room → join → action → switch → leave → failover
- [ ] Multi-user concurrent actions (verify first-wins)
- [ ] Sync leader persistence across reconnects
- [ ] Badge visibility in UI (E2E test)

### E2E Tests (Playwright)

- [ ] User A creates room → sees badge next to own name
- [ ] User B joins → badge still on User A
- [ ] User B clicks play → badge moves to User B (no notification)
- [ ] User B leaves → badge moves back to User A (failover)

---

## Open Questions for Mathias

### Question 1: Failover Selection Strategy (Story 3.2)

Quand sync leader disconnect, qui devient le nouveau?

- **Option A:** Alphabétique par `user_id` (déterministe, testable)
- **Option B:** Dernier qui a agi (logique, mais tracking needed)
- **Option C:** Random (simple, mais non-déterministe)

**Your choice:** ___________

### Question 2: Badge Design (UI)

Quel badge préfères-tu?

- **Option A:** ⚡ (lightning - active/power)
- **Option B:** 🎵 (music note - DJ vibes)
- **Option C:** ● (dot - minimal)
- **Option D:** Color shift (blue user = sync leader)

**Your choice:** ___________

### Question 3: Queue Actions Behavior (Story 3.1)

Adding a track to queue devrait-il changer sync leader?

- **Option A:** NON - seuls play/pause/seek changent sync leader
- **Option B:** OUI - toute action musicale change sync leader

**Recommendation:** Option A (queue ≠ playback control)

**Your choice:** ___________

### Question 4: Tooltip on Badge?

Badge ⚡ devrait-il avoir un tooltip explicatif?

- **Option A:** Pas de tooltip (mystère/découverte)
- **Option B:** Tooltip minimal: "Sync source"
- **Option C:** Tooltip détaillé: "This user's playback is being synced"

**Recommendation:** Option A ou B (pas trop verbeux)

**Your choice:** ___________

---

## Definition of Done - Epic 3

- [ ] Toutes les 4 stories validées (3.0, 3.1, 3.2, 3.3, 3.4)
- [ ] Schema migration créée et testée
- [ ] Backend: sync leader assignment/switch/failover implémenté
- [ ] Backend: polling refactorisé (uniquement sync leader)
- [ ] Frontend: badge UI implémenté dans participant list
- [ ] Frontend: badge update silencieusement (pas de notifications)
- [ ] Tests: unit + integration + E2E passent
- [ ] Migration testée sur DB existante (Story 1.1-1.10)
- [ ] Documentation: architecture.md updated avec sync leader model
- [ ] Story 1.8 polling code refactorisé et validé
- [ ] No regression sur stories 1.1-1.10

---

## Estimated Effort

- **Story 3.0:** 3-4h (initial assignment + badge UI)
- **Story 3.1:** 4-5h (action-based switch + tests)
- **Story 3.2:** 3-4h (failover logic + STALE handling)
- **Story 3.3:** 2-3h (polling refactor Story 1.8)
- **Story 3.4:** 1h (documentation, déjà conceptualisé)

**Total:** ~13-17h (2-3 jours de dev)

---

## Next Steps

1. **Mathias valide cette proposition** (répond aux 4 questions)
2. **J'applique les changements** dans `epics.md`
3. **On implémente Epic 3** après Story 1.10

**Mathias, es-tu OK avec cette refonte complète?** 🎯
