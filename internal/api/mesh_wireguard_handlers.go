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

	type peerInfo struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		TunnelIP     string   `json:"tunnel_ip,omitempty"`
	}

	peers := make([]peerInfo, 0, len(spokes))
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
		for _, network := range spoke.LocalNetworks {
			allowedIPs = append(allowedIPs, network)
		}

		peers = append(peers, peerInfo{
			ID:           spoke.ID,
			Name:         spoke.Name,
			PublicKey:    spoke.WGPublicKey,
			PresharedKey: spoke.WGPresharedKey,
			AllowedIPs:   allowedIPs,
			TunnelIP:     spoke.TunnelIP,
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

	type peerInfo struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		TunnelIP     string   `json:"tunnel_ip,omitempty"`
	}

	peers := make([]peerInfo, 0, len(spokes))
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
		for _, network := range spoke.LocalNetworks {
			allowedIPs = append(allowedIPs, network)
		}

		peers = append(peers, peerInfo{
			ID:           spoke.ID,
			Name:         spoke.Name,
			PublicKey:    spoke.WGPublicKey,
			PresharedKey: spoke.WGPresharedKey,
			AllowedIPs:   allowedIPs,
			TunnelIP:     spoke.TunnelIP,
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
		"spoke_id":       spoke.ID,
		"spoke_name":     spoke.Name,
		"private_key":    spoke.WGPrivateKey,
		"public_key":     spoke.WGPublicKey,
		"preshared_key":  spoke.WGPresharedKey,
		"tunnel_ip":      spoke.TunnelIP,
		"interface_name": "wg0",
		"hub": gin.H{
			"id":         hub.ID,
			"name":       hub.Name,
			"public_key": hub.WGPublicKey,
			"endpoint":   hubEndpoint,
			"allowed_ips": allowedIPs,
		},
		"dns_servers":           dnsServers,
		"persistent_keepalive":  25,
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
