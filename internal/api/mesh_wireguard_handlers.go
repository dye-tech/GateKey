package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/wireguard"
)

// WireGuard Mesh Hub Handlers

// handleWireGuardMeshHubProvision handles WireGuard mesh hub provisioning
func (s *Server) handleWireGuardMeshHubProvision(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get hub by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Verify this is a WireGuard hub
	if hub.GatewayType != db.MeshGatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub is not a WireGuard hub"})
		return
	}

	// Generate WireGuard keypair if not already set
	if hub.WGPrivateKey == "" || hub.WGPublicKey == "" {
		keyPair, err := wireguard.GenerateKeyPair()
		if err != nil {
			s.logger.Error("Failed to generate WireGuard keypair", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate keys"})
			return
		}

		if err := s.meshStore.SetHubWireGuardKeys(ctx, hub.ID, keyPair.PrivateKey, keyPair.PublicKey); err != nil {
			s.logger.Error("Failed to save WireGuard keys", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save keys"})
			return
		}

		hub.WGPrivateKey = keyPair.PrivateKey
		hub.WGPublicKey = keyPair.PublicKey
	}

	// Get authorized spokes (peers) for this hub
	spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get spokes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get spokes"})
		return
	}

	// Get authorized WireGuard mobile clients for this hub
	mobileClients, err := s.meshStore.ListActiveWGMeshPeers(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get mobile clients", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mobile clients"})
		return
	}

	type peerInfo struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		TunnelIP     string   `json:"tunnel_ip,omitempty"`
		IsMobile     bool     `json:"is_mobile,omitempty"`
	}

	peers := make([]peerInfo, 0, len(spokes)+len(mobileClients))

	// Add spokes
	for _, spoke := range spokes {
		// Only include spokes that have WireGuard keys
		if spoke.WGPublicKey == "" {
			continue
		}

		// Build allowed IPs from spoke's local networks + tunnel IP
		allowedIPs := make([]string, 0)
		if spoke.TunnelIP != "" {
			allowedIPs = append(allowedIPs, spoke.TunnelIP+"/32")
		}
		allowedIPs = append(allowedIPs, spoke.LocalNetworks...)

		peers = append(peers, peerInfo{
			ID:           spoke.ID,
			Name:         spoke.Name,
			PublicKey:    spoke.WGPublicKey,
			PresharedKey: spoke.WGPresharedKey,
			AllowedIPs:   allowedIPs,
			TunnelIP:     spoke.TunnelIP,
		})
	}

	// Add mobile clients
	for _, client := range mobileClients {
		peers = append(peers, peerInfo{
			ID:           client.ID,
			Name:         client.Email, // Use email as name for mobile clients
			PublicKey:    client.PublicKey,
			PresharedKey: client.PresharedKey,
			AllowedIPs:   []string{client.AssignedIP + "/32"}, // Mobile clients only get their own IP
			TunnelIP:     client.AssignedIP,
			IsMobile:     true,
		})
	}

	// Update hub status
	publicIP := c.ClientIP()
	if err := s.meshStore.UpdateHubHeartbeat(ctx, hub.ID, publicIP); err != nil {
		s.logger.Error("Failed to update hub heartbeat", zap.Error(err))
	}

	// Determine listen port
	listenPort := hub.WGListenPort
	if listenPort == 0 {
		listenPort = db.DefaultWGListenPort
	}

	s.logger.Info("WireGuard mesh hub provisioned",
		zap.String("hub_id", hub.ID),
		zap.String("hub_name", hub.Name),
		zap.Int("peer_count", len(peers)))

	c.JSON(http.StatusOK, gin.H{
		"hub_id":         hub.ID,
		"hub_name":       hub.Name,
		"private_key":    hub.WGPrivateKey,
		"public_key":     hub.WGPublicKey,
		"listen_port":    listenPort,
		"vpn_subnet":     hub.VPNSubnet,
		"interface_name": "wg0",
		"peers":          peers,
		"full_tunnel":    hub.FullTunnelMode,
		"push_dns":       hub.PushDNS,
		"dns_servers":    hub.DNSServers,
	})
}

// handleWireGuardMeshHubHeartbeat handles WireGuard mesh hub heartbeat
func (s *Server) handleWireGuardMeshHubHeartbeat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token            string `json:"token" binding:"required"`
		PeerCount        int    `json:"peer_count"`
		ConnectedClients int    `json:"connected_clients"`
		Version          string `json:"version"`
		Uptime           int64  `json:"uptime"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get hub by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Update heartbeat and public IP
	publicIP := c.ClientIP()
	if err := s.meshStore.UpdateHubHeartbeat(ctx, hub.ID, publicIP); err != nil {
		s.logger.Error("Failed to update hub heartbeat", zap.Error(err))
	}

	// Update connected counts
	if err := s.meshStore.UpdateHubConnectedCounts(ctx, hub.ID, req.PeerCount, req.ConnectedClients); err != nil {
		s.logger.Error("Failed to update hub connected counts", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleWireGuardMeshHubSyncPeers handles peer sync request from WireGuard mesh hub
func (s *Server) handleWireGuardMeshHubSyncPeers(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get hub by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Get authorized spokes for this hub
	spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get spokes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get spokes"})
		return
	}

	// Get authorized WireGuard mobile clients for this hub
	mobileClients, err := s.meshStore.ListActiveWGMeshPeers(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get mobile clients", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mobile clients"})
		return
	}

	type peerInfo struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		TunnelIP     string   `json:"tunnel_ip,omitempty"`
		IsMobile     bool     `json:"is_mobile,omitempty"`
	}

	peers := make([]peerInfo, 0, len(spokes)+len(mobileClients))

	// Add spokes
	for _, spoke := range spokes {
		// Only include spokes that have WireGuard keys
		if spoke.WGPublicKey == "" {
			continue
		}

		// Build allowed IPs from spoke's local networks + tunnel IP
		allowedIPs := make([]string, 0)
		if spoke.TunnelIP != "" {
			allowedIPs = append(allowedIPs, spoke.TunnelIP+"/32")
		}
		allowedIPs = append(allowedIPs, spoke.LocalNetworks...)

		peers = append(peers, peerInfo{
			ID:           spoke.ID,
			Name:         spoke.Name,
			PublicKey:    spoke.WGPublicKey,
			PresharedKey: spoke.WGPresharedKey,
			AllowedIPs:   allowedIPs,
			TunnelIP:     spoke.TunnelIP,
		})
	}

	// Add mobile clients
	for _, client := range mobileClients {
		peers = append(peers, peerInfo{
			ID:           client.ID,
			Name:         client.Email, // Use email as name for mobile clients
			PublicKey:    client.PublicKey,
			PresharedKey: client.PresharedKey,
			AllowedIPs:   []string{client.AssignedIP + "/32"}, // Mobile clients only get their own IP
			TunnelIP:     client.AssignedIP,
			IsMobile:     true,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"peers": peers,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleWireGuardMeshHubPeerStats handles peer statistics from WireGuard mesh hub
func (s *Server) handleWireGuardMeshHubPeerStats(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
		Peers []struct {
			PublicKey     string `json:"public_key"`
			Endpoint      string `json:"endpoint"`
			LastHandshake int64  `json:"last_handshake"`
			BytesSent     int64  `json:"bytes_sent"`
			BytesReceived int64  `json:"bytes_received"`
		} `json:"peers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get hub by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Update peer stats for each spoke
	for _, peer := range req.Peers {
		// Find spoke by public key
		spoke, err := s.meshStore.GetMeshSpokeByWGPublicKey(ctx, hub.ID, peer.PublicKey)
		if err != nil {
			// Spoke might have been deleted
			continue
		}

		var lastHandshake *time.Time
		if peer.LastHandshake > 0 {
			t := time.Unix(peer.LastHandshake, 0)
			lastHandshake = &t
		}

		// Update spoke status based on peer stats
		status := db.MeshSpokeStatusConnected
		if lastHandshake == nil || time.Since(*lastHandshake) > 3*time.Minute {
			status = db.MeshSpokeStatusDisconnected
		}

		if err := s.meshStore.UpdateMeshSpokeStats(ctx, spoke.ID, status, peer.Endpoint, peer.BytesSent, peer.BytesReceived, lastHandshake); err != nil {
			s.logger.Warn("Failed to update spoke stats",
				zap.String("spoke_id", spoke.ID),
				zap.String("public_key", peer.PublicKey[:8]+"..."),
				zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// WireGuard Mesh Spoke Handlers

// handleWireGuardMeshSpokeProvision handles WireGuard mesh spoke provisioning
func (s *Server) handleWireGuardMeshSpokeProvision(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spoke, err := s.meshStore.GetMeshSpokeByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get spoke by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Verify this is a WireGuard spoke
	if spoke.GatewayType != db.MeshGatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spoke is not a WireGuard spoke"})
		return
	}

	// Get the hub
	hub, err := s.meshStore.GetHub(ctx, spoke.HubID)
	if err != nil {
		s.logger.Error("Failed to get hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub"})
		return
	}

	// Verify hub is also WireGuard type
	if hub.GatewayType != db.MeshGatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub is not a WireGuard hub"})
		return
	}

	// Generate WireGuard keypair for spoke if not already set
	if spoke.WGPrivateKey == "" || spoke.WGPublicKey == "" {
		keyPair, err := wireguard.GenerateKeyPair()
		if err != nil {
			s.logger.Error("Failed to generate WireGuard keypair", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate keys"})
			return
		}

		// Generate preshared key for additional security
		psk, err := wireguard.GeneratePresharedKey()
		if err != nil {
			s.logger.Error("Failed to generate preshared key", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate preshared key"})
			return
		}

		if err := s.meshStore.SetSpokeWireGuardKeys(ctx, spoke.ID, keyPair.PrivateKey, keyPair.PublicKey, psk); err != nil {
			s.logger.Error("Failed to save WireGuard keys", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save keys"})
			return
		}

		spoke.WGPrivateKey = keyPair.PrivateKey
		spoke.WGPublicKey = keyPair.PublicKey
		spoke.WGPresharedKey = psk
	}

	// Allocate tunnel IP if not already assigned
	if spoke.TunnelIP == "" {
		// Get already allocated IPs for this hub
		spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
		if err != nil {
			s.logger.Error("Failed to list spokes", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to allocate IP"})
			return
		}

		allocator, err := wireguard.NewIPAllocator(hub.VPNSubnet)
		if err != nil {
			s.logger.Error("Failed to create IP allocator", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid VPN subnet"})
			return
		}

		// Mark already allocated IPs
		for _, s := range spokes {
			if s.TunnelIP != "" {
				allocator.MarkAllocated(s.TunnelIP)
			}
		}

		tunnelIP, err := allocator.Allocate()
		if err != nil {
			s.logger.Error("Failed to allocate tunnel IP", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no available IPs"})
			return
		}

		if err := s.meshStore.UpdateMeshSpokeTunnelIP(ctx, spoke.ID, tunnelIP); err != nil {
			s.logger.Error("Failed to save tunnel IP", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save tunnel IP"})
			return
		}

		spoke.TunnelIP = tunnelIP
	}

	// Ensure hub has WireGuard keys
	if hub.WGPublicKey == "" {
		// Auto-provision hub keys
		keyPair, err := wireguard.GenerateKeyPair()
		if err != nil {
			s.logger.Error("Failed to generate hub WireGuard keypair", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate hub keys"})
			return
		}

		if err := s.meshStore.SetHubWireGuardKeys(ctx, hub.ID, keyPair.PrivateKey, keyPair.PublicKey); err != nil {
			s.logger.Error("Failed to save hub WireGuard keys", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save hub keys"})
			return
		}

		hub.WGPrivateKey = keyPair.PrivateKey
		hub.WGPublicKey = keyPair.PublicKey
	}

	// Determine hub endpoint
	hubEndpoint := hub.PublicEndpoint
	listenPort := hub.WGListenPort
	if listenPort == 0 {
		listenPort = db.DefaultWGListenPort
	}
	// If endpoint doesn't already have a port, add the WireGuard port
	if !strings.Contains(hubEndpoint, ":") {
		hubEndpoint = fmt.Sprintf("%s:%d", hubEndpoint, listenPort)
	}

	// Build allowed IPs for the hub peer (all traffic or just hub networks)
	var allowedIPs []string
	if hub.FullTunnelMode {
		allowedIPs = []string{"0.0.0.0/0", "::/0"}
	} else {
		// Get hub networks for routing
		allowedIPs = append(allowedIPs, hub.VPNSubnet)
		// Add any additional hub-specific networks here
	}

	// Update spoke status
	publicIP := c.ClientIP()
	if err := s.meshStore.UpdateMeshSpokeHeartbeat(ctx, spoke.ID, publicIP); err != nil {
		s.logger.Error("Failed to update spoke heartbeat", zap.Error(err))
	}

	// Build DNS servers list
	var dnsServers []string
	if hub.PushDNS {
		if len(hub.DNSServers) > 0 {
			dnsServers = hub.DNSServers
		} else {
			dnsServers = []string{"1.1.1.1", "8.8.8.8"}
		}
	}

	s.logger.Info("WireGuard mesh spoke provisioned",
		zap.String("spoke_id", spoke.ID),
		zap.String("spoke_name", spoke.Name),
		zap.String("hub_name", hub.Name),
		zap.String("tunnel_ip", spoke.TunnelIP))

	c.JSON(http.StatusOK, gin.H{
		"spoke_id":             spoke.ID,
		"spoke_name":           spoke.Name,
		"private_key":          spoke.WGPrivateKey,
		"public_key":           spoke.WGPublicKey,
		"preshared_key":        spoke.WGPresharedKey,
		"tunnel_ip":            spoke.TunnelIP,
		"hub_endpoint":         hubEndpoint,
		"hub_public_key":       hub.WGPublicKey,
		"hub_allowed_ips":      allowedIPs,
		"local_networks":       spoke.LocalNetworks,
		"full_tunnel":          hub.FullTunnelMode,
		"dns_servers":          dnsServers,
		"config_version":       hub.ConfigVersion,
		"persistent_keepalive": 25,
	})
}

// handleWireGuardMeshSpokeHeartbeat handles WireGuard mesh spoke heartbeat
func (s *Server) handleWireGuardMeshSpokeHeartbeat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token         string `json:"token" binding:"required"`
		BytesSent     int64  `json:"bytes_sent"`
		BytesReceived int64  `json:"bytes_received"`
		LastHandshake int64  `json:"last_handshake"`
		Version       string `json:"version"`
		Uptime        int64  `json:"uptime"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spoke, err := s.meshStore.GetMeshSpokeByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get spoke by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Update heartbeat and public IP
	publicIP := c.ClientIP()
	if err := s.meshStore.UpdateMeshSpokeHeartbeat(ctx, spoke.ID, publicIP); err != nil {
		s.logger.Error("Failed to update spoke heartbeat", zap.Error(err))
	}

	// Update stats if provided
	if req.BytesSent > 0 || req.BytesReceived > 0 {
		var lastHandshake *time.Time
		if req.LastHandshake > 0 {
			t := time.Unix(req.LastHandshake, 0)
			lastHandshake = &t
		}

		status := db.MeshSpokeStatusConnected
		if err := s.meshStore.UpdateMeshSpokeStats(ctx, spoke.ID, status, publicIP, req.BytesSent, req.BytesReceived, lastHandshake); err != nil {
			s.logger.Error("Failed to update spoke stats", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGenerateWireGuardMeshClientConfig generates a WireGuard config for connecting to a mesh hub
func (s *Server) handleGenerateWireGuardMeshClientConfig(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		HubID string `json:"hubid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hubid is required"})
		return
	}

	// Get the hub
	hub, err := s.meshStore.GetHub(ctx, req.HubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "mesh hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	// Verify hub is WireGuard type
	if hub.GatewayType != db.MeshGatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub is not a WireGuard hub"})
		return
	}

	// Check if hub is online
	if hub.Status != db.MeshHubStatusOnline {
		c.JSON(http.StatusForbidden, gin.H{"error": "mesh hub is not online"})
		return
	}

	// Check if hub has been provisioned (has public key)
	if hub.WGPublicKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub has not been provisioned"})
		return
	}

	// Check if user has access to this hub
	hasAccess, err := s.meshStore.UserHasHubAccess(ctx, user.UserID, hub.ID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to check hub access", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check access"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this mesh hub"})
		return
	}

	// Get VPN certificate validity
	validityHours := s.settingsStore.GetInt(ctx, db.SettingVPNCertValidityHours, 24)
	expiresAt := time.Now().Add(time.Duration(validityHours) * time.Hour)

	// Generate client keypair
	clientKeyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		s.logger.Error("Failed to generate client keypair", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate keys"})
		return
	}

	// Generate preshared key for additional security
	presharedKey, err := wireguard.GeneratePresharedKey()
	if err != nil {
		s.logger.Error("Failed to generate preshared key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate preshared key"})
		return
	}

	// Get already allocated IPs from existing peers
	allocatedIPs, err := s.meshStore.GetHubAllocatedIPs(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get allocated IPs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to allocate IP"})
		return
	}

	// Create IP allocator for the hub's VPN subnet
	allocator, err := wireguard.NewIPAllocator(hub.VPNSubnet)
	if err != nil {
		s.logger.Error("Failed to create IP allocator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid VPN subnet"})
		return
	}

	// Mark already allocated IPs
	for _, ip := range allocatedIPs {
		allocator.MarkAllocated(ip)
	}

	assignedIP, err := allocator.Allocate()
	if err != nil {
		s.logger.Error("Failed to allocate IP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no available IPs"})
		return
	}

	// Build AllowedIPs based on hub settings and user access rules
	var allowedIPs []string
	if hub.FullTunnelMode {
		// Full tunnel mode - route all traffic through VPN
		allowedIPs = []string{"0.0.0.0/0", "::/0"}
	} else {
		// Get access rules for this user from the mesh hub's networks (zero-trust)
		routes, err := s.meshStore.GetUserMeshAccessRules(ctx, hub.ID, user.UserID, user.Groups)
		if err != nil {
			s.logger.Warn("Failed to get user mesh access rules", zap.Error(err))
		}

		// Convert routes to allowed IPs
		for _, route := range routes {
			if route != "" {
				allowedIPs = append(allowedIPs, route)
			}
		}

		// Get spoke local_networks for spokes the user has access to
		// This eliminates the need to duplicate network assignments on the hub
		spokeRoutes, err := s.meshStore.GetUserMeshRoutes(ctx, hub.ID, user.UserID, user.Groups)
		if err != nil {
			s.logger.Warn("Failed to get user mesh routes from spokes", zap.Error(err))
		}
		for _, route := range spokeRoutes {
			if route != "" {
				allowedIPs = append(allowedIPs, route)
			}
		}

		// Deduplicate routes
		seen := make(map[string]bool)
		uniqueIPs := make([]string, 0, len(allowedIPs))
		for _, ip := range allowedIPs {
			if !seen[ip] {
				seen[ip] = true
				uniqueIPs = append(uniqueIPs, ip)
			}
		}
		allowedIPs = uniqueIPs

		// If no routes defined, at least allow the VPN subnet for hub communication
		if len(allowedIPs) == 0 {
			allowedIPs = []string{hub.VPNSubnet}
		}
	}

	// Determine hub endpoint
	hubEndpoint := hub.PublicEndpoint
	if hubEndpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub has no public endpoint configured"})
		return
	}
	// Add port if not already present
	if !strings.Contains(hubEndpoint, ":") {
		if hub.WGListenPort > 0 {
			hubEndpoint = fmt.Sprintf("%s:%d", hubEndpoint, hub.WGListenPort)
		} else {
			hubEndpoint = fmt.Sprintf("%s:51820", hubEndpoint)
		}
	}

	// Generate config ID
	configID := generateUUID()

	// Build DNS config
	var dnsServers string
	if hub.PushDNS && len(hub.DNSServers) > 0 {
		dnsServers = strings.Join(hub.DNSServers, ", ")
	}

	// Generate WireGuard config file content
	var configBuilder strings.Builder
	configBuilder.WriteString("[Interface]\n")
	configBuilder.WriteString(fmt.Sprintf("PrivateKey = %s\n", clientKeyPair.PrivateKey))
	configBuilder.WriteString(fmt.Sprintf("Address = %s\n", assignedIP))
	if dnsServers != "" {
		configBuilder.WriteString(fmt.Sprintf("DNS = %s\n", dnsServers))
	}
	configBuilder.WriteString("\n[Peer]\n")
	configBuilder.WriteString(fmt.Sprintf("PublicKey = %s\n", hub.WGPublicKey))
	configBuilder.WriteString(fmt.Sprintf("PresharedKey = %s\n", presharedKey))
	configBuilder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(allowedIPs, ", ")))
	configBuilder.WriteString(fmt.Sprintf("Endpoint = %s\n", hubEndpoint))
	configBuilder.WriteString("PersistentKeepalive = 25\n")

	configData := configBuilder.String()
	fileName := fmt.Sprintf("mesh-wg-%s.conf", hub.Name)

	// Save config to database for tracking FIRST (peer has FK to this table)
	meshConfig := &db.MeshGeneratedConfig{
		ID:           configID,
		UserID:       user.UserID,
		HubID:        hub.ID,
		HubName:      hub.Name,
		FileName:     fileName,
		ConfigData:   []byte(configData),
		SerialNumber: clientKeyPair.PublicKey[:16], // Use part of public key as identifier
		Fingerprint:  clientKeyPair.PublicKey,
		ExpiresAt:    expiresAt,
	}

	if err := s.meshConfigStore.SaveConfig(ctx, meshConfig); err != nil {
		s.logger.Error("Failed to save mesh config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config"})
		return
	}

	// Register the peer with the hub (so it gets added to the hub's peer list)
	// Must happen AFTER config is saved due to foreign key constraint
	peerID := generateUUID()
	peer := &db.WGMeshPeer{
		ID:           peerID,
		HubID:        hub.ID,
		ConfigID:     configID,
		UserID:       user.UserID,
		Email:        user.Email,
		PublicKey:    clientKeyPair.PublicKey,
		PresharedKey: presharedKey,
		AssignedIP:   assignedIP,
		AllowedIPs:   allowedIPs,
		ExpiresAt:    expiresAt,
	}

	if err := s.meshStore.AddWGMeshPeer(ctx, peer); err != nil {
		s.logger.Error("Failed to register mesh peer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register peer"})
		return
	}

	s.logger.Info("WireGuard mesh config generated and saved",
		zap.String("config_id", configID),
		zap.String("user", user.Email),
		zap.String("hub", hub.Name),
		zap.String("assigned_ip", assignedIP))

	c.JSON(http.StatusOK, gin.H{
		"id":        configID,
		"hubname":   hub.Name,
		"config":    configData,
		"expiresAt": expiresAt.Format(time.RFC3339),
	})
}
