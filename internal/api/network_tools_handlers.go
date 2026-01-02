package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/network"
)

// NetworkToolRequest represents a request to execute a network tool
type NetworkToolRequest struct {
	Tool     string            `json:"tool" binding:"required"`
	Target   string            `json:"target" binding:"required"`
	Port     int               `json:"port,omitempty"`
	Ports    string            `json:"ports,omitempty"`
	Location string            `json:"location,omitempty"` // control-plane, gateway:<id>, hub:<id>, spoke:<id>
	Options  map[string]string `json:"options,omitempty"`
}

// NetworkToolResponse represents the response from a network tool execution
type NetworkToolResponse struct {
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
	Location  string `json:"location"`
	StartedAt string `json:"startedAt"`
}

// handleExecuteNetworkTool executes a network diagnostic tool
func (s *Server) handleExecuteNetworkTool(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth check
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if !admin.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	var req NetworkToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Default location to control-plane
	location := req.Location
	if location == "" {
		location = "control-plane"
	}

	s.logger.Info("Executing network tool",
		zap.String("tool", req.Tool),
		zap.String("target", req.Target),
		zap.String("user", admin.Email),
		zap.String("location", location))

	// Handle remote execution for hub/gateway/spoke
	if location != "control-plane" {
		s.handleRemoteToolExecution(c, req, location, admin.Email)
		return
	}

	var result *network.ToolResult

	switch req.Tool {
	case "ping":
		count := 4
		if countStr, ok := req.Options["count"]; ok {
			if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
				count = n
			}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tool execution failed"})
		return
	}

	c.JSON(http.StatusOK, NetworkToolResponse{
		Tool:      result.Tool,
		Target:    result.Target,
		Status:    result.Status,
		Output:    result.Output,
		Error:     result.Error,
		Duration:  result.Duration.String(),
		Location:  location,
		StartedAt: result.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleRemoteToolExecution dispatches tool execution to a remote agent via session WebSocket
func (s *Server) handleRemoteToolExecution(c *gin.Context, req NetworkToolRequest, location, userEmail string) {
	ctx := c.Request.Context()

	// Parse location to get type and ID (e.g., "hub:uuid" or "gateway:uuid" or "spoke:uuid")
	parts := strings.SplitN(location, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid location format",
			"details": "Location should be 'hub:<id>', 'gateway:<id>', or 'spoke:<id>'",
		})
		return
	}

	locationType := parts[0]
	locationID := parts[1]
	var nodeName string

	// Validate the node exists and is online
	switch locationType {
	case "hub":
		hub, err := s.meshStore.GetHub(ctx, locationID)
		if err != nil {
			s.logger.Error("Failed to get hub", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Hub not found",
				"details": fmt.Sprintf("Could not find hub with ID %s", locationID),
			})
			return
		}
		if hub.Status != "online" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Hub is offline",
				"details": fmt.Sprintf("Hub '%s' is currently %s. Please try again when the hub is online.", hub.Name, hub.Status),
			})
			return
		}
		nodeName = hub.Name

	case "gateway":
		gw, err := s.gatewayStore.GetGateway(ctx, locationID)
		if err != nil {
			s.logger.Error("Failed to get gateway", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Gateway not found",
				"details": fmt.Sprintf("Could not find gateway with ID %s", locationID),
			})
			return
		}
		if !gw.IsActive {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Gateway is offline",
				"details": fmt.Sprintf("Gateway '%s' is currently inactive. Please try again when the gateway is online.", gw.Name),
			})
			return
		}
		nodeName = gw.Name

	case "spoke":
		spoke, err := s.meshStore.GetMeshSpoke(ctx, locationID)
		if err != nil {
			s.logger.Error("Failed to get spoke", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Spoke not found",
				"details": fmt.Sprintf("Could not find spoke with ID %s", locationID),
			})
			return
		}
		if spoke.Status != "connected" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Spoke is disconnected",
				"details": fmt.Sprintf("Spoke '%s' is currently %s. Please try again when the spoke is connected.", spoke.Name, spoke.Status),
			})
			return
		}
		nodeName = spoke.Name

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unknown location type",
			"details": fmt.Sprintf("Location type '%s' is not supported. Use 'hub', 'gateway', or 'spoke'.", locationType),
		})
		return
	}

	// Build the command string for the tool
	command := buildToolCommand(req)
	if command == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid tool",
			"details": fmt.Sprintf("Unknown tool '%s'", req.Tool),
		})
		return
	}

	s.logger.Info("Dispatching tool via session WebSocket",
		zap.String("tool", req.Tool),
		zap.String("target", req.Target),
		zap.String("location", location),
		zap.String("nodeName", nodeName),
		zap.String("command", command),
		zap.String("user", userEmail))

	startTime := time.Now()

	// Execute command through the session manager WebSocket
	// Note: Agents register with their name as NodeID, not their UUID
	output, err := s.sessionMgr.ExecuteCommandSync(ctx, nodeName, command, 60*time.Second)
	if err != nil {
		s.logger.Error("Remote tool execution failed",
			zap.String("location", location),
			zap.Error(err))

		errMsg := err.Error()
		var userFriendlyError, details string

		if strings.Contains(errMsg, "agent not connected") {
			userFriendlyError = "Agent not connected"
			details = fmt.Sprintf("The remote session agent on '%s' is not connected. Ensure the agent is running and has session_enabled=true.", nodeName)
		} else if strings.Contains(errMsg, "timed out") {
			userFriendlyError = "Command timeout"
			details = fmt.Sprintf("The command on '%s' timed out after 60 seconds.", nodeName)
		} else if strings.Contains(errMsg, "send buffer full") {
			userFriendlyError = "Agent busy"
			details = fmt.Sprintf("The agent on '%s' is busy processing other commands. Please try again.", nodeName)
		} else {
			userFriendlyError = "Remote execution failed"
			details = errMsg
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   userFriendlyError,
			"details": details,
		})
		return
	}

	duration := time.Since(startTime)

	c.JSON(http.StatusOK, NetworkToolResponse{
		Tool:      req.Tool,
		Target:    req.Target,
		Status:    "completed",
		Output:    output,
		Duration:  duration.String(),
		Location:  location,
		StartedAt: startTime.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// buildToolCommand constructs a shell command for the given network tool request
func buildToolCommand(req NetworkToolRequest) string {
	switch req.Tool {
	case "ping":
		count := "4"
		if countStr, ok := req.Options["count"]; ok {
			if n, err := strconv.Atoi(countStr); err == nil && n > 0 && n <= 20 {
				count = countStr
			}
		}
		return fmt.Sprintf("ping -c %s %s", count, req.Target)

	case "nslookup":
		return fmt.Sprintf("nslookup %s", req.Target)

	case "traceroute", "tracert":
		return fmt.Sprintf("traceroute -m 20 %s", req.Target)

	case "nc", "netcat":
		if req.Port <= 0 {
			return ""
		}
		return fmt.Sprintf("nc -zv -w 5 %s %d", req.Target, req.Port)

	case "nmap":
		ports := req.Ports
		if ports == "" {
			ports = "22,80,443"
		}
		return fmt.Sprintf("nmap -sT -p %s %s", ports, req.Target)

	default:
		return ""
	}
}

// handleListNetworkTools returns available network tools
func (s *Server) handleListNetworkTools(c *gin.Context) {
	// Auth check
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if !admin.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Get list of execution locations
	ctx := c.Request.Context()
	locations := []map[string]string{
		{"id": "control-plane", "name": "Control Plane", "type": "control-plane"},
	}

	// Add gateways
	gateways, err := s.gatewayStore.ListGateways(ctx)
	if err == nil {
		for _, gw := range gateways {
			if gw.IsActive {
				locations = append(locations, map[string]string{
					"id":   "gateway:" + gw.ID,
					"name": "Gateway: " + gw.Name,
					"type": "gateway",
				})
			}
		}
	}

	// Add mesh hubs
	hubs, err := s.meshStore.ListHubs(ctx)
	if err == nil {
		for _, hub := range hubs {
			if hub.Status == "online" {
				locations = append(locations, map[string]string{
					"id":   "hub:" + hub.ID,
					"name": "Hub: " + hub.Name,
					"type": "hub",
				})
			}
		}
	}

	// Add mesh spokes
	for _, hub := range hubs {
		spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
		if err == nil {
			for _, spoke := range spokes {
				if spoke.Status == "connected" {
					locations = append(locations, map[string]string{
						"id":   "spoke:" + spoke.ID,
						"name": "Spoke: " + spoke.Name,
						"type": "spoke",
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": []map[string]interface{}{
			{
				"name":        "ping",
				"description": "Test ICMP connectivity to a host",
				"options":     []string{"count"},
			},
			{
				"name":        "nslookup",
				"description": "Perform DNS lookup for a hostname",
				"options":     []string{},
			},
			{
				"name":        "traceroute",
				"description": "Trace the route to a host",
				"options":     []string{},
			},
			{
				"name":        "nc",
				"description": "Test TCP connectivity to a port",
				"options":     []string{"port"},
				"required":    []string{"port"},
			},
			{
				"name":        "nmap",
				"description": "Scan ports on a host",
				"options":     []string{"ports"},
			},
		},
		"locations": locations,
	})
}
