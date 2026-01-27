package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GinMiddleware returns a Gin middleware that collects HTTP metrics.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint to avoid recursion
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		HTTPRequestsInFlight.Inc()

		c.Next()

		HTTPRequestsInFlight.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		// Normalize path to avoid high cardinality
		path := normalizePath(c.FullPath())
		if path == "" {
			path = "unknown"
		}

		RecordHTTPRequest(c.Request.Method, path, status, duration)
	}
}

// normalizePath normalizes URL paths to reduce metric cardinality.
// It replaces dynamic path segments (UUIDs, IDs) with placeholders.
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	// The path from c.FullPath() already has :param placeholders
	// e.g., /api/v1/users/:id instead of /api/v1/users/123
	return path
}
