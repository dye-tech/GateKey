package session

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMessageTypes(t *testing.T) {
	expectedTypes := map[string]string{
		"MsgTypeAuth":         "auth",
		"MsgTypeAuthResponse": "auth_response",
		"MsgTypeCommand":      "command",
		"MsgTypeOutput":       "output",
		"MsgTypePing":         "ping",
		"MsgTypePong":         "pong",
		"MsgTypeAgentList":    "agent_list",
		"MsgTypeConnectAgent": "connect_agent",
		"MsgTypeDisconnect":   "disconnect",
		"MsgTypeError":        "error",
	}

	if MsgTypeAuth != expectedTypes["MsgTypeAuth"] {
		t.Errorf("Expected MsgTypeAuth to be %q, got %q", expectedTypes["MsgTypeAuth"], MsgTypeAuth)
	}
	if MsgTypeAuthResponse != expectedTypes["MsgTypeAuthResponse"] {
		t.Errorf("Expected MsgTypeAuthResponse to be %q, got %q", expectedTypes["MsgTypeAuthResponse"], MsgTypeAuthResponse)
	}
	if MsgTypeCommand != expectedTypes["MsgTypeCommand"] {
		t.Errorf("Expected MsgTypeCommand to be %q, got %q", expectedTypes["MsgTypeCommand"], MsgTypeCommand)
	}
	if MsgTypeOutput != expectedTypes["MsgTypeOutput"] {
		t.Errorf("Expected MsgTypeOutput to be %q, got %q", expectedTypes["MsgTypeOutput"], MsgTypeOutput)
	}
	if MsgTypePing != expectedTypes["MsgTypePing"] {
		t.Errorf("Expected MsgTypePing to be %q, got %q", expectedTypes["MsgTypePing"], MsgTypePing)
	}
	if MsgTypePong != expectedTypes["MsgTypePong"] {
		t.Errorf("Expected MsgTypePong to be %q, got %q", expectedTypes["MsgTypePong"], MsgTypePong)
	}
	if MsgTypeAgentList != expectedTypes["MsgTypeAgentList"] {
		t.Errorf("Expected MsgTypeAgentList to be %q, got %q", expectedTypes["MsgTypeAgentList"], MsgTypeAgentList)
	}
	if MsgTypeConnectAgent != expectedTypes["MsgTypeConnectAgent"] {
		t.Errorf("Expected MsgTypeConnectAgent to be %q, got %q", expectedTypes["MsgTypeConnectAgent"], MsgTypeConnectAgent)
	}
	if MsgTypeDisconnect != expectedTypes["MsgTypeDisconnect"] {
		t.Errorf("Expected MsgTypeDisconnect to be %q, got %q", expectedTypes["MsgTypeDisconnect"], MsgTypeDisconnect)
	}
	if MsgTypeError != expectedTypes["MsgTypeError"] {
		t.Errorf("Expected MsgTypeError to be %q, got %q", expectedTypes["MsgTypeError"], MsgTypeError)
	}
}

func TestMessage_JSONMarshalling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	payload := json.RawMessage(`{"test": "value"}`)

	msg := Message{
		Type:      MsgTypeCommand,
		ID:        "msg-123",
		Payload:   payload,
		Timestamp: now,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var result Message
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if result.Type != MsgTypeCommand {
		t.Errorf("Expected type %q, got %q", MsgTypeCommand, result.Type)
	}
	if result.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got %q", result.ID)
	}
}

func TestAuthPayload_JSONMarshalling(t *testing.T) {
	auth := AuthPayload{
		Token:    "secret-token",
		NodeType: "gateway",
		NodeID:   "gw-001",
		NodeName: "Gateway 1",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("Failed to marshal auth payload: %v", err)
	}

	var result AuthPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal auth payload: %v", err)
	}

	if result.Token != auth.Token {
		t.Errorf("Expected token %q, got %q", auth.Token, result.Token)
	}
	if result.NodeType != auth.NodeType {
		t.Errorf("Expected node type %q, got %q", auth.NodeType, result.NodeType)
	}
	if result.NodeID != auth.NodeID {
		t.Errorf("Expected node ID %q, got %q", auth.NodeID, result.NodeID)
	}
	if result.NodeName != auth.NodeName {
		t.Errorf("Expected node name %q, got %q", auth.NodeName, result.NodeName)
	}
}

func TestAuthResponsePayload_JSONMarshalling(t *testing.T) {
	tests := []struct {
		name     string
		response AuthResponsePayload
	}{
		{
			name: "success",
			response: AuthResponsePayload{
				Success: true,
				Message: "Connected",
				AgentID: "agent-uuid-123",
			},
		},
		{
			name: "failure",
			response: AuthResponsePayload{
				Success: false,
				Message: "Invalid token",
				AgentID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var result AuthResponsePayload
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if result.Success != tt.response.Success {
				t.Errorf("Expected success %v, got %v", tt.response.Success, result.Success)
			}
			if result.Message != tt.response.Message {
				t.Errorf("Expected message %q, got %q", tt.response.Message, result.Message)
			}
		})
	}
}

func TestCommandPayload_JSONMarshalling(t *testing.T) {
	cmd := CommandPayload{
		Command: "ls -la /var/log",
		AgentID: "agent-001",
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result CommandPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Command != cmd.Command {
		t.Errorf("Expected command %q, got %q", cmd.Command, result.Command)
	}
	if result.AgentID != cmd.AgentID {
		t.Errorf("Expected agent ID %q, got %q", cmd.AgentID, result.AgentID)
	}
}

func TestOutputPayload_JSONMarshalling(t *testing.T) {
	exitCode := 0
	output := OutputPayload{
		Output:   "file1.txt\nfile2.txt\n",
		IsStderr: false,
		ExitCode: &exitCode,
		Done:     true,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result OutputPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Output != output.Output {
		t.Errorf("Expected output %q, got %q", output.Output, result.Output)
	}
	if result.IsStderr != output.IsStderr {
		t.Errorf("Expected IsStderr %v, got %v", output.IsStderr, result.IsStderr)
	}
	if result.ExitCode == nil || *result.ExitCode != 0 {
		t.Error("Expected exit code 0")
	}
	if !result.Done {
		t.Error("Expected Done to be true")
	}
}

func TestOutputPayload_NilExitCode(t *testing.T) {
	output := OutputPayload{
		Output:   "still running...",
		IsStderr: false,
		ExitCode: nil,
		Done:     false,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result OutputPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.ExitCode != nil {
		t.Error("Expected ExitCode to be nil for running command")
	}
	if result.Done {
		t.Error("Expected Done to be false for running command")
	}
}

func TestAgentInfo_Fields(t *testing.T) {
	now := time.Now()
	info := AgentInfo{
		AgentID:   "agent-uuid-123",
		NodeType:  "gateway",
		NodeID:    "gw-001",
		NodeName:  "Gateway 1",
		Connected: now,
	}

	if info.AgentID != "agent-uuid-123" {
		t.Errorf("Expected AgentID 'agent-uuid-123', got %q", info.AgentID)
	}
	if info.NodeType != "gateway" {
		t.Errorf("Expected NodeType 'gateway', got %q", info.NodeType)
	}
	if info.NodeID != "gw-001" {
		t.Errorf("Expected NodeID 'gw-001', got %q", info.NodeID)
	}
}

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	if manager.agents == nil {
		t.Error("Expected agents map to be initialized")
	}
	if manager.admins == nil {
		t.Error("Expected admins map to be initialized")
	}
	if manager.agentsByNode == nil {
		t.Error("Expected agentsByNode map to be initialized")
	}
	if manager.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestManager_GetConnectedAgents_Empty(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	agents := manager.GetConnectedAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestManager_GetAgentByNodeID_NotFound(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	_, exists := manager.GetAgentByNodeID("non-existent")
	if exists {
		t.Error("Expected agent not to be found")
	}
}

func TestManager_IsAgentConnected_NotConnected(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	if manager.IsAgentConnected("non-existent") {
		t.Error("Expected agent not to be connected")
	}
}

func TestManager_ValidateAgentToken_Callback(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// By default, ValidateAgentToken is nil
	if manager.ValidateAgentToken != nil {
		t.Error("Expected ValidateAgentToken to be nil by default")
	}

	// Set a custom validator
	validCalls := 0
	manager.ValidateAgentToken = func(nodeType, nodeID, token string) bool {
		validCalls++
		return nodeType == "gateway" && token == "valid-token"
	}

	// Test valid token
	if !manager.ValidateAgentToken("gateway", "gw-001", "valid-token") {
		t.Error("Expected valid token to pass")
	}

	// Test invalid token
	if manager.ValidateAgentToken("gateway", "gw-001", "invalid-token") {
		t.Error("Expected invalid token to fail")
	}

	// Test wrong node type
	if manager.ValidateAgentToken("hub", "hub-001", "valid-token") {
		t.Error("Expected wrong node type to fail")
	}

	if validCalls != 3 {
		t.Errorf("Expected 3 validation calls, got %d", validCalls)
	}
}

func TestManager_InternalMaps(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// Verify internal maps start empty
	if len(manager.agents) != 0 {
		t.Errorf("Expected empty agents map, got %d entries", len(manager.agents))
	}
	if len(manager.admins) != 0 {
		t.Errorf("Expected empty admins map, got %d entries", len(manager.admins))
	}
	if len(manager.agentsByNode) != 0 {
		t.Errorf("Expected empty agentsByNode map, got %d entries", len(manager.agentsByNode))
	}
}

func TestConnectedAgent_Fields(t *testing.T) {
	now := time.Now()
	agent := ConnectedAgent{
		Info: AgentInfo{
			AgentID:   "agent-123",
			NodeType:  "spoke",
			NodeID:    "spoke-001",
			NodeName:  "Spoke Node 1",
			Connected: now,
		},
		Send:     make(chan []byte, 256),
		Done:     make(chan struct{}),
		LastPing: now,
	}

	if agent.Info.AgentID != "agent-123" {
		t.Errorf("Expected AgentID 'agent-123', got %q", agent.Info.AgentID)
	}
	if agent.Info.NodeType != "spoke" {
		t.Errorf("Expected NodeType 'spoke', got %q", agent.Info.NodeType)
	}
}

func TestAdminSession_Fields(t *testing.T) {
	admin := AdminSession{
		ID:          "session-uuid-123",
		Send:        make(chan []byte, 256),
		Done:        make(chan struct{}),
		ConnectedTo: "agent-001",
		UserEmail:   "admin@example.com",
	}

	if admin.ID != "session-uuid-123" {
		t.Errorf("Expected ID 'session-uuid-123', got %q", admin.ID)
	}
	if admin.ConnectedTo != "agent-001" {
		t.Errorf("Expected ConnectedTo 'agent-001', got %q", admin.ConnectedTo)
	}
	if admin.UserEmail != "admin@example.com" {
		t.Errorf("Expected UserEmail 'admin@example.com', got %q", admin.UserEmail)
	}
}

func TestManager_Shutdown_Empty(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// Shutdown with no connections should not panic
	manager.Shutdown(nil)

	// Verify maps are reset
	if len(manager.agents) != 0 {
		t.Error("Expected agents map to be cleared")
	}
	if len(manager.admins) != 0 {
		t.Error("Expected admins map to be cleared")
	}
	if len(manager.agentsByNode) != 0 {
		t.Error("Expected agentsByNode map to be cleared")
	}
}

func TestNodeTypes(t *testing.T) {
	// Test that node types are recognized strings
	validNodeTypes := []string{"hub", "gateway", "spoke"}

	auth := AuthPayload{
		NodeType: "gateway",
	}

	found := false
	for _, nt := range validNodeTypes {
		if auth.NodeType == nt {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("NodeType %q not in valid node types", auth.NodeType)
	}
}

func TestMessage_EmptyPayload(t *testing.T) {
	msg := Message{
		Type:      MsgTypePing,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message with empty payload: %v", err)
	}

	var result Message
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Type != MsgTypePing {
		t.Errorf("Expected type %q, got %q", MsgTypePing, result.Type)
	}
	// Payload should be nil or empty for ping
	if len(result.Payload) > 0 && string(result.Payload) != "null" {
		t.Errorf("Expected empty or null payload for ping, got %s", result.Payload)
	}
}
