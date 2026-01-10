package wireguard

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateClientConfig(t *testing.T) {
	// Generate test keys
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	req := ClientConfigRequest{
		GatewayName:         "test-gateway",
		GatewayEndpoint:     "vpn.example.com:51820",
		GatewayPublicKey:    gatewayKeyPair.PublicKey,
		ClientPrivateKey:    clientKeyPair.PrivateKey,
		ClientAddress:       "10.0.0.2/32",
		AllowedIPs:          []string{"10.0.0.0/24", "192.168.1.0/24"},
		DNS:                 []string{"1.1.1.1", "8.8.8.8"},
		PresharedKey:        psk,
		PersistentKeepalive: 25,
		ExpiresAt:           time.Now().Add(24 * time.Hour),
		UserEmail:           "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Verify required sections exist
	if !strings.Contains(content, "[Interface]") {
		t.Error("Config should contain [Interface] section")
	}
	if !strings.Contains(content, "[Peer]") {
		t.Error("Config should contain [Peer] section")
	}

	// Verify Interface section
	if !strings.Contains(content, "PrivateKey = "+clientKeyPair.PrivateKey) {
		t.Error("Config should contain client private key")
	}
	if !strings.Contains(content, "Address = 10.0.0.2/32") {
		t.Error("Config should contain client address")
	}
	if !strings.Contains(content, "DNS = 1.1.1.1, 8.8.8.8") {
		t.Error("Config should contain DNS servers")
	}

	// Verify Peer section
	if !strings.Contains(content, "PublicKey = "+gatewayKeyPair.PublicKey) {
		t.Error("Config should contain gateway public key")
	}
	if !strings.Contains(content, "Endpoint = vpn.example.com:51820") {
		t.Error("Config should contain gateway endpoint")
	}
	if !strings.Contains(content, "AllowedIPs = 10.0.0.0/24, 192.168.1.0/24") {
		t.Error("Config should contain allowed IPs")
	}
	if !strings.Contains(content, "PresharedKey = "+psk) {
		t.Error("Config should contain preshared key")
	}
	if !strings.Contains(content, "PersistentKeepalive = 25") {
		t.Error("Config should contain persistent keepalive")
	}

	// Verify header comments
	if !strings.Contains(content, "# Gateway: test-gateway") {
		t.Error("Config should contain gateway name comment")
	}
	if !strings.Contains(content, "# User: test@example.com") {
		t.Error("Config should contain user email comment")
	}

	// Verify filename
	if !strings.Contains(config.FileName, "gatekey-wg-test-gateway") {
		t.Errorf("FileName should contain gateway name, got: %s", config.FileName)
	}
	if !strings.HasSuffix(config.FileName, ".conf") {
		t.Errorf("FileName should end with .conf, got: %s", config.FileName)
	}
}

func TestGenerateClientConfigMinimal(t *testing.T) {
	// Generate test keys
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Minimal config without optional fields
	req := ClientConfigRequest{
		GatewayName:      "minimal-gateway",
		GatewayEndpoint:  "vpn.example.com:51820",
		GatewayPublicKey: gatewayKeyPair.PublicKey,
		ClientPrivateKey: clientKeyPair.PrivateKey,
		ClientAddress:    "10.0.0.2/32",
		AllowedIPs:       []string{"0.0.0.0/0"},
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		UserEmail:        "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Should not contain optional fields
	if strings.Contains(content, "PresharedKey") {
		t.Error("Config should not contain PresharedKey when not provided")
	}
	if strings.Contains(content, "PersistentKeepalive") {
		t.Error("Config should not contain PersistentKeepalive when not provided")
	}
	if strings.Contains(content, "DNS") {
		t.Error("Config should not contain DNS when not provided")
	}
}

func TestGenerateClientConfigFullTunnel(t *testing.T) {
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Full tunnel config (all traffic routed through VPN)
	req := ClientConfigRequest{
		GatewayName:      "full-tunnel-gateway",
		GatewayEndpoint:  "vpn.example.com:51820",
		GatewayPublicKey: gatewayKeyPair.PublicKey,
		ClientPrivateKey: clientKeyPair.PrivateKey,
		ClientAddress:    "10.0.0.2/32",
		AllowedIPs:       []string{"0.0.0.0/0", "::/0"},
		DNS:              []string{"1.1.1.1"},
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		UserEmail:        "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Verify full tunnel AllowedIPs
	if !strings.Contains(content, "AllowedIPs = 0.0.0.0/0, ::/0") {
		t.Error("Config should contain full tunnel AllowedIPs")
	}
}

func TestGenerateClientConfigDefaultAllowedIPs(t *testing.T) {
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Empty AllowedIPs should default to full tunnel
	req := ClientConfigRequest{
		GatewayName:      "default-gateway",
		GatewayEndpoint:  "vpn.example.com:51820",
		GatewayPublicKey: gatewayKeyPair.PublicKey,
		ClientPrivateKey: clientKeyPair.PrivateKey,
		ClientAddress:    "10.0.0.2/32",
		AllowedIPs:       []string{}, // Empty
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		UserEmail:        "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Should default to full tunnel when AllowedIPs is empty
	if !strings.Contains(content, "AllowedIPs = 0.0.0.0/0, ::/0") {
		t.Error("Config should default to full tunnel when AllowedIPs is empty")
	}
}

func TestGenerateClientConfigWithIPv6Networks(t *testing.T) {
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	// Test with both IPv4 and specific IPv6 networks (split tunnel)
	req := ClientConfigRequest{
		GatewayName:         "ipv6-gateway",
		GatewayEndpoint:     "vpn.example.com:51820",
		GatewayPublicKey:    gatewayKeyPair.PublicKey,
		ClientPrivateKey:    clientKeyPair.PrivateKey,
		ClientAddress:       "10.0.0.2/32",
		AllowedIPs:          []string{"10.0.0.0/24", "192.168.1.0/24", "2001:db8::/32", "fd00::/8"},
		DNS:                 []string{"1.1.1.1", "2606:4700:4700::1111"},
		PresharedKey:        psk,
		PersistentKeepalive: 25,
		ExpiresAt:           time.Now().Add(24 * time.Hour),
		UserEmail:           "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Verify IPv4 and IPv6 networks are in AllowedIPs
	if !strings.Contains(content, "10.0.0.0/24") {
		t.Error("Config should contain IPv4 network 10.0.0.0/24")
	}

	if !strings.Contains(content, "192.168.1.0/24") {
		t.Error("Config should contain IPv4 network 192.168.1.0/24")
	}

	if !strings.Contains(content, "2001:db8::/32") {
		t.Error("Config should contain IPv6 network 2001:db8::/32")
	}

	if !strings.Contains(content, "fd00::/8") {
		t.Error("Config should contain IPv6 network fd00::/8")
	}

	// Verify IPv6 DNS server
	if !strings.Contains(content, "2606:4700:4700::1111") {
		t.Error("Config should contain IPv6 DNS server")
	}
}

func TestGenerateClientConfigIPv6Only(t *testing.T) {
	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Test with only IPv6 networks (no IPv4)
	req := ClientConfigRequest{
		GatewayName:      "ipv6-only-gateway",
		GatewayEndpoint:  "[2001:db8::1]:51820", // IPv6 endpoint
		GatewayPublicKey: gatewayKeyPair.PublicKey,
		ClientPrivateKey: clientKeyPair.PrivateKey,
		ClientAddress:    "fd00::2/128", // IPv6 client address
		AllowedIPs:       []string{"2001:db8::/32", "fd00::/8"},
		DNS:              []string{"2606:4700:4700::1111"},
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		UserEmail:        "test@example.com",
	}

	config, err := GenerateClientConfig(req)
	if err != nil {
		t.Fatalf("GenerateClientConfig failed: %v", err)
	}

	content := string(config.Content)

	// Verify IPv6 address
	if !strings.Contains(content, "Address = fd00::2/128") {
		t.Error("Config should contain IPv6 client address")
	}

	// Verify IPv6 endpoint
	if !strings.Contains(content, "Endpoint = [2001:db8::1]:51820") {
		t.Error("Config should contain IPv6 endpoint")
	}

	// Verify only IPv6 in AllowedIPs
	if !strings.Contains(content, "2001:db8::/32") {
		t.Error("Config should contain IPv6 network")
	}

	// Verify no IPv4 addresses
	if strings.Contains(content, "0.0.0.0") {
		t.Error("Config should not contain IPv4 full tunnel when only IPv6 is specified")
	}
}

func TestGenerateServerConfig(t *testing.T) {
	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	req := ServerConfigRequest{
		InterfaceName: "wg0",
		PrivateKey:    gatewayKeyPair.PrivateKey,
		Address:       "10.0.0.1/24",
		ListenPort:    51820,
	}

	config, err := GenerateServerConfig(req, nil)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	// Verify Interface section
	if !strings.Contains(content, "[Interface]") {
		t.Error("Config should contain [Interface] section")
	}
	if !strings.Contains(content, "PrivateKey = "+gatewayKeyPair.PrivateKey) {
		t.Error("Config should contain server private key")
	}
	if !strings.Contains(content, "Address = 10.0.0.1/24") {
		t.Error("Config should contain server address")
	}
	if !strings.Contains(content, "ListenPort = 51820") {
		t.Error("Config should contain listen port")
	}
}

func TestGenerateServerConfigWithPeers(t *testing.T) {
	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	req := ServerConfigRequest{
		InterfaceName: "wg0",
		PrivateKey:    gatewayKeyPair.PrivateKey,
		Address:       "10.0.0.1/24",
		ListenPort:    51820,
	}

	peers := []PeerConfig{
		{
			PublicKey:    clientKeyPair.PublicKey,
			PresharedKey: psk,
			AllowedIPs:   []string{"10.0.0.2/32"},
			Comment:      "user@example.com",
		},
	}

	config, err := GenerateServerConfig(req, peers)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	// Verify peer section
	if !strings.Contains(content, "[Peer]") {
		t.Error("Config should contain [Peer] section")
	}
	if !strings.Contains(content, "PublicKey = "+clientKeyPair.PublicKey) {
		t.Error("Config should contain peer public key")
	}
	if !strings.Contains(content, "AllowedIPs = 10.0.0.2/32") {
		t.Error("Config should contain peer allowed IPs")
	}
	if !strings.Contains(content, "PresharedKey = "+psk) {
		t.Error("Config should contain peer preshared key")
	}
	if !strings.Contains(content, "# user@example.com") {
		t.Error("Config should contain peer comment")
	}
}

func TestGenerateServerConfigNoPSK(t *testing.T) {
	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	clientKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	req := ServerConfigRequest{
		InterfaceName: "wg0",
		PrivateKey:    gatewayKeyPair.PrivateKey,
		Address:       "10.0.0.1/24",
		ListenPort:    51820,
	}

	peers := []PeerConfig{
		{
			PublicKey:  clientKeyPair.PublicKey,
			AllowedIPs: []string{"10.0.0.2/32"},
			Comment:    "user@example.com",
		},
	}

	config, err := GenerateServerConfig(req, peers)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if strings.Contains(content, "PresharedKey") {
		t.Error("Config should not contain PresharedKey when not provided")
	}
}

func TestGenerateServerConfigWithPostScripts(t *testing.T) {
	gatewayKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	req := ServerConfigRequest{
		InterfaceName: "wg0",
		PrivateKey:    gatewayKeyPair.PrivateKey,
		Address:       "10.0.0.1/24",
		ListenPort:    51820,
		PostUp:        "iptables -A FORWARD -i %i -j ACCEPT",
		PostDown:      "iptables -D FORWARD -i %i -j ACCEPT",
	}

	config, err := GenerateServerConfig(req, nil)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "PostUp = iptables -A FORWARD -i %i -j ACCEPT") {
		t.Error("Config should contain PostUp script")
	}
	if !strings.Contains(content, "PostDown = iptables -D FORWARD -i %i -j ACCEPT") {
		t.Error("Config should contain PostDown script")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with-spaces"},
		{"with/slashes", "with-slashes"},
		{"with:colons", "with-colons"},
		{"with*star", "with-star"},
		{"complex name/with:many*bad?chars", "complex-name-with-many-bad-chars"},
	}

	for _, tc := range tests {
		result := sanitizeFileName(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeFileName(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify default constants are reasonable
	if DefaultPersistentKeepalive != 25 {
		t.Errorf("DefaultPersistentKeepalive should be 25, got %d", DefaultPersistentKeepalive)
	}
	if DefaultListenPort != 51820 {
		t.Errorf("DefaultListenPort should be 51820, got %d", DefaultListenPort)
	}
}
