FROM golang:1.23-alpine AS build

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

# Build with CGO enabled for SQLite support
RUN CGO_ENABLED=1 GOOS=linux go build -o /backend ./cmd/boeuf-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /root/
COPY --from=build /backend .
EXPOSE 8080
CMD ["./backend"]
