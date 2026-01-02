# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set GOTOOLCHAIN to auto for version compatibility
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gatekey-server ./cmd/gatekey-server

# Build gateway binaries for multiple platforms (for download by remote gateways)
RUN mkdir -p /gateway-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /gateway-binaries/gatekey-gateway-linux-amd64 ./cmd/gatekey-gateway && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /gateway-binaries/gatekey-gateway-linux-arm64 ./cmd/gatekey-gateway

# Build client binaries for multiple platforms
RUN mkdir -p /client-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /client-binaries/gatekey-linux-amd64 ./cmd/gatekey && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /client-binaries/gatekey-linux-arm64 ./cmd/gatekey && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o /client-binaries/gatekey-darwin-amd64 ./cmd/gatekey && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o /client-binaries/gatekey-darwin-arm64 ./cmd/gatekey

# Build admin CLI binaries for multiple platforms
RUN mkdir -p /admin-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /admin-binaries/gatekey-admin-linux-amd64 ./cmd/gatekey-admin && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /admin-binaries/gatekey-admin-linux-arm64 ./cmd/gatekey-admin && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o /admin-binaries/gatekey-admin-darwin-amd64 ./cmd/gatekey-admin && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o /admin-binaries/gatekey-admin-darwin-arm64 ./cmd/gatekey-admin

# Build hub binaries for multiple platforms
RUN mkdir -p /hub-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /hub-binaries/gatekey-hub-linux-amd64 ./cmd/gatekey-hub && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /hub-binaries/gatekey-hub-linux-arm64 ./cmd/gatekey-hub

# Build mesh gateway binaries for multiple platforms
RUN mkdir -p /mesh-gateway-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /mesh-gateway-binaries/gatekey-mesh-gateway-linux-amd64 ./cmd/gatekey-mesh-gateway && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /mesh-gateway-binaries/gatekey-mesh-gateway-linux-arm64 ./cmd/gatekey-mesh-gateway

# Runtime stage
FROM alpine:3.23

# OCI labels for supply chain security
LABEL org.opencontainers.image.title="GateKey Server"
LABEL org.opencontainers.image.description="GateKey Zero-Trust VPN Control Plane"
LABEL org.opencontainers.image.vendor="Dye Tech"
LABEL org.opencontainers.image.source="https://github.com/dye-tech/GateKey"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Create non-root user early with explicit UID/GID
# Using UID 65532 (standard nonroot UID used by distroless)
RUN addgroup -g 65532 -S gatekey && \
    adduser -u 65532 -S -G gatekey -h /app -s /sbin/nologin gatekey

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy server binary
COPY --from=builder /gatekey-server /usr/local/bin/gatekey-server

# Copy gateway, client, admin, hub, and mesh binaries for download
COPY --from=builder /gateway-binaries /app/bin
COPY --from=builder /client-binaries /app/bin
COPY --from=builder /admin-binaries /app/bin
COPY --from=builder /hub-binaries /app/bin
COPY --from=builder /mesh-gateway-binaries /app/bin

# Copy frontend assets
COPY web/dist /app/web/dist

# Copy migrations
COPY migrations /app/migrations

# Copy install scripts
COPY scripts /app/scripts

# Set ownership for app directory
RUN chown -R 65532:65532 /app

# Switch to non-root user (using numeric UID for security scanners)
USER 65532:65532

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/gatekey-server"]
CMD ["--config", "/app/configs/gatex.yaml"]
