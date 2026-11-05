# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies and root certificates
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=1.0.0" \
    -o /app/server \
    ./cmd/api/main.go

# Production runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies (ca-certificates for TLS, curl for container health check)
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

# Copy compiled binary and migrations from builder
COPY --from=builder /app/server /app/server
COPY --from=builder /build/migrations /app/migrations

# Set correct ownership
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
