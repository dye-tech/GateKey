.PHONY: build build-gatekey build-gatekey-server build-gatekey-gateway build-gatekey-wireguard-gateway build-gatekey-admin build-gatekey-hub build-gatekey-mesh-gateway build-gatekey-wireguard-hub build-gatekey-wireguard-mesh-gateway test lint clean dev help release release-all

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_DIR=bin
RELEASE_DIR=dist

# Binary output paths
BINARY_GATEKEY=$(BINARY_DIR)/gatekey
BINARY_GATEKEY_SERVER=$(BINARY_DIR)/gatekey-server
BINARY_GATEKEY_GATEWAY=$(BINARY_DIR)/gatekey-gateway
BINARY_GATEKEY_WIREGUARD_GATEWAY=$(BINARY_DIR)/gatekey-wireguard-gateway
BINARY_GATEKEY_ADMIN=$(BINARY_DIR)/gatekey-admin
BINARY_GATEKEY_HUB=$(BINARY_DIR)/gatekey-hub
BINARY_GATEKEY_MESH_GATEWAY=$(BINARY_DIR)/gatekey-mesh-gateway
BINARY_GATEKEY_WIREGUARD_HUB=$(BINARY_DIR)/gatekey-wireguard-hub
BINARY_GATEKEY_WIREGUARD_MESH_GATEWAY=$(BINARY_DIR)/gatekey-wireguard-mesh-gateway

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

# =============================================================================
# Build Targets
# =============================================================================

build: build-gatekey build-gatekey-server build-gatekey-gateway build-gatekey-wireguard-gateway build-gatekey-admin build-gatekey-hub build-gatekey-mesh-gateway build-gatekey-wireguard-hub build-gatekey-wireguard-mesh-gateway ## Build all binaries
	@echo "Build complete"

build-gatekey: ## Build gatekey (VPN client CLI)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY) ./cmd/gatekey

build-gatekey-server: ## Build gatekey-server (control plane)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_SERVER) ./cmd/gatekey-server

build-gatekey-gateway: ## Build gatekey-gateway (OpenVPN gateway agent)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_GATEWAY) ./cmd/gatekey-gateway

build-gatekey-wireguard-gateway: ## Build gatekey-wireguard-gateway (WireGuard gateway agent)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_WIREGUARD_GATEWAY) ./cmd/gatekey-wireguard-gateway

build-gatekey-admin: ## Build gatekey-admin (admin CLI)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_ADMIN) ./cmd/gatekey-admin

build-gatekey-hub: ## Build gatekey-hub (OpenVPN mesh hub)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_HUB) ./cmd/gatekey-hub

build-gatekey-mesh-gateway: ## Build gatekey-mesh-gateway (OpenVPN mesh spoke)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_MESH_GATEWAY) ./cmd/gatekey-mesh-gateway

build-gatekey-wireguard-hub: ## Build gatekey-wireguard-hub (WireGuard mesh hub)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_WIREGUARD_HUB) ./cmd/gatekey-wireguard-hub

build-gatekey-wireguard-mesh-gateway: ## Build gatekey-wireguard-mesh-gateway (WireGuard mesh spoke)
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_GATEKEY_WIREGUARD_MESH_GATEWAY) ./cmd/gatekey-wireguard-mesh-gateway

# =============================================================================
# Release Targets (for Homebrew)
# =============================================================================

# All binaries to release
RELEASE_BINARIES=gatekey gatekey-server gatekey-gateway gatekey-wireguard-gateway gatekey-admin gatekey-hub gatekey-mesh-gateway gatekey-wireguard-hub gatekey-wireguard-mesh-gateway

# Release build function: $(call release-binary,binary-name)
define release-binary
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building $(1) for $$os/$$arch..."; \
		output_name="$(1)-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(RELEASE_DIR)/$$output_name/$(1) ./cmd/$(1); \
		cp README.md LICENSE $(RELEASE_DIR)/$$output_name/ 2>/dev/null || true; \
		tar -czf $(RELEASE_DIR)/$$output_name.tar.gz -C $(RELEASE_DIR) $$output_name; \
		rm -rf $(RELEASE_DIR)/$$output_name; \
		sha256sum $(RELEASE_DIR)/$$output_name.tar.gz >> $(RELEASE_DIR)/checksums.txt; \
	done
	@echo "$(1) release complete"
endef

release: ## Build release archives for all binaries
	@for binary in $(RELEASE_BINARIES); do \
		$(MAKE) release-$$binary; \
	done
	@echo "Release archives created in $(RELEASE_DIR)/"
	@echo "SHA256 checksums:"
	@cat $(RELEASE_DIR)/checksums.txt

release-all: clean release ## Clean and build all release artifacts
	@echo "Full release complete"

release-gatekey: ## Build gatekey release archives
	$(call release-binary,gatekey)

release-gatekey-server: ## Build gatekey-server release archives
	$(call release-binary,gatekey-server)

release-gatekey-gateway: ## Build gatekey-gateway release archives
	$(call release-binary,gatekey-gateway)

release-gatekey-wireguard-gateway: ## Build gatekey-wireguard-gateway release archives
	$(call release-binary,gatekey-wireguard-gateway)

release-gatekey-admin: ## Build gatekey-admin release archives
	$(call release-binary,gatekey-admin)

release-gatekey-hub: ## Build gatekey-hub release archives
	$(call release-binary,gatekey-hub)

release-gatekey-mesh-gateway: ## Build gatekey-mesh-gateway release archives
	$(call release-binary,gatekey-mesh-gateway)

release-gatekey-wireguard-hub: ## Build gatekey-wireguard-hub release archives
	$(call release-binary,gatekey-wireguard-hub)

release-gatekey-wireguard-mesh-gateway: ## Build gatekey-wireguard-mesh-gateway release archives
	$(call release-binary,gatekey-wireguard-mesh-gateway)

# =============================================================================
# Development Targets
# =============================================================================

dev: dev-gatekey-server ## Run gatekey-server in development mode (alias)

dev-gatekey-server: ## Run gatekey-server in development mode
	$(GOCMD) run ./cmd/gatekey-server --config configs/gatekey.yaml

dev-gatekey-gateway: ## Run gatekey-gateway in development mode
	$(GOCMD) run ./cmd/gatekey-gateway --config configs/gateway.yaml

dev-gatekey-wireguard-gateway: ## Run gatekey-wireguard-gateway in development mode
	$(GOCMD) run ./cmd/gatekey-wireguard-gateway --config configs/wireguard-gateway.yaml

dev-gatekey-hub: ## Run gatekey-hub in development mode
	$(GOCMD) run ./cmd/gatekey-hub --config configs/hub.yaml

dev-gatekey-mesh-gateway: ## Run gatekey-mesh-gateway in development mode
	$(GOCMD) run ./cmd/gatekey-mesh-gateway --config configs/mesh-gateway.yaml

dev-gatekey-wireguard-hub: ## Run gatekey-wireguard-hub in development mode
	$(GOCMD) run ./cmd/gatekey-wireguard-hub --config configs/wireguard-hub.yaml

dev-gatekey-wireguard-mesh-gateway: ## Run gatekey-wireguard-mesh-gateway in development mode
	$(GOCMD) run ./cmd/gatekey-wireguard-mesh-gateway --config configs/wireguard-mesh-gateway.yaml

# =============================================================================
# Test Targets
# =============================================================================

test: ## Run all tests
	$(GOTEST) -v -race -cover ./...

test-unit: ## Run unit tests only
	$(GOTEST) -v -short ./...

test-integration: ## Run integration tests
	$(GOTEST) -v -run Integration ./...

test-coverage: ## Run tests with coverage report
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Lint and Format
# =============================================================================

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	$(GOCMD) fmt ./...
	gofumpt -w .

# =============================================================================
# Database Migrations
# =============================================================================

migrate-up: ## Run database migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-reset: ## Reset database (down all, then up)
	migrate -path migrations -database "$(DATABASE_URL)" down -all
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-create: ## Create new migration (usage: make migrate-create name=create_users)
	migrate create -ext sql -dir migrations -seq $(name)

# =============================================================================
# Dependencies
# =============================================================================

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

# =============================================================================
# Frontend
# =============================================================================

frontend-install: ## Install frontend dependencies
	cd web && npm install

frontend-build: ## Build frontend for production
	cd web && npm run build

frontend-dev: ## Run frontend in development mode
	cd web && npm run dev

# =============================================================================
# Clean
# =============================================================================

clean: ## Clean build artifacts
	rm -rf $(BINARY_DIR)
	rm -rf $(RELEASE_DIR)
	rm -f coverage.out coverage.html

# =============================================================================
# Cross-Compile (all platforms)
# =============================================================================

build-gatekey-all: ## Build gatekey for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-amd64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-linux-arm64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-amd64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-darwin-arm64 ./cmd/gatekey
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-windows-amd64.exe ./cmd/gatekey
	@echo "gatekey built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64"

build-gatekey-server-all: ## Build gatekey-server for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-linux-amd64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-linux-arm64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-darwin-amd64 ./cmd/gatekey-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-server-darwin-arm64 ./cmd/gatekey-server
	@echo "gatekey-server built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-gateway-all: ## Build gatekey-gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-amd64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-linux-arm64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-amd64 ./cmd/gatekey-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-gateway-darwin-arm64 ./cmd/gatekey-gateway
	@echo "gatekey-gateway built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-wireguard-gateway-all: ## Build gatekey-wireguard-gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-gateway-linux-amd64 ./cmd/gatekey-wireguard-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-gateway-linux-arm64 ./cmd/gatekey-wireguard-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-gateway-darwin-amd64 ./cmd/gatekey-wireguard-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-gateway-darwin-arm64 ./cmd/gatekey-wireguard-gateway
	@echo "gatekey-wireguard-gateway built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-admin-all: ## Build gatekey-admin for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-linux-amd64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-linux-arm64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-darwin-amd64 ./cmd/gatekey-admin
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-admin-darwin-arm64 ./cmd/gatekey-admin
	@echo "gatekey-admin built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-hub-all: ## Build gatekey-hub for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-linux-amd64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-linux-arm64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-darwin-amd64 ./cmd/gatekey-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-hub-darwin-arm64 ./cmd/gatekey-hub
	@echo "gatekey-hub built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-mesh-gateway-all: ## Build gatekey-mesh-gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-linux-amd64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-linux-arm64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-darwin-amd64 ./cmd/gatekey-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-mesh-gateway-darwin-arm64 ./cmd/gatekey-mesh-gateway
	@echo "gatekey-mesh-gateway built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-wireguard-hub-all: ## Build gatekey-wireguard-hub for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-hub-linux-amd64 ./cmd/gatekey-wireguard-hub
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-hub-linux-arm64 ./cmd/gatekey-wireguard-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-hub-darwin-amd64 ./cmd/gatekey-wireguard-hub
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-hub-darwin-arm64 ./cmd/gatekey-wireguard-hub
	@echo "gatekey-wireguard-hub built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-gatekey-wireguard-mesh-gateway-all: ## Build gatekey-wireguard-mesh-gateway for all platforms
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-mesh-gateway-linux-amd64 ./cmd/gatekey-wireguard-mesh-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-mesh-gateway-linux-arm64 ./cmd/gatekey-wireguard-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-mesh-gateway-darwin-amd64 ./cmd/gatekey-wireguard-mesh-gateway
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/gatekey-wireguard-mesh-gateway-darwin-arm64 ./cmd/gatekey-wireguard-mesh-gateway
	@echo "gatekey-wireguard-mesh-gateway built for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

build-all: build-gatekey-all build-gatekey-server-all build-gatekey-gateway-all build-gatekey-wireguard-gateway-all build-gatekey-admin-all build-gatekey-hub-all build-gatekey-mesh-gateway-all build-gatekey-wireguard-hub-all build-gatekey-wireguard-mesh-gateway-all ## Build all binaries for all platforms
	@echo "All binaries built for all platforms"

# =============================================================================
# Docker
# =============================================================================

docker-build: ## Build Docker image
	docker build -f build/docker/server.Dockerfile -t gatekey:latest .

docker-build-web: ## Build web Docker image
	docker build -f build/docker/web.Dockerfile -t gatekey-web:latest .

docker-compose-up: ## Start with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop docker-compose
	docker-compose down

# =============================================================================
# Certificates (development)
# =============================================================================

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

# =============================================================================
# Help
# =============================================================================

help: ## Show this help
	@echo "GateKey Makefile"
	@echo ""
	@echo "Binaries:"
	@echo "  gatekey                        VPN client CLI (end users)"
	@echo "  gatekey-server                 Control plane server"
	@echo "  gatekey-gateway                OpenVPN gateway agent"
	@echo "  gatekey-wireguard-gateway      WireGuard gateway agent"
	@echo "  gatekey-admin                  Admin CLI for policy management"
	@echo "  gatekey-hub                    OpenVPN mesh hub"
	@echo "  gatekey-mesh-gateway           OpenVPN mesh spoke"
	@echo "  gatekey-wireguard-hub          WireGuard mesh hub"
	@echo "  gatekey-wireguard-mesh-gateway WireGuard mesh spoke"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-35s\033[0m %s\n", $$1, $$2}'

# =============================================================================
# Version Info
# =============================================================================

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
