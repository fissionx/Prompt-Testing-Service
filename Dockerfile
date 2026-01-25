# Production Dockerfile for Gego - GEO Tracker
# Optimized for production deployment with minimal image size

FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata sqlite-dev gcc musl-dev

# Set working directory
WORKDIR /app

# Set GOTOOLCHAIN to auto to allow downloading newer Go versions
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o gego ./cmd/gego/main.go

# Stage 2: Minimal runtime
FROM alpine:latest

# Install ca-certificates for HTTPS requests and SQLite runtime library (required for CGO-enabled binary)
RUN apk --no-cache add ca-certificates tzdata sqlite

# Copy binary
COPY --from=builder /app/gego /usr/local/bin/gego

# Copy migration files
COPY --from=builder /app/internal/db/migrations /migrations

# Create directories
RUN mkdir -p /app/data /app/config /app/logs

# Set environment variables
ENV GEGO_CONFIG_PATH=/app/config/config.yaml
ENV GEGO_DATA_PATH=/app/data
ENV GEGO_LOG_PATH=/app/logs

# Create default configuration using cat heredoc for proper YAML formatting
# Note: POSTGRESQL_URI MUST be set via fly.io secrets for the app to start
# The URI will be overridden by POSTGRESQL_URI environment variable if set
# Setting uri to empty string - it MUST be provided via POSTGRESQL_URI secret
RUN cat > /app/config/config.yaml <<EOF
sql_database:
  provider: sqlite
  uri: /app/data/gego.db
  database: gego

nosql_database:
  provider: postgresql
  uri: ""
  database: gego
EOF

# Expose port
EXPOSE 8989

# Health check removed - Fly.io handles this via fly.toml http_service.checks

# Default command
CMD ["/usr/local/bin/gego", "--host", "0.0.0.0", "--port", "8989"]
