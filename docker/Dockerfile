# Build stage
FROM golang:1.24 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy only necessary source directories
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build arguments for version info (should be provided by CI/CD)
ARG GIT_TAG=dev
ARG GIT_COMMIT=unknown

# Build the application with optimizations and version info
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X 'subvault/internal/version.Version=${GIT_TAG}' -X 'subvault/internal/version.GitCommit=${GIT_COMMIT}'" \
    -o subvault ./cmd/server

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies and gosu for PUID/PGID support
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    sqlite3 \
    tzdata \
    gosu \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/subvault .

# Copy templates and static assets
COPY web/ ./web/

# Create data directory
RUN mkdir -p /app/data

# Copy entrypoint script
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release
ENV DATABASE_PATH=/app/data/subvault.db
ENV PUID=99
ENV PGID=100

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Container starts as root, entrypoint drops privileges via gosu
ENTRYPOINT ["/app/entrypoint.sh"]
