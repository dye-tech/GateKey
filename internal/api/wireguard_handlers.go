package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/agent"
	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/wireguard"
)

// WireGuard User Config Handlers

// handleGenerateWireGuardConfig generates a WireGuard config for the authenticated user
func (s *Server) handleGenerateWireGuardConfig(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		GatewayID string `json:"gateway_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get the gateway
	gateway, err := s.gatewayStore.GetGateway(ctx, req.GatewayID)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway"})
		return
	}

	// Verify gateway is WireGuard type
	if gateway.GatewayType != db.GatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway is not a WireGuard gateway"})
		return
	}

	// Verify gateway is active
	if !gateway.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway is not active"})
		return
	}

	// Verify gateway has been provisioned (has public key)
	if gateway.WGPublicKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway has not been provisioned"})
		return
	}

	// Check if user has access to this gateway
	hasAccess, err := s.gatewayStore.UserHasGatewayAccess(ctx, user.UserID, gateway.ID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to check gateway access", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check access"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access to this gateway"})
		return
	}

	// Check geo-fencing if enabled
	clientIP := c.ClientIP()
	if s.settingsStore.GetBool(ctx, db.SettingGeoFencingEnabled, false) && clientIP != "" {
		allowed, geoErr := s.geoFenceStore.IsIPAllowed(ctx, clientIP, user.UserID, user.Groups)
		if geoErr != nil {
			s.logger.Error("Geo-fence check failed for WireGuard config generation", zap.Error(geoErr))
			// Continue (fail-open) but log the error
		} else if !allowed {
			enforceMode, _ := s.settingsStore.Get(ctx, db.SettingGeoFencingEnforce)
			if enforceMode != nil && enforceMode.Value == "enforce" {
				s.logger.Warn("Geo-fence blocked WireGuard config generation",
					zap.String("user_id", user.UserID),
					zap.String("client_ip", clientIP))
				c.JSON(http.StatusForbidden, gin.H{"error": "geo-fence: connection from this IP is not allowed"})
				return
			}
			s.logger.Warn("Geo-fence would block WireGuard config generation (audit mode)",
				zap.String("user_id", user.UserID),
				zap.String("client_ip", clientIP))
		}
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

	// Generate optional preshared key
	presharedKey, err := wireguard.GeneratePresharedKey()
	if err != nil {
		s.logger.Error("Failed to generate preshared key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate preshared key"})
		return
	}

	// Allocate an IP address for this client
	allocatedIPs, err := s.wgConfigStore.GetAllocatedIPs(ctx, gateway.ID)
	if err != nil {
		s.logger.Error("Failed to get allocated IPs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to allocate IP"})
		return
	}

	allocator, err := wireguard.NewIPAllocator(gateway.VPNSubnet)
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

	// Get user's access rules to build AllowedIPs
	var allowedIPs []string
	var accessRulesForDNS []*db.AccessRule
	if gateway.FullTunnelMode {
		// Full tunnel mode - route all traffic through VPN
		allowedIPs = []string{"0.0.0.0/0", "::/0"}
	} else {
		// Get access rules for this user
		rules, err := s.accessRuleStore.GetUserAccessRules(ctx, user.UserID, user.Groups)
		if err != nil {
			s.logger.Error("Failed to get access rules", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access rules"})
			return
		}
		accessRulesForDNS = rules

		// Build allowed IPs from CIDR and IP-type rules
		for _, rule := range rules {
			if rule.Value == "" {
				continue
			}
			switch rule.RuleType {
			case db.AccessRuleTypeCIDR:
				allowedIPs = append(allowedIPs, rule.Value)
			case db.AccessRuleTypeIP:
				// Convert single IP to /32 CIDR for WireGuard
				allowedIPs = append(allowedIPs, rule.Value+"/32")
			case db.AccessRuleTypeHostname, db.AccessRuleTypeHostnameWildcard:
				// Hostname rules are not directly usable in WireGuard allowed-ips
				// These would need DNS resolution which is not supported here
				continue
			}
		}

		// If no CIDRs, at least allow the VPN subnet
		if len(allowedIPs) == 0 {
			allowedIPs = append(allowedIPs, gateway.VPNSubnet)
		}
	}

	// Build DNS servers list
	var dnsServers []string
	if gateway.PushDNS {
		if len(gateway.DNSServers) > 0 {
			dnsServers = gateway.DNSServers
		} else {
			dnsServers = []string{"1.1.1.1", "8.8.8.8"}
		}
	}

	// Merge in network-level DNS servers from Split DNS rules
	{
		// Collect unique network IDs from the user's access rules
		networkIDSet := make(map[string]bool)
		if !gateway.FullTunnelMode {
			// accessRulesForDNS already fetched above
			for _, rule := range accessRulesForDNS {
				if rule.NetworkID != nil && *rule.NetworkID != "" {
					networkIDSet[*rule.NetworkID] = true
				}
			}
		} else {
			// In full tunnel mode, get all gateway networks
			gwNetworks, gwErr := s.networkStore.GetGatewayNetworks(ctx, gateway.ID)
			if gwErr == nil {
				for _, n := range gwNetworks {
					networkIDSet[n.ID] = true
				}
			}
		}

		networkIDs := make([]string, 0, len(networkIDSet))
		for id := range networkIDSet {
			networkIDs = append(networkIDs, id)
		}

		if len(networkIDs) > 0 {
			dnsRules, dnsErr := s.dnsStore.GetRulesForNetworks(ctx, networkIDs)
			if dnsErr != nil {
				s.logger.Warn("Failed to get DNS rules for WireGuard config", zap.Error(dnsErr))
			} else {
				dnsSet := make(map[string]bool)
				for _, srv := range dnsServers {
					dnsSet[srv] = true
				}
				for _, rule := range dnsRules {
					for _, srv := range rule.DNSServers {
						if !dnsSet[srv] {
							dnsSet[srv] = true
							dnsServers = append(dnsServers, srv)
						}
					}
				}
			}
		}
	}

	// Determine gateway endpoint
	endpoint := gateway.Hostname
	if endpoint == "" {
		endpoint = gateway.PublicIP
	}
	if gateway.WGListenPort != 0 {
		endpoint = fmt.Sprintf("%s:%d", endpoint, gateway.WGListenPort)
	} else {
		endpoint = fmt.Sprintf("%s:%d", endpoint, db.DefaultWGListenPort)
	}

	// Generate the config
	configReq := wireguard.ClientConfigRequest{
		GatewayName:         gateway.Name,
		GatewayEndpoint:     endpoint,
		GatewayPublicKey:    gateway.WGPublicKey,
		ClientPrivateKey:    clientKeyPair.PrivateKey,
		ClientAddress:       assignedIP,
		AllowedIPs:          allowedIPs,
		DNS:                 dnsServers,
		PresharedKey:        presharedKey,
		PersistentKeepalive: 25,
		ExpiresAt:           expiresAt,
		UserEmail:           user.Email,
	}

	generatedConfig, err := wireguard.GenerateClientConfig(configReq)
	if err != nil {
		s.logger.Error("Failed to generate config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate config"})
		return
	}

	// Store the config in database
	configID := uuid.New().String()
	fileName := fmt.Sprintf("%s-%s.conf", strings.ReplaceAll(gateway.Name, " ", "_"), time.Now().Format("20060102"))

	wgConfig := &db.WireGuardConfig{
		ID:               configID,
		UserID:           user.UserID,
		GatewayID:        gateway.ID,
		GatewayName:      gateway.Name,
		FileName:         fileName,
		ConfigData:       generatedConfig.Content,
		ClientPrivateKey: clientKeyPair.PrivateKey,
		ClientPublicKey:  clientKeyPair.PublicKey,
		AssignedIP:       assignedIP,
		PresharedKey:     presharedKey,
		ExpiresAt:        expiresAt,
	}

	if err := s.wgConfigStore.Create(ctx, wgConfig); err != nil {
		s.logger.Error("Failed to store config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store config"})
		return
	}

	s.logger.Info("Generated WireGuard config",
		zap.String("config_id", configID),
		zap.String("user_id", user.UserID),
		zap.String("user_email", user.Email),
		zap.String("gateway_id", gateway.ID),
		zap.String("assigned_ip", assignedIP))

	// Trigger immediate peer sync on the gateway so client can connect immediately
	// This is done async to not delay the response to the client
	go s.triggerGatewayPeerSync(gateway)

	c.JSON(http.StatusOK, gin.H{
		"id":          configID,
		"fileName":    fileName,
		"gateway":     gateway.Name,
		"expiresAt":   expiresAt.Format(time.RFC3339),
		"assignedIp":  assignedIP,
		"downloadUrl": "/api/v1/wireguard/configs/download/" + configID,
	})
}

// handleListUserWireGuardConfigs lists WireGuard configs for the authenticated user
func (s *Server) handleListUserWireGuardConfigs(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	configs, err := s.wgConfigStore.GetUserConfigs(ctx, user.UserID)
	if err != nil {
		s.logger.Error("Failed to list configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs"})
		return
	}

	type configResponse struct {
		ID           string     `json:"id"`
		GatewayName  string     `json:"gateway_name"`
		FileName     string     `json:"file_name"`
		AssignedIP   string     `json:"assigned_ip"`
		ExpiresAt    time.Time  `json:"expires_at"`
		CreatedAt    time.Time  `json:"created_at"`
		DownloadedAt *time.Time `json:"downloaded_at,omitempty"`
		RevokedAt    *time.Time `json:"revoked_at,omitempty"`
		Status       string     `json:"status"`
	}

	response := make([]configResponse, 0, len(configs))
	now := time.Now()
	for _, cfg := range configs {
		status := "active"
		if cfg.RevokedAt != nil {
			status = "revoked"
		} else if now.After(cfg.ExpiresAt) {
			status = "expired"
		}

		response = append(response, configResponse{
			ID:           cfg.ID,
			GatewayName:  cfg.GatewayName,
			FileName:     cfg.FileName,
			AssignedIP:   cfg.AssignedIP,
			ExpiresAt:    cfg.ExpiresAt,
			CreatedAt:    cfg.CreatedAt,
			DownloadedAt: cfg.DownloadedAt,
			RevokedAt:    cfg.RevokedAt,
			Status:       status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// handleDownloadWireGuardConfig downloads a WireGuard config file
func (s *Server) handleDownloadWireGuardConfig(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	ctx := c.Request.Context()

	config, err := s.wgConfigStore.GetByID(ctx, configID)
	if err != nil {
		if err == db.ErrWGConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		if err == db.ErrWGConfigRevoked {
			c.JSON(http.StatusGone, gin.H{"error": "config has been revoked"})
			return
		}
		if err == db.ErrWGConfigExpired {
			c.JSON(http.StatusGone, gin.H{"error": "config has expired"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Verify ownership
	if config.UserID != user.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Mark as downloaded
	if err := s.wgConfigStore.MarkDownloaded(ctx, configID); err != nil {
		s.logger.Warn("Failed to mark config as downloaded", zap.Error(err))
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", config.FileName))
	c.Header("Content-Type", "application/x-wireguard-profile")
	c.Data(http.StatusOK, "application/x-wireguard-profile", config.ConfigData)
}

// handleRevokeWireGuardConfig allows a user to revoke their own config
func (s *Server) handleRevokeWireGuardConfig(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	ctx := c.Request.Context()

	config, err := s.wgConfigStore.GetByID(ctx, configID)
	if err != nil {
		if err == db.ErrWGConfigNotFound || err == db.ErrWGConfigRevoked || err == db.ErrWGConfigExpired {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found or already revoked"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Verify ownership
	if config.UserID != user.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := s.wgConfigStore.Revoke(ctx, configID, "user requested"); err != nil {
		s.logger.Error("Failed to revoke config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke config"})
		return
	}

	s.logger.Info("User revoked WireGuard config",
		zap.String("config_id", configID),
		zap.String("user_id", user.UserID))

	c.JSON(http.StatusOK, gin.H{"message": "config revoked"})
}

// WireGuard Admin Config Handlers

// handleAdminListWireGuardConfigs lists all WireGuard configs (admin)
func (s *Server) handleAdminListWireGuardConfigs(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !admin.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	ctx := c.Request.Context()

	// Parse pagination
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	configs, total, err := s.wgConfigStore.ListAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs"})
		return
	}

	type configResponse struct {
		ID           string     `json:"id"`
		UserID       string     `json:"user_id"`
		UserEmail    string     `json:"user_email"`
		UserName     string     `json:"user_name"`
		GatewayID    string     `json:"gateway_id"`
		GatewayName  string     `json:"gateway_name"`
		FileName     string     `json:"file_name"`
		AssignedIP   string     `json:"assigned_ip"`
		ExpiresAt    time.Time  `json:"expires_at"`
		CreatedAt    time.Time  `json:"created_at"`
		DownloadedAt *time.Time `json:"downloaded_at,omitempty"`
		RevokedAt    *time.Time `json:"revoked_at,omitempty"`
		Status       string     `json:"status"`
	}

	response := make([]configResponse, 0, len(configs))
	now := time.Now()
	for _, cfg := range configs {
		status := "active"
		if cfg.RevokedAt != nil {
			status = "revoked"
		} else if now.After(cfg.ExpiresAt) {
			status = "expired"
		}

		response = append(response, configResponse{
			ID:           cfg.ID,
			UserID:       cfg.UserID,
			UserEmail:    cfg.UserEmail,
			UserName:     cfg.UserName,
			GatewayID:    cfg.GatewayID,
			GatewayName:  cfg.GatewayName,
			FileName:     cfg.FileName,
			AssignedIP:   cfg.AssignedIP,
			ExpiresAt:    cfg.ExpiresAt,
			CreatedAt:    cfg.CreatedAt,
			DownloadedAt: cfg.DownloadedAt,
			RevokedAt:    cfg.RevokedAt,
			Status:       status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": response,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// handleAdminRevokeWireGuardConfig revokes a WireGuard config (admin)
func (s *Server) handleAdminRevokeWireGuardConfig(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !admin.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	configID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "admin revoked"
	}
	if req.Reason == "" {
		req.Reason = "admin revoked"
	}

	ctx := c.Request.Context()

	if err := s.wgConfigStore.Revoke(ctx, configID, req.Reason); err != nil {
		if err == db.ErrWGConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		s.logger.Error("Failed to revoke config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke config"})
		return
	}

	s.logger.Info("Admin revoked WireGuard config",
		zap.String("config_id", configID),
		zap.String("reason", req.Reason))

	c.JSON(http.StatusOK, gin.H{"message": "config revoked"})
}

// handleAdminListWireGuardPeers lists all active WireGuard peers (admin)
func (s *Server) handleAdminListWireGuardPeers(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !admin.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	ctx := c.Request.Context()

	peers, err := s.wgPeerStore.GetAllActivePeers(ctx)
	if err != nil {
		s.logger.Error("Failed to list peers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list peers"})
		return
	}

	type peerResponse struct {
		ID            string     `json:"id"`
		GatewayID     string     `json:"gateway_id"`
		GatewayName   string     `json:"gateway_name"`
		UserID        string     `json:"user_id"`
		UserEmail     string     `json:"user_email"`
		UserName      string     `json:"user_name"`
		UserType      string     `json:"user_type"`
		PublicKey     string     `json:"public_key"`
		AssignedIP    string     `json:"assigned_ip"`
		Endpoint      string     `json:"endpoint"`
		LastHandshake *time.Time `json:"last_handshake,omitempty"`
		BytesSent     int64      `json:"bytes_sent"`
		BytesReceived int64      `json:"bytes_received"`
		ConnectedAt   time.Time  `json:"connected_at"`
	}

	response := make([]peerResponse, 0, len(peers))
	for _, peer := range peers {
		response = append(response, peerResponse{
			ID:            peer.ID,
			GatewayID:     peer.GatewayID,
			GatewayName:   peer.GatewayName,
			UserID:        peer.UserID,
			UserEmail:     peer.UserEmail,
			UserName:      peer.UserName,
			UserType:      peer.UserType,
			PublicKey:     peer.PublicKey[:8] + "...", // Truncate for display
			AssignedIP:    peer.AssignedIP,
			Endpoint:      peer.Endpoint,
			LastHandshake: peer.LastHandshake,
			BytesSent:     peer.BytesSent,
			BytesReceived: peer.BytesReceived,
			ConnectedAt:   peer.ConnectedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"peers": response})
}

// WireGuard Gateway Internal Handlers

// handleWireGuardGatewayProvision handles WireGuard gateway provisioning
func (s *Server) handleWireGuardGatewayProvision(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get gateway by token
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Verify gateway type
	if gateway.GatewayType != db.GatewayTypeWireGuard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway is not a WireGuard gateway"})
		return
	}

	// Generate server keypair if not already set
	if gateway.WGPrivateKey == "" || gateway.WGPublicKey == "" {
		keyPair, err := wireguard.GenerateKeyPair()
		if err != nil {
			s.logger.Error("Failed to generate server keypair", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate keys"})
			return
		}

		if err := s.gatewayStore.SetWireGuardKeys(ctx, gateway.ID, keyPair.PrivateKey, keyPair.PublicKey); err != nil {
			s.logger.Error("Failed to save keys", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save keys"})
			return
		}

		gateway.WGPrivateKey = keyPair.PrivateKey
		gateway.WGPublicKey = keyPair.PublicKey
	}

	// Get current authorized peers (active, non-expired, non-revoked configs)
	configs, err := s.wgConfigStore.GetActiveByGateway(ctx, gateway.ID)
	if err != nil {
		s.logger.Error("Failed to get active configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get peers"})
		return
	}

	type peerInfo struct {
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		UserID       string   `json:"user_id"`
		ConfigID     string   `json:"config_id"`
		ExpiresAt    string   `json:"expires_at"`
	}

	peers := make([]peerInfo, 0, len(configs))
	for _, cfg := range configs {
		peers = append(peers, peerInfo{
			PublicKey:    cfg.ClientPublicKey,
			PresharedKey: cfg.PresharedKey,
			AllowedIPs:   []string{cfg.AssignedIP},
			UserID:       cfg.UserID,
			ConfigID:     cfg.ID,
			ExpiresAt:    cfg.ExpiresAt.Format(time.RFC3339),
		})
	}

	// Update gateway status
	publicIP := c.ClientIP()
	if err := s.gatewayStore.UpdateGatewayStatus(ctx, gateway.ID, publicIP); err != nil {
		s.logger.Error("Failed to update gateway status", zap.Error(err))
	}

	s.logger.Info("WireGuard gateway provisioned",
		zap.String("gateway_id", gateway.ID),
		zap.String("gateway_name", gateway.Name),
		zap.Int("peer_count", len(peers)))

	c.JSON(http.StatusOK, gin.H{
		"gateway_id":     gateway.ID,
		"gateway_name":   gateway.Name,
		"private_key":    gateway.WGPrivateKey,
		"public_key":     gateway.WGPublicKey,
		"listen_port":    gateway.WGListenPort,
		"vpn_subnet":     gateway.VPNSubnet,
		"interface_name": "wg0",
		"peers":          peers,
	})
}

// handleWireGuardGatewayHeartbeat handles WireGuard gateway heartbeat
func (s *Server) handleWireGuardGatewayHeartbeat(c *gin.Context) {
	var req struct {
		Token     string `json:"token" binding:"required"`
		PeerCount int    `json:"peer_count"`
		Version   string `json:"version"`
		Uptime    int64  `json:"uptime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Update heartbeat and public IP
	publicIP := c.ClientIP()
	if err := s.gatewayStore.UpdateGatewayStatus(ctx, gateway.ID, publicIP); err != nil {
		s.logger.Error("Failed to update gateway status", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleWireGuardGatewaySyncPeers handles peer sync request from gateway
func (s *Server) handleWireGuardGatewaySyncPeers(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Get current authorized peers
	configs, err := s.wgConfigStore.GetActiveByGateway(ctx, gateway.ID)
	if err != nil {
		s.logger.Error("Failed to get active configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get peers"})
		return
	}

	type peerInfo struct {
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"preshared_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips"`
		UserID       string   `json:"user_id"`
		ConfigID     string   `json:"config_id"`
		ExpiresAt    string   `json:"expires_at"`
	}

	peers := make([]peerInfo, 0, len(configs))
	for _, cfg := range configs {
		peers = append(peers, peerInfo{
			PublicKey:    cfg.ClientPublicKey,
			PresharedKey: cfg.PresharedKey,
			AllowedIPs:   []string{cfg.AssignedIP},
			UserID:       cfg.UserID,
			ConfigID:     cfg.ID,
			ExpiresAt:    cfg.ExpiresAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"peers": peers,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleWireGuardGatewayPeerStats handles peer statistics from gateway
func (s *Server) handleWireGuardGatewayPeerStats(c *gin.Context) {
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

	ctx := c.Request.Context()

	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	// Update peer stats
	for _, peer := range req.Peers {
		var lastHandshake *time.Time
		if peer.LastHandshake > 0 {
			t := time.Unix(peer.LastHandshake, 0)
			lastHandshake = &t
		}

		// Get the config to look up the user
		config, err := s.wgConfigStore.GetByPublicKey(ctx, gateway.ID, peer.PublicKey)
		if err != nil {
			// Peer might have been revoked/expired
			continue
		}

		// Detect user type: SSO if found in users table, otherwise local
		userType := "local"
		if _, ssoErr := s.userStore.GetSSOUser(ctx, config.UserID); ssoErr == nil {
			userType = "sso"
		}

		// Upsert peer record
		peerRecord := &db.WireGuardPeer{
			ID:            uuid.New().String(),
			GatewayID:     gateway.ID,
			ConfigID:      config.ID,
			UserID:        config.UserID,
			UserType:      userType,
			PublicKey:     peer.PublicKey,
			AssignedIP:    config.AssignedIP,
			AllowedIPs:    []string{config.AssignedIP},
			Endpoint:      peer.Endpoint,
			LastHandshake: lastHandshake,
			BytesSent:     peer.BytesSent,
			BytesReceived: peer.BytesReceived,
			ConnectedAt:   time.Now(),
		}

		if err := s.wgPeerStore.Upsert(ctx, peerRecord); err != nil {
			s.logger.Warn("Failed to update peer stats",
				zap.String("public_key", peer.PublicKey[:8]),
				zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// triggerGatewayPeerSync triggers an immediate peer sync on the WireGuard gateway.
// This is called asynchronously after generating a config so the client can connect immediately
// without waiting for the gateway's next poll interval (default 10 seconds).
func (s *Server) triggerGatewayPeerSync(gateway *db.Gateway) {
	// Construct gateway agent URL from hostname or public IP
	var gatewayHost string
	if gateway.Hostname != "" {
		gatewayHost = gateway.Hostname
	} else if gateway.PublicIP != "" {
		gatewayHost = gateway.PublicIP
	} else {
		s.logger.Warn("Cannot trigger peer sync: gateway has no hostname or public IP",
			zap.String("gateway_id", gateway.ID),
			zap.String("gateway_name", gateway.Name))
		return
	}

	// Agent API runs on port 9444 by default
	agentURL := fmt.Sprintf("http://%s:9444", gatewayHost)

	s.logger.Debug("Triggering immediate peer sync on gateway",
		zap.String("gateway_id", gateway.ID),
		zap.String("gateway_name", gateway.Name),
		zap.String("agent_url", agentURL))

	client := agent.NewClient()
	if err := client.TriggerSync(agentURL, gateway.Token); err != nil {
		// Log warning but don't fail - the gateway will pick up the peer on next regular sync
		s.logger.Warn("Failed to trigger immediate peer sync on gateway (will sync on next interval)",
			zap.String("gateway_id", gateway.ID),
			zap.String("gateway_name", gateway.Name),
			zap.Error(err))
		return
	}

	s.logger.Info("Triggered immediate peer sync on gateway",
		zap.String("gateway_id", gateway.ID),
		zap.String("gateway_name", gateway.Name))
}
