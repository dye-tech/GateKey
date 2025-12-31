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
