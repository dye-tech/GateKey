# GateKey API Reference

## Overview

The GateKey API is a RESTful JSON API. All endpoints use HTTPS and require authentication unless otherwise noted.

Base URL: `https://gatekey.example.com/api/v1`

## Authentication

### Session-based Authentication

Most endpoints require a valid session cookie obtained through OIDC or SAML login.

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
  "needs_reprovision": false
}
```

When `needs_reprovision` is `true`, the gateway should call `/gateway/provision` to get updated configuration.

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
