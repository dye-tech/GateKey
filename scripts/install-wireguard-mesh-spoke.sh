#!/bin/bash
# GateKey WireGuard Mesh Spoke Installer
# This script installs and configures the GateKey WireGuard mesh spoke.
#
# Usage:
#   curl -sSL https://your-gatekey-server/scripts/install-wireguard-mesh-spoke.sh | sudo bash -s -- \
#     --token YOUR_SPOKE_TOKEN \
#     --control-plane https://gatekey.example.com
#
# Or download and run:
#   sudo ./install-wireguard-mesh-spoke.sh --token TOKEN --control-plane https://gatekey.example.com

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONTROL_PLANE_URL=""
SPOKE_TOKEN=""
INSTALL_DIR="/opt/gatekey"
CONFIG_DIR="/etc/gatekey"
BIN_DIR="/usr/local/bin"
WG_CONFIG_DIR="/etc/wireguard"
WG_INTERFACE="wg0"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --control-plane)
            CONTROL_PLANE_URL="$2"
            shift 2
            ;;
        --token)
            SPOKE_TOKEN="$2"
            shift 2
            ;;
        --interface)
            WG_INTERFACE="$2"
            shift 2
            ;;
        --help)
            echo "GateKey WireGuard Mesh Spoke Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --control-plane URL   GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN         Spoke token (from admin UI)"
            echo ""
            echo "Optional options:"
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

if [[ -z "$SPOKE_TOKEN" ]]; then
    echo -e "${RED}Error: --token is required${NC}"
    exit 1
fi

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

echo -e "${GREEN}GateKey WireGuard Mesh Spoke Installer${NC}"
echo "======================================="
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
            apt-get install -y wireguard-tools curl jq iproute2
            ;;
        centos|rhel|fedora|rocky|almalinux)
            if command -v dnf &> /dev/null; then
                dnf install -y epel-release 2>/dev/null || true
                dnf install -y wireguard-tools curl jq iproute
            else
                yum install -y epel-release 2>/dev/null || true
                yum install -y wireguard-tools curl jq iproute
            fi
            ;;
        amzn)
            # Amazon Linux 2 requires special handling for wireguard-tools
            echo "Installing epel-release"
            amazon-linux-extras install -y epel 2>/dev/null || yum install -y epel-release 2>/dev/null || true
            yum clean all && yum makecache

            # Install base dependencies
            yum install -y curl jq iproute

            # Try to install wireguard-tools from EPEL
            if ! yum install -y wireguard-tools 2>/dev/null; then
                echo "wireguard-tools not in EPEL, installing from source..."
                # Install build dependencies
                yum install -y make gcc git

                # Download and compile wireguard-tools
                cd /tmp
                rm -rf wireguard-tools
                git clone https://git.zx2c4.com/wireguard-tools
                cd wireguard-tools/src
                make
                make install
                cd /
                rm -rf /tmp/wireguard-tools

                # Create symlinks if needed
                if [ ! -f /usr/bin/wg ]; then
                    ln -sf /usr/local/bin/wg /usr/bin/wg 2>/dev/null || true
                fi
                if [ ! -f /usr/bin/wg-quick ]; then
                    ln -sf /usr/local/bin/wg-quick /usr/bin/wg-quick 2>/dev/null || true
                fi
            fi

            # Verify wg is available
            if ! command -v wg &> /dev/null; then
                echo -e "${RED}Failed to install wireguard-tools${NC}"
                exit 1
            fi
            ;;
        opensuse*|sles|suse)
            zypper install -y wireguard-tools curl jq iproute2
            ;;
        arch|manjaro)
            pacman -Sy --noconfirm wireguard-tools curl jq iproute2
            ;;
        alpine)
            apk add --no-cache wireguard-tools curl jq iproute2 bash
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac

    # Ensure WireGuard kernel module is loaded
    modprobe wireguard 2>/dev/null || true
}

# Download and install spoke binary
install_spoke_binary() {
    echo -e "${YELLOW}Installing GateKey WireGuard mesh spoke agent...${NC}"

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
    DOWNLOAD_URL="${CONTROL_PLANE_URL}/downloads/gatekey-wireguard-mesh-gateway-linux-${ARCH}"

    if curl -sSL -o "$BIN_DIR/gatekey-wireguard-mesh-gateway" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-wireguard-mesh-gateway"
        echo -e "${GREEN}Spoke binary installed${NC}"
    else
        echo -e "${RED}Error: Could not download spoke binary${NC}"
        exit 1
    fi
}

# Create spoke configuration
create_spoke_config() {
    echo -e "${YELLOW}Creating spoke configuration...${NC}"

    cat > "$CONFIG_DIR/wireguard-mesh-spoke.yaml" << EOF
# GateKey WireGuard Mesh Spoke Configuration
# Generated by install-wireguard-mesh-spoke.sh

# Control plane URL
control_plane_url: "${CONTROL_PLANE_URL}"

# Spoke authentication token
gateway_token: "${SPOKE_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Log level (debug, info, warn, error)
log_level: "info"

# WireGuard settings
wireguard:
  interface: "${WG_INTERFACE}"
  config_path: "${WG_CONFIG_DIR}/${WG_INTERFACE}.conf"
EOF

    chmod 600 "$CONFIG_DIR/wireguard-mesh-spoke.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/wireguard-mesh-spoke.yaml${NC}"
}

# Provision WireGuard keys and configuration from control plane
provision_wireguard() {
    echo -e "${YELLOW}Provisioning WireGuard configuration from control plane...${NC}"

    mkdir -p "$WG_CONFIG_DIR"

    # Call provision API
    PROVISION_RESPONSE=$(curl -sSL -X POST "${CONTROL_PLANE_URL}/api/v1/wg-mesh-spoke/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${SPOKE_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.private_key' > /dev/null 2>&1; then
        WG_PRIVATE_KEY=$(echo "$PROVISION_RESPONSE" | jq -r '.private_key')
        WG_ADDRESS=$(echo "$PROVISION_RESPONSE" | jq -r '.address')
        HUB_PUBLIC_KEY=$(echo "$PROVISION_RESPONSE" | jq -r '.hub_public_key')
        HUB_ENDPOINT=$(echo "$PROVISION_RESPONSE" | jq -r '.hub_endpoint')
        ALLOWED_IPS=$(echo "$PROVISION_RESPONSE" | jq -r '.allowed_ips // "0.0.0.0/0"')

        # Create WireGuard configuration file
        cat > "$WG_CONFIG_DIR/${WG_INTERFACE}.conf" << EOF
# GateKey WireGuard Mesh Spoke Configuration
# Auto-generated - managed by gatekey-wireguard-mesh-gateway

[Interface]
PrivateKey = ${WG_PRIVATE_KEY}
Address = ${WG_ADDRESS}

[Peer]
PublicKey = ${HUB_PUBLIC_KEY}
Endpoint = ${HUB_ENDPOINT}
AllowedIPs = ${ALLOWED_IPS}
PersistentKeepalive = 25
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

    cat > /etc/systemd/system/gatekey-wireguard-mesh-spoke.service << EOF
[Unit]
Description=GateKey WireGuard Mesh Spoke Agent
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-wireguard-mesh-gateway run --config ${CONFIG_DIR}/wireguard-mesh-spoke.yaml
Restart=always
RestartSec=5
User=root
Group=root

# Security
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log /etc/wireguard /etc/gatekey /run /tmp

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

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    systemctl enable gatekey-wireguard-mesh-spoke.service
    systemctl start gatekey-wireguard-mesh-spoke.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_spoke_binary
    create_spoke_config
    provision_wireguard
    create_systemd_service
    enable_ip_forwarding
    start_services

    echo ""
    echo -e "${GREEN}==================================================${NC}"
    echo -e "${GREEN}GateKey WireGuard Mesh Spoke Installation Complete!${NC}"
    echo -e "${GREEN}==================================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-wireguard-mesh-spoke: $(systemctl is-active gatekey-wireguard-mesh-spoke 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-wireguard-mesh-spoke"
    echo "  journalctl -u gatekey-wireguard-mesh-spoke -f"
    echo "  wg show ${WG_INTERFACE}"
    echo ""
}

main
