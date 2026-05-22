# ============================================================
# Multi-stage Containerfile for MistHelper-Go
# Stage 1: Build the Go binary with full SDK
# Stage 2: Minimal runtime image (~25MB)
# ============================================================

# ── Build Stage ──────────────────────────────────────────────
FROM docker.io/library/golang:1.21-alpine AS builder

WORKDIR /build

# Copy dependency manifests first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (internal/ added as features are ported)
COPY cmd/ cmd/

# Build static binary (CGO disabled for scratch compatibility)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/misthelper-go \
    ./cmd/misthelper/

# ── Runtime Stage ────────────────────────────────────────────
FROM docker.io/library/alpine:3.19

# Install minimal runtime dependencies (SSH server, certificates)
RUN apk add --no-cache \
    openssh-server \
    ca-certificates \
    tzdata \
    && mkdir -p /app/data /app/sessions /etc/ssh

# Create non-root user for security
RUN addgroup -S misthelper && \
    adduser -S -G misthelper -h /app -s /bin/sh misthelper && \
    echo "misthelper:misthelper123!" | chpasswd

# Generate SSH host keys
RUN ssh-keygen -A

# Configure SSH server
RUN echo "Port 2200" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config && \
    echo "AllowUsers misthelper" >> /etc/ssh/sshd_config && \
    echo "ForceCommand /app/misthelper-go" >> /etc/ssh/sshd_config && \
    echo "PrintMotd no" >> /etc/ssh/sshd_config

# Copy binary from builder
COPY --from=builder /build/misthelper-go /app/misthelper-go

# Set ownership
RUN chown -R misthelper:misthelper /app

WORKDIR /app

# Expose SSH (2200) and Web UI (8055)
EXPOSE 2200 8055

# Volume for persistent data
VOLUME ["/app/data"]

# Start SSH daemon in foreground
CMD ["/usr/sbin/sshd", "-D", "-e"]
