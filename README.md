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
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#how-it-works">How It Works</a> •
  <a href="#server-setup">Server Setup</a>
</p>

---

GateKey is a zero-trust VPN solution that wraps OpenVPN. Users authenticate via their company's identity provider (Okta, Azure AD, etc.) and get short-lived VPN credentials automatically. No passwords to remember, no certificates to manage.

## Installation

### macOS

```bash
brew install gatekey
```

### Linux

```bash
# Download the latest release
curl -LO https://github.com/dye-tech/GateKey/releases/latest/download/gatekey-linux-amd64
chmod +x gatekey-linux-amd64
sudo mv gatekey-linux-amd64 /usr/local/bin/gatekey
```

### Windows

Download the latest `.exe` from the [releases page](https://github.com/dye-tech/GateKey/releases).

### From Source

```bash
git clone https://github.com/dye-tech/GateKey.git
cd GateKey
make build-client
sudo cp bin/gatekey /usr/local/bin/
```

## Quick Start

**1. Configure your server** (one time setup)

```bash
gatekey config init --server https://vpn.yourcompany.com
```

**2. Login** (opens your browser)

```bash
gatekey login

# Or use an API key for headless/automated environments
gatekey login --api-key gk_your_api_key_here
```

**3. Connect to VPN**

```bash
gatekey connect
```

That's it! You're connected.

### Other Commands

```bash
gatekey status       # Check connection status
gatekey disconnect   # Disconnect from VPN
gatekey list         # List available gateways
gatekey logout       # Clear saved session
```

### Multi-Gateway Support

Connect to multiple gateways simultaneously:

```bash
gatekey connect us-east-1    # Connect to first gateway
gatekey connect eu-west-1    # Connect to second gateway
gatekey status               # Shows all connections
gatekey disconnect us-east-1 # Disconnect from specific gateway
gatekey disconnect --all     # Disconnect from all
```

## How It Works

1. **`gatekey login`** - Opens your browser to authenticate with your company's SSO (Okta, Azure AD, Google, etc.)
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

## Requirements

- **OpenVPN** must be installed:
  - macOS: `brew install openvpn`
  - Ubuntu/Debian: `sudo apt install openvpn`
  - Fedora: `sudo dnf install openvpn`
  - Windows: [OpenVPN Connect](https://openvpn.net/client/)

## Alternative: Web UI + Manual OpenVPN

If you prefer not to use the CLI, you can:

1. Go to `https://vpn.yourcompany.com` in your browser
2. Login with your company credentials
3. Click "Download Config"
4. Import the `.ovpn` file into your OpenVPN client

---

# Server Setup

The following is for administrators setting up GateKey infrastructure.

## Architecture

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

## Server Installation

### Prerequisites

- Go 1.25+
- PostgreSQL 14+
- OpenVPN 2.5+ (on gateway servers)

### Deploy Control Plane

```bash
# Clone
git clone https://github.com/dye-tech/GateKey.git
cd GateKey

# Build
make build

# Setup database
export DATABASE_URL="postgres://gatekey:password@localhost/gatekey?sslmode=disable"
make migrate-up

# Configure (edit configs/gatekey.yaml with your OIDC settings)
cp configs/gatekey.yaml.example configs/gatekey.yaml

# Run
./bin/gatekey-server --config configs/gatekey.yaml
```

### Deploy Gateway

On each gateway server:

```bash
# Download gateway binary
curl -LO https://vpn.yourcompany.com/downloads/gatekey-gateway-linux-amd64
chmod +x gatekey-gateway-linux-amd64
sudo mv gatekey-gateway-linux-amd64 /usr/local/bin/gatekey-gateway

# Or use the install script
curl -sSL https://vpn.yourcompany.com/scripts/install-gateway.sh | sudo bash
```

## Configuration

### OIDC Provider Setup

```yaml
# configs/gatekey.yaml
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

### Network and Access Rules

Use the admin UI or API to:
- Define **Networks** (CIDR blocks like `10.0.0.0/8`)
- Create **Access Rules** (IP/hostname whitelists)
- Assign rules to users or groups

## Components

| Binary | Description |
|--------|-------------|
| `gatekey` | User VPN client (this is what end users install) |
| `gatekey-server` | Control plane server |
| `gatekey-gateway` | Gateway agent (runs alongside OpenVPN) |
| `gatekey-admin` | Admin CLI for managing policies |

## API Reference

See [docs/api.md](docs/api.md) for full API documentation.

### Key Endpoints

- `POST /api/v1/auth/oidc/login` - Initiate SSO login
- `GET /api/v1/auth/api-key/validate` - Validate API key
- `POST /api/v1/configs/generate` - Generate VPN config
- `GET /api/v1/gateways` - List available gateways
- `GET /api/v1/api-keys` - Manage API keys
- `GET /api/v1/admin/networks` - Manage networks
- `GET /api/v1/admin/access-rules` - Manage access rules
- `GET /api/v1/admin/login-logs` - View login activity
- `GET /api/v1/admin/login-logs/stats` - Login statistics

## Security Features

- **Zero Trust**: No network access without authentication
- **Short-Lived Certificates**: Auto-expire after 24 hours (configurable)
- **Per-Identity Firewall**: Each user gets their own firewall rules
- **API Key Authentication**: Programmatic access for CLI, automation, and CI/CD
- **FIPS Compliance**: Built with FIPS-validated crypto (when enabled)
- **Audit Logging**: All access is logged
- **Login Monitoring**: Track all authentication events with IP, location, and status

## Development

```bash
make dev              # Run server in dev mode
make test             # Run tests
make lint             # Run linter
make frontend-dev     # Run frontend in dev mode
```

## License

Apache 2.0
