# WireGuard VPN Support

GateKey supports WireGuard as an alternative VPN protocol alongside OpenVPN. WireGuard gateways offer a modern, high-performance VPN option with simpler configuration and lower overhead.

## Overview

WireGuard is a modern VPN protocol that offers:
- **Faster Performance**: Lower latency and higher throughput than OpenVPN
- **Simpler Configuration**: Single `.conf` file format
- **Smaller Codebase**: Easier to audit and more secure
- **Modern Cryptography**: Curve25519, ChaCha20, Poly1305, BLAKE2s
- **UDP Only**: Optimized for performance (no TCP fallback)

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                      CONTROL PLANE                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    WireGuard Module                       │   │
│  ├──────────────┬──────────────┬──────────────┬─────────────┤   │
│  │  Key Gen     │  Config Gen  │  Peer Mgmt   │  IP Alloc   │   │
│  │ (Curve25519) │ (.conf file) │ (sync peers) │ (CIDR pool) │   │
│  └──────────────┴──────────────┴──────────────┴─────────────┘   │
│                             │                                    │
│  ┌──────────────────────────┴───────────────────────────────┐   │
│  │                    Database Tables                        │   │
│  ├───────────────────────────────────────────────────────────┤   │
│  │  gateways (gateway_type='wireguard', wg_public_key, ...)  │   │
│  │  wireguard_configs (client configs, keys, assigned IPs)   │   │
│  │  wireguard_peers (active connections, handshake status)   │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                 WIREGUARD GATEWAY NODE                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────────────────────────┐ │
│  │   WireGuard      │  │    GateKey WireGuard Gateway Agent   │ │
│  │   Interface      │◄─┤      (gatekey-wireguard-gateway)     │ │
│  │   (wg0)          │  └────────────────┬─────────────────────┘ │
│  └────────┬─────────┘                   │                       │
│           │                              │                       │
│           │ Peer Management              │ API Calls             │
│           │ (wg set, wg show)            │                       │
│           │                              │                       │
│  ┌────────┴──────────────────────────────┴──────────────────┐   │
│  │              Firewall Manager (nftables)                  │   │
│  │         Per-peer rules, zero-trust enforcement            │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Key Differences from OpenVPN

| Feature | OpenVPN | WireGuard |
|---------|---------|-----------|
| Protocol | UDP or TCP | UDP only |
| Port | 1194 (default) | 51820 (default) |
| Cryptography | Configurable (AES, ChaCha20) | Fixed (Curve25519, ChaCha20-Poly1305) |
| Client Config | `.ovpn` file | `.conf` file |
| Key Type | X.509 Certificates | Curve25519 Key Pairs |
| Connection | Certificate-based | Public Key-based |
| Binary | `gatekey-gateway` | `gatekey-wireguard-gateway` |

## Quick Start

### 1. Create a WireGuard Gateway

1. Navigate to **Administration → Gateways**
2. Click **Add Gateway**
3. Select **WireGuard** as the gateway type
4. Fill in the gateway details:
   - **Name**: Unique identifier (e.g., `wg-us-east-1`)
   - **Hostname**: Public DNS name (e.g., `wg.vpn.example.com`)
   - **WireGuard Port**: Default 51820
   - **VPN Subnet**: Client IP range (default: 172.31.255.0/24)
   - **Full Tunnel Mode**: Route all traffic through VPN (optional)
   - **Push DNS**: Push DNS settings to clients (optional)
5. Click **Register Gateway**
6. **Save the authentication token** - it will only be shown once!

### 2. Install the WireGuard Gateway

Run the installer script on your gateway server:

```bash
curl -sSL https://your-gatekey-server/scripts/install-wireguard-gateway.sh | sudo bash -s -- \
  --server https://your-gatekey-server \
  --token YOUR_GATEWAY_TOKEN \
  --name wg-gateway-name
```

Or download and run manually:

```bash
# Download the installer
curl -sSL https://your-gatekey-server/scripts/install-wireguard-gateway.sh -o install-wireguard-gateway.sh
chmod +x install-wireguard-gateway.sh

# Run with your configuration
sudo ./install-wireguard-gateway.sh \
  --server https://your-gatekey-server \
  --token YOUR_GATEWAY_TOKEN \
  --name wg-gateway-name
```

### 3. Verify Gateway Status

```bash
# Check WireGuard interface
sudo wg show wg0

# Check gateway agent status
sudo systemctl status gatekey-wireguard-gateway

# View gateway agent logs
sudo journalctl -u gatekey-wireguard-gateway -f
```

## Client Configuration

### Generating a WireGuard Config

1. Navigate to **Connect** in the web UI
2. Select a WireGuard gateway (marked with purple **WG** badge)
3. Click **Connect**
4. Download the `.conf` file

### Using the Configuration

#### Linux (wg-quick)

```bash
# Install WireGuard tools
sudo apt install wireguard-tools  # Debian/Ubuntu
sudo dnf install wireguard-tools  # Fedora/RHEL

# Import and start the connection
sudo wg-quick up /path/to/gatekey-wg-gateway.conf

# Check status
sudo wg show

# Disconnect
sudo wg-quick down /path/to/gatekey-wg-gateway.conf
```

#### macOS

1. Install the [WireGuard app](https://apps.apple.com/us/app/wireguard/id1451685025) from the App Store
2. Open WireGuard
3. Click **Import Tunnel(s) from File**
4. Select your `.conf` file
5. Click **Activate** to connect

#### Windows

1. Download [WireGuard for Windows](https://www.wireguard.com/install/)
2. Open WireGuard
3. Click **Import tunnel(s) from file**
4. Select your `.conf` file
5. Click **Activate** to connect

#### iOS

1. Install [WireGuard](https://apps.apple.com/us/app/wireguard/id1441195209) from the App Store
2. Open WireGuard
3. Tap **+** → **Create from QR code** or **Create from file or archive**
4. Import your configuration
5. Toggle the tunnel on

#### Android

1. Install [WireGuard](https://play.google.com/store/apps/details?id=com.wireguard.android) from Google Play
2. Open WireGuard
3. Tap **+** → **Import from file or archive** or **Scan from QR code**
4. Import your configuration
5. Toggle the tunnel on

## Configuration File Format

WireGuard configs generated by GateKey follow this format:

```ini
# GateKey WireGuard Configuration
# Gateway: production-gateway
# User: user@example.com
# Generated: 2026-01-05T12:00:00Z
# Expires: 2026-01-06T12:00:00Z
#
# WARNING: This configuration expires at 2026-01-06T12:00:00Z
# After expiration, you must generate a new configuration.

[Interface]
PrivateKey = <client-private-key>
Address = 172.31.255.5/32
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = <gateway-public-key>
Endpoint = wg.vpn.example.com:51820
AllowedIPs = 10.0.0.0/8, 192.168.0.0/16
PresharedKey = <optional-psk>
PersistentKeepalive = 25
```

### Configuration Fields

| Field | Description |
|-------|-------------|
| `PrivateKey` | Client's private key (generated by server) |
| `Address` | Client's VPN IP address |
| `DNS` | DNS servers to use (optional) |
| `PublicKey` | Gateway's public key |
| `Endpoint` | Gateway hostname:port |
| `AllowedIPs` | Networks routed through VPN (based on access rules) |
| `PresharedKey` | Optional additional encryption layer |
| `PersistentKeepalive` | NAT keepalive interval (25 seconds) |

## Gateway Agent Configuration

The WireGuard gateway agent is configured via `/etc/gatekey/wireguard-gateway.yaml`:

```yaml
# Gateway name (must match registered name in control plane)
name: "wg-gateway-name"

# Control plane URL
control_plane_url: "https://gatekey.example.com"

# Gateway authentication token
token: "your-token-here"

# Heartbeat interval (how often to report status)
heartbeat_interval: "30s"

# Peer sync interval (how often to fetch authorized peers)
peer_sync_interval: "10s"

# Stats report interval (how often to report traffic stats)
stats_report_interval: "5s"

# Log level: debug, info, warn, error
log_level: "info"

# WireGuard interface settings
wireguard:
  interface: "wg0"
  listen_port: 51820
  config_path: "/etc/wireguard/wg0.conf"

# Firewall configuration
firewall:
  backend: "nftables"
  chain: "GATEX"
  table: "gatex"
  default_policy: "drop"

# Network configuration
network:
  interface: "wg0"
  network: "172.31.255.0/24"
```

## Gateway Agent Operations

The WireGuard gateway agent performs these operations:

### Heartbeat Loop (every 30s)
- Reports gateway status to control plane
- Sends current peer count and interface status
- Receives configuration updates

### Peer Sync Loop (every 10s)
- Fetches authorized peer list from control plane
- Adds new peers to WireGuard interface
- Removes expired/revoked peers
- Updates firewall rules per peer

### Stats Report Loop (every 5s)
- Collects peer statistics from `wg show`
- Reports bytes sent/received per peer
- Reports last handshake time
- Updates peer connection status

## API Endpoints

### User Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/wireguard/configs/generate` | Generate WireGuard config |
| `GET /api/v1/wireguard/configs/download/:id` | Download .conf file |

### Admin Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/admin/wireguard/configs` | List all WireGuard configs |
| `DELETE /api/v1/admin/wireguard/configs/:id` | Revoke a config |

### Gateway Agent Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/wireguard-gateway/provision` | Provision gateway (get keys, config) |
| `POST /api/v1/wireguard-gateway/heartbeat` | Send heartbeat status |
| `POST /api/v1/wireguard-gateway/sync-peers` | Get authorized peers |
| `POST /api/v1/wireguard-gateway/peer-stats` | Report peer statistics |

## Security Considerations

### Key Management

- **Server-Generated Keys**: The control plane generates all key pairs
- **Private Key in Config**: Client private key is embedded in the `.conf` file
- **No Key Storage on Client**: Keys exist only in the config file
- **Key Rotation**: Generate a new config to get new keys

### Preshared Keys (PSK)

Optionally, GateKey can generate a preshared key for each client:
- Provides post-quantum security layer
- Adds symmetric encryption on top of Curve25519
- Stored with the client config in the database

### Config Expiration

WireGuard configs have the same expiration as OpenVPN configs:
- Default: 24 hours (configurable)
- After expiration, config generation required
- Expired peers are automatically removed from gateway

### Zero-Trust Enforcement

Like OpenVPN gateways, WireGuard gateways enforce zero-trust:
1. **Peer Validation**: Only peers in the authorized list can connect
2. **Firewall Rules**: nftables rules enforce per-peer access
3. **Dynamic Routes**: AllowedIPs based on user's access rules
4. **Real-Time Revocation**: Revoked configs are removed within 10 seconds

## Troubleshooting

### Gateway Shows as Offline

1. Check the gateway agent is running:
   ```bash
   systemctl status gatekey-wireguard-gateway
   ```

2. Verify network connectivity to control plane:
   ```bash
   curl -s https://your-gatekey-server/health
   ```

3. Check logs for errors:
   ```bash
   journalctl -u gatekey-wireguard-gateway -f
   ```

### Connection Not Establishing

1. Verify WireGuard interface is up:
   ```bash
   sudo wg show wg0
   ```

2. Check if peer is authorized:
   ```bash
   sudo wg show wg0 peers
   ```

3. Verify firewall allows UDP 51820:
   ```bash
   sudo nft list ruleset | grep 51820
   ```

### No Handshake

If `latest handshake` is empty or very old:

1. Check client can reach gateway:
   ```bash
   nc -zvu gateway.example.com 51820
   ```

2. Verify client config is not expired (check comments in `.conf`)

3. Check for NAT issues (PersistentKeepalive should be 25)

### Traffic Not Flowing

1. Verify AllowedIPs in config match intended destinations

2. Check gateway firewall rules:
   ```bash
   sudo nft list table inet gatex
   ```

3. Ensure IP forwarding is enabled:
   ```bash
   cat /proc/sys/net/ipv4/ip_forward
   ```

## Comparison: When to Use WireGuard vs OpenVPN

### Use WireGuard When:
- Performance is critical (lower latency, higher throughput)
- UDP access is reliable (not blocked by firewalls)
- Simpler configuration is preferred
- Mobile users (better roaming, faster reconnection)

### Use OpenVPN When:
- UDP is blocked (TCP fallback needed)
- Legacy client support required
- Need configurable cryptography
- Existing OpenVPN infrastructure

## WireGuard Mesh Networking

GateKey supports WireGuard for mesh (hub-and-spoke) networking, providing a high-performance alternative to OpenVPN mesh.

### Architecture

```
                 ┌─────────────────┐
                 │  Control Plane  │
                 │   (GateKey UI)  │
                 └────────┬────────┘
                          │ API / Config Sync
                          ▼
                 ┌─────────────────┐
                 │ WireGuard Hub   │◄── WireGuard Server
                 │(gatekey-wg-hub) │    Runs on public endpoint
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  WG Spoke A │  │  WG Spoke B │  │  WG Spoke C │
│  10.0.0.0/8 │  │ 192.168.0/24│  │ 172.16.0/16 │
└─────────────┘  └─────────────┘  └─────────────┘
  Home Lab         AWS VPC         Office Network
```

### Mesh Binaries

| Binary | Description |
|--------|-------------|
| `gatekey-wireguard-hub` | WireGuard mesh hub server - accepts spoke and client connections |
| `gatekey-wireguard-mesh-gateway` | WireGuard mesh spoke - connects to hub from remote sites |

### WireGuard vs OpenVPN Mesh

| Feature | OpenVPN Mesh | WireGuard Mesh |
|---------|--------------|----------------|
| Protocol | UDP or TCP | UDP only |
| Performance | Good | Excellent (lower latency) |
| Configuration | Complex | Simple |
| NAT Traversal | Works | Better (built-in keepalive) |
| Binary (Hub) | `gatekey-hub` | `gatekey-wireguard-hub` |
| Binary (Spoke) | `gatekey-mesh-gateway` | `gatekey-wireguard-mesh-gateway` |

### Setting Up a WireGuard Mesh Hub

1. Navigate to **Administration → Mesh**
2. Click **Add Hub** and configure:
   - **Name**: Display name for the hub
   - **Hub Type**: Select **WireGuard**
   - **Public Endpoint**: Hostname or IP where spokes will connect
   - **WireGuard Port**: Default 51820
   - **VPN Subnet**: Tunnel IP range (e.g., 172.30.0.0/16)
   - **Full Tunnel Mode**: Route all client traffic through hub
   - **Push DNS**: Push DNS servers to connected clients
   - **Local Networks**: Networks directly reachable from the hub
3. **Save the API Token** - shown only once at creation
4. Deploy the hub:

```bash
# Download and install
curl -sSL https://your-gatekey-server/scripts/install-wireguard-hub.sh | sudo bash -s -- \
  --server https://your-gatekey-server \
  --token YOUR_HUB_TOKEN \
  --name your-hub-name
```

### Adding WireGuard Mesh Spokes

1. In the **Mesh** page, switch to the **Spokes** tab
2. Select the WireGuard hub this spoke will connect to
3. Click **Add Spoke** and configure:
   - **Name**: Identifier for this spoke (e.g., "home-lab")
   - **Description**: Optional description
   - **Local Networks**: CIDR blocks behind this spoke (e.g., 10.0.0.0/8)
4. **Save the Spoke Token** - shown only once
5. Deploy the spoke:

```bash
# Download and install
curl -sSL https://your-gatekey-server/scripts/install-wireguard-mesh-gateway.sh | sudo bash -s -- \
  --server https://your-gatekey-server \
  --token YOUR_SPOKE_TOKEN \
  --name your-spoke-name
```

### Hub Configuration

The WireGuard hub agent is configured via `/etc/gatekey-wireguard-hub/config.yaml`:

```yaml
# Hub name (must match registered name in control plane)
name: "wg-mesh-hub"

# Control plane URL
control_plane_url: "https://gatekey.example.com"

# Hub authentication token
token: "your-token-here"

# Heartbeat interval
heartbeat_interval: "30s"

# Peer sync interval
peer_sync_interval: "10s"

# WireGuard interface settings
wireguard:
  interface: "wg0"
  listen_port: 51820
  config_path: "/etc/wireguard/wg0.conf"

# Firewall configuration
firewall:
  backend: "nftables"
  chain: "GATEX"
  table: "gatex"
  default_policy: "drop"
```

### Spoke Configuration

The WireGuard spoke agent is configured via `/etc/gatekey-wireguard-mesh-gateway/config.yaml`:

```yaml
# Spoke name (must match registered name in control plane)
name: "wg-spoke-homelab"

# Control plane URL
control_plane_url: "https://gatekey.example.com"

# Spoke authentication token
token: "your-token-here"

# Heartbeat interval
heartbeat_interval: "30s"

# WireGuard interface settings
wireguard:
  interface: "wg0"
  config_path: "/etc/wireguard/wg0.conf"

# Local networks to advertise
local_networks:
  - "10.0.0.0/8"
  - "192.168.1.0/24"
```

### Hub Agent Operations

| Operation | Interval | Description |
|-----------|----------|-------------|
| Heartbeat | 30s | Reports hub status, receives config updates |
| Peer Sync | 10s | Fetches authorized spokes and clients |
| Stats Report | 5s | Reports peer bandwidth and handshake status |

### Spoke Agent Operations

| Operation | Interval | Description |
|-----------|----------|-------------|
| Heartbeat | 30s | Reports spoke status to control plane |
| Connection | Persistent | Maintains WireGuard connection to hub |
| Keepalive | 25s | Sends keepalive packets for NAT traversal |

### Verifying Mesh Status

#### On the Hub

```bash
# Check WireGuard interface
sudo wg show wg0

# Check hub agent status
sudo systemctl status gatekey-wireguard-hub

# View connected peers (spokes + clients)
sudo wg show wg0 peers
```

#### On a Spoke

```bash
# Check WireGuard connection to hub
sudo wg show wg0

# Check spoke agent status
sudo systemctl status gatekey-wireguard-mesh-gateway

# Verify route to hub networks
ip route | grep wg0
```

### Troubleshooting Mesh

#### Spoke Not Connecting

1. Verify spoke can reach hub endpoint:
   ```bash
   nc -zvu hub.example.com 51820
   ```

2. Check spoke agent logs:
   ```bash
   journalctl -u gatekey-wireguard-mesh-gateway -f
   ```

3. Verify spoke is registered in control plane (Mesh → Spokes)

#### No Handshake Between Hub and Spoke

1. Check keys match (hub should have spoke's public key):
   ```bash
   sudo wg show wg0 peers
   ```

2. Verify firewall allows UDP 51820 on hub

3. Check for NAT issues (PersistentKeepalive should be 25)

#### Traffic Not Flowing Between Sites

1. Verify spoke's local networks are advertised:
   ```bash
   ip route | grep wg0
   ```

2. Check hub firewall rules:
   ```bash
   sudo nft list table inet gatex
   ```

3. Ensure IP forwarding is enabled on hub:
   ```bash
   cat /proc/sys/net/ipv4/ip_forward
   ```

## See Also

- [Gateway Setup Guide](gateway-setup.md) - OpenVPN gateway setup
- [Mesh Networking](mesh-networking.md) - Hub-and-spoke topology
- [Security Documentation](security.md) - Security model
- [Client Guide](client.md) - CLI client usage
