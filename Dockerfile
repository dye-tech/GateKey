# Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Build backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gatekey-server ./cmd/gatekey-server

# Final image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /gatekey-server /app/gatekey-server

EXPOSE 8080

ENTRYPOINT ["/app/gatekey-server"]
