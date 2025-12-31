.PHONY: build build-server build-gateway build-admin build-client build-hub build-mesh-gateway test lint clean dev migrate-up migrate-down help release release-all release-hub release-mesh-gateway

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_DIR=bin
RELEASE_DIR=dist
SERVER_BINARY=$(BINARY_DIR)/gatekey-server
GATEWAY_BINARY=$(BINARY_DIR)/gatekey-gateway
ADMIN_BINARY=$(BINARY_DIR)/gatekey-admin
CLIENT_BINARY=$(BINARY_DIR)/gatekey
HUB_BINARY=$(BINARY_DIR)/gatekey-hub
MESH_GATEWAY_BINARY=$(BINARY_DIR)/gatekey-mesh-gateway

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags with version injection
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Platforms for release builds
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Default target
all: build

## Build targets

build: build-server build-gateway build-admin build-client build-hub build-mesh-gateway ## Build all binaries
	@echo "Build complete"

build-server: ## Build the control plane server
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(SERVER_BINARY) ./cmd/gatekey-server

build-gateway: ## Build the gateway agent
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(GATEWAY_BINARY) ./cmd/gatekey-gateway

build-admin: ## Build the admin CLI tool
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(ADMIN_BINARY) ./cmd/gatekey-admin

build-client: ## Build the user VPN client
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(CLIENT_BINARY) ./cmd/gatekey

build-hub: ## Build the mesh hub server
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(HUB_BINARY) ./cmd/gatekey-hub

build-mesh-gateway: ## Build the mesh gateway agent
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(MESH_GATEWAY_BINARY) ./cmd/gatekey-mesh-gateway

## Release targets for Homebrew

release: release-client release-server release-gateway release-admin release-hub release-mesh-gateway ## Build release archives for all binaries
	@echo "Release archives created in $(RELEASE_DIR)/"
	@echo "SHA256 checksums:"
	@cat $(RELEASE_DIR)/checksums.txt

release-all: clean release ## Clean and build all release artifacts
	@echo "Full release complete"

release-client: ## Build client release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey for $$os/$$arch..."; \
		output_name="gatekey-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey ./cmd/gatekey; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Client release complete"

release-server: ## Build server release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey-server for $$os/$$arch..."; \
		output_name="gatekey-server-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey-server ./cmd/gatekey-server; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Server release complete"

release-gateway: ## Build gateway release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey-gateway for $$os/$$arch..."; \
		output_name="gatekey-gateway-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey-gateway ./cmd/gatekey-gateway; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Gateway release complete"

release-admin: ## Build admin CLI release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey-admin for $$os/$$arch..."; \
		output_name="gatekey-admin-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey-admin ./cmd/gatekey-admin; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Admin CLI release complete"

release-hub: ## Build mesh hub release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey-hub for $$os/$$arch..."; \
		output_name="gatekey-hub-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey-hub ./cmd/gatekey-hub; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Mesh hub release complete"

release-mesh-gateway: ## Build mesh gateway release archives for all platforms
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gatekey-mesh-gateway for $$os/$$arch..."; \
		output_name="gatekey-mesh-gateway-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/gatekey-mesh-gateway ./cmd/gatekey-mesh-gateway; \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "Mesh gateway release complete"

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
	rm -rf $(RELEASE_DIR)
	rm -f coverage.out coverage.html

## Cross-compile for multiple platforms (legacy targets)

build-client-all: ## Build client for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-amd64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-arm64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-amd64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-arm64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-windows-amd64.exe ./cmd/gatekey
	@echo "Client binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64"

build-server-all: ## Build server for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-linux-amd64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-linux-arm64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-darwin-amd64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-darwin-arm64 ./cmd/gatekey-server
	@echo "Server binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gateway-all: ## Build gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-amd64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-arm64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-amd64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-arm64 ./cmd/gatekey-gateway
	@echo "Gateway binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-admin-all: ## Build admin CLI for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-linux-amd64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-linux-arm64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-darwin-amd64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-darwin-arm64 ./cmd/gatekey-admin
	@echo "Admin CLI binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-hub-all: ## Build mesh hub for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-linux-amd64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-linux-arm64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-darwin-amd64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-darwin-arm64 ./cmd/gatekey-hub
	@echo "Mesh hub binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-mesh-gateway-all: ## Build mesh gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-linux-amd64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-linux-arm64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-darwin-amd64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-darwin-arm64 ./cmd/gatekey-mesh-gateway
	@echo "Mesh gateway binaries built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-all: build-client-all build-server-all build-gateway-all build-admin-all build-hub-all build-mesh-gateway-all ## Build all binaries for all platforms
	@echo "All binaries built"

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

## Version info

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
