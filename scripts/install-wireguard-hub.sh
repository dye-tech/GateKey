#!/bin/bash
# GateKey WireGuard Mesh Hub Installer
# This script installs and configures the GateKey WireGuard mesh hub.
#
# Usage:
#   curl -sSL https://your-gatekey-server/scripts/install-wireguard-hub.sh | sudo bash -s -- \
#     --token YOUR_HUB_TOKEN \
#     --control-plane https://gatekey.example.com
#
# Or download and run:
#   sudo ./install-wireguard-hub.sh --token TOKEN --control-plane https://gatekey.example.com

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONTROL_PLANE_URL=""
HUB_TOKEN=""
INSTALL_DIR="/opt/gatekey"
CONFIG_DIR="/etc/gatekey"
BIN_DIR="/usr/local/bin"
WG_CONFIG_DIR="/etc/wireguard"
WG_PORT="51820"
WG_INTERFACE="wg0"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --control-plane)
            CONTROL_PLANE_URL="$2"
            shift 2
            ;;
        --token)
            HUB_TOKEN="$2"
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
        --help)
            echo "GateKey WireGuard Mesh Hub Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --control-plane URL   GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN         Hub API token (from admin UI)"
            echo ""
            echo "Optional options:"
            echo "  --port PORT           WireGuard listen port (default: 51820)"
            echo "  --interface IFACE     WireGuard interface name (default: wg0)"
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
if [[ -z "$CONTROL_PLANE_URL" ]]; then
    echo -e "${RED}Error: --control-plane is required${NC}"
    exit 1
fi

if [[ -z "$HUB_TOKEN" ]]; then
    echo -e "${RED}Error: --token is required${NC}"
    exit 1
fi

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

echo -e "${GREEN}GateKey WireGuard Mesh Hub Installer${NC}"
echo "====================================="
echo "Control Plane: $CONTROL_PLANE_URL"
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
            if command -v dnf &> /dev/null; then
                dnf install -y wireguard-tools jq iproute nftables
                dnf install -y --allowerasing curl
            else
                amazon-linux-extras install -y epel 2>/dev/null || true
                yum install -y wireguard-tools curl jq iproute nftables
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
            exit 1
            ;;
    esac

    # Ensure WireGuard kernel module is loaded
    modprobe wireguard 2>/dev/null || true
}

# Download and install hub binary
install_hub_binary() {
    echo -e "${YELLOW}Installing GateKey WireGuard hub agent...${NC}"

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
    DOWNLOAD_URL="${CONTROL_PLANE_URL}/downloads/gatekey-wireguard-hub-linux-${ARCH}"

    if curl -sSL -o "$BIN_DIR/gatekey-wireguard-hub" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-wireguard-hub"
        echo -e "${GREEN}Hub binary installed${NC}"
    else
        echo -e "${RED}Error: Could not download hub binary${NC}"
        exit 1
    fi
}

# Create hub configuration
create_hub_config() {
    echo -e "${YELLOW}Creating hub configuration...${NC}"

    cat > "$CONFIG_DIR/wireguard-hub.yaml" << EOF
# GateKey WireGuard Mesh Hub Configuration
# Generated by install-wireguard-hub.sh

# Control plane URL
control_plane_url: "${CONTROL_PLANE_URL}"

# Hub authentication token
token: "${HUB_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Peer sync interval
peer_sync_interval: "10s"

# Stats report interval
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
EOF

    chmod 600 "$CONFIG_DIR/wireguard-hub.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/wireguard-hub.yaml${NC}"
}

# Provision WireGuard keys and configuration from control plane
provision_wireguard() {
    echo -e "${YELLOW}Provisioning WireGuard configuration from control plane...${NC}"

    mkdir -p "$WG_CONFIG_DIR"

    # Call provision API
    PROVISION_RESPONSE=$(curl -sSL -X POST "${CONTROL_PLANE_URL}/api/v1/wg-mesh-hub/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${HUB_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.private_key' > /dev/null 2>&1; then
        WG_PRIVATE_KEY=$(echo "$PROVISION_RESPONSE" | jq -r '.private_key')
        WG_ADDRESS=$(echo "$PROVISION_RESPONSE" | jq -r '.address')
        PROVISION_PORT=$(echo "$PROVISION_RESPONSE" | jq -r '.listen_port // 51820')

        if [[ "$PROVISION_PORT" != "null" ]]; then
            WG_PORT="$PROVISION_PORT"
        fi

        # Create WireGuard configuration file
        cat > "$WG_CONFIG_DIR/${WG_INTERFACE}.conf" << EOF
# GateKey WireGuard Mesh Hub Configuration
# Auto-generated - managed by gatekey-wireguard-hub

[Interface]
PrivateKey = ${WG_PRIVATE_KEY}
Address = ${WG_ADDRESS}
ListenPort = ${WG_PORT}

# Peers are managed dynamically by gatekey-wireguard-hub
EOF

        chmod 600 "$WG_CONFIG_DIR/${WG_INTERFACE}.conf"
        echo -e "${GREEN}WireGuard configuration provisioned${NC}"
    else
        echo -e "${RED}Error: Failed to provision WireGuard configuration${NC}"
        echo -e "${YELLOW}Response: $PROVISION_RESPONSE${NC}"
        exit 1
    fi
}

# Create systemd service
create_systemd_service() {
    echo -e "${YELLOW}Creating systemd service...${NC}"

    cat > /etc/systemd/system/gatekey-wireguard-hub.service << EOF
[Unit]
Description=GateKey WireGuard Mesh Hub Agent
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-wireguard-hub run --config ${CONFIG_DIR}/wireguard-hub.yaml
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

    echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-gatekey-wireguard.conf
    sysctl -p /etc/sysctl.d/99-gatekey-wireguard.conf

    echo -e "${GREEN}IP forwarding enabled${NC}"
}

# Configure firewall
configure_firewall() {
    echo -e "${YELLOW}Configuring firewall...${NC}"

    DEFAULT_IFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    if [ -z "$DEFAULT_IFACE" ]; then
        DEFAULT_IFACE="eth0"
    fi

    if command -v nft &> /dev/null; then
        nft add table inet gatekey 2>/dev/null || true
        nft add chain inet gatekey input "{ type filter hook input priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey input udp dport ${WG_PORT} accept 2>/dev/null || true

        nft add chain inet gatekey forward "{ type filter hook forward priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey forward iifname "${WG_INTERFACE}" accept 2>/dev/null || true
        nft add rule inet gatekey forward oifname "${WG_INTERFACE}" ct state related,established accept 2>/dev/null || true
    fi

    echo -e "${GREEN}Firewall configured${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    systemctl enable gatekey-wireguard-hub.service
    systemctl start gatekey-wireguard-hub.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_hub_binary
    create_hub_config
    provision_wireguard
    create_systemd_service
    enable_ip_forwarding
    configure_firewall
    start_services

    echo ""
    echo -e "${GREEN}================================================${NC}"
    echo -e "${GREEN}GateKey WireGuard Mesh Hub Installation Complete!${NC}"
    echo -e "${GREEN}================================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-wireguard-hub: $(systemctl is-active gatekey-wireguard-hub 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-wireguard-hub"
    echo "  journalctl -u gatekey-wireguard-hub -f"
    echo "  wg show ${WG_INTERFACE}"
    echo ""
}

main
