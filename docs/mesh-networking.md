# GateKey Mesh Networking Guide

This guide explains how to set up and manage hub-and-spoke mesh VPN networks with GateKey.

## Overview

Mesh networking enables secure site-to-site connectivity using a hub-and-spoke topology. Remote sites (spokes) connect outbound to a central hub, eliminating the need for inbound firewall rules on remote networks.

### Key Benefits

- **NAT Traversal**: Spokes initiate outbound connections, working through NAT/firewalls
- **Centralized Management**: All configuration managed from the GateKey control plane
- **Dynamic Routing**: Spokes advertise local networks; hub aggregates and distributes routes
- **Fine-Grained Access Control**: Control which users/groups can access specific networks
- **Client VPN Access**: Users can connect to the mesh network via OpenVPN client

## Architecture

```
                 ┌─────────────────┐
                 │  Control Plane  │
                 │   (GateKey UI)  │
                 └────────┬────────┘
                          │ API / Config Sync
                          ▼
                 ┌─────────────────┐
                 │   Mesh Hub      │◄── OpenVPN Server
                 │  (gatekey-hub)  │    Runs on public endpoint
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Spoke A   │  │   Spoke B   │  │   Spoke C   │
│  10.0.0.0/8 │  │ 192.168.0/24│  │ 172.16.0/16 │
└─────────────┘  └─────────────┘  └─────────────┘
  Home Lab         AWS VPC         Office Network
```

### Components

| Component | Binary | Description |
|-----------|--------|-------------|
| Control Plane | `gatekey-server` | Central management, API, and UI |
| Mesh Hub | `gatekey-hub` | OpenVPN server, route aggregation |
| Mesh Spoke | `gatekey-mesh-gateway` | Connects to hub, advertises local networks |

## Setting Up a Mesh Hub

### 1. Create the Hub

1. Navigate to **Administration → Mesh**
2. Click **Add Hub**
3. Configure the hub:
   - **Name**: Display name (e.g., "primary-hub")
   - **Public Endpoint**: Hostname or IP (e.g., `hub.example.com`)
   - **VPN Port**: OpenVPN port (default: 1194)
   - **VPN Protocol**: UDP (recommended) or TCP
   - **VPN Subnet**: Tunnel IP range (e.g., `172.30.0.0/16`)
   - **Crypto Profile**: FIPS, Modern, or Compatible
   - **TLS-Auth**: Enable for additional security (recommended)
   - **Full Tunnel Mode**: Route all client traffic through hub
   - **Push DNS**: Push DNS servers to connected clients
   - **DNS Servers**: Custom DNS servers (defaults to 1.1.1.1, 8.8.8.8)
   - **Local Networks**: Networks directly reachable from the hub (e.g., `192.168.1.0/24`)
4. Click **Create Hub**
5. **Save the API Token** - shown only once!

### 2. Install the Hub

Copy the install command from the hub details or use:

```bash
curl -sSL https://your-gatekey-server/scripts/install-hub.sh | sudo bash -s -- \
  --token "YOUR_HUB_TOKEN" \
  --control-plane "https://your-gatekey-server"
```

The installer will:
- Install OpenVPN and gatekey-hub binary
- Provision certificates from the control plane
- Configure the OpenVPN server
- Enable IP forwarding and firewall rules
- Start and enable systemd services

### 3. Verify Hub Status

Check the hub status in the web UI:
- **Online**: Hub is sending heartbeats
- **Pending**: Hub hasn't provisioned yet
- **Offline**: No heartbeat received

View logs on the hub server:
```bash
journalctl -u gatekey-hub -f
journalctl -u openvpn-server@mesh -f
```

## Setting Up Mesh Spokes

### 1. Create the Spoke

1. Navigate to **Administration → Mesh → Spokes**
2. Select the hub this spoke will connect to
3. Click **Add Spoke**
4. Configure the spoke:
   - **Name**: Identifier (e.g., "home-lab")
   - **Description**: Optional description
   - **Local Networks**: CIDR blocks behind this spoke (e.g., `10.0.0.0/8`)
5. Click **Create Spoke**
6. **Save the Spoke Token** - shown only once!

### 2. Install the Spoke

```bash
curl -sSL https://your-gatekey-server/scripts/install-mesh-spoke.sh | sudo bash -s -- \
  --token "YOUR_SPOKE_TOKEN" \
  --control-plane "https://your-gatekey-server"
```

### 3. Verify Spoke Status

Check the spoke status in the web UI:
- **Connected**: Spoke is connected to hub
- **Disconnected**: Not currently connected
- **Pending**: Hasn't connected yet

View logs on the spoke server:
```bash
journalctl -u gatekey-mesh-gateway -f
```

## Access Control (Zero-Trust)

GateKey implements a **zero-trust security model** for mesh networking. Access is denied by default - users only get routes to networks they are explicitly authorized to access.

### Hub Access Control

Hub access determines who can connect to the mesh network as a VPN client.

**Managing Hub Access:**
1. Go to **Mesh → Hubs**
2. Click the actions menu on a hub
3. Select **Manage Access**
4. Use tabs to manage:
   - **Users**: Individual user assignments
   - **Groups**: Group-based assignments
   - **Networks**: Network access rules (zero-trust)

Users without hub access:
- Cannot see the hub on the Connect page
- Cannot generate VPN configs for the mesh

### Hub Network Access (Zero-Trust)

The Networks tab controls which networks users can access through access rules. This implements zero-trust - routes are only pushed to clients if they have explicit access rules.

**How it works:**
1. Assign Networks to the hub (from the global Networks list)
2. Each Network contains Access Rules (CIDR definitions)
3. Assign Access Rules to Users or Groups
4. Only users with matching access rules receive routes

**Example:**
- Hub has "Home Lab" network assigned with access rule `192.168.50.0/23`
- User "alice" is assigned to this access rule
- When Alice connects, she receives a route to `192.168.50.0/23`
- User "bob" without this rule cannot reach that network

### Spoke Access Control

Spoke access determines who can route traffic to networks behind specific spokes. This enables network segmentation within the mesh.

**Managing Spoke Access:**
1. Go to **Mesh → Spokes**
2. Select a hub and find the spoke
3. Click the actions menu and select **Manage Access**
4. The modal shows the spoke's local networks
5. Add users or groups

**Use Case Example:**

A spoke advertises two networks:
- `10.0.0.0/24` - Production servers
- `10.0.1.0/24` - Development servers

You can:
- Assign the "DevOps" group to access both networks
- Assign the "Developers" group to access only development (via a spoke that only advertises dev)
- Exclude contractors from production access

### Access Control Flow (Zero-Trust)

```
User requests mesh VPN connection
        │
        ▼
┌───────────────────────────────┐
│ Is user assigned to hub?      │──NO──► Cannot generate config
└───────────────┬───────────────┘
                │ YES
                ▼
┌───────────────────────────────┐
│ Get networks assigned to hub  │
│ (via Networks tab)            │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│ Filter to access rules user   │
│ is assigned to (zero-trust)   │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│ Generate VPN config with      │
│ ONLY authorized routes        │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│ User connects to mesh         │
│ Can only reach networks from  │
│ spokes they have access to    │
└───────────────────────────────┘
```

## Client VPN Access

Users with hub access can connect to the mesh network as VPN clients.

### Generating a Config

1. Navigate to **Connect**
2. Switch to the **Mesh Networks** tab
3. Find the hub you have access to
4. Click **Download Config**
5. Save the `.ovpn` file

### Using the GateKey CLI

The GateKey CLI provides the easiest way to connect:

```bash
# List available mesh hubs
gatekey mesh list

# Connect to a mesh hub
gatekey connect --mesh hub-name

# Check status
gatekey status

# Disconnect
gatekey disconnect
```

The CLI automatically downloads configs and manages connections.

### Manual Connection (OpenVPN Client)

Import the config into any OpenVPN client:

**Linux:**
```bash
sudo openvpn --config mesh-hub-name.ovpn
```

**macOS (Tunnelblick):**
- Double-click the `.ovpn` file to import
- Click "Connect"

**Windows (OpenVPN GUI):**
- Copy the `.ovpn` file to `C:\Users\<username>\OpenVPN\config`
- Right-click the tray icon and connect

**iOS/Android:**
- Import the `.ovpn` file into OpenVPN Connect app

## API Reference

### Hub Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/mesh/hubs` | GET | List all hubs |
| `/api/v1/admin/mesh/hubs` | POST | Create new hub |
| `/api/v1/admin/mesh/hubs/:id` | GET | Get hub details |
| `/api/v1/admin/mesh/hubs/:id` | PUT | Update hub |
| `/api/v1/admin/mesh/hubs/:id` | DELETE | Delete hub |
| `/api/v1/admin/mesh/hubs/:id/provision` | POST | Trigger provision |
| `/api/v1/admin/mesh/hubs/:id/install-script` | GET | Get install script |

### Hub Access Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/mesh/hubs/:id/users` | GET | List hub users |
| `/api/v1/admin/mesh/hubs/:id/users` | POST | Add user to hub |
| `/api/v1/admin/mesh/hubs/:id/users/:userId` | DELETE | Remove user |
| `/api/v1/admin/mesh/hubs/:id/groups` | GET | List hub groups |
| `/api/v1/admin/mesh/hubs/:id/groups` | POST | Add group to hub |
| `/api/v1/admin/mesh/hubs/:id/groups/:groupName` | DELETE | Remove group |
| `/api/v1/admin/mesh/hubs/:id/networks` | GET | List hub networks (zero-trust) |
| `/api/v1/admin/mesh/hubs/:id/networks` | POST | Assign network to hub |
| `/api/v1/admin/mesh/hubs/:id/networks/:networkId` | DELETE | Remove network from hub |

### Spoke Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/mesh/hubs/:id/spokes` | GET | List spokes for hub |
| `/api/v1/admin/mesh/hubs/:id/spokes` | POST | Create spoke |
| `/api/v1/admin/mesh/spokes/:id` | GET | Get spoke details |
| `/api/v1/admin/mesh/spokes/:id` | PUT | Update spoke |
| `/api/v1/admin/mesh/spokes/:id` | DELETE | Delete spoke |
| `/api/v1/admin/mesh/spokes/:id/provision` | POST | Trigger provision |
| `/api/v1/admin/mesh/spokes/:id/install-script` | GET | Get install script |

### Spoke Access Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/mesh/spokes/:id/users` | GET | List spoke users |
| `/api/v1/admin/mesh/spokes/:id/users` | POST | Add user to spoke |
| `/api/v1/admin/mesh/spokes/:id/users/:userId` | DELETE | Remove user |
| `/api/v1/admin/mesh/spokes/:id/groups` | GET | List spoke groups |
| `/api/v1/admin/mesh/spokes/:id/groups` | POST | Add group to spoke |
| `/api/v1/admin/mesh/spokes/:id/groups/:groupName` | DELETE | Remove group |

### User Mesh Access

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/mesh/hubs` | GET | List hubs user can access |
| `/api/v1/mesh/generate-config` | POST | Generate client VPN config |

## Troubleshooting

### Hub Won't Come Online

1. Check the hub service:
   ```bash
   systemctl status gatekey-hub
   ```

2. Verify control plane connectivity:
   ```bash
   curl -I https://your-gatekey-server/health
   ```

3. Check the API token is correct in `/etc/gatekey/hub.yaml`

4. View logs:
   ```bash
   journalctl -u gatekey-hub -f
   ```

### Spoke Won't Connect

1. Check the spoke service:
   ```bash
   systemctl status gatekey-mesh-gateway
   ```

2. Verify hub is reachable:
   ```bash
   nc -vuz hub.example.com 1194  # For UDP
   nc -vz hub.example.com 1194   # For TCP
   ```

3. Check firewall allows outbound to hub

4. View logs:
   ```bash
   journalctl -u gatekey-mesh-gateway -f
   ```

### Routes Not Working

1. Verify spoke's local networks are correct in the UI

2. Check IP forwarding on hub and spokes:
   ```bash
   cat /proc/sys/net/ipv4/ip_forward  # Should be 1
   ```

3. Check routing table:
   ```bash
   ip route show
   ```

4. Verify firewall allows forwarded traffic:
   ```bash
   iptables -L FORWARD -v -n
   ```

### Client Can't Access Spoke Networks

1. Verify user has hub access (check Manage Access modal → Users/Groups tabs)

2. **Zero-Trust Check**: Verify network is assigned to hub (Manage Access → Networks tab)

3. **Access Rules Check**: Verify user has access rules for the target network:
   - Go to Administration → Access Rules
   - Check user/group is assigned to rules covering the target CIDR

4. Check client received correct routes:
   ```bash
   ip route show | grep tun
   ```

5. Test connectivity step by step:
   ```bash
   ping <hub-tunnel-ip>    # Should work
   ping <spoke-tunnel-ip>  # Should work if routes ok
   ping <spoke-network-ip> # Should work if access rules granted
   ```

### No Routes Pushed to Client

If connecting successfully but no routes appear:

1. Check networks are assigned to hub (Manage Access → Networks tab)
2. Check user has access rules within those networks
3. Regenerate the VPN config after making access rule changes

## VPN Settings

### Full Tunnel Mode

When enabled, all client traffic is routed through the VPN (0.0.0.0/0 route). This is useful for:
- Enforcing traffic inspection
- Hiding client's real IP
- Accessing internet through hub's network

**To enable:**
1. When creating a hub, check "Full Tunnel Mode"
2. Or edit an existing hub and enable the setting

### DNS Settings

Push DNS servers to connected clients to ensure name resolution works correctly.

**Configuration:**
1. Enable "Push DNS servers to clients"
2. Add custom DNS servers (e.g., 1.1.1.1, 8.8.8.8)
3. If no servers specified, defaults to 1.1.1.1 and 8.8.8.8

**Use cases:**
- Internal DNS resolution for private domains
- DNS-based ad blocking (Pi-hole, AdGuard)
- Split-horizon DNS scenarios

### Hub Local Networks

Networks directly reachable from the hub server (not via spokes). Configure these in the hub settings to make them available to clients.

**Example:** If the hub is on 192.168.1.0/24 and you want clients to reach that network, add it to the hub's local networks.

## Security Considerations

1. **Token Security**: Hub and spoke tokens provide full provisioning access. Rotate if compromised.

2. **Network Segmentation**: Use spoke access control to limit which users can reach which networks.

3. **Encryption**: Use FIPS or Modern crypto profiles. Avoid Compatible unless required for legacy clients.

4. **TLS-Auth**: Enable TLS-Auth for an additional HMAC authentication layer.

5. **Certificate Lifetime**: Client certificates are short-lived (24 hours default) to limit exposure.

6. **Audit Logging**: All access changes and connections are logged for compliance.

7. **Zero-Trust Access**: Networks must be explicitly assigned to hubs, and users must have access rules to receive routes.
