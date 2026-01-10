package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TopologyResponse represents the full network topology
type TopologyResponse struct {
	Gateways    []TopologyGateway    `json:"gateways"`
	MeshHubs    []TopologyMeshHub    `json:"meshHubs"`
	MeshSpokes  []TopologyMeshSpoke  `json:"meshSpokes"`
	Connections []TopologyConnection `json:"connections"`
}

// TopologyGateway represents a gateway in the topology
type TopologyGateway struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Hostname      string     `json:"hostname"`
	PublicIP      string     `json:"publicIp"`
	VPNPort       int        `json:"vpnPort"`
	VPNProtocol   string     `json:"vpnProtocol"`
	IsActive      bool       `json:"isActive"`
	LastHeartbeat *time.Time `json:"lastHeartbeat"`
	ClientCount   int        `json:"clientCount"`
}

// TopologyMeshHub represents a mesh hub in the topology
type TopologyMeshHub struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	PublicEndpoint  string     `json:"publicEndpoint"`
	PublicIP        string     `json:"publicIp"`
	VPNPort         int        `json:"vpnPort"`
	VPNSubnet       string     `json:"vpnSubnet"`
	ServerTunnelIP  string     `json:"serverTunnelIp"` // Hub's VPN server IP (e.g., 172.30.0.1)
	LocalNetworks   []string   `json:"localNetworks"`
	Status          string     `json:"status"`
	LastHeartbeat   *time.Time `json:"lastHeartbeat"`
	ConnectedSpokes int        `json:"connectedSpokes"`
	ConnectedUsers  int        `json:"connectedUsers"`
	GatewayType     string     `json:"gatewayType"` // openvpn or wireguard
}

// TopologyMeshSpoke represents a mesh spoke in the topology
type TopologyMeshSpoke struct {
	ID            string     `json:"id"`
	HubID         string     `json:"hubId"`
	Name          string     `json:"name"`
	LocalNetworks []string   `json:"localNetworks"`
	TunnelIP      string     `json:"tunnelIp"`
	Status        string     `json:"status"`
	LastSeen      *time.Time `json:"lastSeen"`
	RemoteIP      string     `json:"remoteIp"`
	GatewayType   string     `json:"gatewayType"` // openvpn or wireguard
}

// TopologyConnection represents a connection between nodes
type TopologyConnection struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // hub-spoke, gateway-client
	Status string `json:"status"`
}

// handleGetTopology returns the complete network topology
func (s *Server) handleGetTopology(c *gin.Context) {
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

	// Get gateways
	gateways, err := s.gatewayStore.ListGateways(ctx)
	if err != nil {
		s.logger.Error("Failed to list gateways")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load gateways"})
		return
	}

	// Get mesh hubs
	hubs, err := s.meshStore.ListHubs(ctx)
	if err != nil {
		s.logger.Error("Failed to list mesh hubs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load mesh hubs"})
		return
	}

	// Build response
	response := TopologyResponse{
		Gateways:    make([]TopologyGateway, 0, len(gateways)),
		MeshHubs:    make([]TopologyMeshHub, 0, len(hubs)),
		MeshSpokes:  make([]TopologyMeshSpoke, 0),
		Connections: make([]TopologyConnection, 0),
	}

	// Add gateways
	for _, gw := range gateways {
		response.Gateways = append(response.Gateways, TopologyGateway{
			ID:            gw.ID,
			Name:          gw.Name,
			Hostname:      gw.Hostname,
			PublicIP:      gw.PublicIP,
			VPNPort:       gw.VPNPort,
			VPNProtocol:   gw.VPNProtocol,
			IsActive:      gw.IsActive,
			LastHeartbeat: gw.LastHeartbeat,
			ClientCount:   0, // Populated by frontend from active sessions
		})
	}

	// Add mesh hubs and their spokes
	for _, hub := range hubs {
		// Extract public IP from endpoint (hostname:port format)
		publicIP := hub.PublicEndpoint
		if idx := strings.LastIndex(hub.PublicEndpoint, ":"); idx > 0 {
			publicIP = hub.PublicEndpoint[:idx]
		}

		// Calculate server tunnel IP (first usable IP in subnet, e.g., 172.30.0.1)
		serverTunnelIP := calculateServerTunnelIP(hub.VPNSubnet)

		// Get hub's local networks
		hubDetails, _ := s.meshStore.GetHub(ctx, hub.ID)
		var localNetworks []string
		if hubDetails != nil {
			localNetworks = hubDetails.LocalNetworks
		}

		response.MeshHubs = append(response.MeshHubs, TopologyMeshHub{
			ID:              hub.ID,
			Name:            hub.Name,
			PublicEndpoint:  hub.PublicEndpoint,
			PublicIP:        publicIP,
			VPNPort:         hub.VPNPort,
			VPNSubnet:       hub.VPNSubnet,
			ServerTunnelIP:  serverTunnelIP,
			LocalNetworks:   localNetworks,
			Status:          hub.Status,
			LastHeartbeat:   hub.LastHeartbeat,
			ConnectedSpokes: hub.ConnectedSpokes,
			ConnectedUsers:  hub.ConnectedClients,
			GatewayType:     hub.GatewayType,
		})

		// Get spokes for this hub
		spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
		if err != nil {
			s.logger.Error("Failed to list mesh spokes for hub")
			continue
		}

		for _, spoke := range spokes {
			response.MeshSpokes = append(response.MeshSpokes, TopologyMeshSpoke{
				ID:            spoke.ID,
				HubID:         spoke.HubID,
				Name:          spoke.Name,
				LocalNetworks: spoke.LocalNetworks,
				TunnelIP:      spoke.TunnelIP,
				Status:        spoke.Status,
				LastSeen:      spoke.LastSeen,
				RemoteIP:      spoke.RemoteIP,
				GatewayType:   spoke.GatewayType,
			})

			// Add connection from hub to spoke
			connStatus := "disconnected"
			if spoke.Status == "connected" {
				connStatus = "connected"
			}
			response.Connections = append(response.Connections, TopologyConnection{
				ID:     hub.ID + "-" + spoke.ID,
				Source: "hub-" + hub.ID,
				Target: "spoke-" + spoke.ID,
				Type:   "hub-spoke",
				Status: connStatus,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}

// ActiveSession represents an active VPN session
type ActiveSession struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	UserEmail   string     `json:"userEmail"`
	UserName    string     `json:"userName"`
	GatewayID   string     `json:"gatewayId"`
	GatewayName string     `json:"gatewayName"`
	NodeType    string     `json:"nodeType"` // gateway, hub
	ClientIP    string     `json:"clientIp"`
	VPNAddress  string     `json:"vpnAddress"`
	ConnectedAt time.Time  `json:"connectedAt"`
	BytesSent   int64      `json:"bytesSent"`
	BytesRecv   int64      `json:"bytesRecv"`
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
}

// ActiveSessionsResponse contains all active sessions
type ActiveSessionsResponse struct {
	Sessions []ActiveSession `json:"sessions"`
	Total    int             `json:"total"`
}

// handleGetActiveSessions returns all active VPN sessions
// Sessions are tracked from mesh_connections table
func (s *Server) handleGetActiveSessions(c *gin.Context) {
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

	// Query active mesh connections from database
	sessions, err := s.getActiveMeshConnections(ctx)
	if err != nil {
		s.logger.Error("Failed to get active sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sessions"})
		return
	}

	c.JSON(http.StatusOK, ActiveSessionsResponse{
		Sessions: sessions,
		Total:    len(sessions),
	})
}

// getActiveMeshConnections queries mesh_connections and gateway_connections for active sessions
func (s *Server) getActiveMeshConnections(ctx context.Context) ([]ActiveSession, error) {
	var sessions []ActiveSession

	// First, mark any very stale connections as disconnected (safety net if stats sync isn't working)
	// Connections not seen for 2+ minutes are considered stale
	_, _ = s.db.Pool.Exec(ctx, `
		UPDATE mesh_connections
		SET disconnected_at = NOW(), disconnect_reason = 'stale'
		WHERE disconnected_at IS NULL
		  AND last_seen_at IS NOT NULL
		  AND last_seen_at < NOW() - INTERVAL '2 minutes'
	`)
	_, _ = s.db.Pool.Exec(ctx, `
		UPDATE gateway_connections
		SET disconnected_at = NOW(), disconnect_reason = 'stale'
		WHERE disconnected_at IS NULL
		  AND last_seen_at IS NOT NULL
		  AND last_seen_at < NOW() - INTERVAL '2 minutes'
	`)

	// Query mesh hub connections (supports both SSO and local users)
	meshRows, err := s.db.Pool.Query(ctx, `
		SELECT
			mc.id::text, mc.hub_id::text, mc.user_id,
			COALESCE(u.email, lu.email, ''),
			COALESCE(u.name, lu.username, ''),
			h.name, host(mc.client_ip), host(mc.tunnel_ip),
			mc.bytes_sent, mc.bytes_received, mc.connected_at,
			COALESCE(mc.user_type, 'sso'),
			mc.last_seen_at
		FROM mesh_connections mc
		JOIN mesh_hubs h ON mc.hub_id = h.id
		LEFT JOIN users u ON COALESCE(mc.user_type, 'sso') = 'sso' AND mc.user_id = u.id::text
		LEFT JOIN local_users lu ON mc.user_type = 'local' AND mc.user_id = lu.id::text
		WHERE mc.disconnected_at IS NULL
		ORDER BY mc.connected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer meshRows.Close()

	for meshRows.Next() {
		var sess ActiveSession
		var clientIP, tunnelIP *string
		var userType string
		if err := meshRows.Scan(
			&sess.ID, &sess.GatewayID, &sess.UserID, &sess.UserEmail, &sess.UserName,
			&sess.GatewayName, &clientIP, &tunnelIP,
			&sess.BytesSent, &sess.BytesRecv, &sess.ConnectedAt, &userType,
			&sess.LastSeenAt,
		); err != nil {
			return nil, err
		}
		sess.NodeType = "hub"
		if clientIP != nil {
			sess.ClientIP = *clientIP
		}
		if tunnelIP != nil {
			sess.VPNAddress = *tunnelIP
		}
		// Add (Local) suffix for local users in the name
		if userType == "local" && sess.UserName != "" && !strings.HasSuffix(sess.UserName, "(Local)") {
			sess.UserName = sess.UserName + " (Local)"
		}
		sessions = append(sessions, sess)
	}
	if err := meshRows.Err(); err != nil {
		return nil, err
	}

	// Query gateway connections (supports both SSO and local users)
	gatewayRows, err := s.db.Pool.Query(ctx, `
		SELECT
			gc.id::text, gc.gateway_id::text, gc.user_id,
			COALESCE(u.email, lu.email, ''),
			COALESCE(u.name, lu.username, ''),
			g.name, host(gc.client_ip), host(gc.tunnel_ip),
			gc.bytes_sent, gc.bytes_received, gc.connected_at,
			gc.user_type,
			gc.last_seen_at
		FROM gateway_connections gc
		JOIN gateways g ON gc.gateway_id = g.id
		LEFT JOIN users u ON gc.user_type = 'sso' AND gc.user_id = u.id::text
		LEFT JOIN local_users lu ON gc.user_type = 'local' AND gc.user_id = lu.id::text
		WHERE gc.disconnected_at IS NULL
		ORDER BY gc.connected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer gatewayRows.Close()

	for gatewayRows.Next() {
		var sess ActiveSession
		var clientIP, tunnelIP *string
		var userType string
		if err := gatewayRows.Scan(
			&sess.ID, &sess.GatewayID, &sess.UserID, &sess.UserEmail, &sess.UserName,
			&sess.GatewayName, &clientIP, &tunnelIP,
			&sess.BytesSent, &sess.BytesRecv, &sess.ConnectedAt, &userType,
			&sess.LastSeenAt,
		); err != nil {
			return nil, err
		}
		sess.NodeType = "gateway"
		if clientIP != nil {
			sess.ClientIP = *clientIP
		}
		if tunnelIP != nil {
			sess.VPNAddress = *tunnelIP
		}
		// Add (Local) suffix for local users in the name
		if userType == "local" && sess.UserName != "" && !strings.HasSuffix(sess.UserName, "(Local)") {
			sess.UserName = sess.UserName + " (Local)"
		}
		sessions = append(sessions, sess)
	}
	return sessions, gatewayRows.Err()
}

// DisconnectSessionRequest represents a request to disconnect a specific session
type DisconnectSessionRequest struct {
	Reason string `json:"reason"`
}

// DisconnectUserRequest represents a request to disconnect all sessions for a user
type DisconnectUserRequest struct {
	Reason string `json:"reason"`
}

// handleDisconnectSession disconnects a specific VPN session immediately.
// This targets one individual session, not groups or all users.
// POST /api/v1/admin/sessions/:sessionId/disconnect
func (s *Server) handleDisconnectSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("sessionId")

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

	var req DisconnectSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Disconnected by administrator"
	}
	if req.Reason == "" {
		req.Reason = "Disconnected by administrator"
	}

	// Get admin email for audit
	adminEmail := admin.Email
	if adminEmail == "" {
		adminEmail = "admin"
	}

	// Try to find and disconnect the session in gateway_connections
	var userID, userEmail, gatewayID, vpnAddress string
	var nodeType string
	err = s.db.Pool.QueryRow(ctx, `
		SELECT gc.user_id, COALESCE(u.email, lu.email, ''), gc.gateway_id::text, host(gc.tunnel_ip)
		FROM gateway_connections gc
		LEFT JOIN users u ON gc.user_type = 'sso' AND gc.user_id = u.id::text
		LEFT JOIN local_users lu ON gc.user_type = 'local' AND gc.user_id = lu.id::text
		WHERE gc.id = $1 AND gc.disconnected_at IS NULL
	`, sessionID).Scan(&userID, &userEmail, &gatewayID, &vpnAddress)

	if err == nil {
		nodeType = "gateway"
	} else {
		// Try mesh_connections
		err = s.db.Pool.QueryRow(ctx, `
			SELECT mc.user_id, COALESCE(u.email, lu.email, ''), mc.hub_id::text, host(mc.tunnel_ip)
			FROM mesh_connections mc
			LEFT JOIN users u ON COALESCE(mc.user_type, 'sso') = 'sso' AND mc.user_id = u.id::text
			LEFT JOIN local_users lu ON mc.user_type = 'local' AND mc.user_id = lu.id::text
			WHERE mc.id = $1 AND mc.disconnected_at IS NULL
		`, sessionID).Scan(&userID, &userEmail, &gatewayID, &vpnAddress)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or already disconnected"})
			return
		}
		nodeType = "hub"
	}

	// Create pending disconnect for gateway agent to execute
	store := db.NewPendingDisconnectStore(s.db)
	_, err = store.CreateDisconnectRequest(ctx, &db.DisconnectRequest{
		UserID:       userID,
		UserEmail:    userEmail,
		GatewayID:    &gatewayID,
		NodeType:     nodeType,
		ConnectionID: &sessionID,
		VPNAddress:   &vpnAddress,
		Reason:       req.Reason,
		RequestedBy:  adminEmail,
	})
	if err != nil {
		s.logger.Error("Failed to create disconnect request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create disconnect request"})
		return
	}

	// Also mark the connection as disconnect requested
	isGateway := nodeType == "gateway"
	_ = store.MarkConnectionDisconnectRequested(ctx, sessionID, adminEmail, isGateway)

	s.logger.Info("Session disconnect requested",
		zap.String("sessionId", sessionID),
		zap.String("userEmail", userEmail),
		zap.String("gatewayId", gatewayID),
		zap.String("requestedBy", adminEmail),
		zap.String("reason", req.Reason))

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Disconnect request created",
		"sessionId":  sessionID,
		"userEmail":  userEmail,
		"gatewayId":  gatewayID,
		"nodeType":   nodeType,
		"vpnAddress": vpnAddress,
	})
}

// handleDisconnectUser disconnects ALL active sessions for a specific user.
// This targets one individual user, NOT groups or other users.
// POST /api/v1/admin/users/:id/disconnect
func (s *Server) handleDisconnectUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")

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

	var req DisconnectUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Disconnected by administrator"
	}
	if req.Reason == "" {
		req.Reason = "Disconnected by administrator"
	}

	adminEmail := admin.Email
	if adminEmail == "" {
		adminEmail = "admin"
	}

	// Get user email for display
	var userEmail string
	err = s.db.Pool.QueryRow(ctx, `
		SELECT email FROM users WHERE id = $1
		UNION ALL
		SELECT email FROM local_users WHERE id = $1
		LIMIT 1
	`, userID).Scan(&userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	store := db.NewPendingDisconnectStore(s.db)

	// Get all active connections for this user
	connections, err := store.GetActiveConnectionsForUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get active connections for user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connections"})
		return
	}

	if len(connections) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"message":     "No active sessions found for user",
			"userEmail":   userEmail,
			"disconnects": 0,
		})
		return
	}

	// Create disconnect requests for each connection
	disconnectCount := 0
	for _, conn := range connections {
		connID := conn["id"].(string)
		gwID := conn["gateway_id"].(string)
		nodeType := conn["node_type"].(string)
		vpnAddr := conn["vpn_address"].(string)

		_, err := store.CreateDisconnectRequest(ctx, &db.DisconnectRequest{
			UserID:       userID,
			UserEmail:    userEmail,
			GatewayID:    &gwID,
			NodeType:     nodeType,
			ConnectionID: &connID,
			VPNAddress:   &vpnAddr,
			Reason:       req.Reason,
			RequestedBy:  adminEmail,
		})
		if err != nil {
			s.logger.Error("Failed to create disconnect request for connection")
			continue
		}

		// Mark connection as disconnect requested
		_ = store.MarkConnectionDisconnectRequested(ctx, connID, adminEmail, nodeType == "gateway")
		disconnectCount++
	}

	s.logger.Info("User disconnect requested",
		zap.String("userId", userID),
		zap.String("userEmail", userEmail),
		zap.String("requestedBy", adminEmail),
		zap.Int("disconnectCount", disconnectCount),
		zap.String("reason", req.Reason))

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Disconnect requests created",
		"userEmail":   userEmail,
		"disconnects": disconnectCount,
	})
}

// handleGetPendingDisconnects returns pending disconnect requests for a gateway.
// Gateway agents call this to check if any users need to be disconnected.
// POST /api/v1/gateway/pending-disconnects
func (s *Server) handleGetPendingDisconnects(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		GatewayID string `json:"gateway_id" binding:"required"`
		NodeType  string `json:"node_type"` // gateway or hub
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway_id is required"})
		return
	}
	if req.NodeType == "" {
		req.NodeType = "gateway"
	}

	// Verify gateway token
	token := c.GetHeader("X-Gateway-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing gateway token"})
		return
	}

	// Validate token belongs to this gateway
	var gatewayID string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id::text FROM gateways WHERE gateway_token = $1 AND id = $2
		UNION ALL
		SELECT id::text FROM mesh_hubs WHERE gateway_token = $1 AND id = $2
		LIMIT 1
	`, token, req.GatewayID).Scan(&gatewayID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid gateway token"})
		return
	}

	store := db.NewPendingDisconnectStore(s.db)
	disconnects, err := store.GetPendingDisconnectsForGateway(ctx, req.GatewayID, req.NodeType)
	if err != nil {
		s.logger.Error("Failed to get pending disconnects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending disconnects"})
		return
	}

	// Return disconnect list
	c.JSON(http.StatusOK, gin.H{
		"pending_disconnects": disconnects,
		"count":               len(disconnects),
	})
}

// handleAckDisconnect acknowledges that a disconnect has been executed.
// Gateway agents call this after successfully disconnecting a user.
// POST /api/v1/gateway/ack-disconnect
func (s *Server) handleAckDisconnect(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		DisconnectID string `json:"disconnect_id" binding:"required"`
		GatewayID    string `json:"gateway_id" binding:"required"`
		ConnectionID string `json:"connection_id"`
		Success      bool   `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	token := c.GetHeader("X-Gateway-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing gateway token"})
		return
	}

	store := db.NewPendingDisconnectStore(s.db)

	// Mark disconnect as executed
	err := store.MarkDisconnectExecuted(ctx, req.DisconnectID, req.GatewayID)
	if err != nil {
		s.logger.Error("Failed to mark disconnect as executed")
	}

	// If we have a connection ID, mark it as disconnected by admin
	if req.ConnectionID != "" && req.Success {
		// Try gateway_connections first, then mesh_connections
		_ = store.DisconnectUserConnection(ctx, req.ConnectionID, "admin", true)
		_ = store.DisconnectUserConnection(ctx, req.ConnectionID, "admin", false)
	}

	s.logger.Info("Disconnect acknowledged",
		zap.String("disconnectId", req.DisconnectID),
		zap.String("gatewayId", req.GatewayID),
		zap.Bool("success", req.Success))

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleGetUserActiveSessions returns active sessions for a specific user.
// GET /api/v1/admin/users/:id/sessions
func (s *Server) handleGetUserActiveSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")

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

	store := db.NewPendingDisconnectStore(s.db)
	connections, err := store.GetActiveConnectionsForUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get active connections for user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": connections,
		"count":    len(connections),
	})
}

// DisableUserRequest represents a request to disable a user
type DisableUserRequest struct {
	Reason           string `json:"reason"`
	DisconnectActive bool   `json:"disconnect_active"` // Also disconnect active sessions
}

// handleDisableUser disables a user account and optionally disconnects their active sessions.
// This prevents the user from making new connections without affecting other users.
// POST /api/v1/admin/users/:id/disable
func (s *Server) handleDisableUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")

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

	var req DisableUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.DisconnectActive = true // Default to disconnecting active sessions
	}
	if req.Reason == "" {
		req.Reason = "Disabled by administrator"
	}

	adminEmail := admin.Email
	if adminEmail == "" {
		adminEmail = "admin"
	}

	// Try to disable SSO user first
	var userEmail string
	var userType string
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE users SET is_active = false WHERE id = $1 RETURNING email
	`, userID)
	if err == nil && result.RowsAffected() > 0 {
		_ = s.db.Pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&userEmail)
		userType = "sso"
	} else {
		// Try local user
		result, err = s.db.Pool.Exec(ctx, `
			UPDATE local_users SET is_active = false WHERE id = $1 RETURNING email
		`, userID)
		if err != nil || result.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		_ = s.db.Pool.QueryRow(ctx, `SELECT email FROM local_users WHERE id = $1`, userID).Scan(&userEmail)
		userType = "local"
	}

	disconnectCount := 0
	if req.DisconnectActive {
		// Disconnect all active sessions for this user
		store := db.NewPendingDisconnectStore(s.db)
		connections, _ := store.GetActiveConnectionsForUser(ctx, userID)
		for _, conn := range connections {
			connID := conn["id"].(string)
			gwID := conn["gateway_id"].(string)
			nodeType := conn["node_type"].(string)
			vpnAddr := conn["vpn_address"].(string)

			_, err := store.CreateDisconnectRequest(ctx, &db.DisconnectRequest{
				UserID:       userID,
				UserEmail:    userEmail,
				GatewayID:    &gwID,
				NodeType:     nodeType,
				ConnectionID: &connID,
				VPNAddress:   &vpnAddr,
				Reason:       req.Reason,
				RequestedBy:  adminEmail,
			})
			if err == nil {
				disconnectCount++
			}
		}
	}

	s.logger.Info("User disabled",
		zap.String("userId", userID),
		zap.String("userEmail", userEmail),
		zap.String("userType", userType),
		zap.String("disabledBy", adminEmail),
		zap.Int("disconnectedSessions", disconnectCount),
		zap.String("reason", req.Reason))

	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "User disabled",
		"userEmail":            userEmail,
		"userType":             userType,
		"sessionsDisconnected": disconnectCount,
	})
}

// handleEnableUser re-enables a disabled user account.
// POST /api/v1/admin/users/:id/enable
func (s *Server) handleEnableUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")

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

	adminEmail := admin.Email
	if adminEmail == "" {
		adminEmail = "admin"
	}

	// Try to enable SSO user first
	var userEmail string
	var userType string
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE users SET is_active = true WHERE id = $1 RETURNING email
	`, userID)
	if err == nil && result.RowsAffected() > 0 {
		_ = s.db.Pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&userEmail)
		userType = "sso"
	} else {
		// Try local user
		result, err = s.db.Pool.Exec(ctx, `
			UPDATE local_users SET is_active = true WHERE id = $1 RETURNING email
		`, userID)
		if err != nil || result.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		_ = s.db.Pool.QueryRow(ctx, `SELECT email FROM local_users WHERE id = $1`, userID).Scan(&userEmail)
		userType = "local"
	}

	s.logger.Info("User enabled",
		zap.String("userId", userID),
		zap.String("userEmail", userEmail),
		zap.String("userType", userType),
		zap.String("enabledBy", adminEmail))

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "User enabled",
		"userEmail": userEmail,
		"userType":  userType,
	})
}

// handleGetUserStatus returns the status of a user (active/disabled, active sessions).
// GET /api/v1/admin/users/:id/status
func (s *Server) handleGetUserStatus(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")

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

	// Try SSO user first
	var userEmail, userType string
	var isActive bool
	err = s.db.Pool.QueryRow(ctx, `
		SELECT email, is_active FROM users WHERE id = $1
	`, userID).Scan(&userEmail, &isActive)
	if err == nil {
		userType = "sso"
	} else {
		// Try local user
		err = s.db.Pool.QueryRow(ctx, `
			SELECT email, is_active FROM local_users WHERE id = $1
		`, userID).Scan(&userEmail, &isActive)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		userType = "local"
	}

	// Get active session count
	store := db.NewPendingDisconnectStore(s.db)
	connections, _ := store.GetActiveConnectionsForUser(ctx, userID)

	c.JSON(http.StatusOK, gin.H{
		"userId":         userID,
		"userEmail":      userEmail,
		"userType":       userType,
		"isActive":       isActive,
		"activeSessions": len(connections),
		"sessions":       connections,
	})
}

// calculateServerTunnelIP calculates the server's tunnel IP from a VPN subnet
// For example, 172.30.0.0/16 -> 172.30.0.1
func calculateServerTunnelIP(subnet string) string {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return ""
	}

	// Get the network address and increment to get first usable IP
	ip := ipNet.IP.To4()
	if ip == nil {
		ip = ipNet.IP.To16()
	}
	if ip == nil {
		return ""
	}

	// Increment IP by 1 to get first usable address
	ip[len(ip)-1]++
	return ip.String()
}
