package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gatekey-project/gatekey/internal/config"
)

func TestBuildHSTSHeader(t *testing.T) {
	tests := []struct {
		name              string
		maxAge            int
		includeSubdomains bool
		preload           bool
		expected          string
	}{
		{
			name:              "basic HSTS",
			maxAge:            31536000,
			includeSubdomains: false,
			preload:           false,
			expected:          "max-age=31536000",
		},
		{
			name:              "HSTS with includeSubDomains",
			maxAge:            31536000,
			includeSubdomains: true,
			preload:           false,
			expected:          "max-age=31536000; includeSubDomains",
		},
		{
			name:              "HSTS with preload",
			maxAge:            31536000,
			includeSubdomains: false,
			preload:           true,
			expected:          "max-age=31536000; preload",
		},
		{
			name:              "HSTS with all options",
			maxAge:            31536000,
			includeSubdomains: true,
			preload:           true,
			expected:          "max-age=31536000; includeSubDomains; preload",
		},
		{
			name:              "HSTS with short max-age",
			maxAge:            86400,
			includeSubdomains: true,
			preload:           false,
			expected:          "max-age=86400; includeSubDomains",
		},
		{
			name:              "HSTS with zero max-age",
			maxAge:            0,
			includeSubdomains: false,
			preload:           false,
			expected:          "max-age=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHSTSHeader(tt.maxAge, tt.includeSubdomains, tt.preload)
			if result != tt.expected {
				t.Errorf("buildHSTSHeader() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_BasicHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: false,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			HSTSEnabled:            true,
			HSTSMaxAge:             31536000,
			HSTSIncludeSubdomains:  true,
			HSTSPreload:            false,
			FrameAncestors:         "self",
			PermissionsPolicy:      "geolocation=(), camera=()",
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check basic security headers
	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "geolocation=(), camera=()"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := w.Header().Get(tt.header)
			if got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}

	// HSTS should NOT be set when TLS is disabled
	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS should not be set when TLS is disabled, got %q", hsts)
	}
}

func TestSecurityHeadersMiddleware_HSTSWithTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: true,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			HSTSEnabled:            true,
			HSTSMaxAge:             31536000,
			HSTSIncludeSubdomains:  true,
			HSTSPreload:            false,
			FrameAncestors:         "self",
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// HSTS should be set when TLS is enabled
	hsts := w.Header().Get("Strict-Transport-Security")
	expected := "max-age=31536000; includeSubDomains"
	if hsts != expected {
		t.Errorf("HSTS = %q, want %q", hsts, expected)
	}
}

func TestSecurityHeadersMiddleware_DynamicFrameAncestors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: false,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			FrameAncestors:         "dynamic",
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	tests := []struct {
		name          string
		host          string
		expectedInCSP string
	}{
		{
			name:          "company A tenant",
			host:          "vpn.companya.com",
			expectedInCSP: "frame-ancestors 'self' https://vpn.companya.com",
		},
		{
			name:          "company B tenant",
			host:          "vpn.companyb.com",
			expectedInCSP: "frame-ancestors 'self' https://vpn.companyb.com",
		},
		{
			name:          "host with port",
			host:          "vpn.example.com:8443",
			expectedInCSP: "frame-ancestors 'self' https://vpn.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			csp := w.Header().Get("Content-Security-Policy")
			if csp == "" {
				t.Fatal("CSP header not set")
			}

			if !contains(csp, tt.expectedInCSP) {
				t.Errorf("CSP = %q, should contain %q", csp, tt.expectedInCSP)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_StaticFrameAncestors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: false,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			FrameAncestors:         "https://allowed.example.com",
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "vpn.tenant.com"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if !contains(csp, "frame-ancestors https://allowed.example.com") {
		t.Errorf("CSP = %q, should contain static frame-ancestors", csp)
	}
}

func TestSecurityHeadersMiddleware_CustomCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customCSP := "default-src 'none'; script-src 'self'"

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: false,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			ContentSecurityPolicy:  customCSP,
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp != customCSP {
		t.Errorf("CSP = %q, want %q", csp, customCSP)
	}
}

func TestSecurityHeadersMiddleware_CSPContainsSecureDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSEnabled: false,
		},
		Security: config.SecurityConfig{
			SecurityHeadersEnabled: true,
			FrameAncestors:         "self",
		},
	}

	router := gin.New()
	router.Use(securityHeadersMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")

	// Verify secure defaults are present
	requiredDirectives := []string{
		"default-src 'self'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'self'",
	}

	for _, directive := range requiredDirectives {
		if !contains(csp, directive) {
			t.Errorf("CSP missing required directive %q, got %q", directive, csp)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
