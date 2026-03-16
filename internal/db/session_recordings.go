package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SessionRecording represents a recorded session
type SessionRecording struct {
	ID              string
	SessionType     string
	UserID          string
	UserEmail       string
	TargetNodeID    *string
	TargetNodeName  *string
	StoragePath     string
	FileSizeBytes   int64
	DurationSeconds int
	RecordingFormat string
	IsComplete      bool
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

// SessionRecordingStore handles session recording persistence
type SessionRecordingStore struct {
	db *DB
}

// NewSessionRecordingStore creates a new session recording store
func NewSessionRecordingStore(db *DB) *SessionRecordingStore {
	return &SessionRecordingStore{db: db}
}

// Create inserts a new session recording
func (s *SessionRecordingStore) Create(ctx context.Context, rec *SessionRecording) error {
	metadataBytes, err := json.Marshal(rec.Metadata)
	if err != nil {
		metadataBytes = []byte("{}")
	}

	err = s.db.Pool.QueryRow(ctx, `
		INSERT INTO session_recordings (id, session_type, user_id, user_email, target_node_id, target_node_name, storage_path, recording_format, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, rec.ID, rec.SessionType, rec.UserID, rec.UserEmail, rec.TargetNodeID, rec.TargetNodeName, rec.StoragePath, rec.RecordingFormat, metadataBytes).Scan(&rec.ID, &rec.CreatedAt)
	return err
}

// Complete marks a recording as complete
func (s *SessionRecordingStore) Complete(ctx context.Context, id string, fileSize int64, durationSec int) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE session_recordings
		SET is_complete = true, file_size_bytes = $2, duration_seconds = $3, completed_at = NOW()
		WHERE id = $1
	`, id, fileSize, durationSec)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("recording not found: %s", id)
	}
	return nil
}

// Get returns a single session recording by ID
func (s *SessionRecordingStore) Get(ctx context.Context, id string) (*SessionRecording, error) {
	var rec SessionRecording
	var metadataBytes []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, session_type, user_id, user_email, target_node_id, target_node_name,
		       storage_path, file_size_bytes, duration_seconds, recording_format,
		       is_complete, metadata, created_at, completed_at
		FROM session_recordings WHERE id = $1
	`, id).Scan(
		&rec.ID, &rec.SessionType, &rec.UserID, &rec.UserEmail,
		&rec.TargetNodeID, &rec.TargetNodeName, &rec.StoragePath,
		&rec.FileSizeBytes, &rec.DurationSeconds, &rec.RecordingFormat,
		&rec.IsComplete, &metadataBytes, &rec.CreatedAt, &rec.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.Metadata = make(map[string]interface{})
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &rec.Metadata)
	}
	return &rec, nil
}

// List returns session recordings ordered by creation time
func (s *SessionRecordingStore) List(ctx context.Context, limit, offset int) ([]*SessionRecording, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, session_type, user_id, user_email, target_node_id, target_node_name,
		       storage_path, file_size_bytes, duration_seconds, recording_format,
		       is_complete, metadata, created_at, completed_at
		FROM session_recordings
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []*SessionRecording
	for rows.Next() {
		var rec SessionRecording
		var metadataBytes []byte
		if err := rows.Scan(
			&rec.ID, &rec.SessionType, &rec.UserID, &rec.UserEmail,
			&rec.TargetNodeID, &rec.TargetNodeName, &rec.StoragePath,
			&rec.FileSizeBytes, &rec.DurationSeconds, &rec.RecordingFormat,
			&rec.IsComplete, &metadataBytes, &rec.CreatedAt, &rec.CompletedAt,
		); err != nil {
			return nil, err
		}
		rec.Metadata = make(map[string]interface{})
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &rec.Metadata)
		}
		recordings = append(recordings, &rec)
	}
	return recordings, rows.Err()
}

// Delete removes a session recording by ID
func (s *SessionRecordingStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM session_recordings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("recording not found: %s", id)
	}
	return nil
}

// DeleteOlderThan removes recordings older than the specified number of days
func (s *SessionRecordingStore) DeleteOlderThan(ctx context.Context, days int) (int, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM session_recordings
		WHERE created_at < NOW() - ($1 || ' days')::interval
	`, fmt.Sprintf("%d", days))
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
