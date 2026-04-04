<p align="center">
  <img src="docs/logo.png" alt="GateKey" width="400">
</p>

<h3 align="center">Zero Trust VPN &mdash; Authenticate First, Connect Second</h3>

<p align="center">
  Wrap OpenVPN and WireGuard with SSO, short-lived certificates, and per-user firewall rules.<br>
  No passwords to remember. No certificates to manage. Just <code>gatekey login</code> and <code>gatekey connect</code>.
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
  <a href="https://gatekey.net/docs/getting-started/quickstart"><strong>Documentation</strong></a> &nbsp;&bull;&nbsp;
  <a href="https://gatekey.net/docs/getting-started/quickstart"><strong>Quick Start</strong></a> &nbsp;&bull;&nbsp;
  <a href="https://github.com/dye-tech/gatekey-helm-chart"><strong>Helm Chart</strong></a> &nbsp;&bull;&nbsp;
  <a href="https://gatekey.net/docs/api/overview"><strong>API Reference</strong></a>
</p>

---

<p align="center">
  <img src="docs/diagrams/terminal-demo.svg" alt="GateKey CLI Demo" width="720">
</p>

---

## Why GateKey?

Traditional VPNs weren't built for zero trust. GateKey fixes that.

| | Traditional VPN | GateKey |
|---|---|---|
| **Certificates** | Long-lived (years), manually rotated | Short-lived (24h), auto-rotating |
| **Authentication** | Separate VPN passwords | SSO via Okta, Azure AD, Google, any OIDC/SAML |
| **Network access** | Full network after connect | Per-user firewall rules (least privilege) |
| **Access control** | Static, IP-based | Dynamic, identity-based, role-aware |
| **Certificate management** | Manual provisioning & revocation | Fully automatic with embedded CA |
| **Protocol support** | Pick one | OpenVPN + WireGuard, same security model |

---

## Features

<table>
<tr>
<td width="33%" valign="top">

### Zero Trust Security
Every connection is authenticated. Short-lived certificates auto-expire, and per-user nftables rules enforce least-privilege access at the network layer.

</td>
<td width="33%" valign="top">

### SSO Integration
Okta, Azure AD, Google Workspace, or any OIDC/SAML provider. No separate VPN passwords &mdash; users authenticate with existing credentials.

</td>
<td width="33%" valign="top">

### Dual Protocol
Choose OpenVPN for maximum compatibility (FIPS 140-3) or WireGuard for peak performance. Both use the same zero-trust model.

</td>
</tr>
<tr>
<td width="33%" valign="top">

### Multi-Gateway
Connect to multiple VPN gateways simultaneously. Automatic interface management &mdash; access resources across networks without reconnecting.

</td>
<td width="33%" valign="top">

### Mesh Networking
Hub-and-spoke topology for site-to-site connectivity. Connect remote offices through a central mesh hub with zero-trust controls.

</td>
<td width="33%" valign="top">

### Kubernetes Native
Deploy with Helm in minutes. Native secret storage, horizontal scaling, and seamless integration with your existing K8s infrastructure.

</td>
</tr>
</table>

---

## Quick Start

### Install

```bash
brew tap dye-tech/gatekey && brew install gatekey
```

<details>
<summary>Other installation methods</summary>

**Download binary:**
```bash
# Linux (amd64)
curl -LO https://github.com/dye-tech/GateKey/releases/latest/download/gatekey-linux-amd64.tar.gz
tar -xzf gatekey-linux-amd64.tar.gz && sudo mv gatekey /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/dye-tech/GateKey/releases/latest/download/gatekey-darwin-arm64.tar.gz
tar -xzf gatekey-darwin-arm64.tar.gz && sudo mv gatekey /usr/local/bin/
```

**Build from source:**
```bash
git clone https://github.com/dye-tech/GateKey.git && cd GateKey
make build-gatekey && sudo cp bin/gatekey /usr/local/bin/
```

</details>

### Connect

```bash
# 1. Point at your company's GateKey server
gatekey config init --server https://vpn.yourcompany.com

# 2. Authenticate via SSO (opens browser)
gatekey login

# 3. Connect
gatekey connect
```

That's it. Your credentials auto-refresh &mdash; no certificate management needed.

### CLI Reference

| Command | Description |
|---------|-------------|
| `gatekey login` | Authenticate with SSO or API key |
| `gatekey connect [gateway]` | Connect to VPN (optionally specify gateway) |
| `gatekey disconnect [--all]` | Disconnect from VPN |
| `gatekey status` | Check connection status |
| `gatekey list` | List available gateways |
| `gatekey config init` | Configure server URL |

**Multi-gateway example:**
```bash
gatekey connect us-east-1    # Connect to first gateway
gatekey connect eu-west-1    # Connect to second simultaneously
gatekey status               # Shows all active connections
```

---

## Architecture

<p align="center">
  <img src="docs/diagrams/readme-architecture.svg" alt="GateKey Architecture" width="800">
</p>

**Control Plane** &mdash; Handles authentication, certificate generation, and policy management. Deploys as a single Go binary + PostgreSQL.

**Gateway Agents** &mdash; Lightweight agents on each VPN node that sync firewall rules and manage VPN tunnels. Available for both OpenVPN and WireGuard.

**Per-Identity Firewall** &mdash; Each connected user gets individualized nftables rules based on their role and group membership. Rules update in real-time when policies change.

---

## Deploy

### Kubernetes (Recommended)

```bash
helm repo add gatekey https://dye-tech.github.io/gatekey-helm-chart
helm repo update
helm install gatekey gatekey/gatekey -n gatekey --create-namespace
```

See [gatekey-helm-chart](https://github.com/dye-tech/gatekey-helm-chart) for configuration options.

### Docker Compose

```yaml
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
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://gatekey:password@postgres/gatekey?sslmode=disable
      GATEKEY_ADMIN_PASSWORD: your-secure-password
    depends_on: [postgres]

  gatekey-web:
    image: dyetech/gatekey-web:latest
    ports: ["80:8080"]
    depends_on: [gatekey-server]

volumes:
  postgres_data:
```

```bash
docker-compose up -d
```

<details>
<summary>More deployment options</summary>

**Docker (server only):**
```bash
docker run -d --name gatekey-server -p 8080:8080 \
  -e DATABASE_URL="postgres://gatekey:password@host.docker.internal/gatekey?sslmode=disable" \
  -e GATEKEY_ADMIN_PASSWORD="your-secure-password" \
  dyetech/gatekey-server:latest
```

**Build from source:**
```bash
git clone https://github.com/dye-tech/GateKey.git && cd GateKey
make build-gatekey-server
export DATABASE_URL="postgres://gatekey:password@localhost/gatekey?sslmode=disable"
make migrate-up
./bin/gatekey-server --config configs/gatekey.yaml
```

</details>

> For gateway setup, identity provider configuration, and admin CLI usage, see the [full documentation](https://gatekey.net/docs/getting-started/quickstart).

---

## Security

GateKey implements a zero-trust model where **all traffic is denied by default**. Users access only the resources explicitly permitted by their access rules.

- **Short-lived certificates** &mdash; Auto-expire after 24 hours (configurable)
- **Per-identity firewall** &mdash; Individualized nftables rules per connected user
- **Embedded CA** &mdash; No external PKI infrastructure needed
- **FIPS 140-3 compliance** &mdash; Available with OpenVPN crypto profiles
- **Audit logging** &mdash; All authentication and access events logged
- **Geo-fencing** &mdash; IP-based access restrictions
- **JIT access** &mdash; Temporary, approval-based resource access with auto-revocation
- **Session recording** &mdash; Terminal recording and proxy logging

---

## Components

| Binary | Description |
|--------|-------------|
| `gatekey` | VPN client CLI |
| `gatekey-server` | Control plane (API + embedded CA) |
| `gatekey-gateway` | OpenVPN gateway agent |
| `gatekey-wireguard-gateway` | WireGuard gateway agent |
| `gatekey-admin` | Admin CLI for policy management |
| `gatekey-hub` / `gatekey-wireguard-hub` | Mesh hub (OpenVPN / WireGuard) |
| `gatekey-mesh-gateway` / `gatekey-wireguard-mesh-gateway` | Mesh spoke (OpenVPN / WireGuard) |

**Docker images:** [`dyetech/gatekey-server`](https://hub.docker.com/r/dyetech/gatekey-server) &bull; [`dyetech/gatekey-web`](https://hub.docker.com/r/dyetech/gatekey-web) &bull; [`dyetech/gatekey-gateway`](https://hub.docker.com/r/dyetech/gatekey-gateway) &bull; [`dyetech/gatekey-wireguard-gateway`](https://hub.docker.com/r/dyetech/gatekey-wireguard-gateway)

---

## Development

```bash
make dev          # Run server in dev mode
make test         # Run tests
make lint         # Run linter
make frontend-dev # Run frontend dev server
make build        # Build all binaries
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

---

## Documentation

Full documentation is available at **[gatekey.net](https://gatekey.net)**

| Guide | Description |
|-------|-------------|
| [Quick Start](https://gatekey.net/docs/getting-started/quickstart) | Get connected in 5 minutes |
| [Server Setup](https://gatekey.net/docs/admin-guide/server-setup) | Deploy the control plane |
| [Gateway Setup](https://gatekey.net/docs/admin-guide/gateway-setup) | Configure VPN gateways |
| [Identity Providers](https://gatekey.net/docs/admin-guide/identity-providers) | SSO configuration (OIDC/SAML) |
| [Access Control](https://gatekey.net/docs/admin-guide/access-control) | Firewall rules and policies |
| [Mesh Networking](https://gatekey.net/docs/user-guide/mesh-networking) | Site-to-site connectivity |
| [API Reference](https://gatekey.net/docs/api/overview) | REST API documentation |
| [Architecture](https://gatekey.net/docs/architecture/overview) | System design and components |

---

## License

[Apache 2.0](LICENSE)
