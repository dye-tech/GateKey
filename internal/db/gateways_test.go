package db

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Token should be base64 URL encoded
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Errorf("Token is not valid base64 URL encoding: %v", err)
	}

	// Decoded token should be 32 bytes
	if len(decoded) != 32 {
		t.Errorf("Expected decoded token to be 32 bytes, got %d", len(decoded))
	}
}

func TestGenerateToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestGatewayErrors(t *testing.T) {
	if ErrGatewayNotFound.Error() == "" {
		t.Error("ErrGatewayNotFound should have an error message")
	}
	if ErrGatewayExists.Error() == "" {
		t.Error("ErrGatewayExists should have an error message")
	}

	// Verify error messages are distinct
	if ErrGatewayNotFound.Error() == ErrGatewayExists.Error() {
		t.Error("Error messages should be distinct")
	}
}

func TestGatewayConstants(t *testing.T) {
	// Test default values
	if DefaultVPNSubnet != "172.31.255.0/24" {
		t.Errorf("Expected DefaultVPNSubnet to be '172.31.255.0/24', got %q", DefaultVPNSubnet)
	}

	if DefaultWGListenPort != 51820 {
		t.Errorf("Expected DefaultWGListenPort to be 51820, got %d", DefaultWGListenPort)
	}
}

func TestGatewayTypeConstants(t *testing.T) {
	if GatewayTypeOpenVPN != "openvpn" {
		t.Errorf("Expected GatewayTypeOpenVPN to be 'openvpn', got %q", GatewayTypeOpenVPN)
	}

	if GatewayTypeWireGuard != "wireguard" {
		t.Errorf("Expected GatewayTypeWireGuard to be 'wireguard', got %q", GatewayTypeWireGuard)
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

func TestGateway_Fields(t *testing.T) {
	now := time.Now()
	gw := Gateway{
		ID:             "gw-uuid-123",
		Name:           "Production Gateway",
		GatewayType:    GatewayTypeOpenVPN,
		Hostname:       "vpn.example.com",
		PublicIP:       "203.0.113.50",
		VPNPort:        1194,
		VPNProtocol:    "udp",
		CryptoProfile:  CryptoProfileModern,
		VPNSubnet:      "10.8.0.0/24",
		TLSAuthEnabled: true,
		TLSAuthKey:     "tls-auth-key-content",
		FullTunnelMode: false,
		PushDNS:        true,
		DNSServers:     []string{"8.8.8.8", "8.8.4.4"},
		ConfigVersion:  "abc123",
		Token:          "gateway-token",
		PublicKey:      "gateway-public-key",
		IsActive:       true,
		LastHeartbeat:  &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if gw.ID != "gw-uuid-123" {
		t.Errorf("Expected ID 'gw-uuid-123', got %q", gw.ID)
	}
	if gw.Name != "Production Gateway" {
		t.Errorf("Expected Name 'Production Gateway', got %q", gw.Name)
	}
	if gw.GatewayType != GatewayTypeOpenVPN {
		t.Errorf("Expected GatewayType 'openvpn', got %q", gw.GatewayType)
	}
	if gw.VPNPort != 1194 {
		t.Errorf("Expected VPNPort 1194, got %d", gw.VPNPort)
	}
	if len(gw.DNSServers) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(gw.DNSServers))
	}
	if !gw.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestGateway_WireGuardFields(t *testing.T) {
	gw := Gateway{
		ID:           "wg-uuid-123",
		Name:         "WireGuard Gateway",
		GatewayType:  GatewayTypeWireGuard,
		WGPrivateKey: "wg-private-key",
		WGPublicKey:  "wg-public-key",
		WGListenPort: 51821,
	}

	if gw.GatewayType != GatewayTypeWireGuard {
		t.Errorf("Expected GatewayType 'wireguard', got %q", gw.GatewayType)
	}
	if gw.WGPrivateKey != "wg-private-key" {
		t.Errorf("Expected WGPrivateKey 'wg-private-key', got %q", gw.WGPrivateKey)
	}
	if gw.WGPublicKey != "wg-public-key" {
		t.Errorf("Expected WGPublicKey 'wg-public-key', got %q", gw.WGPublicKey)
	}
	if gw.WGListenPort != 51821 {
		t.Errorf("Expected WGListenPort 51821, got %d", gw.WGListenPort)
	}
}

func TestNewGatewayStore(t *testing.T) {
	store := NewGatewayStore(nil)
	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

func TestGatewayUser_Fields(t *testing.T) {
	now := time.Now()
	user := GatewayUser{
		UserID:    "user-uuid-123",
		Email:     "user@example.com",
		Name:      "Test User",
		CreatedAt: now,
	}

	if user.UserID != "user-uuid-123" {
		t.Errorf("Expected UserID 'user-uuid-123', got %q", user.UserID)
	}
	if user.Email != "user@example.com" {
		t.Errorf("Expected Email 'user@example.com', got %q", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("Expected Name 'Test User', got %q", user.Name)
	}
}

func TestGatewayGroup_Fields(t *testing.T) {
	now := time.Now()
	group := GatewayGroup{
		GroupName: "Engineering",
		CreatedAt: now,
	}

	if group.GroupName != "Engineering" {
		t.Errorf("Expected GroupName 'Engineering', got %q", group.GroupName)
	}
}

func TestGateway_LastHeartbeat_Nil(t *testing.T) {
	gw := Gateway{
		ID:            "gw-uuid",
		Name:          "Test Gateway",
		LastHeartbeat: nil, // No heartbeat yet
	}

	if gw.LastHeartbeat != nil {
		t.Error("Expected LastHeartbeat to be nil")
	}
}

func TestGateway_DNSServers_Empty(t *testing.T) {
	gw := Gateway{
		ID:         "gw-uuid",
		Name:       "Test Gateway",
		PushDNS:    false,
		DNSServers: nil,
	}

	if len(gw.DNSServers) != 0 {
		t.Errorf("Expected empty DNSServers, got %d", len(gw.DNSServers))
	}
}

func TestGenerateToken_EncodingConsistency(t *testing.T) {
	// Generate multiple tokens and verify they all use URL-safe base64
	for i := 0; i < 10; i++ {
		token, err := GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// URL-safe base64 should not contain + or /
		for _, c := range token {
			if c == '+' || c == '/' {
				t.Errorf("Token contains non-URL-safe characters: %s", token)
				break
			}
		}
	}
}

func TestGateway_AllCryptoProfiles(t *testing.T) {
	profiles := []string{CryptoProfileModern, CryptoProfileFIPS, CryptoProfileCompatible}

	for _, profile := range profiles {
		gw := Gateway{
			ID:            "gw-uuid",
			Name:          "Test Gateway",
			CryptoProfile: profile,
		}

		if gw.CryptoProfile == "" {
			t.Errorf("Expected CryptoProfile to be set to %q", profile)
		}
	}
}

func TestGateway_AllGatewayTypes(t *testing.T) {
	types := []string{GatewayTypeOpenVPN, GatewayTypeWireGuard}

	for _, gwType := range types {
		gw := Gateway{
			ID:          "gw-uuid",
			Name:        "Test Gateway",
			GatewayType: gwType,
		}

		if gw.GatewayType == "" {
			t.Errorf("Expected GatewayType to be set to %q", gwType)
		}
	}
}
