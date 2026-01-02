// Package auth provides authentication middleware for GateKey.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/models"
)

const (
	// ContextKeyUser is the key for the user in the Gin context.
	ContextKeyUser = "gatekey_user"
	// ContextKeySession is the key for the session in the Gin context.
	ContextKeySession = "gatekey_session"
	// ContextKeyAPIKey is the key for the API key in the Gin context.
	ContextKeyAPIKey = "gatekey_api_key"
	// ContextKeyScopes is the key for API key scopes in the Gin context.
	ContextKeyScopes = "gatekey_scopes"
)

// Middleware provides authentication middleware for Gin.
type Middleware struct {
	manager     *Manager
	apiKeyStore *db.APIKeyStore
	cookieName  string
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(manager *Manager, cookieName string) *Middleware {
	return &Middleware{
		manager:    manager,
		cookieName: cookieName,
	}
}

// SetAPIKeyStore sets the API key store for API key authentication.
func (m *Middleware) SetAPIKeyStore(store *db.APIKeyStore) {
	m.apiKeyStore = store
}

// RequireAuth is middleware that requires a valid session.
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, user, err := m.extractAndValidateSession(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication error",
			})
			return
		}

		if session == nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		// Store in context
		c.Set(ContextKeyUser, user)
		c.Set(ContextKeySession, session)
		c.Next()
	}
}

// RequireAdmin is middleware that requires an admin user.
func (m *Middleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, user, err := m.extractAndValidateSession(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication error",
			})
			return
		}

		if session == nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
			})
			return
		}

		c.Set(ContextKeyUser, user)
		c.Set(ContextKeySession, session)
		c.Next()
	}
}

// OptionalAuth is middleware that validates a session if present but doesn't require it.
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, user, err := m.extractAndValidateSession(c)
		if err != nil {
			// Log error but continue
			c.Next()
			return
		}

		if session != nil && user != nil {
			c.Set(ContextKeyUser, user)
			c.Set(ContextKeySession, session)
		}
		c.Next()
	}
}

// extractAndValidateSession extracts the session token and validates it.
func (m *Middleware) extractAndValidateSession(c *gin.Context) (*models.Session, *models.User, error) {
	token := m.extractToken(c)
	if token == "" {
		return nil, nil, nil
	}

	return m.manager.ValidateSession(c.Request.Context(), token)
}

// extractToken extracts the session token from the request.
func (m *Middleware) extractToken(c *gin.Context) string {
	token, _ := m.extractTokenWithType(c)
	return token
}

// extractTokenWithType extracts the token and indicates if it's an API key.
func (m *Middleware) extractTokenWithType(c *gin.Context) (token string, isAPIKey bool) {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
			// Check if it's an API key (starts with gk_)
			if strings.HasPrefix(token, "gk_") {
				return token, true
			}
			return token, false
		}
	}

	// Try cookie (never an API key)
	cookie, err := c.Cookie(m.cookieName)
	if err == nil && cookie != "" {
		return cookie, false
	}

	return "", false
}

// RequireAuthOrAPIKey is middleware that accepts either a session or API key.
func (m *Middleware) RequireAuthOrAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, isAPIKey := m.extractTokenWithType(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		if isAPIKey {
			// Validate API key
			if m.apiKeyStore == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "API key authentication not configured",
				})
				return
			}

			keyHash := db.HashAPIKey(token)
			apiKey, ssoUser, err := m.apiKeyStore.ValidateKey(c.Request.Context(), keyHash)
			if err != nil {
				if err == db.ErrAPIKeyRevoked {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "API key has been revoked",
					})
					return
				}
				if err == db.ErrAPIKeyExpired {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "API key has expired",
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}
			if apiKey == nil || ssoUser == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}

			// Update last used (async to not slow down request)
			go func() {
				bgCtx := context.Background()
				_ = m.apiKeyStore.UpdateLastUsed(bgCtx, apiKey.ID, c.ClientIP())
			}()

			// Convert SSOUser to models.User for consistency
			userID, _ := uuid.Parse(ssoUser.ID)
			user := &models.User{
				ID:       userID,
				Email:    ssoUser.Email,
				Name:     ssoUser.Name,
				Groups:   ssoUser.Groups,
				IsAdmin:  ssoUser.IsAdmin,
				IsActive: ssoUser.IsActive,
			}

			c.Set(ContextKeyUser, user)
			c.Set(ContextKeyAPIKey, apiKey)
			c.Set(ContextKeyScopes, apiKey.Scopes)
		} else {
			// Validate session token
			session, user, err := m.manager.ValidateSession(c.Request.Context(), token)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Authentication error",
				})
				return
			}
			if session == nil || user == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid session",
				})
				return
			}

			c.Set(ContextKeyUser, user)
			c.Set(ContextKeySession, session)
			// Sessions have full access (no scope restrictions)
			c.Set(ContextKeyScopes, []string{"*"})
		}

		c.Next()
	}
}

// RequireAdminOrAPIKey is middleware that requires admin access via session or API key.
func (m *Middleware) RequireAdminOrAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, isAPIKey := m.extractTokenWithType(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		if isAPIKey {
			// Validate API key
			if m.apiKeyStore == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "API key authentication not configured",
				})
				return
			}

			keyHash := db.HashAPIKey(token)
			apiKey, ssoUser, err := m.apiKeyStore.ValidateKey(c.Request.Context(), keyHash)
			if err != nil {
				if err == db.ErrAPIKeyRevoked {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "API key has been revoked",
					})
					return
				}
				if err == db.ErrAPIKeyExpired {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "API key has expired",
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}
			if apiKey == nil || ssoUser == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}

			if !ssoUser.IsAdmin {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Admin access required",
				})
				return
			}

			// Update last used (async)
			go func() {
				bgCtx := context.Background()
				_ = m.apiKeyStore.UpdateLastUsed(bgCtx, apiKey.ID, c.ClientIP())
			}()

			// Convert SSOUser to models.User for consistency
			userID, _ := uuid.Parse(ssoUser.ID)
			user := &models.User{
				ID:       userID,
				Email:    ssoUser.Email,
				Name:     ssoUser.Name,
				Groups:   ssoUser.Groups,
				IsAdmin:  ssoUser.IsAdmin,
				IsActive: ssoUser.IsActive,
			}

			c.Set(ContextKeyUser, user)
			c.Set(ContextKeyAPIKey, apiKey)
			c.Set(ContextKeyScopes, apiKey.Scopes)
		} else {
			// Validate session token
			session, user, err := m.manager.ValidateSession(c.Request.Context(), token)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Authentication error",
				})
				return
			}
			if session == nil || user == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid session",
				})
				return
			}

			if !user.IsAdmin {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Admin access required",
				})
				return
			}

			c.Set(ContextKeyUser, user)
			c.Set(ContextKeySession, session)
			c.Set(ContextKeyScopes, []string{"*"})
		}

		c.Next()
	}
}

// GetUser returns the authenticated user from the context.
func GetUser(c *gin.Context) *models.User {
	if user, ok := c.Get(ContextKeyUser); ok {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// GetSession returns the current session from the context.
func GetSession(c *gin.Context) *models.Session {
	if session, ok := c.Get(ContextKeySession); ok {
		if s, ok := session.(*models.Session); ok {
			return s
		}
	}
	return nil
}

// GetAPIKey returns the API key from the context if authenticated via API key.
func GetAPIKey(c *gin.Context) *db.APIKey {
	if apiKey, ok := c.Get(ContextKeyAPIKey); ok {
		if k, ok := apiKey.(*db.APIKey); ok {
			return k
		}
	}
	return nil
}

// GetScopes returns the scopes from the context.
func GetScopes(c *gin.Context) []string {
	if scopes, ok := c.Get(ContextKeyScopes); ok {
		if s, ok := scopes.([]string); ok {
			return s
		}
	}
	return nil
}

// HasScope checks if the current context has a specific scope.
func HasScope(c *gin.Context, scope string) bool {
	scopes := GetScopes(c)
	if scopes == nil {
		return false
	}
	for _, s := range scopes {
		if s == "*" || s == scope {
			return true
		}
	}
	return false
}

// GatewayAuthMiddleware provides authentication for gateway-to-server communication.
type GatewayAuthMiddleware struct {
	gatewayRepo *models.GatewayRepository
}

// NewGatewayAuthMiddleware creates a new gateway authentication middleware.
func NewGatewayAuthMiddleware(gatewayRepo *models.GatewayRepository) *GatewayAuthMiddleware {
	return &GatewayAuthMiddleware{
		gatewayRepo: gatewayRepo,
	}
}

// RequireGatewayAuth validates gateway authentication tokens.
func (m *GatewayAuthMiddleware) RequireGatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		token := c.GetHeader("X-Gateway-Token")
		if token == "" {
			// Try Authorization header
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Gateway ") {
				token = strings.TrimPrefix(authHeader, "Gateway ")
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Gateway authentication required",
			})
			return
		}

		// Validate token
		tokenHash := hashToken(token)
		gateway, err := m.gatewayRepo.ValidateToken(c.Request.Context(), tokenHash)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication error",
			})
			return
		}

		if gateway == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid gateway token",
			})
			return
		}

		// Store gateway in context
		c.Set("gateway", gateway)
		c.Next()
	}
}

// GetGateway returns the authenticated gateway from the context.
func GetGateway(c *gin.Context) *models.Gateway {
	if gateway, ok := c.Get("gateway"); ok {
		if g, ok := gateway.(*models.Gateway); ok {
			return g
		}
	}
	return nil
}
