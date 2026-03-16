package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// ==================== User JIT Endpoints ====================

// handleListJITResources returns resources the user does NOT currently have access to
func (s *Server) handleListJITResources(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Get all active access rules
	allRules, err := s.accessRuleStore.ListAccessRules(ctx)
	if err != nil {
		s.logger.Error("Failed to list access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list resources"})
		return
	}

	// Get user's current access rules
	userRules, err := s.accessRuleStore.GetUserAccessRules(ctx, user.UserID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to get user access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user access"})
		return
	}

	// Build set of user's current rule IDs
	userRuleIDs := make(map[string]bool)
	for _, r := range userRules {
		userRuleIDs[r.ID] = true
	}

	type resource struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	var resources []resource

	// Add access rules the user doesn't have
	for _, r := range allRules {
		if r.IsActive && !userRuleIDs[r.ID] {
			resources = append(resources, resource{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				Type:        "access_rule",
			})
		}
	}

	// Get all active proxy apps
	allApps, err := s.proxyAppStore.ListActiveProxyApplications(ctx)
	if err != nil {
		s.logger.Error("Failed to list proxy apps", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list resources"})
		return
	}

	// Get user's current proxy apps
	userApps, err := s.proxyAppStore.GetUserProxyApplications(ctx, user.UserID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to get user proxy apps", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user access"})
		return
	}

	userAppIDs := make(map[string]bool)
	for _, a := range userApps {
		userAppIDs[a.ID] = true
	}

	for _, a := range allApps {
		if !userAppIDs[a.ID] {
			resources = append(resources, resource{
				ID:          a.ID,
				Name:        a.Name,
				Description: a.Description,
				Type:        "proxy_app",
			})
		}
	}

	// Get all mesh hubs
	allHubs, err := s.meshStore.ListHubs(ctx)
	if err != nil {
		s.logger.Error("Failed to list mesh hubs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list resources"})
		return
	}

	// Get user's current mesh hubs
	userHubs, err := s.meshStore.GetHubsForUser(ctx, user.UserID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to get user mesh hubs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user access"})
		return
	}

	userHubIDs := make(map[string]bool)
	for _, h := range userHubs {
		userHubIDs[h.ID] = true
	}

	for _, h := range allHubs {
		if !userHubIDs[h.ID] {
			resources = append(resources, resource{
				ID:          h.ID,
				Name:        h.Name,
				Description: h.Description,
				Type:        "mesh_hub",
			})
		}
	}

	if resources == nil {
		resources = []resource{}
	}

	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

// handleCreateJITRequest creates a new JIT access request
func (s *Server) handleCreateJITRequest(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var body struct {
		ResourceType    string `json:"resource_type" binding:"required"`
		ResourceID      string `json:"resource_id" binding:"required"`
		Justification   string `json:"justification" binding:"required"`
		DurationMinutes int    `json:"duration_minutes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Validate duration
	if body.DurationMinutes < 15 || body.DurationMinutes > 480 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duration must be between 15 and 480 minutes"})
		return
	}

	// Validate resource type
	validTypes := map[string]bool{"access_rule": true, "network": true, "gateway": true, "proxy_app": true, "mesh_hub": true}
	if !validTypes[body.ResourceType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource type"})
		return
	}

	// Validate resource exists and get its name
	var resourceName string
	switch body.ResourceType {
	case "access_rule":
		rule, err := s.accessRuleStore.GetAccessRule(ctx, body.ResourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "access rule not found"})
			return
		}
		resourceName = rule.Name
		// Check if user already has access
		userRules, err := s.accessRuleStore.GetUserAccessRules(ctx, user.UserID, user.Groups)
		if err == nil {
			for _, r := range userRules {
				if r.ID == body.ResourceID {
					c.JSON(http.StatusConflict, gin.H{"error": "you already have access to this resource"})
					return
				}
			}
		}
	case "proxy_app":
		app, err := s.proxyAppStore.GetProxyApplication(ctx, body.ResourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "proxy application not found"})
			return
		}
		resourceName = app.Name
		userApps, err := s.proxyAppStore.GetUserProxyApplications(ctx, user.UserID, user.Groups)
		if err == nil {
			for _, a := range userApps {
				if a.ID == body.ResourceID {
					c.JSON(http.StatusConflict, gin.H{"error": "you already have access to this resource"})
					return
				}
			}
		}
	case "mesh_hub":
		hub, err := s.meshStore.GetHub(ctx, body.ResourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mesh hub not found"})
			return
		}
		resourceName = hub.Name
		userHubs, err := s.meshStore.GetHubsForUser(ctx, user.UserID, user.Groups)
		if err == nil {
			for _, h := range userHubs {
				if h.ID == body.ResourceID {
					c.JSON(http.StatusConflict, gin.H{"error": "you already have access to this resource"})
					return
				}
			}
		}
	case "network":
		net, err := s.networkStore.GetNetwork(ctx, body.ResourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "network not found"})
			return
		}
		resourceName = net.Name
	case "gateway":
		gw, err := s.gatewayStore.GetGateway(ctx, body.ResourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gateway not found"})
			return
		}
		resourceName = gw.Name
	}

	// Check if user already has an active grant for this resource
	existingGrant, err := s.jitStore.GetActiveGrantForResource(ctx, user.UserID, body.ResourceType, body.ResourceID)
	if err != nil {
		s.logger.Error("Failed to check existing grants", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing access"})
		return
	}
	if existingGrant != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "you already have an active JIT grant for this resource"})
		return
	}

	req := &db.JITAccessRequest{
		RequesterID:          user.UserID,
		RequesterEmail:       user.Email,
		ResourceType:         body.ResourceType,
		ResourceID:           body.ResourceID,
		ResourceName:         resourceName,
		Justification:        body.Justification,
		RequestedDurationMin: body.DurationMinutes,
		Status:               "pending",
		ExpiresAt:            time.Now().Add(60 * time.Minute),
	}

	if err := s.jitStore.CreateRequest(ctx, req); err != nil {
		s.logger.Error("Failed to create JIT request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"request": gin.H{
			"id":               req.ID,
			"resource_type":    req.ResourceType,
			"resource_id":      req.ResourceID,
			"resource_name":    req.ResourceName,
			"justification":    req.Justification,
			"duration_minutes": req.RequestedDurationMin,
			"status":           req.Status,
			"expires_at":       req.ExpiresAt.Format(time.RFC3339),
			"created_at":       req.CreatedAt.Format(time.RFC3339),
		},
	})
}

// handleListMyJITRequests lists the authenticated user's JIT requests
func (s *Server) handleListMyJITRequests(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	requests, err := s.jitStore.ListUserRequests(ctx, user.UserID)
	if err != nil {
		s.logger.Error("Failed to list user JIT requests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list requests"})
		return
	}

	result := make([]gin.H, 0, len(requests))
	for _, r := range requests {
		item := gin.H{
			"id":               r.ID,
			"resource_type":    r.ResourceType,
			"resource_id":      r.ResourceID,
			"resource_name":    r.ResourceName,
			"justification":    r.Justification,
			"duration_minutes": r.RequestedDurationMin,
			"status":           r.Status,
			"expires_at":       r.ExpiresAt.Format(time.RFC3339),
			"created_at":       r.CreatedAt.Format(time.RFC3339),
		}
		if r.ApproverEmail != nil {
			item["approver_email"] = *r.ApproverEmail
		}
		if r.ApprovalNote != nil {
			item["approval_note"] = *r.ApprovalNote
		}
		if r.DecidedAt != nil {
			item["decided_at"] = r.DecidedAt.Format(time.RFC3339)
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"requests": result})
}

// handleCancelJITRequest cancels a pending JIT request
func (s *Server) handleCancelJITRequest(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Verify the request belongs to this user
	req, err := s.jitStore.GetRequest(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	if req.RequesterID != user.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only cancel your own requests"})
		return
	}

	if err := s.jitStore.CancelRequest(ctx, id); err != nil {
		s.logger.Error("Failed to cancel JIT request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request canceled"})
}

// handleListMyJITGrants lists active JIT grants for the authenticated user
func (s *Server) handleListMyJITGrants(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	grants, err := s.jitStore.GetActiveGrantsForUser(ctx, user.UserID)
	if err != nil {
		s.logger.Error("Failed to list user JIT grants", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list grants"})
		return
	}

	result := make([]gin.H, 0, len(grants))
	for _, g := range grants {
		result = append(result, gin.H{
			"id":            g.ID,
			"request_id":    g.RequestID,
			"resource_type": g.ResourceType,
			"resource_id":   g.ResourceID,
			"is_active":     g.IsActive,
			"granted_at":    g.GrantedAt.Format(time.RFC3339),
			"expires_at":    g.ExpiresAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"grants": result})
}

// ==================== Admin JIT Endpoints ====================

// handleListAllJITRequests lists all JIT requests (admin)
func (s *Server) handleListAllJITRequests(c *gin.Context) {
	ctx := c.Request.Context()

	status := c.Query("status")
	limit := 100
	offset := 0

	requests, err := s.jitStore.ListRequests(ctx, status, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list JIT requests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list requests"})
		return
	}

	result := make([]gin.H, 0, len(requests))
	for _, r := range requests {
		item := gin.H{
			"id":               r.ID,
			"requester_id":     r.RequesterID,
			"requester_email":  r.RequesterEmail,
			"resource_type":    r.ResourceType,
			"resource_id":      r.ResourceID,
			"resource_name":    r.ResourceName,
			"justification":    r.Justification,
			"duration_minutes": r.RequestedDurationMin,
			"status":           r.Status,
			"expires_at":       r.ExpiresAt.Format(time.RFC3339),
			"created_at":       r.CreatedAt.Format(time.RFC3339),
		}
		if r.ApproverID != nil {
			item["approver_id"] = *r.ApproverID
		}
		if r.ApproverEmail != nil {
			item["approver_email"] = *r.ApproverEmail
		}
		if r.ApprovalNote != nil {
			item["approval_note"] = *r.ApprovalNote
		}
		if r.DecidedAt != nil {
			item["decided_at"] = r.DecidedAt.Format(time.RFC3339)
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"requests": result})
}

// handleApproveJITRequest approves a JIT request (admin)
func (s *Server) handleApproveJITRequest(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)

	grant, err := s.jitStore.ApproveRequest(ctx, id, user.UserID, user.Email, body.Note)
	if err != nil {
		s.logger.Error("Failed to approve JIT request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "request approved",
		"grant": gin.H{
			"id":            grant.ID,
			"request_id":    grant.RequestID,
			"user_id":       grant.UserID,
			"resource_type": grant.ResourceType,
			"resource_id":   grant.ResourceID,
			"is_active":     grant.IsActive,
			"granted_at":    grant.GrantedAt.Format(time.RFC3339),
			"expires_at":    grant.ExpiresAt.Format(time.RFC3339),
		},
	})
}

// handleDenyJITRequest denies a JIT request (admin)
func (s *Server) handleDenyJITRequest(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := s.jitStore.DenyRequest(ctx, id, user.UserID, user.Email, body.Note); err != nil {
		s.logger.Error("Failed to deny JIT request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request denied"})
}

// handleRevokeJITGrant revokes an active JIT grant (admin)
func (s *Server) handleRevokeJITGrant(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	reason := body.Reason
	if reason == "" {
		reason = "revoked by admin"
	}

	if err := s.jitStore.RevokeGrant(ctx, id, user.Email, reason); err != nil {
		s.logger.Error("Failed to revoke JIT grant", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "grant revoked"})
}

// handleGetJITStats returns JIT access statistics (admin)
func (s *Server) handleGetJITStats(c *gin.Context) {
	ctx := c.Request.Context()

	pending, active, err := s.jitStore.GetGrantStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get JIT stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_requests": pending,
		"active_grants":    active,
	})
}
