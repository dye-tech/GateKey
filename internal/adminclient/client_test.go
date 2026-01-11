package adminclient

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := &Config{
		ServerURL: "https://example.com",
	}
	client := NewClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestDisableUserResponse_Struct(t *testing.T) {
	resp := DisableUserResponse{
		Success:            true,
		Message:            "User disabled successfully",
		UserEmail:          "test@example.com",
		UserType:           "sso",
		SessionsTerminated: 3,
		ConfigsRevoked:     5,
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}
	if resp.Message != "User disabled successfully" {
		t.Errorf("Expected Message 'User disabled successfully', got %q", resp.Message)
	}
	if resp.UserEmail != "test@example.com" {
		t.Errorf("Expected UserEmail 'test@example.com', got %q", resp.UserEmail)
	}
	if resp.UserType != "sso" {
		t.Errorf("Expected UserType 'sso', got %q", resp.UserType)
	}
	if resp.SessionsTerminated != 3 {
		t.Errorf("Expected SessionsTerminated 3, got %d", resp.SessionsTerminated)
	}
	if resp.ConfigsRevoked != 5 {
		t.Errorf("Expected ConfigsRevoked 5, got %d", resp.ConfigsRevoked)
	}
}

func TestEnableUserResponse_Struct(t *testing.T) {
	resp := EnableUserResponse{
		Success:   true,
		Message:   "User enabled successfully",
		UserEmail: "test@example.com",
		UserType:  "local",
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}
	if resp.Message != "User enabled successfully" {
		t.Errorf("Expected Message 'User enabled successfully', got %q", resp.Message)
	}
	if resp.UserEmail != "test@example.com" {
		t.Errorf("Expected UserEmail 'test@example.com', got %q", resp.UserEmail)
	}
	if resp.UserType != "local" {
		t.Errorf("Expected UserType 'local', got %q", resp.UserType)
	}
}

func TestUserSession_Struct(t *testing.T) {
	connectedAt := time.Now()

	session := UserSession{
		ID:          "session-123",
		GatewayID:   "gw-456",
		GatewayName: "production-gateway",
		ClientIP:    "192.168.1.100",
		VPNAddress:  "10.0.0.50",
		NodeType:    "gateway",
		ConnectedAt: connectedAt,
	}

	if session.ID != "session-123" {
		t.Errorf("Expected ID 'session-123', got %q", session.ID)
	}
	if session.GatewayID != "gw-456" {
		t.Errorf("Expected GatewayID 'gw-456', got %q", session.GatewayID)
	}
	if session.GatewayName != "production-gateway" {
		t.Errorf("Expected GatewayName 'production-gateway', got %q", session.GatewayName)
	}
	if session.ClientIP != "192.168.1.100" {
		t.Errorf("Expected ClientIP '192.168.1.100', got %q", session.ClientIP)
	}
	if session.VPNAddress != "10.0.0.50" {
		t.Errorf("Expected VPNAddress '10.0.0.50', got %q", session.VPNAddress)
	}
	if session.NodeType != "gateway" {
		t.Errorf("Expected NodeType 'gateway', got %q", session.NodeType)
	}
	if !session.ConnectedAt.Equal(connectedAt) {
		t.Errorf("Expected ConnectedAt %v, got %v", connectedAt, session.ConnectedAt)
	}
}

func TestUserSession_HubNodeType(t *testing.T) {
	session := UserSession{
		ID:          "session-789",
		GatewayID:   "hub-123",
		GatewayName: "mesh-hub-1",
		NodeType:    "hub",
	}

	if session.NodeType != "hub" {
		t.Errorf("Expected NodeType 'hub', got %q", session.NodeType)
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := &Config{
		ServerURL: "https://gatekey.example.com",
		APIKey:    "test-api-key",
	}

	if cfg.ServerURL != "https://gatekey.example.com" {
		t.Errorf("Expected ServerURL 'https://gatekey.example.com', got %q", cfg.ServerURL)
	}
	if cfg.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got %q", cfg.APIKey)
	}
}
