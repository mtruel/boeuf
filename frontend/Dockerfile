FROM node:22-slim AS dev
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile

FROM dev AS build
COPY . .
RUN pnpm build

FROM alpine:latest AS prod
WORKDIR /app
COPY --from=build /app/dist ./dist
# Keep the container alive so the volume remains available
CMD ["tail", "-f", "/dev/null"]
