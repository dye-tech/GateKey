package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/session"
)

// handleAgentWebSocket handles WebSocket connections from agents (hub/gateway/spoke)
func (s *Server) handleAgentWebSocket(c *gin.Context) {
	if s.sessionMgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session manager not initialized"})
		return
	}

	s.sessionMgr.HandleAgentConnection(c.Writer, c.Request)
}

// handleAdminSessionWebSocket handles WebSocket connections from admin UI for remote sessions
func (s *Server) handleAdminSessionWebSocket(c *gin.Context) {
	if s.sessionMgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session manager not initialized"})
		return
	}

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

	s.logger.Info("Admin starting remote session",
		zap.String("user", admin.Email))

	s.sessionMgr.HandleAdminConnection(c.Writer, c.Request, admin.Email)
}

// handleGetConnectedAgents returns list of agents connected for remote sessions
func (s *Server) handleGetConnectedAgents(c *gin.Context) {
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

	if s.sessionMgr == nil {
		c.JSON(http.StatusOK, gin.H{"agents": []session.AgentInfo{}})
		return
	}

	agents := s.sessionMgr.GetConnectedAgents()
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// validateAgentToken validates an agent's token for session authentication
func (s *Server) validateAgentToken(nodeType, nodeID, token string) bool {
	ctx := context.Background()

	switch nodeType {
	case "hub":
		// Validate hub token
		hub, err := s.meshStore.GetHubByName(nodeID)
		if err != nil {
			s.logger.Warn("Failed to get hub for session auth",
				zap.String("nodeId", nodeID),
				zap.Error(err))
			return false
		}
		return hub != nil && hub.APIToken == token

	case "gateway":
		// Validate gateway token
		gw, err := s.gatewayStore.GetGatewayByName(ctx, nodeID)
		if err != nil {
			s.logger.Warn("Failed to get gateway for session auth",
				zap.String("nodeId", nodeID),
				zap.Error(err))
			return false
		}
		return gw != nil && gw.Token == token

	case "spoke":
		// Validate spoke token
		spoke, err := s.meshStore.GetMeshSpokeByName(nodeID)
		if err != nil {
			s.logger.Warn("Failed to get spoke for session auth",
				zap.String("nodeId", nodeID),
				zap.Error(err))
			return false
		}
		return spoke != nil && spoke.Token == token
	}

	return false
}
