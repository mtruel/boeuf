FROM node:22-slim AS dev
WORKDIR /app

# Install pnpm globally with cache
RUN --mount=type=cache,target=/root/.npm \
    npm install -g pnpm

# Copy dependency files first for better caching
COPY package.json pnpm-lock.yaml ./

# Install dependencies with cache mount
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

FROM dev AS build
# Copy source code
COPY . .

# Build with cache
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm build

FROM alpine:latest AS prod
WORKDIR /app
COPY --from=build /app/dist ./dist
# Keep the container alive so the volume remains available
CMD ["tail", "-f", "/dev/null"]
