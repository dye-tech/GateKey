#!/bin/bash
# GateKey WireGuard Gateway Installer
# This script installs and configures the GateKey WireGuard gateway agent.
#
# Usage:
#   curl -sSL https://your-gatekey-server/install-wireguard-gateway.sh | bash -s -- \
#     --server https://gatekey.example.com \
#     --token YOUR_GATEWAY_TOKEN \
#     --name my-wireguard-gateway
#
# Or download and run:
#   ./install-wireguard-gateway.sh --server https://gatekey.example.com --token TOKEN --name my-gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GATEKEY_SERVER=""
GATEWAY_TOKEN=""
GATEWAY_NAME=""
INSTALL_DIR="/opt/gatekey"
CONFIG_DIR="/etc/gatekey"
BIN_DIR="/usr/local/bin"
WG_CONFIG_DIR="/etc/wireguard"
WG_PORT="51820"
WG_INTERFACE="wg0"
VPN_NETWORK="172.31.255.0/24"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --server)
            GATEKEY_SERVER="$2"
            shift 2
            ;;
        --token)
            GATEWAY_TOKEN="$2"
            shift 2
            ;;
        --name)
            GATEWAY_NAME="$2"
            shift 2
            ;;
        --port)
            WG_PORT="$2"
            shift 2
            ;;
        --interface)
            WG_INTERFACE="$2"
            shift 2
            ;;
        --network)
            VPN_NETWORK="$2"
            shift 2
            ;;
        --help)
            echo "GateKey WireGuard Gateway Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --server URL      GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN     Gateway authentication token (from admin UI)"
            echo "  --name NAME       Gateway name (must match the registered name)"
            echo ""
            echo "Optional options:"
            echo "  --port PORT       WireGuard listen port (default: 51820)"
            echo "  --interface IFACE WireGuard interface name (default: wg0)"
            echo "  --network CIDR    VPN network CIDR (default: 172.31.255.0/24)"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$GATEKEY_SERVER" ]]; then
    echo -e "${RED}Error: --server is required${NC}"
    exit 1
fi

if [[ -z "$GATEWAY_TOKEN" ]]; then
    echo -e "${RED}Error: --token is required${NC}"
    exit 1
fi

if [[ -z "$GATEWAY_NAME" ]]; then
    echo -e "${RED}Error: --name is required${NC}"
    exit 1
fi

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

echo -e "${GREEN}GateKey WireGuard Gateway Installer${NC}"
echo "===================================="
echo "Server: $GATEKEY_SERVER"
echo "Gateway: $GATEWAY_NAME"
echo ""

# Detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        echo -e "${RED}Error: Unable to detect OS${NC}"
        exit 1
    fi
}

# Install wireguard-tools from source (wg command)
install_wireguard_tools() {
    echo -e "${YELLOW}Installing wireguard-tools from source...${NC}"

    # Install build dependencies
    if command -v apt-get &> /dev/null; then
        apt-get install -y build-essential libmnl-dev pkg-config
    elif command -v dnf &> /dev/null; then
        dnf install -y gcc make libmnl-devel
    elif command -v yum &> /dev/null; then
        yum install -y gcc make libmnl-devel
    fi

    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    # Clone and build wireguard-tools
    git clone https://git.zx2c4.com/wireguard-tools 2>/dev/null || git clone https://github.com/WireGuard/wireguard-tools.git
    cd wireguard-tools/src
    make
    make install

    cd /
    rm -rf "$TEMP_DIR"

    # Verify installation
    if command -v wg &> /dev/null; then
        echo -e "${GREEN}wireguard-tools (wg command) installed successfully${NC}"
    else
        echo -e "${RED}Failed to install wireguard-tools${NC}"
        exit 1
    fi
}

# Install wireguard-go (userspace WireGuard implementation)
install_wireguard_go() {
    echo -e "${YELLOW}Installing wireguard-go userspace implementation...${NC}"

    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture for wireguard-go: $ARCH${NC}"
            exit 1
            ;;
    esac

    # Download pre-built wireguard-go binary from control plane or build from source
    WG_GO_URL="${GATEKEY_SERVER}/downloads/wireguard-go-linux-${GO_ARCH}"

    if curl -sSL -o /usr/local/bin/wireguard-go "$WG_GO_URL" 2>/dev/null && [ -s /usr/local/bin/wireguard-go ]; then
        chmod +x /usr/local/bin/wireguard-go
        echo -e "${GREEN}wireguard-go installed from server${NC}"
    else
        # Fallback: try to install Go and build from source
        echo -e "${YELLOW}Downloading wireguard-go failed, attempting to build from source...${NC}"

        # Check if Go is installed
        if ! command -v go &> /dev/null; then
            echo -e "${YELLOW}Installing Go...${NC}"
            GO_VERSION="1.21.5"
            curl -sSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" | tar -C /usr/local -xzf -
            export PATH=$PATH:/usr/local/go/bin
        fi

        # Clone and build wireguard-go
        TEMP_DIR=$(mktemp -d)
        cd "$TEMP_DIR"
        git clone https://git.zx2c4.com/wireguard-go 2>/dev/null || git clone https://github.com/WireGuard/wireguard-go.git
        cd wireguard-go
        make
        cp wireguard-go /usr/local/bin/
        chmod +x /usr/local/bin/wireguard-go
        cd /
        rm -rf "$TEMP_DIR"
        echo -e "${GREEN}wireguard-go built and installed${NC}"
    fi

    # Verify installation
    if /usr/local/bin/wireguard-go --version 2>/dev/null || [ -x /usr/local/bin/wireguard-go ]; then
        echo -e "${GREEN}wireguard-go ready${NC}"
    else
        echo -e "${RED}Failed to install wireguard-go${NC}"
        exit 1
    fi
}

# Install dependencies
install_dependencies() {
    echo -e "${YELLOW}Installing dependencies...${NC}"

    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y wireguard-tools curl jq iproute2 nftables
            ;;
        centos|rhel|fedora|rocky|almalinux)
            if command -v dnf &> /dev/null; then
                dnf install -y epel-release 2>/dev/null || true
                dnf install -y wireguard-tools curl jq iproute nftables
            else
                yum install -y epel-release 2>/dev/null || true
                yum install -y wireguard-tools curl jq iproute nftables
            fi
            ;;
        amzn)
            # Amazon Linux 2 and Amazon Linux 2023
            if command -v dnf &> /dev/null; then
                # Amazon Linux 2023 - kernel has WireGuard built-in
                dnf install -y wireguard-tools jq iproute nftables
                dnf install -y --allowerasing curl
            else
                # Amazon Linux 2
                amazon-linux-extras install -y epel 2>/dev/null || true
                yum install -y curl jq iproute nftables git

                # Check if kernel supports WireGuard
                if modprobe wireguard 2>/dev/null; then
                    echo -e "${GREEN}WireGuard kernel module available${NC}"
                    yum install -y wireguard-tools 2>/dev/null || install_wireguard_tools
                else
                    echo -e "${YELLOW}WireGuard kernel module not available, installing wireguard-go (userspace)...${NC}"

                    # Install wireguard-tools (for wg command) - try package first, build from source if needed
                    if ! yum install -y wireguard-tools 2>/dev/null; then
                        echo -e "${YELLOW}wireguard-tools package not available, building from source...${NC}"
                        install_wireguard_tools
                    fi

                    # Install wireguard-go from source or binary
                    install_wireguard_go
                fi
            fi
            ;;
        opensuse*|sles|suse)
            zypper install -y wireguard-tools curl jq iproute2 nftables
            ;;
        arch|manjaro)
            pacman -Sy --noconfirm wireguard-tools curl jq iproute2 nftables
            ;;
        alpine)
            apk add --no-cache wireguard-tools curl jq iproute2 nftables bash
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo -e "${YELLOW}Supported: ubuntu, debian, centos, rhel, fedora, rocky, almalinux, amzn, opensuse, arch, manjaro, alpine${NC}"
            exit 1
            ;;
    esac

    # Ensure WireGuard kernel module is loaded
    modprobe wireguard 2>/dev/null || true
}

# Download and install gateway binary
install_gateway_binary() {
    echo -e "${YELLOW}Installing GateKey WireGuard gateway agent...${NC}"

    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"

    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    # Download binary from control plane
    DOWNLOAD_URL="${GATEKEY_SERVER}/downloads/gatekey-wireguard-gateway-linux-${ARCH}"

    if curl -sSL -o "$BIN_DIR/gatekey-wireguard-gateway" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-wireguard-gateway"
        echo -e "${GREEN}Gateway binary installed${NC}"
    else
        echo -e "${YELLOW}Could not download binary from server, using local build...${NC}"
        # For development, copy from local build
        if [[ -f ./bin/gatekey-wireguard-gateway ]]; then
            cp ./bin/gatekey-wireguard-gateway "$BIN_DIR/gatekey-wireguard-gateway"
            chmod +x "$BIN_DIR/gatekey-wireguard-gateway"
        else
            echo -e "${RED}Error: Could not install gateway binary${NC}"
            exit 1
        fi
    fi
}

# Create gateway configuration
create_gateway_config() {
    echo -e "${YELLOW}Creating gateway configuration...${NC}"

    cat > "$CONFIG_DIR/wireguard-gateway.yaml" << EOF
# GateKey WireGuard Gateway Configuration
# Generated by install-wireguard-gateway.sh

# Gateway name (must match registered name in control plane)
name: "${GATEWAY_NAME}"

# Control plane URL
control_plane_url: "${GATEKEY_SERVER}"

# Gateway authentication token
token: "${GATEWAY_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Peer sync interval (how often to fetch authorized peers)
peer_sync_interval: "10s"

# Stats report interval (how often to report peer traffic stats)
stats_report_interval: "5s"

# Log level (debug, info, warn, error)
log_level: "info"

# WireGuard settings
wireguard:
  interface: "${WG_INTERFACE}"
  listen_port: ${WG_PORT}
  config_path: "${WG_CONFIG_DIR}/${WG_INTERFACE}.conf"

# Firewall configuration
firewall:
  backend: "nftables"
  chain: "GATEX"
  table: "gatex"
  default_policy: "drop"

# Network configuration
network:
  interface: "${WG_INTERFACE}"
  network: "${VPN_NETWORK}"
EOF

    chmod 600 "$CONFIG_DIR/wireguard-gateway.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/wireguard-gateway.yaml${NC}"
}

# Provision WireGuard keys and configuration from control plane
provision_wireguard() {
    echo -e "${YELLOW}Provisioning WireGuard configuration from control plane...${NC}"

    mkdir -p "$WG_CONFIG_DIR"

    # Call provision API to get WireGuard configuration
    PROVISION_RESPONSE=$(curl -sSL -X POST "${GATEKEY_SERVER}/api/v1/wireguard-gateway/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${GATEWAY_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.private_key' > /dev/null 2>&1; then
        WG_PRIVATE_KEY=$(echo "$PROVISION_RESPONSE" | jq -r '.private_key')
        WG_ADDRESS=$(echo "$PROVISION_RESPONSE" | jq -r '.address')
        PROVISION_PORT=$(echo "$PROVISION_RESPONSE" | jq -r '.listen_port // 51820')

        if [[ "$PROVISION_PORT" != "null" ]]; then
            WG_PORT="$PROVISION_PORT"
        fi

        # Create WireGuard configuration file
        cat > "$WG_CONFIG_DIR/${WG_INTERFACE}.conf" << EOF
# GateKey WireGuard Gateway Configuration
# Auto-generated - managed by gatekey-wireguard-gateway

[Interface]
PrivateKey = ${WG_PRIVATE_KEY}
Address = ${WG_ADDRESS}
ListenPort = ${WG_PORT}

# Peers are managed dynamically by gatekey-wireguard-gateway
# Do not manually edit peer entries
EOF

        chmod 600 "$WG_CONFIG_DIR/${WG_INTERFACE}.conf"
        echo -e "${GREEN}WireGuard configuration provisioned from control plane${NC}"
    else
        echo -e "${RED}Error: Failed to provision WireGuard configuration from control plane${NC}"
        echo -e "${YELLOW}Response: $PROVISION_RESPONSE${NC}"
        exit 1
    fi
}

# Create systemd service for gateway agent
create_systemd_service() {
    echo -e "${YELLOW}Creating systemd service...${NC}"

    cat > /etc/systemd/system/gatekey-wireguard-gateway.service << EOF
[Unit]
Description=GateKey WireGuard Gateway Agent
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-wireguard-gateway run --config ${CONFIG_DIR}/wireguard-gateway.yaml
Restart=always
RestartSec=5
User=root
Group=root

# Security
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log /etc/wireguard /etc/gatekey /run

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    echo -e "${GREEN}Systemd service created${NC}"
}

# Enable IP forwarding
enable_ip_forwarding() {
    echo -e "${YELLOW}Enabling IP forwarding...${NC}"

    # Enable IPv4 forwarding
    echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-gatekey-wireguard.conf
    sysctl -p /etc/sysctl.d/99-gatekey-wireguard.conf

    echo -e "${GREEN}IP forwarding enabled${NC}"
}

# Configure firewall
configure_firewall() {
    echo -e "${YELLOW}Configuring firewall...${NC}"

    # Get the default interface for NAT masquerade
    DEFAULT_IFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    if [ -z "$DEFAULT_IFACE" ]; then
        DEFAULT_IFACE="eth0"
    fi
    echo "Default interface: $DEFAULT_IFACE"

    # Detect firewall type and configure
    if command -v firewall-cmd &> /dev/null && systemctl is-active --quiet firewalld; then
        echo "Using firewalld"
        firewall-cmd --permanent --add-port=${WG_PORT}/udp
        firewall-cmd --permanent --add-masquerade
        firewall-cmd --reload
    elif command -v nft &> /dev/null; then
        echo "Using nftables"
        # Input rules for WireGuard port
        nft add table inet gatekey 2>/dev/null || true
        nft add chain inet gatekey input "{ type filter hook input priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey input udp dport ${WG_PORT} accept 2>/dev/null || true

        # Forward rules for VPN traffic
        nft add chain inet gatekey forward "{ type filter hook forward priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey forward iifname "${WG_INTERFACE}" accept 2>/dev/null || true
        nft add rule inet gatekey forward oifname "${WG_INTERFACE}" ct state related,established accept 2>/dev/null || true

        # NAT masquerade for VPN traffic
        nft add table ip nat 2>/dev/null || true
        nft add chain ip nat postrouting "{ type nat hook postrouting priority 100; }" 2>/dev/null || true
        nft add rule ip nat postrouting ip saddr ${VPN_NETWORK} oifname "${DEFAULT_IFACE}" masquerade 2>/dev/null || true

        echo "nftables NAT configured for ${VPN_NETWORK} -> ${DEFAULT_IFACE}"
    elif command -v ufw &> /dev/null; then
        echo "Using ufw"
        ufw allow ${WG_PORT}/udp
        # Add iptables rules as fallback for NAT
        iptables -I FORWARD 1 -i ${WG_INTERFACE} -j ACCEPT 2>/dev/null || true
        iptables -I FORWARD 2 -o ${WG_INTERFACE} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
        iptables -t nat -A POSTROUTING -s ${VPN_NETWORK} -o ${DEFAULT_IFACE} -j MASQUERADE 2>/dev/null || true
    elif command -v iptables &> /dev/null; then
        echo "Using iptables"
        iptables -A INPUT -p udp --dport ${WG_PORT} -j ACCEPT
        # Forward rules for VPN traffic
        iptables -I FORWARD 1 -i ${WG_INTERFACE} -j ACCEPT
        iptables -I FORWARD 2 -o ${WG_INTERFACE} -m state --state RELATED,ESTABLISHED -j ACCEPT
        # NAT masquerade
        iptables -t nat -A POSTROUTING -s ${VPN_NETWORK} -o ${DEFAULT_IFACE} -j MASQUERADE

        # Save iptables rules if possible
        if command -v iptables-save &> /dev/null; then
            iptables-save > /etc/sysconfig/iptables 2>/dev/null || \
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
        fi
    fi

    echo -e "${GREEN}Firewall configured${NC}"
}

# Test connection to control plane
test_connection() {
    echo -e "${YELLOW}Testing connection to control plane...${NC}"

    HEALTH_CHECK=$(curl -sSL "${GATEKEY_SERVER}/health" 2>/dev/null || echo "failed")

    if echo "$HEALTH_CHECK" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
        echo -e "${GREEN}Successfully connected to control plane${NC}"
    else
        echo -e "${YELLOW}Warning: Could not verify connection to control plane${NC}"
    fi
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    # Enable and start gateway agent (it will bring up WireGuard interface)
    systemctl enable gatekey-wireguard-gateway.service
    systemctl start gatekey-wireguard-gateway.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_gateway_binary
    create_gateway_config
    provision_wireguard
    create_systemd_service
    enable_ip_forwarding
    configure_firewall
    test_connection
    start_services

    echo ""
    echo -e "${GREEN}================================================${NC}"
    echo -e "${GREEN}GateKey WireGuard Gateway Installation Complete!${NC}"
    echo -e "${GREEN}================================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-wireguard-gateway: $(systemctl is-active gatekey-wireguard-gateway 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-wireguard-gateway"
    echo "  journalctl -u gatekey-wireguard-gateway -f"
    echo "  wg show ${WG_INTERFACE}"
    echo ""
    echo "VPN endpoint: $(hostname -I | awk '{print $1}'):${WG_PORT}/udp"
    echo "Interface: ${WG_INTERFACE}"
    echo ""
}

main
