package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// handleGatewayFlowReport receives network flow data from gateway agents
func (s *Server) handleGatewayFlowReport(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Token string `json:"token" binding:"required"`
		Flows []struct {
			UserID        string `json:"userId"`
			UserEmail     string `json:"userEmail"`
			SourceIP      string `json:"sourceIp"`
			DestIP        string `json:"destIp"`
			DestPort      int    `json:"destPort"`
			Protocol      string `json:"protocol"`
			BytesSent     int64  `json:"bytesSent"`
			BytesReceived int64  `json:"bytesReceived"`
			FlowStart     string `json:"flowStart"`
			FlowEnd       string `json:"flowEnd,omitempty"`
		} `json:"flows"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate gateway token
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Convert to flow log entries
	var flows []*db.NetworkFlowLog
	for _, f := range req.Flows {
		flowStart, _ := time.Parse(time.RFC3339, f.FlowStart)
		if flowStart.IsZero() {
			flowStart = time.Now()
		}
		var flowEnd *time.Time
		if f.FlowEnd != "" {
			if t, err := time.Parse(time.RFC3339, f.FlowEnd); err == nil {
				flowEnd = &t
			}
		}

		flows = append(flows, &db.NetworkFlowLog{
			GatewayID:     gateway.ID,
			GatewayName:   gateway.Name,
			UserID:        f.UserID,
			UserEmail:     f.UserEmail,
			SourceIP:      f.SourceIP,
			DestIP:        f.DestIP,
			DestPort:      f.DestPort,
			Protocol:      f.Protocol,
			BytesSent:     f.BytesSent,
			BytesReceived: f.BytesReceived,
			FlowStart:     flowStart,
			FlowEnd:       flowEnd,
		})
	}

	if len(flows) > 0 {
		if err := s.flowStore.BatchInsert(ctx, flows); err != nil {
			s.logger.Error("Failed to store flow logs", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store flows"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"stored": len(flows)})
}

// handleListFlowLogs lists network flow logs with filtering
func (s *Server) handleListFlowLogs(c *gin.Context) {
	filter := db.NetworkFlowFilter{
		GatewayID: c.Query("gateway_id"),
		UserEmail: c.Query("user_email"),
		DestIP:    c.Query("dest_ip"),
		Protocol:  c.Query("protocol"),
		Limit:     100,
	}
	if l := c.Query("limit"); l != "" {
		if v, _ := strconv.Atoi(l); v > 0 && v <= 500 {
			filter.Limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, _ := strconv.Atoi(o); v >= 0 {
			filter.Offset = v
		}
	}

	logs, err := s.flowStore.List(c.Request.Context(), filter)
	if err != nil {
		s.logger.Error("Failed to list flow logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flow logs"})
		return
	}

	result := make([]gin.H, 0, len(logs))
	for _, f := range logs {
		item := gin.H{
			"id":             f.ID,
			"gateway_name":   f.GatewayName,
			"user_email":     f.UserEmail,
			"source_ip":      f.SourceIP,
			"dest_ip":        f.DestIP,
			"dest_port":      f.DestPort,
			"protocol":       f.Protocol,
			"bytes_sent":     f.BytesSent,
			"bytes_received": f.BytesReceived,
			"flow_start":     f.FlowStart.Format(time.RFC3339),
			"created_at":     f.CreatedAt.Format(time.RFC3339),
		}
		if f.FlowEnd != nil {
			item["flow_end"] = f.FlowEnd.Format(time.RFC3339)
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"flows": result})
}

// handleGetFlowStats returns network flow statistics
func (s *Server) handleGetFlowStats(c *gin.Context) {
	stats, err := s.flowStore.GetStats(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to get flow stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// handleGetTopDestinations returns the most accessed destinations
func (s *Server) handleGetTopDestinations(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if v, _ := strconv.Atoi(l); v > 0 && v <= 100 {
			limit = v
		}
	}

	destinations, err := s.flowStore.GetTopDestinations(c.Request.Context(), limit)
	if err != nil {
		s.logger.Error("Failed to get top destinations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top destinations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"destinations": destinations})
}

// handleGetUserFlowActivity returns flow activity for a specific user
func (s *Server) handleGetUserFlowActivity(c *gin.Context) {
	userID := c.Param("userId")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, _ := strconv.Atoi(l); v > 0 && v <= 500 {
			limit = v
		}
	}

	flows, err := s.flowStore.GetUserActivity(c.Request.Context(), userID, limit)
	if err != nil {
		s.logger.Error("Failed to get user flow activity", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user activity"})
		return
	}

	result := make([]gin.H, 0, len(flows))
	for _, f := range flows {
		item := gin.H{
			"id":             f.ID,
			"gateway_name":   f.GatewayName,
			"dest_ip":        f.DestIP,
			"dest_port":      f.DestPort,
			"protocol":       f.Protocol,
			"bytes_sent":     f.BytesSent,
			"bytes_received": f.BytesReceived,
			"flow_start":     f.FlowStart.Format(time.RFC3339),
			"created_at":     f.CreatedAt.Format(time.RFC3339),
		}
		if f.FlowEnd != nil {
			item["flow_end"] = f.FlowEnd.Format(time.RFC3339)
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"flows": result})
}
