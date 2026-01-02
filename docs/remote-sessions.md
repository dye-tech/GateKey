# Remote Sessions

Remote Sessions allow administrators to execute shell commands on gateways, mesh hubs, and mesh spokes directly from the control plane. This feature enables network troubleshooting, diagnostics, and system management without requiring direct SSH access to remote nodes.

## Overview

Remote sessions use an outbound WebSocket connection from agents to the control plane. This design:

- **No inbound firewall rules needed** - Agents connect outbound to the control plane
- **Secure by default** - All communication is encrypted over TLS
- **Real-time command execution** - Interactive shell sessions with streaming output
- **Centralized management** - Manage all nodes from a single interface

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Admin CLI     │────▶│  Control Plane   │◀────│   Hub Agent     │
│ gatekey-admin   │     │   (WebSocket)    │     │ session_enabled │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │   Spoke Agent    │
                        │ session_enabled  │
                        └──────────────────┘
```

Agents maintain a persistent WebSocket connection to the control plane. When an administrator requests a command, the control plane forwards it to the appropriate agent, which executes it and streams the output back.

## Enabling Remote Sessions

### Gateway Configuration

Add `session_enabled: true` to your gateway configuration:

```yaml
# /etc/gatekey/gateway.yaml
name: gateway-1
control_plane_url: https://vpn.company.com
token: gw_xxxxx

session_enabled: true
```

### Mesh Hub Configuration

Add `session_enabled: true` to your hub configuration:

```yaml
# /etc/gatekey/hub.yaml
name: hub-1
control_plane_url: https://vpn.company.com
token: hub_xxxxx

session_enabled: true
```

### Mesh Spoke Configuration

Add `session_enabled: true` to your spoke configuration:

```yaml
# /etc/gatekey/spoke.yaml
name: spoke-datacenter
control_plane_url: https://vpn.company.com
hub_endpoint: hub.example.com:1194
token: spoke_xxxxx

session_enabled: true
```

After updating the configuration, restart the agent service:

```bash
sudo systemctl restart gatekey-gateway  # or gatekey-hub, gatekey-spoke
```

## Managing Remote Sessions via CLI

### List Connected Agents

View all agents with remote sessions enabled and connected:

```bash
gatekey-admin session list
```

Output:
```
Agent ID    Type      Node Name         Connected
----------- --------- ----------------- -------------------
hub-1       hub       hub-1             2024-01-15 10:30:45
spoke-dc    spoke     spoke-datacenter  2024-01-15 10:32:12
gw-west     gateway   gateway-west      2024-01-15 10:28:00
```

### Execute a Single Command

Run a command on a specific agent:

```bash
# Check network configuration
gatekey-admin session exec hub-1 "ip addr"

# View OpenVPN status
gatekey-admin session exec hub-1 "systemctl status openvpn"

# Check connectivity
gatekey-admin session exec spoke-datacenter "ping -c 3 10.0.0.1"

# View routing table
gatekey-admin session exec gw-west "ip route"
```

### Interactive Shell Session

Start an interactive session with an agent:

```bash
gatekey-admin session connect hub-1
```

This opens an interactive shell where you can run multiple commands:

```
$ gatekey-admin session connect hub-1
Connected to hub-1. Type 'exit' to disconnect.

$ ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 ...
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> ...
3: tun0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> ...

$ systemctl status openvpn
● openvpn.service - OpenVPN service
   Active: active (running) since Mon 2024-01-15 10:30:00 UTC

$ cat /etc/openvpn/server.conf | head -10
port 1194
proto udp
dev tun
...

$ exit
Disconnecting...
```

## Managing Remote Sessions via Admin API

### Enable/Disable Remote Sessions

Use the update commands to toggle remote sessions:

```bash
# Enable on a gateway
gatekey-admin gateway update <gateway-id> --session=true

# Disable on a mesh hub
gatekey-admin mesh hub update <hub-id> --session=false

# Enable on a mesh spoke
gatekey-admin mesh spoke update <spoke-id> --session=true
```

## Network Troubleshooting

The admin CLI provides built-in network diagnostic tools that can run from the control plane or any connected agent.

### Available Tools

| Tool | Description | Example |
|------|-------------|---------|
| `ping` | Test ICMP connectivity | `gatekey-admin troubleshoot ping 8.8.8.8` |
| `nslookup` | DNS lookup | `gatekey-admin troubleshoot nslookup google.com` |
| `traceroute` | Trace route to host | `gatekey-admin troubleshoot traceroute 10.0.0.1` |
| `nc` | Test TCP connectivity | `gatekey-admin troubleshoot nc api.internal.com --port 443` |
| `nmap` | Port scanning | `gatekey-admin troubleshoot nmap 10.0.0.1 --ports 22,80,443` |

### Running Tools from Remote Locations

Execute diagnostics from a specific node using the `--location` flag:

```bash
# Ping from the control plane (default)
gatekey-admin troubleshoot ping 10.0.0.1

# Ping from a specific hub
gatekey-admin troubleshoot ping 10.0.0.1 --location hub:<hub-id>

# DNS lookup from a spoke
gatekey-admin troubleshoot nslookup internal.database.local --location spoke:<spoke-id>

# Test port from a gateway
gatekey-admin troubleshoot nc database.internal.com --port 5432 --location gateway:<gateway-id>
```

### List Available Tools and Locations

```bash
gatekey-admin troubleshoot list
```

Output:
```
Available Tools:
  ping         - Test ICMP connectivity to a host
  nslookup     - Perform DNS lookup for a hostname
  traceroute   - Trace the route to a host
  nc           - Test TCP connectivity to a port
  nmap         - Scan ports on a host

Execution Locations:
  control-plane                  (Control Plane)
  gateway:abc123                 (Gateway: gateway-west)
  hub:def456                     (Hub: hub-1)
  spoke:ghi789                   (Spoke: spoke-datacenter)
```

## Security Considerations

### Authentication

- Remote sessions require admin authentication
- All commands are executed with the agent's service account privileges
- Sessions are tied to the authenticated admin user

### Audit Logging

All remote session activity is logged:

- Session connections and disconnections
- Commands executed
- Command output (configurable)

View session activity in audit logs:

```bash
gatekey-admin audit list --action session
```

### Best Practices

1. **Limit session access** - Only enable remote sessions on nodes that require remote management
2. **Use least privilege** - Agents run as non-root service accounts by default
3. **Monitor activity** - Review audit logs regularly for unexpected session activity
4. **Secure the control plane** - Ensure TLS is properly configured on the control plane

## Troubleshooting

### Agent Not Appearing in Session List

1. **Check configuration:**
   ```bash
   grep session_enabled /etc/gatekey/hub.yaml
   ```
   Ensure `session_enabled: true` is set.

2. **Check agent status:**
   ```bash
   systemctl status gatekey-hub
   ```

3. **Check agent logs:**
   ```bash
   journalctl -u gatekey-hub -f
   ```
   Look for WebSocket connection messages.

4. **Verify network connectivity:**
   The agent must be able to reach the control plane on the WebSocket endpoint.

### Command Execution Fails

1. **Check agent connectivity:**
   ```bash
   gatekey-admin session list
   ```
   Verify the agent shows as connected.

2. **Check command timeout:**
   Long-running commands may timeout (default: 60 seconds).

3. **Check agent logs** for execution errors.

### Connection Drops

WebSocket connections may drop due to:
- Network interruptions
- Load balancer timeouts (configure idle timeout > 60s)
- Agent restarts

Agents automatically reconnect when the connection is restored.

## See Also

- [Admin CLI Guide](admin-cli.md) - Complete CLI reference
- [Gateway Setup](gateway-setup.md) - Gateway installation and configuration
- [Mesh Networking](mesh-networking.md) - Hub and spoke setup
- [Security](security.md) - Security best practices
