# GateKey Client

The GateKey Client (`gatekey`) is a user-facing VPN client that wraps OpenVPN to provide seamless, zero-configuration VPN connectivity. It handles authentication via your browser and automatically manages VPN configurations.

## Installation

### From Binary

Download the appropriate binary for your platform from the releases page and place it in your PATH:

```bash
# Linux/macOS
sudo mv gatekey /usr/local/bin/
sudo chmod +x /usr/local/bin/gatekey

# Verify installation
gatekey version
```

### From Source

```bash
git clone https://github.com/gatekey-project/gatekey.git
cd gatekey
make build-client
sudo mv bin/gatekey /usr/local/bin/
```

## Prerequisites

- **OpenVPN**: The client requires OpenVPN to be installed on your system
  - Linux: `sudo apt install openvpn` or `sudo dnf install openvpn`
  - macOS: `brew install openvpn`
  - Windows: Download from [OpenVPN website](https://openvpn.net/community-downloads/)

- **Root/Admin privileges**: OpenVPN requires elevated privileges to create network interfaces

## Quick Start

```bash
# 1. Initialize with your GateKey server URL
gatekey config init --server https://vpn.company.com

# 2. Authenticate (opens browser)
gatekey login

# 3. Connect to VPN
gatekey connect

# 4. Check status
gatekey status

# 5. Disconnect when done
gatekey disconnect
```

## Commands

### login

Authenticate with the GateKey server using your identity provider (OIDC/SAML).

```bash
gatekey login [flags]
```

**Flags:**
- `--no-browser` - Print the login URL instead of opening a browser (useful for headless systems)

**How it works:**
1. Opens your default browser to the GateKey login page
2. You authenticate with your identity provider (Okta, Azure AD, Google, etc.)
3. After successful authentication, the token is saved locally
4. The browser shows a success message and you can close it

**Example:**
```bash
# Normal login (opens browser)
gatekey login

# Headless/SSH login (copy URL to another browser)
gatekey login --no-browser
```

### logout

Clear saved credentials from your local machine.

```bash
gatekey logout
```

### connect

Connect to a VPN gateway. The client supports connecting to multiple gateways simultaneously.

```bash
gatekey connect [gateway] [flags]
```

**Flags:**
- `-g, --gateway string` - Gateway name to connect to

**Behavior:**
- If only one gateway is available, connects automatically
- If multiple gateways exist and none specified, lists available options
- Downloads a fresh, short-lived VPN configuration
- Starts OpenVPN in daemon mode
- Requires sudo/root for OpenVPN
- **Multi-gateway**: Each connection gets a unique tun interface (tun0, tun1, etc.)

**Examples:**
```bash
# Connect to default/only gateway
gatekey connect

# Connect to a specific gateway
gatekey connect us-east-1

# Connect to multiple gateways simultaneously
gatekey connect us-east-1    # Gets tun0
gatekey connect eu-west-1    # Gets tun1

# Using the flag
gatekey connect -g eu-west-1
```

### disconnect

Disconnect from VPN gateway(s).

```bash
gatekey disconnect [gateway] [flags]
```

**Flags:**
- `-a, --all` - Disconnect from all gateways

**Aliases:** `stop`

**Behavior:**
- If a gateway name is specified, disconnects from that specific gateway
- If no gateway specified and only one is connected, disconnects from it
- If no gateway specified and multiple are connected, disconnects from all
- Use `--all` to explicitly disconnect from all gateways

**Examples:**
```bash
# Disconnect from single/all gateway(s)
gatekey disconnect

# Disconnect from a specific gateway
gatekey disconnect us-east-1

# Disconnect from all gateways explicitly
gatekey disconnect --all
```

This gracefully terminates the OpenVPN process(es) and cleans up.

### status

Show the current VPN connection status for all connected gateways.

```bash
gatekey status [flags]
```

**Flags:**
- `--json` - Output in JSON format (useful for scripting)

**Single connection output:**
```
Status: Connected
Gateway:      us-east-1
Interface:    tun0
Connected at: 2024-01-15T10:30:00Z
Duration:     2h15m30s
Local IP:     10.8.0.5
PID:          12345

Routes:
  10.0.0.0/8
```

**Multiple connections output:**
```
Status: Connected to 2 gateways

  us-east-1:
    Status:    Connected
    Interface: tun0
    Duration:  2h15m30s
    PID:       12345

  eu-west-1:
    Status:    Connected
    Interface: tun1
    Duration:  1h30m15s
    PID:       12346
```

**JSON output (multiple connections):**
```json
{
  "connected": true,
  "connections": [
    {
      "connected": true,
      "gateway": "us-east-1",
      "gateway_id": "gw-001",
      "connected_at": "2024-01-15T10:30:00Z",
      "pid": 12345,
      "tun_interface": "tun0"
    },
    {
      "connected": true,
      "gateway": "eu-west-1",
      "gateway_id": "gw-002",
      "connected_at": "2024-01-15T11:00:00Z",
      "pid": 12346,
      "tun_interface": "tun1"
    }
  ]
}
```

### list

List available VPN gateways.

```bash
gatekey list
```

**Example output:**
```
Available Gateways:
-------------------
✓ us-east-1
  Description: US East Coast Gateway
  Location:    Virginia, USA
  Hostname:    vpn-us-east.example.com
  Status:      online

✓ eu-west-1
  Description: EU West Gateway
  Location:    Dublin, Ireland
  Hostname:    vpn-eu-west.example.com
  Status:      online
```

### config

Manage client configuration.

```bash
gatekey config <subcommand>
```

**Subcommands:**

#### config init

Initialize configuration with a server URL.

```bash
gatekey config init --server https://vpn.company.com
```

#### config show

Display current configuration.

```bash
gatekey config show
```

**Example output:**
```
GateKey Client Configuration
==========================
Config file:    /home/user/.gatekey/config.yaml
Server URL:     https://vpn.company.com
OpenVPN binary: openvpn
Config dir:     /home/user/.gatekey/configs
Log level:      info
```

#### config set

Set a configuration value.

```bash
gatekey config set <key> <value>
```

**Available keys:**
- `server_url` or `server` - GateKey server URL
- `openvpn_binary` or `openvpn` - Path to OpenVPN binary
- `config_dir` - Directory for VPN configs
- `log_level` - Logging level (debug, info, warn, error)

**Examples:**
```bash
gatekey config set server https://vpn.newserver.com
gatekey config set openvpn /usr/local/bin/openvpn
gatekey config set log_level debug
```

## Global Flags

These flags can be used with any command:

| Flag | Description |
|------|-------------|
| `--server string` | GateKey server URL (overrides config) |
| `--config string` | Config file path (default: ~/.gatekey/config.yaml) |
| `-h, --help` | Help for the command |

## Configuration

### Config File Location

The client stores configuration in `~/.gatekey/config.yaml`:

```yaml
server_url: https://vpn.company.com
openvpn_binary: openvpn
config_dir: /home/user/.gatekey/configs
log_level: info
```

### Data Directory

All client data is stored in `~/.gatekey/`:

```
~/.gatekey/
├── config.yaml              # Client configuration
├── token                    # Authentication token (encrypted)
├── state.json               # Multi-connection state
├── <gateway>.ovpn           # Gateway-specific VPN configuration
├── openvpn-<gateway>.pid    # Gateway-specific OpenVPN process ID
├── openvpn-<gateway>.log    # Gateway-specific OpenVPN log file
└── configs/                 # Downloaded configurations
```

For example, with two gateways connected:
```
~/.gatekey/
├── config.yaml
├── token
├── state.json
├── us-east-1.ovpn
├── openvpn-us-east-1.pid
├── openvpn-us-east-1.log
├── eu-west-1.ovpn
├── openvpn-eu-west-1.pid
├── openvpn-eu-west-1.log
└── configs/
```

## Authentication Flow

The client uses a browser-based OAuth flow:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │     │   Server    │     │     IdP     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ 1. Start local    │                   │
       │    callback server│                   │
       │                   │                   │
       │ 2. Open browser ──┼──────────────────►│
       │                   │                   │
       │                   │◄── 3. Auth ───────│
       │                   │                   │
       │◄── 4. Callback ───│                   │
       │    with token     │                   │
       │                   │                   │
       │ 5. Save token     │                   │
       │    locally        │                   │
       └───────────────────┴───────────────────┘
```

1. Client starts a temporary local HTTP server on a random port
2. Browser opens to GateKey server with callback URL
3. User authenticates with their identity provider
4. Server redirects to local callback with token
5. Client saves token securely

## VPN Connection Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │     │   Server    │     │   Gateway   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ 1. Request config │                   │
       │   (with token) ──►│                   │
       │                   │                   │
       │◄── 2. Short-lived │                   │
       │    .ovpn config   │                   │
       │                   │                   │
       │ 3. Start OpenVPN ─┼──────────────────►│
       │                   │                   │
       │                   │◄── 4. Validate ───│
       │                   │    certificate    │
       │                   │                   │
       │◄──────────────────┼── 5. Connected ───│
       │                   │                   │
       └───────────────────┴───────────────────┘
```

1. Client requests VPN config from server with auth token
2. Server generates short-lived certificate and .ovpn file
3. Client starts OpenVPN with the configuration
4. Gateway validates certificate with control plane
5. Connection established with identity-based firewall rules

## Troubleshooting

### "OpenVPN not found"

Ensure OpenVPN is installed and in your PATH:

```bash
# Check if OpenVPN is available
which openvpn

# If not found, install it
# Ubuntu/Debian
sudo apt install openvpn

# Fedora/RHEL
sudo dnf install openvpn

# macOS
brew install openvpn
```

Or specify the path:
```bash
gatekey config set openvpn /path/to/openvpn
```

### "Authentication required"

Your session has expired. Re-authenticate:

```bash
gatekey login
```

### "Permission denied" or sudo issues

OpenVPN requires root privileges. The client will prompt for sudo password. If you're having issues:

```bash
# Run with explicit sudo
sudo gatekey connect
```

### Connection fails immediately

Check the gateway-specific OpenVPN log:

```bash
cat ~/.gatekey/openvpn-<gateway-name>.log
```

Common issues:
- Firewall blocking UDP 1194 or TCP 443
- Certificate expired (re-run `connect` for a fresh config)
- DNS resolution issues

### "Already connected to <gateway>"

You're already connected to that specific gateway. Either:
- Disconnect and reconnect: `gatekey disconnect <gateway> && gatekey connect <gateway>`
- Connect to a different gateway: `gatekey connect <other-gateway>`

### Multi-gateway issues

When connected to multiple gateways:

```bash
# Check all connection statuses
gatekey status

# Check specific gateway log
cat ~/.gatekey/openvpn-<gateway>.log

# Disconnect from problematic gateway only
gatekey disconnect <gateway>
```

If routes conflict between gateways, ensure each gateway routes to different networks.

### Headless/SSH systems

Use `--no-browser` and copy the URL:

```bash
gatekey login --no-browser
# Copy the printed URL to a browser on another machine
```

## Security Considerations

1. **Token Storage**: Tokens are stored with 0600 permissions (owner read/write only)
2. **Short-lived Configs**: VPN configurations expire after 24 hours by default
3. **Certificate Validation**: All certificates are validated against the GateKey CA
4. **No Credential Storage**: Passwords are never stored; authentication is via IdP

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GATEKEY_SERVER` | Default server URL |
| `GATEKEY_CONFIG` | Config file path |
| `GATEKEY_LOG_LEVEL` | Log level (debug, info, warn, error) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication required |
| 3 | Connection failed |
| 4 | Already connected |

## See Also

- [Architecture Overview](architecture.md)
- [Deployment Guide](deployment.md)
- [API Reference](api.md)
- [FIPS Compliance](fips-compliance.md)
