#!/bin/bash
# GateKey Mesh Hub Installer
# This script installs and configures the GateKey mesh hub alongside OpenVPN.
#
# Usage:
#   curl -sSL https://your-gatekey-server/scripts/install-hub.sh | sudo bash -s -- \
#     --token YOUR_HUB_TOKEN \
#     --control-plane https://gatekey.example.com
#
# Or download and run:
#   sudo ./install-hub.sh --token TOKEN --control-plane https://gatekey.example.com

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
OPENVPN_CONFIG_DIR="/etc/openvpn/server"

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
        --help)
            echo "GateKey Mesh Hub Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --control-plane URL   GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN         Hub API token (from admin UI)"
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

echo -e "${GREEN}GateKey Mesh Hub Installer${NC}"
echo "=========================="
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
            apt-get install -y openvpn curl jq nftables iptables
            ;;
        centos|rhel|fedora|rocky|almalinux)
            if command -v dnf &> /dev/null; then
                if [[ "$VERSION" == "9" ]] || [[ "$VERSION" =~ ^9\. ]]; then
                    dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm || dnf install -y epel-release
                    /usr/bin/crb enable 2>/dev/null || true
                else
                    dnf install -y epel-release
                fi
                dnf install -y openvpn curl jq nftables iptables
            else
                yum install -y epel-release
                yum install -y openvpn curl jq nftables iptables
            fi
            ;;
        amzn)
            # Amazon Linux 2 and Amazon Linux 2023
            if command -v dnf &> /dev/null; then
                # Amazon Linux 2023 uses dnf
                # Handle curl-minimal conflict by using --allowerasing
                dnf install -y openvpn jq nftables iptables
                dnf install -y --allowerasing curl
            else
                # Amazon Linux 2 uses yum and needs EPEL for OpenVPN
                amazon-linux-extras install -y epel 2>/dev/null || yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
                yum install -y openvpn curl jq nftables iptables
            fi
            ;;
        opensuse*|sles|suse)
            # openSUSE and SUSE Linux Enterprise
            zypper install -y openvpn curl jq nftables iptables
            ;;
        arch|manjaro)
            # Arch Linux and Manjaro
            pacman -Sy --noconfirm openvpn curl jq nftables iptables
            ;;
        alpine)
            # Alpine Linux
            apk add --no-cache openvpn curl jq bash nftables iptables
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo -e "${YELLOW}Supported: ubuntu, debian, centos, rhel, fedora, rocky, almalinux, amzn (Amazon Linux), opensuse, sles, arch, manjaro, alpine${NC}"
            exit 1
            ;;
    esac
}

# Download and install hub binary
install_hub_binary() {
    echo -e "${YELLOW}Installing GateKey hub binary...${NC}"

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
    DOWNLOAD_URL="${CONTROL_PLANE_URL}/downloads/gatekey-hub-linux-${ARCH}"

    if curl -sSL -o "$BIN_DIR/gatekey-hub" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-hub"
        echo -e "${GREEN}Hub binary installed${NC}"
    else
        echo -e "${RED}Error: Could not download hub binary${NC}"
        exit 1
    fi
}

# Create hub configuration
create_hub_config() {
    echo -e "${YELLOW}Creating hub configuration...${NC}"

    cat > "$CONFIG_DIR/hub.yaml" << EOF
# GateKey Mesh Hub Configuration
# Generated by install-hub.sh

# Control plane URL
control_plane_url: "${CONTROL_PLANE_URL}"

# Hub API token
api_token: "${HUB_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Log level (debug, info, warn, error)
log_level: "info"

# OpenVPN settings
openvpn:
  config_dir: "${OPENVPN_CONFIG_DIR}"
EOF

    chmod 600 "$CONFIG_DIR/hub.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/hub.yaml${NC}"
}

# Provision certificates from control plane
provision_certificates() {
    echo -e "${YELLOW}Provisioning from control plane...${NC}"

    mkdir -p "$OPENVPN_CONFIG_DIR"
    mkdir -p "$OPENVPN_CONFIG_DIR/ccd"

    # Call provision API to get certificates and config
    PROVISION_RESPONSE=$(curl -sSL -X POST "${CONTROL_PLANE_URL}/api/v1/mesh-hub/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${HUB_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.cacert' > /dev/null 2>&1; then
        echo "$PROVISION_RESPONSE" | jq -r '.cacert' > "$OPENVPN_CONFIG_DIR/ca.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.servercert' > "$OPENVPN_CONFIG_DIR/server.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.serverkey' > "$OPENVPN_CONFIG_DIR/server.key"
        chmod 600 "$OPENVPN_CONFIG_DIR/server.key"

        # Extract DH params if provided
        DH_PARAMS=$(echo "$PROVISION_RESPONSE" | jq -r '.dhparams // ""')
        if [[ -n "$DH_PARAMS" && "$DH_PARAMS" != "null" ]]; then
            echo "$DH_PARAMS" > "$OPENVPN_CONFIG_DIR/dh.pem"
        else
            echo -e "${YELLOW}Generating DH parameters (this may take a moment)...${NC}"
            openssl dhparam -out "$OPENVPN_CONFIG_DIR/dh.pem" 2048
        fi

        # Extract TLS auth key if provided
        TLS_AUTH=$(echo "$PROVISION_RESPONSE" | jq -r '.tlsauthkey // ""')
        if [[ -n "$TLS_AUTH" && "$TLS_AUTH" != "null" ]]; then
            echo "$TLS_AUTH" > "$OPENVPN_CONFIG_DIR/ta.key"
            chmod 600 "$OPENVPN_CONFIG_DIR/ta.key"
        fi

        # Extract VPN settings
        VPN_PORT=$(echo "$PROVISION_RESPONSE" | jq -r '.vpnport // 1194')
        VPN_PROTOCOL=$(echo "$PROVISION_RESPONSE" | jq -r '.vpnprotocol // "udp"')
        VPN_SUBNET=$(echo "$PROVISION_RESPONSE" | jq -r '.vpnsubnet // "172.30.0.0/16"')

        echo -e "${GREEN}Certificates provisioned from control plane${NC}"
    else
        echo -e "${RED}Error: Failed to provision from control plane${NC}"
        echo -e "${YELLOW}Response: $PROVISION_RESPONSE${NC}"
        exit 1
    fi
}

# Configure OpenVPN server
configure_openvpn() {
    echo -e "${YELLOW}Configuring OpenVPN server...${NC}"

    # Convert CIDR to network/netmask for OpenVPN server directive
    # e.g., 172.30.0.0/16 -> 172.30.0.0 255.255.0.0
    NETWORK=$(echo "$VPN_SUBNET" | cut -d'/' -f1)
    PREFIX=$(echo "$VPN_SUBNET" | cut -d'/' -f2)
    case $PREFIX in
        8)  NETMASK="255.0.0.0" ;;
        16) NETMASK="255.255.0.0" ;;
        24) NETMASK="255.255.255.0" ;;
        *)  NETMASK="255.255.0.0" ;;
    esac

    # Create OpenVPN server configuration
    cat > "$OPENVPN_CONFIG_DIR/hub.conf" << EOF
# OpenVPN Mesh Hub Server Configuration
# Managed by GateKey Hub

# Network settings
port ${VPN_PORT}
proto ${VPN_PROTOCOL}
dev tun

# Certificate settings
ca ${OPENVPN_CONFIG_DIR}/ca.crt
cert ${OPENVPN_CONFIG_DIR}/server.crt
key ${OPENVPN_CONFIG_DIR}/server.key
dh ${OPENVPN_CONFIG_DIR}/dh.pem
EOF

    # Add TLS auth if key exists
    if [[ -f "$OPENVPN_CONFIG_DIR/ta.key" ]]; then
        echo "tls-auth ${OPENVPN_CONFIG_DIR}/ta.key 0" >> "$OPENVPN_CONFIG_DIR/hub.conf"
    fi

    cat >> "$OPENVPN_CONFIG_DIR/hub.conf" << EOF

# Network configuration
server ${NETWORK} ${NETMASK}
topology subnet

# Client config directory for per-gateway routing (iroute)
client-config-dir ${OPENVPN_CONFIG_DIR}/ccd

# Allow client-to-client communication
client-to-client

# Keep tunnel alive
keepalive 10 120

# Encryption settings (FIPS compliant)
cipher AES-256-GCM
auth SHA384
tls-version-min 1.2

# Persist settings
persist-key
persist-tun

# Logging
status /var/log/openvpn/hub-status.log
log-append /var/log/openvpn/hub.log
verb 1

# Script security for hooks
script-security 3

# Client connect/disconnect hooks
client-connect ${OPENVPN_CONFIG_DIR}/hub-connect.sh
client-disconnect ${OPENVPN_CONFIG_DIR}/hub-disconnect.sh
EOF

    # Create hook scripts (simple pass-through - TLS cert validation handles auth)
    cat > "$OPENVPN_CONFIG_DIR/hub-connect.sh" << 'CONNECTEOF'
#!/bin/bash
# Client connect hook - certificate validation is handled by OpenVPN TLS
# Additional authentication can be added here if needed
exit 0
CONNECTEOF

    cat > "$OPENVPN_CONFIG_DIR/hub-disconnect.sh" << 'DISCONNECTEOF'
#!/bin/bash
# Client disconnect hook
exit 0
DISCONNECTEOF

    chmod +x "$OPENVPN_CONFIG_DIR"/*.sh

    # Create log directory
    mkdir -p /var/log/openvpn

    echo -e "${GREEN}OpenVPN configuration created${NC}"
}

# Create systemd service for hub
create_systemd_service() {
    echo -e "${YELLOW}Creating systemd service...${NC}"

    cat > /etc/systemd/system/gatekey-hub.service << EOF
[Unit]
Description=GateKey Mesh Hub
After=network.target openvpn-server@hub.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-hub run --config ${CONFIG_DIR}/hub.yaml
Restart=always
RestartSec=5
User=root
Group=root

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log /etc/openvpn/server /etc/gatekey

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    echo -e "${GREEN}Systemd service created${NC}"
}

# Enable IP forwarding
enable_ip_forwarding() {
    echo -e "${YELLOW}Enabling IP forwarding...${NC}"

    echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-gatekey-hub.conf
    sysctl -p /etc/sysctl.d/99-gatekey-hub.conf

    echo -e "${GREEN}IP forwarding enabled${NC}"
}

# Configure firewall
configure_firewall() {
    echo -e "${YELLOW}Configuring firewall...${NC}"

    DEFAULT_IFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    if [ -z "$DEFAULT_IFACE" ]; then
        DEFAULT_IFACE="eth0"
    fi

    if command -v firewall-cmd &> /dev/null && systemctl is-active --quiet firewalld; then
        firewall-cmd --permanent --add-port=${VPN_PORT}/${VPN_PROTOCOL}
        firewall-cmd --permanent --add-masquerade
        firewall-cmd --reload
    elif command -v nft &> /dev/null; then
        nft add table inet gatekey 2>/dev/null || true
        nft add chain inet gatekey input "{ type filter hook input priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey input ${VPN_PROTOCOL} dport ${VPN_PORT} accept 2>/dev/null || true
        # Forward rules for VPN traffic
        nft add chain inet gatekey forward "{ type filter hook forward priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey forward iifname "tun0" accept 2>/dev/null || true
        nft add rule inet gatekey forward oifname "tun0" ct state related,established accept 2>/dev/null || true
        # NAT masquerade
        nft add table ip nat 2>/dev/null || true
        nft add chain ip nat postrouting "{ type nat hook postrouting priority 100; }" 2>/dev/null || true
        nft add rule ip nat postrouting ip saddr ${VPN_SUBNET} oifname "${DEFAULT_IFACE}" masquerade 2>/dev/null || true
    elif command -v iptables &> /dev/null; then
        iptables -A INPUT -p ${VPN_PROTOCOL} --dport ${VPN_PORT} -j ACCEPT
        # Forward rules for VPN traffic
        iptables -I FORWARD 1 -i tun0 -j ACCEPT
        iptables -I FORWARD 2 -o tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT
        # NAT masquerade
        iptables -t nat -A POSTROUTING -s ${VPN_SUBNET} -o ${DEFAULT_IFACE} -j MASQUERADE
    fi

    echo -e "${GREEN}Firewall configured${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    systemctl enable openvpn-server@hub.service 2>/dev/null || true
    systemctl start openvpn-server@hub.service 2>/dev/null || true

    systemctl enable gatekey-hub.service
    systemctl start gatekey-hub.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_hub_binary
    create_hub_config
    provision_certificates
    configure_openvpn
    create_systemd_service
    enable_ip_forwarding
    configure_firewall
    start_services

    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}GateKey Mesh Hub Installation Complete!${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-hub: $(systemctl is-active gatekey-hub 2>/dev/null || echo 'unknown')"
    echo "  - openvpn-server@hub: $(systemctl is-active openvpn-server@hub 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-hub"
    echo "  systemctl status openvpn-server@hub"
    echo "  journalctl -u gatekey-hub -f"
    echo ""
    echo "Hub endpoint: $(hostname -I | awk '{print $1}'):${VPN_PORT}/${VPN_PROTOCOL}"
    echo ""
}

main
