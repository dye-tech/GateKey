# Network Troubleshooting

GateKey provides built-in network diagnostic tools that can be executed from the control plane or any connected gateway, hub, or spoke. These tools help diagnose connectivity issues, verify network paths, and test service availability across your VPN infrastructure.

## Overview

The troubleshooting tools are available through the `gatekey-admin troubleshoot` command. Tools can run locally on the control plane or remotely on any agent with remote sessions enabled.

## Available Tools

### ping

Test ICMP connectivity to a host.

```bash
# Basic ping (4 packets)
gatekey-admin troubleshoot ping 8.8.8.8

# Custom packet count
gatekey-admin troubleshoot ping google.com --count 10

# Ping from a remote location
gatekey-admin troubleshoot ping 10.0.0.1 --location hub:abc123
```

**Use cases:**
- Verify basic network connectivity
- Measure latency and packet loss
- Test if a host is reachable

### nslookup

Perform DNS lookup for a hostname.

```bash
# Basic DNS lookup
gatekey-admin troubleshoot nslookup google.com

# Lookup internal hostname from a spoke
gatekey-admin troubleshoot nslookup database.internal.local --location spoke:xyz789
```

**Use cases:**
- Verify DNS resolution
- Check if internal DNS is working through the VPN
- Debug DNS-related connectivity issues

### traceroute

Trace the network path to a destination host.

```bash
# Trace route to external host
gatekey-admin troubleshoot traceroute 8.8.8.8

# Trace route to internal host from hub
gatekey-admin troubleshoot traceroute 192.168.1.1 --location hub:abc123
```

**Use cases:**
- Identify routing issues
- Find where packets are being dropped
- Verify traffic flows through expected paths

### nc (netcat)

Test TCP connectivity to a specific port.

```bash
# Test HTTPS connectivity
gatekey-admin troubleshoot nc api.example.com --port 443

# Test database connectivity from a spoke
gatekey-admin troubleshoot nc db.internal.com --port 5432 --location spoke:xyz789

# Test SSH
gatekey-admin troubleshoot nc server.example.com --port 22
```

**Use cases:**
- Verify service availability
- Test firewall rules
- Check if a specific port is open

### nmap

Scan ports on a target host.

```bash
# Scan common ports
gatekey-admin troubleshoot nmap 10.0.0.1 --ports 22,80,443

# Scan a range of ports
gatekey-admin troubleshoot nmap server.internal.com --ports 1-1000

# Scan from a gateway
gatekey-admin troubleshoot nmap 192.168.1.100 --ports 22,3389 --location gateway:def456
```

**Use cases:**
- Discover open ports on a host
- Verify firewall configurations
- Audit service exposure

## Execution Locations

Tools can run from different locations in your network:

| Location | Syntax | Description |
|----------|--------|-------------|
| Control Plane | `--location control-plane` | Default. Runs on the control plane server |
| Gateway | `--location gateway:<id>` | Runs on a specific gateway |
| Mesh Hub | `--location hub:<id>` | Runs on a mesh hub |
| Mesh Spoke | `--location spoke:<id>` | Runs on a mesh spoke |

### List Available Locations

```bash
gatekey-admin troubleshoot list
```

This shows all available tools and execution locations:

```
Available Tools:
  ping         - Test ICMP connectivity to a host
  nslookup     - Perform DNS lookup for a hostname
  traceroute   - Trace the route to a host
  nc           - Test TCP connectivity to a port
  nmap         - Scan ports on a host

Execution Locations:
  control-plane                  (Control Plane)
  gateway:abc123                 (Gateway: us-east-gateway)
  hub:def456                     (Hub: primary-hub)
  spoke:ghi789                   (Spoke: datacenter-spoke)
```

### Remote Execution Requirements

To execute tools on remote agents:

1. The agent must have `session_enabled: true` in its configuration
2. The agent must be connected to the control plane
3. The required tools must be installed on the agent

## Common Troubleshooting Scenarios

### Scenario 1: User Cannot Access Internal Resource

1. **Verify VPN connection:**
   ```bash
   gatekey-admin connection list
   ```

2. **Test connectivity from the hub:**
   ```bash
   gatekey-admin troubleshoot ping 10.0.0.100 --location hub:abc123
   ```

3. **Check DNS resolution:**
   ```bash
   gatekey-admin troubleshoot nslookup app.internal.com --location hub:abc123
   ```

4. **Test service port:**
   ```bash
   gatekey-admin troubleshoot nc app.internal.com --port 443 --location hub:abc123
   ```

### Scenario 2: Spoke Cannot Reach Hub

1. **Check spoke status:**
   ```bash
   gatekey-admin mesh spoke list
   ```

2. **Ping hub from spoke:**
   ```bash
   gatekey-admin troubleshoot ping <hub-tunnel-ip> --location spoke:xyz789
   ```

3. **Trace route to hub:**
   ```bash
   gatekey-admin troubleshoot traceroute <hub-public-ip> --location spoke:xyz789
   ```

### Scenario 3: DNS Not Working Through VPN

1. **Test from control plane:**
   ```bash
   gatekey-admin troubleshoot nslookup internal.domain.com
   ```

2. **Test from hub:**
   ```bash
   gatekey-admin troubleshoot nslookup internal.domain.com --location hub:abc123
   ```

3. **Test DNS server directly:**
   ```bash
   gatekey-admin troubleshoot nc 10.0.0.53 --port 53 --location hub:abc123
   ```

### Scenario 4: Verify Firewall Rules

1. **Scan expected open ports:**
   ```bash
   gatekey-admin troubleshoot nmap 10.0.0.50 --ports 22,80,443
   ```

2. **Test from different locations:**
   ```bash
   # From control plane
   gatekey-admin troubleshoot nc 10.0.0.50 --port 443

   # From hub
   gatekey-admin troubleshoot nc 10.0.0.50 --port 443 --location hub:abc123

   # From spoke
   gatekey-admin troubleshoot nc 10.0.0.50 --port 443 --location spoke:xyz789
   ```

## Tool Output

Each tool provides structured output including:

- **Status:** success, failed, or timeout
- **Output:** Command output from the tool
- **Duration:** How long the command took
- **Location:** Where the command was executed
- **Timestamp:** When the command started

Example output:

```
Executing ping to 8.8.8.8 from hub:abc123...

PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=12.3 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.8 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=12.1 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=117 time=11.9 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 11.800/12.025/12.300/0.183 ms

Status:   success
Duration: 3.2s
```

## Timeouts and Limits

| Setting | Value | Description |
|---------|-------|-------------|
| Command timeout | 60 seconds | Maximum execution time per command |
| Ping count limit | 20 | Maximum ping packets |
| Traceroute hops | 20 | Maximum traceroute hops |
| Output size | 10 KB | Maximum output returned |

## Security Considerations

1. **Admin-only access:** Only authenticated administrators can run troubleshooting tools
2. **Audit logging:** All tool executions are logged with user, target, and location
3. **Limited scope:** Tools are restricted to diagnostic purposes
4. **Rate limiting:** Commands are rate-limited to prevent abuse

## See Also

- [Remote Sessions](remote-sessions.md) - Interactive shell access to agents
- [Admin CLI Guide](admin-cli.md) - Complete CLI reference
- [Mesh Networking](mesh-networking.md) - Hub and spoke troubleshooting
- [Gateway Setup](gateway-setup.md) - Gateway configuration
