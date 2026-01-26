# Consolidated Dockerfile for all GateKey Go binaries
# Usage: docker build --build-arg BINARY=gatekey-server --build-arg RUNTIME_DEPS="ca-certificates tzdata" .

# Build stage
FROM golang:1.25-alpine AS builder

ARG BINARY=gatekey-server

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the specified binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app-binary ./cmd/${BINARY}

# Final image
FROM alpine:3.23

# Runtime dependencies vary by binary type:
#   server:    ca-certificates tzdata
#   openvpn:   ca-certificates tzdata openvpn nftables iptables iproute2
#   wireguard: ca-certificates tzdata wireguard-tools nftables iptables iproute2
ARG RUNTIME_DEPS="ca-certificates tzdata"
RUN apk add --no-cache ${RUNTIME_DEPS}

WORKDIR /app

COPY --from=builder /app-binary /app/gatekey

# Copy install scripts for server
COPY --from=builder /app/scripts /app/scripts

# Server listens on 8080
EXPOSE 8080

ENTRYPOINT ["/app/gatekey"]
