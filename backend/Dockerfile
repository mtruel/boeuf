FROM golang:1.23-alpine AS build

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build with CGO enabled and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -o /backend ./cmd/boeuf-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
RUN mkdir -p /app/data
WORKDIR /app
COPY --from=build /backend .
EXPOSE 8080
CMD ["./backend"]
