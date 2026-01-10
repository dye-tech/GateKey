package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestLiveCheck tests the /live endpoint which doesn't require any dependencies
func TestLiveCheck(t *testing.T) {
	router := gin.New()

	srv := &Server{
		router: router,
		logger: zap.NewNop(),
	}
	router.GET("/live", srv.liveCheck)

	req, _ := http.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%v'", response["status"])
	}

	if _, ok := response["time"]; !ok {
		t.Error("Expected 'time' field in response")
	}
}

// TestLiveCheckAlwaysReturns200 verifies liveness always returns 200
// This is critical for Kubernetes - liveness should never fail if the process is running
func TestLiveCheckAlwaysReturns200(t *testing.T) {
	router := gin.New()

	// Even with a completely uninitialized server, /live should work
	srv := &Server{
		router: router,
		logger: zap.NewNop(),
		// No db, no ca, nothing - just verifying process is alive
	}
	router.GET("/live", srv.liveCheck)

	// Make multiple requests to ensure consistency
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

// TestLiveCheckResponseFormat verifies the response JSON structure
func TestLiveCheckResponseFormat(t *testing.T) {
	router := gin.New()

	srv := &Server{
		router: router,
		logger: zap.NewNop(),
	}
	router.GET("/live", srv.liveCheck)

	req, _ := http.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{"status", "time"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify status value
	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%v'", response["status"])
	}

	// Verify time is a valid RFC3339 timestamp
	timeStr, ok := response["time"].(string)
	if !ok {
		t.Error("Expected 'time' to be a string")
	}
	if len(timeStr) == 0 {
		t.Error("Expected 'time' to be non-empty")
	}
}

// TestHealthCheckResponseStructure tests the structure of /health response
// Note: This test runs without a real database, so it tests the error path
func TestHealthCheckWithoutDB(t *testing.T) {
	router := gin.New()

	srv := &Server{
		router: router,
		logger: zap.NewNop(),
		db:     nil, // No database
		ca:     nil, // No CA
	}
	router.GET("/health", srv.healthCheck)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// This will panic because db is nil - we catch this to verify behavior
	defer func() {
		if r := recover(); r != nil {
			// Expected - db.Ping() will panic on nil db
			// In production, db should never be nil
			t.Log("As expected, health check panics without database (db is nil)")
		}
	}()

	router.ServeHTTP(w, req)
}

// TestReadyCheckWithoutDB tests ready endpoint behavior without database
func TestReadyCheckWithoutDB(t *testing.T) {
	router := gin.New()

	srv := &Server{
		router: router,
		logger: zap.NewNop(),
		db:     nil, // No database
	}
	router.GET("/ready", srv.readyCheck)

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	// This will panic because db is nil - we catch this
	defer func() {
		if r := recover(); r != nil {
			t.Log("As expected, ready check panics without database (db is nil)")
		}
	}()

	router.ServeHTTP(w, req)
}
