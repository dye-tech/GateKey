#!/bin/bash
# GateKey Gateway Installer
# This script installs and configures the GateKey gateway agent alongside OpenVPN.
#
# Usage:
#   curl -sSL https://your-gatekey-server/install-gateway.sh | bash -s -- \
#     --server https://gatekey.example.com \
#     --token YOUR_GATEWAY_TOKEN \
#     --name my-gateway
#
# Or download and run:
#   ./install-gateway.sh --server https://gatekey.example.com --token TOKEN --name my-gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
gatekey_SERVER=""
GATEWAY_TOKEN=""
GATEWAY_NAME=""
INSTALL_DIR="/opt/gatekey"
CONFIG_DIR="/etc/gatekey"
BIN_DIR="/usr/local/bin"
OPENVPN_CONFIG_DIR="/etc/openvpn/server"
VPN_PORT="1194"
VPN_PROTOCOL="udp"
VPN_NETWORK="172.31.255.0/24"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --server)
            gatekey_SERVER="$2"
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
            VPN_PORT="$2"
            shift 2
            ;;
        --protocol)
            VPN_PROTOCOL="$2"
            shift 2
            ;;
        --network)
            VPN_NETWORK="$2"
            shift 2
            ;;
        --help)
            echo "GateKey Gateway Installer"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Required options:"
            echo "  --server URL      GateKey control plane URL (e.g., https://gatekey.example.com)"
            echo "  --token TOKEN     Gateway authentication token (from admin UI)"
            echo "  --name NAME       Gateway name (must match the registered name)"
            echo ""
            echo "Optional options:"
            echo "  --port PORT       OpenVPN port (default: 1194)"
            echo "  --protocol PROTO  OpenVPN protocol: udp or tcp (default: udp)"
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
if [[ -z "$gatekey_SERVER" ]]; then
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

echo -e "${GREEN}GateKey Gateway Installer${NC}"
echo "========================"
echo "Server: $gatekey_SERVER"
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
                # Handle RHEL/CentOS version-specific EPEL installation
                if [[ "$VERSION" == "10" ]] || [[ "$VERSION" =~ ^10\. ]]; then
                    echo -e "${YELLOW}Detected RHEL/CentOS 10...${NC}"
                    dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-10.noarch.rpm || true
                    /usr/bin/crb enable 2>/dev/null || true
                    # Install base dependencies (but NOT EPEL openvpn - it's broken on RHEL 10)
                    dnf install -y curl jq
                    # Build OpenVPN from source for RHEL 10 due to OpenSSL compatibility issues
                    install_openvpn_from_source
                elif [[ "$VERSION" == "9" ]] || [[ "$VERSION" =~ ^9\. ]]; then
                    echo -e "${YELLOW}Detected RHEL/CentOS 9, installing EPEL 9...${NC}"
                    dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm || dnf install -y epel-release
                    /usr/bin/crb enable 2>/dev/null || true
                    dnf install -y openvpn curl jq
                else
                    dnf install -y epel-release
                    dnf install -y openvpn curl jq
                fi
            else
                yum install -y epel-release
                yum install -y openvpn curl jq
            fi
            ;;
        amzn)
            # Amazon Linux 2 and Amazon Linux 2023
            if command -v dnf &> /dev/null; then
                # Amazon Linux 2023 uses dnf
                # Note: easy-rsa is NOT needed for gateway (certs come from control plane)
                # Handle curl-minimal conflict by using --allowerasing
                dnf install -y openvpn jq
                dnf install -y --allowerasing curl
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
            apk add --no-cache openvpn curl jq bash openssl
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo -e "${YELLOW}Supported: ubuntu, debian, centos, rhel, fedora, rocky, almalinux, amzn (Amazon Linux), opensuse, sles, arch, manjaro, alpine${NC}"
            exit 1
            ;;
    esac
}

# Build OpenVPN from source (required for RHEL 10 due to OpenSSL compatibility)
install_openvpn_from_source() {
    echo -e "${YELLOW}Building OpenVPN from source (RHEL 10 workaround)...${NC}"

    # Remove broken EPEL openvpn if installed
    dnf remove -y openvpn 2>/dev/null || true

    # Install build dependencies
    dnf install -y gcc make autoconf automake libtool \
        openssl-devel lzo-devel lz4-devel pam-devel \
        libcap-ng-devel systemd-devel libnl3-devel

    # Download and build OpenVPN
    OPENVPN_VERSION="2.6.12"
    cd /tmp

    if [[ ! -f "openvpn-${OPENVPN_VERSION}.tar.gz" ]]; then
        curl -sSLO "https://swupdate.openvpn.org/community/releases/openvpn-${OPENVPN_VERSION}.tar.gz"
    fi

    rm -rf "openvpn-${OPENVPN_VERSION}"
    tar xzf "openvpn-${OPENVPN_VERSION}.tar.gz"
    cd "openvpn-${OPENVPN_VERSION}"

    ./configure --prefix=/usr --sysconfdir=/etc --enable-systemd
    make -j$(nproc)
    make install

    # Create systemd service template
    cat > /etc/systemd/system/openvpn-server@.service << 'SVCEOF'
[Unit]
Description=OpenVPN service for %I
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
ExecStart=/usr/sbin/openvpn --config /etc/openvpn/server/%i.conf
Restart=on-failure

[Install]
WantedBy=multi-user.target
SVCEOF

    systemctl daemon-reload

    # Verify installation
    if /usr/sbin/openvpn --version | head -1 | grep -q "OpenVPN"; then
        echo -e "${GREEN}OpenVPN built and installed successfully${NC}"
    else
        echo -e "${RED}Failed to build OpenVPN${NC}"
        exit 1
    fi

    cd /tmp
}

# Download and install gateway binary
install_gateway_binary() {
    echo -e "${YELLOW}Installing GateKey gateway agent...${NC}"

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
    DOWNLOAD_URL="${gatekey_SERVER}/downloads/gatekey-gateway-linux-${ARCH}"

    if curl -sSL -o "$BIN_DIR/gatekey-gateway" "$DOWNLOAD_URL"; then
        chmod +x "$BIN_DIR/gatekey-gateway"
        echo -e "${GREEN}Gateway binary installed${NC}"
    else
        echo -e "${YELLOW}Could not download binary from server, using local build...${NC}"
        # For development, copy from local build
        if [[ -f ./bin/gatekey-gateway ]]; then
            cp ./bin/gatekey-gateway "$BIN_DIR/gatekey-gateway"
            chmod +x "$BIN_DIR/gatekey-gateway"
        else
            echo -e "${RED}Error: Could not install gateway binary${NC}"
            exit 1
        fi
    fi
}

# Create gateway configuration
create_gateway_config() {
    echo -e "${YELLOW}Creating gateway configuration...${NC}"

    cat > "$CONFIG_DIR/gateway.yaml" << EOF
# GateKey Gateway Configuration
# Generated by install-gateway.sh

# Gateway name (must match registered name in control plane)
name: "${GATEWAY_NAME}"

# Control plane URL
control_plane_url: "${gatekey_SERVER}"

# Gateway authentication token
token: "${GATEWAY_TOKEN}"

# Heartbeat interval
heartbeat_interval: "30s"

# Log level (debug, info, warn, error)
log_level: "info"

# OpenVPN settings
openvpn:
  config_dir: "${OPENVPN_CONFIG_DIR}"
  port: ${VPN_PORT}
  protocol: "${VPN_PROTOCOL}"
  network: "${VPN_NETWORK}"
EOF

    chmod 600 "$CONFIG_DIR/gateway.yaml"
    echo -e "${GREEN}Configuration created at ${CONFIG_DIR}/gateway.yaml${NC}"
}

# Provision certificates from control plane
provision_certificates() {
    echo -e "${YELLOW}Provisioning certificates from control plane...${NC}"

    mkdir -p "$OPENVPN_CONFIG_DIR"

    # Call provision API to get certificates
    PROVISION_RESPONSE=$(curl -sSL -X POST "${gatekey_SERVER}/api/v1/gateway/provision" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"${GATEWAY_TOKEN}\"}" 2>/dev/null)

    if echo "$PROVISION_RESPONSE" | jq -e '.ca_cert' > /dev/null 2>&1; then
        echo "$PROVISION_RESPONSE" | jq -r '.ca_cert' > "$OPENVPN_CONFIG_DIR/ca.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.server_cert' > "$OPENVPN_CONFIG_DIR/server.crt"
        echo "$PROVISION_RESPONSE" | jq -r '.server_key' > "$OPENVPN_CONFIG_DIR/server.key"
        chmod 600 "$OPENVPN_CONFIG_DIR/server.key"

        # Extract VPN settings from provision response
        PROVISION_PORT=$(echo "$PROVISION_RESPONSE" | jq -r '.vpn_port // 1194')
        PROVISION_PROTO=$(echo "$PROVISION_RESPONSE" | jq -r '.vpn_protocol // "udp"')
        if [[ "$PROVISION_PORT" != "null" ]]; then
            VPN_PORT="$PROVISION_PORT"
        fi
        if [[ "$PROVISION_PROTO" != "null" ]]; then
            VPN_PROTOCOL="$PROVISION_PROTO"
        fi

        echo -e "${GREEN}Certificates provisioned from control plane${NC}"
    else
        echo -e "${RED}Error: Failed to provision certificates from control plane${NC}"
        echo -e "${YELLOW}Response: $PROVISION_RESPONSE${NC}"
        exit 1
    fi

    # Generate DH parameters (not provided by API)
    echo -e "${YELLOW}Generating DH parameters (this may take a moment)...${NC}"
    openssl dhparam -out "$OPENVPN_CONFIG_DIR/dh.pem" 2048

    # Generate TLS auth key
    openvpn --genkey secret "$OPENVPN_CONFIG_DIR/ta.key"
    chmod 600 "$OPENVPN_CONFIG_DIR/ta.key"
}

# Configure OpenVPN server
configure_openvpn() {
    echo -e "${YELLOW}Configuring OpenVPN server...${NC}"

    mkdir -p "$OPENVPN_CONFIG_DIR"

    # Create OpenVPN server configuration
    cat > "$OPENVPN_CONFIG_DIR/server.conf" << EOF
# OpenVPN Server Configuration
# Managed by GateKey Gateway

# Network settings
port ${VPN_PORT}
proto ${VPN_PROTOCOL}
dev tun

# Certificate settings
ca ${OPENVPN_CONFIG_DIR}/ca.crt
cert ${OPENVPN_CONFIG_DIR}/server.crt
key ${OPENVPN_CONFIG_DIR}/server.key
dh ${OPENVPN_CONFIG_DIR}/dh.pem
tls-auth ${OPENVPN_CONFIG_DIR}/ta.key 0

# Network configuration
server 10.8.0.0 255.255.255.0
topology subnet
# Note: Routes and DNS are pushed dynamically by the client-connect hook
# based on user's access rules and gateway's full_tunnel_mode setting

# Keep tunnel alive
keepalive 10 120

# Encryption settings
cipher AES-256-GCM
auth SHA384
tls-version-min 1.2

# Persist settings
persist-key
persist-tun

# Logging
status /var/log/openvpn/status.log
log-append /var/log/openvpn/openvpn.log
verb 1

# GateKey authentication hooks
script-security 3
auth-user-pass-verify ${OPENVPN_CONFIG_DIR}/gatekey-auth.sh via-env
client-connect ${OPENVPN_CONFIG_DIR}/gatekey-connect.sh
client-disconnect ${OPENVPN_CONFIG_DIR}/gatekey-disconnect.sh

# Require both client certificate AND auth-user-pass for maximum security
# Certificate provides identity, auth token provides revocation capability
verify-client-cert require
EOF

    # Create hook scripts
    cat > "$OPENVPN_CONFIG_DIR/gatekey-auth.sh" << 'AUTHEOF'
#!/bin/bash
/usr/local/bin/gatekey-gateway hook --type auth-user-pass-verify --config /etc/gatekey/gateway.yaml
AUTHEOF

    cat > "$OPENVPN_CONFIG_DIR/gatekey-connect.sh" << 'CONNECTEOF'
#!/bin/bash
# $1 is the client config file path passed by OpenVPN - routes are written here
/usr/local/bin/gatekey-gateway hook --type client-connect --config /etc/gatekey/gateway.yaml "$1"
CONNECTEOF

    cat > "$OPENVPN_CONFIG_DIR/gatekey-disconnect.sh" << 'DISCONNECTEOF'
#!/bin/bash
/usr/local/bin/gatekey-gateway hook --type client-disconnect --config /etc/gatekey/gateway.yaml
DISCONNECTEOF

    chmod +x "$OPENVPN_CONFIG_DIR"/*.sh

    # Create log directory
    mkdir -p /var/log/openvpn

    echo -e "${GREEN}OpenVPN configuration created${NC}"
}

# Create systemd service for gateway agent
create_systemd_service() {
    echo -e "${YELLOW}Creating systemd service...${NC}"

    cat > /etc/systemd/system/gatekey-gateway.service << EOF
[Unit]
Description=GateKey Gateway Agent
After=network.target openvpn-server@server.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/gatekey-gateway run --config ${CONFIG_DIR}/gateway.yaml
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

    # Enable IPv4 forwarding
    echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-gatekey.conf
    sysctl -p /etc/sysctl.d/99-gatekey.conf

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
        firewall-cmd --permanent --add-port=${VPN_PORT}/${VPN_PROTOCOL}
        firewall-cmd --permanent --add-masquerade
        firewall-cmd --reload
    elif command -v nft &> /dev/null; then
        echo "Using nftables"
        # Input rules for VPN port
        nft add table inet gatekey 2>/dev/null || true
        nft add chain inet gatekey input "{ type filter hook input priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey input udp dport ${VPN_PORT} accept 2>/dev/null || true

        # Forward rules for VPN traffic
        nft add chain inet gatekey forward "{ type filter hook forward priority 0; }" 2>/dev/null || true
        nft add rule inet gatekey forward iifname "tun0" accept 2>/dev/null || true
        nft add rule inet gatekey forward oifname "tun0" ct state related,established accept 2>/dev/null || true

        # NAT masquerade for VPN traffic
        nft add table ip nat 2>/dev/null || true
        nft add chain ip nat postrouting "{ type nat hook postrouting priority 100; }" 2>/dev/null || true
        nft add rule ip nat postrouting ip saddr ${VPN_NETWORK} oifname "${DEFAULT_IFACE}" masquerade 2>/dev/null || true

        echo "nftables NAT configured for ${VPN_NETWORK} -> ${DEFAULT_IFACE}"
    elif command -v ufw &> /dev/null; then
        echo "Using ufw"
        ufw allow ${VPN_PORT}/${VPN_PROTOCOL}
        # ufw requires manual NAT config in /etc/ufw/before.rules
        # Add iptables rules as fallback
        iptables -I FORWARD 1 -i tun0 -j ACCEPT 2>/dev/null || true
        iptables -I FORWARD 2 -o tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
        iptables -t nat -A POSTROUTING -s ${VPN_NETWORK} -o ${DEFAULT_IFACE} -j MASQUERADE 2>/dev/null || true
    elif command -v iptables &> /dev/null; then
        echo "Using iptables"
        iptables -A INPUT -p ${VPN_PROTOCOL} --dport ${VPN_PORT} -j ACCEPT
        # Forward rules for VPN traffic
        iptables -I FORWARD 1 -i tun0 -j ACCEPT
        iptables -I FORWARD 2 -o tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT
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

    HEALTH_CHECK=$(curl -sSL "${gatekey_SERVER}/health" 2>/dev/null || echo "failed")

    if echo "$HEALTH_CHECK" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
        echo -e "${GREEN}Successfully connected to control plane${NC}"
    else
        echo -e "${YELLOW}Warning: Could not verify connection to control plane${NC}"
    fi
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    # Enable and start OpenVPN
    systemctl enable openvpn-server@server.service 2>/dev/null || true

    # Enable and start gateway agent
    systemctl enable gatekey-gateway.service
    systemctl start gatekey-gateway.service

    echo -e "${GREEN}Services started${NC}"
}

# Main installation flow
main() {
    detect_os
    echo "Detected OS: $OS $VERSION"

    install_dependencies
    install_gateway_binary
    create_gateway_config
    provision_certificates
    configure_openvpn
    create_systemd_service
    enable_ip_forwarding
    configure_firewall
    test_connection
    start_services

    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}GateKey Gateway Installation Complete!${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "Services:"
    echo "  - gatekey-gateway: $(systemctl is-active gatekey-gateway 2>/dev/null || echo 'unknown')"
    echo "  - openvpn-server@server: $(systemctl is-active openvpn-server@server 2>/dev/null || echo 'unknown')"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status gatekey-gateway"
    echo "  systemctl status openvpn-server@server"
    echo "  journalctl -u gatekey-gateway -f"
    echo "  journalctl -u openvpn-server@server -f"
    echo ""
    echo "VPN endpoint: $(hostname -I | awk '{print $1}'):${VPN_PORT}/${VPN_PROTOCOL}"
    echo ""
}

main
