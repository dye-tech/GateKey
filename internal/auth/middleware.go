// Package auth provides authentication middleware for GateKey.
package auth

import (
	"net/http"
	"strings"

	"github.com/gatekey-project/gatekey/internal/models"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUser is the key for the user in the Gin context.
	ContextKeyUser = "gatekey_user"
	// ContextKeySession is the key for the session in the Gin context.
	ContextKeySession = "gatekey_session"
)

// Middleware provides authentication middleware for Gin.
type Middleware struct {
	manager    *Manager
	cookieName string
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(manager *Manager, cookieName string) *Middleware {
	return &Middleware{
		manager:    manager,
		cookieName: cookieName,
	}
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
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Try cookie
	cookie, err := c.Cookie(m.cookieName)
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
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
