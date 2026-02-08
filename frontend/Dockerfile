FROM node:22-alpine AS build
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@latest --activate
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/dist ./dist
# Keep the container alive so the volume remains available
CMD ["tail", "-f", "/dev/null"]
