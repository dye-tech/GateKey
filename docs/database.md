# GateKey Database Schema

This document describes the PostgreSQL database schema used by GateKey.

## Overview

GateKey uses PostgreSQL 16+ with the following extensions:
- `pgcrypto` - Cryptographic functions
- `uuid-ossp` - UUID generation

## Table Categories

| Category | Tables |
|----------|--------|
| Authentication | `users`, `local_users`, `sessions`, `admin_sessions`, `sso_sessions`, `oauth_states` |
| Identity Providers | `oidc_providers`, `saml_providers` |
| VPN Infrastructure | `gateways`, `networks`, `gateway_networks` |
| Access Control | `access_rules`, `user_access_rules`, `group_access_rules`, `user_gateways`, `group_gateways` |
| Certificates & Configs | `pki_ca`, `certificates`, `configs`, `generated_configs` |
| Connections | `connections` |
| Web Proxy | `proxy_applications`, `user_proxy_applications`, `group_proxy_applications`, `proxy_access_logs` |
| Policy Engine | `policies`, `policy_rules` |
| System | `system_settings`, `audit_logs` |

---

## Authentication Tables

### users

SSO users synchronized from identity providers.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `external_id` | VARCHAR(255) | ID from the identity provider |
| `provider` | VARCHAR(100) | Provider name (e.g., "google", "okta") |
| `email` | VARCHAR(255) | User's email address (unique) |
| `name` | VARCHAR(255) | Display name |
| `groups` | JSONB | Array of group names from IdP |
| `attributes` | JSONB | Additional attributes from IdP |
| `is_admin` | BOOLEAN | Whether user has admin privileges |
| `is_active` | BOOLEAN | Whether user account is active |
| `last_login_at` | TIMESTAMPTZ | Last login timestamp |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

**Unique Constraints:** `(provider, external_id)`, `email`

### local_users

Local admin accounts (not from SSO).

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `username` | VARCHAR(100) | Login username (unique) |
| `password_hash` | TEXT | Bcrypt password hash |
| `email` | VARCHAR(255) | Email address |
| `is_admin` | BOOLEAN | Admin flag (always true for local users) |
| `last_login_at` | TIMESTAMPTZ | Last login timestamp |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### sessions

User sessions for the web UI.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | References `users.id` |
| `token` | VARCHAR(64) | Session token (unique) |
| `ip_address` | INET | Client IP address |
| `user_agent` | TEXT | Browser user agent |
| `expires_at` | TIMESTAMPTZ | Session expiration time |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `revoked_at` | TIMESTAMPTZ | Revocation timestamp (if revoked) |

### admin_sessions

Sessions for local admin users.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | References `local_users.id` |
| `token` | VARCHAR(64) | Session token (unique) |
| `ip_address` | INET | Client IP address |
| `user_agent` | TEXT | Browser user agent |
| `expires_at` | TIMESTAMPTZ | Session expiration time |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

### sso_sessions

Lightweight SSO session cache for quick lookups.

| Column | Type | Description |
|--------|------|-------------|
| `token` | VARCHAR(255) | Primary key, session token |
| `user_id` | VARCHAR(255) | User identifier |
| `username` | VARCHAR(255) | Username |
| `email` | VARCHAR(255) | Email address |
| `name` | VARCHAR(255) | Display name |
| `groups` | JSONB | User's groups |
| `provider` | VARCHAR(255) | Identity provider name |
| `is_admin` | BOOLEAN | Admin flag |
| `expires_at` | TIMESTAMPTZ | Expiration time |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

### oauth_states

Temporary state storage for OAuth/OIDC/SAML flows.

| Column | Type | Description |
|--------|------|-------------|
| `state` | VARCHAR(255) | Primary key, OAuth state parameter |
| `provider` | VARCHAR(255) | Provider name |
| `provider_type` | VARCHAR(50) | "oidc" or "saml" |
| `nonce` | VARCHAR(255) | OIDC nonce for replay protection |
| `relay_state` | VARCHAR(255) | SAML relay state |
| `cli_callback_url` | TEXT | Callback URL for CLI authentication |
| `expires_at` | TIMESTAMPTZ | State expiration time |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

---

## Identity Provider Tables

### oidc_providers

OIDC identity provider configurations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(100) | Provider identifier (unique) |
| `display_name` | VARCHAR(255) | Display name for UI |
| `issuer` | TEXT | OIDC issuer URL |
| `client_id` | VARCHAR(255) | OAuth client ID |
| `client_secret` | TEXT | OAuth client secret (encrypted) |
| `redirect_url` | TEXT | OAuth redirect URL |
| `scopes` | JSONB | OAuth scopes array |
| `admin_group` | VARCHAR(255) | Group name that grants admin access |
| `is_enabled` | BOOLEAN | Whether provider is enabled |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### saml_providers

SAML identity provider configurations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(100) | Provider identifier (unique) |
| `display_name` | VARCHAR(255) | Display name for UI |
| `idp_metadata_url` | TEXT | IdP metadata URL |
| `entity_id` | TEXT | Service Provider entity ID |
| `acs_url` | TEXT | Assertion Consumer Service URL |
| `admin_group` | VARCHAR(255) | Group name that grants admin access |
| `is_enabled` | BOOLEAN | Whether provider is enabled |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

---

## VPN Infrastructure Tables

### gateways

VPN gateway servers.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(100) | Gateway name (unique) |
| `hostname` | VARCHAR(255) | Public hostname |
| `public_ip` | INET | Public IP address |
| `vpn_port` | INTEGER | OpenVPN port (default: 1194) |
| `vpn_protocol` | VARCHAR(10) | "udp" or "tcp" |
| `vpn_subnet` | CIDR | VPN client subnet (default: 172.31.255.0/24) |
| `tls_auth_enabled` | BOOLEAN | Enable TLS-Auth for additional security (default: true) |
| `tls_auth_key` | TEXT | TLS-Auth static key (generated during provisioning) |
| `full_tunnel_mode` | BOOLEAN | Route all traffic through VPN (default: false) |
| `push_dns` | BOOLEAN | Push DNS servers to clients (default: false) |
| `dns_servers` | TEXT[] | Array of DNS server IPs to push |
| `config_version` | VARCHAR(64) | SHA256 hash of config settings (auto-computed by trigger) |
| `token` | VARCHAR(64) | Gateway authentication token |
| `public_key` | TEXT | Gateway's public key |
| `config` | JSONB | Additional configuration |
| `crypto_profile` | VARCHAR(50) | "modern", "fips", or "compatible" |
| `is_active` | BOOLEAN | Whether gateway is active |
| `last_heartbeat` | TIMESTAMPTZ | Last heartbeat from gateway |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

**Constraint:** At least one of `hostname` or `public_ip` must be set.

**Config Version:** The `config_version` is automatically computed by a database trigger whenever gateway settings change (crypto_profile, vpn_port, vpn_protocol, vpn_subnet, tls_auth_enabled, tls_auth_key, full_tunnel_mode, push_dns, dns_servers). This enables push-based configuration updates - when the gateway's version doesn't match the server's, it triggers automatic reprovisioning.

**Tunnel Modes:**
- `full_tunnel_mode = false` (default): Split tunnel - only routes for user's access rules are pushed
- `full_tunnel_mode = true`: Full tunnel - all traffic routed through VPN (0.0.0.0/0)

**DNS Settings:**
- `push_dns = false` (default): Client uses their own DNS
- `push_dns = true`: Push DNS servers to clients
- `dns_servers`: Array of DNS IPs (defaults to 1.1.1.1, 8.8.8.8 if empty and push_dns is true)

### networks

Network CIDR blocks that gateways can route to.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(255) | Network name (unique) |
| `description` | TEXT | Description |
| `cidr` | CIDR | Network CIDR block |
| `is_active` | BOOLEAN | Whether network is active |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### gateway_networks

Many-to-many relationship between gateways and networks.

| Column | Type | Description |
|--------|------|-------------|
| `gateway_id` | UUID | References `gateways.id` |
| `network_id` | UUID | References `networks.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(gateway_id, network_id)`

---

## Access Control Tables

### access_rules

Firewall rules defining what resources users can access via VPN.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(255) | Rule name |
| `description` | TEXT | Description |
| `rule_type` | VARCHAR(50) | "ip", "cidr", "hostname", or "hostname_wildcard" |
| `value` | VARCHAR(512) | IP, CIDR, or hostname pattern |
| `port_range` | VARCHAR(50) | Port range (e.g., "80", "8080-8090") |
| `protocol` | VARCHAR(20) | "tcp", "udp", or "any" |
| `network_id` | UUID | Optional reference to `networks.id` |
| `is_active` | BOOLEAN | Whether rule is active |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

**Valid rule_type values:** `ip`, `cidr`, `hostname`, `hostname_wildcard`

### user_access_rules

Assigns access rules directly to users.

| Column | Type | Description |
|--------|------|-------------|
| `user_id` | UUID | References `users.id` |
| `access_rule_id` | UUID | References `access_rules.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(user_id, access_rule_id)`

### group_access_rules

Assigns access rules to groups (from IdP).

| Column | Type | Description |
|--------|------|-------------|
| `group_name` | VARCHAR(255) | Group name from IdP |
| `access_rule_id` | UUID | References `access_rules.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(group_name, access_rule_id)`

### user_gateways

Assigns gateways directly to users.

| Column | Type | Description |
|--------|------|-------------|
| `user_id` | VARCHAR(255) | User identifier (email) |
| `gateway_id` | UUID | References `gateways.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(user_id, gateway_id)`

### group_gateways

Assigns gateways to groups.

| Column | Type | Description |
|--------|------|-------------|
| `group_name` | VARCHAR(255) | Group name from IdP |
| `gateway_id` | UUID | References `gateways.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(group_name, gateway_id)`

---

## Certificate & Config Tables

### pki_ca

Certificate Authority storage.

| Column | Type | Description |
|--------|------|-------------|
| `id` | VARCHAR(50) | Primary key (default: "default") |
| `certificate_pem` | TEXT | CA certificate in PEM format |
| `private_key_pem` | TEXT | CA private key in PEM format (encrypted) |
| `serial_number` | VARCHAR(100) | Current serial number |
| `not_before` | TIMESTAMPTZ | Certificate validity start |
| `not_after` | TIMESTAMPTZ | Certificate validity end |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### certificates

Issued client certificates.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | References `users.id` |
| `session_id` | UUID | References `sessions.id` |
| `serial_number` | VARCHAR(64) | Certificate serial number (unique) |
| `subject` | VARCHAR(255) | Certificate subject DN |
| `not_before` | TIMESTAMPTZ | Validity start |
| `not_after` | TIMESTAMPTZ | Validity end |
| `fingerprint` | VARCHAR(64) | Certificate fingerprint |
| `is_revoked` | BOOLEAN | Whether certificate is revoked |
| `revoked_at` | TIMESTAMPTZ | Revocation timestamp |
| `revocation_reason` | VARCHAR(100) | Revocation reason |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

### configs

VPN configuration file metadata.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | References `users.id` |
| `session_id` | UUID | References `sessions.id` |
| `certificate_id` | UUID | References `certificates.id` |
| `gateway_id` | UUID | References `gateways.id` |
| `file_name` | VARCHAR(255) | Config file name |
| `expires_at` | TIMESTAMPTZ | Config expiration time |
| `downloaded_at` | TIMESTAMPTZ | Download timestamp |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

### generated_configs

Stores generated .ovpn configuration files.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | VARCHAR(255) | User identifier |
| `gateway_id` | UUID | References `gateways.id` |
| `gateway_name` | VARCHAR(255) | Gateway name at generation time |
| `file_name` | VARCHAR(255) | Config file name |
| `config_data` | BYTEA | Encrypted config file content |
| `serial_number` | VARCHAR(255) | Certificate serial number |
| `fingerprint` | VARCHAR(255) | Certificate fingerprint |
| `cli_callback_url` | VARCHAR(1024) | CLI callback URL |
| `expires_at` | TIMESTAMPTZ | Config expiration time |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `downloaded_at` | TIMESTAMPTZ | Download timestamp |

---

## Connection Tables

### connections

Active and historical VPN connections.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | References `users.id` |
| `session_id` | UUID | References `sessions.id` |
| `certificate_id` | UUID | References `certificates.id` |
| `gateway_id` | UUID | References `gateways.id` |
| `client_ip` | INET | Client's real IP address |
| `vpn_ipv4` | INET | Assigned VPN IPv4 address |
| `vpn_ipv6` | INET | Assigned VPN IPv6 address |
| `bytes_sent` | BIGINT | Bytes sent to client |
| `bytes_received` | BIGINT | Bytes received from client |
| `connected_at` | TIMESTAMPTZ | Connection start time |
| `disconnected_at` | TIMESTAMPTZ | Disconnection time |
| `disconnect_reason` | VARCHAR(100) | Reason for disconnection |

**Index:** Partial index on `disconnected_at IS NULL` for active connections.

---

## Web Proxy Tables

### proxy_applications

Web applications accessible via the reverse proxy.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(255) | Application name |
| `slug` | VARCHAR(100) | URL slug (unique) |
| `description` | TEXT | Description |
| `internal_url` | TEXT | Backend URL to proxy to |
| `icon_url` | TEXT | Icon URL for UI |
| `is_active` | BOOLEAN | Whether app is active |
| `preserve_host_header` | BOOLEAN | Preserve original Host header |
| `strip_prefix` | BOOLEAN | Strip /proxy/{slug} prefix |
| `inject_headers` | JSONB | Headers to inject into requests |
| `allowed_headers` | JSONB | Headers to pass through |
| `websocket_enabled` | BOOLEAN | Enable WebSocket support |
| `timeout_seconds` | INTEGER | Request timeout |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### user_proxy_applications

Assigns proxy apps directly to users.

| Column | Type | Description |
|--------|------|-------------|
| `user_id` | VARCHAR(255) | User identifier |
| `proxy_app_id` | UUID | References `proxy_applications.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(user_id, proxy_app_id)`

### group_proxy_applications

Assigns proxy apps to groups.

| Column | Type | Description |
|--------|------|-------------|
| `group_name` | VARCHAR(255) | Group name from IdP |
| `proxy_app_id` | UUID | References `proxy_applications.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**Primary Key:** `(group_name, proxy_app_id)`

### proxy_access_logs

Audit log for proxy requests.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `proxy_app_id` | UUID | References `proxy_applications.id` |
| `user_id` | VARCHAR(255) | User identifier |
| `user_email` | VARCHAR(255) | User email |
| `request_method` | VARCHAR(10) | HTTP method |
| `request_path` | TEXT | Request path |
| `response_status` | INTEGER | HTTP response status |
| `response_time_ms` | INTEGER | Response time in milliseconds |
| `client_ip` | INET | Client IP address |
| `user_agent` | TEXT | Browser user agent |
| `created_at` | TIMESTAMPTZ | Request timestamp |

---

## Policy Engine Tables

### policies

Named policy containers.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(100) | Policy name (unique) |
| `description` | TEXT | Description |
| `priority` | INTEGER | Evaluation priority (lower = first) |
| `is_enabled` | BOOLEAN | Whether policy is enabled |
| `created_by` | UUID | References `users.id` |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

### policy_rules

Individual rules within a policy.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `policy_id` | UUID | References `policies.id` |
| `action` | VARCHAR(10) | "allow" or "deny" |
| `subject` | JSONB | Who the rule applies to |
| `resource` | JSONB | What resource is being accessed |
| `conditions` | JSONB | Additional conditions |
| `priority` | INTEGER | Rule priority within policy |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

---

## System Tables

### system_settings

Key-value store for system configuration.

| Column | Type | Description |
|--------|------|-------------|
| `key` | VARCHAR(255) | Primary key, setting name |
| `value` | TEXT | Setting value |
| `description` | TEXT | Setting description |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

**Common Settings:**
- `session_duration_hours` - Session lifetime
- `secure_cookies` - Require HTTPS for cookies
- `vpn_cert_validity_hours` - VPN certificate lifetime
- `require_fips` - Require FIPS compliance
- `allowed_crypto_profiles` - Comma-separated allowed profiles
- `min_tls_version` - Minimum TLS version

### audit_logs

Security audit trail.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `timestamp` | TIMESTAMPTZ | Event timestamp |
| `event` | VARCHAR(100) | Event type |
| `actor_id` | UUID | References `users.id` |
| `actor_email` | VARCHAR(255) | Actor's email |
| `actor_ip` | INET | Actor's IP address |
| `resource_type` | VARCHAR(50) | Type of resource affected |
| `resource_id` | UUID | ID of resource affected |
| `details` | JSONB | Additional event details |
| `success` | BOOLEAN | Whether action succeeded |

---

## Entity Relationship Diagram

```
                                    ┌─────────────┐
                                    │   users     │
                                    └──────┬──────┘
                                           │
           ┌───────────────┬───────────────┼───────────────┬───────────────┐
           │               │               │               │               │
           ▼               ▼               ▼               ▼               ▼
    ┌──────────────┐ ┌──────────┐ ┌──────────────┐ ┌────────────┐ ┌───────────────┐
    │user_gateways │ │ sessions │ │user_access   │ │certificates│ │user_proxy     │
    │              │ │          │ │_rules        │ │            │ │_applications  │
    └───────┬──────┘ └────┬─────┘ └──────┬───────┘ └─────┬──────┘ └───────┬───────┘
            │             │              │               │                │
            ▼             │              ▼               │                ▼
    ┌──────────────┐      │       ┌─────────────┐        │        ┌──────────────┐
    │   gateways   │◄─────┼───────│access_rules │        │        │proxy         │
    └───────┬──────┘      │       └──────┬──────┘        │        │_applications │
            │             │              │               │        └──────────────┘
            ▼             │              ▼               │
    ┌──────────────┐      │       ┌─────────────┐        │
    │gateway       │      │       │group_access │        │
    │_networks     │      │       │_rules       │        │
    └───────┬──────┘      │       └─────────────┘        │
            │             │                              │
            ▼             │                              ▼
    ┌──────────────┐      │                       ┌─────────────┐
    │   networks   │      │                       │ connections │
    └──────────────┘      │                       └─────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │   configs   │
                   └─────────────┘
```

---

## Migrations

Migrations are located in `/migrations/` and use the golang-migrate format:

| Migration | Description |
|-----------|-------------|
| 000001 | Initial schema |
| 000002 | Networks and access rules |
| 000003 | Nullable gateway public IP |
| 000004 | Fix gateway active default |
| 000005 | User/group gateway assignments |
| 000006 | OAuth SSO sessions |
| 000007 | OIDC admin group |
| 000008 | System settings |
| 000009 | Generated configs |
| 000010 | Gateway crypto profile |
| 000011 | Gateway VPN subnet |
| 000012 | PKI CA storage |
| 000013 | FIPS requirement |
| 000014 | Proxy applications |
| 000015 | Remove proxy access rules |
| ... | ... |
| 000020 | Gateway full tunnel mode, push DNS, and DNS servers |

Run migrations with:
```bash
migrate -path ./migrations -database "postgres://..." up
```
