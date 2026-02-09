# Boeuf - Guide de Déploiement Simplifié

Ce guide décrit le déploiement simplifié de Boeuf avec une seule image Docker.

## Architecture

Le déploiement utilise une architecture unifiée :
- **Image unique** contenant frontend (Vue.js), backend (Go), et reverse proxy (Caddy)
- **Un seul service** dans docker-compose
- **Volume unique** pour les données SQLite
- **Configuration minimale** avec seulement les variables essentielles

## Prérequis

- Docker et Docker Compose installés
- Credentials Spotify API (client ID et secret)

## Déploiement rapide

### 1. Configuration

Copiez le fichier d'exemple et éditez-le :

```bash
cp .env.example .env
nano .env  # ou votre éditeur préféré
```

Configurez au minimum ces variables obligatoires :

```env
SPOTIFY_CLIENT_ID=votre_client_id_spotify
SPOTIFY_CLIENT_SECRET=votre_client_secret_spotify
APP_SECRET=0123456789abcdef0123456789abcdef  # Générez-en un nouveau !
EXPOSE_PORT=3000
```

**Important** : Générez un nouveau `APP_SECRET` :
```bash
openssl rand -hex 16
```

### 2. Lancement

```bash
docker compose up -d
```

L'application sera accessible sur `http://localhost:3000` (ou le port défini dans `EXPOSE_PORT`).

### 3. Vérification

Vérifiez que le service est démarré :

```bash
docker compose ps
docker compose logs -f boeuf
```

Testez l'application :
```bash
curl http://localhost:3000/api/health
# Devrait retourner: {"status":"ok"}
```

### 4. Arrêt

```bash
docker compose down
```

Pour supprimer également les données :
```bash
docker compose down -v
```

## Configuration avancée

### Variables d'environnement optionnelles

Si vous souhaitez personnaliser davantage, ajoutez ces variables dans `.env` :

```env
# Durée de validité des sessions (défaut: 24 heures)
SESSION_DURATION_HOURS=48

# Nombre max de sessions actives par utilisateur (défaut: 10, 0 = illimité)
MAX_ACTIVE_SESSIONS_PER_USER=5

# URL publique pour OAuth (requis en production)
PUBLIC_URL=https://boeuf.example.com

# URI de redirection OAuth Spotify (doit correspondre à la config Spotify)
SPOTIFY_REDIRECT_URI=https://boeuf.example.com/auth/spotify/callback
```

### Déploiement en production

Pour un déploiement en production :

1. **Configurez un nom de domaine** et pointez-le vers votre serveur

2. **Mettez à jour les variables d'environnement** :
   ```env
   PUBLIC_URL=https://votre-domaine.com
   SPOTIFY_REDIRECT_URI=https://votre-domaine.com/auth/spotify/callback
   EXPOSE_PORT=80
   ```

3. **Ajoutez HTTPS** (recommandé) :
   - Modifiez le `Caddyfile` pour activer HTTPS automatique
   - Ou utilisez un reverse proxy externe (nginx, Traefik)

4. **Enregistrez l'URL de redirection** dans la configuration de votre application Spotify :
   - Allez sur https://developer.spotify.com/dashboard
   - Sélectionnez votre application
   - Ajoutez `https://votre-domaine.com/auth/spotify/callback` dans "Redirect URIs"

5. **Démarrez le service** :
   ```bash
   docker compose up -d
   ```

### Mise à jour

Pour mettre à jour l'application :

```bash
docker compose down
docker compose pull  # Si vous utilisez des images pré-buildées
docker compose up -d --build
```

### Sauvegarde

Les données sont stockées dans un volume Docker. Pour sauvegarder :

```bash
# Créer une sauvegarde
docker run --rm -v boeuf_boeuf_data:/data -v $(pwd):/backup alpine tar czf /backup/boeuf-backup.tar.gz -C /data .

# Restaurer une sauvegarde
docker run --rm -v boeuf_boeuf_data:/data -v $(pwd):/backup alpine tar xzf /backup/boeuf-backup.tar.gz -C /data
```

## Résolution de problèmes

### Le service ne démarre pas

Vérifiez les logs :
```bash
docker compose logs -f boeuf
```

### Erreur de connexion Spotify

Vérifiez que :
- Vos credentials Spotify sont corrects
- L'URL de redirection est bien enregistrée dans la console Spotify
- Les variables `PUBLIC_URL` et `SPOTIFY_REDIRECT_URI` correspondent

### Base de données corrompue

Arrêtez le service et supprimez le volume :
```bash
docker compose down -v
docker compose up -d
```

### Port déjà utilisé

Changez le `EXPOSE_PORT` dans `.env` :
```env
EXPOSE_PORT=8080
```

## Support

Pour plus d'informations, consultez :
- README.md - Documentation générale
- Architecture: `_bmad-output/planning-artifacts/architecture.md`
- Spécifications: `_bmad-output/planning-artifacts/prd.md`
