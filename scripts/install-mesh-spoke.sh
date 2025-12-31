#!/bin/bash
# GateKey Mesh Spoke Installer
# This script installs and configures the GateKey mesh spoke (connects TO a hub).
#
# Usage:
#   curl -sSL https://your-gatekey-server/scripts/install-mesh-spoke.sh | sudo bash -s -- \
#     --token YOUR_SPOKE_TOKEN \
#     --control-plane https://gatekey.example.com
#
# Or download and run:
#   sudo ./install-mesh-spoke.sh --token TOKEN --control-plane https://gatekey.example.com

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
OPENVPN_CONFIG_DIR="/etc/openvpn/client"

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
        --help)
            echo "GateKey Mesh Spoke Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --control-plane URL   GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN         Spoke token (from admin UI)"
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

echo -e "${GREEN}GateKey Mesh Spoke Installer${NC}"
echo "=============================="
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
            apt-get install -y openvpn curl jq
            ;;
        centos|rhel|fedora|rocky|almalinux)
            if command -v dnf &> /dev/null; then
                if [[ "$VERSION" == "9" ]] || [[ "$VERSION" =~ ^9\. ]]; then
                    dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm || dnf install -y epel-release
                    /usr/bin/crb enable 2>/dev/null || true
                else
                    dnf install -y epel-release
                fi
                dnf install -y openvpn curl jq
            else
                yum install -y epel-release
                yum install -y openvpn curl jq
            fi
            ;;
        amzn)
            # Amazon Linux 2 and Amazon Linux 2023
            if command -v dnf &> /dev/null; then
                # Amazon Linux 2023 uses dnf
                dnf install -y openvpn curl jq
            else
                # Amazon Linux 2 uses yum and needs EPEL for OpenVPN
                amazon-linux-extras install -y epel 2>/dev/null || yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
                yum install -y openvpn curl jq
            fi
            ;;
        opensuse*|sles|suse)
            # openSUSE and SUSE Linux Enterprise
            zypper install -y openvpn curl jq
            ;;
        arch|manjaro)
            # Arch Linux and Manjaro
            pacman -Sy --noconfirm openvpn curl jq
            ;;
        alpine)
            # Alpine Linux
            apk add --no-cache openvpn curl jq bash
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo -e "${YELLOW}Supported: ubuntu, debian, centos, rhel, fedora, rocky, almalinux, amzn (Amazon Linux), opensuse, sles, arch, manjaro, alpine${NC}"
            exit 1
            ;;
    esac
}

# Download and install mesh spoke binary
install_spoke_binary() {
    echo -e "${YELLOW}Installing GateKey mesh spoke binary...${NC}"

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
    DOWNLOAD_URL="${CONTROL_PLANE_URL}/downloads/gatekey-mesh-spoke-linux-${ARCH}"

    echo -e "${YELLOW}Downloading from: ${DOWNLOAD_URL}${NC}"
    if curl -sSL -o "$BIN_DIR/gatekey-mesh-spoke" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-mesh-spoke"

        # Verify the binary can execute
        if ! "$BIN_DIR/gatekey-mesh-spoke" --help > /dev/null 2>&1; then
            echo -e "${RED}Error: Downloaded binary cannot execute (architecture mismatch?)${NC}"
            echo -e "${YELLOW}Expected architecture: ${ARCH}${NC}"
            echo -e "${YELLOW}Binary info: $(file $BIN_DIR/gatekey-mesh-spoke)${NC}"
            exit 1
        fi

        echo -e "${GREEN}Mesh spoke binary installed (${ARCH})${NC}"
    else
        echo -e "${RED}Error: Could not download mesh spoke binary${NC}"
        exit 1
    fi
}

# Create spoke configuration
create_spoke_config() {
    echo -e "${YELLOW}Creating spoke configuration...${NC}"

    mkdir -p "$CONFIG_DIR"
    mkdir -p /etc/gatekey-mesh

    cat > "$CONFIG_DIR/mesh-spoke.yaml" << EOF
# GateKey Mesh Spoke Configuration
# Generated by install-mesh-spoke.sh

# Control plane URL
control_plane_url: "${CONTROL_PLANE_URL}"

# Gateway/Spoke token
gateway_token: "${SPOKE_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Log level (debug, info, warn, error)
log_level: "info"

# OpenVPN client settings
openvpn:
  config_dir: "${OPENVPN_CONFIG_DIR}"
EOF

    chmod 600 "$CONFIG_DIR/mesh-spoke.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/mesh-spoke.yaml${NC}"
}

# Provision certificates from control plane
provision_certificates() {
    echo -e "${YELLOW}Provisioning from control plane...${NC}"

    mkdir -p "$OPENVPN_CONFIG_DIR"

    # Call provision API to get certificates and config
    PROVISION_RESPONSE=$(curl -sSL -X POST "${CONTROL_PLANE_URL}/api/v1/mesh-spoke/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${SPOKE_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.caCert' > /dev/null 2>&1; then
        echo "$PROVISION_RESPONSE" | jq -r '.caCert' > "$OPENVPN_CONFIG_DIR/ca.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.clientCert' > "$OPENVPN_CONFIG_DIR/client.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.clientKey' > "$OPENVPN_CONFIG_DIR/client.key"
        chmod 600 "$OPENVPN_CONFIG_DIR/client.key"

        # Extract TLS auth key if provided
        TLS_AUTH=$(echo "$PROVISION_RESPONSE" | jq -r '.tlsAuthKey // ""')
        if [[ -n "$TLS_AUTH" && "$TLS_AUTH" != "null" ]]; then
            echo "$TLS_AUTH" > "$OPENVPN_CONFIG_DIR/ta.key"
            chmod 600 "$OPENVPN_CONFIG_DIR/ta.key"
        fi

        # Extract hub connection info
        HUB_ENDPOINT=$(echo "$PROVISION_RESPONSE" | jq -r '.hubEndpoint')
        VPN_PORT=$(echo "$PROVISION_RESPONSE" | jq -r '.hubVpnPort // 1194')
        VPN_PROTOCOL=$(echo "$PROVISION_RESPONSE" | jq -r '.hubVpnProtocol // "udp"')
        LOCAL_NETWORKS=$(echo "$PROVISION_RESPONSE" | jq -r '.localNetworks // []')

        echo -e "${GREEN}Certificates provisioned from control plane${NC}"
    else
        echo -e "${RED}Error: Failed to provision from control plane${NC}"
        echo -e "${YELLOW}Response: $PROVISION_RESPONSE${NC}"
        exit 1
    fi
}

# Configure OpenVPN client
configure_openvpn() {
    echo -e "${YELLOW}Configuring OpenVPN client...${NC}"

    # Create OpenVPN client configuration
    cat > "$OPENVPN_CONFIG_DIR/mesh-spoke.conf" << EOF
# OpenVPN Mesh Spoke Client Configuration
# Managed by GateKey Mesh Spoke

client
dev tun
proto ${VPN_PROTOCOL}
remote ${HUB_ENDPOINT} ${VPN_PORT}

resolv-retry infinite
nobind
persist-key
persist-tun

# Certificate settings
ca ${OPENVPN_CONFIG_DIR}/ca.crt
cert ${OPENVPN_CONFIG_DIR}/client.crt
key ${OPENVPN_CONFIG_DIR}/client.key
EOF

    # Add TLS auth if key exists
    if [[ -f "$OPENVPN_CONFIG_DIR/ta.key" ]]; then
        echo "tls-auth ${OPENVPN_CONFIG_DIR}/ta.key 1" >> "$OPENVPN_CONFIG_DIR/mesh-spoke.conf"
    fi

    cat >> "$OPENVPN_CONFIG_DIR/mesh-spoke.conf" << EOF

# Encryption settings (FIPS compliant)
cipher AES-256-GCM
auth SHA384
tls-version-min 1.2

# Verify server certificate
remote-cert-tls server

# Logging
status /var/log/openvpn/mesh-spoke-status.log
log-append /var/log/openvpn/mesh-spoke.log
verb 3
EOF

    # Create log directory
    mkdir -p /var/log/openvpn

    echo -e "${GREEN}OpenVPN client configuration created${NC}"
}

# Create systemd service for mesh spoke
create_systemd_service() {
    echo -e "${YELLOW}Creating systemd services...${NC}"

    # OpenVPN client service
    cat > /etc/systemd/system/openvpn-mesh-spoke.service << EOF
[Unit]
Description=OpenVPN Mesh Spoke Connection
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
ExecStart=/usr/sbin/openvpn --config ${OPENVPN_CONFIG_DIR}/mesh-spoke.conf
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    # Mesh spoke agent service
    cat > /etc/systemd/system/gatekey-mesh-spoke.service << EOF
[Unit]
Description=GateKey Mesh Spoke Agent
After=network.target openvpn-mesh-spoke.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-mesh-spoke run --config ${CONFIG_DIR}/mesh-spoke.yaml
Restart=always
RestartSec=5
User=root
Group=root

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log /etc/openvpn/client /etc/gatekey

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    echo -e "${GREEN}Systemd services created${NC}"
}

# Enable IP forwarding
enable_ip_forwarding() {
    echo -e "${YELLOW}Enabling IP forwarding...${NC}"

    echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-gatekey-mesh.conf
    sysctl -p /etc/sysctl.d/99-gatekey-mesh.conf

    echo -e "${GREEN}IP forwarding enabled${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    systemctl enable openvpn-mesh-spoke.service
    systemctl start openvpn-mesh-spoke.service

    systemctl enable gatekey-mesh-spoke.service
    systemctl start gatekey-mesh-spoke.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_spoke_binary
    create_spoke_config
    provision_certificates
    configure_openvpn
    create_systemd_service
    enable_ip_forwarding
    start_services

    echo ""
    echo -e "${GREEN}================================================${NC}"
    echo -e "${GREEN}GateKey Mesh Spoke Installation Complete!${NC}"
    echo -e "${GREEN}================================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-mesh-spoke: $(systemctl is-active gatekey-mesh-spoke 2>/dev/null || echo 'unknown')"
    echo "  - openvpn-mesh-spoke: $(systemctl is-active openvpn-mesh-spoke 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-mesh-spoke"
    echo "  systemctl status openvpn-mesh-spoke"
    echo "  journalctl -u gatekey-mesh-spoke -f"
    echo "  journalctl -u openvpn-mesh-spoke -f"
    echo ""
    echo "Connecting to hub: ${HUB_ENDPOINT}:${VPN_PORT}/${VPN_PROTOCOL}"
    echo ""
}

main
