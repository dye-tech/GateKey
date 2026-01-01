# GateKey System Compatibility

This document outlines platform support and feature availability for all GateKey binaries.

## Binary Overview

| Binary | Purpose | Server-side | Client-side |
|--------|---------|-------------|-------------|
| `gatekey` | VPN client for end users | No | Yes |
| `gatekey-server` | Control plane (API, UI, PKI) | Yes | No |
| `gatekey-gateway` | Gateway agent with firewall | Yes | No |
| `gatekey-hub` | Mesh hub with firewall | Yes | No |
| `gatekey-mesh-gateway` | Mesh spoke (connects to hub) | Yes | No |

## Platform Support Matrix

### gatekey (VPN Client)

The user-facing VPN client. Wraps OpenVPN for easy VPN connections.

| Platform | Architecture | Supported | Notes |
|----------|-------------|-----------|-------|
| Linux | amd64 | Yes | Full support |
| Linux | arm64 | Yes | Full support |
| macOS | amd64 (Intel) | Yes | Full support |
| macOS | arm64 (Apple Silicon) | Yes | Full support |
| Windows | amd64 | Planned | Coming soon |

**Dependencies**: OpenVPN client must be installed.

### gatekey-server (Control Plane)

Central management server. Runs the API, web UI, and PKI.

| Platform | Architecture | Supported | Notes |
|----------|-------------|-----------|-------|
| Linux | amd64 | Yes | Production recommended |
| Linux | arm64 | Yes | Full support |
| Docker/K8s | any | Yes | Recommended deployment |
| macOS | any | Dev only | Not for production |
| Windows | any | No | Not supported |

**Dependencies**: PostgreSQL database.

### gatekey-gateway (Gateway Agent)

Runs alongside OpenVPN on gateway servers. Provides per-client firewall enforcement.

| Platform | Architecture | Supported | Firewall | Notes |
|----------|-------------|-----------|----------|-------|
| Linux | amd64 | Yes | nftables | Full support |
| Linux | arm64 | Yes | nftables | Full support |
| macOS | any | No | N/A | No nftables support |
| Windows | any | No | N/A | Not supported |

**Dependencies**:
- OpenVPN server
- nftables (firewall enforcement)
- iptables (fallback, optional)

### gatekey-hub (Mesh Hub)

Mesh VPN hub server. Accepts spoke and client connections with zero-trust firewall.

| Platform | Architecture | Supported | Firewall | Notes |
|----------|-------------|-----------|----------|-------|
| Linux | amd64 | Yes | nftables | Full support |
| Linux | arm64 | Yes | nftables | Full support |
| macOS | any | No | N/A | No nftables support |
| Windows | any | No | N/A | Not supported |

**Dependencies**:
- OpenVPN server
- nftables (zero-trust firewall enforcement)
- iptables (optional)

### gatekey-mesh-gateway (Mesh Spoke)

Connects remote sites to a mesh hub. Runs as OpenVPN client.

| Platform | Architecture | Supported | Notes |
|----------|-------------|-----------|-------|
| Linux | amd64 | Yes | Full support |
| Linux | arm64 | Yes | Full support |
| macOS | any | No | Server-side only |
| Windows | any | No | Not supported |

**Dependencies**: OpenVPN client.

## Feature Availability by Platform

### Firewall Features

| Feature | Linux | macOS | Windows |
|---------|-------|-------|---------|
| nftables rules | Yes | No | No |
| Per-client firewall | Yes | No | No |
| Zero-trust enforcement | Yes | No | No |
| IP-based access rules | Yes | No | No |
| CIDR-based access rules | Yes | No | No |
| Hostname resolution | Yes | No | No |
| Port/protocol filtering | Yes | No | No |

### VPN Features

| Feature | Linux | macOS | Windows |
|---------|-------|-------|---------|
| OpenVPN connections | Yes | Yes | Planned |
| Multi-gateway support | Yes | Yes | Planned |
| Mesh hub connections | Yes | Yes | Planned |
| Auto-reconnect | Yes | Yes | Planned |
| Browser-based login | Yes | Yes | Planned |

## Recommended Deployment Configurations

### Production Gateway/Hub Server

```
OS: Linux (RHEL 9, Ubuntu 22.04+, Debian 12+, Rocky 9)
Architecture: amd64 or arm64
Firewall: nftables (installed automatically)
Resources: 2+ CPU cores, 2+ GB RAM
```

### Development/Testing

```
OS: Any Linux distribution
Architecture: amd64 or arm64
Can run control plane + gateway on same machine
```

### Client Workstations

```
OS: Linux, macOS (Windows coming soon)
Dependencies: OpenVPN client
No special permissions needed (except for VPN connection)
```

## Linux Distribution Support

### Fully Tested

- RHEL 9 / Rocky Linux 9 / AlmaLinux 9
- Ubuntu 22.04 LTS, 24.04 LTS
- Debian 12 (Bookworm)
- Fedora 39+
- Amazon Linux 2023

### Community Supported

- Arch Linux
- openSUSE Leap/Tumbleweed
- Alpine Linux (containers)

### Legacy (Limited Support)

- RHEL 8 / Rocky 8 / AlmaLinux 8
- Ubuntu 20.04 LTS
- Debian 11 (Bullseye)
- Amazon Linux 2

## Known Limitations

### macOS

- **No server-side binaries**: Gateway, hub, and mesh-spoke require Linux
- **No firewall enforcement**: nftables not available on macOS
- **Client only**: Only the `gatekey` client binary is supported

### Windows

- **Not yet supported**: Windows support is planned for the VPN client
- **Use WSL2**: For development, you can run server components in WSL2

### Containers

- **Privileged mode**: Gateway and hub containers need `NET_ADMIN` capability for firewall
- **Host networking**: Mesh components may require host networking for VPN tunnels

## Checking System Compatibility

Run the FIPS check command to verify system compatibility:

```bash
gatekey fips-check
```

This will show:
- OpenSSL/crypto library status
- Available ciphers
- System FIPS mode (if applicable)
- OpenVPN version and cipher support
