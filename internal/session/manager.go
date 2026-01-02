// Package session provides remote session management for hub/gateway/spoke nodes.
// Agents connect outbound to the control plane via WebSocket, allowing the control
// plane to dispatch commands without requiring inbound firewall rules on agents.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message types for WebSocket communication
const (
	MsgTypeAuth         = "auth"
	MsgTypeAuthResponse = "auth_response"
	MsgTypeCommand      = "command"
	MsgTypeOutput       = "output"
	MsgTypePing         = "ping"
	MsgTypePong         = "pong"
	MsgTypeAgentList    = "agent_list"
	MsgTypeConnectAgent = "connect_agent"
	MsgTypeDisconnect   = "disconnect"
	MsgTypeError        = "error"
)

// Message is the base WebSocket message structure
type Message struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// AuthPayload is sent by agents to authenticate
type AuthPayload struct {
	Token    string `json:"token"`
	NodeType string `json:"nodeType"` // hub, gateway, spoke
	NodeID   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
}

// AuthResponsePayload is sent back after authentication
type AuthResponsePayload struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	AgentID string `json:"agentId,omitempty"`
}

// CommandPayload is sent from admin to execute a command
type CommandPayload struct {
	Command string `json:"command"`
	AgentID string `json:"agentId,omitempty"` // Used by admin to specify target
}

// OutputPayload contains command output
type OutputPayload struct {
	Output   string `json:"output"`
	IsStderr bool   `json:"isStderr,omitempty"`
	ExitCode *int   `json:"exitCode,omitempty"` // nil while running, set when done
	Done     bool   `json:"done"`
}

// AgentInfo describes a connected agent
type AgentInfo struct {
	AgentID   string    `json:"agentId"`
	NodeType  string    `json:"nodeType"`
	NodeID    string    `json:"nodeId"`
	NodeName  string    `json:"nodeName"`
	Connected time.Time `json:"connected"`
}

// ConnectedAgent represents an agent connected to the session manager
type ConnectedAgent struct {
	Info     AgentInfo
	Conn     *websocket.Conn
	Send     chan []byte
	Done     chan struct{}
	LastPing time.Time
	mutex    sync.Mutex
}

// AdminSession represents an admin connected for remote sessions
type AdminSession struct {
	ID          string
	Conn        *websocket.Conn
	Send        chan []byte
	Done        chan struct{}
	ConnectedTo string // AgentID currently connected to
	UserEmail   string
	mutex       sync.Mutex
}

// Manager handles remote session connections
type Manager struct {
	agents          map[string]*ConnectedAgent    // AgentID -> agent
	admins          map[string]*AdminSession      // SessionID -> admin
	agentsByNode    map[string]string             // NodeID -> AgentID (for lookup)
	pendingCommands map[string]chan OutputPayload // MsgID -> output channel (for sync commands)
	mutex           sync.RWMutex
	logger          *zap.Logger
	upgrader        websocket.Upgrader

	// Token validation function
	ValidateAgentToken func(nodeType, nodeID, token string) bool
}

// NewManager creates a new session manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		agents:       make(map[string]*ConnectedAgent),
		admins:       make(map[string]*AdminSession),
		agentsByNode: make(map[string]string),
		logger:       logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// HandleAgentConnection handles WebSocket connections from agents
func (m *Manager) HandleAgentConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade agent connection", zap.Error(err))
		return
	}

	// Wait for auth message
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		m.logger.Warn("Failed to read auth message", zap.Error(err))
		conn.Close()
		return
	}

	var msg Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		m.logger.Warn("Invalid auth message", zap.Error(err))
		conn.Close()
		return
	}

	if msg.Type != MsgTypeAuth {
		m.sendError(conn, "Expected auth message")
		conn.Close()
		return
	}

	var auth AuthPayload
	if err := json.Unmarshal(msg.Payload, &auth); err != nil {
		m.sendError(conn, "Invalid auth payload")
		conn.Close()
		return
	}

	// Validate token
	if m.ValidateAgentToken != nil && !m.ValidateAgentToken(auth.NodeType, auth.NodeID, auth.Token) {
		m.sendAuthResponse(conn, false, "Invalid token", "")
		conn.Close()
		return
	}

	// Create agent
	agentID := uuid.New().String()
	agent := &ConnectedAgent{
		Info: AgentInfo{
			AgentID:   agentID,
			NodeType:  auth.NodeType,
			NodeID:    auth.NodeID,
			NodeName:  auth.NodeName,
			Connected: time.Now(),
		},
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Done:     make(chan struct{}),
		LastPing: time.Now(),
	}

	// Register agent
	m.mutex.Lock()
	// Remove old connection if exists
	if oldAgentID, exists := m.agentsByNode[auth.NodeID]; exists {
		if oldAgent, ok := m.agents[oldAgentID]; ok {
			close(oldAgent.Done)
			oldAgent.Conn.Close()
			delete(m.agents, oldAgentID)
		}
	}
	m.agents[agentID] = agent
	m.agentsByNode[auth.NodeID] = agentID
	m.mutex.Unlock()

	m.logger.Info("Agent connected",
		zap.String("agentId", agentID),
		zap.String("nodeType", auth.NodeType),
		zap.String("nodeName", auth.NodeName))

	// Send auth response
	m.sendAuthResponse(conn, true, "Connected", agentID)

	// Reset deadline
	_ = conn.SetReadDeadline(time.Time{})

	// Start agent handlers
	go m.agentWriter(agent)
	go m.agentReader(agent)

	// Notify admins of agent list change
	m.broadcastAgentList()
}

// HandleAdminConnection handles WebSocket connections from admin UI
func (m *Manager) HandleAdminConnection(w http.ResponseWriter, r *http.Request, userEmail string) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade admin connection", zap.Error(err))
		return
	}

	sessionID := uuid.New().String()
	admin := &AdminSession{
		ID:        sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Done:      make(chan struct{}),
		UserEmail: userEmail,
	}

	m.mutex.Lock()
	m.admins[sessionID] = admin
	m.mutex.Unlock()

	m.logger.Info("Admin session started",
		zap.String("sessionId", sessionID),
		zap.String("user", userEmail))

	// Send current agent list
	m.sendAgentList(admin)

	// Start admin handlers
	go m.adminWriter(admin)
	go m.adminReader(admin)
}

func (m *Manager) agentWriter(agent *ConnectedAgent) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-agent.Done:
			return
		case msg := <-agent.Send:
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := agent.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				m.logger.Warn("Failed to write to agent", zap.Error(err))
				m.removeAgent(agent.Info.AgentID)
				return
			}
		case <-ticker.C:
			// Send ping
			msg := Message{Type: MsgTypePing, Timestamp: time.Now()}
			data, _ := json.Marshal(msg)
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := agent.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				m.logger.Warn("Failed to ping agent", zap.Error(err))
				m.removeAgent(agent.Info.AgentID)
				return
			}
		}
	}
}

func (m *Manager) agentReader(agent *ConnectedAgent) {
	defer m.removeAgent(agent.Info.AgentID)

	agent.Conn.SetReadLimit(1024 * 1024) // 1MB max message
	agent.Conn.SetPongHandler(func(string) error {
		agent.mutex.Lock()
		agent.LastPing = time.Now()
		agent.mutex.Unlock()
		return nil
	})

	for {
		select {
		case <-agent.Done:
			return
		default:
		}

		_, msgBytes, err := agent.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.logger.Warn("Agent connection closed unexpectedly", zap.Error(err))
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			m.logger.Warn("Invalid message from agent", zap.Error(err))
			continue
		}

		switch msg.Type {
		case MsgTypePong:
			agent.mutex.Lock()
			agent.LastPing = time.Now()
			agent.mutex.Unlock()

		case MsgTypeOutput:
			// Check if this is a response to a pending sync command
			if msg.ID != "" {
				var outputPayload OutputPayload
				if err := json.Unmarshal(msg.Payload, &outputPayload); err == nil {
					if m.forwardOutputToPending(msg.ID, outputPayload) {
						// Output was consumed by a sync command, don't forward to admin
						continue
					}
				}
			}
			// Forward output to connected admin
			m.forwardOutputToAdmin(agent.Info.AgentID, msg)
		}
	}
}

func (m *Manager) adminWriter(admin *AdminSession) {
	for {
		select {
		case <-admin.Done:
			return
		case msg := <-admin.Send:
			admin.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := admin.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				m.logger.Warn("Failed to write to admin", zap.Error(err))
				m.removeAdmin(admin.ID)
				return
			}
		}
	}
}

func (m *Manager) adminReader(admin *AdminSession) {
	defer m.removeAdmin(admin.ID)

	admin.Conn.SetReadLimit(64 * 1024) // 64KB max message

	for {
		select {
		case <-admin.Done:
			return
		default:
		}

		_, msgBytes, err := admin.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.logger.Warn("Admin connection closed unexpectedly", zap.Error(err))
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			m.logger.Warn("Invalid message from admin", zap.Error(err))
			continue
		}

		switch msg.Type {
		case MsgTypeConnectAgent:
			var payload struct {
				AgentID string `json:"agentId"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				m.sendAdminError(admin, "Invalid connect payload")
				continue
			}

			// Verify agent exists
			m.mutex.RLock()
			_, agentExists := m.agents[payload.AgentID]
			m.mutex.RUnlock()

			if !agentExists {
				m.sendAdminError(admin, "Agent not found or disconnected")
				continue
			}

			admin.mutex.Lock()
			admin.ConnectedTo = payload.AgentID
			admin.mutex.Unlock()

			m.logger.Info("Admin connected to agent",
				zap.String("adminSession", admin.ID),
				zap.String("agentId", payload.AgentID))

			// Send agent_connected response
			m.sendAdminMessage(admin, Message{
				Type:      "agent_connected",
				Payload:   msg.Payload,
				Timestamp: time.Now(),
			})

		case MsgTypeCommand:
			var cmdPayload CommandPayload
			if err := json.Unmarshal(msg.Payload, &cmdPayload); err != nil {
				m.sendAdminError(admin, "Invalid command payload")
				continue
			}

			// Use connected agent if not specified
			agentID := cmdPayload.AgentID
			if agentID == "" {
				admin.mutex.Lock()
				agentID = admin.ConnectedTo
				admin.mutex.Unlock()
			}

			if agentID == "" {
				m.sendAdminError(admin, "No agent selected")
				continue
			}

			// Forward command to agent
			m.sendCommandToAgent(agentID, cmdPayload.Command, msg.ID)

		case MsgTypeDisconnect:
			admin.mutex.Lock()
			admin.ConnectedTo = ""
			admin.mutex.Unlock()

		case MsgTypeAgentList:
			m.sendAgentList(admin)
		}
	}
}

func (m *Manager) removeAgent(agentID string) {
	m.mutex.Lock()
	agent, exists := m.agents[agentID]
	if exists {
		delete(m.agents, agentID)
		delete(m.agentsByNode, agent.Info.NodeID)
		select {
		case <-agent.Done:
		default:
			close(agent.Done)
		}
		agent.Conn.Close()
	}
	m.mutex.Unlock()

	if exists {
		m.logger.Info("Agent disconnected", zap.String("agentId", agentID))
		m.broadcastAgentList()
	}
}

func (m *Manager) removeAdmin(sessionID string) {
	m.mutex.Lock()
	admin, exists := m.admins[sessionID]
	if exists {
		delete(m.admins, sessionID)
		select {
		case <-admin.Done:
		default:
			close(admin.Done)
		}
		admin.Conn.Close()
	}
	m.mutex.Unlock()

	if exists {
		m.logger.Info("Admin session ended", zap.String("sessionId", sessionID))
	}
}

func (m *Manager) sendError(conn *websocket.Conn, message string) {
	msg := Message{
		Type:      MsgTypeError,
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(map[string]string{"message": message})
	msg.Payload = payload
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (m *Manager) sendAuthResponse(conn *websocket.Conn, success bool, message, agentID string) {
	payload := AuthResponsePayload{
		Success: success,
		Message: message,
		AgentID: agentID,
	}
	payloadBytes, _ := json.Marshal(payload)
	msg := Message{
		Type:      MsgTypeAuthResponse,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (m *Manager) sendAdminError(admin *AdminSession, message string) {
	payload, _ := json.Marshal(map[string]string{"message": message})
	msg := Message{
		Type:      MsgTypeError,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case admin.Send <- data:
	default:
	}
}

func (m *Manager) sendAdminMessage(admin *AdminSession, msg Message) {
	data, _ := json.Marshal(msg)
	select {
	case admin.Send <- data:
	default:
	}
}

func (m *Manager) sendAgentList(admin *AdminSession) {
	m.mutex.RLock()
	agents := make([]AgentInfo, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent.Info)
	}
	m.mutex.RUnlock()

	payload, _ := json.Marshal(map[string]interface{}{"agents": agents})
	msg := Message{
		Type:      MsgTypeAgentList,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case admin.Send <- data:
	default:
	}
}

func (m *Manager) broadcastAgentList() {
	m.mutex.RLock()
	agents := make([]AgentInfo, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent.Info)
	}
	admins := make([]*AdminSession, 0, len(m.admins))
	for _, admin := range m.admins {
		admins = append(admins, admin)
	}
	m.mutex.RUnlock()

	payload, _ := json.Marshal(map[string]interface{}{"agents": agents})
	msg := Message{
		Type:      MsgTypeAgentList,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)

	for _, admin := range admins {
		select {
		case admin.Send <- data:
		default:
		}
	}
}

func (m *Manager) sendCommandToAgent(agentID, command, msgID string) {
	m.mutex.RLock()
	agent, exists := m.agents[agentID]
	m.mutex.RUnlock()

	if !exists {
		m.logger.Warn("Agent not found for command", zap.String("agentId", agentID))
		return
	}

	payload, _ := json.Marshal(CommandPayload{Command: command})
	msg := Message{
		Type:      MsgTypeCommand,
		ID:        msgID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)

	select {
	case agent.Send <- data:
		m.logger.Debug("Command sent to agent",
			zap.String("agentId", agentID),
			zap.String("command", command))
	default:
		m.logger.Warn("Agent send buffer full", zap.String("agentId", agentID))
	}
}

func (m *Manager) forwardOutputToAdmin(agentID string, msg Message) {
	m.mutex.RLock()
	var targetAdmins []*AdminSession
	for _, admin := range m.admins {
		admin.mutex.Lock()
		if admin.ConnectedTo == agentID {
			targetAdmins = append(targetAdmins, admin)
		}
		admin.mutex.Unlock()
	}
	m.mutex.RUnlock()

	data, _ := json.Marshal(msg)
	for _, admin := range targetAdmins {
		select {
		case admin.Send <- data:
		default:
		}
	}
}

// GetConnectedAgents returns list of connected agents
func (m *Manager) GetConnectedAgents() []AgentInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agents := make([]AgentInfo, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent.Info)
	}
	return agents
}

// GetAgentByNodeID returns agent ID for a node
func (m *Manager) GetAgentByNodeID(nodeID string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	agentID, exists := m.agentsByNode[nodeID]
	return agentID, exists
}

// IsAgentConnected checks if an agent is connected
func (m *Manager) IsAgentConnected(agentID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.agents[agentID]
	return exists
}

// ExecuteCommandSync sends a command to an agent and waits for the complete output.
// This is used by network tools to run diagnostic commands through the session WebSocket.
// Returns the combined output and any error.
func (m *Manager) ExecuteCommandSync(ctx context.Context, nodeID, command string, timeout time.Duration) (string, error) {
	// Find agent by node ID
	agentID, exists := m.GetAgentByNodeID(nodeID)
	if !exists {
		return "", fmt.Errorf("agent not connected for node %s", nodeID)
	}

	m.mutex.RLock()
	agent, agentExists := m.agents[agentID]
	m.mutex.RUnlock()

	if !agentExists {
		return "", fmt.Errorf("agent %s not found", agentID)
	}

	// Create a unique message ID to track this command
	msgID := uuid.New().String()

	// Create channels for collecting output
	outputChan := make(chan OutputPayload, 100)
	doneChan := make(chan struct{})

	// Register a temporary output handler
	m.mutex.Lock()
	if m.pendingCommands == nil {
		m.pendingCommands = make(map[string]chan OutputPayload)
	}
	m.pendingCommands[msgID] = outputChan
	m.mutex.Unlock()

	// Cleanup on exit
	defer func() {
		m.mutex.Lock()
		delete(m.pendingCommands, msgID)
		m.mutex.Unlock()
		close(outputChan)
	}()

	// Send the command
	payload, _ := json.Marshal(CommandPayload{Command: command})
	msg := Message{
		Type:      MsgTypeCommand,
		ID:        msgID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)

	select {
	case agent.Send <- data:
	default:
		return "", fmt.Errorf("agent send buffer full")
	}

	// Collect output with timeout
	var output strings.Builder
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-outputChan:
				if !ok {
					return
				}
				output.WriteString(out.Output)
				if out.Done {
					close(doneChan)
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return output.String(), ctx.Err()
	case <-timeoutTimer.C:
		return output.String(), fmt.Errorf("command timed out after %v", timeout)
	case <-doneChan:
		return output.String(), nil
	}
}

// forwardOutputToPending forwards output to pending synchronous command if applicable
func (m *Manager) forwardOutputToPending(msgID string, payload OutputPayload) bool {
	m.mutex.RLock()
	ch, exists := m.pendingCommands[msgID]
	m.mutex.RUnlock()

	if exists {
		select {
		case ch <- payload:
			return true
		default:
		}
	}
	return false
}

// Shutdown gracefully shuts down the session manager
func (m *Manager) Shutdown(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, agent := range m.agents {
		close(agent.Done)
		agent.Conn.Close()
	}
	for _, admin := range m.admins {
		close(admin.Done)
		admin.Conn.Close()
	}

	m.agents = make(map[string]*ConnectedAgent)
	m.admins = make(map[string]*AdminSession)
	m.agentsByNode = make(map[string]string)
}
