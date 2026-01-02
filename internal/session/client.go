package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// AgentClient connects to the control plane for remote session support
type AgentClient struct {
	controlPlaneURL string
	token           string
	nodeType        string
	nodeID          string
	nodeName        string
	logger          *zap.Logger

	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	connected bool
	agentID   string
	mutex     sync.RWMutex

	// Current running command
	currentCmd    *exec.Cmd
	cmdMutex      sync.Mutex
	cmdCancelFunc context.CancelFunc
}

// AgentClientConfig holds configuration for the agent client
type AgentClientConfig struct {
	ControlPlaneURL string
	Token           string
	NodeType        string // hub, gateway, spoke
	NodeID          string
	NodeName        string
	Logger          *zap.Logger
}

// NewAgentClient creates a new agent client for remote sessions
func NewAgentClient(cfg *AgentClientConfig) *AgentClient {
	return &AgentClient{
		controlPlaneURL: cfg.ControlPlaneURL,
		token:           cfg.Token,
		nodeType:        cfg.NodeType,
		nodeID:          cfg.NodeID,
		nodeName:        cfg.NodeName,
		logger:          cfg.Logger,
		send:            make(chan []byte, 256),
		done:            make(chan struct{}),
	}
}

// Start connects to the control plane and maintains the connection
func (c *AgentClient) Start(ctx context.Context) {
	go c.connectionLoop(ctx)
}

// Stop gracefully stops the agent client
func (c *AgentClient) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case <-c.done:
		return
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
	}

	// Cancel any running command
	c.cmdMutex.Lock()
	if c.cmdCancelFunc != nil {
		c.cmdCancelFunc()
	}
	c.cmdMutex.Unlock()
}

func (c *AgentClient) connectionLoop(ctx context.Context) {
	reconnectDelay := 5 * time.Second
	maxDelay := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		err := c.connect(ctx)
		if err != nil {
			c.logger.Warn("Failed to connect to control plane",
				zap.Error(err),
				zap.Duration("retryIn", reconnectDelay))

			select {
			case <-ctx.Done():
				return
			case <-c.done:
				return
			case <-time.After(reconnectDelay):
			}

			// Exponential backoff
			reconnectDelay = reconnectDelay * 2
			if reconnectDelay > maxDelay {
				reconnectDelay = maxDelay
			}
			continue
		}

		// Reset delay on successful connection
		reconnectDelay = 5 * time.Second

		// Run message handlers
		c.runHandlers(ctx)

		c.logger.Info("Disconnected from control plane, reconnecting...")
	}
}

func (c *AgentClient) connect(ctx context.Context) error {
	// Build WebSocket URL
	u, err := url.Parse(c.controlPlaneURL)
	if err != nil {
		return fmt.Errorf("invalid control plane URL: %w", err)
	}

	// Convert http(s) to ws(s)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = "/ws/agent"

	c.logger.Info("Connecting to control plane",
		zap.String("url", u.String()))

	// Connect with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, httpResp, err := dialer.DialContext(ctx, u.String(), nil)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mutex.Lock()
	c.conn = conn
	c.mutex.Unlock()

	// Send auth message
	auth := AuthPayload{
		Token:    c.token,
		NodeType: c.nodeType,
		NodeID:   c.nodeID,
		NodeName: c.nodeName,
	}
	authBytes, _ := json.Marshal(auth)
	msg := Message{
		Type:      MsgTypeAuth,
		Payload:   authBytes,
		Timestamp: time.Now(),
	}
	msgBytes, _ := json.Marshal(msg)

	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Wait for auth response
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, respBytes, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	var resp Message
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		conn.Close()
		return fmt.Errorf("invalid auth response: %w", err)
	}

	if resp.Type != MsgTypeAuthResponse {
		conn.Close()
		return fmt.Errorf("unexpected response type: %s", resp.Type)
	}

	var authResp AuthResponsePayload
	if err := json.Unmarshal(resp.Payload, &authResp); err != nil {
		conn.Close()
		return fmt.Errorf("invalid auth response payload: %w", err)
	}

	if !authResp.Success {
		conn.Close()
		return fmt.Errorf("auth failed: %s", authResp.Message)
	}

	_ = conn.SetReadDeadline(time.Time{})

	c.mutex.Lock()
	c.connected = true
	c.agentID = authResp.AgentID
	c.mutex.Unlock()

	c.logger.Info("Connected to control plane",
		zap.String("agentId", authResp.AgentID))

	return nil
}

func (c *AgentClient) runHandlers(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)

	localDone := make(chan struct{})

	// Writer
	go func() {
		defer wg.Done()
		c.writer(localDone)
	}()

	// Reader
	go func() {
		defer wg.Done()
		c.reader(ctx, localDone)
	}()

	wg.Wait()

	c.mutex.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mutex.Unlock()
}

func (c *AgentClient) writer(done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-c.done:
			return
		case msg := <-c.send:
			c.mutex.RLock()
			conn := c.conn
			c.mutex.RUnlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.logger.Warn("Failed to write message", zap.Error(err))
				return
			}
		case <-ticker.C:
			// Send pong in response to server pings
			c.mutex.RLock()
			conn := c.conn
			c.mutex.RUnlock()

			if conn == nil {
				return
			}

			msg := Message{Type: MsgTypePong, Timestamp: time.Now()}
			data, _ := json.Marshal(msg)
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

func (c *AgentClient) reader(ctx context.Context, done chan struct{}) {
	defer close(done)

	c.mutex.RLock()
	conn := c.conn
	c.mutex.RUnlock()

	if conn == nil {
		return
	}

	conn.SetReadLimit(64 * 1024) // 64KB max

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("Connection closed unexpectedly", zap.Error(err))
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			c.logger.Warn("Invalid message", zap.Error(err))
			continue
		}

		switch msg.Type {
		case MsgTypePing:
			// Respond with pong
			pong := Message{Type: MsgTypePong, Timestamp: time.Now()}
			data, _ := json.Marshal(pong)
			select {
			case c.send <- data:
			default:
			}

		case MsgTypeCommand:
			var cmdPayload CommandPayload
			if err := json.Unmarshal(msg.Payload, &cmdPayload); err != nil {
				c.logger.Warn("Invalid command payload", zap.Error(err))
				continue
			}

			c.logger.Info("Received command",
				zap.String("command", cmdPayload.Command))

			// Execute command in background
			go c.executeCommand(ctx, cmdPayload.Command, msg.ID)
		}
	}
}

func (c *AgentClient) executeCommand(ctx context.Context, command, msgID string) {
	// Create command context with cancellation
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

	c.cmdMutex.Lock()
	// Cancel previous command if running
	if c.cmdCancelFunc != nil {
		c.cmdCancelFunc()
	}
	c.cmdCancelFunc = cancel
	c.cmdMutex.Unlock()

	defer func() {
		c.cmdMutex.Lock()
		c.cmdCancelFunc = nil
		c.cmdMutex.Unlock()
		cancel()
	}()

	// Execute via shell
	cmd := exec.CommandContext(cmdCtx, "/bin/sh", "-c", command)

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.sendOutput(msgID, fmt.Sprintf("Error: %v\n", err), false, intPtr(1), true)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.sendOutput(msgID, fmt.Sprintf("Error: %v\n", err), false, intPtr(1), true)
		return
	}

	c.cmdMutex.Lock()
	c.currentCmd = cmd
	c.cmdMutex.Unlock()

	if err := cmd.Start(); err != nil {
		c.sendOutput(msgID, fmt.Sprintf("Error starting command: %v\n", err), false, intPtr(1), true)
		return
	}

	// Stream output
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.streamOutput(msgID, stdout, false)
	}()

	go func() {
		defer wg.Done()
		c.streamOutput(msgID, stderr, true)
	}()

	wg.Wait()

	// Wait for command to complete
	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	c.cmdMutex.Lock()
	c.currentCmd = nil
	c.cmdMutex.Unlock()

	// Send completion
	c.sendOutput(msgID, "", false, &exitCode, true)
}

func (c *AgentClient) streamOutput(msgID string, r io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for long lines
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		c.sendOutput(msgID, line, isStderr, nil, false)
	}
}

func (c *AgentClient) sendOutput(msgID, output string, isStderr bool, exitCode *int, done bool) {
	payload := OutputPayload{
		Output:   output,
		IsStderr: isStderr,
		ExitCode: exitCode,
		Done:     done,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := Message{
		Type:      MsgTypeOutput,
		ID:        msgID,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}
	msgBytes, _ := json.Marshal(msg)

	select {
	case c.send <- msgBytes:
	default:
		c.logger.Warn("Send buffer full, dropping output")
	}
}

func intPtr(i int) *int {
	return &i
}

// IsConnected returns whether the client is connected
func (c *AgentClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetAgentID returns the assigned agent ID
func (c *AgentClient) GetAgentID() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.agentID
}
