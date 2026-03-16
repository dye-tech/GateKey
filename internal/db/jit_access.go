package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// JITAccessRequest represents a just-in-time access request
type JITAccessRequest struct {
	ID                   string
	RequesterID          string
	RequesterEmail       string
	ResourceType         string
	ResourceID           string
	ResourceName         string
	Justification        string
	RequestedDurationMin int
	Status               string
	ApproverID           *string
	ApproverEmail        *string
	ApprovalNote         *string
	ExpiresAt            time.Time
	CreatedAt            time.Time
	DecidedAt            *time.Time
}

// JITAccessGrant represents an active just-in-time access grant
type JITAccessGrant struct {
	ID           string
	RequestID    string
	UserID       string
	ResourceType string
	ResourceID   string
	IsActive     bool
	GrantedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	RevokedBy    *string
	RevokeReason *string
}

// JITAccessStore handles JIT access request and grant persistence
type JITAccessStore struct {
	db *DB
}

// NewJITAccessStore creates a new JIT access store
func NewJITAccessStore(db *DB) *JITAccessStore {
	return &JITAccessStore{db: db}
}

// CreateRequest inserts a new JIT access request
func (s *JITAccessStore) CreateRequest(ctx context.Context, req *JITAccessRequest) error {
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO jit_access_requests (requester_id, requester_email, resource_type, resource_id,
			resource_name, justification, requested_duration_minutes, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, req.RequesterID, req.RequesterEmail, req.ResourceType, req.ResourceID,
		req.ResourceName, req.Justification, req.RequestedDurationMin, req.Status, req.ExpiresAt).Scan(
		&req.ID, &req.CreatedAt,
	)
	return err
}

// GetRequest retrieves a JIT access request by ID
func (s *JITAccessStore) GetRequest(ctx context.Context, id string) (*JITAccessRequest, error) {
	var req JITAccessRequest
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, requester_id, requester_email, resource_type, resource_id, resource_name,
			justification, requested_duration_minutes, status, approver_id, approver_email,
			approval_note, expires_at, created_at, decided_at
		FROM jit_access_requests WHERE id = $1
	`, id).Scan(&req.ID, &req.RequesterID, &req.RequesterEmail, &req.ResourceType, &req.ResourceID,
		&req.ResourceName, &req.Justification, &req.RequestedDurationMin, &req.Status,
		&req.ApproverID, &req.ApproverEmail, &req.ApprovalNote, &req.ExpiresAt, &req.CreatedAt, &req.DecidedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("JIT access request not found")
	}
	return &req, err
}

// ListRequests lists JIT access requests with optional status filter
func (s *JITAccessStore) ListRequests(ctx context.Context, status string, limit, offset int) ([]*JITAccessRequest, error) {
	var rows pgx.Rows
	var err error

	if status != "" {
		rows, err = s.db.Pool.Query(ctx, `
			SELECT id, requester_id, requester_email, resource_type, resource_id, resource_name,
				justification, requested_duration_minutes, status, approver_id, approver_email,
				approval_note, expires_at, created_at, decided_at
			FROM jit_access_requests WHERE status = $1
			ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`, status, limit, offset)
	} else {
		rows, err = s.db.Pool.Query(ctx, `
			SELECT id, requester_id, requester_email, resource_type, resource_id, resource_name,
				justification, requested_duration_minutes, status, approver_id, approver_email,
				approval_note, expires_at, created_at, decided_at
			FROM jit_access_requests
			ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*JITAccessRequest
	for rows.Next() {
		var r JITAccessRequest
		if err := rows.Scan(&r.ID, &r.RequesterID, &r.RequesterEmail, &r.ResourceType, &r.ResourceID,
			&r.ResourceName, &r.Justification, &r.RequestedDurationMin, &r.Status,
			&r.ApproverID, &r.ApproverEmail, &r.ApprovalNote, &r.ExpiresAt, &r.CreatedAt, &r.DecidedAt); err != nil {
			return nil, err
		}
		requests = append(requests, &r)
	}
	return requests, rows.Err()
}

// ListUserRequests lists JIT access requests for a specific user
func (s *JITAccessStore) ListUserRequests(ctx context.Context, userID string) ([]*JITAccessRequest, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, requester_id, requester_email, resource_type, resource_id, resource_name,
			justification, requested_duration_minutes, status, approver_id, approver_email,
			approval_note, expires_at, created_at, decided_at
		FROM jit_access_requests WHERE requester_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*JITAccessRequest
	for rows.Next() {
		var r JITAccessRequest
		if err := rows.Scan(&r.ID, &r.RequesterID, &r.RequesterEmail, &r.ResourceType, &r.ResourceID,
			&r.ResourceName, &r.Justification, &r.RequestedDurationMin, &r.Status,
			&r.ApproverID, &r.ApproverEmail, &r.ApprovalNote, &r.ExpiresAt, &r.CreatedAt, &r.DecidedAt); err != nil {
			return nil, err
		}
		requests = append(requests, &r)
	}
	return requests, rows.Err()
}

// CancelRequest cancels a pending JIT access request
func (s *JITAccessStore) CancelRequest(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE jit_access_requests SET status = 'cancelled', decided_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("request not found or not in pending status")
	}
	return nil
}

// ApproveRequest approves a JIT access request and creates a grant
func (s *JITAccessStore) ApproveRequest(ctx context.Context, requestID, approverID, approverEmail, note string) (*JITAccessGrant, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update the request status
	var req JITAccessRequest
	err = tx.QueryRow(ctx, `
		UPDATE jit_access_requests
		SET status = 'approved', approver_id = $2, approver_email = $3, approval_note = $4, decided_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING id, requester_id, resource_type, resource_id, requested_duration_minutes
	`, requestID, approverID, approverEmail, note).Scan(
		&req.ID, &req.RequesterID, &req.ResourceType, &req.ResourceID, &req.RequestedDurationMin,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("request not found or not in pending status")
	}
	if err != nil {
		return nil, err
	}

	// Create the grant with expires_at = NOW() + requested_duration_minutes
	var grant JITAccessGrant
	err = tx.QueryRow(ctx, `
		INSERT INTO jit_access_grants (request_id, user_id, resource_type, resource_id, is_active, expires_at)
		VALUES ($1, $2, $3, $4, true, NOW() + ($5 || ' minutes')::interval)
		RETURNING id, request_id, user_id, resource_type, resource_id, is_active, granted_at, expires_at
	`, requestID, req.RequesterID, req.ResourceType, req.ResourceID, fmt.Sprintf("%d", req.RequestedDurationMin)).Scan(
		&grant.ID, &grant.RequestID, &grant.UserID, &grant.ResourceType, &grant.ResourceID,
		&grant.IsActive, &grant.GrantedAt, &grant.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &grant, nil
}

// DenyRequest denies a JIT access request
func (s *JITAccessStore) DenyRequest(ctx context.Context, requestID, approverID, approverEmail, note string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE jit_access_requests
		SET status = 'denied', approver_id = $2, approver_email = $3, approval_note = $4, decided_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, requestID, approverID, approverEmail, note)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("request not found or not in pending status")
	}
	return nil
}

// GetActiveGrantsForUser returns all active grants for a user
func (s *JITAccessStore) GetActiveGrantsForUser(ctx context.Context, userID string) ([]*JITAccessGrant, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, request_id, user_id, resource_type, resource_id, is_active, granted_at, expires_at,
			revoked_at, revoked_by, revoke_reason
		FROM jit_access_grants
		WHERE user_id = $1 AND is_active = true AND expires_at > NOW()
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []*JITAccessGrant
	for rows.Next() {
		var g JITAccessGrant
		if err := rows.Scan(&g.ID, &g.RequestID, &g.UserID, &g.ResourceType, &g.ResourceID,
			&g.IsActive, &g.GrantedAt, &g.ExpiresAt, &g.RevokedAt, &g.RevokedBy, &g.RevokeReason); err != nil {
			return nil, err
		}
		grants = append(grants, &g)
	}
	return grants, rows.Err()
}

// GetActiveGrantForResource checks if a user has an active grant for a specific resource
func (s *JITAccessStore) GetActiveGrantForResource(ctx context.Context, userID, resourceType, resourceID string) (*JITAccessGrant, error) {
	var g JITAccessGrant
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, request_id, user_id, resource_type, resource_id, is_active, granted_at, expires_at,
			revoked_at, revoked_by, revoke_reason
		FROM jit_access_grants
		WHERE user_id = $1 AND resource_type = $2 AND resource_id = $3 AND is_active = true AND expires_at > NOW()
		LIMIT 1
	`, userID, resourceType, resourceID).Scan(&g.ID, &g.RequestID, &g.UserID, &g.ResourceType, &g.ResourceID,
		&g.IsActive, &g.GrantedAt, &g.ExpiresAt, &g.RevokedAt, &g.RevokedBy, &g.RevokeReason)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

// RevokeGrant revokes an active JIT access grant
func (s *JITAccessStore) RevokeGrant(ctx context.Context, grantID, revokedBy, reason string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE jit_access_grants
		SET is_active = false, revoked_at = NOW(), revoked_by = $2, revoke_reason = $3
		WHERE id = $1 AND is_active = true
	`, grantID, revokedBy, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("grant not found or already revoked")
	}
	return nil
}

// RevokeExpiredGrants revokes all expired active grants
func (s *JITAccessStore) RevokeExpiredGrants(ctx context.Context) (int, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE jit_access_grants
		SET is_active = false, revoked_by = 'system', revoke_reason = 'expired'
		WHERE is_active = true AND expires_at <= NOW()
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

// ExpirePendingRequests expires pending requests that have passed their expiry time
func (s *JITAccessStore) ExpirePendingRequests(ctx context.Context) (int, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE jit_access_requests SET status = 'expired'
		WHERE status = 'pending' AND expires_at <= NOW()
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

// GetGrantStats returns counts of pending requests and active grants
func (s *JITAccessStore) GetGrantStats(ctx context.Context) (pending int, active int, err error) {
	err = s.db.Pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM jit_access_requests WHERE status = 'pending'),
			(SELECT COUNT(*) FROM jit_access_grants WHERE is_active = true AND expires_at > NOW())
	`).Scan(&pending, &active)
	return
}
