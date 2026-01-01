# GateKey - Zero Trust VPN

## Project Overview
GateKey is a zero-trust VPN solution that wraps OpenVPN to provide software-defined perimeter capabilities. It maintains 100% compatibility with existing OpenVPN clients while adding modern authentication, authorization, and access control.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| IdP Protocols | OIDC + SAML (both) | Enterprise compatibility from day one |
| Database | PostgreSQL | Production-grade, widely supported |
| PKI | Embedded CA | Simple deployment, full control |
| Deployment | Standalone binary | Easy adoption, K8s support included |
| Frontend | React/TypeScript web UI | Users can log in and download short-lived .ovpn configs |
| OpenVPN Integration | Hook scripts | 100% compatibility, no protocol reimplementation |
| Firewall | nftables (iptables fallback) | Modern Linux, per-identity rules |
| Framework | Gin (Go) + React (TS) | Well-supported, good performance |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      GATEKEY CONTROL PLANE                      │
├─────────────────────────────────────────────────────────────────┤
│  Web UI (React) │ REST API (Gin) │ Embedded CA (PKI)           │
├─────────────────────────────────────────────────────────────────┤
│  Auth Service   │ Policy Service │ Session Svc │ K8s Secrets   │
├─────────────────────────────────────────────────────────────────┤
│                         PostgreSQL                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GATEKEY GATEWAY                            │
│  OpenVPN Server (Stock) ◄─ Hook Executor (Go)                   │
│  Firewall Manager (nftables) - Per-identity rules               │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
gatekey/
├── cmd/
│   ├── gatekey/           # User VPN client (main binary for end users)
│   ├── gatekey-server/    # Control plane server
│   ├── gatekey-gateway/   # Gateway agent (runs alongside OpenVPN)
│   └── gatekey-admin/     # Admin CLI tool
├── internal/
│   ├── api/               # REST API (Gin handlers, middleware, routes)
│   ├── auth/              # Authentication (OIDC, SAML, local auth)
│   ├── pki/               # Embedded CA, certificate generation
│   ├── client/            # VPN client logic
│   ├── k8s/               # Kubernetes integration (secrets)
│   ├── db/                # Database stores
│   ├── openvpn/           # .ovpn generation, hook handlers
│   └── config/            # Configuration loading
├── web/                   # React frontend
├── migrations/            # PostgreSQL migrations
├── configs/               # Example configurations
└── docs/                  # Documentation
```

## Binaries

| Binary | Description | Platform |
|--------|-------------|----------|
| `gatekey` | User VPN client - this is what end users install | Linux, macOS |
| `gatekey-server` | Control plane server | Linux (Docker/K8s) |
| `gatekey-gateway` | Gateway agent (runs alongside OpenVPN) | Linux only |
| `gatekey-hub` | Mesh hub with zero-trust firewall | Linux only |
| `gatekey-mesh-gateway` | Mesh spoke (connects to hub) | Linux only |
| `gatekey-admin` | Admin CLI for managing policies | Linux, macOS |

**Note**: Gateway, hub, and mesh components require Linux for nftables firewall support.
See [docs/compatibility.md](docs/compatibility.md) for full platform support matrix.

## Build Commands

```bash
make build          # Build all binaries
make build-client   # Build just the VPN client
make test           # Run tests
make dev            # Run server in development mode
make frontend-build # Build frontend
```

## Configuration

Configuration is loaded from `configs/gatekey.yaml` or environment variables prefixed with `GATEKEY_`.

## User Flow

### CLI Client (Recommended)
```bash
gatekey config init --server https://vpn.company.com
gatekey login       # Opens browser for SSO
gatekey connect     # Downloads config and connects
gatekey disconnect  # Disconnect from VPN
```

### Multi-Gateway Support
The client supports connecting to multiple gateways simultaneously. Each gateway gets its own tun interface (tun0, tun1, etc.) to avoid conflicts.

```bash
gatekey connect gateway1      # Connect to first gateway (tun0)
gatekey connect gateway2      # Connect to second gateway (tun1)
gatekey status                # Shows all active connections
gatekey disconnect gateway1   # Disconnect from specific gateway
gatekey disconnect --all      # Disconnect from all gateways
```

### Web UI Flow
1. User accesses web UI at https://gatekey.example.com
2. Redirected to IdP (OIDC or SAML) for authentication
3. After auth, user sees dashboard with available gateways
4. User requests a VPN config (short-lived .ovpn file generated)
5. User downloads config and connects with any OpenVPN client
6. Gateway validates connection via hooks, applies firewall rules
7. User accesses only resources allowed by their policy

## API Endpoints

### Authentication
- `GET /api/v1/auth/oidc/login` - Initiate OIDC login
- `GET /api/v1/auth/oidc/callback` - OIDC callback
- `POST /api/v1/auth/local/login` - Local admin login
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/auth/session` - Get current session

### Configs
- `POST /api/v1/configs/generate` - Generate short-lived .ovpn
- `GET /api/v1/configs/download/:id` - Download .ovpn file

### Networks (Admin)
- `GET /api/v1/admin/networks` - List networks
- `POST /api/v1/admin/networks` - Create network
- `PUT /api/v1/admin/networks/:id` - Update network
- `DELETE /api/v1/admin/networks/:id` - Delete network

### Access Rules (Admin)
- `GET /api/v1/admin/access-rules` - List access rules
- `POST /api/v1/admin/access-rules` - Create access rule
- `PUT /api/v1/admin/access-rules/:id` - Update access rule
- `DELETE /api/v1/admin/access-rules/:id` - Delete access rule

### Gateways (Admin)
- `GET /api/v1/admin/gateways` - List gateways
- `POST /api/v1/admin/gateways` - Register gateway
- `PUT /api/v1/admin/gateways/:id` - Update gateway
- `DELETE /api/v1/admin/gateways/:id` - Delete gateway

### VPN Configs (Admin)
- `GET /api/v1/admin/configs` - List all gateway configs with user info
- `DELETE /api/v1/admin/configs/:id` - Revoke a gateway config
- `GET /api/v1/admin/mesh/configs` - List all mesh configs with user info
- `DELETE /api/v1/admin/mesh/configs/:id` - Revoke a mesh config

### Gateway Internal (for gateway agents)
- `POST /api/v1/gateway/heartbeat` - Send heartbeat status
- `POST /api/v1/gateway/verify` - Verify client connection
- `POST /api/v1/gateway/connect` - Report client connection
- `POST /api/v1/gateway/disconnect` - Report client disconnection
- `POST /api/v1/gateway/provision` - Provision OpenVPN server certificates
- `POST /api/v1/gateway/client-rules` - Get access rules for a connected client
- `POST /api/v1/gateway/all-rules` - Get all rules for periodic refresh

### Mesh Networking (Admin)
- `GET /api/v1/admin/mesh/hubs` - List mesh hubs
- `POST /api/v1/admin/mesh/hubs` - Create mesh hub
- `PUT /api/v1/admin/mesh/hubs/:id` - Update mesh hub (includes TLS Auth, Full Tunnel, DNS settings)
- `DELETE /api/v1/admin/mesh/hubs/:id` - Delete mesh hub
- `GET /api/v1/admin/mesh/hubs/:id/networks` - Get hub networks (zero-trust access control)
- `POST /api/v1/admin/mesh/hubs/:id/networks` - Assign network to hub
- `DELETE /api/v1/admin/mesh/hubs/:id/networks/:networkId` - Remove network from hub
- `GET /api/v1/admin/mesh/hubs/:id/spokes` - List spokes for hub
- `POST /api/v1/admin/mesh/hubs/:id/spokes` - Create spoke
- `PUT /api/v1/admin/mesh/spokes/:id` - Update spoke (includes Full Tunnel, DNS settings)

### User Mesh Access
- `GET /api/v1/mesh/hubs` - List hubs user can access
- `POST /api/v1/mesh/generate-config` - Generate client VPN config with zero-trust routes

### CA Management (Admin)
- `GET /api/v1/settings/ca/list` - List all CAs
- `POST /api/v1/settings/ca/prepare-rotation` - Prepare new CA for rotation
- `POST /api/v1/settings/ca/activate/:id` - Activate a pending CA
- `POST /api/v1/settings/ca/revoke/:id` - Revoke a CA
- `GET /api/v1/settings/ca/fingerprint` - Get active CA fingerprint

## Database Tables

- `local_users` - Local admin users
- `users` - User accounts (synced from IdP)
- `sessions` - Active sessions
- `auth_providers` - OIDC/SAML provider configs
- `gateways` - Registered gateway nodes
- `networks` - CIDR network blocks
- `access_rules` - IP/hostname whitelist rules
- `user_access_rules` - User to rule assignments
- `group_access_rules` - Group to rule assignments
- `generated_configs` - Generated gateway VPN configurations (with user, gateway, expiry, revocation tracking)
- `mesh_generated_configs` - Generated mesh VPN configurations (with user, hub, expiry, revocation tracking)
- `audit_logs` - Audit trail
- `pki_ca` - CA certificates with status, fingerprint, description
- `ca_rotation_events` - CA rotation audit trail
- `mesh_hubs` - Mesh hub servers (TLS Auth, Full Tunnel, DNS settings, local networks)
- `mesh_gateways` - Mesh spokes/gateways (Full Tunnel, DNS settings, local networks)
- `mesh_hub_networks` - Networks assigned to hubs (zero-trust access control)
- `mesh_hub_users` - User access to hubs
- `mesh_hub_groups` - Group access to hubs
- `mesh_gateway_users` - User access to spokes
- `mesh_gateway_groups` - Group access to spokes

## Kubernetes Integration

When running in Kubernetes:
- Initial admin password is saved to secret `gatekey-admin-init`
- Retrieve with: `kubectl get secret gatekey-admin-init -o jsonpath='{.data.admin-password}' | base64 -d`
- Service account needs `create`, `update`, `delete` permissions on secrets

## Security Features

- **Zero Trust**: No network access without authentication
- **Short-Lived Certificates**: Auto-expire after 24 hours (configurable)
- **Per-Identity Firewall**: Each user gets their own firewall rules using nftables
- **Real-Time Rule Enforcement**: Access rules take effect within 10 seconds
- **Audit Logging**: All access is logged
- **FIPS Ready**: Configurable crypto profiles (modern, fips, compatible)
- **Configurable VPN Subnet**: Per-gateway VPN subnet configuration (default: 172.31.255.0/24)
- **TLS-Auth**: Optional TLS-Auth key for additional security layer
- **Push-Based Config Updates**: Gateway auto-reprovisions when settings change
- **Graceful CA Rotation**: Zero-downtime CA rotation with dual-trust period

## Crypto Profiles

| Profile | Ciphers | Use Case |
|---------|---------|----------|
| `modern` | AES-256-GCM, CHACHA20-POLY1305 | Default, best performance |
| `fips` | AES-256-GCM, AES-128-GCM | FIPS 140-3 compliance required |
| `compatible` | AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC | Legacy client support |

## Real-Time Firewall Enforcement

When access rules change:
1. Gateway agent polls control plane every 10 seconds
2. Agent detects rule changes and updates nftables rules
3. Client traffic is immediately blocked/allowed based on new rules
4. No client reconnection required

## Push-Based Configuration Updates

When gateway settings change (crypto profile, port, protocol, subnet, TLS-Auth):
1. Database trigger computes new `config_version` (SHA256 hash of settings)
2. Gateway sends its config version in each heartbeat
3. Server responds with `needs_reprovision: true` if versions differ
4. Gateway auto-provisions new certs/config from control plane
5. OpenVPN restarts automatically to apply changes

This enables centralized management - change settings in the UI and gateways update automatically within 30 seconds.

## CA Rotation

GateKey supports graceful CA rotation with zero-downtime using a dual-trust period.

### CA Status Lifecycle
- `active`: Currently issuing certificates
- `pending`: Generated but not yet activated (for rotation)
- `retired`: No longer issuing, but still trusted for verification
- `revoked`: Revoked, no longer trusted

### Rotation Process
1. **Prepare**: `POST /settings/ca/prepare-rotation` creates new CA in pending state
2. **Activate**: `POST /settings/ca/activate/:id` retires old CA, activates new one
3. **Auto-detect**: Gateways detect change via `ca_fingerprint` in heartbeat response
4. **Reprovision**: Gateways automatically reprovision with new certificates
5. **Cleanup**: Optionally revoke old CA after grace period

### API Endpoints
- `GET /settings/ca/list` - List all CAs
- `POST /settings/ca/prepare-rotation` - Generate new pending CA
- `POST /settings/ca/activate/:id` - Activate pending CA
- `POST /settings/ca/revoke/:id` - Revoke a CA
- `GET /settings/ca/fingerprint` - Get active CA fingerprint

### Database Tables
- `pki_ca` - CA storage with status, fingerprint, description
- `ca_rotation_events` - Audit trail for rotation events

## TLS-Auth Support

TLS-Auth provides an additional HMAC signature layer for all TLS control channel packets.

- **Enable/Disable**: Per-gateway toggle in admin UI
- **Key Generation**: Automatic during gateway provisioning
- **Key Storage**: Stored in control plane DB, included in client configs
- **Rotation**: Change TLS-Auth setting to regenerate key and trigger reprovision

## Tunnel Modes

GateKey supports two tunnel modes per gateway:

### Split Tunnel (Default)
- Only routes traffic for networks the user has access to
- Routes are pushed dynamically based on user's access rules (CIDR type)
- User's default internet traffic goes through their normal connection
- DNS is NOT overridden by default

### Full Tunnel Mode
- Routes ALL traffic through the VPN (0.0.0.0/0)
- Enabled per-gateway via `full_tunnel_mode` setting
- Uses OpenVPN's `redirect-gateway def1 bypass-dhcp` directive
- Useful for enforcing all traffic inspection

### DNS Settings
- **Push DNS**: When enabled, pushes DNS server settings to VPN clients
- **DNS Servers**: Configurable list of DNS server IPs (e.g., `1.1.1.1, 8.8.8.8`)
- If Push DNS is enabled but no servers configured, defaults to `1.1.1.1` and `8.8.8.8`
- DNS pushing is disabled by default to avoid overriding client's local DNS

### Dynamic Route Pushing
Routes are pushed to clients dynamically during connection based on:
1. User's access rules (from `user_access_rules` table)
2. Group access rules (from `group_access_rules` table via user's IdP groups)
3. Gateway's full tunnel mode setting

For CIDR-type access rules, routes are automatically converted to OpenVPN push directives (e.g., `route 192.168.50.0 255.255.254.0`).

## Mesh Networking

Hub-and-spoke VPN topology for site-to-site connectivity.

### Key Features
- **Zero-Trust Access**: Routes only pushed for networks with explicit access rules
- **Hub Networks**: Assign global Networks to hubs via Manage Access → Networks tab
- **Full Tunnel Mode**: Route all client traffic through hub
- **Push DNS**: Push DNS servers to mesh clients
- **Local Networks**: Networks directly reachable from hub (not via spokes)

### Zero-Trust Model
1. Create Networks (Administration → Networks)
2. Create Access Rules within Networks (Administration → Access Rules)
3. Assign Access Rules to Users/Groups
4. Assign Networks to Hub (Mesh → Hubs → Manage Access → Networks)
5. Users only receive routes for networks they have access rules for

### CLI Commands
```bash
gatekey connect --mesh <hub-name>   # Connect to mesh hub
gatekey mesh list                   # List available mesh hubs
gatekey status                      # Show connection status
gatekey disconnect                  # Disconnect from VPN
```

## Admin Config Management

Centralized management of all VPN configurations across all users.

### Features
- **Gateway Configs**: View all gateway VPN configurations with user attribution
- **Mesh Configs**: View all mesh hub configurations with user attribution
- **Filtering**: Filter by user (email/name) and status (active/revoked/expired)
- **Revocation**: Revoke any active config with reason tracking
- **User Attribution**: Each config shows user email and name (from users or local_users table)

### Database Queries
Config listings join with both `users` (OIDC/SAML) and `local_users` tables to display:
- User email (from `users.email` or `local_users.email`)
- User name (from `users.name` or `local_users.username`)

### Web UI
Navigate to **Administration → All Configs** to:
- View all gateway configs (Gateway Configs tab)
- View all mesh configs (Mesh Configs tab)
- Filter by user or status
- Revoke active configs with reason
