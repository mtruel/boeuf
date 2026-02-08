---
stepsCompleted: ['step-01-init', 'step-02-discovery', 'step-03-success', 'step-04-journeys', 'step-05-domain', 'step-06-innovation', 'step-07-project-type', 'step-08-scoping', 'step-09-functional', 'step-10-nonfunctional', 'step-11-polish']
inputDocuments: ['_bmad-output/planning-artifacts/brainstorming-session-2026-01-14.md']
workflowType: 'prd'
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 1
  projectDocs: 0
classification:
  projectType: 'web_app'
  domain: 'entertainment_social_music'
  complexity: 'medium'
  projectContext: 'greenfield'
  userMotivation: 'scratch_own_itch'
  keyInsights:
    - 'Utilisateur actif de Spotify Jam Session - connaissance terrain'
    - 'Pain point: bugs et instabilité de la solution existante'
    - 'Dimension éthique: problème rémunération artistes Spotify'
    - 'Opportunité: feature inexistante sur autres services (Deezer, Qobuz, Tidal)'
---

# Product Requirements Document - boeuf

**Author:** Mathias
**Date:** 2026-01-22

## Executive Summary

**boeuf** est un service de synchronisation musicale qui permet à plusieurs utilisateurs d'écouter la même musique ensemble, en temps réel, chacun depuis son propre appareil.

**Problème résolu :** Les services d'écoute collaborative (comme Spotify Jam) sont instables et enfermés dans un seul écosystème.

**Proposition de valeur :**

- 🎵 **Partage & Découverte** — Écouter ensemble crée un lien social
- 🔓 **Libération** — Pas enfermé dans Spotify, vision multi-plateforme
- ⚡ **Stabilité & Résilience** — Récupération automatique après déconnexion

**Principe technique :** boeuf ne diffuse PAS de musique. Il orchestre les services de streaming existants (Spotify, Deezer, Qobuz...) pour synchroniser la lecture entre participants.

**MVP :** Web app (Vue.js + Go) avec Spotify uniquement. Sync play/pause/queue, host tournant, résilience basique.

**Cible initiale :** Mathias et ses amis l'utilisent au quotidien.

## Success Criteria

### User Success

Le moment "ça marche" pour les utilisateurs :

- Synchronisation de la file d'attente qui fonctionne du premier coup
- Pause/Play synchronisé sans décalage perceptible
- Utilisation fluide, pas besoin de "réessayer" ou "rafraîchir"

### Business Success

**Objectif 3 mois :** Mathias et ses amis utilisent boeuf au quotidien (pas juste "une fois pour tester")

**Métrique clé :** Nombre de sessions/semaine par utilisateur actif

### Technical Success

- **Fiabilité** : Pas de crash en cours de session
- **Résilience** : Récupération automatique si un utilisateur perd la connexion
- **Synchronisation** : ≤ 2-3 secondes de décalage maximum

### Measurable Outcomes

| Critère | Cible MVP | Cible Vision |
|---------|-----------|--------------|
| Décalage de synchronisation | ≤ 3 sec | ≤ 1 sec |
| Récupération après déconnexion | < 10 sec | < 5 sec |
| Crash en session | < 5% des sessions | < 0.1% |
| Plateformes supportées | 1 (Spotify) | 3+ (Deezer, Qobuz, Tidal...) |
| Usage régulier (Mathias + amis) | 1x/semaine | Quotidien |

### Différenciation vs Spotify Jam

Ce qui fait boeuf > Spotify Jam :

- 🏆 **Multi-plateforme** — Pas enfermé dans Spotify
- 🏆 **Stabilité** — Ça ne plante pas au milieu d'une session
- 🏆 **Résilience** — Récupération automatique après perte de connexion

## Product Scope

### MVP - Minimum Viable Product

- Créer/rejoindre une session (lien/code)
- Synchronisation Play/Pause/File d'attente
- Host tournant (qui agit = host)
- Récupération basique après déconnexion
- **1 plateforme : Spotify**

### Growth Features (Post-MVP)

- Multi-plateforme (Deezer, Qobuz)
- Réactions sur les morceaux (musique = message)
- Système de rôles (Admin/DJ pour grands groupes)
- Export playlist vers son service de streaming
- Comptes utilisateurs avec historique personnel

### Vision (Future)

- Tous les services de streaming supportés
- Chaque utilisateur sur SA plateforme préférée (mix cross-plateforme)
- Gestion des titres non disponibles sur certaines plateformes
- Désynchronisation/resynchronisation volontaire

## User Journeys

### Journey 1 : Marc, l'initiateur — "Partager sa journée musicale"

**Qui est Marc ?**
Marc, 29 ans, développeur en télétravail à Lyon. Son pote Jules habite à Bordeaux. Ils se sont rencontrés en école d'ingé et partagent les mêmes goûts musicaux depuis des années. Ils ne se voient que 2-3 fois par an, mais ils bossent tous les deux avec de la musique dans les oreilles toute la journée.

**Sa frustration**
Ils s'envoient des liens Spotify par WhatsApp, mais c'est pas pareil. Quand Marc découvre un album, il aimerait que Jules l'écoute *en même temps*, pas 3 jours plus tard. La musique partagée en simultané, ça crée un lien.

**Son parcours avec boeuf**

1. **Matin, 9h** — Marc ouvre boeuf depuis son navigateur
2. Il connecte son compte Spotify (déjà fait une fois, c'est mémorisé)
3. Il crée une nouvelle session → reçoit un lien
4. Il envoie le lien à Jules sur WhatsApp : *"Allez, on bosse ensemble aujourd'hui 🎧"*
5. Jules clique, rejoint en 10 secondes
6. Marc lance un album qu'il vient de découvrir
7. → **Ça joue chez Jules au même moment** 🎉
8. Jules ajoute un morceau à la file → Marc le voit apparaître
9. Ils passent la journée à bosser "ensemble", à 500km de distance

**Moment de succès**
À la pause café, Jules lui envoie : *"Putain il est bien cet album, je connaissais pas"*. Marc sourit. Mission accomplie.

---

### Journey 2 : Jules, l'invité — "Rejoindre sans friction"

**Qui est Jules ?**
Jules, 28 ans, data analyst à Bordeaux. Moins geek que Marc, il utilise Spotify mais ne creuse pas trop. Il fait confiance à Marc pour lui faire découvrir des trucs.

**Son parcours avec boeuf**

1. Jules reçoit un lien WhatsApp de Marc
2. Il clique → boeuf s'ouvre dans son navigateur
3. Il connecte son Spotify (première fois = autorisation OAuth)
4. Il rejoint la session de Marc
5. → **La musique de Marc démarre chez lui, synchronisée**
6. Il voit la file d'attente, il voit ce qui joue
7. Il ajoute un morceau qu'il aime bien → Marc le voit
8. Plus tard, Jules met pause pour un appel → ça pause chez Marc aussi
9. Il reprend → ça reprend chez les deux

**Moment de succès**
Jules n'a pas eu à "comprendre" comment ça marche. Il a cliqué, ça a marché. Zéro friction.

---

### Journey 3 : Edge Case — Perte de connexion (Marc déconnecté)

**Situation**
Marc est host. Son WiFi plante.

**Ce qui se passe**

1. Le morceau en cours **continue de jouer chez Jules** (Spotify local)
2. boeuf détecte que Marc est déconnecté
3. Jules devient automatiquement le nouveau host (transparent)
4. Marc revient en ligne après 30 secondes
5. boeuf resynchronise Marc sur la position actuelle de Jules
6. → **Continuité totale**, personne n'a remarqué

**Ce que ça révèle comme requirement**

- Détection de déconnexion rapide
- Failover automatique du host
- Resynchronisation transparente au retour

---

### Journey 4 : Edge Case — Conflit (deux actions simultanées)

**Situation**
Marc et Jules cliquent sur "morceau suivant" au même moment.

**Ce qui se passe**

1. Les deux requêtes arrivent quasi-simultanément au serveur
2. boeuf choisit une des deux (ex: premier arrivé)
3. L'action s'exécute normalement
4. L'historique montre les deux actions rapprochées (optionnel)
5. Pas de notification intrusive — c'est pas grave

**Ce que ça révèle comme requirement**

- Gestion "last write wins" ou "first wins" simple
- Pas de lock complexe nécessaire
- Historique des actions pour transparence

---

### Journey Requirements Summary

| Journey | Capabilities requises |
|---------|----------------------|
| Marc (initiateur) | Création session, lien partageable, connexion Spotify, sync play/queue |
| Jules (invité) | Rejoindre via lien, OAuth Spotify, voir file d'attente, contribuer |
| Déconnexion | Détection, failover host, resync au retour |
| Conflit | Résolution simple, historique actions |

## Domain Considerations

### Positionnement juridique

**Principe clé :** boeuf ne diffuse PAS de musique. boeuf orchestre les services de streaming existants.

- Les utilisateurs utilisent leurs propres comptes Spotify/Deezer/etc.
- La musique est streamée par les services natifs, pas par boeuf
- boeuf contrôle uniquement quelle musique est jouée, pas la diffusion elle-même
- **Implication :** Pas de conflit anticipé avec les Terms of Service (à vérifier par plateforme)

### Considérations techniques

- **OAuth 2.0** : Gestion sécurisée des tokens d'accès aux APIs de streaming
- **Rate limiting** : Respect des quotas API de chaque plateforme
- **RGPD** : Applicable (utilisateurs français/européens) — consentement, droit à l'oubli

### Risques identifiés

| Risque | Impact | Mitigation |
|--------|--------|------------|
| Changement d'API Spotify | Élevé | Architecture modulaire, abstraction des APIs |
| Blocage par une plateforme | Moyen | Multi-plateforme dès que possible |
| Tokens expirés/révoqués | Faible | Refresh automatique, gestion gracieuse des erreurs |

## Innovation & Novel Patterns

### Detected Innovation Areas

**Type d'innovation :** Innovation de modèle, pas de technologie pure

| Aspect | Description | Niveau d'innovation |
|--------|-------------|--------------------|
| **Libération de feature** | Écoute synchronisée sortie de l'écosystème Spotify | ✅ Innovant |
| **Vision multi-plateforme** | Utilisateurs Deezer + Qobuz + Spotify ensemble | ✅ Très innovant (n'existe pas) |
| **Dimension éthique** | Alternative pour quitter Spotify sans perdre la feature | ✅ Différenciant |
| **Host tournant** | Architecture de synchronisation dynamique | 🟡 Approche technique intéressante |

### Market Context & Competitive Landscape

**Concurrent direct :** Spotify Jam Session

- Avantage Spotify : Intégré, zéro friction pour utilisateurs Spotify
- Limite Spotify : Enfermé dans l'écosystème, instable, pas d'éthique

**Positionnement boeuf :**

- Service indépendant qui libère la feature
- Cible : utilisateurs qui veulent plus de liberté / éthique
- Opportunité : Deezer, Qobuz, Tidal n'ont PAS cette feature

### Validation Approach

1. **Phase 1 — MVP Spotify** : Prouver que l'orchestration externe fonctionne
2. **Phase 2 — 2ème plateforme** : Prouver le multi-plateforme
3. **Phase 3 — Mix cross-plateforme** : Prouver la valeur unique (groupe mixte)

### Risk Mitigation

| Risque innovation | Impact | Action |
|-------------------|--------|--------|
| API Spotify insuffisante | Bloquant | **Valider les capacités API avant de coder** |
| Adoption faible (Spotify Jam suffisant) | Moyen | Cibler early adopters éthiques / multi-plateforme |
| Complexité multi-plateforme sous-estimée | Élevé | Architecture modulaire dès le MVP |

## Web App Specific Requirements

### Project-Type Overview

boeuf est une **Single Page Application (SPA)** construite avec Vue.js, communiquant en temps réel via WebSockets avec un backend Go. L'application orchestre la synchronisation musicale entre utilisateurs sans diffuser de contenu audio elle-même.

### Technical Architecture Considerations

**Architecture Frontend**

- **Type** : SPA (Single Page Application)
- **Framework** : Vue.js
- **Communication** : WebSockets pour le temps réel
- **État** : Gestion locale de l'état de session + sync serveur

**Architecture Backend**

- **Langage** : Go
- **Base de données** : SQLite (migration Postgres possible)
- **Temps réel** : WebSockets (gorilla/websocket ou nhooyr/websocket)
- **API externe** : Spotify Web API (OAuth 2.0)

### Browser Support

| Navigateur | Version minimum | Priorité |
|------------|-----------------|----------|
| Chrome | Latest - 2 | 🔴 Haute |
| Firefox | Latest - 2 | 🔴 Haute |
| Safari | Latest - 2 | 🔴 Haute |
| Edge | Latest - 2 | 🟡 Moyenne |
| Mobile browsers | Responsive | 🟡 Moyenne |

### Responsive Design

| Device | Support | Notes |
|--------|---------|-------|
| Desktop | ✅ Primary | Expérience principale |
| Tablet | ✅ Responsive | Layout adaptatif |
| Mobile | ✅ Responsive | Fonctionnel, pas optimisé natif |

### Performance Targets

| Métrique | Cible MVP | Cible Vision |
|----------|-----------|-------------|
| First Contentful Paint | < 2s | < 1s |
| Time to Interactive | < 3s | < 2s |
| WebSocket latency | < 500ms | < 200ms |
| Sync delay | ≤ 3s | ≤ 1s |

### SEO Strategy

**MVP :** Aucune — Distribution par bouche-à-oreille et liens directs

**Future :** Landing page statique si besoin de référencement

### Accessibility Level

**MVP :** Basique

- Contraste suffisant
- Navigation clavier fonctionnelle
- Labels sur les boutons principaux

**Post-MVP :** WCAG AA si adoption large

### Implementation Considerations

- **Pas de SSR nécessaire** (pas de SEO)
- **PWA possible** plus tard (offline, install)
- **Pas d'app native** pour le MVP
- **Déploiement** : Static hosting (Vercel, Netlify) + Backend (Fly.io, Railway)

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach :** Problem-solving MVP — résoudre UN problème : écouter ensemble à distance

**Philosophie :** Le strict minimum pour que Marc et Jules disent "ça marche, on peut écouter ensemble"

**Ressources estimées :** 1 développeur fullstack (Mathias), quelques semaines de dev

### MVP Feature Set (Phase 1)

**Core User Journeys Supported :**

- ✅ Marc crée une session et invite Jules
- ✅ Jules rejoint via lien
- ✅ Synchronisation fonctionne dans les deux sens

**Must-Have Capabilities :**

| Feature | Justification |
|---------|---------------|
| Créer une session | Point d'entrée obligatoire |
| Générer un lien/code de partage | Invitation d'autres utilisateurs |
| Rejoindre une session | L'autre côté du flow |
| Connexion Spotify (OAuth) | Accès à l'API de contrôle |
| Sync Play/Pause | Cœur de la valeur |
| Sync file d'attente | Contribution partagée |
| Host tournant | Qui agit = host |
| Récupération après déconnexion | Résilience de base |

**Plateforme MVP :** Spotify uniquement

### Post-MVP Features

**Phase 1.5 (Quick wins après MVP) :**

- Historique simple des morceaux joués
- Affichage de qui a ajouté quoi

**Phase 2 (Growth) :**

- Multi-plateforme : Deezer
- Multi-plateforme : Qobuz
- Gestion des titres non disponibles sur certaines plateformes
- Réactions sur les morceaux (musique = message)
- Export playlist vers son service de streaming
- Comptes utilisateurs avec historique personnel

**Phase 3 (Vision) :**

- Tous les services de streaming supportés
- Mix cross-plateforme (Spotify + Deezer users ensemble)
- Système de rôles (Admin/DJ pour grands groupes)
- Désynchronisation/resynchronisation volontaire

### Risk Mitigation Strategy

**Risque technique #1 — API Spotify insuffisante**

- Impact : 🔴 Bloquant
- Mitigation : **Spike technique AVANT de coder** — valider que l'API permet de contrôler playback et queue

**Risque technique #2 — Latence WebSocket**

- Impact : 🟡 Moyen
- Mitigation : Tests early, fallback polling si nécessaire

**Risque business — Changement API Spotify**

- Impact : 🟡 Moyen
- Mitigation : Architecture modulaire avec abstraction des APIs dès le MVP

**Risque marché — Adoption faible**

- Impact : 🟡 Moyen
- Mitigation : Cibler early adopters éthiques et multi-plateforme, pas les users satisfaits de Spotify Jam

## Functional Requirements

### Session Management

- **FR1:** Un utilisateur peut créer une nouvelle session d'écoute
- **FR2:** Un utilisateur peut générer un lien de partage pour sa session
- **FR3:** Un utilisateur peut rejoindre une session existante via un lien ou code
- **FR4:** Un utilisateur peut quitter une session
- **FR5:** Un utilisateur peut voir les participants actuels de la session

### Streaming Platform Integration

- **FR6:** Un utilisateur peut connecter son compte Spotify via OAuth
- **FR7:** Un utilisateur peut déconnecter son compte Spotify
- **FR8:** Le système peut lire l'état de lecture actuel d'un utilisateur (morceau, position, état play/pause)
- **FR9:** Le système peut contrôler la lecture d'un utilisateur (play, pause, skip)
- **FR10:** Le système peut lire la file d'attente d'un utilisateur
- **FR11:** Le système peut modifier la file d'attente d'un utilisateur

### Playback Synchronization

- **FR12:** Quand un participant lance un morceau, il se lance chez tous les participants
- **FR13:** Quand un participant met pause, la lecture se met en pause chez tous les participants
- **FR14:** Quand un participant reprend la lecture, elle reprend chez tous les participants
- **FR15:** Quand un participant ajoute un morceau à la file d'attente, il apparaît chez tous les participants
- **FR16:** Quand un participant réorganise la file d'attente, elle se réorganise chez tous les participants
- **FR17:** La synchronisation respecte une tolérance de ≤3 secondes de décalage

### Host Management (Dynamic)

- **FR18:** Le participant qui effectue une action devient automatiquement le host
- **FR19:** Le système sélectionne automatiquement un nouveau host si le host actuel se déconnecte
- **FR20:** Le changement de host est transparent pour les participants

### Connection Resilience

- **FR21:** Le système détecte quand un participant perd sa connexion
- **FR22:** Un participant déconnecté peut se reconnecter à sa session
- **FR23:** Un participant reconnecté est automatiquement resynchronisé avec l'état actuel de la session
- **FR24:** La musique continue de jouer chez les autres participants pendant une déconnexion

### Conflict Resolution

- **FR25:** Quand deux participants agissent simultanément, le système résout le conflit (first wins ou last wins)
- **FR26:** Le système ne bloque pas les actions en cas de conflit

### User Interface

- **FR27:** Un utilisateur peut voir le morceau actuellement en cours de lecture
- **FR28:** Un utilisateur peut voir la file d'attente partagée
- **FR29:** Un utilisateur peut voir l'état de connexion des participants

## Non-Functional Requirements

### Performance

| Métrique | Cible MVP | Mesure |
|----------|-----------|--------|
| Latence sync (action → propagation) | ≤ 3 secondes | Temps entre action host et réception chez participants |
| Latence WebSocket | < 500ms | Round-trip time |
| First Contentful Paint | < 2 secondes | Chargement initial de l'app |
| Time to Interactive | < 3 secondes | App utilisable |

- **NFR1:** Une action de synchronisation (play/pause/skip) doit se propager à tous les participants en ≤3 secondes
- **NFR2:** L'interface doit être interactive en moins de 3 secondes après chargement
- **NFR3:** Les mises à jour de la file d'attente doivent apparaître chez tous les participants en ≤2 secondes

### Security

- **NFR4:** Les tokens OAuth Spotify sont stockés de manière sécurisée (pas en clair côté client)
- **NFR5:** Les communications entre client et serveur sont chiffrées (HTTPS/WSS)
- **NFR6:** Un utilisateur ne peut accéder qu'aux sessions où il est participant
- **NFR7:** Les sessions inactives expirent après 24h

### Reliability

- **NFR8:** Le système détecte une déconnexion utilisateur en ≤5 secondes
- **NFR9:** Un utilisateur déconnecté peut se reconnecter et resynchroniser en ≤10 secondes
- **NFR10:** La perte de connexion d'un participant n'interrompt pas la session pour les autres
- **NFR11:** Le système gère gracieusement les erreurs API Spotify (retry, feedback utilisateur)

### Integration

- **NFR12:** Le système respecte les rate limits de l'API Spotify
- **NFR13:** Les tokens OAuth sont rafraîchis automatiquement avant expiration
- **NFR14:** L'architecture permet d'ajouter d'autres services de streaming (abstraction)

### Accessibility (Basique MVP)

- **NFR15:** Les éléments interactifs sont accessibles au clavier
- **NFR16:** Les contrastes de couleurs respectent un ratio minimum de 4.5:1
