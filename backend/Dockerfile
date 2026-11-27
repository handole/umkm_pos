# ==============================================================================
# Multi-Stage Dockerfile for Go (Gin) Modular Monolith Backend
# Stage 1: Build static binary with compiler optimizations
# Stage 2: Lightweight runtime image (Alpine Linux ~15-20 MB)
# ==============================================================================

# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Compile static binary with symbol stripping (-w -s)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/api-server \
    ./cmd/api

# Stage 2: Minimal Runtime
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies: timezone and SSL certificates
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

# Set Indonesian Western Time (WIB) default
ENV TZ=Asia/Jakarta

# Copy compiled binary, swagger docs, and certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /build/api-server /app/api-server
COPY --from=builder /build/docs /app/docs

# Use non-root user
USER appuser

EXPOSE 8080

# Health check probe
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api-server"]
