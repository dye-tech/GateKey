package openvpn

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/models"
	"github.com/gatekey-project/gatekey/internal/pki"
)

func TestConfigGenerator_Generate(t *testing.T) {
	// Create a test CA
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Create config generator
	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	// Issue a test certificate
	certReq := pki.CertificateRequest{
		CommonName: "test-user",
		Email:      "test@example.com",
		ValidFor:   24 * time.Hour,
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Generate config
	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "test-gateway",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "udp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		DNS:         []string{"10.8.0.1"},
	}

	config, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	// Verify config content
	content := string(config.Content)

	if !strings.Contains(content, "client") {
		t.Error("Config should contain 'client' directive")
	}

	if !strings.Contains(content, "remote vpn.example.com 1194") {
		t.Error("Config should contain correct remote directive")
	}

	if !strings.Contains(content, "proto udp") {
		t.Error("Config should contain 'proto udp'")
	}

	if !strings.Contains(content, "<ca>") {
		t.Error("Config should contain embedded CA certificate")
	}

	if !strings.Contains(content, "<cert>") {
		t.Error("Config should contain embedded client certificate")
	}

	if !strings.Contains(content, "<key>") {
		t.Error("Config should contain embedded client key")
	}

	if !strings.Contains(content, "dhcp-option DNS 10.8.0.1") {
		t.Error("Config should contain DNS option")
	}

	// Verify filename
	if !strings.HasPrefix(config.FileName, "gatekey-test-gateway-") {
		t.Errorf("Unexpected filename: %s", config.FileName)
	}

	if !strings.HasSuffix(config.FileName, ".ovpn") {
		t.Error("Filename should end with .ovpn")
	}
}

func TestConfigGenerator_WithRoutes(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "test-gateway",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "tcp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Routes: []Route{
			{Network: "10.0.0.0", Netmask: "255.0.0.0"},
			{Network: "192.168.1.0", Netmask: "255.255.255.0"},
		},
	}

	config, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(config.Content)

	if !strings.Contains(content, "route 10.0.0.0 255.0.0.0") {
		t.Error("Config should contain first route")
	}

	if !strings.Contains(content, "route 192.168.1.0 255.255.255.0") {
		t.Error("Config should contain second route")
	}

	if !strings.Contains(content, "proto tcp") {
		t.Error("Config should contain 'proto tcp'")
	}
}

func TestConfigGenerator_WithIPv6Routes(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "test-gateway-ipv6",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "udp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	// Test with both IPv4 and IPv6 routes
	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Routes: []Route{
			{Network: "10.0.0.0", Netmask: "255.0.0.0"},
			{Network: "192.168.1.0", Netmask: "255.255.255.0"},
		},
		RoutesV6: []RouteV6{
			{CIDR: "2001:db8::/32"},
			{CIDR: "fd00::/8"},
		},
	}

	config, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(config.Content)

	// Verify IPv4 routes
	if !strings.Contains(content, "route 10.0.0.0 255.0.0.0") {
		t.Error("Config should contain first IPv4 route")
	}

	if !strings.Contains(content, "route 192.168.1.0 255.255.255.0") {
		t.Error("Config should contain second IPv4 route")
	}

	// Verify IPv6 routes
	if !strings.Contains(content, "route-ipv6 2001:db8::/32") {
		t.Error("Config should contain first IPv6 route")
	}

	if !strings.Contains(content, "route-ipv6 fd00::/8") {
		t.Error("Config should contain second IPv6 route")
	}
}

func TestConfigGenerator_IPv6OnlyRoutes(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "test-gateway-ipv6-only",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "udp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	// Test with only IPv6 routes (no IPv4)
	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Routes:      []Route{}, // No IPv4 routes
		RoutesV6: []RouteV6{
			{CIDR: "2001:db8:1::/48"},
		},
	}

	config, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(config.Content)

	// Should not contain any IPv4 route directive
	if strings.Contains(content, "route 10.") || strings.Contains(content, "route 192.") {
		t.Error("Config should not contain IPv4 routes when only IPv6 routes are provided")
	}

	// Verify IPv6 route is present
	if !strings.Contains(content, "route-ipv6 2001:db8:1::/48") {
		t.Error("Config should contain the IPv6 route")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with-space"},
		{"with/slash", "with-slash"},
		{"with:colon", "with-colon"},
		{"with*star", "with-star"},
		{"with?question", "with-question"},
		{"with\"quote", "withquote"},
		{"with<angle>", "withangle"},
		{"with|pipe", "withpipe"},
	}

	for _, test := range tests {
		result := sanitizeFileName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFileName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"vpn.example.com", false},
		{"example.com", false},
		{"localhost", false},
		{"", false},
		{"not-an-ip", false},
		{"192.168.1", false},
		{"192.168.1.256", false},
	}

	for _, test := range tests {
		result := isIPAddress(test.input)
		if result != test.expected {
			t.Errorf("isIPAddress(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestGetCryptoSettings_Modern(t *testing.T) {
	settings := GetCryptoSettings(CryptoProfileModern)

	if settings.Cipher != "AES-256-GCM" {
		t.Errorf("Expected Cipher 'AES-256-GCM', got %q", settings.Cipher)
	}
	if settings.Auth != "SHA256" {
		t.Errorf("Expected Auth 'SHA256', got %q", settings.Auth)
	}
	if settings.TLSVersionMin != "1.2" {
		t.Errorf("Expected TLSVersionMin '1.2', got %q", settings.TLSVersionMin)
	}
	if !strings.Contains(settings.DataCiphers, "CHACHA20-POLY1305") {
		t.Errorf("Expected DataCiphers to contain 'CHACHA20-POLY1305', got %q", settings.DataCiphers)
	}
	if settings.CryptoProfile == "" {
		t.Error("Expected non-empty CryptoProfile description")
	}
}

func TestGetCryptoSettings_FIPS(t *testing.T) {
	settings := GetCryptoSettings(CryptoProfileFIPS)

	if settings.Cipher != "AES-256-GCM" {
		t.Errorf("Expected Cipher 'AES-256-GCM', got %q", settings.Cipher)
	}
	if settings.Auth != "SHA384" {
		t.Errorf("Expected Auth 'SHA384', got %q", settings.Auth)
	}
	if settings.TLSVersionMin != "1.2" {
		t.Errorf("Expected TLSVersionMin '1.2', got %q", settings.TLSVersionMin)
	}
	// FIPS should not have ChaCha20
	if strings.Contains(settings.DataCiphers, "CHACHA") {
		t.Errorf("FIPS profile should not contain CHACHA, got %q", settings.DataCiphers)
	}
	if !strings.Contains(settings.CryptoProfile, "FIPS") {
		t.Errorf("Expected CryptoProfile to mention 'FIPS', got %q", settings.CryptoProfile)
	}
}

func TestGetCryptoSettings_Compatible(t *testing.T) {
	settings := GetCryptoSettings(CryptoProfileCompatible)

	if settings.Cipher != "AES-256-CBC" {
		t.Errorf("Expected Cipher 'AES-256-CBC' for compatibility, got %q", settings.Cipher)
	}
	// TLS 1.2 minimum enforced for security even in compatible mode
	// TLS 1.0/1.1 are deprecated and have known vulnerabilities
	if settings.TLSVersionMin != "1.2" {
		t.Errorf("Expected TLSVersionMin '1.2' (security requirement), got %q", settings.TLSVersionMin)
	}
	// Compatible should include CBC ciphers
	if !strings.Contains(settings.DataCiphers, "AES-256-CBC") {
		t.Errorf("Expected DataCiphers to contain 'AES-256-CBC', got %q", settings.DataCiphers)
	}
	if !strings.Contains(settings.CryptoProfile, "Compatible") {
		t.Errorf("Expected CryptoProfile to mention 'Compatible', got %q", settings.CryptoProfile)
	}
}

func TestGetCryptoSettings_Default(t *testing.T) {
	// Unknown profile should default to modern
	settings := GetCryptoSettings("unknown")
	modernSettings := GetCryptoSettings(CryptoProfileModern)

	if settings.Cipher != modernSettings.Cipher {
		t.Errorf("Default profile should match modern, got Cipher %q", settings.Cipher)
	}
	if settings.DataCiphers != modernSettings.DataCiphers {
		t.Errorf("Default profile should match modern, got DataCiphers %q", settings.DataCiphers)
	}
}

func TestCryptoProfileConstants(t *testing.T) {
	if CryptoProfileModern != "modern" {
		t.Errorf("Expected CryptoProfileModern to be 'modern', got %q", CryptoProfileModern)
	}
	if CryptoProfileFIPS != "fips" {
		t.Errorf("Expected CryptoProfileFIPS to be 'fips', got %q", CryptoProfileFIPS)
	}
	if CryptoProfileCompatible != "compatible" {
		t.Errorf("Expected CryptoProfileCompatible to be 'compatible', got %q", CryptoProfileCompatible)
	}
}

func TestGenerateTLSAuthKey(t *testing.T) {
	key, err := GenerateTLSAuthKey()
	if err != nil {
		t.Fatalf("GenerateTLSAuthKey failed: %v", err)
	}

	keyStr := string(key)

	// Verify the key format
	if !strings.Contains(keyStr, "-----BEGIN OpenVPN Static key V1-----") {
		t.Error("Key should contain BEGIN marker")
	}
	if !strings.Contains(keyStr, "-----END OpenVPN Static key V1-----") {
		t.Error("Key should contain END marker")
	}

	// Key should have 16 lines of hex (256 bytes = 16 * 16 bytes per line)
	lines := strings.Split(keyStr, "\n")
	hexLineCount := 0
	for _, line := range lines {
		// Each hex line should be exactly 32 characters (16 bytes as hex)
		if len(line) == 32 {
			// Verify it's all hex
			allHex := true
			for _, c := range line {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					allHex = false
					break
				}
			}
			if allHex {
				hexLineCount++
			}
		}
	}

	if hexLineCount != 16 {
		t.Errorf("Expected 16 hex lines, got %d", hexLineCount)
	}
}

func TestGenerateTLSAuthKey_Uniqueness(t *testing.T) {
	key1, err := GenerateTLSAuthKey()
	if err != nil {
		t.Fatalf("GenerateTLSAuthKey failed: %v", err)
	}

	key2, err := GenerateTLSAuthKey()
	if err != nil {
		t.Fatalf("GenerateTLSAuthKey failed: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("Two generated TLS-Auth keys should be different")
	}
}

func TestGenerateServerConfig(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "udp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		StatusLog:      "/var/log/openvpn-status.log",
		PushOptions:    []string{"route 10.0.0.0 255.0.0.0", "dhcp-option DNS 8.8.8.8"},
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	// Verify basic directives
	if !strings.Contains(content, "port 1194") {
		t.Error("Config should contain 'port 1194'")
	}
	if !strings.Contains(content, "proto udp") {
		t.Error("Config should contain 'proto udp'")
	}
	if !strings.Contains(content, "dev tun0") {
		t.Error("Config should contain 'dev tun0'")
	}
	if !strings.Contains(content, "server 10.8.0.0 255.255.255.0") {
		t.Error("Config should contain server directive")
	}
	if !strings.Contains(content, "ca /etc/openvpn/ca.crt") {
		t.Error("Config should contain CA path")
	}

	// Verify push options
	if !strings.Contains(content, "push \"route 10.0.0.0 255.0.0.0\"") {
		t.Error("Config should contain route push option")
	}
	if !strings.Contains(content, "push \"dhcp-option DNS 8.8.8.8\"") {
		t.Error("Config should contain DNS push option")
	}
}

func TestGenerateServerConfig_WithTLSAuth(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "udp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		TLSAuthPath:    "/etc/openvpn/ta.key",
		StatusLog:      "/var/log/openvpn-status.log",
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "tls-auth /etc/openvpn/ta.key 0") {
		t.Error("Config should contain tls-auth directive")
	}
}

func TestGenerateServerConfig_WithScripts(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "udp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		StatusLog:      "/var/log/openvpn-status.log",
		Scripts: ScriptPaths{
			AuthUserPassVerify: "/etc/openvpn/auth.sh",
			ClientConnect:      "/etc/openvpn/connect.sh",
			ClientDisconnect:   "/etc/openvpn/disconnect.sh",
		},
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "auth-user-pass-verify /etc/openvpn/auth.sh via-file") {
		t.Error("Config should contain auth-user-pass-verify script")
	}
	if !strings.Contains(content, "client-connect /etc/openvpn/connect.sh") {
		t.Error("Config should contain client-connect script")
	}
	if !strings.Contains(content, "client-disconnect /etc/openvpn/disconnect.sh") {
		t.Error("Config should contain client-disconnect script")
	}
	if !strings.Contains(content, "script-security 2") {
		t.Error("Config should contain script-security when scripts are used")
	}
}

func TestGenerateServerConfig_WithCRL(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "tcp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		CRLPath:        "/etc/openvpn/crl.pem",
		StatusLog:      "/var/log/openvpn-status.log",
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "crl-verify /etc/openvpn/crl.pem") {
		t.Error("Config should contain crl-verify directive")
	}
	if !strings.Contains(content, "proto tcp") {
		t.Error("Config should contain 'proto tcp'")
	}
}

func TestGenerateServerConfig_WithManagement(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "udp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		ManagementAddr: "127.0.0.1",
		StatusLog:      "/var/log/openvpn-status.log",
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "management 127.0.0.1 7505") {
		t.Error("Config should contain management directive")
	}
}

func TestRoute_Struct(t *testing.T) {
	route := Route{
		Network: "10.0.0.0",
		Netmask: "255.0.0.0",
	}

	if route.Network != "10.0.0.0" {
		t.Errorf("Expected Network '10.0.0.0', got %q", route.Network)
	}
	if route.Netmask != "255.0.0.0" {
		t.Errorf("Expected Netmask '255.0.0.0', got %q", route.Netmask)
	}
}

func TestRouteV6_Struct(t *testing.T) {
	route := RouteV6{
		CIDR: "2001:db8::/32",
	}

	if route.CIDR != "2001:db8::/32" {
		t.Errorf("Expected CIDR '2001:db8::/32', got %q", route.CIDR)
	}
}

func TestGeneratedConfig_Struct(t *testing.T) {
	now := time.Now()
	cfg := GeneratedConfig{
		Content:   []byte("config content"),
		FileName:  "gatekey-test.ovpn",
		ExpiresAt: now,
	}

	if string(cfg.Content) != "config content" {
		t.Errorf("Expected Content 'config content', got %q", string(cfg.Content))
	}
	if cfg.FileName != "gatekey-test.ovpn" {
		t.Errorf("Expected FileName 'gatekey-test.ovpn', got %q", cfg.FileName)
	}
}

func TestScriptPaths_Struct(t *testing.T) {
	scripts := ScriptPaths{
		AuthUserPassVerify: "/path/to/auth.sh",
		TLSVerify:          "/path/to/tls.sh",
		ClientConnect:      "/path/to/connect.sh",
		ClientDisconnect:   "/path/to/disconnect.sh",
	}

	if scripts.AuthUserPassVerify != "/path/to/auth.sh" {
		t.Errorf("Expected AuthUserPassVerify '/path/to/auth.sh', got %q", scripts.AuthUserPassVerify)
	}
	if scripts.TLSVerify != "/path/to/tls.sh" {
		t.Errorf("Expected TLSVerify '/path/to/tls.sh', got %q", scripts.TLSVerify)
	}
}

func TestConfigGenerator_WithTLSAuth(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Generate a TLS auth key
	tlsAuthKey, err := GenerateTLSAuthKey()
	if err != nil {
		t.Fatalf("Failed to generate TLS auth key: %v", err)
	}

	generator, err := NewConfigGenerator(ca, tlsAuthKey)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
		Email:      "test@example.com",
		ValidFor:   24 * time.Hour,
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	gateway := &models.Gateway{
		ID:             uuid.New(),
		Name:           "test-gateway",
		Hostname:       "vpn.example.com",
		VPNPort:        1194,
		VPNProtocol:    "udp",
		TLSAuthEnabled: true,
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	cfg, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(cfg.Content)

	// Verify TLS-Auth is included
	if !strings.Contains(content, "<tls-auth>") {
		t.Error("Config should contain <tls-auth> when TLSAuthEnabled")
	}
	if !strings.Contains(content, "key-direction 1") {
		t.Error("Config should contain 'key-direction 1' for client")
	}
}

func TestConfigGenerator_FallbackToPublicIP(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Gateway with no Hostname, only PublicIP
	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "ip-only-gateway",
		Hostname:    "", // Empty hostname
		PublicIP:    "203.0.113.50",
		VPNPort:     1194,
		VPNProtocol: "udp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	cfg, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(cfg.Content)

	// Should use PublicIP when Hostname is empty
	if !strings.Contains(content, "remote 203.0.113.50 1194") {
		t.Error("Config should use PublicIP when Hostname is empty")
	}
}

func TestConfigGenerator_DefaultProtocol(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Gateway with empty protocol
	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "no-protocol-gateway",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "", // Empty protocol
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	cfg, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(cfg.Content)

	// Should default to UDP
	if !strings.Contains(content, "proto udp") {
		t.Error("Config should default to 'proto udp' when VPNProtocol is empty")
	}
}

func TestConfigGenerator_WithCryptoProfile(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	gateway := &models.Gateway{
		ID:          uuid.New(),
		Name:        "fips-gateway",
		Hostname:    "vpn.example.com",
		VPNPort:     1194,
		VPNProtocol: "udp",
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	req := GenerateRequest{
		Gateway:       gateway,
		User:          user,
		Certificate:   issued,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		CryptoProfile: CryptoProfileFIPS,
	}

	cfg, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(cfg.Content)

	// FIPS profile should use SHA384
	if !strings.Contains(content, "auth SHA384") {
		t.Error("FIPS config should contain 'auth SHA384'")
	}
}

func TestConfigGenerator_WithTLSAuthFromRequest(t *testing.T) {
	pkiCfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := pki.NewCA(pkiCfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Create generator WITHOUT TLS auth
	generator, err := NewConfigGenerator(ca, nil)
	if err != nil {
		t.Fatalf("Failed to create config generator: %v", err)
	}

	certReq := pki.CertificateRequest{
		CommonName: "test-user",
	}

	issued, err := ca.IssueClientCertificate(certReq)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Generate a TLS auth key for the request
	tlsAuthKey, err := GenerateTLSAuthKey()
	if err != nil {
		t.Fatalf("Failed to generate TLS auth key: %v", err)
	}

	gateway := &models.Gateway{
		ID:             uuid.New(),
		Name:           "tls-auth-gateway",
		Hostname:       "vpn.example.com",
		VPNPort:        1194,
		VPNProtocol:    "udp",
		TLSAuthEnabled: true,
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}

	req := GenerateRequest{
		Gateway:     gateway,
		User:        user,
		Certificate: issued,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		TLSAuthKey:  string(tlsAuthKey), // Pass TLS auth key in request
	}

	cfg, err := generator.Generate(req)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	content := string(cfg.Content)

	// Should include TLS auth from request
	if !strings.Contains(content, "<tls-auth>") {
		t.Error("Config should contain <tls-auth> when passed via request")
	}
}

func TestGenerateServerConfig_WithTLSVerify(t *testing.T) {
	cfg := ServerConfig{
		Port:           1194,
		Protocol:       "udp",
		Device:         "tun0",
		ServerNetwork:  "10.8.0.0",
		ServerNetmask:  "255.255.255.0",
		CACertPath:     "/etc/openvpn/ca.crt",
		ServerCertPath: "/etc/openvpn/server.crt",
		ServerKeyPath:  "/etc/openvpn/server.key",
		DHPath:         "/etc/openvpn/dh.pem",
		StatusLog:      "/var/log/openvpn-status.log",
		Scripts: ScriptPaths{
			TLSVerify: "/etc/openvpn/tls-verify.sh",
		},
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "tls-verify /etc/openvpn/tls-verify.sh") {
		t.Error("Config should contain tls-verify script")
	}
}

func TestGenerateServerConfig_WithClientConfigDir(t *testing.T) {
	cfg := ServerConfig{
		Port:            1194,
		Protocol:        "udp",
		Device:          "tun0",
		ServerNetwork:   "10.8.0.0",
		ServerNetmask:   "255.255.255.0",
		CACertPath:      "/etc/openvpn/ca.crt",
		ServerCertPath:  "/etc/openvpn/server.crt",
		ServerKeyPath:   "/etc/openvpn/server.key",
		DHPath:          "/etc/openvpn/dh.pem",
		StatusLog:       "/var/log/openvpn-status.log",
		ClientConfigDir: "/etc/openvpn/ccd",
	}

	config, err := GenerateServerConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateServerConfig failed: %v", err)
	}

	content := string(config)

	if !strings.Contains(content, "client-config-dir /etc/openvpn/ccd") {
		t.Error("Config should contain client-config-dir directive")
	}
}

// TestGetCryptoSettings_TLS12MinimumEnforced verifies that all crypto profiles
// enforce TLS 1.2 as the minimum version. This is a security requirement since
// TLS 1.0 and 1.1 have known vulnerabilities and are deprecated.
func TestGetCryptoSettings_TLS12MinimumEnforced(t *testing.T) {
	profiles := []string{
		CryptoProfileModern,
		CryptoProfileFIPS,
		CryptoProfileCompatible,
	}

	for _, profile := range profiles {
		t.Run(profile, func(t *testing.T) {
			settings := GetCryptoSettings(profile)

			// All profiles must enforce TLS 1.2 minimum
			if settings.TLSVersionMin != "1.2" && settings.TLSVersionMin != "1.3" {
				t.Errorf("Profile %q: TLSVersionMin must be at least '1.2', got %q. "+
					"TLS 1.0/1.1 are deprecated and have known vulnerabilities.",
					profile, settings.TLSVersionMin)
			}
		})
	}
}
