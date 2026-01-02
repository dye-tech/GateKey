package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
			ClientCount:   0, // TODO: get from active connections
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
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	UserEmail   string    `json:"userEmail"`
	UserName    string    `json:"userName"`
	GatewayID   string    `json:"gatewayId"`
	GatewayName string    `json:"gatewayName"`
	NodeType    string    `json:"nodeType"` // gateway, hub
	ClientIP    string    `json:"clientIp"`
	VPNAddress  string    `json:"vpnAddress"`
	ConnectedAt time.Time `json:"connectedAt"`
	BytesSent   int64     `json:"bytesSent"`
	BytesRecv   int64     `json:"bytesRecv"`
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

// getActiveMeshConnections queries the mesh_connections table for active sessions
func (s *Server) getActiveMeshConnections(ctx context.Context) ([]ActiveSession, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT
			mc.id, mc.hub_id, u.id, u.email, COALESCE(u.name, ''),
			h.name, host(mc.client_ip), host(mc.tunnel_ip),
			mc.bytes_sent, mc.bytes_received, mc.connected_at
		FROM mesh_connections mc
		JOIN users u ON mc.user_id = u.id
		JOIN mesh_hubs h ON mc.hub_id = h.id
		WHERE mc.disconnected_at IS NULL
		ORDER BY mc.connected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ActiveSession
	for rows.Next() {
		var s ActiveSession
		var clientIP, tunnelIP *string
		if err := rows.Scan(
			&s.ID, &s.GatewayID, &s.UserID, &s.UserEmail, &s.UserName,
			&s.GatewayName, &clientIP, &tunnelIP,
			&s.BytesSent, &s.BytesRecv, &s.ConnectedAt,
		); err != nil {
			return nil, err
		}
		s.NodeType = "hub"
		if clientIP != nil {
			s.ClientIP = *clientIP
		}
		if tunnelIP != nil {
			s.VPNAddress = *tunnelIP
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
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

