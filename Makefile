.PHONY: build build-server build-gateway build-admin build-client test lint clean dev migrate-up migrate-down help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_DIR=bin
SERVER_BINARY=$(BINARY_DIR)/gatekey-server
GATEWAY_BINARY=$(BINARY_DIR)/gatekey-gateway
ADMIN_BINARY=$(BINARY_DIR)/gatekey-admin
CLIENT_BINARY=$(BINARY_DIR)/gatekey

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: build

## Build targets

build: build-server build-gateway build-admin build-client ## Build all binaries
	@echo "Build complete"

build-server: ## Build the control plane server
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(SERVER_BINARY) ./cmd/gatekey-server

build-gateway: ## Build the gateway agent
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(GATEWAY_BINARY) ./cmd/gatekey-gateway

build-admin: ## Build the admin CLI tool
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(ADMIN_BINARY) ./cmd/gatekey-admin

build-client: ## Build the user VPN client
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(CLIENT_BINARY) ./cmd/gatekey

## Development targets

dev: ## Run in development mode
	$(GOCMD) run ./cmd/gatekey-server --config configs/gatekey.yaml

dev-gateway: ## Run gateway in development mode
	$(GOCMD) run ./cmd/gatekey-gateway --config configs/gateway.yaml

## Test targets

test: ## Run all tests
	$(GOTEST) -v -race -cover ./...

test-unit: ## Run unit tests only
	$(GOTEST) -v -short ./...

test-integration: ## Run integration tests
	$(GOTEST) -v -run Integration ./tests/...

test-coverage: ## Run tests with coverage report
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Lint and format

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	$(GOCMD) fmt ./...
	gofumpt -w .

## Database migrations

migrate-up: ## Run database migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-reset: ## Reset database (down all, then up)
	migrate -path migrations -database "$(DATABASE_URL)" down -all
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-create: ## Create new migration (usage: make migrate-create name=create_users)
	migrate create -ext sql -dir migrations -seq $(name)

## Dependencies

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

## Frontend

frontend-install: ## Install frontend dependencies
	cd web && npm install

frontend-build: ## Build frontend for production
	cd web && npm run build

frontend-dev: ## Run frontend in development mode
	cd web && npm run dev

## Clean

clean: ## Clean build artifacts
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	cd web && rm -rf dist node_modules

## Cross-compile for multiple platforms

build-client-all: ## Build client for all platforms
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-amd64 ./cmd/gatekey
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-arm64 ./cmd/gatekey
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-amd64 ./cmd/gatekey
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-arm64 ./cmd/gatekey
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-windows-amd64.exe ./cmd/gatekey
	@echo "Client binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64"

build-gateway-all: ## Build gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-amd64 ./cmd/gatekey-gateway
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-arm64 ./cmd/gatekey-gateway
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-amd64 ./cmd/gatekey-gateway
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-arm64 ./cmd/gatekey-gateway
	@echo "Gateway binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

## Docker

docker-build: ## Build Docker image
	docker build -t gatekey:latest .

docker-build-web: ## Build web Docker image
	docker build -f Dockerfile.web -t gatekey-web:latest .

docker-compose-up: ## Start with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop docker-compose
	docker-compose down

## Certificates (development)

gen-ca: ## Generate development CA
	@mkdir -p certs
	openssl genrsa -out certs/ca.key 4096
	openssl req -new -x509 -days 365 -key certs/ca.key -out certs/ca.crt \
		-subj "/C=US/ST=State/L=City/O=GateKey/CN=GateKey Development CA"

gen-server-cert: ## Generate development server certificate
	@mkdir -p certs
	openssl genrsa -out certs/server.key 2048
	openssl req -new -key certs/server.key -out certs/server.csr \
		-subj "/C=US/ST=State/L=City/O=GateKey/CN=localhost"
	openssl x509 -req -days 365 -in certs/server.csr -CA certs/ca.crt \
		-CAkey certs/ca.key -CAcreateserial -out certs/server.crt

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
