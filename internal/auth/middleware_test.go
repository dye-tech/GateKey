package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/config"
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
