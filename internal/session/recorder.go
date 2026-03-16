package session

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RecordingConfig holds configuration for the recorder
type RecordingConfig struct {
	StorageDir      string // Base directory for recordings
	Enabled         bool
	MaskPatterns    []string // Regex patterns to mask in output
	TerminalEnabled bool
}

// ActiveRecording represents an in-progress recording
type ActiveRecording struct {
	ID        string
	File      *os.File
	GzWriter  *gzip.Writer
	StartTime time.Time
	ByteCount int64
	mutex     sync.Mutex
}

// Recorder manages session recordings
type Recorder struct {
	Config       RecordingConfig
	recordings   map[string]*ActiveRecording // adminSessionID -> recording
	mutex        sync.RWMutex
	maskPatterns []*regexp.Regexp
	logger       *zap.Logger
}

// NewRecorder creates a new recorder
func NewRecorder(config RecordingConfig, logger *zap.Logger) *Recorder {
	return &Recorder{
		Config:     config,
		recordings: make(map[string]*ActiveRecording),
		logger:     logger,
	}
}

// StartRecording begins recording a session. Returns the storage path.
func (r *Recorder) StartRecording(adminSessionID, recordingID, userEmail, targetNodeName string) (string, error) {
	if !r.Config.Enabled || !r.Config.TerminalEnabled {
		return "", nil
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(r.Config.StorageDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create file path: <storageDir>/<recordingID>.cast.gz
	fileName := fmt.Sprintf("%s.cast.gz", recordingID)
	filePath := filepath.Join(r.Config.StorageDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create recording file: %w", err)
	}

	gzWriter := gzip.NewWriter(file)

	// Write asciicast v2 header
	header := map[string]interface{}{
		"version":   2,
		"width":     120,
		"height":    40,
		"timestamp": time.Now().Unix(),
		"title":     fmt.Sprintf("Session: %s on %s", userEmail, targetNodeName),
		"env": map[string]string{
			"TERM": "xterm-256color",
		},
	}
	headerBytes, _ := json.Marshal(header)
	gzWriter.Write(headerBytes)
	gzWriter.Write([]byte("\n"))

	rec := &ActiveRecording{
		ID:        recordingID,
		File:      file,
		GzWriter:  gzWriter,
		StartTime: time.Now(),
	}

	r.mutex.Lock()
	r.recordings[adminSessionID] = rec
	r.mutex.Unlock()

	r.logger.Info("Started session recording",
		zap.String("recording_id", recordingID),
		zap.String("user", userEmail),
		zap.String("target", targetNodeName),
		zap.String("path", filePath))

	return filePath, nil
}

// WriteOutput writes an output event to the recording
func (r *Recorder) WriteOutput(adminSessionID, output string) {
	r.mutex.RLock()
	rec, exists := r.recordings[adminSessionID]
	r.mutex.RUnlock()

	if !exists {
		return
	}

	rec.mutex.Lock()
	defer rec.mutex.Unlock()

	elapsed := time.Since(rec.StartTime).Seconds()
	// Asciicast v2 event: [elapsed, "o", "output"]
	maskedOutput := r.maskOutput(output)
	event := []interface{}{elapsed, "o", maskedOutput}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return
	}
	eventBytes = append(eventBytes, '\n')
	n, _ := rec.GzWriter.Write(eventBytes)
	rec.ByteCount += int64(n)
}

// StopRecording stops recording and returns file size and duration
func (r *Recorder) StopRecording(adminSessionID string) (fileSize int64, durationSec int, err error) {
	r.mutex.Lock()
	rec, exists := r.recordings[adminSessionID]
	if exists {
		delete(r.recordings, adminSessionID)
	}
	r.mutex.Unlock()

	if !exists {
		return 0, 0, nil
	}

	rec.mutex.Lock()
	defer rec.mutex.Unlock()

	duration := int(time.Since(rec.StartTime).Seconds())

	// Close gzip writer and file
	if err := rec.GzWriter.Close(); err != nil {
		r.logger.Warn("Failed to close gzip writer", zap.Error(err))
	}

	// Get file size
	stat, statErr := rec.File.Stat()
	var size int64
	if statErr == nil {
		size = stat.Size()
	}

	if err := rec.File.Close(); err != nil {
		r.logger.Warn("Failed to close recording file", zap.Error(err))
	}

	r.logger.Info("Stopped session recording",
		zap.String("recording_id", rec.ID),
		zap.Int64("file_size", size),
		zap.Int("duration_seconds", duration))

	return size, duration, nil
}

// SetMaskPatterns compiles and sets the mask patterns
func (r *Recorder) SetMaskPatterns(patterns []string) {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		} else {
			r.logger.Warn("Invalid mask pattern, skipping", zap.Error(err))
		}
	}
	r.mutex.Lock()
	r.maskPatterns = compiled
	r.mutex.Unlock()
}

// maskOutput applies masking patterns to output text
func (r *Recorder) maskOutput(output string) string {
	r.mutex.RLock()
	patterns := r.maskPatterns
	r.mutex.RUnlock()

	for _, re := range patterns {
		output = re.ReplaceAllString(output, "[REDACTED]")
	}
	return output
}

// IsRecording checks if a session is being recorded
func (r *Recorder) IsRecording(adminSessionID string) bool {
	r.mutex.RLock()
	_, exists := r.recordings[adminSessionID]
	r.mutex.RUnlock()
	return exists
}
