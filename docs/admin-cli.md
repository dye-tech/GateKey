# GateKey Admin CLI

The GateKey Admin CLI (`gatekey-admin`) provides command-line administration for GateKey deployments. It enables full management of gateways, networks, access rules, users, API keys, mesh networking, certificates, and more.

## Installation

### From Binary

Download the appropriate binary for your platform:

```bash
# Linux (x64)
curl -LO https://your-server/bin/gatekey-admin-linux-amd64
chmod +x gatekey-admin-linux-amd64
sudo mv gatekey-admin-linux-amd64 /usr/local/bin/gatekey-admin

# macOS (Apple Silicon)
curl -LO https://your-server/bin/gatekey-admin-darwin-arm64
chmod +x gatekey-admin-darwin-arm64
sudo mv gatekey-admin-darwin-arm64 /usr/local/bin/gatekey-admin

# Verify installation
gatekey-admin version
```

### From Source

```bash
git clone https://github.com/gatekey-project/gatekey.git
cd gatekey
make build-admin
sudo mv bin/gatekey-admin /usr/local/bin/
```

## Quick Start

```bash
# 1. Initialize with your GateKey server URL
gatekey-admin config init --server https://vpn.company.com

# 2. Authenticate (opens browser for SSO)
gatekey-admin login

# Or authenticate with API key
gatekey-admin login --api-key gk_your_api_key_here

# 3. Start managing your deployment
gatekey-admin gateway list
gatekey-admin user list
gatekey-admin api-key list
```

## Authentication

The admin CLI supports two authentication methods:

### Browser-based SSO Login

```bash
gatekey-admin login
```

Opens your default browser to authenticate with your identity provider (Okta, Azure AD, Google, etc.). After successful authentication, the session token is saved locally.

Use `--no-browser` to print the login URL for headless systems:
```bash
gatekey-admin login --no-browser
```

### API Key Login

```bash
gatekey-admin login --api-key gk_your_api_key_here
```

Validates and stores the API key for subsequent commands. API keys are ideal for automation and CI/CD pipelines.

### Logout

```bash
gatekey-admin logout
```

Clears saved credentials (session token and API key).

## Global Flags

| Flag | Description |
|------|-------------|
| `--server string` | GateKey server URL (overrides config) |
| `--api-key string` | API key for authentication (per-command) |
| `--config string` | Config file path (default: ~/.gatekey-admin/config.yaml) |
| `-o, --output string` | Output format: table, json, yaml (default: table) |
| `-h, --help` | Help for the command |

## Configuration

### config init

Initialize configuration with a server URL:

```bash
gatekey-admin config init --server https://vpn.company.com
```

### config show

Display current configuration:

```bash
gatekey-admin config show
```

### config set

Set a configuration value:

```bash
gatekey-admin config set server https://new-server.com
gatekey-admin config set output json
```

## Gateway Management

### gateway list

List all VPN gateways:

```bash
gatekey-admin gateway list
gatekey-admin gateway list -o json
```

### gateway create

Create a new gateway:

```bash
gatekey-admin gateway create \
  --name "us-east-1" \
  --hostname "vpn-us-east.example.com" \
  --port 1194 \
  --protocol udp \
  --vpn-subnet "172.31.255.0/24"
```

**Options:**
| Flag | Description |
|------|-------------|
| `--name` | Gateway display name (required) |
| `--hostname` | Public hostname or IP (required) |
| `--port` | OpenVPN port (default: 1194) |
| `--protocol` | udp or tcp (default: udp) |
| `--vpn-subnet` | VPN client IP range (default: 172.31.255.0/24) |
| `--crypto-profile` | modern, fips, or compatible (default: modern) |
| `--full-tunnel` | Enable full tunnel mode |
| `--push-dns` | Push DNS servers to clients |
| `--dns-servers` | DNS servers to push (comma-separated) |
| `--tls-auth` | Enable TLS-Auth |

### gateway update

Update a gateway:

```bash
gatekey-admin gateway update <gateway-id> \
  --hostname "new-hostname.example.com" \
  --port 443
```

### gateway delete

Delete a gateway:

```bash
gatekey-admin gateway delete <gateway-id>
```

### gateway reprovision

Regenerate gateway certificates and configuration:

```bash
gatekey-admin gateway reprovision <gateway-id>
```

## Network Management

### network list

List all networks:

```bash
gatekey-admin network list
```

### network create

Create a new network:

```bash
gatekey-admin network create \
  --name "Production Servers" \
  --cidr "10.0.0.0/8" \
  --description "Production infrastructure"
```

### network update

Update a network:

```bash
gatekey-admin network update <network-id> \
  --name "New Name" \
  --description "Updated description"
```

### network delete

Delete a network:

```bash
gatekey-admin network delete <network-id>
```

## Access Rule Management

### access-rule list

List all access rules:

```bash
gatekey-admin access-rule list
```

### access-rule create

Create a new access rule:

```bash
# CIDR-based rule
gatekey-admin access-rule create \
  --name "Production Access" \
  --type cidr \
  --value "10.0.0.0/24" \
  --ports "443,80" \
  --protocol tcp

# Hostname-based rule
gatekey-admin access-rule create \
  --name "API Access" \
  --type hostname \
  --value "api.internal.com" \
  --ports "443"

# Wildcard rule
gatekey-admin access-rule create \
  --name "All Internal Sites" \
  --type wildcard \
  --value "*.internal.com"
```

**Rule Types:**
- `ip` - Single IP address
- `cidr` - Network range (also pushed as route)
- `hostname` - Exact hostname match
- `wildcard` - Pattern matching (*.example.com)

### access-rule update

Update an access rule:

```bash
gatekey-admin access-rule update <rule-id> \
  --ports "443,8443"
```

### access-rule delete

Delete an access rule:

```bash
gatekey-admin access-rule delete <rule-id>
```

## User Management

### user list

List all SSO users:

```bash
gatekey-admin user list
gatekey-admin user list -o json
```

### user get

Get details for a specific user:

```bash
gatekey-admin user get <user-id>
gatekey-admin user get --email user@example.com
```

### user revoke-configs

Revoke all VPN configurations for a user:

```bash
gatekey-admin user revoke-configs <user-id>
```

This invalidates all active VPN sessions for the user.

## Local User Management

Local users are for environments without SSO.

### local-user list

List all local users:

```bash
gatekey-admin local-user list
```

### local-user create

Create a local user:

```bash
gatekey-admin local-user create \
  --email admin@example.com \
  --name "Admin User" \
  --password "secure-password" \
  --admin
```

### local-user delete

Delete a local user:

```bash
gatekey-admin local-user delete <user-id>
```

## Group Management

### group list

List all groups (synced from IdP):

```bash
gatekey-admin group list
```

### group members

List members of a group:

```bash
gatekey-admin group members <group-name>
```

### group rules

List access rules assigned to a group:

```bash
gatekey-admin group rules <group-name>
```

## API Key Management

### api-key list

List all API keys:

```bash
gatekey-admin api-key list

# List keys for a specific user
gatekey-admin api-key list --user user@example.com
```

### api-key create

Create an API key:

```bash
# Create for yourself
gatekey-admin api-key create "My CLI Key"

# Create for another user
gatekey-admin api-key create "Service Key" --user user@example.com

# With expiration
gatekey-admin api-key create "Temp Key" --expires 30d

# With limited scopes
gatekey-admin api-key create "Read Only" --scopes read:gateways,read:networks
```

**Options:**
| Flag | Description |
|------|-------------|
| `--user` | Create key for this user (admin only) |
| `--expires` | Expiration time: 30d, 90d, 1y, never |
| `--scopes` | Comma-separated list of scopes |
| `--description` | Optional description |

**Important:** The raw API key is only shown once at creation. Save it securely!

### api-key revoke

Revoke an API key:

```bash
gatekey-admin api-key revoke <key-id>
```

### api-key revoke-all

Revoke all API keys for a user:

```bash
gatekey-admin api-key revoke-all --user user@example.com
```

## Mesh VPN Management

### mesh hub list

List all mesh hubs:

```bash
gatekey-admin mesh hub list
```

### mesh hub create

Create a mesh hub:

```bash
gatekey-admin mesh hub create \
  --name "primary-hub" \
  --endpoint "hub.example.com" \
  --port 1194 \
  --protocol udp \
  --vpn-subnet "172.30.0.0/16"
```

### mesh hub update

Update a mesh hub:

```bash
gatekey-admin mesh hub update <hub-id> \
  --endpoint "new-hub.example.com"
```

### mesh hub delete

Delete a mesh hub:

```bash
gatekey-admin mesh hub delete <hub-id>
```

### mesh hub provision

Generate install script for a hub:

```bash
gatekey-admin mesh hub provision <hub-id>
```

### mesh spoke list

List all mesh spokes:

```bash
gatekey-admin mesh spoke list
gatekey-admin mesh spoke list --hub <hub-id>
```

### mesh spoke create

Create a mesh spoke:

```bash
gatekey-admin mesh spoke create \
  --hub <hub-id> \
  --name "home-lab" \
  --networks "10.0.0.0/8,192.168.1.0/24"
```

### mesh spoke update

Update a mesh spoke:

```bash
gatekey-admin mesh spoke update <spoke-id> \
  --networks "10.0.0.0/8"
```

### mesh spoke delete

Delete a mesh spoke:

```bash
gatekey-admin mesh spoke delete <spoke-id>
```

### mesh spoke provision

Generate install script for a spoke:

```bash
gatekey-admin mesh spoke provision <spoke-id>
```

## Certificate Authority Management

### ca show

Display CA information:

```bash
gatekey-admin ca show
```

Shows certificate details, expiration, and fingerprint.

### ca rotate

Rotate the CA certificate:

```bash
gatekey-admin ca rotate
```

**Warning:** This invalidates all existing certificates. All gateways will need to be reprovisioned.

### ca list

List all CA certificates (active and archived):

```bash
gatekey-admin ca list
```

### ca activate

Activate a specific CA certificate:

```bash
gatekey-admin ca activate <ca-id>
```

### ca revoke

Revoke a CA certificate:

```bash
gatekey-admin ca revoke <ca-id>
```

## Audit Log Management

### audit list

View audit logs:

```bash
# Recent events
gatekey-admin audit list

# Filter by action
gatekey-admin audit list --action login
gatekey-admin audit list --action api_key_created

# Filter by user
gatekey-admin audit list --user user@example.com

# Date range
gatekey-admin audit list --since 2024-01-01 --until 2024-01-31

# Combine filters
gatekey-admin audit list --action vpn_connect --since 2024-01-01 -o json
```

## Connection Management

### connection list

List active VPN connections:

```bash
gatekey-admin connection list
gatekey-admin connection list --gateway <gateway-id>
```

### connection disconnect

Disconnect a user:

```bash
gatekey-admin connection disconnect <connection-id>

# Disconnect all connections for a user
gatekey-admin connection disconnect --user user@example.com
```

## Version Information

```bash
gatekey-admin version
```

Shows version, commit hash, and build time.

## Output Formats

The CLI supports multiple output formats:

```bash
# Table format (default)
gatekey-admin gateway list

# JSON format
gatekey-admin gateway list -o json

# YAML format
gatekey-admin gateway list -o yaml
```

JSON and YAML outputs are useful for scripting and automation.

## Configuration File

Configuration is stored in `~/.gatekey-admin/config.yaml`:

```yaml
server_url: https://vpn.company.com
api_key: gk_your_api_key_here
output: table
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GATEKEY_SERVER` | Default server URL |
| `GATEKEY_API_KEY` | API key for authentication |
| `GATEKEY_CONFIG` | Config file path |

## Scripting Examples

### Create Gateway with Networks

```bash
#!/bin/bash

# Create gateway
GATEWAY_ID=$(gatekey-admin gateway create \
  --name "new-gateway" \
  --hostname "gw.example.com" \
  -o json | jq -r '.id')

# Create network
NETWORK_ID=$(gatekey-admin network create \
  --name "Production" \
  --cidr "10.0.0.0/8" \
  -o json | jq -r '.id')

echo "Gateway: $GATEWAY_ID"
echo "Network: $NETWORK_ID"
```

### Bulk User Management

```bash
#!/bin/bash

# Revoke all configs for deactivated users
gatekey-admin user list -o json | \
  jq -r '.users[] | select(.is_active == false) | .id' | \
  while read user_id; do
    gatekey-admin user revoke-configs "$user_id"
    gatekey-admin api-key revoke-all --user "$user_id"
  done
```

### Export Configuration

```bash
#!/bin/bash

# Export all configuration to JSON
mkdir -p backup

gatekey-admin gateway list -o json > backup/gateways.json
gatekey-admin network list -o json > backup/networks.json
gatekey-admin access-rule list -o json > backup/access-rules.json
gatekey-admin mesh hub list -o json > backup/mesh-hubs.json
gatekey-admin mesh spoke list -o json > backup/mesh-spokes.json
```

## See Also

- [API Keys Guide](api-keys.md) - API key management
- [Client Guide](client.md) - End-user VPN client
- [Mesh Networking](mesh-networking.md) - Hub-and-spoke topology
- [API Reference](api.md) - REST API documentation
- [Security](security.md) - Security best practices
