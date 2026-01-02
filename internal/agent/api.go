// Package agent provides the remote execution API for hub/gateway/spoke binaries.
// This allows the control plane to dispatch network diagnostic tools to remote nodes.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/network"
)

// Config holds the agent API configuration
type Config struct {
	ListenAddr string // e.g., ":9443"
	APIToken   string // Token to authenticate requests from control plane
	NodeType   string // "hub", "gateway", or "spoke"
	NodeName   string // Name of this node
	Logger     *zap.Logger
}

// Server is the agent API server
type Server struct {
	config *Config
	router *gin.Engine
	server *http.Server
	logger *zap.Logger
}

// ToolRequest is the request to execute a network tool
type ToolRequest struct {
	Token   string            `json:"token" binding:"required"`
	Tool    string            `json:"tool" binding:"required"`
	Target  string            `json:"target" binding:"required"`
	Port    int               `json:"port,omitempty"`
	Ports   string            `json:"ports,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// ToolResponse is the response from tool execution
type ToolResponse struct {
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
	StartedAt string `json:"startedAt"`
}

// StatusResponse is the response from the status endpoint
type StatusResponse struct {
	OK       bool   `json:"ok"`
	NodeType string `json:"nodeType"`
	NodeName string `json:"nodeName"`
	Version  string `json:"version"`
}

// NewServer creates a new agent API server
func NewServer(cfg *Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		config: cfg,
		router: router,
		logger: cfg.Logger,
	}

	// Register routes
	router.GET("/health", s.handleHealth)
	router.GET("/status", s.handleStatus)
	router.POST("/api/v1/tools/execute", s.handleExecuteTool)

	return s
}

// Start starts the agent API server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.config.ListenAddr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	s.logger.Info("Starting agent API server",
		zap.String("addr", s.config.ListenAddr),
		zap.String("nodeType", s.config.NodeType),
		zap.String("nodeName", s.config.NodeName))

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (s *Server) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, StatusResponse{
		OK:       true,
		NodeType: s.config.NodeType,
		NodeName: s.config.NodeName,
		Version:  "1.0.0",
	})
}

func (s *Server) handleExecuteTool(c *gin.Context) {
	var req ToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Verify token
	if req.Token != s.config.APIToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	s.logger.Info("Executing network tool",
		zap.String("tool", req.Tool),
		zap.String("target", req.Target))

	ctx := c.Request.Context()
	var result *network.ToolResult
	var err error

	switch req.Tool {
	case "ping":
		count := 4
		if countStr, ok := req.Options["count"]; ok {
			fmt.Sscanf(countStr, "%d", &count)
		}
		result, err = network.ExecutePing(ctx, req.Target, count)

	case "nslookup":
		result, err = network.ExecuteNslookup(ctx, req.Target)

	case "traceroute", "tracert":
		result, err = network.ExecuteTraceroute(ctx, req.Target)

	case "nc", "netcat":
		if req.Port <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Port is required for netcat"})
			return
		}
		result, err = network.ExecuteNetcat(ctx, req.Target, req.Port)

	case "nmap":
		result, err = network.ExecuteNmap(ctx, req.Target, req.Ports)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Unknown tool: " + req.Tool,
			"available_tools": network.AvailableTools(),
		})
		return
	}

	if err != nil {
		s.logger.Error("Tool execution failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tool execution failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ToolResponse{
		Tool:      result.Tool,
		Target:    result.Target,
		Status:    result.Status,
		Output:    result.Output,
		Error:     result.Error,
		Duration:  result.Duration.String(),
		StartedAt: result.StartedAt.Format(time.RFC3339),
	})
}

// Client is used by the control plane to call remote agents
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new agent client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExecuteTool executes a tool on a remote agent
func (c *Client) ExecuteTool(agentURL, token string, req ToolRequest) (*ToolResponse, error) {
	req.Token = token

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(agentURL+"/api/v1/tools/execute", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var result ToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetStatus gets the status of a remote agent
func (c *Client) GetStatus(agentURL string) (*StatusResponse, error) {
	resp, err := c.httpClient.Get(agentURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var result StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
