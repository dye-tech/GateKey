package telemetry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	router.GET("/users/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "User: "+c.Param("id"))
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"simple_route", "/test", http.StatusOK},
		{"parameterized_route", "/users/123", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGinMiddleware_ErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "Error")
	})

	router.GET("/not-found", func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"server_error", "/error", http.StatusInternalServerError},
		{"not_found", "/not-found", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGinMiddleware_TracePropagation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	router.GET("/trace", func(c *gin.Context) {
		// Verify that the context has been updated
		span := SpanFromContext(c.Request.Context())
		if span == nil {
			t.Error("Expected span in context")
		}
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/trace", nil)

	// Add trace context headers
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGinMiddleware_NoPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	// NoRoute handler for unmatched paths
	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/unknown-path", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
