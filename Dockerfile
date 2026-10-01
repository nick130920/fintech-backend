# Build stage
FROM golang:1.25.14-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -ldflags '-s -w' -o app ./cmd/server

# Final stage
FROM alpine:3.22.6@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8

# curl is the explicit runtime dependency for the image health check.
RUN apk --no-cache add ca-certificates curl tzdata \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

# Copy runtime assets needed by the default file://migrations configuration.
COPY --from=builder --chown=app:app /app/app ./app
COPY --from=builder --chown=app:app /app/migrations ./migrations

# Dokploy probes the internal HTTP health endpoint after the application starts.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD curl -fsS http://127.0.0.1:${PORT:-8080}/health || exit 1

USER app

# Expose the default port; PORT remains configurable at runtime.
EXPOSE 8080

# Exec-form CMD preserves direct SIGTERM delivery to the application.
CMD ["./app"]
