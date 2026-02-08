---
stepsCompleted: [1, 2]
inputDocuments: []
session_topic: 'Service de Jam Session collaborative pour streaming musical synchronisé (boeuf)'
session_goals: 'Exploration large : fonctionnalités, architecture technique, défis multi-plateformes, scénarios d usage, différenciation, et stratégie MVP'
selected_approach: 'ai-recommended'
techniques_used: ['SCAMPER Method', 'First Principles Thinking', 'Resource Constraints']
ideas_generated: []
context_file: '_bmad/bmm/data/project-context-template.md'
---

# Session de Brainstorming - boeuf

**Date :** 2026-01-14
**Participant :** Mathias
**Facilitateur :** Mary (Business Analyst)

## Session Overview

**Topic :** Service de Jam Session collaborative pour streaming musical synchronisé (boeuf)

**Goals :** Exploration large couvrant :

- 🎵 Fonctionnalités & expérience utilisateur
- 🔧 Solutions techniques & architecture système
- 🌐 Défis multi-plateformes (Spotify, Deezer, Qobuz)
- 👥 Scénarios d'usage & personas utilisateurs
- 💡 Différenciation & proposition de valeur unique
- 🎯 Stratégie MVP & priorisation des features

### Context Guidance

Cette session s'appuie sur les domaines d'exploration clés pour le développement produit logiciel :

- Problèmes utilisateurs et points de friction
- Idées de fonctionnalités et capacités
- Approches techniques d'implémentation
- Expérience utilisateur et interactions
- Modèle d'affaires et création de valeur
- Différenciation marché
- Risques techniques et défis
- Métriques de succès

### Session Setup

**Vision Initiale :**
Service externe permettant à plusieurs utilisateurs d'écouter la même musique ensemble en temps réel, chacun depuis son propre ordinateur. File d'attente partagée où chaque utilisateur peut ajouter des morceaux. Phase 1 : support d'une seule plateforme de streaming. Vision future : support multi-plateformes avec gestion des titres non disponibles sur certaines plateformes.

---

## Technique Selection

**Approche :** Techniques Recommandées par l'IA
**Contexte d'Analyse :** Service de Jam Session collaborative avec focus sur exploration large multi-dimensionnelle

**Techniques Recommandées :**

1. **SCAMPER Method** (Phase 1 - Exploration Large)
   - Créativité systématique à travers 7 lentilles : Substitute, Combine, Adapt, Modify, Put to other uses, Eliminate, Reverse
   - Génération de 50+ variations de fonctionnalités et approches

2. **First Principles Thinking** (Phase 2 - Défis Techniques)
   - Déconstruction des suppositions pour reconstruire depuis les vérités fondamentales
   - Architecture claire basée sur principes, pas sur suppositions

3. **Resource Constraints** (Phase 3 - Stratégie MVP)
   - Contraintes extrêmes pour forcer l'identification du noyau essentiel
   - Priorisation claire : essentiel vs nice-to-have

**Rationale IA :** Séquence conçue pour innovation systématique → architecture robuste → priorisation claire, adaptée à un projet de service complexe avec défis techniques multi-plateformes.

---

## Phase 1 : SCAMPER Method - Exploration Large

### S - SUBSTITUTE (Substituer)

**Substituer la musique par autre chose :**

- ❌ Podcasts : Partage moins intéressant, les gens n'ont pas forcément envie d'écouter simultanément
- ❌ Livres audio : Temps d'écoute trop longs, usage différent
- ❌ Sons d'ambiance : Pas d'intérêt identifié
- ❌ Vidéos : Existe déjà (Netflix, etc.) - hors scope de boeuf

**Substituer Spotify/Deezer par d'autres sources :**

- ✅ Fichiers locaux : Intéressant !
- ✅ YouTube : Oui, potentiel
- ✅ Tous les services de streaming musicaux : Très intéressant
- 💡 **INSIGHT CLÉ :** Idéalement, chaque utilisateur pourrait utiliser SA propre plateforme de streaming
  - ⚠️ Défi identifié : Gestion des titres non disponibles sur toutes les plateformes
- ❌ Radio en direct : Moins d'intérêt - les utilisateurs peuvent juste écouter la même radio et discuter ailleurs
- ❌ Vinyles numérisés : Équivalent fichiers locaux, pas le but

**Substituer la synchronisation temps-réel :**

- ❌ Playlist partagée asynchrone : Existe déjà, moins intéressant car pas de simultanéité = moins social
- ❌ Time capsule musicale : Pas d'intérêt identifié
- 💡 **INSIGHT CLÉ :** La simultanéité EST le cœur de la valeur sociale

**Substituer l'ordinateur par d'autres devices :**

- ✅ Smartphone + Ordinateur : Bonne base pour contrôler
- 💡 **INSIGHT ARCHITECTURE :** boeuf ne joue PAS la musique !
  - L'app native (Spotify, Deezer...) se charge de jouer le son
  - boeuf se charge UNIQUEMENT de la synchronisation des files d'attente
  - Séparation claire des responsabilités

### C - COMBINE (Combiner)

**Combiner écoute + fonctionnalités sociales :**

- 💡 **CONCEPT CLÉ : Musique = Message**
  - Historique de toutes les musiques jouées dans la session
  - Chaque musique est comme un message dans un chat
  - Les utilisateurs peuvent répondre/réagir à une musique spécifique
  - Like, commentaires, discussions autour d'un morceau
- ✅ Chat texte : Oui, intégré à la conversation musicale
- ✅ Réactions emoji : Oui, sur les morceaux
- ❌ Chat vocal : Non, pas intéressant
- ❌ Vidéo : Non, trop lourd

**Combiner boeuf + autres apps/services :**

- 🔮 Bot Discord : Intéressant pour plus tard, pas essentiel au début
- ❌ Twitch, jeux vidéo, apps de rencontre : Pas une bonne idée

**Combiner file d'attente + mécanismes :**

- 💡 **CONCEPT CLÉ : Modes de contrôle selon le contexte**
  - **Mode de base (2-3 personnes)** : Tout le monde peut éditer et réorganiser librement
    - Les utilisateurs se connaissent et se font confiance
    - Pas besoin de restrictions
  - **Mode groupe (beaucoup d'utilisateurs)** : Votes, points, restrictions
    - À explorer plus tard, pas prioritaire
- 💡 **DJ Tournant** : Intéressant quand beaucoup d'utilisateurs
  - Un "chef" qui décide
  - Limite le chaos
  - Rotation du contrôle

**Combiner plusieurs sessions :**

- 🔮 Sessions publiques/privées : Possible mais lourd, pas nécessaire tout de suite
- 💡 **FOCUS MVP : Quelque chose de simple d'abord**

**Cas d'usage principaux identifiés :**

- ✅ **Travail** : 2 utilisateurs dans des bureaux différents, même musique dans le casque
- ✅ **Soirées à distance** : Amis qui font la fête ensemble mais séparés
- ✅ **Soirées au même endroit** : Aussi intéressant (chacun son casque ?)
- 🤔 Sport ensemble : Possible mais incertain

### A - ADAPT (Adapter)

**Vision fondamentale de boeuf :**

- 💡 **CONCEPT CLÉ : boeuf = Partage & Découverte musicale**
  - Faire découvrir des morceaux à ses amis
  - Partager sa musique avec d'autres personnes

**Adapter pour contextes spécifiques :**

- 🤔 Version focus/travail : Pas très différente de découverte, juste moins de partage actif
- 🤔 Version soirée : Pas besoin d'effets visuels, mais système de choix adapté aux personnes présentes - pas le but premier
- ❌ Différents types de relations : Pas nécessaire, pas besoin de lien particulier pour écouter ensemble

**Adapter fonctionnalités d'autres apps :**

- ❌ Limité car les fonctionnalités existantes sont liées à leur service de streaming
- 💡 **INSIGHT :** Obligation de faire quelque chose d'extérieur/indépendant

**Adapter selon le nombre d'utilisateurs :**

| Taille | Mode | Contrôles | Rôles |
|--------|------|-----------|-------|
| **2-3 personnes** | Intime | Tout partagé, tout le monde peut tout faire (même pause) | Égalitaire |
| **~10 personnes** | Petit groupe | Administrateur pour éviter interruptions/pauses intempestives | Admin + Participants |
| **50+ personnes** | Radio/DJ | Mode structuré avec contrôle centralisé | DJ(s) + Modérateurs + Auditeurs |

**💡 CONCEPT CLÉ : Système de Rôles**

| Rôle | Permissions |
|------|-------------|
| **Auditeur** | Écoute + Propose des musiques (suggestions) |
| **Admin/DJ** | Contrôle direct de la file d'attente, peut changer/réorganiser |
| **Modérateur** | Gestion des utilisateurs (à définir) |

- Les auditeurs peuvent **proposer** des musiques
- Les admins/DJs peuvent **décider** et modifier directement

### M - MODIFY (Modifier/Magnifier)

**Modifier l'expérience de synchronisation :**

- 💡 **DÉCISION TECHNIQUE :** Synchronisation parfaite NON nécessaire
  - Tolérance acceptable : **1-2 secondes** de décalage
  - Simplifie l'architecture technique

**Amplifier le côté social :**

- ✅ Voir qui a ajouté quoi : Intéressant
- ✅ Savoir quel utilisateur a passé quelle musique
- ✅ Notifications quand quelqu'un réagit à "ta" musique : Intéressant
- ✅ Statistiques de session : Pas mal aussi (qui a contribué, morceaux populaires)

**Modifier la file d'attente :**

- ✅ File d'attente infinie : Oui, pas de raison de limiter
- 🤔 Priorité aux morceaux votés : Pertinent seulement avec beaucoup d'utilisateurs
- 💡 **CONCEPT : Système de propositions (mode grand groupe)**
  - Utilisateurs proposent des morceaux
  - Propositions visibles par tous
  - Les autres peuvent voter ou proposer une alternative
  - Admin/DJ voit les plus votés et décide
- ❌ Joker/passe-droit : Non, contre l'esprit de partage ensemble

**Réduire la friction :**

- ⚠️ Configuration service de streaming nécessaire : Pas simple, étapes requises
- ✅ Compte sur l'app boeuf : Peut-être pas nécessaire pour écouter
- ✅ Lien de partage simple : Oui, à explorer
- 📋 Compte service de streaming : OBLIGATOIRE pour utiliser boeuf

**Amplifier la découverte :**

- ✅ Suggestions automatiques basées sur les goûts du groupe : Intéressant
- 💡 **PISTE TECHNIQUE :** Technologies open source comme MusicBrainz
  - Défi technique à explorer

### P - PUT TO OTHER USES (Autres usages)

**Usages professionnels :**

- ✅ DJs qui préparent un set ensemble : Intéressant
- ❌ Cours de musique en groupe : Non
- ❌ Ambiance coworking : Non

**Usages éducatifs :**

- 🤔 Professeur qui fait découvrir la musique : Cas d'usage possible, mais pas de feature spécifique
- ❌ Apprentissage langues / Histoire musique : Pas le but

**Usages événementiels :**

- ❌ Silent disco virtuelle : Hors scope, trop éloigné
- ❌ Mariage/anniversaire à distance : Pas d'intérêt
- ❌ Concert watch party : Pas d'intérêt

**Usages communautaires :**

- ❌ Fan clubs : Non
- 🤔 Groupes de découverte par genre : Juste des sessions spécifiques, pas de feature dédiée
- ❌ "Book club" albums : Pas de développement spécifique

**Usages thérapeutiques :**

- ❌ Méditation/relaxation : Pas d'intérêt

💡 **INSIGHT :** La plupart des usages alternatifs fonctionnent avec les features de base - pas besoin de développement spécifique. Le focus reste sur le **partage et la découverte musicale entre amis/connaissances**.

### E - ELIMINATE (Éliminer)

**Éliminer des complexités - Compte utilisateur :**

- ✅ Sessions sans compte possibles : Lien + service de streaming + code de session
- 💡 **DÉCISION :** Compte optionnel mais utile
  - Sans compte : Peut créer/rejoindre une session
  - Avec compte : Historique de toutes les sessions (créées ou rejointes)
  - L'historique de la session elle-même est toujours sauvegardé (avec son code)

**Éliminer des complexités - Interface :**

- 💡 **DÉCISION MVP : Web d'abord**
  - Application web prioritaire (usage au travail, navigateur)
  - Version mobile web peut suffire si bien faite
  - App native : possible plus tard, pas essentiel

**Features attendues mais pas essentielles :**

- ✅ **Export playlist** : Intéressant - rapatrier la playlist de session vers son service de streaming
- 🤔 **Accès aux playlists dans boeuf** : Pas essentiel si on peut ajouter depuis le service de streaming directement
- 💡 **DÉCISION :** Essentiel de pouvoir mettre en file d'attente depuis le service de streaming natif, sans passer par boeuf
- ❌ Profils utilisateurs détaillés : Pas nécessaire
- ❌ Système d'amis : Non
- 🔮 Système de followers : Peut-être intéressant plus tard
  - Suivre un utilisateur actif
  - Notification quand il lance une session
  - Pas MVP

**Frictions techniques - Décisions MVP :**

- ✅ **Une seule plateforme** : Spotify (bonne API) pour le MVP
- ✅ **Pas de support mobile natif** : Web app simple au début
- ✅ **Pas de chat intégré** : WhatsApp/Discord à côté au début

**MVP Essentiel :**

- ✅ Synchronisation des files d'attente
- ✅ Historique texte simple des morceaux joués
- 🔧 Logs développeur : Historique complet des actions (réorganisation, etc.)

**Ce que boeuf N'EST PAS :**

| ❌ N'est PAS | Pourquoi |
|-------------|----------|
| Clone de Spotify | Ne diffuse pas de musique |
| Réseau social musical | Juste partage de session, pas de commentaires/profils sociaux |
| Service de streaming | S'attache à un service existant |

💡 **DÉFINITION CLAIRE :** boeuf est un service qui s'attache à un service de streaming et permet de **contribuer à la file d'attente** et de la **partager entre plusieurs utilisateurs**.

### R - REVERSE / REARRANGE (Inverser / Réorganiser)

**Inverser le flow :**

- 🤔 Créer vs Rejoindre : Dépend du contexte (ami a déjà une session → rejoindre, sinon → créer et inviter)
- 💡 **INSIGHT ARCHITECTURAL MAJEUR :**
  - Ce n'est PAS juste une synchronisation de file d'attente
  - C'est une **synchronisation COMPLÈTE des applications de streaming**
  - Si je clique sur un morceau dans Spotify → il joue chez TOUT LE MONDE
  - Pas besoin de passer par la file d'attente pour lancer un morceau

**🔥 PRINCIPE FONDAMENTAL DE SYNCHRONISATION :**

| Action | Comportement |
|--------|--------------|
| Morceau lancé | → Lance chez tout le monde |
| File d'attente réorganisée | → Réorganisée chez tout le monde |
| Morceau change | → Change chez tout le monde |
| Pause | → Pause chez tout le monde (par défaut, peut-être configurable) |

**Inverser les rôles :**

- ❌ 100% algorithme : Non
- ✅ Tout le monde DJ : Oui, c'est le principe de base
  - Tout le monde peut changer les morceaux tout le temps
  - Sauf grandes sessions → système de rôles (Admin/DJ vs Auditeur)

**Réorganiser l'expérience :**

- ❌ Réorganisation automatique (par mood, tempo) : Non, les utilisateurs décident
- ❌ Commencer par la fin : Pas intéressant

**Inverser la découverte :**

- 🤔 Partager ce qu'on aime vs faire découvrir : C'est aux utilisateurs de décider, pas une feature
- ❌ Blind listening : Feature compliquée pour pas grand chose

**Inverser la synchronisation :**

- 🔮 **Désynchronisation volontaire** : Intéressant pour plus tard
  - Chacun écoute à son rythme
  - Voir où en sont les autres
  - "Rattraper" les autres quand on rejoint en cours
  - Se resynchroniser plus tard
  - ⚠️ Complexe à implémenter → Pas MVP

---

## 📊 Synthèse Phase 1 - SCAMPER

### Insights Architecturaux Clés

1. **boeuf = Orchestrateur, pas lecteur** - Les apps natives jouent le son
2. **Synchronisation COMPLÈTE** - Pas juste la queue, TOUT (play, pause, skip, réorganisation)
3. **Tolérance 1-2 secondes** - Pas besoin de synchro milliseconde
4. **Multi-plateforme par utilisateur** (vision future) - Chacun sa plateforme préférée

### Concepts Produit Clés

1. **Musique = Message** - Historique comme chat, réactions aux morceaux
2. **Système de rôles adaptatif** - Égalitaire (2-3) → Admin (10) → DJ/Modérateur (50+)
3. **Compte optionnel** - Sessions possibles sans compte, compte = historique personnel
4. **Export playlist** - Rapatrier la session vers son service de streaming

### Définition MVP

- ✅ Une plateforme : Spotify
- ✅ Web app d'abord
- ✅ Synchronisation complète (play, pause, queue)
- ✅ Historique simple des morceaux
- ❌ Pas de chat intégré (WhatsApp à côté)
- ❌ Pas d'app mobile native
- ❌ Pas de multi-plateforme

---

## Phase 2 : First Principles Thinking - Défis Techniques

### Vérités Fondamentales Incontournables

**Prérequis utilisateurs :**

- Les utilisateurs doivent utiliser habituellement une plateforme de streaming musical
- Les utilisateurs doivent avoir lancé leur application de streaming
- Les utilisateurs doivent être connectés au service boeuf

**Prérequis techniques :**

- boeuf doit pouvoir LIRE (via API/connexion) la musique actuellement jouée par chaque utilisateur
- boeuf doit pouvoir ÉCRIRE (contrôler) la lecture chez chaque utilisateur
- La latence réseau est variable → nécessite une référence commune (horloge ?)

**Réalité des APIs :**

- Spotify : API riche, beaucoup de fonctionnalités
- Autres services (Qobuz, Deezer...) : APIs plus limitées, moins permissives

### Suppositions Potentiellement Fausses

⚠️ **SUPPOSITION RISQUÉE IDENTIFIÉE :**
> "On pourra connecter toutes les applications de streaming"

**Réalité probable :** Certains services ne seront PAS compatibles

- APIs trop limitées (pas de contrôle de lecture)
- Restrictions d'usage commercial
- Pas d'accès à la position de lecture en temps réel

### Besoins API Identifiés

| Type | Besoin | Pourquoi |
|------|--------|----------|
| **LIRE** | Morceau en cours | Savoir ce que chacun écoute |
| **LIRE** | Position de lecture | Savoir OÙ dans le morceau |
| **LIRE** | File d'attente | Connaître ce qui vient après |
| **ÉCRIRE** | Lancer un morceau | Synchroniser le play |
| **ÉCRIRE** | Mettre en pause | Synchroniser la pause |
| **ÉCRIRE** | Modifier file d'attente | Partager les ajouts |
| **ÉCRIRE** | Seek (position) | Rattraper/synchroniser |

**Statut API Spotify :** À investiguer, probablement OK
**Statut autres services :** À investiguer individuellement
**💡 DÉCISION MVP :** Commencer par Spotify, complexifier plus tard

### Architecture de Synchronisation - Options à Explorer

**Option A : Serveur boeuf central**

- ✅ Source de vérité unique
- ⚠️ Problème : Le serveur n'a pas accès aux catalogues Spotify directement

**Option B : Utilisateur "Host" + Followers**

- ✅ Plus simple conceptuellement
- ✅ Le host a accès à son propre Spotify
- 🤔 À explorer davantage

**💡 QUESTION OUVERTE :** Quelle architecture choisir ? À approfondir.

### Gestion de la Déconnexion

**Quand un utilisateur perd la connexion :**

1. Notifier qu'il est désynchronisé
2. Proposer des options :
   - **Resynchronisation immédiate** avec le host/groupe
   - **Rattraper à son rythme** (garder un retard volontaire)

### 💡 DÉCISION ARCHITECTURE : Modèle "Host Spotify"

**Choix validé : Option A - Contrôle DANS Spotify, boeuf synchronise en arrière-plan**

```
┌─────────────┐     contrôle      ┌─────────────┐
│   HOST      │ ───────────────── │   SPOTIFY   │
│  (Alice)    │                   │   (Alice)   │
└─────────────┘                   └──────┬──────┘
                                         │
                                    détecte état
                                         │
                                         ▼
                                  ┌─────────────┐
                                  │    BOEUF    │
                                  │   SERVER    │
                                  └──────┬──────┘
                                         │
                              réplique l'état
                            ┌────────────┼────────────┐
                            ▼            ▼            ▼
                     ┌──────────┐ ┌──────────┐ ┌──────────┐
                     │ SPOTIFY  │ │ SPOTIFY  │ │ SPOTIFY  │
                     │  (Bob)   │ │ (Charlie)│ │ (Diana)  │
                     └──────────┘ └──────────┘ └──────────┘
```

**Flow :**

1. Host (Alice) contrôle SA lecture Spotify normalement
2. boeuf détecte les changements d'état du host (polling ou webhook)
3. boeuf réplique l'état vers tous les autres participants via leurs APIs Spotify

**Avantages :**

- ✅ UX naturelle : l'utilisateur utilise Spotify comme d'habitude
- ✅ Pas besoin d'interface de contrôle complexe dans boeuf
- ✅ Le host a toujours la "vérité"

**Défis techniques à résoudre :**

- ⚠️ Comment détecter rapidement les changements du host ? (polling fréquent ? webhooks ?)
- ⚠️ Latence entre action du host et réplication chez les autres

### 💡 DÉCISION : Host Tournant avec Failover

**Modèle choisi : Host tournant dynamique**

```
Session boeuf : Alice (host), Bob, Charlie
        │
        ▼
┌───────────────────────────────────────────────────┐
│  Alice fait PLAY → Alice = host → sync vers tous  │
└───────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────────┐
│  Bob change morceau → Bob = host → sync vers tous │
└───────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────────┐
│  Alice perd connexion → Bob devient host auto     │
└───────────────────────────────────────────────────┘
```

**Règles :**

1. **Tout le monde peut agir** (dans les petits groupes)
2. **Celui qui agit devient host** temporairement
3. **Host = référence** pour le timing du morceau en cours
4. **Failover automatique** : Si le host perd la connexion → nouveau host sélectionné automatiquement
5. **Le nouveau host** devient la source de vérité (timing, file d'attente)

**Défis techniques identifiés :**

- ⚠️ **Gestion des conflits** : Que se passe-t-il si Alice et Bob agissent en même temps ?
  - Option simple : "Dernier qui agit gagne" ?
  - À réfléchir et tester
- ⚠️ **Sélection du nouveau host** : Critères ? (premier connecté ? meilleure connexion ?)
- ⚠️ **Transition fluide** : Comment assurer la continuité lors du changement de host ?

---

## Phase 3 : Resource Constraints - Stratégie MVP

### Contrainte 1 : 3 Features Maximum

**Les 3 features ESSENTIELLES du MVP boeuf :**

| # | Feature | Description |
|---|---------|-------------|
| 1 | **Créer/Rejoindre session** | Créer une session avec un code, inviter des amis, rejoindre via lien/code |
| 2 | **Synchronisation complète** | Play/pause + file d'attente synchronisés entre tous les participants |
| 3 | **Historique basique** | Liste simple des morceaux qui ont été joués dans la session |

**Ce qui est HORS MVP :**

- ❌ Réactions/likes sur les morceaux
- ❌ Chat intégré
- ❌ Export de playlist
- ❌ Système de rôles (admin/DJ)
- ❌ Comptes utilisateurs avec historique personnel
- ❌ Multi-plateforme (Deezer, Qobuz...)
- ❌ Suggestions automatiques

### Contrainte 2 : Parcours Utilisateur MVP (Démo)

```
1.  Alice ouvre boeuf (web app)
2.  Alice connecte son compte Spotify
3.  Alice crée une session → reçoit un code/lien
4.  Alice envoie le lien à Bob (WhatsApp, SMS...)
5.  Bob ouvre le lien, connecte son Spotify
6.  Bob rejoint la session
7.  Alice lance un morceau dans Spotify
8.  → Le morceau se lance chez Bob aussi ! 🎉
9.  Bob ajoute un morceau à la file d'attente
10. → La file d'attente se met à jour chez Alice !
11. Bob lance un morceau dans Spotify
12. → Le morceau se lance chez Alice aussi ! 🎉 (host tournant)
13. Les deux voient l'historique des morceaux passés
```

**✅ Ce flow démontre :**

- Création/jonction de session
- Synchronisation bidirectionnelle (Alice → Bob ET Bob → Alice)
- Host tournant (les deux peuvent contrôler)
- File d'attente partagée
- Historique basique

### Contrainte 3 : Stack Technique MVP

| Couche | Choix | Raison |
|--------|-------|--------|
| **Backend** | Go | Léger, performant, économe en ressources serveur |
| **Base de données** | SQLite + ORM | Simple pour MVP, migration Postgres possible plus tard |
| **Frontend** | Vue.js | Expérience existante, framework solide |
| **Temps réel** | WebSockets | Recommandé (voir ci-dessous) |

**Recommandation pour le temps réel :**

| Technologie | Avantages | Inconvénients | Verdict |
|-------------|-----------|---------------|---------|
| **WebSockets** | Bidirectionnel, temps réel vrai, standard | Connexion persistante à gérer | ✅ **Recommandé** |
| Server-Sent Events (SSE) | Simple, unidirectionnel serveur→client | Pas de client→serveur natif | ❌ Insuffisant |
| Polling | Très simple | Latence, gaspillage ressources | ❌ Pas adapté |

**💡 Pourquoi WebSockets pour boeuf :**

- Communication **bidirectionnelle** nécessaire (host ↔ followers)
- **Faible latence** critique pour la synchronisation musicale
- Go a d'excellentes libs WebSocket (gorilla/websocket, nhooyr/websocket)
- Vue.js gère très bien les WebSockets

**Stack finale MVP :**

```
┌─────────────────────────────────────────────────────────┐
│                      FRONTEND                           │
│                      Vue.js                             │
│                        │                                │
│                   WebSockets                            │
│                        │                                │
├─────────────────────────────────────────────────────────┤
│                      BACKEND                            │
│                       Go                                │
│            ┌──────────┴──────────┐                      │
│            │                     │                      │
│       SQLite + ORM          Spotify API                 │
└─────────────────────────────────────────────────────────┘
```

---

# 🎯 SYNTHÈSE FINALE - Session de Brainstorming boeuf

## 1. Vision Produit

### Ce qu'est boeuf
>
> **boeuf est un service de synchronisation musicale qui s'attache à un service de streaming (Spotify) et permet à plusieurs utilisateurs d'écouter la même musique ensemble, en temps réel, chacun depuis son propre appareil.**

### Proposition de valeur

- **Partage & Découverte** : Faire découvrir des morceaux à ses amis
- **Écoute simultanée** : La simultanéité crée le lien social
- **Contrôle partagé** : Tout le monde peut contribuer à la file d'attente

### Ce que boeuf N'EST PAS

| ❌ | Pourquoi |
|----|----------|
| Clone de Spotify | Ne diffuse pas de musique, s'attache à Spotify |
| Réseau social musical | Pas de profils, followers, commentaires sociaux |
| Service de streaming | Orchestre la synchronisation uniquement |

## 2. Architecture Technique

### Modèle de synchronisation : Host Tournant

```
┌─────────────────────────────────────────────────────────────┐
│  Qui agit devient HOST → boeuf synchronise vers les autres  │
└─────────────────────────────────────────────────────────────┘

- Tout le monde peut contrôler (petits groupes)
- Celui qui agit devient host temporairement  
- Host = référence pour le timing
- Failover automatique si host déconnecté
```

### Décisions techniques clés

| Décision | Choix | Raison |
|----------|-------|--------|
| Tolérance de synchro | 1-2 secondes | Simplifie l'architecture |
| Contrôle | Dans Spotify, pas dans boeuf | UX naturelle |
| Détection changements | Polling ou webhooks | À investiguer |
| Communication | WebSockets | Bidirectionnel, faible latence |

### Stack MVP

| Couche | Technologie |
|--------|-------------|
| Backend | Go (léger, performant) |
| Base de données | SQLite + ORM (migration Postgres possible) |
| Frontend | Vue.js |
| Temps réel | WebSockets |
| API externe | Spotify Web API |

## 3. MVP Défini

### 3 Features Essentielles

| # | Feature | Description |
|---|---------|-------------|
| 1 | **Créer/Rejoindre session** | Code/lien de session, invitation simple |
| 2 | **Synchronisation complète** | Play/pause + file d'attente + morceau en cours |
| 3 | **Historique basique** | Liste des morceaux joués dans la session |

### Parcours Utilisateur MVP

```
1.  Alice ouvre boeuf (web app)
2.  Alice connecte son compte Spotify
3.  Alice crée une session → reçoit un code/lien
4.  Alice envoie le lien à Bob (WhatsApp, SMS...)
5.  Bob ouvre le lien, connecte son Spotify
6.  Bob rejoint la session
7.  Alice lance un morceau dans Spotify
8.  → Le morceau se lance chez Bob ! 🎉
9.  Bob ajoute un morceau à la file d'attente
10. → La file d'attente se met à jour chez Alice !
11. Bob lance un morceau dans Spotify
12. → Le morceau se lance chez Alice ! 🎉 (host tournant)
13. Les deux voient l'historique des morceaux passés
```

## 4. Roadmap Future (Post-MVP)

### Phase 2 - Social & UX

- [ ] Musique = Message (réactions, likes, commentaires sur morceaux)
- [ ] Voir qui a ajouté quoi
- [ ] Notifications quand quelqu'un réagit à "ta" musique
- [ ] Statistiques de session
- [ ] Export playlist vers Spotify

### Phase 3 - Modes avancés

- [ ] Système de rôles (Admin/DJ, Auditeur, Modérateur)
- [ ] Système de propositions et votes (grands groupes)
- [ ] DJ tournant automatique
- [ ] Comptes utilisateurs avec historique personnel

### Phase 4 - Multi-plateforme

- [ ] Support Deezer
- [ ] Support Qobuz
- [ ] Support YouTube Music
- [ ] Gestion des titres non disponibles cross-plateforme

### Phase 5 - Extensions

- [ ] Bot Discord
- [ ] Désynchronisation/resynchronisation volontaire
- [ ] App mobile native
- [ ] Suggestions automatiques (MusicBrainz)

## 5. Questions Ouvertes & Défis Techniques

### À investiguer

| Question | Priorité |
|----------|----------|
| Capacités exactes de l'API Spotify (read/write playback, queue) | 🔴 Haute |
| Polling vs Webhooks pour détecter les changements | 🔴 Haute |
| Gestion des conflits (2 users agissent en même temps) | 🟡 Moyenne |
| Sélection automatique du nouveau host | 🟡 Moyenne |
| Latence réelle WebSocket → acceptable ? | 🟡 Moyenne |

### Risques identifiés

- ⚠️ Limitations API Spotify non connues
- ⚠️ Autres services de streaming potentiellement incompatibles
- ⚠️ Gestion des conflits multi-host à concevoir

---

## 📋 Prochaines Étapes Recommandées

1. **Investiguer l'API Spotify** - Confirmer les capacités read/write playback
2. **Prototype technique** - Tester la synchro entre 2 comptes Spotify
3. **PRD (Product Requirements Document)** - Formaliser les exigences
4. **Architecture détaillée** - Concevoir la solution technique complète

---

*Session de brainstorming terminée le 2026-01-22*
*Facilitateur : Mary (Business Analyst)*
*Participant : Mathias*
*Techniques utilisées : SCAMPER, First Principles Thinking, Resource Constraints*
