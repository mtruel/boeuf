# Spike : Validation API Spotify pour boeuf

## Objectif

Valider que l'API Spotify fournit toutes les capacités nécessaires pour implémenter le MVP de boeuf, notamment :

- ✅ OAuth 2.0 authentication
- ✅ Player control (play/pause/next/seek)
- ✅ Queue management
- ✅ Synchronisation timing

## Setup

### Prérequis

- Go 1.21+
- Compte Spotify Premium (requis pour le Player API)
- Créer une app sur [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)

### Configuration

1. Créer une app Spotify :
   - Aller sur <https://developer.spotify.com/dashboard>
   - Cliquer "Create app"
   - Nom : "boeuf-spike"
   - Redirect URI : `http://localhost:8080/callback`

2. Copier les credentials :

   ```bash
   cp .env.example .env
   # Éditer .env avec CLIENT_ID et CLIENT_SECRET
   ```

3. Installer les dépendances :

   ```bash
   go mod download
   ```

## Tests à exécuter

### Test 1 : OAuth Flow

```bash
go run main.go
# Ouvrir http://localhost:8080/login
# Se connecter avec Spotify
# Vérifier que le token est obtenu
```

### Test 2 : Player Control

```bash
go run test_player_control.go
# Vérifier que le player démarre/pause depuis l'API
```

### Test 3 : Queue Management

```bash
go run test_queue.go
# Ajouter un morceau à la queue
# Vérifier qu'il apparaît dans GET /me/player/queue
```

### Test 4 : Synchronisation Timing

```bash
go run test_sync_timing.go
# Mesurer le décalage entre 2 users qui seekent simultanément
```

## Résultats Attendus

- ✅ OAuth flow complet fonctionnel
- ✅ Contrôle effectif du player Spotify externe
- ✅ Ajout/lecture de la queue fonctionnel
- ⚠️ Décalage de synchronisation ≤ 3 secondes

## Notes

Voir [spotify-api-spike.md](../_bmad-output/implementation-artifacts/spotify-api-spike.md) pour l'analyse complète.
