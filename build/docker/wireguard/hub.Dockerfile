# GateKey WireGuard Mesh Hub Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Allow auto-download of newer toolchain if needed
ENV GOTOOLCHAIN=auto

WORKDIR /build

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the wireguard hub binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o gatekey-wireguard-hub \
    ./cmd/gatekey-wireguard-hub

# Runtime stage
FROM alpine:3.23

# Install runtime dependencies including WireGuard tools and nftables
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wireguard-tools \
    nftables \
    iptables \
    iproute2 \
    bash

# Create non-root user (hub needs root for firewall and interface management)
RUN addgroup -g 1000 gatex && \
    adduser -u 1000 -G gatex -s /bin/sh -D gatex

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gatekey-wireguard-hub /app/gatekey-wireguard-hub

# Create directories for WireGuard
RUN mkdir -p /etc/wireguard /var/log/wireguard /etc/gatekey-wireguard-hub

# WireGuard default port (UDP only)
EXPOSE 51820/udp

# Hub needs to run as root for firewall and WireGuard interface management
ENTRYPOINT ["/app/gatekey-wireguard-hub"]
CMD ["run"]
