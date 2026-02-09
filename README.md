# Boeuf - Spotify Sync Listening

Une application de synchronisation d'écoute Spotify pour partager une session audio en temps réel.

## Structure du projet

```
boeuf/
├── frontend/           # Application Vue 3 (SPA)
├── backend/            # API Go + WebSocket
├── deploy/             # Documentation déploiement
├── spike-spotify-api/  # Spike technique Spotify API
├── docker-compose.yml  # Configuration Docker Compose
├── Caddyfile           # Configuration reverse proxy
└── .env.example        # Variables d'environnement
```

## Développement local

### Prérequis

- Node.js + pnpm (frontend)
- Go 1.21+ avec CGO activé (backend SQLite)
- Docker + Docker Compose (déploiement)

### Démarrage rapide

#### Frontend

```bash
cd frontend
pnpm install
pnpm dev
```

#### Backend

```bash
cd backend
go run ./cmd/boeuf-server
```

**Tester l'API:**

```bash
# Health check (backend direct)
curl http://localhost:8080/health
# Devrait retourner: {"status":"ok"}

# Health check via proxy Vite (intégration FE↔BE)
curl http://localhost:5173/api/health
# Devrait retourner: {"status":"ok"}
```

**Tests automatisés:**

```bash
# Tests unitaires frontend
cd frontend && pnpm test:unit

# Tests backend
cd backend && go test ./...

# Tests e2e (intégration FE↔BE complète)
cd frontend && pnpm test:e2e
```

#### Déploiement complet

Le déploiement utilise une image Docker unique qui contient le frontend, le backend et le reverse proxy Caddy.

```bash
cp .env.example .env
# Éditer .env avec vos credentials Spotify
docker compose up -d
```

L'application sera accessible sur `http://localhost:3000` (ou le port défini dans `EXPOSE_PORT`).

## GitHub Actions & CI/CD

### Tests automatiques sur Pull Requests

Tous les PRs vers `main` ou `dev` déclenchent automatiquement :
- **Tests backend** : `go test ./...`
- **Tests frontend** : `pnpm test:unit`

Ces tests doivent réussir avant toute fusion.

### Build automatique des images Docker sur Pull Requests

**Nouveau !** À chaque commit sur un PR, les images Docker sont automatiquement buildées et publiées :

- **Workflow** : `PR Build`
- **Images publiées** :
  - `ghcr.io/<owner>/boeuf-backend:pr-<number>`
  - `ghcr.io/<owner>/boeuf-frontend:pr-<number>`
  - `ghcr.io/<owner>/boeuf-backend:pr-<number>-<commit-sha>`
  - `ghcr.io/<owner>/boeuf-frontend:pr-<number>-<commit-sha>`

Un commentaire est automatiquement ajouté sur le PR avec les tags des images après chaque build réussi. Vous pouvez ainsi tester n'importe quelle version d'un PR avant de le merger.

### Build manuel (optionnel)

Si besoin, vous pouvez aussi déclencher **manuellement** un build :

1. Allez dans l'onglet **Actions** du repository GitHub
2. Sélectionnez le workflow **"Manual Build Check"**
3. Cliquez sur **"Run workflow"**
4. Entrez le numéro du PR à builder
5. Le workflow va builder et publier les images Docker avec le tag `pr-<number>`

### Déploiement automatisé (GitHub Actions + Watchtower)

**Pour plus de détails sur le déploiement, consultez [DEPLOYMENT.md](./DEPLOYMENT.md)**

## Configuration des variables d'environnement

### Variables requises

Ces variables doivent être définies dans le fichier `.env` :

| Variable | Description | Exemple |
|----------|-------------|---------|
| `SPOTIFY_CLIENT_ID` | Client ID de votre application Spotify | `abc123def456` |
| `SPOTIFY_CLIENT_SECRET` | Client Secret de votre application Spotify | `xyz789uvw012` |
| `APP_SECRET` | Clé secrète pour le chiffrement (32 caractères hexadécimaux) | `0123456789abcdef0123456789abcdef` |
| `EXPOSE_PORT` | Port exposé sur l'hôte pour accéder à l'application | `3000` |

**Générer un APP_SECRET :**
```bash
openssl rand -hex 16
```

### Variables optionnelles

Ces variables ont des valeurs par défaut et peuvent être omises :

| Variable | Description | Défaut | Exemple |
|----------|-------------|--------|---------|
| `SESSION_DURATION_HOURS` | Durée de validité des sessions et invitations (en heures) | `24` | `48` |
| `MAX_ACTIVE_SESSIONS_PER_USER` | Nombre maximum de sessions actives par utilisateur (0 = illimité) | `10` | `5` |
| `PUBLIC_URL` | URL publique pour les callbacks OAuth | `http://localhost:3000` | `https://boeuf.example.com` |
| `SPOTIFY_REDIRECT_URI` | URI de redirection OAuth Spotify | `http://localhost:3000/auth/spotify/callback` | `https://boeuf.example.com/auth/spotify/callback` |

### Notes importantes

- **APP_SECRET** : Cette clé est utilisée pour chiffrer les tokens Spotify et sécuriser les sessions. Elle doit faire exactement 32 caractères hexadécimaux (16 octets). Pour simplifier le déploiement, la même clé est utilisée pour le chiffrement et les sessions. Pour une sécurité maximale en production critique, vous pourriez vouloir utiliser des clés séparées.
- **PUBLIC_URL et SPOTIFY_REDIRECT_URI** : En production, ces URLs doivent correspondre à votre domaine public et être enregistrées dans la configuration de votre application Spotify.
- Les données de la base de données SQLite sont stockées dans un volume Docker persistant.

## Déploiement automatisé (GitHub Actions)

À chaque push sur `main` ou `dev`, l'image Docker unifiée est publiée sur GHCR :

- `ghcr.io/<owner>/boeuf:main` (branche main)
- `ghcr.io/<owner>/boeuf:dev` (branche dev)
- `ghcr.io/<owner>/boeuf:latest` (dernière version de main)

Pour utiliser l'image pré-construite, commentez la ligne `build: .` dans `docker-compose.yml` et décommentez :
```yaml
image: ghcr.io/mtruel/boeuf:dev
```

## Stack technique

- **Frontend**: Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS v4 + shadcn-vue
- **Backend**: Go + SQLite + GORM + WebSocket
- **Infrastructure**: Caddy (reverse proxy + HTTPS) + Docker Compose

## Documentation

- Architecture: `_bmad-output/planning-artifacts/architecture.md`
- Spécifications: `_bmad-output/planning-artifacts/prd.md`
- Règles projet: `_bmad-output/project-context.md`
