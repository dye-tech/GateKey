package api

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Slug validation regex - alphanumeric with dashes, lowercase
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// ---- Admin Handlers ----

// handleListProxyApps returns all proxy applications (admin only)
func (s *Server) handleListProxyApps(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	apps, err := s.proxyAppStore.ListProxyApplications(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to list proxy applications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": apps})
}

// handleCreateProxyApp creates a new proxy application
func (s *Server) handleCreateProxyApp(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req struct {
		Name               string            `json:"name" binding:"required"`
		Slug               string            `json:"slug" binding:"required"`
		Description        string            `json:"description"`
		InternalURL        string            `json:"internal_url" binding:"required"`
		IconURL            *string           `json:"icon_url"`
		IsActive           *bool             `json:"is_active"`
		PreserveHostHeader *bool             `json:"preserve_host_header"`
		StripPrefix        *bool             `json:"strip_prefix"`
		InjectHeaders      map[string]string `json:"inject_headers"`
		AllowedHeaders     []string          `json:"allowed_headers"`
		WebsocketEnabled   *bool             `json:"websocket_enabled"`
		TimeoutSeconds     *int              `json:"timeout_seconds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate slug format
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRegex.MatchString(slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug must be lowercase alphanumeric with dashes, 1-100 characters"})
		return
	}

	// Reserved slugs
	reservedSlugs := []string{"api", "auth", "admin", "health", "metrics", "scripts", "downloads"}
	for _, reserved := range reservedSlugs {
		if slug == reserved {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slug '" + slug + "' is reserved"})
			return
		}
	}

	// Validate internal URL
	internalURL := strings.TrimSpace(req.InternalURL)
	if !strings.HasPrefix(internalURL, "http://") && !strings.HasPrefix(internalURL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal_url must start with http:// or https://"})
		return
	}

	// Set defaults
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	preserveHostHeader := false
	if req.PreserveHostHeader != nil {
		preserveHostHeader = *req.PreserveHostHeader
	}
	stripPrefix := true
	if req.StripPrefix != nil {
		stripPrefix = *req.StripPrefix
	}
	websocketEnabled := true
	if req.WebsocketEnabled != nil {
		websocketEnabled = *req.WebsocketEnabled
	}
	timeoutSeconds := 30
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		timeoutSeconds = *req.TimeoutSeconds
	}
	if req.InjectHeaders == nil {
		req.InjectHeaders = make(map[string]string)
	}
	if req.AllowedHeaders == nil {
		req.AllowedHeaders = []string{"*"}
	}

	app := &db.ProxyApplication{
		Name:               strings.TrimSpace(req.Name),
		Slug:               slug,
		Description:        strings.TrimSpace(req.Description),
		InternalURL:        internalURL,
		IconURL:            req.IconURL,
		IsActive:           isActive,
		PreserveHostHeader: preserveHostHeader,
		StripPrefix:        stripPrefix,
		InjectHeaders:      req.InjectHeaders,
		AllowedHeaders:     req.AllowedHeaders,
		WebsocketEnabled:   websocketEnabled,
		TimeoutSeconds:     timeoutSeconds,
	}

	if err := s.proxyAppStore.CreateProxyApplication(c.Request.Context(), app); err != nil {
		if err == db.ErrProxyAppExists {
			c.JSON(http.StatusConflict, gin.H{"error": "application with this slug already exists"})
			return
		}
		s.logger.Error("Failed to create proxy application", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create application"})
		return
	}

	s.logger.Info("Proxy application created",
		zap.String("id", app.ID),
		zap.String("slug", app.Slug),
		zap.String("admin", user.Email))

	c.JSON(http.StatusCreated, app)
}

// handleGetProxyApp returns a single proxy application
func (s *Server) handleGetProxyApp(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	app, err := s.proxyAppStore.GetProxyApplication(c.Request.Context(), id)
	if err != nil {
		if err == db.ErrProxyAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
			return
		}
		s.logger.Error("Failed to get proxy application", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get application"})
		return
	}

	c.JSON(http.StatusOK, app)
}

// handleUpdateProxyApp updates a proxy application
func (s *Server) handleUpdateProxyApp(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")

	// Get existing app
	app, err := s.proxyAppStore.GetProxyApplication(c.Request.Context(), id)
	if err != nil {
		if err == db.ErrProxyAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
			return
		}
		s.logger.Error("Failed to get proxy application", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get application"})
		return
	}

	var req struct {
		Name               *string           `json:"name"`
		Slug               *string           `json:"slug"`
		Description        *string           `json:"description"`
		InternalURL        *string           `json:"internal_url"`
		IconURL            *string           `json:"icon_url"`
		IsActive           *bool             `json:"is_active"`
		PreserveHostHeader *bool             `json:"preserve_host_header"`
		StripPrefix        *bool             `json:"strip_prefix"`
		InjectHeaders      map[string]string `json:"inject_headers"`
		AllowedHeaders     []string          `json:"allowed_headers"`
		WebsocketEnabled   *bool             `json:"websocket_enabled"`
		TimeoutSeconds     *int              `json:"timeout_seconds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields if provided
	if req.Name != nil {
		app.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		if !slugRegex.MatchString(slug) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slug must be lowercase alphanumeric with dashes"})
			return
		}
		app.Slug = slug
	}
	if req.Description != nil {
		app.Description = *req.Description
	}
	if req.InternalURL != nil {
		internalURL := strings.TrimSpace(*req.InternalURL)
		if !strings.HasPrefix(internalURL, "http://") && !strings.HasPrefix(internalURL, "https://") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "internal_url must start with http:// or https://"})
			return
		}
		app.InternalURL = internalURL
	}
	if req.IconURL != nil {
		app.IconURL = req.IconURL
	}
	if req.IsActive != nil {
		app.IsActive = *req.IsActive
	}
	if req.PreserveHostHeader != nil {
		app.PreserveHostHeader = *req.PreserveHostHeader
	}
	if req.StripPrefix != nil {
		app.StripPrefix = *req.StripPrefix
	}
	if req.InjectHeaders != nil {
		app.InjectHeaders = req.InjectHeaders
	}
	if req.AllowedHeaders != nil {
		app.AllowedHeaders = req.AllowedHeaders
	}
	if req.WebsocketEnabled != nil {
		app.WebsocketEnabled = *req.WebsocketEnabled
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		app.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := s.proxyAppStore.UpdateProxyApplication(c.Request.Context(), app); err != nil {
		if err == db.ErrProxyAppExists {
			c.JSON(http.StatusConflict, gin.H{"error": "application with this slug already exists"})
			return
		}
		s.logger.Error("Failed to update proxy application", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update application"})
		return
	}

	s.logger.Info("Proxy application updated",
		zap.String("id", app.ID),
		zap.String("slug", app.Slug),
		zap.String("admin", user.Email))

	c.JSON(http.StatusOK, app)
}

// handleDeleteProxyApp deletes a proxy application
func (s *Server) handleDeleteProxyApp(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	if err := s.proxyAppStore.DeleteProxyApplication(c.Request.Context(), id); err != nil {
		if err == db.ErrProxyAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
			return
		}
		s.logger.Error("Failed to delete proxy application", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete application"})
		return
	}

	s.logger.Info("Proxy application deleted",
		zap.String("id", id),
		zap.String("admin", user.Email))

	c.JSON(http.StatusOK, gin.H{"message": "application deleted"})
}

// ---- User Assignment Handlers ----

func (s *Server) handleGetProxyAppUsers(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	users, err := s.proxyAppStore.GetAppUsers(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get proxy app users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *Server) handleAssignProxyAppToUser(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.proxyAppStore.AssignAppToUser(c.Request.Context(), req.UserID, id); err != nil {
		s.logger.Error("Failed to assign proxy app to user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user assigned"})
}

func (s *Server) handleRemoveProxyAppFromUser(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	userID := c.Param("userId")

	if err := s.proxyAppStore.RemoveAppFromUser(c.Request.Context(), userID, id); err != nil {
		s.logger.Error("Failed to remove user from proxy app", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed"})
}

// ---- Group Assignment Handlers ----

func (s *Server) handleGetProxyAppGroups(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	groups, err := s.proxyAppStore.GetAppGroups(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get proxy app groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (s *Server) handleAssignProxyAppToGroup(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	var req struct {
		GroupName string `json:"group_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.proxyAppStore.AssignAppToGroup(c.Request.Context(), req.GroupName, id); err != nil {
		s.logger.Error("Failed to assign proxy app to group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group assigned"})
}

func (s *Server) handleRemoveProxyAppFromGroup(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	groupName := c.Param("groupName")

	if err := s.proxyAppStore.RemoveAppFromGroup(c.Request.Context(), groupName, id); err != nil {
		s.logger.Error("Failed to remove group from proxy app", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group removed"})
}

// ---- Audit Logs Handler ----

func (s *Server) handleGetProxyAppLogs(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id := c.Param("id")
	logs, err := s.proxyAppStore.GetProxyAccessLogs(c.Request.Context(), id, 100)
	if err != nil {
		s.logger.Error("Failed to get proxy access logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// ---- User Portal Handler ----

// handleListUserProxyApps returns proxy applications the authenticated user can access
func (s *Server) handleListUserProxyApps(c *gin.Context) {
	userID, groups, err := s.getCurrentUserInfo(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	apps, err := s.proxyAppStore.GetUserProxyApplications(c.Request.Context(), userID, groups)
	if err != nil {
		s.logger.Error("Failed to get user proxy applications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get applications"})
		return
	}

	// Build response with proxy URLs
	type appResponse struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Slug        string    `json:"slug"`
		Description string    `json:"description"`
		IconURL     *string   `json:"icon_url"`
		ProxyURL    string    `json:"proxy_url"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var response []appResponse
	for _, app := range apps {
		response = append(response, appResponse{
			ID:          app.ID,
			Name:        app.Name,
			Slug:        app.Slug,
			Description: app.Description,
			IconURL:     app.IconURL,
			ProxyURL:    "/proxy/" + app.Slug + "/",
			CreatedAt:   app.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"applications": response})
}
