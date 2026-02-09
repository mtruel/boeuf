# Summary of Deployment Simplification Changes

## Overview
This document summarizes all changes made to simplify the Boeuf deployment process from a multi-container setup to a single Docker image approach.

## What Changed

### 1. New Files Created

#### `Dockerfile` (Root)
- **Purpose**: Multi-stage build that creates a unified image
- **Stages**:
  - Stage 1: Build frontend (Vue.js) with Node.js and pnpm
  - Stage 2: Build backend (Go) with CGO for SQLite support
  - Stage 3: Final image based on Caddy Alpine, containing:
    - Frontend static files in `/usr/share/caddy`
    - Backend binary in `/app/backend`
    - Caddy reverse proxy configuration
- **Startup**: Uses `/start.sh` script to run backend in background and Caddy in foreground

#### `.dockerignore`
- **Purpose**: Optimize Docker build by excluding unnecessary files
- **Excludes**: Git files, documentation, dependencies (installed during build), build artifacts, test files

#### `DEPLOYMENT.md`
- **Purpose**: Comprehensive deployment guide for end users
- **Contents**: Quick start, configuration details, production deployment guide, backup/restore procedures, troubleshooting

### 2. Modified Files

#### `docker-compose.yml`
**Before**: 3 services (caddy, frontend, backend) with shared volumes and network
**After**: 1 service (boeuf) with:
- Single build context (root directory)
- Single volume for data (`boeuf_data`)
- All environment variables configured
- Maps `APP_SECRET` to both `SESSION_KEY` and `ENCRYPTION_KEY`
- Proper defaults for all optional variables

#### `.env.example`
**Before**: 11 required/optional variables
**After**: 4 required variables:
- `SPOTIFY_CLIENT_ID` (required)
- `SPOTIFY_CLIENT_SECRET` (required)
- `APP_SECRET` (required)
- `EXPOSE_PORT` (required)

**Commented optional variables**:
- `SESSION_DURATION_HOURS` (default: 24)
- `MAX_ACTIVE_SESSIONS_PER_USER` (default: 10)
- `PUBLIC_URL` (default: http://localhost:3000)
- `SPOTIFY_REDIRECT_URI` (default: http://localhost:3000/auth/spotify/callback)

**Removed variables** (now internal):
- `PORT` (fixed to 8080 internally)
- `NODE_ENV` (fixed to production)
- `SESSION_KEY` (derived from APP_SECRET)
- `ENCRYPTION_KEY` (derived from APP_SECRET)
- `ENV` (fixed to production)
- `DATABASE_PATH` (fixed to /app/data/boeuf.db)

#### `README.md`
**Added**:
- Complete environment variables documentation table
- Required vs optional variables distinction
- Instructions for generating `APP_SECRET`
- Security notes about key usage
- Reference to `DEPLOYMENT.md` for detailed deployment guide

### 3. Unchanged Files (Backend Code)

No changes were made to backend code. The backend already uses environment variables correctly:
- `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` for Spotify API
- `SESSION_KEY` and `ENCRYPTION_KEY` for security
- `PORT`, `NODE_ENV`, `SESSION_DURATION_HOURS`, etc. with proper defaults

## Architecture Comparison

### Before (Multi-Container)
```
┌─────────────────────────────────────────┐
│ Docker Compose                           │
├─────────────────────────────────────────┤
│ ┌─────────┐  ┌──────────┐  ┌─────────┐ │
│ │ Caddy   │  │ Frontend │  │ Backend │ │
│ │ (proxy) │  │ (build)  │  │ (API)   │ │
│ └─────────┘  └──────────┘  └─────────┘ │
│      │            │              │       │
│      └────────────┴──────────────┘       │
│         (volumes + network)               │
└─────────────────────────────────────────┘
```

### After (Single Container)
```
┌───────────────────────────────┐
│ Docker Compose                 │
├───────────────────────────────┤
│ ┌───────────────────────────┐ │
│ │ Boeuf (unified image)     │ │
│ │  ┌──────────────────────┐ │ │
│ │  │ Caddy (port 80)      │ │ │
│ │  │  - Serves frontend   │ │ │
│ │  │  - Proxies /api/     │ │ │
│ │  │  - Proxies /auth/    │ │ │
│ │  └──────────────────────┘ │ │
│ │  ┌──────────────────────┐ │ │
│ │  │ Backend (port 8080)  │ │ │
│ │  │  - Go API + WebSocket│ │ │
│ │  │  - SQLite DB         │ │ │
│ │  └──────────────────────┘ │ │
│ └───────────────────────────┘ │
└───────────────────────────────┘
```

## Benefits

1. **Simplified Deployment**: Single `docker compose up -d` command
2. **Reduced Configuration**: Only 4 required environment variables
3. **Hidden Complexity**: Users don't need to understand Caddy, reverse proxy, or multi-container orchestration
4. **Single Image**: Easier to distribute, version, and deploy
5. **Consistent Defaults**: Sensible defaults for all optional settings
6. **Better Documentation**: Clear separation between required and optional configuration

## User Experience

### Old Workflow
1. Copy `.env.example` to `.env`
2. Fill in 11+ environment variables
3. Understand multiple services (caddy, frontend, backend)
4. Run `docker compose up`
5. Hope everything connects properly

### New Workflow
1. Copy `.env.example` to `.env`
2. Fill in 4 essential variables (Spotify credentials, app secret, port)
3. Run `docker compose up -d`
4. Done!

## Migration Path

For existing users:
1. The old `docker-compose.prod.yml` and `docker-compose.dev.yml` still exist for reference
2. Backend code unchanged, so no breaking changes
3. Environment variables are backward compatible (just subset of what was before)
4. Data persists in volume (no data migration needed)

## Security Considerations

1. **APP_SECRET Reuse**: For simplicity, APP_SECRET is used for both SESSION_KEY and ENCRYPTION_KEY. This is acceptable for most use cases but documented for users who need higher security.
2. **Process Management**: Backend runs without a supervisor. Container restart policy handles crashes.
3. **Default Secrets**: `.env.example` includes placeholder secrets with clear instructions to generate new ones.

## Testing Recommendations

The unified build should be tested:
1. Build succeeds: `docker build -t boeuf .`
2. Container starts: `docker compose up -d`
3. Health check: `curl http://localhost:3000/api/health`
4. Frontend loads: Open `http://localhost:3000` in browser
5. Spotify OAuth works: Test login flow
6. WebSocket works: Create and join a session
7. Data persists: Stop/start container, verify sessions remain

## Known Limitations

1. **Backend Crash Handling**: If backend process crashes (not the container), Caddy continues running but API is unavailable until container restart
2. **Build Time**: Unified build takes longer than pre-built images, but only needs to be done once
3. **Development Workflow**: The simplified setup is optimized for production deployment, development workflow may need adjustments

## Next Steps (Optional)

Future improvements could include:
1. Process supervisor (s6-overlay, supervisord) for better backend process management
2. Health checks in docker-compose for automatic restart on backend failure
3. Separate SESSION_KEY and ENCRYPTION_KEY derivation from APP_SECRET
4. CI/CD pipeline to publish pre-built unified images to GHCR
