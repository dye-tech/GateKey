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

# Runtime stage
FROM alpine:3.23

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy server binary
COPY --from=builder /gatekey-server /usr/local/bin/gatekey-server

# Copy gateway and client binaries for download
COPY --from=builder /gateway-binaries /app/bin
COPY --from=builder /client-binaries /app/bin

# Copy frontend assets
COPY web/dist /app/web/dist

# Copy migrations
COPY migrations /app/migrations

# Copy install scripts
COPY scripts /app/scripts

# Create non-root user
RUN adduser -D -g '' gatekey && \
    chown -R gatekey:gatekey /app

USER gatekey

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/gatekey-server"]
CMD ["--config", "/app/configs/gatex.yaml"]
