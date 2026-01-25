# Build backend
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gatekey-server ./cmd/gatekey-server

# Final image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 gatekey && \
    adduser -u 1000 -G gatekey -s /sbin/nologin -D gatekey

WORKDIR /app

COPY --from=builder /gatekey-server /app/gatekey-server

# Create data directory for PKI storage
RUN mkdir -p /app/data && chown -R gatekey:gatekey /app

USER gatekey

EXPOSE 8080

ENTRYPOINT ["/app/gatekey-server"]
