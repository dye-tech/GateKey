package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// CreateAPIKeyRequest is the request for creating an API key
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   string   `json:"expires_in"` // e.g., "30d", "1y", "never"
}

// CreateAPIKeyForUserRequest is the request for creating an API key for a specific user
type CreateAPIKeyForUserRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   string   `json:"expires_in"`
}

// APIKeyResponse is the response for an API key
type APIKeyResponse struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	KeyPrefix          string     `json:"key_prefix"`
	Scopes             []string   `json:"scopes"`
	IsAdminProvisioned bool       `json:"is_admin_provisioned"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	IsRevoked          bool       `json:"is_revoked"`
	CreatedAt          time.Time  `json:"created_at"`
}

// AdminAPIKeyResponse extends APIKeyResponse with user information for admin views
type AdminAPIKeyResponse struct {
	APIKeyResponse
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

// CreateAPIKeyResponse includes the raw key (only shown once)
type CreateAPIKeyResponse struct {
	APIKeyResponse
	RawKey string `json:"raw_key"` // Only returned on creation
}

// parseExpiration parses an expiration string like "30d", "1y", "never"
func parseExpiration(expiresIn string) *time.Time {
	if expiresIn == "" || expiresIn == "never" {
		return nil
	}

	var duration time.Duration
	switch {
	case len(expiresIn) > 1 && expiresIn[len(expiresIn)-1] == 'd':
		days := 0
		if _, err := time.ParseDuration(expiresIn[:len(expiresIn)-1] + "h"); err == nil {
			// Parse as hours and multiply
		}
		// Simple parsing
		if n, _ := parseInt(expiresIn[:len(expiresIn)-1]); n > 0 {
			days = n
		}
		duration = time.Duration(days) * 24 * time.Hour
	case len(expiresIn) > 1 && expiresIn[len(expiresIn)-1] == 'y':
		years := 0
		if n, _ := parseInt(expiresIn[:len(expiresIn)-1]); n > 0 {
			years = n
		}
		duration = time.Duration(years) * 365 * 24 * time.Hour
	case len(expiresIn) > 1 && expiresIn[len(expiresIn)-1] == 'h':
		if d, err := time.ParseDuration(expiresIn); err == nil {
			duration = d
		}
	default:
		// Try parsing as duration
		if d, err := time.ParseDuration(expiresIn); err == nil {
			duration = d
		}
	}

	if duration > 0 {
		t := time.Now().Add(duration)
		return &t
	}
	return nil
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0, nil
		}
	}
	return n, nil
}

// toAPIKeyResponse converts an APIKey to a response
func toAPIKeyResponse(key *db.APIKey) APIKeyResponse {
	return APIKeyResponse{
		ID:                 key.ID,
		Name:               key.Name,
		Description:        key.Description,
		KeyPrefix:          key.KeyPrefix,
		Scopes:             key.Scopes,
		IsAdminProvisioned: key.IsAdminProvisioned,
		ExpiresAt:          key.ExpiresAt,
		LastUsedAt:         key.LastUsedAt,
		IsRevoked:          key.IsRevoked,
		CreatedAt:          key.CreatedAt,
	}
}

// toAdminAPIKeyResponse converts an APIKeyWithUser to an admin response
func toAdminAPIKeyResponse(key *db.APIKeyWithUser) AdminAPIKeyResponse {
	return AdminAPIKeyResponse{
		APIKeyResponse: toAPIKeyResponse(&key.APIKey),
		UserID:         key.UserID,
		UserEmail:      key.UserEmail,
		UserName:       key.UserName,
	}
}

// handleListUserAPIKeys lists API keys for the authenticated user
func (s *Server) handleListUserAPIKeys(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keys, err := s.apiKeyStore.ListByUser(c.Request.Context(), user.UserID)
	if err != nil {
		s.logger.Error("Failed to list API keys", zap.Error(err), zap.String("user_id", user.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	response := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = toAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

// handleCreateUserAPIKey creates an API key for the authenticated user
func (s *Server) handleCreateUserAPIKey(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate the API key
	rawKey, keyHash, keyPrefix, err := db.GenerateAPIKey()
	if err != nil {
		s.logger.Error("Failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Parse expiration
	expiresAt := parseExpiration(req.ExpiresIn)

	// Set default scopes if none provided
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"} // Full access by default for user-created keys
	}

	apiKey := &db.APIKey{
		UserID:             user.UserID,
		Name:               req.Name,
		Description:        req.Description,
		KeyHash:            keyHash,
		KeyPrefix:          keyPrefix,
		Scopes:             scopes,
		IsAdminProvisioned: false,
		ExpiresAt:          expiresAt,
	}

	if err := s.apiKeyStore.Create(c.Request.Context(), apiKey); err != nil {
		s.logger.Error("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	// Return the raw key only on creation
	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKeyResponse: toAPIKeyResponse(apiKey),
		RawKey:         rawKey,
	})
}

// handleGetUserAPIKey gets a specific API key for the authenticated user
func (s *Server) handleGetUserAPIKey(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keyID := c.Param("id")
	key, err := s.apiKeyStore.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if err == db.ErrAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		s.logger.Error("Failed to get API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	// Ensure user owns this key
	if key.UserID != user.UserID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, toAPIKeyResponse(key))
}

// handleRevokeUserAPIKey revokes an API key for the authenticated user
func (s *Server) handleRevokeUserAPIKey(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keyID := c.Param("id")
	key, err := s.apiKeyStore.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if err == db.ErrAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		s.logger.Error("Failed to get API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	// Ensure user owns this key
	if key.UserID != user.UserID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if err := s.apiKeyStore.Revoke(c.Request.Context(), keyID, user.UserID, "Revoked by user"); err != nil {
		s.logger.Error("Failed to revoke API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// Admin handlers

// handleAdminListAPIKeys lists all API keys (admin only)
func (s *Server) handleAdminListAPIKeys(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keys, err := s.apiKeyStore.ListAllWithUserInfo(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to list API keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	response := make([]AdminAPIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = toAdminAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

// handleAdminCreateAPIKey creates an API key for any user (admin only)
func (s *Server) handleAdminCreateAPIKey(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		UserID      string   `json:"user_id" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Scopes      []string `json:"scopes"`
		ExpiresIn   string   `json:"expires_in"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate the API key
	rawKey, keyHash, keyPrefix, err := db.GenerateAPIKey()
	if err != nil {
		s.logger.Error("Failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Parse expiration
	expiresAt := parseExpiration(req.ExpiresIn)

	// Set default scopes
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}

	adminIDStr := admin.UserID
	apiKey := &db.APIKey{
		UserID:             req.UserID,
		Name:               req.Name,
		Description:        req.Description,
		KeyHash:            keyHash,
		KeyPrefix:          keyPrefix,
		Scopes:             scopes,
		IsAdminProvisioned: true,
		ProvisionedBy:      &adminIDStr,
		ExpiresAt:          expiresAt,
	}

	if err := s.apiKeyStore.Create(c.Request.Context(), apiKey); err != nil {
		s.logger.Error("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKeyResponse: toAPIKeyResponse(apiKey),
		RawKey:         rawKey,
	})
}

// handleAdminGetAPIKey gets a specific API key (admin only)
func (s *Server) handleAdminGetAPIKey(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keyID := c.Param("id")
	key, err := s.apiKeyStore.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if err == db.ErrAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		s.logger.Error("Failed to get API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	c.JSON(http.StatusOK, toAPIKeyResponse(key))
}

// handleAdminRevokeAPIKey revokes any API key (admin only)
func (s *Server) handleAdminRevokeAPIKey(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	keyID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	reason := req.Reason
	if reason == "" {
		reason = "Revoked by admin"
	}

	if err := s.apiKeyStore.Revoke(c.Request.Context(), keyID, admin.UserID, reason); err != nil {
		if err == db.ErrAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		s.logger.Error("Failed to revoke API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// handleAdminListUserAPIKeys lists API keys for a specific user (admin only)
func (s *Server) handleAdminListUserAPIKeys(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID := c.Param("id")

	keys, err := s.apiKeyStore.ListByUser(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to list API keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	response := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = toAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

// handleAdminCreateUserAPIKey creates an API key for a specific user (admin only)
func (s *Server) handleAdminCreateUserAPIKey(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID := c.Param("id")

	var req CreateAPIKeyForUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate the API key
	rawKey, keyHash, keyPrefix, err := db.GenerateAPIKey()
	if err != nil {
		s.logger.Error("Failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Parse expiration
	expiresAt := parseExpiration(req.ExpiresIn)

	// Set default scopes
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}

	adminIDStr := admin.UserID
	apiKey := &db.APIKey{
		UserID:             userID,
		Name:               req.Name,
		Description:        req.Description,
		KeyHash:            keyHash,
		KeyPrefix:          keyPrefix,
		Scopes:             scopes,
		IsAdminProvisioned: true,
		ProvisionedBy:      &adminIDStr,
		ExpiresAt:          expiresAt,
	}

	if err := s.apiKeyStore.Create(c.Request.Context(), apiKey); err != nil {
		s.logger.Error("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKeyResponse: toAPIKeyResponse(apiKey),
		RawKey:         rawKey,
	})
}

// handleAdminRevokeUserAPIKeys revokes all API keys for a user (admin only)
func (s *Server) handleAdminRevokeUserAPIKeys(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	reason := req.Reason
	if reason == "" {
		reason = "Revoked by admin"
	}

	count, err := s.apiKeyStore.RevokeAllForUser(c.Request.Context(), userID, admin.UserID, reason)
	if err != nil {
		s.logger.Error("Failed to revoke API keys")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API keys revoked",
		"count":   count,
	})
}

// handleAdminDeleteUserAPIKeys permanently deletes all API keys for a user (admin only)
func (s *Server) handleAdminDeleteUserAPIKeys(c *gin.Context) {
	admin, err := s.getAuthenticatedUser(c)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID := c.Param("id")

	count, err := s.apiKeyStore.DeleteAllForUser(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to delete API keys", zap.String("user_id", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API keys"})
		return
	}

	s.logger.Info("Admin deleted all API keys for user",
		zap.String("admin_id", admin.UserID),
		zap.String("user_id", userID),
		zap.Int64("deleted_count", count))

	c.JSON(http.StatusOK, gin.H{
		"message":      "API keys deleted permanently",
		"deletedCount": count,
	})
}

// handleValidateAPIKey validates an API key and returns user info (for CLI login)
func (s *Server) handleValidateAPIKey(c *gin.Context) {
	// Extract API key from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	apiKeyRaw := strings.TrimPrefix(authHeader, "Bearer ")
	if apiKeyRaw == authHeader || !strings.HasPrefix(apiKeyRaw, "gk_") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format"})
		return
	}

	keyHash := db.HashAPIKey(apiKeyRaw)
	apiKey, user, err := s.apiKeyStore.ValidateKey(c.Request.Context(), keyHash)
	if err != nil {
		if err == db.ErrAPIKeyRevoked {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key has been revoked"})
			return
		}
		if err == db.ErrAPIKeyExpired {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key has expired"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}
	if apiKey == nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	// Update last used
	go s.apiKeyStore.UpdateLastUsed(c.Request.Context(), apiKey.ID, c.ClientIP())

	// Determine admin status - check both IsAdmin flag and group membership
	isAdmin := user.IsAdmin
	if !isAdmin && user.Provider != "" && len(user.Groups) > 0 {
		// Check if user is in admin group for their provider
		if oidcProvider, err := s.providerStore.GetOIDCProvider(c.Request.Context(), user.Provider); err == nil && oidcProvider.AdminGroup != "" {
			for _, group := range user.Groups {
				if group == oidcProvider.AdminGroup {
					isAdmin = true
					break
				}
			}
		}
		if !isAdmin {
			if samlProvider, err := s.providerStore.GetSAMLProvider(c.Request.Context(), user.Provider); err == nil && samlProvider.AdminGroup != "" {
				for _, group := range user.Groups {
					if group == samlProvider.AdminGroup {
						isAdmin = true
						break
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"is_admin": isAdmin,
		"scopes":   apiKey.Scopes,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
		"api_key": gin.H{
			"id":         apiKey.ID,
			"name":       apiKey.Name,
			"expires_at": apiKey.ExpiresAt,
		},
	})
}
