<p align="center">
  <img src="docs/logo.png" alt="GateKey" width="400">
</p>

<p align="center">
  <strong>Zero Trust VPN - Authenticate First, Connect Second</strong>
</p>

<p align="center">
  <a href="https://github.com/dye-tech/GateKey/actions/workflows/ci.yml"><img src="https://github.com/dye-tech/GateKey/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/dye-tech/GateKey/actions/workflows/codeql.yml"><img src="https://github.com/dye-tech/GateKey/actions/workflows/codeql.yml/badge.svg" alt="CodeQL"></a>
  <a href="https://goreportcard.com/report/github.com/dye-tech/GateKey"><img src="https://goreportcard.com/badge/github.com/dye-tech/GateKey" alt="Go Report Card"></a>
  <a href="https://github.com/dye-tech/GateKey/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"></a>
  <a href="https://golang.org/doc/go1.25"><img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go" alt="Go Version"></a>
  <a href="https://hub.docker.com/r/dyetech/gatekey-server"><img src="https://img.shields.io/docker/v/dyetech/gatekey-server?label=Docker&logo=docker" alt="Docker"></a>
</p>

<p align="center">
  <a href="#for-end-users">End Users</a> •
  <a href="#for-administrators">Administrators</a> •
  <a href="#how-it-works">How It Works</a> •
  <a href="https://github.com/dye-tech/gatekey-helm-chart">Helm Chart</a>
</p>

---

GateKey is a zero-trust VPN solution that wraps OpenVPN. Users authenticate via their company's identity provider (Okta, Azure AD, etc.) and get short-lived VPN credentials automatically. No passwords to remember, no certificates to manage.

## Table of Contents

- [For End Users](#for-end-users)
  - [Prerequisites](#prerequisites)
  - [Install the Client](#install-the-client)
  - [Connect to VPN](#connect-to-vpn)
  - [Client Commands](#client-commands)
- [For Administrators](#for-administrators)
  - [Architecture Overview](#architecture-overview)
  - [Server Setup](#server-setup)
  - [Web UI Setup](#web-ui-setup)
  - [Gateway Setup](#gateway-setup)
  - [Admin CLI Setup](#admin-cli-setup)
  - [Configuration](#configuration)
- [How It Works](#how-it-works)
- [Components Reference](#components-reference)
- [API Reference](#api-reference)
- [Security Features](#security-features)
- [Development](#development)
- [License](#license)

---

# For End Users

This section is for employees who need to connect to your company's VPN.

## Prerequisites

You need OpenVPN installed on your machine:

| Platform | Installation |
|----------|--------------|
| macOS | `brew install openvpn` |
| Ubuntu/Debian | `sudo apt install openvpn` |
| Fedora | `sudo dnf install openvpn` |
| Windows | [Download OpenVPN Connect](https://openvpn.net/client/) |

## Install the Client

### macOS / Linux (Homebrew)

```bash
brew tap dye-tech/gatekey
brew install gatekey
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/dye-tech/GateKey/releases).

```bash
# Linux (amd64)
curl -LO https://github.com/dye-tech/GateKey/releases/latest/download/gatekey-linux-amd64.tar.gz
tar -xzf gatekey-linux-amd64.tar.gz
sudo mv gatekey /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/dye-tech/GateKey/releases/latest/download/gatekey-darwin-arm64.tar.gz
tar -xzf gatekey-darwin-arm64.tar.gz
sudo mv gatekey /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/dye-tech/GateKey.git
cd GateKey
make build-client
sudo cp bin/gatekey /usr/local/bin/
```

## Connect to VPN

### Step 1: Configure Your Server

Run this once to point the client at your company's GateKey server:

```bash
gatekey config init --server https://vpn.yourcompany.com
```

### Step 2: Login

Authenticate with your company credentials:

```bash
gatekey login
```

This opens your browser for SSO login (Okta, Azure AD, Google, etc.).

**For headless/automated environments**, use an API key:

```bash
gatekey login --api-key gk_your_api_key_here
```

### Step 3: Connect

```bash
gatekey connect
```

That's it! You're connected.

## Client Commands

| Command | Description |
|---------|-------------|
| `gatekey login` | Authenticate with SSO or API key |
| `gatekey connect` | Connect to VPN |
| `gatekey disconnect` | Disconnect from VPN |
| `gatekey status` | Check connection status |
| `gatekey list` | List available gateways |
| `gatekey logout` | Clear saved session |
| `gatekey config init` | Configure server URL |

### Multi-Gateway Support

Connect to multiple gateways simultaneously:

```bash
gatekey connect us-east-1    # Connect to first gateway
gatekey connect eu-west-1    # Connect to second gateway
gatekey status               # Shows all connections
gatekey disconnect us-east-1 # Disconnect from specific gateway
gatekey disconnect --all     # Disconnect from all
```

### Alternative: Web UI

If you prefer not to use the CLI:

1. Go to `https://vpn.yourcompany.com` in your browser
2. Login with your company credentials
3. Click "Download Config"
4. Import the `.ovpn` file into your OpenVPN client

---

# For Administrators

This section is for IT administrators setting up GateKey infrastructure.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    GATEKEY CONTROL PLANE                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Web UI     │  │   REST API   │  │ Embedded CA  │          │
│  │  (React)     │  │    (Go)      │  │   (PKI)      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                           │                                     │
│  ┌────────────────────────┴───────────────────────────────┐    │
│  │                     PostgreSQL                          │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GATEKEY GATEWAY                            │
│  ┌──────────────────┐  ┌──────────────────┐                    │
│  │  OpenVPN Server  │◄─┤  Gateway Agent   │                    │
│  │    (Stock)       │  │  (Hook Handler)  │                    │
│  └──────────────────┘  └──────────────────┘                    │
│                              │                                  │
│  ┌───────────────────────────┴──────────────────────────────┐  │
│  │           Per-Identity Firewall Rules (nftables)          │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Server Setup

The GateKey server is the control plane that handles authentication, certificate generation, and policy management.

### Prerequisites

- PostgreSQL 14+
- Go 1.25+ (if building from source)

### Option 1: Kubernetes (Recommended)

```bash
# Add the Helm repository
helm repo add gatekey https://dye-tech.github.io/gatekey-helm-chart
helm repo update

# Install with default settings
helm install gatekey gatekey/gatekey -n gatekey --create-namespace

# Or install with custom admin password
helm install gatekey gatekey/gatekey -n gatekey --create-namespace \
  --set secrets.adminPassword="your-secure-password"
```

Retrieve the auto-generated admin password:

```bash
kubectl get secret gatekey-admin-password -n gatekey -o jsonpath='{.data.admin-password}' | base64 -d
```

See [gatekey-helm-chart](https://github.com/dye-tech/gatekey-helm-chart) for all configuration options.

### Option 2: Docker

```bash
docker run -d \
  --name gatekey-server \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://gatekey:password@host.docker.internal/gatekey?sslmode=disable" \
  -e GATEKEY_ADMIN_PASSWORD="your-secure-password" \
  dyetech/gatekey-server:latest
```

### Option 3: Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: gatekey
      POSTGRES_PASSWORD: password
      POSTGRES_DB: gatekey
    volumes:
      - postgres_data:/var/lib/postgresql/data

  gatekey-server:
    image: dyetech/gatekey-server:latest
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://gatekey:password@postgres/gatekey?sslmode=disable
      GATEKEY_ADMIN_PASSWORD: your-secure-password
    depends_on:
      - postgres

  gatekey-web:
    image: dyetech/gatekey-web:latest
    ports:
      - "80:8080"
    depends_on:
      - gatekey-server

volumes:
  postgres_data:
```

Run:

```bash
docker-compose up -d
```

### Option 4: Build from Source

```bash
# Clone
git clone https://github.com/dye-tech/GateKey.git
cd GateKey

# Build server
make build-server

# Setup database
export DATABASE_URL="postgres://gatekey:password@localhost/gatekey?sslmode=disable"
make migrate-up

# Configure
cp configs/gatekey.yaml.example configs/gatekey.yaml
# Edit configs/gatekey.yaml with your settings

# Run
./bin/gatekey-server --config configs/gatekey.yaml
```

---

## Web UI Setup

The Web UI provides a browser-based interface for users to download VPN configs and for admins to manage the system.

### Option 1: Docker (Standalone)

```bash
docker run -d \
  --name gatekey-web \
  -p 80:8080 \
  dyetech/gatekey-web:latest
```

**Note:** Configure your reverse proxy to route `/api` requests to the gatekey-server.

### Option 2: Nginx Reverse Proxy

If running the server and web UI separately, use nginx to proxy API requests:

```nginx
server {
    listen 80;
    server_name vpn.yourcompany.com;

    # Web UI
    location / {
        proxy_pass http://gatekey-web:8080;
    }

    # API requests to server
    location /api {
        proxy_pass http://gatekey-server:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Option 3: Build from Source

```bash
cd GateKey

# Install frontend dependencies
cd web && npm install

# Build frontend
npm run build
cd ..

# The built files are in web/dist/
# Serve with any static file server or nginx
```

---

## Gateway Setup

The Gateway runs alongside OpenVPN and handles certificate validation and per-user firewall rules.

### Prerequisites

- Linux server with root access
- OpenVPN 2.5+
- nftables (for firewall rules)

### Option 1: Install Script (Recommended)

```bash
curl -sSL https://vpn.yourcompany.com/scripts/install-gateway.sh | sudo bash
```

This script:
- Downloads the gateway binary
- Installs OpenVPN if not present
- Configures the gateway service
- Sets up firewall rules

### Option 2: Manual Installation

```bash
# Download gateway binary
curl -LO https://vpn.yourcompany.com/downloads/gatekey-gateway-linux-amd64
chmod +x gatekey-gateway-linux-amd64
sudo mv gatekey-gateway-linux-amd64 /usr/local/bin/gatekey-gateway

# Create config directory
sudo mkdir -p /etc/gatekey

# Create config file
sudo cat > /etc/gatekey/gateway.yaml << EOF
server_url: https://vpn.yourcompany.com
gateway_token: your-gateway-registration-token
openvpn_config: /etc/openvpn/server.conf
EOF

# Create systemd service
sudo cat > /etc/systemd/system/gatekey-gateway.service << EOF
[Unit]
Description=GateKey Gateway Agent
After=network.target openvpn.service

[Service]
Type=simple
ExecStart=/usr/local/bin/gatekey-gateway --config /etc/gatekey/gateway.yaml
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable gatekey-gateway
sudo systemctl start gatekey-gateway
```

### Option 3: Build from Source

```bash
cd GateKey
make build-gateway
sudo cp bin/gatekey-gateway /usr/local/bin/
```

---

## Admin CLI Setup

The Admin CLI (`gatekey-admin`) allows administrators to manage users, policies, and gateways from the command line.

### Installation

#### Download Binary

```bash
# Linux
curl -LO https://vpn.yourcompany.com/downloads/gatekey-admin-linux-amd64
chmod +x gatekey-admin-linux-amd64
sudo mv gatekey-admin-linux-amd64 /usr/local/bin/gatekey-admin

# macOS
curl -LO https://vpn.yourcompany.com/downloads/gatekey-admin-darwin-arm64
chmod +x gatekey-admin-darwin-arm64
sudo mv gatekey-admin-darwin-arm64 /usr/local/bin/gatekey-admin
```

#### Build from Source

```bash
cd GateKey
make build-admin
sudo cp bin/gatekey-admin /usr/local/bin/
```

### Usage

```bash
# Login as admin
gatekey-admin login --server https://vpn.yourcompany.com

# List users
gatekey-admin users list

# Create API key for a user
gatekey-admin api-keys create --user user@example.com --name "CI/CD Key"

# List gateways
gatekey-admin gateways list

# Manage access rules
gatekey-admin rules list
gatekey-admin rules create --name "Engineering" --cidr "10.0.0.0/8"
```

---

## Configuration

### OIDC Provider Setup

Configure your identity provider in `configs/gatekey.yaml`:

```yaml
auth:
  oidc:
    enabled: true
    providers:
      - name: "okta"
        display_name: "Company SSO"
        issuer: "https://yourcompany.okta.com"
        client_id: "your-client-id"
        client_secret: "your-client-secret"
        redirect_url: "https://vpn.yourcompany.com/api/v1/auth/oidc/callback"
        scopes: ["openid", "profile", "email", "groups"]
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `GATEKEY_ADMIN_PASSWORD` | Initial admin password | Auto-generated |
| `GATEKEY_JWT_SECRET` | JWT signing secret | Auto-generated |
| `GATEKEY_CA_VALIDITY_DAYS` | CA certificate validity | 3650 |
| `GATEKEY_CERT_VALIDITY_HOURS` | Client cert validity | 24 |

### Network and Access Rules

Use the admin UI or API to:
- Define **Networks** (CIDR blocks like `10.0.0.0/8`)
- Create **Access Rules** (IP/hostname whitelists)
- Assign rules to users or groups

---

# How It Works

1. **`gatekey login`** - Opens your browser to authenticate with your company's SSO
2. **`gatekey connect`** - Downloads a short-lived VPN config (valid ~24 hours) and connects using OpenVPN
3. Your firewall rules are automatically applied based on your role/group membership
4. Configs auto-refresh, so you never deal with expired certificates

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Your Computer  │    │  GateKey Server  │    │  Your Company   │
│                 │    │                  │    │    Network      │
│  ┌───────────┐  │    │  ┌────────────┐  │    │                 │
│  │  gatekey  │──┼────┼─►│   Auth +   │  │    │  ┌───────────┐  │
│  │   CLI     │  │    │  │  PKI       │  │    │  │ Internal  │  │
│  └───────────┘  │    │  └────────────┘  │    │  │ Services  │  │
│       │         │    │                  │    │  └───────────┘  │
│       ▼         │    │  ┌────────────┐  │    │       ▲         │
│  ┌───────────┐  │    │  │  OpenVPN   │  │    │       │         │
│  │  OpenVPN  │──┼────┼─►│  Gateway   │──┼────┼───────┘         │
│  └───────────┘  │    │  └────────────┘  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

---

# Components Reference

## Binaries

| Binary | Description | Who Uses It |
|--------|-------------|-------------|
| `gatekey` | VPN client CLI | End users |
| `gatekey-server` | Control plane server | Administrators |
| `gatekey-gateway` | Gateway agent (runs with OpenVPN) | Gateway servers |
| `gatekey-admin` | Admin CLI for policy management | Administrators |
| `gatekey-hub` | Multi-gateway hub coordinator | Large deployments |
| `gatekey-mesh-gateway` | Mesh networking gateway | Mesh deployments |

## Docker Images

| Image | Description |
|-------|-------------|
| [`dyetech/gatekey-server`](https://hub.docker.com/r/dyetech/gatekey-server) | Control plane (API + embedded CA) |
| [`dyetech/gatekey-web`](https://hub.docker.com/r/dyetech/gatekey-web) | Web UI (nginx + React) |

---

# API Reference

See [docs/api.md](docs/api.md) for full API documentation.

### Key Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/auth/oidc/login` | Initiate SSO login |
| `GET /api/v1/auth/api-key/validate` | Validate API key |
| `POST /api/v1/configs/generate` | Generate VPN config |
| `GET /api/v1/gateways` | List available gateways |
| `GET /api/v1/api-keys` | Manage API keys |
| `GET /api/v1/admin/networks` | Manage networks |
| `GET /api/v1/admin/access-rules` | Manage access rules |
| `GET /api/v1/admin/login-logs` | View login activity |

---

# Security Features

- **Zero Trust**: No network access without authentication
- **Short-Lived Certificates**: Auto-expire after 24 hours (configurable)
- **Per-Identity Firewall**: Each user gets their own firewall rules
- **API Key Authentication**: Programmatic access for CLI and automation
- **FIPS Compliance**: Built with FIPS-validated crypto (when enabled)
- **Audit Logging**: All access is logged
- **Login Monitoring**: Track all authentication events with IP, location, and status

---

# Development

```bash
make dev              # Run server in dev mode
make test             # Run tests
make lint             # Run linter
make frontend-dev     # Run frontend in dev mode
make build            # Build all binaries
make build-client     # Build only client
make build-server     # Build only server
make build-gateway    # Build only gateway
make build-admin      # Build only admin CLI
```

---

# License

Apache 2.0
