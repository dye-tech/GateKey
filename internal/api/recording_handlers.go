package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// handleListRecordings lists session recordings
func (s *Server) handleListRecordings(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	recordings, err := s.recordingStore.List(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list recordings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recordings"})
		return
	}

	result := make([]gin.H, 0, len(recordings))
	for _, r := range recordings {
		item := gin.H{
			"id":               r.ID,
			"session_type":     r.SessionType,
			"user_email":       r.UserEmail,
			"target_node_name": r.TargetNodeName,
			"file_size_bytes":  r.FileSizeBytes,
			"duration_seconds": r.DurationSeconds,
			"is_complete":      r.IsComplete,
			"created_at":       r.CreatedAt.Format(time.RFC3339),
		}
		if r.CompletedAt != nil {
			item["completed_at"] = r.CompletedAt.Format(time.RFC3339)
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"recordings": result})
}

// handleGetRecording gets a single recording metadata
func (s *Server) handleGetRecording(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	rec, err := s.recordingStore.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "recording not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               rec.ID,
		"session_type":     rec.SessionType,
		"user_id":          rec.UserID,
		"user_email":       rec.UserEmail,
		"target_node_id":   rec.TargetNodeID,
		"target_node_name": rec.TargetNodeName,
		"storage_path":     rec.StoragePath,
		"file_size_bytes":  rec.FileSizeBytes,
		"duration_seconds": rec.DurationSeconds,
		"is_complete":      rec.IsComplete,
		"created_at":       rec.CreatedAt.Format(time.RFC3339),
		"completed_at":     rec.CompletedAt,
	})
}

// handleStreamRecording streams the recording data for replay
func (s *Server) handleStreamRecording(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	rec, err := s.recordingStore.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "recording not found"})
		return
	}

	file, err := os.Open(rec.StoragePath)
	if err != nil {
		s.logger.Error("Failed to open recording file", zap.Error(err), zap.String("path", rec.StoragePath))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "recording file not found"})
		return
	}
	defer file.Close()

	// Decompress gzip
	gz, err := gzip.NewReader(file)
	if err != nil {
		s.logger.Error("Failed to decompress recording", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read recording"})
		return
	}
	defer gz.Close()

	c.Header("Content-Type", "application/x-asciicast")
	c.Header("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, gz)
}

// handleDeleteRecording deletes a recording
func (s *Server) handleDeleteRecording(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	rec, err := s.recordingStore.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "recording not found"})
		return
	}

	// Delete file
	if rec.StoragePath != "" {
		os.Remove(rec.StoragePath)
	}

	if err := s.recordingStore.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete recording", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recording"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recording deleted"})
}

// handleGetRecordingSettings gets recording settings
func (s *Server) handleGetRecordingSettings(c *gin.Context) {
	ctx := c.Request.Context()
	c.JSON(http.StatusOK, gin.H{
		"enabled":        s.settingsStore.GetString(ctx, db.SettingRecordingEnabled, "false"),
		"storage_path":   s.settingsStore.GetString(ctx, db.SettingRecordingStoragePath, "/var/lib/gatekey/recordings"),
		"retention_days": s.settingsStore.GetInt(ctx, db.SettingRecordingRetentionDays, 90),
	})
}

// handleUpdateRecordingSettings updates recording settings
func (s *Server) handleUpdateRecordingSettings(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		Enabled       *string `json:"enabled"`
		StoragePath   *string `json:"storage_path"`
		RetentionDays *string `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Enabled != nil {
		_ = s.settingsStore.Set(ctx, db.SettingRecordingEnabled, *body.Enabled)
		if s.sessionRecorder != nil {
			s.sessionRecorder.Config.Enabled = *body.Enabled == "true"
		}
	}
	if body.StoragePath != nil {
		_ = s.settingsStore.Set(ctx, db.SettingRecordingStoragePath, *body.StoragePath)
		if s.sessionRecorder != nil {
			s.sessionRecorder.Config.StorageDir = *body.StoragePath
		}
	}
	if body.RetentionDays != nil {
		_ = s.settingsStore.Set(ctx, db.SettingRecordingRetentionDays, *body.RetentionDays)
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}
