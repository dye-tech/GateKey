package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
)

func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	router := gin.New()
	router.Use(GinMiddleware())

	router.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify metrics were recorded
	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/api/v1/test", "200")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter value 1, got %f", *m.Counter.Value)
	}
}

func TestGinMiddleware_SkipsMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.Reset()

	router := gin.New()
	router.Use(GinMiddleware())

	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "metrics")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify metrics were NOT recorded for /metrics path
	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/metrics", "200")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 0 {
		t.Errorf("Expected counter value 0 for /metrics, got %f", *m.Counter.Value)
	}
}

func TestGinMiddleware_InFlightGauge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(GinMiddleware())

	var gaugeValue float64
	router.GET("/slow", func(c *gin.Context) {
		// Check in-flight gauge during request
		var m dto.Metric
		if err := HTTPRequestsInFlight.Write(&m); err == nil {
			gaugeValue = *m.Gauge.Value
		}
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/slow", nil)

	router.ServeHTTP(w, req)

	// During request, gauge should have been at least 1
	if gaugeValue < 1 {
		t.Errorf("Expected in-flight gauge >= 1 during request, got %f", gaugeValue)
	}
}

func TestGinMiddleware_UnknownPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.Reset()

	router := gin.New()
	router.Use(GinMiddleware())

	// No routes defined - should use "unknown" path
	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nonexistent", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Verify metrics were recorded with "unknown" path
	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "unknown", "404")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter value 1, got %f", *m.Counter.Value)
	}
}

func TestGinMiddleware_ErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.Reset()

	router := gin.New()
	router.Use(GinMiddleware())

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "Error")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/error", "500")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter value 1, got %f", *m.Counter.Value)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/api/v1/users/:id", "/api/v1/users/:id"},
		{"/api/v1/test", "/api/v1/test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
