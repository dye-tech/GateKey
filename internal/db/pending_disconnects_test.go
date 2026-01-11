package db

import (
	"testing"
	"time"
)

func TestNewPendingDisconnectStore(t *testing.T) {
	store := NewPendingDisconnectStore(nil)

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

func TestPendingDisconnect_Struct(t *testing.T) {
	now := time.Now()
	gatewayID := "gw-123"
	connectionID := "conn-456"
	vpnAddress := "10.0.0.1"
	executedAt := now.Add(time.Minute)
	executedBy := "gw-789"

	pd := PendingDisconnect{
		ID:           "pd-123",
		UserID:       "user-123",
		UserEmail:    "test@example.com",
		GatewayID:    &gatewayID,
		NodeType:     "gateway",
		ConnectionID: &connectionID,
		VPNAddress:   &vpnAddress,
		Reason:       "Test disconnect",
		RequestedBy:  "admin@example.com",
		RequestedAt:  now,
		ExecutedAt:   &executedAt,
		ExecutedBy:   &executedBy,
		ExpiresAt:    now.Add(5 * time.Minute),
		CreatedAt:    now,
	}

	// Verify all fields are set correctly
	if pd.ID != "pd-123" {
		t.Errorf("Expected ID 'pd-123', got %q", pd.ID)
	}
	if pd.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got %q", pd.UserID)
	}
	if pd.UserEmail != "test@example.com" {
		t.Errorf("Expected UserEmail 'test@example.com', got %q", pd.UserEmail)
	}
	if pd.GatewayID == nil || *pd.GatewayID != "gw-123" {
		t.Errorf("Expected GatewayID 'gw-123', got %v", pd.GatewayID)
	}
	if pd.NodeType != "gateway" {
		t.Errorf("Expected NodeType 'gateway', got %q", pd.NodeType)
	}
	if pd.ConnectionID == nil || *pd.ConnectionID != "conn-456" {
		t.Errorf("Expected ConnectionID 'conn-456', got %v", pd.ConnectionID)
	}
	if pd.VPNAddress == nil || *pd.VPNAddress != "10.0.0.1" {
		t.Errorf("Expected VPNAddress '10.0.0.1', got %v", pd.VPNAddress)
	}
	if pd.Reason != "Test disconnect" {
		t.Errorf("Expected Reason 'Test disconnect', got %q", pd.Reason)
	}
	if pd.RequestedBy != "admin@example.com" {
		t.Errorf("Expected RequestedBy 'admin@example.com', got %q", pd.RequestedBy)
	}
}

func TestDisconnectRequest_Struct(t *testing.T) {
	gatewayID := "gw-123"
	connectionID := "conn-456"
	vpnAddress := "10.0.0.1"

	req := DisconnectRequest{
		UserID:       "user-123",
		UserEmail:    "test@example.com",
		GatewayID:    &gatewayID,
		NodeType:     "gateway",
		ConnectionID: &connectionID,
		VPNAddress:   &vpnAddress,
		Reason:       "Test disconnect",
		RequestedBy:  "admin@example.com",
	}

	// Verify all fields are set correctly
	if req.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got %q", req.UserID)
	}
	if req.UserEmail != "test@example.com" {
		t.Errorf("Expected UserEmail 'test@example.com', got %q", req.UserEmail)
	}
	if req.GatewayID == nil || *req.GatewayID != "gw-123" {
		t.Errorf("Expected GatewayID 'gw-123', got %v", req.GatewayID)
	}
	if req.NodeType != "gateway" {
		t.Errorf("Expected NodeType 'gateway', got %q", req.NodeType)
	}
	if req.ConnectionID == nil || *req.ConnectionID != "conn-456" {
		t.Errorf("Expected ConnectionID 'conn-456', got %v", req.ConnectionID)
	}
	if req.VPNAddress == nil || *req.VPNAddress != "10.0.0.1" {
		t.Errorf("Expected VPNAddress '10.0.0.1', got %v", req.VPNAddress)
	}
	if req.Reason != "Test disconnect" {
		t.Errorf("Expected Reason 'Test disconnect', got %q", req.Reason)
	}
	if req.RequestedBy != "admin@example.com" {
		t.Errorf("Expected RequestedBy 'admin@example.com', got %q", req.RequestedBy)
	}
}

func TestPendingDisconnect_NilOptionalFields(t *testing.T) {
	now := time.Now()

	// Test with nil optional fields
	pd := PendingDisconnect{
		ID:           "pd-123",
		UserID:       "user-123",
		UserEmail:    "test@example.com",
		GatewayID:    nil, // nil = all gateways
		NodeType:     "gateway",
		ConnectionID: nil,
		VPNAddress:   nil,
		Reason:       "Test disconnect",
		RequestedBy:  "admin@example.com",
		RequestedAt:  now,
		ExecutedAt:   nil, // not yet executed
		ExecutedBy:   nil,
		ExpiresAt:    now.Add(5 * time.Minute),
		CreatedAt:    now,
	}

	if pd.GatewayID != nil {
		t.Error("Expected GatewayID to be nil")
	}
	if pd.ConnectionID != nil {
		t.Error("Expected ConnectionID to be nil")
	}
	if pd.VPNAddress != nil {
		t.Error("Expected VPNAddress to be nil")
	}
	if pd.ExecutedAt != nil {
		t.Error("Expected ExecutedAt to be nil")
	}
	if pd.ExecutedBy != nil {
		t.Error("Expected ExecutedBy to be nil")
	}
}

func TestDisconnectRequest_NilOptionalFields(t *testing.T) {
	// Test with nil optional fields
	req := DisconnectRequest{
		UserID:       "user-123",
		UserEmail:    "test@example.com",
		GatewayID:    nil, // nil = all gateways
		NodeType:     "hub",
		ConnectionID: nil,
		VPNAddress:   nil,
		Reason:       "Test disconnect",
		RequestedBy:  "admin@example.com",
	}

	if req.GatewayID != nil {
		t.Error("Expected GatewayID to be nil")
	}
	if req.ConnectionID != nil {
		t.Error("Expected ConnectionID to be nil")
	}
	if req.VPNAddress != nil {
		t.Error("Expected VPNAddress to be nil")
	}
	if req.NodeType != "hub" {
		t.Errorf("Expected NodeType 'hub', got %q", req.NodeType)
	}
}

func TestPendingDisconnect_ValidNodeTypes(t *testing.T) {
	validNodeTypes := []string{"gateway", "hub"}

	for _, nodeType := range validNodeTypes {
		pd := PendingDisconnect{
			NodeType: nodeType,
		}
		if pd.NodeType != nodeType {
			t.Errorf("Expected NodeType %q, got %q", nodeType, pd.NodeType)
		}
	}
}
