# GateKey Gateway Setup Guide

This guide explains how to register and deploy GateKey VPN gateways.

## Overview

A GateKey gateway consists of:
1. **OpenVPN Server** - Standard OpenVPN server handling VPN connections
2. **GateKey Gateway Agent** - Agent that integrates OpenVPN with the GateKey control plane

The gateway agent:
- Sends heartbeats to the control plane
- Validates client connections via hook scripts
- Reports connection/disconnection events
- Can apply per-identity firewall rules

## Prerequisites

- Root/sudo access on the gateway server
- Outbound HTTPS access to the GateKey control plane
- Inbound access on the VPN port (default: UDP 1194)
- Ubuntu 20.04+, Debian 11+, RHEL 8+, or Fedora 35+

## Quick Start

### 1. Register the Gateway

1. Log in to the GateKey web UI as an administrator
2. Navigate to **Admin > Gateways**
3. Click **Add Gateway**
4. Fill in the gateway details:
   - **Name**: Unique identifier (e.g., `us-east-1`)
   - **Hostname**: Public DNS name (e.g., `vpn-us-east.example.com`)
   - **Public IP**: (optional) Will be auto-detected
   - **VPN Port**: Default 1194
   - **Protocol**: UDP or TCP
   - **Crypto Profile**: Modern (default), FIPS, or Compatible
   - **VPN Subnet**: Client IP range (default: 172.31.255.0/24)
   - **TLS Auth**: Enable/disable TLS-Auth for additional security
   - **Full Tunnel Mode**: Route all traffic through VPN (default: disabled)
   - **Push DNS Servers**: Push DNS settings to clients (default: disabled)
   - **DNS Servers**: Custom DNS servers to push (e.g., `1.1.1.1, 8.8.8.8`)
5. Click **Register Gateway**
6. **Save the authentication token** - it will only be shown once!

#### Crypto Profiles

| Profile | Ciphers | Description |
|---------|---------|-------------|
| **Modern** | AES-256-GCM, CHACHA20-POLY1305 | Best performance and security |
| **FIPS** | AES-256-GCM, AES-128-GCM | FIPS 140-3 compliance requirements |
| **Compatible** | AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC | Legacy OpenVPN 2.3.x client support |

### 2. Install the Gateway

Run the installer script on your gateway server:

```bash
curl -sSL https://your-gatekey-server/scripts/install-gateway.sh | sudo bash -s -- \
  --server https://your-gatekey-server \
  --token YOUR_GATEWAY_TOKEN \
  --name gateway-name
```

Or download and run manually:

```bash
# Download the installer
curl -sSL https://your-gatekey-server/scripts/install-gateway.sh -o install-gateway.sh
chmod +x install-gateway.sh

# Run with your configuration
sudo ./install-gateway.sh \
  --server https://your-gatekey-server \
  --token YOUR_GATEWAY_TOKEN \
  --name gateway-name
```

### 3. Automatic Certificate Provisioning

The installer automatically provisions certificates from the GateKey control plane:

1. **CA Certificate** - Retrieved from the control plane's embedded CA
2. **Server Certificate** - Issued by the control plane for this gateway
3. **Server Key** - Generated and returned by the control plane
4. **DH Parameters** - Pre-generated and included

All certificates are stored in `/etc/openvpn/server/` and the OpenVPN server is configured automatically.

**What the installer does:**
- Installs OpenVPN and the GateKey gateway agent
- Provisions certificates from the control plane API
- Configures OpenVPN with the selected crypto profile
- Sets up NAT/masquerade for VPN traffic forwarding
- Enables IP forwarding
- Starts and enables all services
- Verifies everything is running

### 4. Start Services

```bash
# Start OpenVPN
sudo systemctl start openvpn-server@server

# Check gateway agent status
sudo systemctl status gatekey-gateway

# View gateway agent logs
sudo journalctl -u gatekey-gateway -f
```

## Kubernetes Deployment

For Kubernetes deployments, use the provided manifests:

```bash
# Navigate to deploy directory
cd deploy/gateway

# Customize the configuration
# 1. Edit configmap.yaml with your settings
# 2. Edit secret.yaml with your token and certificates

# Apply the manifests
kubectl apply -k .
```

### Required Secrets

Create the gateway secret with your actual values:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gatekey-gateway-secret
  namespace: gatekey
type: Opaque
stringData:
  gateway.yaml: |
    control_plane_url: "https://gatekey.example.com"
    token: "YOUR_ACTUAL_TOKEN"
    heartbeat_interval: "30s"
    log_level: "info"
```

Create the PKI secret with your certificates:

```bash
kubectl create secret generic gatekey-gateway-pki \
  --from-file=ca.crt=./ca.crt \
  --from-file=server.crt=./server.crt \
  --from-file=server.key=./server.key \
  --from-file=dh.pem=./dh.pem \
  -n gatekey
```

## Configuration Options

### Gateway Agent Configuration

The gateway agent is configured via `/etc/gatekey/gateway.yaml`:

```yaml
# Control plane URL
control_plane_url: "https://gatekey.example.com"

# Gateway authentication token
token: "your-token-here"

# Heartbeat interval (how often to report status)
heartbeat_interval: "30s"

# Log level: debug, info, warn, error
log_level: "info"
```

### Environment Variables

The gateway agent supports environment variables with the `gatekey_` prefix:

- `gatekey_CONTROL_PLANE_URL` - Control plane URL
- `gatekey_TOKEN` - Gateway authentication token
- `PUBLIC_IP` - Public IP address (auto-detected if not set)

### Installer Options

```
--server URL      GateKey control plane URL (required)
--token TOKEN     Gateway authentication token (required)
--name NAME       Gateway name (required)
--port PORT       OpenVPN port (default: 1194)
--protocol PROTO  Protocol: udp or tcp (default: udp)
--network CIDR    VPN network CIDR (default: 172.31.255.0/24)
```

## Monitoring

### Check Gateway Status

In the GateKey web UI:
1. Go to **Admin > Gateways**
2. View the status column:
   - **Online** - Gateway is active and sending heartbeats
   - **Offline** - No heartbeat received recently

### View Logs

```bash
# Gateway agent logs
sudo journalctl -u gatekey-gateway -f

# OpenVPN logs
sudo journalctl -u openvpn-server@server -f

# Combined view
sudo journalctl -u gatekey-gateway -u openvpn-server@server -f
```

### Troubleshooting

**Gateway shows as Offline:**
1. Check the gateway agent is running: `systemctl status gatekey-gateway`
2. Verify network connectivity to control plane
3. Check logs for errors: `journalctl -u gatekey-gateway`

**OpenVPN connections failing:**
1. Verify OpenVPN is running: `systemctl status openvpn-server@server`
2. Check certificate configuration
3. Verify firewall allows VPN port

**Hook scripts failing:**
1. Check the gateway agent logs for hook errors
2. Verify the hook scripts are executable
3. Test manually: `/usr/local/bin/gatekey-gateway hook --type auth-user-pass-verify`

## Security Considerations

1. **Token Security**: The gateway token provides full access to the gateway API. Keep it secure and rotate if compromised.

2. **Network Segmentation**: Consider placing the control plane API on a private network accessible only to gateways.

3. **Certificate Management**: Use short-lived certificates from the GateKey CA for clients.

4. **Firewall Rules**: The gateway agent can apply per-identity firewall rules using nftables/iptables.

## Real-Time Firewall Enforcement

The gateway agent enforces access rules in real-time using nftables:

### How It Works

1. **Client Connects**: Gateway agent writes client info to `/var/run/gatekey/clients/`
2. **Rules Fetched**: Agent calls `POST /api/v1/gateway/client-rules` to get allowed destinations
3. **Firewall Applied**: nftables rules are created with default DENY policy
4. **Periodic Refresh**: Every 10 seconds, agent checks for rule changes
5. **Immediate Update**: When rules change, firewall is updated without client reconnection
6. **Client Disconnects**: Firewall rules are removed automatically

### Rule Refresh Behavior

When an administrator adds or removes an access rule:
- Gateway agent detects the change within 10 seconds
- nftables rules are updated for all connected clients
- Client traffic is immediately blocked/allowed based on new rules
- No VPN reconnection required

### Configuration

```yaml
# /etc/gatekey/gateway.yaml
rule_refresh_interval: "10s"  # How often to check for rule changes
```

## Push-Based Configuration Updates

GateKey supports automatic configuration updates via a push mechanism. When you change gateway settings in the control plane, the gateway automatically detects the change and reprovisions itself.

### How It Works

1. **Config Versioning**: The control plane computes a SHA256 hash of critical gateway settings (crypto profile, VPN port, protocol, subnet, TLS-Auth settings)
2. **Heartbeat Check**: Gateway sends its current config version in each heartbeat
3. **Version Comparison**: Server compares versions and responds with `needs_reprovision: true` if they differ
4. **Auto-Reprovision**: Gateway automatically fetches new certificates/config and restarts OpenVPN

### Settings That Trigger Reprovision

When you change any of these settings, the gateway will automatically update:

| Setting | Description |
|---------|-------------|
| **Crypto Profile** | Changing between modern, fips, or compatible |
| **VPN Port** | Changing the OpenVPN listen port |
| **VPN Protocol** | Switching between UDP and TCP |
| **VPN Subnet** | Changing the client IP address range |
| **TLS-Auth** | Enabling/disabling TLS-Auth or rotating the key |
| **Full Tunnel Mode** | Switching between split tunnel and full tunnel |
| **Push DNS** | Enabling/disabling DNS server pushing |
| **DNS Servers** | Changing the list of DNS servers to push |

### Timeline

- Changes are detected within the heartbeat interval (default: 30 seconds)
- Reprovision typically completes in 5-10 seconds
- OpenVPN restarts automatically to apply new config
- Connected clients will need to reconnect after OpenVPN restart

### Manual Reprovision

You can also trigger a manual reprovision by restarting the gateway agent:

```bash
sudo systemctl restart gatekey-gateway
```

On startup, if the gateway has no stored config version, it will provision from the control plane.

## TLS-Auth Configuration

TLS-Auth provides an additional layer of security by requiring a shared static key.

### Enabling TLS-Auth

1. When registering a gateway, check **Enable TLS Authentication**
2. During gateway provisioning, a TLS-Auth key is generated and:
   - Stored in the control plane database
   - Sent to the gateway for OpenVPN configuration
   - Included in generated client configurations

### Disabling TLS-Auth

Disable TLS-Auth for simpler configurations (e.g., direct IP connections):

1. Edit the gateway in **Admin > Gateways**
2. Uncheck **Enable TLS Authentication**
3. The gateway will automatically reprovision without TLS-Auth

### Key Storage

- **Server Key**: Stored at `/etc/openvpn/server/ta.key` on the gateway
- **Client Key**: Embedded in generated `.ovpn` configuration files
- **Database**: TLS-Auth key is stored in the `gateways.tls_auth_key` column

## Tunnel Modes

GateKey supports two tunnel modes, configurable per-gateway:

### Split Tunnel (Default)

By default, gateways operate in split tunnel mode:
- Only routes traffic for networks the user has access to
- Routes are pushed **dynamically** based on user's access rules
- Internet traffic uses the client's normal connection
- DNS is NOT overridden

This is the recommended mode for most deployments as it:
- Reduces VPN bandwidth usage
- Allows users to access local resources
- Only secures traffic that needs protection

### Full Tunnel Mode

Enable full tunnel mode when you need to route ALL client traffic through the VPN:

1. Edit the gateway in **Admin > Gateways**
2. Enable **Full Tunnel Mode**
3. Save changes

When enabled:
- All traffic is routed through the VPN (`0.0.0.0/0`)
- Uses OpenVPN's `redirect-gateway def1 bypass-dhcp` directive
- Client's internet traffic goes through the VPN server
- Useful for traffic inspection, compliance, or security requirements

### Settings That Trigger Tunnel Mode

| Setting | Effect |
|---------|--------|
| **Full Tunnel Mode OFF** | Push routes for user's CIDR access rules only |
| **Full Tunnel Mode ON** | Push `redirect-gateway def1 bypass-dhcp` (all traffic) |

## DNS Configuration

Control whether the VPN pushes DNS settings to clients:

### Push DNS (Default: Disabled)

By default, GateKey does NOT push DNS settings, allowing clients to use their local DNS resolver.

To enable DNS pushing:
1. Edit the gateway in **Admin > Gateways**
2. Enable **Push DNS Servers**
3. Optionally configure custom DNS servers
4. Save changes

### Custom DNS Servers

When Push DNS is enabled, you can specify custom DNS servers:

1. Enable **Push DNS Servers**
2. Enter DNS server IPs in the **DNS Servers** field
3. Use comma-separated values (e.g., `1.1.1.1, 8.8.8.8`)

If no custom DNS servers are configured, defaults to `1.1.1.1` and `8.8.8.8`.

### DNS Settings Table

| Push DNS | DNS Servers | Behavior |
|----------|-------------|----------|
| OFF | - | Client uses local DNS (default) |
| ON | Empty | Pushes `1.1.1.1` and `8.8.8.8` |
| ON | Custom list | Pushes specified DNS servers |

### Use Cases

- **Split tunnel + no DNS push**: User accesses VPN resources but uses local DNS (most common)
- **Split tunnel + DNS push**: User accesses VPN resources with corporate DNS (internal domain resolution)
- **Full tunnel + DNS push**: All traffic and DNS through VPN (high security environments)

## Dynamic Route Pushing

Routes are pushed to clients dynamically during connection, not statically configured in the OpenVPN server.

### How It Works

1. Client connects to gateway
2. Gateway calls control plane's `/api/v1/gateway/connect` endpoint
3. Control plane looks up user's access rules:
   - Direct user assignments (`user_access_rules`)
   - Group assignments (`group_access_rules`)
4. For each CIDR-type access rule, a route is generated
5. Routes are returned in the response and pushed to the client

### Route Format

CIDR access rules are converted to OpenVPN route push directives:

| Access Rule CIDR | OpenVPN Route Pushed |
|------------------|---------------------|
| `192.168.50.0/23` | `route 192.168.50.0 255.255.254.0` |
| `10.0.0.0/8` | `route 10.0.0.0 255.0.0.0` |
| `172.16.0.0/12` | `route 172.16.0.0 255.240.0.0` |

### Benefits

- **Dynamic access control**: Add/remove access rules without gateway reconfiguration
- **Per-user routing**: Different users get different routes based on their permissions
- **Immediate effect**: New access rules apply on next client connection
- **Centralized management**: All routing controlled from the control plane

## API Reference

The gateway communicates with the control plane via these endpoints:

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/gateway/heartbeat` | Send status updates, receive config version |
| `POST /api/v1/gateway/verify` | Validate client connections |
| `POST /api/v1/gateway/connect` | Report client connections |
| `POST /api/v1/gateway/disconnect` | Report client disconnections |
| `POST /api/v1/gateway/provision` | Provision server certificates, TLS-Auth key, and config |
| `POST /api/v1/gateway/client-rules` | Get access rules for a specific client |
| `POST /api/v1/gateway/all-rules` | Get all rules for periodic refresh |

All requests include the gateway token in the request body.

### Heartbeat Response

The heartbeat endpoint now returns config management information:

```json
{
  "status": "ok",
  "gateway_id": "uuid",
  "gateway_name": "us-east-1",
  "config_version": "sha256-hash",
  "needs_reprovision": false
}
```

When `needs_reprovision` is `true`, the gateway agent automatically calls the provision endpoint.
