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

```bash
cp .env.example .env
# Éditer .env avec vos credentials Spotify
docker compose up
```

## Stack technique

- **Frontend**: Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS v4 + shadcn-vue
- **Backend**: Go + SQLite + GORM + WebSocket
- **Infrastructure**: Caddy (reverse proxy + HTTPS) + Docker Compose

## Documentation

- Architecture: `_bmad-output/planning-artifacts/architecture.md`
- Spécifications: `_bmad-output/planning-artifacts/prd.md`
- Règles projet: `_bmad-output/project-context.md`
