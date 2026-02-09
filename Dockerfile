# Unified Dockerfile for Boeuf - Production-ready image with backend, frontend, and Caddy

# Stage 1: Build Backend
FROM golang:1.23-alpine AS backend-build

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build/backend

# Copy backend dependency files first for better caching
COPY backend/go.mod backend/go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy backend source code
COPY backend/ ./

# Build with CGO enabled and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -o /backend ./cmd/boeuf-server

# Stage 2: Build Frontend
FROM node:22-slim AS frontend-build

WORKDIR /build/frontend

# Install pnpm globally with cache
RUN --mount=type=cache,target=/root/.npm \
    npm install -g pnpm

# Copy frontend dependency files first for better caching
COPY frontend/package.json frontend/pnpm-lock.yaml ./

# Install dependencies with cache mount
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# Copy frontend source code
COPY frontend/ ./

# Build frontend with cache
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm build

# Stage 3: Production - Unified image with Caddy, Backend, and Frontend
FROM caddy:2.10.2-alpine

# Install runtime dependencies for backend (SQLite)
RUN apk --no-cache add ca-certificates sqlite-libs

# Create directories
RUN mkdir -p /app/backend /app/data /usr/share/caddy

# Copy backend binary
COPY --from=backend-build /backend /app/backend/backend

# Copy frontend build
COPY --from=frontend-build /build/frontend/dist /usr/share/caddy

# Copy Caddyfile
COPY Caddyfile /etc/caddy/Caddyfile

# Copy startup script
COPY <<'EOF' /start.sh
#!/bin/sh
set -e

# Start backend in background
cd /app/backend
./backend &

# Start Caddy in foreground
caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
EOF

RUN chmod +x /start.sh

# Expose port 80 for Caddy
EXPOSE 80

# Use the startup script
CMD ["/start.sh"]
