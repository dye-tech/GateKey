package session

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewAgentClient(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.controlPlaneURL != cfg.ControlPlaneURL {
		t.Errorf("Expected control plane URL %q, got %q", cfg.ControlPlaneURL, client.controlPlaneURL)
	}
	if client.token != cfg.Token {
		t.Errorf("Expected token %q, got %q", cfg.Token, client.token)
	}
	if client.nodeType != cfg.NodeType {
		t.Errorf("Expected node type %q, got %q", cfg.NodeType, client.nodeType)
	}
	if client.nodeID != cfg.NodeID {
		t.Errorf("Expected node ID %q, got %q", cfg.NodeID, client.nodeID)
	}
	if client.nodeName != cfg.NodeName {
		t.Errorf("Expected node name %q, got %q", cfg.NodeName, client.nodeName)
	}
}

func TestAgentClient_IsConnected_Initial(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	// Initially not connected
	if client.IsConnected() {
		t.Error("Expected client to not be connected initially")
	}
}

func TestAgentClient_GetAgentID_Initial(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	// Initially empty agent ID
	if client.GetAgentID() != "" {
		t.Errorf("Expected empty agent ID initially, got %q", client.GetAgentID())
	}
}

func TestAgentClient_Stop_NotStarted(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	// Stop should not panic even if not started
	client.Stop()

	// Calling Stop again should also not panic
	client.Stop()
}

func TestAgentClient_SendChannelSize(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	// Verify send channel has expected buffer size
	if cap(client.send) != 256 {
		t.Errorf("Expected send channel capacity 256, got %d", cap(client.send))
	}
}

func TestAgentClientConfig_AllNodeTypes(t *testing.T) {
	logger := zap.NewNop()
	nodeTypes := []string{"hub", "gateway", "spoke"}

	for _, nodeType := range nodeTypes {
		t.Run(nodeType, func(t *testing.T) {
			cfg := &AgentClientConfig{
				ControlPlaneURL: "https://control.example.com",
				Token:           "test-token",
				NodeType:        nodeType,
				NodeID:          nodeType + "-001",
				NodeName:        nodeType + " Node 1",
				Logger:          logger,
			}

			client := NewAgentClient(cfg)
			if client.nodeType != nodeType {
				t.Errorf("Expected node type %q, got %q", nodeType, client.nodeType)
			}
		})
	}
}

func TestIntPtr(t *testing.T) {
	ptr := intPtr(42)

	if ptr == nil {
		t.Fatal("Expected non-nil pointer")
	}
	if *ptr != 42 {
		t.Errorf("Expected value 42, got %d", *ptr)
	}

	// Test that modifying through pointer works
	*ptr = 100
	if *ptr != 100 {
		t.Errorf("Expected modified value 100, got %d", *ptr)
	}
}

func TestIntPtr_Zero(t *testing.T) {
	ptr := intPtr(0)

	if ptr == nil {
		t.Fatal("Expected non-nil pointer for zero value")
	}
	if *ptr != 0 {
		t.Errorf("Expected value 0, got %d", *ptr)
	}
}

func TestIntPtr_Negative(t *testing.T) {
	ptr := intPtr(-1)

	if ptr == nil {
		t.Fatal("Expected non-nil pointer for negative value")
	}
	if *ptr != -1 {
		t.Errorf("Expected value -1, got %d", *ptr)
	}
}

func TestAgentClient_DoneChannelInitialized(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          logger,
	}

	client := NewAgentClient(cfg)

	if client.done == nil {
		t.Error("Expected done channel to be initialized")
	}
}

func TestAgentClientConfig_EmptyFields(t *testing.T) {
	logger := zap.NewNop()
	cfg := &AgentClientConfig{
		ControlPlaneURL: "",
		Token:           "",
		NodeType:        "",
		NodeID:          "",
		NodeName:        "",
		Logger:          logger,
	}

	// Should not panic with empty config
	client := NewAgentClient(cfg)
	if client == nil {
		t.Fatal("Expected non-nil client even with empty config")
	}
}

func TestAgentClient_NilLogger(t *testing.T) {
	cfg := &AgentClientConfig{
		ControlPlaneURL: "https://control.example.com",
		Token:           "test-token",
		NodeType:        "gateway",
		NodeID:          "gw-001",
		NodeName:        "Gateway 1",
		Logger:          nil, // nil logger
	}

	// Should not panic with nil logger
	client := NewAgentClient(cfg)
	if client == nil {
		t.Fatal("Expected non-nil client even with nil logger")
	}
}

func TestAgentClient_URLSchemes(t *testing.T) {
	logger := zap.NewNop()
	urls := []struct {
		input    string
		expected string
	}{
		{"https://control.example.com", "https://control.example.com"},
		{"http://control.example.com", "http://control.example.com"},
		{"wss://control.example.com", "wss://control.example.com"},
		{"ws://control.example.com", "ws://control.example.com"},
	}

	for _, u := range urls {
		t.Run(u.input, func(t *testing.T) {
			cfg := &AgentClientConfig{
				ControlPlaneURL: u.input,
				Token:           "test-token",
				NodeType:        "gateway",
				NodeID:          "gw-001",
				NodeName:        "Gateway 1",
				Logger:          logger,
			}

			client := NewAgentClient(cfg)
			if client.controlPlaneURL != u.expected {
				t.Errorf("Expected URL %q, got %q", u.expected, client.controlPlaneURL)
			}
		})
	}
}
