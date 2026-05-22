# ============================================================
# Multi-stage Containerfile for MistHelper-Go
# Stage 1: Build the Go binary with full SDK
# Stage 2: Minimal runtime image (~25MB)
# ============================================================

# ── Build Stage ──────────────────────────────────────────────
FROM docker.io/library/golang:1.25-alpine AS builder

WORKDIR /build

# Copy dependency manifests first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (cmd/ entrypoint + internal/ packages)
COPY cmd/ cmd/
COPY internal/ internal/

# Build static binary (CGO disabled for scratch compatibility)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/misthelper-go \
    ./cmd/misthelper/

# ── Runtime Stage ────────────────────────────────────────────
FROM docker.io/library/alpine:3.19

# Install minimal runtime dependencies (no sshd — Go binary is the SSH server)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && mkdir -p /app/data /app/sessions

# Create non-root user for security (no chpasswd — Go SSH uses its own auth)
RUN addgroup -S misthelper && \
    adduser -S -G misthelper -h /app -s /bin/sh misthelper

# Copy binary from builder
COPY --from=builder /build/misthelper-go /app/misthelper-go

# Set ownership
RUN chown -R misthelper:misthelper /app

WORKDIR /app

# Switch to non-root user (Go SSH server listens on unprivileged port 2200)
USER misthelper

# Expose SSH (2200) and Web UI (8055)
EXPOSE 2200 8055

# Volume for persistent data
VOLUME ["/app/data"]

# Run the Go binary directly — it handles SSH (port 2200) and web (port 8055) internally
CMD ["/app/misthelper-go"]
