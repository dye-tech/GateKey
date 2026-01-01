# GateKey API Reference

## Overview

The GateKey API is a RESTful JSON API. All endpoints use HTTPS and require authentication unless otherwise noted.

Base URL: `https://gatekey.example.com/api/v1`

## Authentication

### Session-based Authentication

Most endpoints require a valid session cookie obtained through OIDC or SAML login.

### API Key Authentication

API keys provide programmatic access without browser-based SSO. Use API keys for CLI tools, automation, and CI/CD pipelines.

**Header Format:**
```
Authorization: Bearer gk_<base64-encoded-key>
```

**Example:**
```bash
curl -H "Authorization: Bearer gk_dGhpcyBpcyBhIHNhbXBsZQ..." \
  https://vpn.example.com/api/v1/gateways
```

**Key Format:**
- Prefix: `gk_` (identifies GateKey API keys)
- Body: Base64-encoded 32 random bytes
- Example: `gk_dGhpcyBpcyBhIHNhbXBsZSBhcGkga2V5...`

API keys inherit the permissions of their owner. Admin-provisioned keys can be scoped to limit access.

### Gateway Authentication

Gateway endpoints require the `X-Gateway-Token` header:

```
X-Gateway-Token: <gateway-token>
```

## Endpoints

### Health Check

#### GET /health

Check server health. No authentication required.

**Response:**
```json
{
  "status": "healthy",
  "time": "2024-01-15T10:30:00Z"
}
```

#### GET /ready

Check server readiness. No authentication required.

**Response:**
```json
{
  "status": "ready",
  "time": "2024-01-15T10:30:00Z"
}
```

---

### Authentication

#### GET /auth/providers

List available authentication providers. No authentication required.

**Response:**
```json
{
  "providers": [
    {
      "type": "oidc",
      "name": "default",
      "display_name": "Login with SSO",
      "login_url": "/api/v1/auth/oidc/login?provider=default"
    }
  ]
}
```

#### GET /auth/oidc/login

Initiate OIDC login flow.

**Query Parameters:**
- `provider` (optional): Provider name, defaults to "default"

**Response:** Redirect to IdP

#### GET /auth/oidc/callback

OIDC callback endpoint. Handled automatically.

#### GET /auth/saml/login

Initiate SAML login flow.

**Query Parameters:**
- `provider` (optional): Provider name

**Response:** Redirect to IdP

#### POST /auth/saml/acs

SAML Assertion Consumer Service. Handled automatically.

#### GET /auth/saml/metadata

Get SAML Service Provider metadata.

**Response:** XML metadata

#### GET /auth/session

Get current session information.

**Response:**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "groups": ["engineering", "vpn-users"],
    "is_admin": false
  },
  "session": {
    "id": "session-id",
    "expires_at": "2024-01-16T10:30:00Z"
  }
}
```

#### POST /auth/logout

Log out and invalidate session.

**Response:**
```json
{
  "success": true
}
```

---

### VPN Configurations

#### POST /configs/generate

Generate a new OpenVPN configuration.

**Request:**
```json
{
  "gateway_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**
```json
{
  "id": "config-id",
  "file_name": "gatekey-gateway-1-20240115.ovpn",
  "expires_at": "2024-01-16T10:30:00Z",
  "download_url": "/api/v1/configs/download/config-id"
}
```

#### GET /configs/download/:id

Download a generated configuration file.

**Response:** `.ovpn` file download

---

### Certificates

#### GET /certs/ca

Get the CA certificate.

**Response:** PEM-encoded certificate

#### POST /certs/revoke

Revoke a certificate (admin only).

**Request:**
```json
{
  "certificate_id": "550e8400-e29b-41d4-a716-446655440000",
  "reason": "key_compromise"
}
```

**Response:**
```json
{
  "success": true
}
```

---

### CA Management (Admin)

GateKey supports graceful CA rotation with zero-downtime. The rotation process uses a dual-trust period where both old and new CAs are trusted simultaneously.

#### GET /settings/ca/list

List all CAs in the system.

**Response:**
```json
{
  "cas": [
    {
      "id": "default",
      "status": "active",
      "fingerprint": "sha256:abc123...",
      "notBefore": "2024-12-31T00:00:00Z",
      "notAfter": "2026-12-30T23:59:59Z",
      "description": "Primary CA",
      "createdAt": "2024-12-31T00:00:00Z"
    },
    {
      "id": "ca-2025-01",
      "status": "pending",
      "fingerprint": "sha256:def456...",
      "notBefore": "2025-01-01T00:00:00Z",
      "notAfter": "2027-01-01T23:59:59Z",
      "description": "Prepared for rotation",
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ]
}
```

**CA Status Values:**
- `active`: Currently issuing certificates
- `pending`: Generated but not yet activated (for rotation)
- `retired`: No longer issuing, but still trusted for verification
- `revoked`: Revoked, no longer trusted

#### POST /settings/ca/prepare-rotation

Prepare a new CA for rotation. This generates a new CA in `pending` status.

**Request:**
```json
{
  "description": "Q1 2025 CA rotation"
}
```

**Response:**
```json
{
  "id": "ca-2025-01",
  "status": "pending",
  "fingerprint": "sha256:def456...",
  "notBefore": "2025-01-01T00:00:00Z",
  "notAfter": "2027-01-01T23:59:59Z",
  "message": "New CA prepared. Activate when ready."
}
```

#### POST /settings/ca/activate/:id

Activate a pending CA. This:
1. Retires the current active CA (still trusted for verification)
2. Activates the new CA (starts issuing new certificates)
3. Records the rotation event for audit

**Response:**
```json
{
  "message": "CA activated successfully",
  "previousCA": "default",
  "newCA": "ca-2025-01"
}
```

After activation, gateways and mesh components will detect the CA change via fingerprint comparison in heartbeat responses and can auto-reprovision.

#### POST /settings/ca/revoke/:id

Revoke a CA. Revoked CAs are no longer trusted for any purpose.

**Response:**
```json
{
  "message": "CA revoked successfully"
}
```

#### GET /settings/ca/fingerprint

Get the fingerprint of the currently active CA.

**Response:**
```json
{
  "fingerprint": "sha256:abc123..."
}
```

---

### Policies (Admin)

#### GET /policies

List all policies.

**Response:**
```json
{
  "policies": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "engineering-access",
      "description": "Access for engineering team",
      "priority": 10,
      "is_enabled": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### POST /policies

Create a new policy.

**Request:**
```json
{
  "name": "engineering-access",
  "description": "Access for engineering team",
  "priority": 10,
  "is_enabled": true,
  "rules": [
    {
      "action": "allow",
      "subject": {
        "groups": ["engineering"]
      },
      "resource": {
        "gateways": ["prod-gateway"],
        "networks": ["10.0.0.0/8"]
      }
    }
  ]
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "engineering-access",
  ...
}
```

#### GET /policies/:id

Get a specific policy.

#### PUT /policies/:id

Update a policy.

#### DELETE /policies/:id

Delete a policy.

---

### Gateway (Internal API)

These endpoints are used by gateway agents.

#### POST /gateway/verify

Verify a client connection.

**Headers:**
- `X-Gateway-Token`: Gateway authentication token

**Request:**
```json
{
  "type": "auth-user-pass-verify",
  "common_name": "user@example.com",
  "trusted_ip": "10.8.0.5",
  "untrusted_ip": "203.0.113.50",
  "tls_serial": "1234567890abcdef",
  "tls_fingerprint": "sha256:..."
}
```

**Response:**
```json
{
  "allow": true,
  "message": "Access granted",
  "client_config": [
    "push \"route 10.0.0.0 255.0.0.0\""
  ]
}
```

#### POST /gateway/connect

Report client connection.

**Request:**
```json
{
  "type": "client-connect",
  "common_name": "user@example.com",
  "trusted_ip": "10.8.0.5",
  "untrusted_ip": "203.0.113.50",
  "ifconfig_local": "10.8.0.5"
}
```

**Response:**
```json
{
  "allow": true,
  "client_config": [
    "push \"route 10.0.0.0 255.0.0.0\""
  ]
}
```

#### POST /gateway/disconnect

Report client disconnection.

**Request:**
```json
{
  "type": "client-disconnect",
  "common_name": "user@example.com",
  "trusted_ip": "10.8.0.5",
  "bytes_received": 123456,
  "bytes_sent": 654321,
  "time_connected": 3600
}
```

#### POST /gateway/heartbeat

Gateway heartbeat. Reports status and receives configuration update signals.

**Request:**
```json
{
  "token": "gateway-auth-token",
  "public_ip": "203.0.113.1",
  "active_clients": 5,
  "openvpn_running": true,
  "config_version": "sha256-hash-of-current-config"
}
```

**Response:**
```json
{
  "status": "ok",
  "gateway_id": "gateway-uuid",
  "gateway_name": "prod-gateway",
  "config_version": "sha256-hash-from-server",
  "needs_reprovision": false,
  "ca_fingerprint": "sha256:abc123..."
}
```

When `needs_reprovision` is `true`, the gateway should call `/gateway/provision` to get updated configuration.

The `ca_fingerprint` field contains the SHA256 fingerprint of the currently active CA certificate. Gateways can compare this with their local CA fingerprint to detect CA rotation and trigger reprovisioning.

#### POST /gateway/provision

Provision or reprovision gateway certificates and configuration.

**Request:**
```json
{
  "token": "gateway-auth-token"
}
```

**Response:**
```json
{
  "gateway_id": "gateway-uuid",
  "gateway_name": "prod-gateway",
  "ca_cert": "-----BEGIN CERTIFICATE-----...",
  "server_cert": "-----BEGIN CERTIFICATE-----...",
  "server_key": "-----BEGIN PRIVATE KEY-----...",
  "vpn_subnet": "172.31.255.0/24",
  "vpn_network": "172.31.255.0",
  "vpn_netmask": "255.255.255.0",
  "vpn_port": 1194,
  "vpn_protocol": "udp",
  "crypto_profile": "modern",
  "tls_auth_enabled": true,
  "tls_auth_key": "-----BEGIN OpenVPN Static key V1-----..."
}
```

The `tls_auth_key` is only included when `tls_auth_enabled` is `true`.

---

### Users

#### GET /users/me

Get current user information.

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "groups": ["engineering"],
  "is_admin": false,
  "last_login_at": "2024-01-15T10:30:00Z"
}
```

#### GET /users/me/connections

Get current user's connections.

**Response:**
```json
{
  "connections": [
    {
      "id": "connection-id",
      "gateway_name": "prod-gateway",
      "connected_at": "2024-01-15T10:00:00Z",
      "vpn_ip": "10.8.0.5"
    }
  ]
}
```

---

### API Keys

#### GET /api-keys

List current user's API keys.

**Response:**
```json
{
  "api_keys": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "CI/CD Pipeline",
      "key_prefix": "gk_dGhpcyBp...",
      "scopes": ["*"],
      "expires_at": "2024-07-15T00:00:00Z",
      "last_used_at": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### POST /api-keys

Create a new API key. The full key is only returned once.

**Request:**
```json
{
  "name": "CI/CD Pipeline",
  "description": "For automated deployments",
  "expires_in_days": 90
}
```

**Response:**
```json
{
  "api_key": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "CI/CD Pipeline",
    "key": "gk_dGhpcyBpcyBhIHNhbXBsZSBhcGkga2V5...",
    "key_prefix": "gk_dGhpcyBp...",
    "expires_at": "2024-04-15T00:00:00Z"
  },
  "message": "Save this key now - it will not be shown again"
}
```

#### DELETE /api-keys/:id

Revoke an API key.

**Response:**
```json
{
  "success": true
}
```

#### GET /auth/api-key/validate

Validate the current API key and return user information.

**Response:**
```json
{
  "valid": true,
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "is_admin": false,
  "scopes": ["*"],
  "expires_at": "2024-07-15T00:00:00Z"
}
```

---

### Admin API Keys

#### GET /admin/api-keys

List all API keys (admin only).

#### POST /admin/api-keys

Create an API key for any user (admin only).

**Request:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Provisioned Key",
  "scopes": ["read:gateways", "vpn:connect"],
  "expires_in_days": 30
}
```

#### DELETE /admin/api-keys/:id

Revoke any API key (admin only).

#### GET /admin/users/:id/api-keys

List a specific user's API keys (admin only).

#### POST /admin/users/:id/api-keys

Create an API key for a specific user (admin only).

#### DELETE /admin/users/:id/api-keys

Revoke all API keys for a user (admin only).

---

### Admin

#### GET /admin/gateways

List all gateways.

**Response:**
```json
{
  "gateways": [
    {
      "id": "gateway-id",
      "name": "prod-gateway",
      "hostname": "vpn.example.com",
      "publicIp": "203.0.113.1",
      "vpnPort": 1194,
      "vpnProtocol": "udp",
      "cryptoProfile": "modern",
      "vpnSubnet": "172.31.255.0/24",
      "tlsAuthEnabled": true,
      "fullTunnelMode": false,
      "pushDns": false,
      "dnsServers": [],
      "isActive": true,
      "lastHeartbeat": "2024-01-15T10:30:00Z",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### POST /admin/gateways

Register a new gateway.

**Request:**
```json
{
  "name": "prod-gateway",
  "hostname": "vpn.example.com",
  "public_ip": "203.0.113.1",
  "vpn_port": 1194,
  "vpn_protocol": "udp",
  "crypto_profile": "modern",
  "vpn_subnet": "172.31.255.0/24",
  "tls_auth_enabled": true,
  "full_tunnel_mode": false,
  "push_dns": false,
  "dns_servers": ["1.1.1.1", "8.8.8.8"]
}
```

**Response:**
```json
{
  "id": "gateway-id",
  "name": "prod-gateway",
  "hostname": "vpn.example.com",
  "vpnPort": 1194,
  "vpnProtocol": "udp",
  "cryptoProfile": "modern",
  "tlsAuthEnabled": true,
  "token": "gateway-auth-token",
  "message": "Gateway registered successfully. Save the token - it will not be shown again."
}
```

#### PUT /admin/gateways/:id

Update a gateway.

**Request:**
```json
{
  "name": "prod-gateway",
  "hostname": "vpn.example.com",
  "public_ip": "203.0.113.1",
  "vpn_port": 1194,
  "vpn_protocol": "udp",
  "crypto_profile": "fips",
  "vpn_subnet": "172.31.255.0/24",
  "tls_auth_enabled": true,
  "full_tunnel_mode": false,
  "push_dns": true,
  "dns_servers": ["1.1.1.1", "8.8.8.8"]
}
```

Changing `crypto_profile`, `vpn_port`, `vpn_protocol`, `vpn_subnet`, `tls_auth_enabled`, `full_tunnel_mode`, `push_dns`, or `dns_servers` will update the gateway's `config_version`, triggering automatic reprovisioning on the next heartbeat.

#### DELETE /admin/gateways/:id

Delete a gateway.

#### GET /admin/connections

List all active connections.

**Query Parameters:**
- `gateway_id` (optional): Filter by gateway
- `user_id` (optional): Filter by user

#### GET /admin/audit

Get audit logs.

**Query Parameters:**
- `event` (optional): Filter by event type
- `limit` (optional): Number of records (default: 50)
- `offset` (optional): Pagination offset

**Response:**
```json
{
  "logs": [
    {
      "id": "log-id",
      "timestamp": "2024-01-15T10:30:00Z",
      "event": "auth.login",
      "actor_email": "user@example.com",
      "actor_ip": "203.0.113.50",
      "resource_type": "session",
      "success": true
    }
  ],
  "total": 100
}
```

---

### Mesh Networking (Admin)

Manage mesh hubs and spokes for site-to-site VPN connectivity.

#### GET /admin/mesh/hubs

List all mesh hubs.

**Response:**
```json
{
  "hubs": [
    {
      "id": "hub-uuid",
      "name": "primary-hub",
      "publicEndpoint": "hub.example.com",
      "vpnPort": 1194,
      "vpnProtocol": "udp",
      "vpnSubnet": "172.30.0.0/16",
      "cryptoProfile": "modern",
      "tlsAuthEnabled": true,
      "fullTunnelMode": false,
      "pushDns": true,
      "dnsServers": ["1.1.1.1", "8.8.8.8"],
      "localNetworks": ["192.168.1.0/24"],
      "status": "online",
      "lastHeartbeat": "2024-01-15T10:30:00Z",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### POST /admin/mesh/hubs

Create a new mesh hub.

**Request:**
```json
{
  "name": "primary-hub",
  "publicEndpoint": "hub.example.com",
  "vpnPort": 1194,
  "vpnProtocol": "udp",
  "vpnSubnet": "172.30.0.0/16",
  "cryptoProfile": "modern",
  "tlsAuthEnabled": true,
  "fullTunnelMode": false,
  "pushDns": true,
  "dnsServers": ["1.1.1.1", "8.8.8.8"],
  "localNetworks": ["192.168.1.0/24"]
}
```

**Response:**
```json
{
  "id": "hub-uuid",
  "name": "primary-hub",
  "token": "hub-auth-token",
  "message": "Hub created successfully. Save the token - it will not be shown again."
}
```

#### GET /admin/mesh/hubs/:id

Get a specific mesh hub.

#### PUT /admin/mesh/hubs/:id

Update a mesh hub.

#### DELETE /admin/mesh/hubs/:id

Delete a mesh hub.

#### POST /admin/mesh/hubs/:id/provision

Trigger hub reprovisioning.

#### GET /admin/mesh/hubs/:id/install-script

Get the hub installation script.

**Response:** Shell script for installing the hub

#### Hub Access Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/mesh/hubs/:id/users` | GET | List users assigned to hub |
| `/admin/mesh/hubs/:id/users` | POST | Add user to hub |
| `/admin/mesh/hubs/:id/users/:userId` | DELETE | Remove user from hub |
| `/admin/mesh/hubs/:id/groups` | GET | List groups assigned to hub |
| `/admin/mesh/hubs/:id/groups` | POST | Add group to hub |
| `/admin/mesh/hubs/:id/groups/:groupName` | DELETE | Remove group from hub |
| `/admin/mesh/hubs/:id/networks` | GET | List networks assigned to hub |
| `/admin/mesh/hubs/:id/networks` | POST | Assign network to hub |
| `/admin/mesh/hubs/:id/networks/:networkId` | DELETE | Remove network from hub |

#### Spoke Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/mesh/hubs/:id/spokes` | GET | List spokes for hub |
| `/admin/mesh/hubs/:id/spokes` | POST | Create spoke |
| `/admin/mesh/spokes/:id` | GET | Get spoke details |
| `/admin/mesh/spokes/:id` | PUT | Update spoke |
| `/admin/mesh/spokes/:id` | DELETE | Delete spoke |
| `/admin/mesh/spokes/:id/provision` | POST | Trigger spoke provision |
| `/admin/mesh/spokes/:id/install-script` | GET | Get spoke install script |

#### Spoke Access Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/mesh/spokes/:id/users` | GET | List users assigned to spoke |
| `/admin/mesh/spokes/:id/users` | POST | Add user to spoke |
| `/admin/mesh/spokes/:id/users/:userId` | DELETE | Remove user from spoke |
| `/admin/mesh/spokes/:id/groups` | GET | List groups assigned to spoke |
| `/admin/mesh/spokes/:id/groups` | POST | Add group to spoke |
| `/admin/mesh/spokes/:id/groups/:groupName` | DELETE | Remove group from spoke |

---

### Mesh User Access

Endpoints for users to access mesh networks.

#### GET /mesh/hubs

List mesh hubs the current user has access to.

**Response:**
```json
{
  "hubs": [
    {
      "id": "hub-uuid",
      "name": "primary-hub",
      "publicEndpoint": "hub.example.com",
      "status": "online",
      "networks": ["10.0.0.0/8", "192.168.0.0/16"]
    }
  ]
}
```

#### POST /mesh/generate-config

Generate a VPN configuration for connecting to a mesh hub.

**Request:**
```json
{
  "hub_id": "hub-uuid"
}
```

**Response:**
```json
{
  "config_id": "config-uuid",
  "file_name": "mesh-primary-hub.ovpn",
  "expires_at": "2024-01-16T10:30:00Z",
  "download_url": "/api/v1/mesh/download/config-uuid"
}
```

---

### Mesh Hub Internal API

These endpoints are used by mesh hub agents.

#### POST /mesh/hub/heartbeat

Hub heartbeat. Reports status and receives configuration updates.

**Headers:**
- `X-Hub-Token`: Hub authentication token

**Request:**
```json
{
  "token": "hub-auth-token",
  "public_ip": "203.0.113.1",
  "connected_spokes": 5,
  "connected_clients": 10,
  "openvpn_running": true,
  "config_version": "sha256-hash"
}
```

**Response:**
```json
{
  "status": "ok",
  "hub_id": "hub-uuid",
  "config_version": "sha256-hash-from-server",
  "needs_reprovision": false,
  "ca_fingerprint": "sha256:abc123..."
}
```

#### POST /mesh/hub/provision

Provision or reprovision hub certificates and configuration.

**Response:**
```json
{
  "hub_id": "hub-uuid",
  "ca_cert": "-----BEGIN CERTIFICATE-----...",
  "server_cert": "-----BEGIN CERTIFICATE-----...",
  "server_key": "-----BEGIN PRIVATE KEY-----...",
  "vpn_subnet": "172.30.0.0/16",
  "vpn_port": 1194,
  "vpn_protocol": "udp",
  "crypto_profile": "modern",
  "tls_auth_key": "-----BEGIN OpenVPN Static key V1-----...",
  "spokes": [
    {
      "id": "spoke-uuid",
      "name": "home-lab",
      "local_networks": ["10.0.0.0/8"]
    }
  ]
}
```

---

### Mesh Spoke Internal API

These endpoints are used by mesh spoke agents.

#### POST /mesh/spoke/heartbeat

Spoke heartbeat.

**Headers:**
- `X-Spoke-Token`: Spoke authentication token

**Request:**
```json
{
  "token": "spoke-auth-token",
  "connected": true,
  "hub_ip": "172.30.0.1"
}
```

#### POST /mesh/spoke/provision

Provision spoke certificates and configuration.

**Response:**
```json
{
  "spoke_id": "spoke-uuid",
  "hub_endpoint": "hub.example.com:1194",
  "ca_cert": "-----BEGIN CERTIFICATE-----...",
  "client_cert": "-----BEGIN CERTIFICATE-----...",
  "client_key": "-----BEGIN PRIVATE KEY-----...",
  "local_networks": ["10.0.0.0/8"],
  "tls_auth_key": "-----BEGIN OpenVPN Static key V1-----..."
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### HTTP Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error
