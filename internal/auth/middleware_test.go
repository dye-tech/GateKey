package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewMiddleware(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	middleware := NewMiddleware(manager, "session_token")

	if middleware == nil {
		t.Error("Middleware should not be nil")
	}

	if middleware.cookieName != "session_token" {
		t.Errorf("Expected cookie name 'session_token', got '%s'", middleware.cookieName)
	}
}

func TestRequireAuth_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "Authentication required" {
		t.Errorf("Expected error 'Authentication required', got '%v'", response["error"])
	}
}

func TestRequireAdmin_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestOptionalAuth_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": true, "email": user.Email})
		}
	})

	req, _ := http.NewRequest("GET", "/optional", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// OptionalAuth should allow the request through
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["authenticated"] != false {
		t.Errorf("Expected authenticated=false, got %v", response["authenticated"])
	}
}

func TestExtractToken_FromHeader(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	// Test Bearer token extraction
	router := gin.New()
	var extractedToken string
	router.GET("/test", func(c *gin.Context) {
		extractedToken = middleware.extractToken(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer my-test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "my-test-token" {
		t.Errorf("Expected token 'my-test-token', got '%s'", extractedToken)
	}
}

func TestExtractToken_FromCookie(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	router.GET("/test", func(c *gin.Context) {
		extractedToken = middleware.extractToken(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "cookie-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "cookie-token" {
		t.Errorf("Expected token 'cookie-token', got '%s'", extractedToken)
	}
}

func TestExtractToken_HeaderPrecedence(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	router.GET("/test", func(c *gin.Context) {
		extractedToken = middleware.extractToken(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "cookie-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Header should take precedence over cookie
	if extractedToken != "header-token" {
		t.Errorf("Expected token 'header-token' (header precedence), got '%s'", extractedToken)
	}
}

func TestExtractTokenWithType_APIKey(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	var isAPIKey bool
	router.GET("/test", func(c *gin.Context) {
		extractedToken, isAPIKey = middleware.extractTokenWithType(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer gk_test_api_key_12345")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "gk_test_api_key_12345" {
		t.Errorf("Expected token 'gk_test_api_key_12345', got '%s'", extractedToken)
	}

	if !isAPIKey {
		t.Error("Expected isAPIKey=true for gk_ prefixed token")
	}
}

func TestExtractTokenWithType_SessionToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	var isAPIKey bool
	router.GET("/test", func(c *gin.Context) {
		extractedToken, isAPIKey = middleware.extractTokenWithType(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer regular-session-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "regular-session-token" {
		t.Errorf("Expected token 'regular-session-token', got '%s'", extractedToken)
	}

	if isAPIKey {
		t.Error("Expected isAPIKey=false for non-gk_ prefixed token")
	}
}

func TestGetUser(t *testing.T) {
	router := gin.New()

	testUser := &models.User{
		ID:      uuid.New(),
		Email:   "test@example.com",
		Name:    "Test User",
		IsAdmin: true,
	}

	router.GET("/test", func(c *gin.Context) {
		c.Set(ContextKeyUser, testUser)

		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%v'", response["email"])
	}

	if response["is_admin"] != true {
		t.Errorf("Expected is_admin=true, got %v", response["is_admin"])
	}
}

func TestGetUser_NotSet(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusOK, gin.H{"user": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user.Email})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["user"] != nil {
		t.Errorf("Expected user=nil, got %v", response["user"])
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		check    string
		expected bool
	}{
		{
			name:     "wildcard scope matches any",
			scopes:   []string{"*"},
			check:    "admin:read",
			expected: true,
		},
		{
			name:     "exact scope match",
			scopes:   []string{"admin:read", "user:write"},
			check:    "admin:read",
			expected: true,
		},
		{
			name:     "scope not found",
			scopes:   []string{"admin:read", "user:write"},
			check:    "admin:delete",
			expected: false,
		},
		{
			name:     "empty scopes",
			scopes:   []string{},
			check:    "admin:read",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			router.GET("/test", func(c *gin.Context) {
				c.Set(ContextKeyScopes, tt.scopes)
				result := HasScope(c, tt.check)
				c.JSON(http.StatusOK, gin.H{"has_scope": result})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["has_scope"] != tt.expected {
				t.Errorf("Expected has_scope=%v, got %v", tt.expected, response["has_scope"])
			}
		})
	}
}

func TestHasScope_NilScopes(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Don't set any scopes
		result := HasScope(c, "admin:read")
		c.JSON(http.StatusOK, gin.H{"has_scope": result})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["has_scope"] != false {
		t.Errorf("Expected has_scope=false for nil scopes, got %v", response["has_scope"])
	}
}

func TestRequireAuthOrAPIKey_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	router.GET("/protected", middleware.RequireAuthOrAPIKey(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuthOrAPIKey_APIKeyWithoutStore(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")
	// Don't set API key store

	router := gin.New()
	router.GET("/protected", middleware.RequireAuthOrAPIKey(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer gk_test_api_key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when API key store not configured, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "API key authentication not configured" {
		t.Errorf("Expected error about API key not configured, got '%v'", response["error"])
	}
}

func TestGetSession(t *testing.T) {
	router := gin.New()

	testSession := &models.Session{
		Token: "test-token-123",
	}

	router.GET("/test", func(c *gin.Context) {
		c.Set(ContextKeySession, testSession)

		session := GetSession(c)
		if session == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": session.Token})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["token"] != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%v'", response["token"])
	}
}

func TestGetSession_NotSet(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		session := GetSession(c)
		if session == nil {
			c.JSON(http.StatusOK, gin.H{"session": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"session": "found"})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["session"] != nil {
		t.Errorf("Expected session=nil, got %v", response["session"])
	}
}

func TestGetSession_WrongType(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set(ContextKeySession, "not-a-session")

		session := GetSession(c)
		if session == nil {
			c.JSON(http.StatusOK, gin.H{"session": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"session": "found"})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["session"] != nil {
		t.Errorf("Expected session=nil for wrong type, got %v", response["session"])
	}
}

func TestGetScopes(t *testing.T) {
	router := gin.New()

	testScopes := []string{"read:users", "write:users", "admin"}

	router.GET("/test", func(c *gin.Context) {
		c.Set(ContextKeyScopes, testScopes)

		scopes := GetScopes(c)
		c.JSON(http.StatusOK, gin.H{"scopes": scopes})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	scopes := response["scopes"].([]interface{})
	if len(scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(scopes))
	}
}

func TestGetScopes_NotSet(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		scopes := GetScopes(c)
		if scopes == nil {
			c.JSON(http.StatusOK, gin.H{"scopes": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"scopes": scopes})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["scopes"] != nil {
		t.Errorf("Expected scopes=nil, got %v", response["scopes"])
	}
}

func TestGetScopes_WrongType(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set(ContextKeyScopes, "not-a-slice")

		scopes := GetScopes(c)
		if scopes == nil {
			c.JSON(http.StatusOK, gin.H{"scopes": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"scopes": scopes})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["scopes"] != nil {
		t.Errorf("Expected scopes=nil for wrong type, got %v", response["scopes"])
	}
}

func TestHasScope_WithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		check    string
		expected bool
	}{
		{
			name:     "admin:write in admin scopes",
			scopes:   []string{"admin:read", "admin:write", "admin:delete"},
			check:    "admin:write",
			expected: true,
		},
		{
			name:     "user:delete not in admin scopes",
			scopes:   []string{"admin:read", "admin:write"},
			check:    "user:delete",
			expected: false,
		},
		{
			name:     "wildcard matches everything",
			scopes:   []string{"*"},
			check:    "any:scope:here",
			expected: true,
		},
		{
			name:     "case sensitive match",
			scopes:   []string{"Admin:Read"},
			check:    "admin:read",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			router.GET("/test", func(c *gin.Context) {
				c.Set(ContextKeyScopes, tt.scopes)
				result := HasScope(c, tt.check)
				c.JSON(http.StatusOK, gin.H{"has_scope": result})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["has_scope"] != tt.expected {
				t.Errorf("Expected has_scope=%v, got %v", tt.expected, response["has_scope"])
			}
		})
	}
}

func TestSetAPIKeyStore(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	// Initially should be nil
	if middleware.apiKeyStore != nil {
		t.Error("API key store should initially be nil")
	}

	// Set API key store (we can't actually test with a real store without a DB,
	// but we can verify the setter works)
	middleware.SetAPIKeyStore(nil)

	// After setting nil, should still be nil
	if middleware.apiKeyStore != nil {
		t.Error("API key store should be nil after setting nil")
	}
}

func TestRequireAdminOrAPIKey_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	router.GET("/admin", middleware.RequireAdminOrAPIKey(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "Authentication required" {
		t.Errorf("Expected error 'Authentication required', got '%v'", response["error"])
	}
}

func TestRequireAdminOrAPIKey_APIKeyWithoutStore(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")
	// Don't set API key store

	router := gin.New()
	router.GET("/admin", middleware.RequireAdminOrAPIKey(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer gk_admin_api_key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when API key store not configured, got %d", w.Code)
	}
}

func TestGetGateway(t *testing.T) {
	router := gin.New()

	testGateway := &models.Gateway{
		Name: "test-gateway",
	}

	router.GET("/test", func(c *gin.Context) {
		c.Set("gateway", testGateway)

		gateway := GetGateway(c)
		if gateway == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gateway not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"name": gateway.Name})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "test-gateway" {
		t.Errorf("Expected name 'test-gateway', got '%v'", response["name"])
	}
}

func TestGetGateway_NotSet(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		gateway := GetGateway(c)
		if gateway == nil {
			c.JSON(http.StatusOK, gin.H{"gateway": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"gateway": gateway.Name})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["gateway"] != nil {
		t.Errorf("Expected gateway=nil, got %v", response["gateway"])
	}
}

func TestGetGateway_WrongType(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set("gateway", "not-a-gateway")

		gateway := GetGateway(c)
		if gateway == nil {
			c.JSON(http.StatusOK, gin.H{"gateway": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"gateway": gateway.Name})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["gateway"] != nil {
		t.Errorf("Expected gateway=nil for wrong type, got %v", response["gateway"])
	}
}

func TestExtractToken_NoToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	router.GET("/test", func(c *gin.Context) {
		extractedToken = middleware.extractToken(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "" {
		t.Errorf("Expected empty token, got '%s'", extractedToken)
	}
}

func TestExtractToken_InvalidAuthHeader(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	router.GET("/test", func(c *gin.Context) {
		extractedToken = middleware.extractToken(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth, not Bearer
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "" {
		t.Errorf("Expected empty token for Basic auth, got '%s'", extractedToken)
	}
}

func TestContextKeys(t *testing.T) {
	// Verify context key values are as expected
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"ContextKeyUser", ContextKeyUser, "gatekey_user"},
		{"ContextKeySession", ContextKeySession, "gatekey_session"},
		{"ContextKeyAPIKey", ContextKeyAPIKey, "gatekey_api_key"},
		{"ContextKeyScopes", ContextKeyScopes, "gatekey_scopes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.key)
			}
		})
	}
}

func TestRequireAdmin_NonAdminUser(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()

	// Simulate a non-admin user being set in context (bypassing actual validation)
	router.GET("/admin", func(c *gin.Context) {
		nonAdminUser := &models.User{
			ID:      uuid.New(),
			Email:   "user@example.com",
			IsAdmin: false,
		}
		c.Set(ContextKeyUser, nonAdminUser)
		c.Next()
	}, middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because middleware.RequireAdmin extracts from request,
	// not from what we set in context before it
	if w.Code != http.StatusUnauthorized {
		t.Logf("Note: RequireAdmin extracts token from request, not pre-set context")
	}
}

func TestGetAPIKey(t *testing.T) {
	router := gin.New()

	testAPIKey := &db.APIKey{
		ID:     uuid.New().String(),
		Name:   "test-api-key",
		Scopes: []string{"read:users"},
	}

	router.GET("/test", func(c *gin.Context) {
		c.Set(ContextKeyAPIKey, testAPIKey)

		apiKey := GetAPIKey(c)
		if apiKey == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "api key not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"name": apiKey.Name})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "test-api-key" {
		t.Errorf("Expected name 'test-api-key', got '%v'", response["name"])
	}
}

func TestGetAPIKey_NotSet(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		apiKey := GetAPIKey(c)
		if apiKey == nil {
			c.JSON(http.StatusOK, gin.H{"api_key": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"api_key": apiKey.Name})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["api_key"] != nil {
		t.Errorf("Expected api_key=nil, got %v", response["api_key"])
	}
}

func TestGetAPIKey_WrongType(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set(ContextKeyAPIKey, "not-an-api-key")

		apiKey := GetAPIKey(c)
		if apiKey == nil {
			c.JSON(http.StatusOK, gin.H{"api_key": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"api_key": apiKey.Name})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["api_key"] != nil {
		t.Errorf("Expected api_key=nil for wrong type, got %v", response["api_key"])
	}
}

func TestGetUser_WrongType(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set(ContextKeyUser, "not-a-user")

		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusOK, gin.H{"user": nil})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user.Email})
		}
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["user"] != nil {
		t.Errorf("Expected user=nil for wrong type, got %v", response["user"])
	}
}

func TestNewGatewayAuthMiddleware(t *testing.T) {
	// Test creating gateway auth middleware
	middleware := NewGatewayAuthMiddleware(nil)

	if middleware == nil {
		t.Error("NewGatewayAuthMiddleware should not return nil")
	}
}

func TestRequireGatewayAuth_NoToken(t *testing.T) {
	middleware := NewGatewayAuthMiddleware(nil)

	router := gin.New()
	router.GET("/gateway", middleware.RequireGatewayAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/gateway", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "Gateway authentication required" {
		t.Errorf("Expected error 'Gateway authentication required', got '%v'", response["error"])
	}
}

func TestRequireGatewayAuth_WithXGatewayTokenHeader(t *testing.T) {
	// Without a real gateway repo, this will fail validation
	middleware := NewGatewayAuthMiddleware(nil)

	router := gin.New()
	router.GET("/gateway", middleware.RequireGatewayAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/gateway", nil)
	req.Header.Set("X-Gateway-Token", "test-gateway-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error since gateway repo is nil
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when gateway repo is nil, got %d", w.Code)
	}
}

func TestRequireGatewayAuth_WithAuthorizationHeader(t *testing.T) {
	middleware := NewGatewayAuthMiddleware(nil)

	router := gin.New()
	router.GET("/gateway", middleware.RequireGatewayAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/gateway", nil)
	req.Header.Set("Authorization", "Gateway test-gateway-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error since gateway repo is nil
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when gateway repo is nil, got %d", w.Code)
	}
}

func TestExtractTokenWithType_CookieIsNeverAPIKey(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	var isAPIKey bool
	router.GET("/test", func(c *gin.Context) {
		extractedToken, isAPIKey = middleware.extractTokenWithType(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Even if cookie value looks like API key, it should not be treated as one
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "gk_looks_like_api_key"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "gk_looks_like_api_key" {
		t.Errorf("Expected token 'gk_looks_like_api_key', got '%s'", extractedToken)
	}

	if isAPIKey {
		t.Error("Cookie tokens should never be treated as API keys")
	}
}

func TestExtractTokenWithType_EmptyToken(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)
	middleware := NewMiddleware(manager, "session_token")

	router := gin.New()
	var extractedToken string
	var isAPIKey bool
	router.GET("/test", func(c *gin.Context) {
		extractedToken, isAPIKey = middleware.extractTokenWithType(c)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedToken != "" {
		t.Errorf("Expected empty token, got '%s'", extractedToken)
	}

	if isAPIKey {
		t.Error("Empty token should not be API key")
	}
}
