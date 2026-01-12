# GateKey WireGuard Mesh Gateway (Spoke) Dockerfile
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

# Build the wireguard mesh gateway binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o gatekey-wireguard-mesh-gateway \
    ./cmd/gatekey-wireguard-mesh-gateway

# Runtime stage
FROM alpine:3.23

# Install runtime dependencies including WireGuard tools
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wireguard-tools \
    iproute2 \
    bash

# Create non-root user (spoke needs root for interface management)
RUN addgroup -g 1000 gatex && \
    adduser -u 1000 -G gatex -s /bin/sh -D gatex

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gatekey-wireguard-mesh-gateway /app/gatekey-wireguard-mesh-gateway

# Create directories for WireGuard
RUN mkdir -p /etc/wireguard /var/log/wireguard /etc/gatekey-wireguard-mesh-gateway

# Spoke connects outbound to hub, no port exposure needed
# but expose for debugging/monitoring
EXPOSE 51820/udp

# Spoke needs to run as root for WireGuard interface management
ENTRYPOINT ["/app/gatekey-wireguard-mesh-gateway"]
CMD ["run"]
