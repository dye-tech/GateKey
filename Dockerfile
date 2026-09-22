# Consolidated Dockerfile for all GateKey Go binaries
# Usage: docker build --build-arg BINARY=gatekey-server --build-arg RUNTIME_DEPS="ca-certificates tzdata" .

# Build with Chainguard's maintained Go toolchain (git and a shell are included).
# Use --pull when building to pick up the latest base-image security fixes.
FROM cgr.dev/chainguard/go:latest AS builder

# Build with the image's patched toolchain rather than downloading another one.
ENV GOTOOLCHAIN=local

ARG BINARY=gatekey-server

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the specified binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app-binary ./cmd/${BINARY}

# Gateways and hubs need a shell and OS tools to configure VPNs and networking.
FROM cgr.dev/chainguard/wolfi-base:latest

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
