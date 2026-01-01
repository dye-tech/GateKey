# GateKey Architecture

## Overview

GateKey is a Software Defined Perimeter (SDP) solution that wraps OpenVPN to provide zero-trust VPN capabilities while maintaining 100% compatibility with existing OpenVPN clients.

## System Components

### Control Plane (`gatekey-server`)

The control plane is the central management component that handles:

- **Authentication**: OIDC and SAML integration with identity providers
- **Authorization**: Policy-based access control
- **Certificate Management**: Embedded PKI for short-lived certificates
- **Configuration Generation**: Dynamic .ovpn file generation
- **Session Management**: User session tracking and validation
- **Gateway Management**: Registration and monitoring of gateway nodes
- **Audit Logging**: Comprehensive audit trail

```
┌─────────────────────────────────────────────────────────────────┐
│                      CONTROL PLANE                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │   Web UI     │  │   REST API   │  │   gRPC API   │           │
│  │  (React/TS)  │  │   (Go/Gin)   │  │  (internal)  │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│         └─────────────────┼──────────────────┘                   │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐    │
│  │                    Core Services                         │    │
│  ├─────────────┬─────────────┬─────────────┬───────────────┤    │
│  │  Auth Svc   │  Policy Svc │   PKI Svc   │  Session Svc  │    │
│  │ (OIDC/SAML) │  (RBAC/ACL) │ (Cert Gen)  │  (State Mgmt) │    │
│  └─────────────┴─────────────┴─────────────┴───────────────┘    │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐    │
│  │                    Data Layer                            │    │
│  │                    PostgreSQL                            │    │
│  └──────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Gateway Agent (`gatekey-gateway`)

The gateway agent runs alongside OpenVPN on each gateway node:

- **Hook Handling**: Processes OpenVPN hook callbacks
- **Firewall Management**: Per-identity nftables/iptables rules
- **Connection Reporting**: Reports connection state to control plane
- **Health Monitoring**: Sends heartbeats to control plane

```
┌─────────────────────────────────────────────────────────────────┐
│                      GATEWAY NODE                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐                     │
│  │  OpenVPN Server  │  │  GateKey Gateway   │                     │
│  │    (Stock)       │◄─┤     Agent        │                     │
│  └────────┬─────────┘  └────────┬─────────┘                     │
│           │                      │                               │
│           │ Hook Scripts         │ API Calls                     │
│           │                      │                               │
│  ┌────────┴──────────────────────┴──────────────────────────┐   │
│  │              Firewall Manager (nftables)                  │   │
│  │         Per-identity rules, narrow route enforcement      │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Mesh Hub (`gatekey-hub`)

The mesh hub enables site-to-site VPN connectivity using a hub-and-spoke topology:

- **OpenVPN Server**: Runs the mesh OpenVPN server for spoke connections
- **Route Aggregation**: Collects routes from all connected spokes
- **Client VPN Access**: Allows authorized users to connect as VPN clients
- **Control Plane Sync**: Syncs configuration and access rules from control plane
- **Health Monitoring**: Sends heartbeats to control plane

```
┌─────────────────────────────────────────────────────────────────┐
│                      MESH HUB NODE                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐                     │
│  │  OpenVPN Server  │  │   GateKey Hub    │                     │
│  │    (Mesh)        │◄─┤   (gatekey-hub)  │                     │
│  └────────┬─────────┘  └────────┬─────────┘                     │
│           │                      │                               │
│           │ Spoke/Client         │ API Sync                      │
│           │ Connections          │                               │
│           │                      │                               │
│  ┌────────┴──────────────────────┴──────────────────────────┐   │
│  │              Route Aggregation & Distribution             │   │
│  │         Collects spoke routes, pushes to clients          │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Mesh Spoke (`gatekey-mesh-gateway`)

The mesh spoke connects remote sites to the mesh hub:

- **Outbound Connection**: Initiates connection to hub (works behind NAT)
- **Local Network Advertisement**: Advertises local networks to the hub
- **Automatic Reconnection**: Maintains persistent connection to hub
- **Control Plane Sync**: Receives configuration updates

```
┌─────────────────────────────────────────────────────────────────┐
│                      MESH SPOKE NODE                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐                     │
│  │  OpenVPN Client  │  │   Mesh Gateway   │                     │
│  │  (to Hub)        │◄─┤ (gatekey-mesh-gw)│                     │
│  └────────┬─────────┘  └────────┬─────────┘                     │
│           │                      │                               │
│           │ VPN Tunnel           │ Local Network                 │
│           │ to Hub               │ Routing                       │
│           │                      │                               │
│  ┌────────┴──────────────────────┴──────────────────────────┐   │
│  │              Local Network Access (10.0.0.0/8, etc.)      │   │
│  │         Routes traffic between hub and local networks     │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow

### User Authentication Flow

```
┌──────┐     ┌─────────┐     ┌──────────────┐     ┌─────┐
│ User │────▶│ Web UI  │────▶│ Control Plane│────▶│ IdP │
└──────┘     └─────────┘     └──────────────┘     └─────┘
    │                              │                  │
    │    1. Access Web UI          │                  │
    │◄────────────────────────────▶│                  │
    │                              │                  │
    │    2. Redirect to IdP        │                  │
    │─────────────────────────────▶│─────────────────▶│
    │                              │                  │
    │    3. Authenticate           │                  │
    │◄────────────────────────────────────────────────│
    │                              │                  │
    │    4. Callback with token    │                  │
    │─────────────────────────────▶│◄─────────────────│
    │                              │                  │
    │    5. Create session         │                  │
    │◄─────────────────────────────│                  │
```

### VPN Connection Flow

```
┌──────┐     ┌────────────┐     ┌──────────────┐     ┌─────────┐
│ User │────▶│ OpenVPN    │────▶│   Gateway    │────▶│ Control │
│      │     │  Client    │     │   Agent      │     │ Plane   │
└──────┘     └────────────┘     └──────────────┘     └─────────┘
    │              │                   │                   │
    │  1. Connect  │                   │                   │
    │─────────────▶│                   │                   │
    │              │  2. TLS Handshake │                   │
    │              │──────────────────▶│                   │
    │              │                   │  3. Verify Cert   │
    │              │                   │──────────────────▶│
    │              │                   │  4. Auth Result   │
    │              │                   │◀──────────────────│
    │              │                   │                   │
    │              │  5. Connect Hook  │                   │
    │              │──────────────────▶│                   │
    │              │                   │  6. Get Policies  │
    │              │                   │──────────────────▶│
    │              │                   │  7. Policy Rules  │
    │              │                   │◀──────────────────│
    │              │                   │                   │
    │              │                   │  8. Apply FW Rules│
    │              │                   │───────┐           │
    │              │                   │◀──────┘           │
    │              │                   │                   │
    │              │  9. Push Config   │                   │
    │              │◀──────────────────│                   │
    │              │                   │                   │
    │  10. Tunnel  │                   │                   │
    │◀────────────▶│◀─────────────────▶│                   │
```

### Mesh Networking Flow

```
┌──────┐     ┌────────────┐     ┌──────────────┐     ┌─────────┐
│ Spoke│────▶│    Hub     │────▶│   Control    │     │  User   │
│      │     │            │     │   Plane      │     │ Client  │
└──────┘     └────────────┘     └──────────────┘     └─────────┘
    │              │                   │                   │
    │  1. Connect  │                   │                   │
    │─────────────▶│                   │                   │
    │              │  2. Verify Token  │                   │
    │              │──────────────────▶│                   │
    │              │  3. Auth OK       │                   │
    │              │◀──────────────────│                   │
    │              │                   │                   │
    │  4. Advertise│                   │                   │
    │  local nets  │                   │                   │
    │─────────────▶│                   │                   │
    │              │                   │                   │
    │              │                   │  5. User connects │
    │              │◀──────────────────┼──────────────────│
    │              │                   │                   │
    │              │  6. Push routes   │                   │
    │              │  (based on access)│                   │
    │              │──────────────────▶│                   │
    │              │                   │                   │
    │  7. Traffic  │◀────────────────────────────────────▶│
    │   flows via  │                   │                   │
    │     hub      │                   │                   │
```

## Security Model

> **For detailed security documentation, see [security.md](security.md)**

### Zero Trust Principles

1. **Never Trust, Always Verify**: Every connection is authenticated and authorized
2. **Least Privilege**: Users only access resources explicitly allowed by policy
3. **Assume Breach**: Short-lived certificates limit exposure window
4. **Continuous Verification**: Sessions are validated on each connection
5. **Default Deny**: All traffic is blocked unless explicitly allowed by access rules

### Defense in Depth

Security is enforced at three points:

1. **Config Generation**: User must have gateway access to generate VPN config
2. **Connection Verification**: Gateway re-verifies user access at connection time
3. **Firewall Enforcement**: Per-user firewall rules with default DENY policy

This means even if a user obtains a valid `.ovpn` file, they cannot bypass security:
- Access is re-checked when they connect
- Only traffic to explicitly permitted destinations is allowed
- Certificate is bound to specific gateway

### Certificate Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    Certificate Lifecycle                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐         │
│  │ Request │──▶│ Issue   │──▶│ Active  │──▶│ Expire  │         │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘         │
│       │                           │              │               │
│       │                           │              │               │
│       │                      ┌────┴────┐         │               │
│       │                      │ Revoke  │─────────┘               │
│       │                      └─────────┘                         │
│       │                                                          │
│  Typical lifetime: 24 hours                                      │
│  User must re-authenticate to get new certificate                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Firewall Rules

Per-identity firewall rules are applied at the gateway level:

```
# Example nftables rules for user "alice@example.com"
table inet gatekey {
    chain forward {
        type filter hook forward priority 0; policy drop;

        # Allow traffic from alice's VPN IP to allowed networks
        ip saddr 10.8.0.5 ip daddr 192.168.1.0/24 accept
        ip saddr 10.8.0.5 ip daddr 10.0.0.10 tcp dport 443 accept

        # Drop all other traffic from this VPN IP
        ip saddr 10.8.0.5 drop
    }
}
```

## Database Schema

### Core Tables

- **users**: User accounts synced from IdP
- **sessions**: Active user sessions
- **certificates**: Issued certificates for revocation tracking
- **policies**: Access control policies
- **policy_rules**: Rules within policies
- **gateways**: Registered gateway nodes
- **connections**: Active and historical VPN connections
- **audit_logs**: Audit trail

### Mesh Networking Tables

- **mesh_hubs**: Mesh hub configurations and status
- **mesh_spokes**: Spoke gateways connected to hubs
- **mesh_hub_users**: User access assignments to hubs
- **mesh_hub_groups**: Group access assignments to hubs
- **mesh_hub_networks**: Network assignments to hubs (zero-trust)
- **mesh_spoke_users**: User access assignments to spokes
- **mesh_spoke_groups**: Group access assignments to spokes

### Entity Relationships

```
┌─────────┐      ┌──────────┐      ┌─────────────┐
│  users  │──────│ sessions │──────│ certificates│
└────┬────┘      └──────────┘      └─────────────┘
     │
     │      ┌──────────────┐      ┌─────────────┐
     └──────│ connections  │──────│  gateways   │
            └──────────────┘      └─────────────┘

┌──────────┐      ┌──────────────┐
│ policies │──────│ policy_rules │
└──────────┘      └──────────────┘
```

## Deployment Architecture

### Single Region

```
┌─────────────────────────────────────────────────────────────────┐
│                         Cloud Region                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────┐                   │
│  │  Load Balancer   │    │    Database      │                   │
│  │     (HTTPS)      │    │   (PostgreSQL)   │                   │
│  └────────┬─────────┘    └────────┬─────────┘                   │
│           │                       │                              │
│           ▼                       │                              │
│  ┌──────────────────┐            │                              │
│  │  Control Plane   │◄───────────┘                              │
│  │   (gatekey-server) │                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐    ┌──────────────────┐                   │
│  │  Gateway Node 1  │    │  Gateway Node 2  │                   │
│  │   (OpenVPN +     │    │   (OpenVPN +     │                   │
│  │    gatekey-gw)     │    │    gatekey-gw)     │                   │
│  └──────────────────┘    └──────────────────┘                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Multi-Region

```
┌────────────────────┐    ┌────────────────────┐
│     US-EAST        │    │     EU-WEST        │
├────────────────────┤    ├────────────────────┤
│                    │    │                    │
│ ┌────────────────┐ │    │ ┌────────────────┐ │
│ │ Control Plane  │◄├────┼─┤ Control Plane  │ │
│ │   (Primary)    │ │    │ │   (Replica)    │ │
│ └───────┬────────┘ │    │ └───────┬────────┘ │
│         │          │    │         │          │
│ ┌───────┴────────┐ │    │ ┌───────┴────────┐ │
│ │    Gateway     │ │    │ │    Gateway     │ │
│ └────────────────┘ │    │ └────────────────┘ │
│                    │    │                    │
└────────────────────┘    └────────────────────┘
```

## Technology Stack

### Backend
- **Language**: Go 1.25+
- **Web Framework**: Gin
- **Database**: PostgreSQL
- **Authentication**: OIDC (go-oidc), SAML (crewjam/saml)
- **Firewall**: nftables (google/nftables)

### Frontend
- **Framework**: React 18
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Bundler**: Vite

### Infrastructure
- **VPN**: OpenVPN (stock)
- **Container**: Docker (optional)
- **Orchestration**: Kubernetes (optional)

## See Also

- [Mesh Networking Guide](mesh-networking.md) - Hub-and-spoke VPN topology
- [Security Documentation](security.md) - Security model and best practices
- [API Reference](api.md) - REST API documentation
- [Client Guide](client.md) - CLI client usage
