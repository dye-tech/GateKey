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

```
┌─────────────┐         ┌─────────────┐         ┌──────────────┐
│   Users     │◄───────►│  Gateways   │◄───────►│   Networks   │
│  (SSO/Local)│         │  (VPN nodes)│         │  (CIDR blocks)│
└──────┬──────┘         └─────────────┘         └──────────────┘
       │                                                │
       │                                                │
       ▼                                                ▼
┌─────────────┐                                 ┌──────────────┐
│   Groups    │◄───────────────────────────────►│ Access Rules │
│ (from IdP)  │                                 │ (IP/hostname)│
└─────────────┘                                 └──────────────┘
```

### Access Control Layers

| Layer | Entity | Purpose |
|-------|--------|---------|
| **1. Gateway Access** | User/Group → Gateway | Controls who can connect to a VPN gateway |
| **2. Network Routes** | Network → Gateway | Controls what CIDR blocks are advertised |
| **3. Access Rules** | User/Group → Access Rule | Controls what specific IPs/hosts users can reach |

### Permission Flow

```
User requests VPN access
         │
         ▼
┌─────────────────────────────┐
│ 1. Is user assigned to      │ ──NO──► Access Denied
│    this gateway?            │
└──────────────┬──────────────┘
               │ YES
               ▼
┌─────────────────────────────┐
│ 2. Generate config with     │
│    short-lived certificate  │
└──────────────┬──────────────┘
               │
               ▼
User connects with OpenVPN client
               │
               ▼
┌─────────────────────────────┐
│ 3. Gateway verifies:        │
│    - Certificate valid?     │ ──NO──► Connection Rejected
│    - User still has access? │
│    - Account active?        │
└──────────────┬──────────────┘
               │ YES
               ▼
┌─────────────────────────────┐
│ 4. Retrieve user's access   │
│    rules and apply firewall │
│    (Default: DENY ALL)      │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ 5. User can only reach      │
│    destinations permitted   │
│    by their access rules    │
└─────────────────────────────┘
```

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

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Admin UI       │────►│  Control Plane  │────►│  Gateway Agent  │
│  (Rule Change)  │     │  (Database)     │     │  (nftables)     │
└─────────────────┘     └────────┬────────┘     └────────┬────────┘
                                 │                       │
                                 ▼                       ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │ /gateway/       │     │  Client Traffic │
                        │ client-rules    │────►│  Immediately    │
                        │ all-rules       │     │  Blocked/Allowed│
                        └─────────────────┘     └─────────────────┘
```

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
