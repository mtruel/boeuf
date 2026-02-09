# Multi-stage Dockerfile for unified Boeuf deployment
# Builds frontend, backend, and serves both through Caddy

# Stage 1: Build Frontend
FROM node:22-slim AS frontend-build
WORKDIR /app

# Install pnpm globally with cache
RUN --mount=type=cache,target=/root/.npm \
    npm install -g pnpm

# Copy frontend dependency files first for better caching
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# Copy frontend source and build
COPY frontend/ ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm build

# Stage 2: Build Backend
FROM golang:1.23-alpine AS backend-build

# Install build dependencies for CGO (required for SQLite)
RUN apk update && apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy backend dependency files first for better caching
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy backend source code
COPY backend/ ./

# Build with CGO enabled and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -o /backend ./cmd/boeuf-server

# Stage 3: Final image with Caddy
FROM caddy:2.10.2-alpine

# Install runtime dependencies for backend (SQLite)
RUN apk update && apk add --no-cache ca-certificates sqlite-libs

# Create necessary directories
RUN mkdir -p /app/data /usr/share/caddy

# Copy backend binary
COPY --from=backend-build /backend /app/backend

# Copy frontend build
COPY --from=frontend-build /app/dist /usr/share/caddy

# Copy Caddyfile
COPY Caddyfile /etc/caddy/Caddyfile

# Expose HTTP port
EXPOSE 80

# Create a startup script to run both backend and Caddy
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'cd /app && ./backend &' >> /start.sh && \
    echo 'exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile' >> /start.sh && \
    chmod +x /start.sh

# Set working directory
WORKDIR /app

CMD ["/start.sh"]
