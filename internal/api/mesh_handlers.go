package api

import (
	"crypto"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/pki"
)

// ==================== Admin Hub Management ====================

func (s *Server) handleListMeshHubs(c *gin.Context) {
	ctx := c.Request.Context()

	hubs, err := s.meshStore.ListHubs(ctx)
	if err != nil {
		s.logger.Error("Failed to list mesh hubs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list mesh hubs"})
		return
	}

	result := make([]gin.H, 0, len(hubs))
	activeThreshold := 2 * time.Minute
	now := time.Now()

	for _, hub := range hubs {
		isOnline := hub.LastHeartbeat != nil && now.Sub(*hub.LastHeartbeat) < activeThreshold
		status := hub.Status
		if isOnline && status != db.MeshHubStatusError {
			status = db.MeshHubStatusOnline
		} else if !isOnline && status == db.MeshHubStatusOnline {
			status = db.MeshHubStatusOffline
		}

		hubData := gin.H{
			"id":                hub.ID,
			"name":              hub.Name,
			"description":       hub.Description,
			"publicEndpoint":    hub.PublicEndpoint,
			"vpnPort":           hub.VPNPort,
			"vpnProtocol":       hub.VPNProtocol,
			"vpnSubnet":         hub.VPNSubnet,
			"cryptoProfile":     hub.CryptoProfile,
			"tlsAuthEnabled":    hub.TLSAuthEnabled,
			"status":            status,
			"statusMessage":     hub.StatusMessage,
			"connectedSpokes": hub.ConnectedSpokes,
			"connectedClients":  hub.ConnectedClients,
			"createdAt":         hub.CreatedAt.Format(time.RFC3339),
			"updatedAt":         hub.UpdatedAt.Format(time.RFC3339),
		}
		if hub.LastHeartbeat != nil {
			hubData["lastHeartbeat"] = hub.LastHeartbeat.Format(time.RFC3339)
		}
		result = append(result, hubData)
	}

	c.JSON(http.StatusOK, gin.H{"hubs": result})
}

func (s *Server) handleCreateMeshHub(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		PublicEndpoint string `json:"publicEndpoint" binding:"required"`
		VPNPort        int    `json:"vpnPort"`
		VPNProtocol    string `json:"vpnProtocol"`
		VPNSubnet      string `json:"vpnSubnet"`
		CryptoProfile  string `json:"cryptoProfile"`
		TLSAuthEnabled bool   `json:"tlsAuthEnabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate API token for hub
	apiToken, err := db.GenerateMeshToken()
	if err != nil {
		s.logger.Error("Failed to generate API token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API token"})
		return
	}

	// Determine control plane URL from request
	// Check X-Forwarded-Proto header first (for reverse proxy/Istio)
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	controlPlaneURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	hub := &db.MeshHub{
		Name:            req.Name,
		Description:     req.Description,
		PublicEndpoint:  req.PublicEndpoint,
		VPNPort:         req.VPNPort,
		VPNProtocol:     req.VPNProtocol,
		VPNSubnet:       req.VPNSubnet,
		CryptoProfile:   req.CryptoProfile,
		TLSAuthEnabled:  req.TLSAuthEnabled,
		APIToken:        apiToken,
		ControlPlaneURL: controlPlaneURL,
		Status:          db.MeshHubStatusPending,
	}

	if err := s.meshStore.CreateHub(ctx, hub); err != nil {
		if err == db.ErrMeshHubExists {
			c.JSON(http.StatusConflict, gin.H{"error": "hub with this name already exists"})
			return
		}
		s.logger.Error("Failed to create mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mesh hub"})
		return
	}

	// Return the created hub with API token
	c.JSON(http.StatusCreated, gin.H{
		"hub": gin.H{
			"id":              hub.ID,
			"name":            hub.Name,
			"description":     hub.Description,
			"publicEndpoint":  hub.PublicEndpoint,
			"vpnPort":         hub.VPNPort,
			"vpnProtocol":     hub.VPNProtocol,
			"vpnSubnet":       hub.VPNSubnet,
			"cryptoProfile":   hub.CryptoProfile,
			"tlsAuthEnabled":  hub.TLSAuthEnabled,
			"apiToken":        apiToken, // Only shown once at creation
			"controlPlaneUrl": controlPlaneURL,
			"status":          hub.Status,
		},
		"message": "Hub created. Use the install script to set up the hub server.",
	})
}

func (s *Server) handleGetMeshHub(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	hub, err := s.meshStore.GetHub(ctx, hubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hub": gin.H{
			"id":                hub.ID,
			"name":              hub.Name,
			"description":       hub.Description,
			"publicEndpoint":    hub.PublicEndpoint,
			"vpnPort":           hub.VPNPort,
			"vpnProtocol":       hub.VPNProtocol,
			"vpnSubnet":         hub.VPNSubnet,
			"cryptoProfile":     hub.CryptoProfile,
			"tlsAuthEnabled":    hub.TLSAuthEnabled,
			"controlPlaneUrl":   hub.ControlPlaneURL,
			"status":            hub.Status,
			"statusMessage":     hub.StatusMessage,
			"connectedSpokes": hub.ConnectedSpokes,
			"connectedClients":  hub.ConnectedClients,
			"hasCACert":         hub.CACert != "",
			"hasServerCert":     hub.ServerCert != "",
			"createdAt":         hub.CreatedAt.Format(time.RFC3339),
			"updatedAt":         hub.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (s *Server) handleUpdateMeshHub(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	var req struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		PublicEndpoint string `json:"publicEndpoint"`
		VPNPort        int    `json:"vpnPort"`
		VPNProtocol    string `json:"vpnProtocol"`
		VPNSubnet      string `json:"vpnSubnet"`
		CryptoProfile  string `json:"cryptoProfile"`
		TLSAuthEnabled *bool  `json:"tlsAuthEnabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing hub
	hub, err := s.meshStore.GetHub(ctx, hubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	// Update fields
	if req.Name != "" {
		hub.Name = req.Name
	}
	if req.Description != "" {
		hub.Description = req.Description
	}
	if req.PublicEndpoint != "" {
		hub.PublicEndpoint = req.PublicEndpoint
	}
	if req.VPNPort > 0 {
		hub.VPNPort = req.VPNPort
	}
	if req.VPNProtocol != "" {
		hub.VPNProtocol = req.VPNProtocol
	}
	if req.VPNSubnet != "" {
		hub.VPNSubnet = req.VPNSubnet
	}
	if req.CryptoProfile != "" {
		hub.CryptoProfile = req.CryptoProfile
	}
	if req.TLSAuthEnabled != nil {
		hub.TLSAuthEnabled = *req.TLSAuthEnabled
	}

	if err := s.meshStore.UpdateHub(ctx, hub); err != nil {
		if err == db.ErrMeshHubExists {
			c.JSON(http.StatusConflict, gin.H{"error": "hub with this name already exists"})
			return
		}
		s.logger.Error("Failed to update mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update mesh hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "hub updated"})
}

func (s *Server) handleDeleteMeshHub(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	if err := s.meshStore.DeleteHub(ctx, hubID); err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to delete mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete mesh hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "hub deleted"})
}

func (s *Server) handleProvisionMeshHub(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	hub, err := s.meshStore.GetHub(ctx, hubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	// Check if we have a CA
	if s.ca == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PKI not initialized"})
		return
	}

	// Generate mesh CA (separate from main CA for isolation)
	meshCACert, meshCAKey, err := s.ca.GenerateSubCA(fmt.Sprintf("GateKey Mesh CA - %s", hub.Name))
	if err != nil {
		s.logger.Error("Failed to generate mesh CA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mesh CA"})
		return
	}

	// Generate server certificate for hub using the mesh CA
	serverCert, serverKey, err := s.ca.GenerateServerCertWithCA(meshCACert, meshCAKey, hub.Name, []string{
		strings.Split(hub.PublicEndpoint, ":")[0], // Extract hostname from endpoint
	})
	if err != nil {
		s.logger.Error("Failed to generate server certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate server certificate"})
		return
	}

	// Generate DH params placeholder (hub will generate actual DH params)
	dhParams := "# DH parameters will be generated on the hub server\n"

	// Generate TLS-Auth key if enabled
	var tlsAuthKey string
	if hub.TLSAuthEnabled {
		tlsAuthKey, err = generateTLSAuthKey()
		if err != nil {
			s.logger.Error("Failed to generate TLS-Auth key", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TLS-Auth key"})
			return
		}
	}

	// Update hub with PKI
	if err := s.meshStore.UpdateHubPKI(ctx, hubID, meshCACert, meshCAKey, serverCert, serverKey, dhParams, tlsAuthKey); err != nil {
		s.logger.Error("Failed to update hub PKI", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update hub PKI"})
		return
	}

	// Compute config version hash
	configVersion := computeConfigVersion(hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet, hub.CryptoProfile, hub.TLSAuthEnabled)

	c.JSON(http.StatusOK, gin.H{
		"message":       "hub provisioned successfully",
		"configVersion": configVersion,
		"hasCACert":     true,
		"hasServerCert": true,
	})
}

func (s *Server) handleMeshHubInstallScript(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	hub, err := s.meshStore.GetHub(ctx, hubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	// Generate install script
	script := generateMeshHubInstallScript(hub)

	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=install-hub-%s.sh", hub.Name))
	c.String(http.StatusOK, script)
}

// Hub user/group access control

func (s *Server) handleGetMeshHubUsers(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	users, err := s.meshStore.GetHubUsers(ctx, hubID)
	if err != nil {
		s.logger.Error("Failed to get hub users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *Server) handleAssignMeshHubUser(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.meshStore.AssignUserToHub(ctx, hubID, req.UserID); err != nil {
		s.logger.Error("Failed to assign user to hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign user to hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user assigned to hub"})
}

func (s *Server) handleRemoveMeshHubUser(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")
	userID := c.Param("userId")

	if err := s.meshStore.RemoveUserFromHub(ctx, hubID, userID); err != nil {
		s.logger.Error("Failed to remove user from hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user from hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed from hub"})
}

func (s *Server) handleGetMeshHubGroups(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	groups, err := s.meshStore.GetHubGroups(ctx, hubID)
	if err != nil {
		s.logger.Error("Failed to get hub groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (s *Server) handleAssignMeshHubGroup(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	var req struct {
		GroupName string `json:"groupName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.meshStore.AssignGroupToHub(ctx, hubID, req.GroupName); err != nil {
		s.logger.Error("Failed to assign group to hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign group to hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group assigned to hub"})
}

func (s *Server) handleRemoveMeshHubGroup(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")
	groupName := c.Param("groupName")

	if err := s.meshStore.RemoveGroupFromHub(ctx, hubID, groupName); err != nil {
		s.logger.Error("Failed to remove group from hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove group from hub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group removed from hub"})
}

// ==================== Admin Mesh Spoke Management ====================

func (s *Server) handleListMeshSpokes(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hubID)
	if err != nil {
		s.logger.Error("Failed to list mesh spokes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list mesh spokes"})
		return
	}

	result := make([]gin.H, 0, len(spokes))
	activeThreshold := 2 * time.Minute
	now := time.Now()

	for _, gw := range spokes {
		isConnected := gw.LastSeen != nil && now.Sub(*gw.LastSeen) < activeThreshold
		status := gw.Status
		if isConnected && status != db.MeshSpokeStatusError {
			status = db.MeshSpokeStatusConnected
		} else if !isConnected && status == db.MeshSpokeStatusConnected {
			status = db.MeshSpokeStatusDisconnected
		}

		gwData := gin.H{
			"id":            gw.ID,
			"hubId":         gw.HubID,
			"name":          gw.Name,
			"description":   gw.Description,
			"localNetworks": gw.LocalNetworks,
			"tunnelIp":      gw.TunnelIP,
			"status":        status,
			"statusMessage": gw.StatusMessage,
			"bytesSent":     gw.BytesSent,
			"bytesReceived": gw.BytesReceived,
			"remoteIp":      gw.RemoteIP,
			"createdAt":     gw.CreatedAt.Format(time.RFC3339),
			"updatedAt":     gw.UpdatedAt.Format(time.RFC3339),
		}
		if gw.LastSeen != nil {
			gwData["lastSeen"] = gw.LastSeen.Format(time.RFC3339)
		}
		result = append(result, gwData)
	}

	c.JSON(http.StatusOK, gin.H{"spokes": result})
}

func (s *Server) handleCreateMeshSpoke(c *gin.Context) {
	ctx := c.Request.Context()
	hubID := c.Param("id")

	var req struct {
		Name          string   `json:"name" binding:"required"`
		Description   string   `json:"description"`
		LocalNetworks []string `json:"localNetworks"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify hub exists
	_, err := s.meshStore.GetHub(ctx, hubID)
	if err != nil {
		if err == db.ErrMeshHubNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
			return
		}
		s.logger.Error("Failed to get mesh hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hub"})
		return
	}

	// Generate spoke token
	token, err := db.GenerateMeshToken()
	if err != nil {
		s.logger.Error("Failed to generate spoke token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate spoke token"})
		return
	}

	gw := &db.MeshSpoke{
		HubID:         hubID,
		Name:          req.Name,
		Description:   req.Description,
		LocalNetworks: req.LocalNetworks,
		Token:         token,
		Status:        db.MeshSpokeStatusPending,
	}

	if err := s.meshStore.CreateMeshSpoke(ctx, gw); err != nil {
		if err == db.ErrMeshSpokeExists {
			c.JSON(http.StatusConflict, gin.H{"error": "spoke with this name already exists in this hub"})
			return
		}
		s.logger.Error("Failed to create mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mesh spoke"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"spoke": gin.H{
			"id":            gw.ID,
			"hubId":         gw.HubID,
			"name":          gw.Name,
			"description":   gw.Description,
			"localNetworks": gw.LocalNetworks,
			"token":         token, // Only shown once at creation
			"status":        gw.Status,
		},
		"message": "Spoke created. Use the install script to set up the spoke.",
	})
}

func (s *Server) handleGetMeshSpoke(c *gin.Context) {
	ctx := c.Request.Context()
	gwID := c.Param("id")

	gw, err := s.meshStore.GetMeshSpoke(ctx, gwID)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "spoke not found"})
			return
		}
		s.logger.Error("Failed to get mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh spoke"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"spoke": gin.H{
			"id":            gw.ID,
			"hubId":         gw.HubID,
			"name":          gw.Name,
			"description":   gw.Description,
			"localNetworks": gw.LocalNetworks,
			"tunnelIp":      gw.TunnelIP,
			"status":        gw.Status,
			"statusMessage": gw.StatusMessage,
			"bytesSent":     gw.BytesSent,
			"bytesReceived": gw.BytesReceived,
			"remoteIp":      gw.RemoteIP,
			"hasClientCert": gw.ClientCert != "",
			"createdAt":     gw.CreatedAt.Format(time.RFC3339),
			"updatedAt":     gw.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (s *Server) handleUpdateMeshSpoke(c *gin.Context) {
	ctx := c.Request.Context()
	gwID := c.Param("id")

	var req struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		LocalNetworks []string `json:"localNetworks"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gw, err := s.meshStore.GetMeshSpoke(ctx, gwID)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "spoke not found"})
			return
		}
		s.logger.Error("Failed to get mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh spoke"})
		return
	}

	if req.Name != "" {
		gw.Name = req.Name
	}
	if req.Description != "" {
		gw.Description = req.Description
	}
	if req.LocalNetworks != nil {
		gw.LocalNetworks = req.LocalNetworks
	}

	if err := s.meshStore.UpdateMeshSpoke(ctx, gw); err != nil {
		if err == db.ErrMeshSpokeExists {
			c.JSON(http.StatusConflict, gin.H{"error": "spoke with this name already exists in this hub"})
			return
		}
		s.logger.Error("Failed to update mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update mesh spoke"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "spoke updated"})
}

func (s *Server) handleDeleteMeshSpoke(c *gin.Context) {
	ctx := c.Request.Context()
	gwID := c.Param("id")

	if err := s.meshStore.DeleteMeshSpoke(ctx, gwID); err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "spoke not found"})
			return
		}
		s.logger.Error("Failed to delete mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete mesh spoke"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "spoke deleted"})
}

func (s *Server) handleProvisionMeshSpoke(c *gin.Context) {
	ctx := c.Request.Context()
	gwID := c.Param("id")

	gw, err := s.meshStore.GetMeshSpoke(ctx, gwID)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "spoke not found"})
			return
		}
		s.logger.Error("Failed to get mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh spoke"})
		return
	}

	// Get the hub to access its CA
	hub, err := s.meshStore.GetHub(ctx, gw.HubID)
	if err != nil {
		s.logger.Error("Failed to get hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub"})
		return
	}

	if hub.CACert == "" || hub.CAKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub not provisioned - run hub provision first"})
		return
	}

	// Generate client certificate signed by hub's CA
	clientCert, clientKey, err := s.ca.GenerateClientCertWithCA(
		hub.CACert, hub.CAKey,
		fmt.Sprintf("mesh-gateway-%s", gw.Name),
		nil,
	)
	if err != nil {
		s.logger.Error("Failed to generate client certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate client certificate"})
		return
	}

	// Assign tunnel IP (simple sequential allocation based on gateway count)
	// TODO: Implement proper IP allocation from hub's VPN subnet
	tunnelIP := fmt.Sprintf("172.30.0.%d", 10) // Placeholder

	if err := s.meshStore.UpdateMeshSpokePKI(ctx, gwID, clientCert, clientKey, tunnelIP); err != nil {
		s.logger.Error("Failed to update gateway PKI", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update gateway PKI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "spoke provisioned successfully",
		"tunnelIp":      tunnelIP,
		"hasClientCert": true,
	})
}

func (s *Server) handleMeshSpokeInstallScript(c *gin.Context) {
	ctx := c.Request.Context()
	gwID := c.Param("id")

	gw, err := s.meshStore.GetMeshSpoke(ctx, gwID)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "spoke not found"})
			return
		}
		s.logger.Error("Failed to get mesh gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh spoke"})
		return
	}

	hub, err := s.meshStore.GetHub(ctx, gw.HubID)
	if err != nil {
		s.logger.Error("Failed to get hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub"})
		return
	}

	script := generateMeshSpokeInstallScript(gw, hub)

	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=install-spoke-%s.sh", gw.Name))
	c.String(http.StatusOK, script)
}

// ==================== Spoke User/Group Access ====================

func (s *Server) handleGetMeshSpokeUsers(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")

	users, err := s.meshStore.GetSpokeUsers(ctx, spokeID)
	if err != nil {
		s.logger.Error("Failed to get spoke users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get spoke users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *Server) handleAssignMeshSpokeUser(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.meshStore.AddUserToSpoke(ctx, spokeID, req.UserID); err != nil {
		s.logger.Error("Failed to assign user to spoke", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user assigned"})
}

func (s *Server) handleRemoveMeshSpokeUser(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")
	userID := c.Param("userId")

	if err := s.meshStore.RemoveUserFromSpoke(ctx, spokeID, userID); err != nil {
		s.logger.Error("Failed to remove user from spoke", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed"})
}

func (s *Server) handleGetMeshSpokeGroups(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")

	groups, err := s.meshStore.GetSpokeGroups(ctx, spokeID)
	if err != nil {
		s.logger.Error("Failed to get spoke groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get spoke groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (s *Server) handleAssignMeshSpokeGroup(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")

	var req struct {
		GroupName string `json:"groupName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.meshStore.AddGroupToSpoke(ctx, spokeID, req.GroupName); err != nil {
		s.logger.Error("Failed to assign group to spoke", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group assigned"})
}

func (s *Server) handleRemoveMeshSpokeGroup(c *gin.Context) {
	ctx := c.Request.Context()
	spokeID := c.Param("id")
	groupName := c.Param("groupName")

	if err := s.meshStore.RemoveGroupFromSpoke(ctx, spokeID, groupName); err != nil {
		s.logger.Error("Failed to remove group from spoke", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group removed"})
}

// ==================== Hub Internal API (Hub → Control Plane) ====================

func (s *Server) handleMeshHubHeartbeat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token            string `json:"token" binding:"required"`
		Status           string `json:"status"`
		StatusMessage    string `json:"statusMessage"`
		ConnectedSpokes  int    `json:"connectedSpokes"`
		ConnectedClients int    `json:"connectedClients"`
		ConfigVersion    string `json:"configVersion"`
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

	status := req.Status
	if status == "" {
		status = db.MeshHubStatusOnline
	}

	if err := s.meshStore.UpdateHubStatus(ctx, hub.ID, status, req.StatusMessage, req.ConnectedSpokes, req.ConnectedClients); err != nil {
		s.logger.Error("Failed to update hub status", zap.Error(err))
	}

	// Check if config version matches
	expectedVersion := computeConfigVersion(hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet, hub.CryptoProfile, hub.TLSAuthEnabled)
	needsReprovision := req.ConfigVersion != "" && req.ConfigVersion != expectedVersion

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"needsReprovision": needsReprovision,
		"configVersion":    expectedVersion,
	})
}

func (s *Server) handleMeshHubProvisionRequest(c *gin.Context) {
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

	// Auto-provision if hub has no PKI yet
	if hub.CACert == "" || hub.CAKey == "" {
		s.logger.Info("Auto-provisioning hub PKI", zap.String("hub", hub.Name))

		// Check if we have a CA
		if s.ca == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PKI not initialized"})
			return
		}

		// Generate mesh CA (separate from main CA for isolation)
		meshCACert, meshCAKey, err := s.ca.GenerateSubCA(fmt.Sprintf("GateKey Mesh CA - %s", hub.Name))
		if err != nil {
			s.logger.Error("Failed to generate mesh CA", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mesh CA"})
			return
		}

		// Generate server certificate for hub using the mesh CA
		serverCert, serverKey, err := s.ca.GenerateServerCertWithCA(meshCACert, meshCAKey, hub.Name, []string{
			strings.Split(hub.PublicEndpoint, ":")[0], // Extract hostname from endpoint
		})
		if err != nil {
			s.logger.Error("Failed to generate server certificate", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate server certificate"})
			return
		}

		// Generate DH params placeholder (hub will generate actual DH params)
		dhParams := "# DH parameters will be generated on the hub server\n"

		// Generate TLS-Auth key if enabled
		var tlsAuthKey string
		if hub.TLSAuthEnabled {
			tlsAuthKey, err = generateTLSAuthKey()
			if err != nil {
				s.logger.Error("Failed to generate TLS-Auth key", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TLS-Auth key"})
				return
			}
		}

		// Update hub with PKI
		if err := s.meshStore.UpdateHubPKI(ctx, hub.ID, meshCACert, meshCAKey, serverCert, serverKey, dhParams, tlsAuthKey); err != nil {
			s.logger.Error("Failed to update hub PKI", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update hub PKI"})
			return
		}

		// Update local hub reference with new values
		hub.CACert = meshCACert
		hub.CAKey = meshCAKey
		hub.ServerCert = serverCert
		hub.ServerKey = serverKey
		hub.DHParams = dhParams
		hub.TLSAuthKey = tlsAuthKey

		s.logger.Info("Hub auto-provisioned successfully", zap.String("hub", hub.Name))
	}

	// Build full CA chain (Mesh CA + Root CA) for proper verification
	fullCAChain := hub.CACert
	if s.ca != nil {
		rootCACert := string(s.ca.CertificatePEM())
		fullCAChain = hub.CACert + "\n" + rootCACert
	}

	c.JSON(http.StatusOK, gin.H{
		"cacert":         fullCAChain,
		"servercert":     hub.ServerCert,
		"serverkey":      hub.ServerKey,
		"dhparams":       hub.DHParams,
		"tlsauthenabled": hub.TLSAuthEnabled,
		"tlsauthkey":     hub.TLSAuthKey,
		"vpnport":        hub.VPNPort,
		"vpnprotocol":    hub.VPNProtocol,
		"vpnsubnet":      hub.VPNSubnet,
		"cryptoprofile":  hub.CryptoProfile,
		"configversion":  computeConfigVersion(hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet, hub.CryptoProfile, hub.TLSAuthEnabled),
	})
}

func (s *Server) handleMeshHubGetRoutes(c *gin.Context) {
	ctx := c.Request.Context()
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}
	token = strings.TrimPrefix(token, "Bearer ")

	hub, err := s.meshStore.GetHubByToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	routes, err := s.meshStore.GetAllMeshRoutes(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to get mesh routes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get routes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func (s *Server) handleMeshHubGetSpokes(c *gin.Context) {
	ctx := c.Request.Context()
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}
	token = strings.TrimPrefix(token, "Bearer ")

	hub, err := s.meshStore.GetHubByToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	spokes, err := s.meshStore.ListMeshSpokesByHub(ctx, hub.ID)
	if err != nil {
		s.logger.Error("Failed to list spokes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list spokes"})
		return
	}

	result := make([]gin.H, 0, len(spokes))
	for _, gw := range spokes {
		result = append(result, gin.H{
			"id":            gw.ID,
			"name":          gw.Name,
			"localNetworks": gw.LocalNetworks,
			"tunnelIp":      gw.TunnelIP,
			"clientCert":    gw.ClientCert,
			"status":        gw.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"spokes": result})
}

func (s *Server) handleMeshSpokeConnected(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token    string `json:"token" binding:"required"`
		RemoteIP string `json:"remoteIp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Authenticate hub
	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Find the gateway by remote IP or certificate CN
	// For now, just log the event
	s.logger.Info("Mesh gateway connected to hub",
		zap.String("hub", hub.Name),
		zap.String("remoteIp", req.RemoteIP))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleMeshSpokeDisconnected(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token     string `json:"token" binding:"required"`
		SpokeID string `json:"spokeId"`
		RemoteIP  string `json:"remoteIp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	s.logger.Info("Mesh gateway disconnected from hub",
		zap.String("hub", hub.Name),
		zap.String("gatewayId", req.SpokeID),
		zap.String("remoteIp", req.RemoteIP))

	if req.SpokeID != "" {
		_ = s.meshStore.UpdateMeshSpokeStatus(ctx, req.SpokeID, db.MeshSpokeStatusDisconnected, "Disconnected", "", 0, 0)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleMeshClientConnected(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token    string `json:"token" binding:"required"`
		UserID   string `json:"userId"`
		ClientIP string `json:"clientIp"`
		TunnelIP string `json:"tunnelIp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	s.logger.Info("Mesh client connected",
		zap.String("hub", hub.Name),
		zap.String("userId", req.UserID),
		zap.String("tunnelIp", req.TunnelIP))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleMeshClientDisconnected(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token    string `json:"token" binding:"required"`
		UserID   string `json:"userId"`
		TunnelIP string `json:"tunnelIp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hub, err := s.meshStore.GetHubByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	s.logger.Info("Mesh client disconnected",
		zap.String("hub", hub.Name),
		zap.String("userId", req.UserID),
		zap.String("tunnelIp", req.TunnelIP))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ==================== Spoke Internal API (Spoke → Control Plane) ====================

func (s *Server) handleMeshSpokeProvisionRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gw, err := s.meshStore.GetMeshSpokeByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	hub, err := s.meshStore.GetHub(ctx, gw.HubID)
	if err != nil {
		s.logger.Error("Failed to get hub", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub"})
		return
	}

	// Auto-provision hub if it doesn't have PKI yet
	if hub.CACert == "" || hub.CAKey == "" {
		s.logger.Info("Auto-provisioning hub PKI for spoke request", zap.String("hub", hub.Name), zap.String("spoke", gw.Name))

		// Check if we have a CA
		if s.ca == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PKI not initialized"})
			return
		}

		// Generate mesh CA (separate from main CA for isolation)
		meshCACert, meshCAKey, err := s.ca.GenerateSubCA(fmt.Sprintf("GateKey Mesh CA - %s", hub.Name))
		if err != nil {
			s.logger.Error("Failed to generate mesh CA", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mesh CA"})
			return
		}

		// Generate server certificate for hub using the mesh CA
		serverCert, serverKey, err := s.ca.GenerateServerCertWithCA(meshCACert, meshCAKey, hub.Name, []string{
			strings.Split(hub.PublicEndpoint, ":")[0], // Extract hostname from endpoint
		})
		if err != nil {
			s.logger.Error("Failed to generate server certificate", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate server certificate"})
			return
		}

		// Generate DH params placeholder (hub will generate actual DH params)
		dhParams := "# DH parameters will be generated on the hub server\n"

		// Generate TLS-Auth key if enabled
		var tlsAuthKey string
		if hub.TLSAuthEnabled {
			tlsAuthKey, err = generateTLSAuthKey()
			if err != nil {
				s.logger.Error("Failed to generate TLS-Auth key", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TLS-Auth key"})
				return
			}
		}

		// Update hub with PKI
		if err := s.meshStore.UpdateHubPKI(ctx, hub.ID, meshCACert, meshCAKey, serverCert, serverKey, dhParams, tlsAuthKey); err != nil {
			s.logger.Error("Failed to update hub PKI", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update hub PKI"})
			return
		}

		// Update local hub reference with new values
		hub.CACert = meshCACert
		hub.CAKey = meshCAKey
		hub.ServerCert = serverCert
		hub.ServerKey = serverKey
		hub.DHParams = dhParams
		hub.TLSAuthKey = tlsAuthKey

		s.logger.Info("Hub auto-provisioned successfully for spoke", zap.String("hub", hub.Name))
	}

	// Generate client certificate if not already provisioned
	clientCert := gw.ClientCert
	clientKey := gw.ClientKey
	tunnelIP := gw.TunnelIP

	if clientCert == "" || clientKey == "" {
		s.logger.Info("Generating client certificate for spoke", zap.String("spoke", gw.Name))

		// Generate client certificate signed by hub's CA
		cert, key, err := s.ca.GenerateClientCertWithCA(
			hub.CACert, hub.CAKey,
			fmt.Sprintf("mesh-gateway-%s", gw.Name),
			nil,
		)
		if err != nil {
			s.logger.Error("Failed to generate spoke client cert", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate certificate"})
			return
		}
		clientCert = cert
		clientKey = key

		// Allocate tunnel IP if not already assigned (simple placeholder for now)
		if tunnelIP == "" {
			tunnelIP = fmt.Sprintf("172.30.0.%d", 10)
		}

		// Save the generated certificate and tunnel IP
		if err := s.meshStore.UpdateMeshSpokePKI(ctx, gw.ID, clientCert, clientKey, tunnelIP); err != nil {
			s.logger.Error("Failed to save spoke provision", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save provision"})
			return
		}

		// Update status to connected
		if err := s.meshStore.UpdateMeshSpokeStatus(ctx, gw.ID, db.MeshSpokeStatusConnected, "", "", 0, 0); err != nil {
			s.logger.Warn("Failed to update spoke status", zap.Error(err))
		}
	}

	// Build full CA chain (Mesh CA + Root CA) for proper verification
	fullCAChain := hub.CACert
	if s.ca != nil {
		rootCACert := string(s.ca.CertificatePEM())
		fullCAChain = hub.CACert + "\n" + rootCACert
	}

	c.JSON(http.StatusOK, gin.H{
		"gatewayId":      gw.ID,
		"hubEndpoint":    hub.PublicEndpoint,
		"hubVpnPort":     hub.VPNPort,
		"hubVpnProtocol": hub.VPNProtocol,
		"caCert":         fullCAChain,
		"clientCert":     clientCert,
		"clientKey":      clientKey,
		"tunnelIp":       tunnelIP,
		"localNetworks":  gw.LocalNetworks,
		"tlsAuthEnabled": hub.TLSAuthEnabled,
		"tlsAuthKey":     hub.TLSAuthKey,
		"cryptoProfile":  hub.CryptoProfile,
	})
}

func (s *Server) handleMeshSpokeHeartbeat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token         string `json:"token" binding:"required"`
		Status        string `json:"status"`
		StatusMessage string `json:"statusMessage"`
		RemoteIP      string `json:"remoteIp"`
		BytesSent     int64  `json:"bytesSent"`
		BytesReceived int64  `json:"bytesReceived"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gw, err := s.meshStore.GetMeshSpokeByToken(ctx, req.Token)
	if err != nil {
		if err == db.ErrMeshSpokeNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		s.logger.Error("Failed to get gateway by token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		return
	}

	status := req.Status
	if status == "" {
		status = db.MeshSpokeStatusConnected
	}

	if err := s.meshStore.UpdateMeshSpokeStatus(ctx, gw.ID, status, req.StatusMessage, req.RemoteIP, req.BytesSent, req.BytesReceived); err != nil {
		s.logger.Error("Failed to update gateway status", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ==================== Helper Functions ====================

func computeConfigVersion(vpnPort int, vpnProtocol, vpnSubnet, cryptoProfile string, tlsAuthEnabled bool) string {
	data := fmt.Sprintf("%d|%s|%s|%s|%v", vpnPort, vpnProtocol, vpnSubnet, cryptoProfile, tlsAuthEnabled)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func generateTLSAuthKey() (string, error) {
	// Generate a 2048-bit key for TLS-Auth
	key := make([]byte, 256)
	if _, err := cryptoRand.Read(key); err != nil {
		return "", err
	}

	// Format as OpenVPN TLS-Auth key
	var sb strings.Builder
	sb.WriteString("#\n# 2048 bit OpenVPN static key\n#\n")
	sb.WriteString("-----BEGIN OpenVPN Static key V1-----\n")
	for i := 0; i < len(key); i += 16 {
		end := i + 16
		if end > len(key) {
			end = len(key)
		}
		sb.WriteString(hex.EncodeToString(key[i:end]))
		sb.WriteString("\n")
	}
	sb.WriteString("-----END OpenVPN Static key V1-----\n")

	return sb.String(), nil
}

func generateMeshHubInstallScript(hub *db.MeshHub) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# GateKey Mesh Hub Install Script
# Hub: %s
# Generated: %s

echo "Installing GateKey Mesh Hub..."

# Configuration
HUB_NAME="%s"
CONTROL_PLANE_URL="%s"
API_TOKEN="%s"
VPN_PORT=%d
VPN_PROTOCOL="%s"

# Create directories
mkdir -p /etc/gatekey-hub
mkdir -p /var/log/gatekey-hub

# Download the hub binary
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

echo "Downloading gatekey-hub binary..."
curl -sSL "${CONTROL_PLANE_URL}/downloads/gatekey-hub-linux-${ARCH}" -o /usr/local/bin/gatekey-hub
chmod +x /usr/local/bin/gatekey-hub

# Create configuration file
cat > /etc/gatekey-hub/config.yaml << EOF
name: ${HUB_NAME}
control_plane_url: ${CONTROL_PLANE_URL}
api_token: ${API_TOKEN}
vpn_port: ${VPN_PORT}
vpn_protocol: ${VPN_PROTOCOL}
log_level: info
EOF

# Create systemd service
cat > /etc/systemd/system/gatekey-hub.service << EOF
[Unit]
Description=GateKey Mesh Hub
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gatekey-hub run --config /etc/gatekey-hub/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the service
systemctl daemon-reload
systemctl enable gatekey-hub
systemctl start gatekey-hub

echo ""
echo "GateKey Mesh Hub installed successfully!"
echo "Hub Name: ${HUB_NAME}"
echo "VPN Port: ${VPN_PORT}/${VPN_PROTOCOL}"
echo ""
echo "Check status: systemctl status gatekey-hub"
echo "View logs: journalctl -u gatekey-hub -f"
`,
		hub.Name,
		time.Now().Format(time.RFC3339),
		hub.Name,
		hub.ControlPlaneURL,
		hub.APIToken,
		hub.VPNPort,
		hub.VPNProtocol,
	)
}

func generateMeshSpokeInstallScript(gw *db.MeshSpoke, hub *db.MeshHub) string {
	localNetworks := strings.Join(gw.LocalNetworks, ",")
	return fmt.Sprintf(`#!/bin/bash
set -e

# GateKey Mesh Spoke Install Script
# Spoke: %s
# Hub: %s
# Generated: %s

echo "Installing GateKey Mesh Spoke..."

# Configuration
SPOKE_NAME="%s"
CONTROL_PLANE_URL="%s"
SPOKE_TOKEN="%s"
HUB_ENDPOINT="%s"
LOCAL_NETWORKS="%s"

# Create directories
mkdir -p /etc/gatekey-spoke
mkdir -p /var/log/gatekey-spoke

# Download the mesh spoke binary
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

echo "Downloading gatekey-mesh-spoke binary..."
curl -sSL "${CONTROL_PLANE_URL}/downloads/gatekey-mesh-spoke-linux-${ARCH}" -o /usr/local/bin/gatekey-mesh-spoke
chmod +x /usr/local/bin/gatekey-mesh-spoke

# Create configuration file
cat > /etc/gatekey-spoke/config.yaml << EOF
name: ${SPOKE_NAME}
control_plane_url: ${CONTROL_PLANE_URL}
spoke_token: ${SPOKE_TOKEN}
hub_endpoint: ${HUB_ENDPOINT}
local_networks:
EOF

# Add local networks
IFS=',' read -ra NETWORKS <<< "${LOCAL_NETWORKS}"
for net in "${NETWORKS[@]}"; do
    echo "  - ${net}" >> /etc/gatekey-spoke/config.yaml
done

cat >> /etc/gatekey-spoke/config.yaml << EOF
log_level: info
EOF

# Create systemd service
cat > /etc/systemd/system/gatekey-mesh-spoke.service << EOF
[Unit]
Description=GateKey Mesh Spoke
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gatekey-mesh-spoke run --config /etc/gatekey-spoke/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the service
systemctl daemon-reload
systemctl enable gatekey-mesh-spoke
systemctl start gatekey-mesh-spoke

echo ""
echo "GateKey Mesh Spoke installed successfully!"
echo "Spoke Name: ${SPOKE_NAME}"
echo "Hub Endpoint: ${HUB_ENDPOINT}"
echo "Local Networks: ${LOCAL_NETWORKS}"
echo ""
echo "Check status: systemctl status gatekey-mesh-spoke"
echo "View logs: journalctl -u gatekey-mesh-spoke -f"
`,
		gw.Name,
		hub.Name,
		time.Now().Format(time.RFC3339),
		gw.Name,
		hub.ControlPlaneURL,
		gw.Token,
		hub.PublicEndpoint,
		localNetworks,
	)
}

// ==================== User Mesh Hub API (Client Access) ====================

// handleListUserMeshHubs lists mesh hubs the authenticated user has access to
func (s *Server) handleListUserMeshHubs(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	hubs, err := s.meshStore.GetHubsForUser(ctx, user.UserID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to get mesh hubs for user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get mesh hubs"})
		return
	}

	result := make([]gin.H, 0, len(hubs))
	for _, hub := range hubs {
		// Only show online hubs to users
		if hub.Status != db.MeshHubStatusOnline {
			continue
		}
		result = append(result, gin.H{
			"id":              hub.ID,
			"name":            hub.Name,
			"description":     hub.Description,
			"status":          hub.Status,
			"connectedspokes": hub.ConnectedSpokes,
		})
	}

	c.JSON(http.StatusOK, gin.H{"hubs": result})
}

// handleGenerateMeshClientConfig generates a VPN config for connecting to a mesh hub
func (s *Server) handleGenerateMeshClientConfig(c *gin.Context) {
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

	// Check if hub is online
	if hub.Status != db.MeshHubStatusOnline {
		c.JSON(http.StatusForbidden, gin.H{"error": "mesh hub is not online"})
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

	// Check if hub has PKI set up
	if hub.CACert == "" || hub.CAKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mesh hub PKI not configured"})
		return
	}

	// Issue client certificate using hub's CA
	certValidity := 24 * time.Hour
	if s.config.PKI.CertValidity > 0 {
		certValidity = s.config.PKI.CertValidity
	}

	clientCert, clientKey, err := issueClientCertFromPEM(hub.CACert, hub.CAKey, user.Email, certValidity)
	if err != nil {
		s.logger.Error("Failed to issue mesh client certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate certificate"})
		return
	}

	// Get routes the user can access
	routes, err := s.meshStore.GetUserMeshRoutes(ctx, hub.ID, user.UserID, user.Groups)
	if err != nil {
		s.logger.Warn("Failed to get user mesh routes", zap.Error(err))
		// Continue without routes - not a fatal error
	}

	// Build full CA chain (Mesh CA + Root CA) for proper TLS verification
	// The hub's server cert is signed by Mesh CA, which is signed by Root CA
	fullCAChain := hub.CACert
	if s.ca != nil {
		rootCACert := string(s.ca.CertificatePEM())
		if !strings.HasSuffix(fullCAChain, "\n") {
			fullCAChain += "\n"
		}
		fullCAChain += rootCACert
	}

	// Generate OpenVPN config
	config := generateMeshClientOVPNConfig(hub, fullCAChain, clientCert, clientKey, routes)

	c.JSON(http.StatusOK, gin.H{
		"hubname": hub.Name,
		"config":  config,
	})
}

// issueClientCertFromPEM issues a client certificate using the provided CA cert and key PEM strings
func issueClientCertFromPEM(caCertPEM, caKeyPEM, commonName string, validity time.Duration) (certPEM, keyPEM string, err error) {
	// Parse CA certificate
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse CA private key
	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA private key PEM")
	}

	var caKey crypto.Signer
	switch caKeyBlock.Type {
	case "RSA PRIVATE KEY":
		caKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	case "EC PRIVATE KEY":
		caKey, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
	case "PRIVATE KEY":
		parsedKey, parseErr := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
		if parseErr != nil {
			return "", "", fmt.Errorf("failed to parse PKCS8 private key: %w", parseErr)
		}
		var ok bool
		caKey, ok = parsedKey.(crypto.Signer)
		if !ok {
			return "", "", fmt.Errorf("private key is not a signer")
		}
	default:
		return "", "", fmt.Errorf("unsupported private key type: %s", caKeyBlock.Type)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA private key: %w", err)
	}

	// Generate client private key (ECDSA P-256)
	clientKey, err := pki.GenerateECDSAKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client key: %w", err)
	}

	// Generate serial number
	serialNumber, err := cryptoRand.Int(cryptoRand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		EmailAddresses:        []string{commonName},
		NotBefore:             now.Add(-5 * time.Minute), // Allow for clock skew
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(cryptoRand.Reader, template, caCert, clientKey.Public(), caKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	}))

	return certPEM, keyPEM, nil
}

// generateMeshClientOVPNConfig generates an OpenVPN client config for mesh hub access
func generateMeshClientOVPNConfig(hub *db.MeshHub, caChain, clientCert, clientKey string, routes []string) string {
	var sb strings.Builder

	sb.WriteString("# GateKey Mesh VPN Configuration\n")
	sb.WriteString(fmt.Sprintf("# Hub: %s\n", hub.Name))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("client\n")
	sb.WriteString("dev tun\n")
	sb.WriteString(fmt.Sprintf("proto %s\n", hub.VPNProtocol))
	sb.WriteString(fmt.Sprintf("remote %s %d\n", hub.PublicEndpoint, hub.VPNPort))
	sb.WriteString("\n")
	sb.WriteString("resolv-retry infinite\n")
	sb.WriteString("nobind\n")
	sb.WriteString("persist-key\n")
	sb.WriteString("persist-tun\n")
	sb.WriteString("\n")

	// Crypto settings based on profile
	switch hub.CryptoProfile {
	case "fips":
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("auth SHA384\n")
		sb.WriteString("tls-cipher TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384\n")
	case "modern":
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("auth SHA256\n")
	default:
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("auth SHA256\n")
	}
	sb.WriteString("tls-version-min 1.2\n")
	sb.WriteString("\n")

	sb.WriteString("remote-cert-tls server\n")
	sb.WriteString("\n")

	// Add routes for spoke networks
	if len(routes) > 0 {
		sb.WriteString("# Routes to spoke networks\n")
		for _, route := range routes {
			netIP, netmask, err := cidrToNetmask(route)
			if err == nil {
				sb.WriteString(fmt.Sprintf("route %s %s\n", netIP, netmask))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("verb 3\n")
	sb.WriteString("\n")

	// Inline certificates (full CA chain: Mesh CA + Root CA)
	sb.WriteString("<ca>\n")
	sb.WriteString(caChain)
	if !strings.HasSuffix(caChain, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("</ca>\n\n")

	sb.WriteString("<cert>\n")
	sb.WriteString(clientCert)
	if !strings.HasSuffix(clientCert, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("</cert>\n\n")

	sb.WriteString("<key>\n")
	sb.WriteString(clientKey)
	if !strings.HasSuffix(clientKey, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("</key>\n")

	// Add TLS auth if enabled
	if hub.TLSAuthEnabled && hub.TLSAuthKey != "" {
		sb.WriteString("\n<tls-auth>\n")
		sb.WriteString(hub.TLSAuthKey)
		if !strings.HasSuffix(hub.TLSAuthKey, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("</tls-auth>\n")
		sb.WriteString("key-direction 1\n")
	}

	return sb.String()
}
