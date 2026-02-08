# Deployment Guide

> **Note**: Les fichiers de déploiement (`docker-compose.yml`, `Caddyfile`, `.env.example`) sont maintenant à la racine du projet pour plus de simplicité.

## Local Development

### Prerequisites

- Docker & Docker Compose
- pnpm (for frontend development)
- Go 1.23+ (for backend development)

### Start the stack

```bash
# À exécuter depuis la racine du projet
docker compose up -d
```

### Verify services

```bash
# Backend health check
curl http://localhost:8080/health

# Frontend 
curl http://localhost:3000/
```

### Stop the stack

```bash
docker compose down
```

### View logs

```bash
docker compose logs -f
```

### Rebuild after changes

```bash
docker compose build
docker compose up -d
```

## Service URLs

- **Frontend**: `http://localhost:3000` (or custom `EXPOSE_PORT`)
- **Backend API**: `http://localhost:3000/api/health`
- **Backend (direct)**: `http://localhost:8080/health` (internal only in Docker network)

## Production Deployment with HTTPS

### Prerequisites

- A domain name pointing to your server (e.g., `boeuf.example.com`)
- Ports 80 and 443 open on your server

### Configure Caddy for HTTPS

Edit `Caddyfile` to use your domain instead of `:80`:

```caddyfile
boeuf.example.com {
    root * /usr/share/caddy
    file_server

    # Reverse proxy for API
    handle_path /api/* {
        reverse_proxy backend:8080
    }

    # Handle SPA routing for Vue
    handle {
        try_files {path} /index.html
    }

    log {
        output stdout
    }
}
```

Caddy will automatically obtain and renew Let's Encrypt SSL certificates.

### Update docker-compose for production

```yaml
# In docker-compose.yml, update caddy ports:
ports:
  - "80:80"
  - "443:443"
```

Then start with:

```bash
docker compose up -d
```

Caddy will handle HTTPS automatically. WebSocket connections will use WSS (secure WebSocket) automatically over HTTPS.

- **Frontend**: <http://localhost:3000>
- **Backend**: <http://localhost:8080>
- **Backend Health**: <http://localhost:8080/health>

## Architecture

- **Frontend**: Vue 3 + Vite + TypeScript → Nginx
- **Backend**: Go → Standalone binary
- **Network**: Internal `boeuf-network` for inter-service communication
