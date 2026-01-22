# GateKey Security Model

## Overview

GateKey implements a **Zero Trust Software Defined Perimeter (SDP)** security model. The core principle is **"Never Trust, Always Verify"** - no user or device is trusted by default, and every access request is fully authenticated, authorized, and verified before being granted.

## Default Deny Policy

**All traffic is blocked by default.** Users can only access resources that are explicitly permitted by:
1. Being assigned to a gateway (directly or via group membership)
2. Having access rules that permit specific destinations

This is fundamentally different from traditional VPNs where connecting grants full network access.

## Permission Model

### Entity Relationships

![Entity Relationships](diagrams/entity-relationships.svg)

### Access Control Layers

| Layer | Entity | Purpose |
|-------|--------|---------|
| **1. Gateway Access** | User/Group → Gateway | Controls who can connect to a VPN gateway |
| **2. Network Routes** | Network → Gateway | Controls what CIDR blocks are advertised |
| **3. Access Rules** | User/Group → Access Rule | Controls what specific IPs/hosts users can reach |

### Permission Flow

![Permission Flow](diagrams/permission-flow.svg)

## Security Enforcement Points

GateKey enforces security at **three distinct points**, providing defense in depth:

### 1. Config Generation (`POST /api/v1/configs/generate`)

When a user requests a VPN configuration file:

- **Authentication**: User must have valid session (SSO or local)
- **Gateway Status**: Gateway must be active
- **Access Check**: User must be assigned to gateway (directly or via group)
- **Certificate Binding**: Certificate is bound to specific gateway ID

```go
// Checks performed:
1. Verify user is authenticated
2. Verify gateway exists and is active
3. Verify user has gateway access (UserHasGatewayAccess)
4. Generate short-lived certificate (default: 24 hours)
5. Embed gateway ID in certificate metadata
```

**If any check fails, config generation is denied.**

### 2. Gateway Verify (`POST /api/v1/gateway/verify`)

When OpenVPN attempts to authenticate a client connection:

- **Certificate Validity**: Certificate must not be expired or revoked
- **Gateway Binding**: Certificate must have been issued for this specific gateway
- **User Lookup**: User must exist in the system
- **Account Status**: User account must be active
- **Access Recheck**: User must still have gateway access (may have been revoked)

```go
// Checks performed:
1. Verify gateway token (proves request is from legitimate gateway)
2. Verify certificate serial exists and is not expired
3. Verify certificate was issued for THIS gateway
4. Look up user by email (certificate CN)
5. Verify user account is active
6. Verify user still has gateway access
```

**If any check fails, connection is rejected with specific reason.**

### 3. Gateway Connect (`POST /api/v1/gateway/connect`)

When a connection is established, firewall rules are applied:

- **User Verification**: Re-verify user exists and has access
- **Access Rules**: Retrieve all access rules for user (direct + group-based)
- **Firewall Rules**: Generate firewall rules with default DENY policy
- **Rule Application**: Gateway agent applies nftables/iptables rules

```json
// Response to gateway agent:
{
  "status": "connected",
  "user_id": "...",
  "user_email": "alice@example.com",
  "default_policy": "deny",
  "firewall_rules": [
    {
      "action": "allow",
      "rule_type": "cidr",
      "value": "10.0.0.0/24",
      "port_range": "443",
      "protocol": "tcp"
    }
  ]
}
```

**Only traffic matching explicit allow rules is permitted. All other traffic is dropped.**

## Why Multiple Enforcement Points?

Even if a user obtains a valid `.ovpn` config file, they cannot bypass security because:

| Scenario | Protection |
|----------|------------|
| User shares config file with another person | Certificate CN contains original user's email; verification looks up that user |
| Admin removes user's gateway access after config was generated | Verify step re-checks access at connection time |
| User account is disabled | Verify step checks account status |
| Config file used on different gateway | Certificate is bound to specific gateway ID |
| Certificate expires | Standard X.509 expiration check |
| User connects but tries to access unauthorized resource | Firewall rules only permit explicit destinations |

## Access Rules

Access rules define what resources a user can access within the VPN network.

### Rule Types

| Type | Example | Description |
|------|---------|-------------|
| `ip` | `192.168.1.100` | Single IP address |
| `cidr` | `10.0.0.0/24` | CIDR range |
| `hostname` | `api.internal.com` | Exact hostname |
| `hostname_wildcard` | `*.internal.com` | Wildcard hostname |

### Rule Properties

- **Port Range**: Optional - `443`, `8000-9000`, or `*` for all
- **Protocol**: Optional - `tcp`, `udp`, or `*` for all
- **Network Scope**: Optional - restrict rule to specific network

### Rule Assignment

Rules can be assigned to:
- **Individual users** (by user ID)
- **Groups** (by group name from IdP)

A user's effective access is the union of:
- Rules directly assigned to them
- Rules assigned to any group they belong to

## Short-Lived Certificates

Certificates are designed to be short-lived to limit the exposure window:

| Setting | Default | Purpose |
|---------|---------|---------|
| Certificate Validity | 24 hours | Limits time window if certificate is compromised |
| Session Duration | 8 hours | User must re-authenticate via IdP |

After certificate expiration, users must:
1. Re-authenticate with their identity provider
2. Generate a new configuration
3. Reconnect with the new certificate

## CA Rotation

GateKey supports graceful CA rotation with zero-downtime using a dual-trust period.

### CA Lifecycle States

| Status | Issuing | Trusted | Description |
|--------|---------|---------|-------------|
| `active` | Yes | Yes | Currently issuing new certificates |
| `pending` | No | Yes | Prepared for rotation, not yet activated |
| `retired` | No | Yes | No longer issuing, but still trusted for verification |
| `revoked` | No | No | Completely untrusted |

### Rotation Process

![CA Rotation Process](diagrams/ca-rotation.svg)

### Automatic CA Rotation Detection

Gateways, Mesh Hubs, and Mesh Spokes automatically detect CA rotation:

1. **Heartbeat Response**: Each heartbeat includes `ca_fingerprint` (SHA256 of active CA)
2. **Fingerprint Comparison**: Agent compares with local CA fingerprint
3. **Auto-Reprovision**: If fingerprints differ, agent triggers reprovisioning
4. **Certificate Update**: Agent receives new CA and server certificates
5. **Service Restart**: OpenVPN restarts with new certificates

### Audit Trail

All CA rotation events are logged to `ca_rotation_events` table:
- CA preparation
- CA activation (with old/new fingerprints)
- CA revocation

### Best Practices

1. **Plan rotation during low-traffic periods** - Though zero-downtime, reduces complexity
2. **Allow grace period** - Keep old CA retired (not revoked) for 24-48 hours
3. **Monitor heartbeats** - Verify all gateways detected the change
4. **Test with one gateway first** - Verify rotation works before activating for all

## Firewall Implementation

The gateway agent applies per-user firewall rules using nftables:

```bash
# Example rules for user alice@example.com (VPN IP: 10.8.0.5)
table inet gatekey_alice {
    chain forward {
        type filter hook forward priority 0; policy drop;

        # Allow rules from user's access rules
        ip saddr 10.8.0.5 ip daddr 10.0.0.0/24 tcp dport 443 accept
        ip saddr 10.8.0.5 ip daddr 192.168.1.100 accept

        # Default: drop all other traffic from this user
        ip saddr 10.8.0.5 drop
    }
}
```

Key characteristics:
- **Isolated chains**: Each user gets their own firewall chain
- **Default drop**: Policy is DROP, not ACCEPT
- **Dynamic updates**: Rules are added on connect, removed on disconnect
- **Specific sources**: Rules only apply to user's VPN IP

## Real-Time Rule Enforcement

Access rules are enforced in real-time without requiring client reconnection:

### Architecture

![Real-Time Enforcement Architecture](diagrams/realtime-enforcement.svg)

### Flow

1. **Admin Changes Rule**: Administrator adds/removes access rule in web UI
2. **Database Updated**: Control plane updates `access_rules` table
3. **Agent Polls**: Gateway agent calls `/api/v1/gateway/all-rules` every 10 seconds
4. **Change Detected**: Agent compares current rules with previous state
5. **Firewall Updated**: nftables rules updated for all connected clients
6. **Traffic Blocked**: Client traffic to removed destinations is immediately blocked

### API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/gateway/client-rules` | Get rules for a specific client on connect |
| `POST /api/v1/gateway/all-rules` | Get all rules with user/group assignments for refresh |

### Client Rules Response

```json
{
  "user_id": "abc123",
  "client_ip": "10.8.0.5",
  "allowed": [
    {"type": "ip", "value": "192.168.1.100", "port": "3306", "protocol": "tcp"},
    {"type": "cidr", "value": "10.0.0.0/24", "port": "", "protocol": ""}
  ],
  "default": "deny"
}
```

### Timing

| Event | Latency |
|-------|---------|
| Rule change in UI | Immediate |
| Gateway detects change | ≤10 seconds |
| Firewall updated | <100ms |
| Traffic blocked | Immediate |

**Total time from rule change to enforcement: <15 seconds**

## Geo-Fencing

GateKey supports IP-based geo-fencing to restrict VPN connections based on the client's source IP address. This provides an additional layer of security beyond identity-based access control.

### Whitelist Model

Geo-fencing uses a whitelist-only model:
- Create rules that define allowed IP ranges (CIDR notation)
- Assign rules globally, to groups, or to individual users
- Only connections from listed IPs are permitted
- Connections from unlisted IPs are blocked (or logged in audit mode)

### Rule Hierarchy

Rules follow a hierarchical priority (most specific wins):

| Priority | Level | Applies To |
|----------|-------|------------|
| 1 (highest) | User | Rules assigned directly to the user |
| 2 | Group | Rules assigned to user's groups |
| 3 (lowest) | Global | Default rules for everyone |

If a user has user-specific rules, only those are evaluated. Otherwise, group rules apply. If neither exists, global rules are used.

### Enforcement Points

Geo-fencing is checked at two points for defense-in-depth:

1. **Gateway Verify** (`handleGatewayVerify`) - During certificate verification
2. **Gateway Connect** (`handleGatewayConnect`) - When connection is established

This ensures connections are blocked even if someone obtains a valid certificate.

### Enforcement Modes

| Mode | Behavior |
|------|----------|
| **Enforce** | Block connections from unlisted IPs |
| **Audit** | Log violations but allow connections (for testing) |

### Configuration

Navigate to **Administration → Geo-Fencing** to:
1. Enable/disable geo-fencing
2. Select enforcement mode (enforce or audit)
3. Create rules with allowed IP ranges
4. Assign rules globally, to groups, or to users

### Example Use Cases

- **Country Restriction**: Allow only IP ranges from specific countries
- **Office Only**: Restrict access to corporate network IPs
- **Remote Worker Exception**: Allow specific users from any IP
- **Contractor Restrictions**: Limit contractors to office network only

## Audit Logging

All security-relevant events are logged:

- User authentication (success/failure)
- Config generation requests
- Gateway connection attempts
- Access denials with reasons
- Rule changes
- Admin actions

## Best Practices

### For Administrators

1. **Assign users to gateways explicitly** - Don't leave gateways open
2. **Use groups from IdP** - Manage access via identity provider groups
3. **Create specific access rules** - Avoid overly broad CIDR ranges
4. **Review access regularly** - Audit who has access to what
5. **Monitor audit logs** - Watch for unusual patterns

### For Users

1. **Don't share config files** - They're tied to your identity
2. **Regenerate configs regularly** - Don't reuse expired configs
3. **Report lost devices** - Admin can revoke certificates

## Comparison with Traditional VPN

| Aspect | Traditional VPN | GateKey |
|--------|-----------------|---------|
| Default Policy | Allow all after connect | Deny all |
| Access Control | Network-level | User + Resource level |
| Certificate Life | Years | Hours |
| Access Revocation | Manual certificate revocation | Immediate (access check on connect) |
| Audit Trail | Connection logs only | Full resource access logging |
| Group Integration | None | Native IdP group support |

---

## Security Hardening

GateKey implements multiple layers of security hardening to protect sensitive data and prevent common attack vectors.

### Certificate Revocation Lists (CRL)

GateKey provides a complete certificate revocation infrastructure for invalidating compromised certificates.

#### CRL Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/pki/crl` | GET | Public | Download CRL in DER format |
| `/pki/crl.pem` | GET | Public | Download CRL in PEM format |
| `/pki/check/:serial` | GET | Public | Check if a certificate is revoked |
| `/admin/pki/revoke` | POST | Admin | Revoke a certificate |
| `/admin/pki/revocations` | GET | Admin | List all revoked certificates |
| `/admin/pki/revocations/:serial` | DELETE | Admin | Unrevoke a certificate |

#### Revocation Reasons (RFC 5280)

| Code | Reason |
|------|--------|
| 0 | Unspecified |
| 1 | Key Compromise |
| 2 | CA Compromise |
| 3 | Affiliation Changed |
| 4 | Superseded |
| 5 | Cessation of Operation |
| 6 | Certificate Hold |

#### CRL Caching

- CRLs are cached with 24-hour validity
- Cache is automatically invalidated when certificates are revoked
- HTTP Cache-Control headers enable client-side caching (1 hour)

#### Usage Example

```bash
# Revoke a certificate
curl -X POST https://control-plane/admin/pki/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"serial_number": "1234567890", "reason": 1, "notes": "Key compromise suspected"}'

# Check revocation status
curl https://control-plane/pki/check/1234567890

# Download CRL
curl -o revoked.crl https://control-plane/pki/crl
```

### Immediate User Session Termination

GateKey provides the ability to immediately terminate active VPN sessions for specific users. This is critical for security incident response and access revocation.

#### Capabilities

| Action | Effect | Use Case |
|--------|--------|----------|
| **Disable User** | Terminates all sessions, revokes configs, prevents login | Employee offboarding, security incidents |
| **Disconnect Session** | Terminates a specific VPN session | Maintenance, troubleshooting |
| **Disconnect All Sessions** | Terminates all sessions for a user | Immediate access revocation |

#### Key Security Features

- **Individual User Targeting**: Only the specific user is affected - other users in the same groups are not impacted
- **Infrastructure Preservation**: Gateways, hubs, and spokes remain running and serving other users
- **Immediate Effect**: Sessions are terminated within seconds via the gateway polling mechanism
- **Audit Trail**: All disconnect actions are logged with the admin who initiated them

#### Usage

**Web UI:**
1. Navigate to Admin → Users & Groups
2. Find the user and click Actions → Disable User

**CLI:**
```bash
# Disable user and disconnect all sessions
gatekey-admin user disable USER_ID --reason "Security concern"

# Disconnect sessions without disabling account
gatekey-admin user disconnect USER_ID

# View active sessions for a user
gatekey-admin user sessions USER_ID

# Re-enable a disabled user
gatekey-admin user enable USER_ID
```

**API:**
```bash
# Disable user
curl -X POST https://control-plane/api/v1/admin/users/$USER_ID/disable \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Security incident", "disconnect_active": true}'

# Disconnect all sessions for a user
curl -X POST https://control-plane/api/v1/admin/users/$USER_ID/disconnect \
  -H "Authorization: Bearer $TOKEN"
```

### Remote Sessions and Network Tools

GateKey provides Remote Sessions and Network Tools features that allow administrators to run diagnostic commands on gateways (ping, traceroute, netstat, etc.) and establish SSH-like sessions for troubleshooting.

#### Security Model

This feature executes shell commands on gateway agents. This is **intentional functionality**, not a vulnerability:

| Security Control | Implementation |
|-----------------|----------------|
| **Authentication** | Only authenticated administrators can access remote sessions |
| **Authorization** | Requires admin role or specific API key scope |
| **Execution Location** | Commands run on the gateway agent, NOT the control plane |
| **Gateway Authentication** | Gateway agents authenticate with tokens before accepting commands |
| **Audit Logging** | All remote session commands are logged with timestamps and user IDs |

#### Design Decision: Shell Execution

Remote shell execution is the **core feature** of the Remote Sessions functionality. Alternative approaches (pre-defined commands only, containerized execution) would severely limit the troubleshooting capabilities that make this feature valuable.

**Mitigations in place:**
- Commands cannot target the control plane (only gateways)
- Gateway tokens are rotated regularly
- Session activity is fully auditable
- Network isolation between gateways limits blast radius

#### Usage Restrictions

- Remote Sessions are disabled by default on gateway agents
- Must be explicitly enabled in gateway configuration
- Should only be enabled on gateways where remote troubleshooting is required

### CA Private Key Encryption

CA private keys are encrypted at rest in the database using AES-256-GCM authenticated encryption. **Encryption is now enabled by default** with automatic key generation.

#### How It Works

1. **Auto-Generation**: If no encryption key is configured, GateKey automatically generates a cryptographically secure 256-bit key on first startup
2. **Persistence**: The generated key is stored in the `system_settings` database table, ensuring cluster-wide consistency across multiple pods/replicas
3. **Configuration Override**: You can still provide your own key via environment variable or config file

#### Design Decision: Database Key Storage

The encryption key is stored in the same database as the encrypted CA private keys. This design was chosen for:

- **Kubernetes Simplicity**: No need for persistent volumes to store a key file
- **Cluster Consistency**: All pods automatically use the same key without shared filesystem requirements
- **Zero Configuration**: Works out-of-the-box with no manual key generation required

**Security Trade-offs:**
- An attacker with full database access could retrieve both the key and encrypted data
- However, this still protects against:
  - Database backup exposure (backups don't include application context to find the key)
  - Accidental data leaks (encrypted data is meaningless without understanding the key location)
  - Defense in depth (multiple layers of protection)

For higher security requirements, set the key via environment variable (stored in Kubernetes secrets):

```bash
export GATEX_SECURITY_CA_KEY_ENCRYPTION_KEY="$(openssl rand -base64 32)"
```

#### Configuration (Optional)

Provide your own key instead of using auto-generation:

```yaml
security:
  ca_key_encryption_key: "base64-encoded-32-byte-key"
```

To generate a key manually:

```bash
openssl rand -base64 32
```

#### Features

- **AES-256-GCM**: Provides both confidentiality and integrity
- **Random Nonce**: Each encryption uses a unique 12-byte nonce
- **Backward Compatible**: Unencrypted keys are still readable (migration-friendly)
- **Automatic Detection**: System detects whether keys are encrypted or plaintext
- **Secure Wiping**: Keys are wiped from memory after use
- **Cluster-Wide**: Auto-generated key is shared across all replicas via database

#### Storage Format

Encrypted keys are stored as: `ENC:base64(nonce + ciphertext + tag)`

### Secure Memory Wiping

Sensitive data like Pre-Shared Keys (PSKs) are securely wiped from disk to prevent recovery.

#### Wiping Process

When PSK temporary files are deleted:

1. **Random Overwrite**: File is overwritten with cryptographically random data
2. **Sync**: Data is flushed to disk with `fsync()`
3. **Zero Overwrite**: File is overwritten with zeros
4. **Sync**: Data is flushed again
5. **Delete**: File is finally removed

This multi-pass approach prevents data recovery from:
- File system journaling
- SSD wear-leveling (to the extent possible)
- Deleted file recovery tools

#### File Permissions

PSK files are created with `0600` permissions (owner read/write only) before any sensitive data is written.

### Password Hashing (Argon2id)

Local user passwords are hashed using Argon2id, the winner of the Password Hashing Competition.

#### Parameters

| Parameter | Value | Purpose |
|-----------|-------|---------|
| Time Cost | 3 iterations | OWASP recommends 2-3 |
| Memory | 64 MB | Increases GPU attack cost |
| Parallelism | 4 threads | Balances security and performance |
| Key Length | 32 bytes | 256-bit output |
| Salt | 16 bytes | Unique per password |

#### Hash Format

```
$argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
```

#### Timing Attack Prevention

When a user doesn't exist, password hashing is still performed to prevent timing attacks that could enumerate valid usernames.

### SSRF Protection

Server-Side Request Forgery (SSRF) attacks are prevented for proxy and external URL features.

#### Protected Features

- Proxy configuration with custom URLs
- Webhook callbacks
- External service integrations

#### Blocked Destinations

The following are blocked to prevent internal network access:

| Type | Examples |
|------|----------|
| Private IPv4 | `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` |
| Loopback | `127.0.0.0/8`, `::1` |
| Link-local | `169.254.0.0/16`, `fe80::/10` |
| Private DNS | `.local`, `.internal`, `.localhost` |

#### DNS Rebinding Prevention

- DNS resolution is performed before the request
- Resolved IP is checked against blocklist
- Request is blocked if resolved IP is internal

#### Configurable CA Bundles

For proxy TLS verification, custom CA bundles can be configured:

```yaml
security:
  proxy_ca_bundle: "/path/to/ca-bundle.crt"
```

This allows:
- Using internal/corporate CAs for proxies
- Strict TLS verification (no InsecureSkipVerify)
- Custom trust anchors per environment

### API Key Scopes

API keys support fine-grained permission scopes to limit access.

#### Available Scopes

| Scope | Permissions |
|-------|-------------|
| `admin` | Full administrative access |
| `gateway:read` | Read gateway configuration |
| `gateway:write` | Modify gateway settings |
| `user:read` | Read user information |
| `user:write` | Manage users |
| `config:generate` | Generate VPN configurations |
| `audit:read` | Read audit logs |

#### Scope Enforcement

Scopes are enforced via middleware on all API endpoints:

```go
pkiRoutes.Use(s.requireScope(ScopeAdmin))
```

API keys without required scopes receive `403 Forbidden`.

### Security Headers

All HTTP responses include security headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-XSS-Protection` | `1; mode=block` | XSS filtering |
| `Strict-Transport-Security` | `max-age=31536000` | Force HTTPS |
| `Content-Security-Policy` | Restrictive policy | Prevent XSS/injection |

### Database Security

#### Prepared Statements

All database queries use parameterized prepared statements to prevent SQL injection:

```go
// Safe: parameterized query
db.QueryRow(ctx, "SELECT * FROM users WHERE email = $1", email)

// Never used: string concatenation
db.QueryRow(ctx, "SELECT * FROM users WHERE email = '" + email + "'")
```

#### Connection Security

- TLS required for production database connections (`ssl_mode: require` default)
- Connection pooling with secure defaults
- Automatic connection health checks

#### Database TLS Configuration

GateKey requires encrypted connections to PostgreSQL by default. The SSL/TLS mode can be configured via environment variable or config file.

##### SSL Modes

| Mode | Description | Recommended For |
|------|-------------|-----------------|
| `disable` | No encryption (NOT recommended) | Local development only |
| `require` | Encrypted, no certificate verification | **Default** - Internal networks with self-signed certs |
| `verify-ca` | Encrypted, verify CA signature | Production with internal CA |
| `verify-full` | Encrypted, verify CA + hostname | Production with public CA |

##### Design Decision: Why `require` is the Default

The default SSL mode is `require` (encrypted but no certificate verification) rather than `verify-full` for practical reasons:

1. **Kubernetes In-Cluster Databases**: Most Kubernetes PostgreSQL deployments (Bitnami, CloudNativePG, etc.) don't generate TLS certificates by default. Using `verify-full` would break out-of-the-box deployments.

2. **Managed Database Services**: AWS RDS and similar services use their own CA certificates that require downloading and configuring the RDS CA bundle. `verify-ca` works but requires additional setup.

3. **Internal Network Security**: In most deployments, the database is within the same Kubernetes cluster or private VPC, where network-level encryption is less critical than external-facing services.

**For production deployments with external databases**, upgrade to `verify-ca` or `verify-full`:

```yaml
database:
  ssl_mode: "verify-ca"
  ssl_root_cert: "/path/to/rds-ca-bundle.pem"  # For AWS RDS
```

##### Configuration

**Environment Variable (Recommended):**

```bash
# Set SSL mode via environment variable
export DATABASE_SSL_MODE=require
```

**Config File:**

```yaml
database:
  url: "${DATABASE_URL}"
  ssl_mode: "${DATABASE_SSL_MODE}"  # Defaults to 'require'
  # For verify-ca/verify-full modes:
  ssl_root_cert: "/path/to/ca.crt"
  ssl_cert: "/path/to/client.crt"  # Optional: for mTLS
  ssl_key: "/path/to/client.key"   # Optional: for mTLS
```

**Database URL:**

The `sslmode` parameter in the database URL also controls TLS:

```
postgres://user:pass@host:5432/db?sslmode=require
```

##### PostgreSQL Server Configuration

For TLS to work, PostgreSQL must be configured with SSL:

```conf
# postgresql.conf
ssl = on
ssl_cert_file = '/path/to/server.crt'
ssl_key_file = '/path/to/server.key'
```

##### Kubernetes Deployment

When deploying on Kubernetes, the included manifests automatically:

1. Generate self-signed TLS certificates for PostgreSQL via init container
2. Configure PostgreSQL with SSL enabled
3. Set `DATABASE_SSL_MODE=require` on the application

To override the SSL mode in Kubernetes:

```yaml
env:
  - name: DATABASE_SSL_MODE
    value: "verify-ca"  # Override the default
  - name: DATABASE_SSL_ROOT_CERT
    value: "/path/to/ca.crt"
```

##### Security Best Practices

1. **Production**: Use `require` at minimum, `verify-ca` or `verify-full` preferred
2. **Never use `disable`** in production - credentials transmitted in cleartext
3. **Certificate rotation**: PostgreSQL certs should be rotated annually
4. **Network isolation**: Even with TLS, database should not be publicly accessible

### Cryptographic Standards

| Purpose | Algorithm | Standard |
|---------|-----------|----------|
| Password Hashing | Argon2id | PHC Winner |
| Key Encryption | AES-256-GCM | NIST FIPS 197/SP 800-38D |
| TLS | TLS 1.2+ | RFC 5246/8446 |
| Certificates | ECDSA P-256 / RSA 2048+ | NIST FIPS 186-4 |
| Random Generation | crypto/rand | OS CSPRNG |

### Security Headers (Multi-Tenant Support)

GateKey includes comprehensive security headers that are **enabled by default** and designed for multi-tenant deployments (e.g., `vpn.companya.com`, `vpn.companyb.com`).

#### Headers Applied

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevents MIME type sniffing |
| `X-Frame-Options` | `SAMEORIGIN` | Legacy clickjacking protection |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS protection |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer info |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=(), payment=()` | Restricts browser features |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforces HTTPS (when TLS enabled) |
| `Content-Security-Policy` | Dynamic (see below) | Main XSS/injection defense |

#### Multi-Tenant CSP Configuration

For multi-tenant deployments, the `frame-ancestors` directive is dynamically set based on the request's `Host` header:

```yaml
security:
  security_headers_enabled: true
  frame_ancestors: "dynamic"  # Automatically uses Host header
```

This allows:
- `vpn.companya.com` can only be framed by `vpn.companya.com`
- `vpn.companyb.com` can only be framed by `vpn.companyb.com`
- Prevents cross-tenant clickjacking attacks

#### Configuration Options

**Environment Variables:**

```bash
# Enable/disable security headers (default: true)
export GATEX_SECURITY_SECURITY_HEADERS_ENABLED=true

# HSTS settings (only applied when TLS is enabled)
export GATEX_SECURITY_HSTS_ENABLED=true
export GATEX_SECURITY_HSTS_MAX_AGE=31536000
export GATEX_SECURITY_HSTS_INCLUDE_SUBDOMAINS=true
export GATEX_SECURITY_HSTS_PRELOAD=false

# Frame ancestors: "self", "dynamic", or specific domains
export GATEX_SECURITY_FRAME_ANCESTORS=dynamic

# Custom CSP (overrides default if set)
export GATEX_SECURITY_CONTENT_SECURITY_POLICY=""

# Permissions policy
export GATEX_SECURITY_PERMISSIONS_POLICY="geolocation=(), camera=(), microphone=(), payment=()"
```

**Config File:**

```yaml
security:
  security_headers_enabled: true
  hsts_enabled: true
  hsts_max_age: 31536000
  hsts_include_subdomains: true
  hsts_preload: false
  frame_ancestors: "dynamic"  # or "'self'" or "https://allowed.domain.com"
  content_security_policy: ""  # Empty = use secure defaults
  permissions_policy: "geolocation=(), camera=(), microphone=(), payment=()"
```

#### Frame Ancestors Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `dynamic` | Uses request `Host` header | Multi-tenant SaaS deployments |
| `self` | Only same-origin allowed | Single-tenant deployments |
| `https://example.com` | Specific domain(s) | Embedding in known parent app |

#### CSP Directives Design Decisions

The default Content Security Policy balances security with React/modern frontend framework compatibility:

```
default-src 'self';
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https:;
font-src 'self' data:;
connect-src 'self' wss: https:;
frame-ancestors [dynamic];
base-uri 'self';
form-action 'self'
```

**Design Rationale:**

| Directive | Decision | Reason |
|-----------|----------|--------|
| `'unsafe-inline'` (scripts) | **Kept** | Required for React production builds and inline event handlers |
| `'unsafe-eval'` | **Removed** | Not needed for React production builds; eval() is a significant XSS vector |
| `'unsafe-inline'` (styles) | **Kept** | Required for Tailwind CSS and React styled components |
| `data:` (images/fonts) | **Kept** | Common pattern for base64-encoded icons and fonts |

**Why not use nonces?**

CSP nonces (`'nonce-xyz123'`) provide stronger security than `'unsafe-inline'` but require:
1. Server-side rendering to inject nonces into HTML
2. Build-time modifications to inject nonces into bundled scripts
3. Additional complexity in the deployment pipeline

For an admin-only UI like GateKey, the risk/complexity trade-off favors simplicity. The UI is only accessible to authenticated administrators, not public users.

**Custom CSP Override:**

Organizations requiring stricter CSP can override the default:

```yaml
security:
  content_security_policy: "default-src 'self'; script-src 'self' 'nonce-${REQUEST_NONCE}'; ..."
```

### SSRF Protection

Server-Side Request Forgery (SSRF) protection is **enabled by default** to prevent the reverse proxy from being abused to access internal networks.

#### Protected IP Ranges

When enabled, requests to these ranges are blocked:
- RFC 1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- RFC 3927: `169.254.0.0/16` (includes AWS metadata endpoint)
- Loopback: `127.0.0.0/8`, `::1`
- IPv6 ULA: `fc00::/7`
- IPv6 Link-Local: `fe80::/10`

#### Configuration

```yaml
security:
  # SSRF protection (default: true)
  proxy_ssrf_protection: true

  # DNS rebinding protection (default: true)
  proxy_dns_rebinding_protection: true

  # TLS verification for proxy targets (default: true)
  proxy_tls_verify: true

  # Custom CA bundle for proxy TLS verification
  proxy_ca_bundle: "/path/to/ca-bundle.crt"

  # Minimum TLS version for proxy connections
  proxy_tls_min_version: "1.2"
```

**Note:** If you need to proxy to internal RFC1918 addresses (legitimate VPN use case), you can disable SSRF protection:

```yaml
security:
  proxy_ssrf_protection: false  # Only disable if proxying to internal networks
```

### Recent Security Hardening (v1.2+)

The following security improvements were added to address common attack vectors:

#### CLI Callback URL Validation

CLI authentication flows now validate callback URLs to prevent SSRF and open redirect attacks. Only the following callback URL schemes are allowed:

| Scheme | Example | Purpose |
|--------|---------|---------|
| `http://localhost:*` | `http://localhost:8080/callback` | Desktop CLI clients |
| `http://127.0.0.1:*` | `http://127.0.0.1:8080/callback` | Desktop CLI clients |
| `http://[::1]:*` | `http://[::1]:8080/callback` | IPv6 localhost |
| `gatekey://` | `gatekey://callback` | Mobile apps (custom scheme) |

External URLs (e.g., `https://evil.com/steal-token`) are rejected with a `400 Bad Request` error.

#### Rate Limiting for SSO Endpoints

OIDC and SAML authentication endpoints are now rate-limited to prevent abuse:

| Endpoint | Rate Limit | Purpose |
|----------|------------|---------|
| `/auth/oidc/login` | 50/min per IP | Prevent login flood attacks |
| `/auth/oidc/callback` | 50/min per IP | Prevent callback abuse |
| `/auth/saml/login` | 50/min per IP | Prevent login flood attacks |
| `/auth/saml/acs` | 50/min per IP | Prevent assertion replay attacks |
| `/auth/cli/login` | 50/min per IP | Prevent CLI abuse |

Requests exceeding the limit receive `429 Too Many Requests`.

#### Default API Key Expiration

API keys now have a default expiration of **90 days** if no expiration is specified. This follows the principle of least privilege and limits exposure from compromised keys.

To create a non-expiring key, explicitly set `"expires_in": "never"`:

```json
{
  "name": "service-key",
  "expires_in": "never"  // Required for non-expiring keys
}
```

#### Role-Based API Key Scopes

API keys now receive role-appropriate default scopes:

| User Role | Default Scopes | Purpose |
|-----------|----------------|---------|
| Admin | `["admin"]` | Full administrative access |
| Regular User | `["vpn:connect", "gateways:read", "mesh:read"]` | VPN access only |

This replaces the previous default of `["*"]` which granted excessive permissions.

#### SAML ACS CORS Hardening

The SAML Assertion Consumer Service (ACS) endpoint no longer includes permissive CORS headers (`Access-Control-Allow-Origin: *`). SAML IdPs use browser form POSTs which don't require CORS, so removing these headers prevents unnecessary exposure.

#### TLS Configuration Warnings

The server now logs warnings for insecure TLS configurations at startup:

| Configuration | Log Level | Message |
|---------------|-----------|---------|
| `ssl_mode=disable` | WARN | Database connections are unencrypted |
| `ssl_mode=require` | WARN | Server certificate is not verified |
| `proxy_tls_verify=false` | WARN | Proxy targets are not verified |

#### RSA-2048 Deprecation

RSA-2048 keys are deprecated for new CA generation. A warning is logged if `key_algorithm: rsa2048` is configured:

```
DEPRECATION WARNING: RSA-2048 key algorithm is deprecated and will be removed
in a future version. Migrate to ecdsa256, ecdsa384, or rsa4096.
```

**Recommended key algorithms:**
- `ecdsa256` - 128 bits of security (default, recommended)
- `ecdsa384` - 192 bits of security (high security)
- `rsa4096` - 140 bits of security (legacy compatibility)

#### Reduced Sensitive Data in Logs

Redirect URLs containing tokens are no longer logged in full. Instead, only the callback base URL and user email are logged, preventing token exposure in log files.

### Security Checklist

#### Deployment

- [x] SSRF protection enabled (default)
- [x] Security headers enabled (default)
- [x] TLS verification for proxy targets enabled (default)
- [ ] Set `CA_KEY_ENCRYPTION_KEY` environment variable
- [ ] Configure `PROXY_CA_BUNDLE` if using HTTPS proxy
- [ ] Enable TLS for database connections
- [ ] Use HTTPS for all external endpoints
- [ ] Configure firewall to block direct database access
- [ ] Set up CRL distribution point in certificate configurations

#### Operations

- [ ] Rotate CA keys annually (or on compromise)
- [ ] Review audit logs weekly
- [ ] Check CRL for unexpected revocations
- [ ] Verify API key scopes are minimal
- [ ] Test backup/restore procedures
- [ ] Monitor for failed authentication attempts

#### Incident Response

- [ ] Document certificate revocation procedure
- [ ] Have CA rotation runbook ready
- [ ] Know how to disable compromised API keys
- [ ] Have database encryption key backup in secure location
