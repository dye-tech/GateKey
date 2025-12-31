// Package models provides repository interfaces and implementations.
package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserRepository handles user database operations.
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	groupsJSON, _ := json.Marshal(user.Groups)

	query := `
		INSERT INTO users (id, external_id, provider, email, name, groups, attributes, is_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		user.ID,
		user.ExternalID,
		user.Provider,
		user.Email,
		user.Name,
		groupsJSON,
		user.Attributes,
		user.IsAdmin,
		user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, external_id, provider, email, name, groups, attributes,
		       is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	var user User
	var groupsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.ExternalID,
		&user.Provider,
		&user.Email,
		&user.Name,
		&groupsJSON,
		&user.Attributes,
		&user.IsAdmin,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(groupsJSON, &user.Groups)
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, external_id, provider, email, name, groups, attributes,
		       is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`

	var user User
	var groupsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.ExternalID,
		&user.Provider,
		&user.Email,
		&user.Name,
		&groupsJSON,
		&user.Attributes,
		&user.IsAdmin,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(groupsJSON, &user.Groups)
	return &user, nil
}

// GetByProviderID retrieves a user by provider and external ID.
func (r *UserRepository) GetByProviderID(ctx context.Context, provider, externalID string) (*User, error) {
	query := `
		SELECT id, external_id, provider, email, name, groups, attributes,
		       is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE provider = $1 AND external_id = $2`

	var user User
	var groupsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, provider, externalID).Scan(
		&user.ID,
		&user.ExternalID,
		&user.Provider,
		&user.Email,
		&user.Name,
		&groupsJSON,
		&user.Attributes,
		&user.IsAdmin,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(groupsJSON, &user.Groups)
	return &user, nil
}

// Upsert creates or updates a user based on provider and external ID.
func (r *UserRepository) Upsert(ctx context.Context, user *User) error {
	groupsJSON, _ := json.Marshal(user.Groups)

	query := `
		INSERT INTO users (id, external_id, provider, email, name, groups, attributes, is_admin, is_active, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (provider, external_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			groups = EXCLUDED.groups,
			attributes = EXCLUDED.attributes,
			last_login_at = EXCLUDED.last_login_at,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		user.ID,
		user.ExternalID,
		user.Provider,
		user.Email,
		user.Name,
		groupsJSON,
		user.Attributes,
		user.IsAdmin,
		user.IsActive,
		user.LastLoginAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// UpdateLastLogin updates the last login timestamp.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// SessionRepository handles session database operations.
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new session repository.
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session.
func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`

	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
	).Scan(&session.CreatedAt)
}

// GetByToken retrieves a session by token hash.
func (r *SessionRepository) GetByToken(ctx context.Context, tokenHash string) (*Session, error) {
	query := `
		SELECT id, user_id, token, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions
		WHERE token = $1 AND revoked_at IS NULL AND expires_at > NOW()`

	var session Session
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.RevokedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Revoke revokes a session.
func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// RevokeAllForUser revokes all sessions for a user.
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Pool.Exec(ctx, query, userID)
	return err
}

// CertificateRepository handles certificate database operations.
type CertificateRepository struct {
	db *DB
}

// NewCertificateRepository creates a new certificate repository.
func NewCertificateRepository(db *DB) *CertificateRepository {
	return &CertificateRepository{db: db}
}

// Create creates a new certificate record.
func (r *CertificateRepository) Create(ctx context.Context, cert *Certificate) error {
	query := `
		INSERT INTO certificates (id, user_id, session_id, serial_number, subject, not_before, not_after, fingerprint)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`

	if cert.ID == uuid.Nil {
		cert.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		cert.ID,
		cert.UserID,
		cert.SessionID,
		cert.SerialNumber,
		cert.Subject,
		cert.NotBefore,
		cert.NotAfter,
		cert.Fingerprint,
	).Scan(&cert.CreatedAt)
}

// GetBySerial retrieves a certificate by serial number.
func (r *CertificateRepository) GetBySerial(ctx context.Context, serial string) (*Certificate, error) {
	query := `
		SELECT id, user_id, session_id, serial_number, subject, not_before, not_after,
		       fingerprint, is_revoked, revoked_at, revocation_reason, created_at
		FROM certificates WHERE serial_number = $1`

	var cert Certificate
	err := r.db.Pool.QueryRow(ctx, query, serial).Scan(
		&cert.ID,
		&cert.UserID,
		&cert.SessionID,
		&cert.SerialNumber,
		&cert.Subject,
		&cert.NotBefore,
		&cert.NotAfter,
		&cert.Fingerprint,
		&cert.IsRevoked,
		&cert.RevokedAt,
		&cert.RevocationReason,
		&cert.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// GetByFingerprint retrieves a certificate by fingerprint.
func (r *CertificateRepository) GetByFingerprint(ctx context.Context, fingerprint string) (*Certificate, error) {
	query := `
		SELECT id, user_id, session_id, serial_number, subject, not_before, not_after,
		       fingerprint, is_revoked, revoked_at, revocation_reason, created_at
		FROM certificates WHERE fingerprint = $1`

	var cert Certificate
	err := r.db.Pool.QueryRow(ctx, query, fingerprint).Scan(
		&cert.ID,
		&cert.UserID,
		&cert.SessionID,
		&cert.SerialNumber,
		&cert.Subject,
		&cert.NotBefore,
		&cert.NotAfter,
		&cert.Fingerprint,
		&cert.IsRevoked,
		&cert.RevokedAt,
		&cert.RevocationReason,
		&cert.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// Revoke revokes a certificate.
func (r *CertificateRepository) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	query := `UPDATE certificates SET is_revoked = TRUE, revoked_at = NOW(), revocation_reason = $2 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id, reason)
	return err
}

// ListRevoked returns all revoked certificates for CRL generation.
func (r *CertificateRepository) ListRevoked(ctx context.Context) ([]Certificate, error) {
	query := `
		SELECT id, user_id, session_id, serial_number, subject, not_before, not_after,
		       fingerprint, is_revoked, revoked_at, revocation_reason, created_at
		FROM certificates WHERE is_revoked = TRUE AND not_after > NOW()`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		var cert Certificate
		err := rows.Scan(
			&cert.ID,
			&cert.UserID,
			&cert.SessionID,
			&cert.SerialNumber,
			&cert.Subject,
			&cert.NotBefore,
			&cert.NotAfter,
			&cert.Fingerprint,
			&cert.IsRevoked,
			&cert.RevokedAt,
			&cert.RevocationReason,
			&cert.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// GatewayRepository handles gateway database operations.
type GatewayRepository struct {
	db *DB
}

// NewGatewayRepository creates a new gateway repository.
func NewGatewayRepository(db *DB) *GatewayRepository {
	return &GatewayRepository{db: db}
}

// Create creates a new gateway.
func (r *GatewayRepository) Create(ctx context.Context, gateway *Gateway) error {
	query := `
		INSERT INTO gateways (id, name, hostname, public_ip, vpn_port, vpn_protocol, token, public_key, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	if gateway.ID == uuid.Nil {
		gateway.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		gateway.ID,
		gateway.Name,
		gateway.Hostname,
		gateway.PublicIP,
		gateway.VPNPort,
		gateway.VPNProtocol,
		gateway.Token,
		gateway.PublicKey,
		gateway.Config,
	).Scan(&gateway.CreatedAt, &gateway.UpdatedAt)
}

// GetByID retrieves a gateway by ID.
func (r *GatewayRepository) GetByID(ctx context.Context, id uuid.UUID) (*Gateway, error) {
	query := `
		SELECT id, name, hostname, public_ip, vpn_port, vpn_protocol, token, public_key,
		       config, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE id = $1`

	var gateway Gateway
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&gateway.ID,
		&gateway.Name,
		&gateway.Hostname,
		&gateway.PublicIP,
		&gateway.VPNPort,
		&gateway.VPNProtocol,
		&gateway.Token,
		&gateway.PublicKey,
		&gateway.Config,
		&gateway.IsActive,
		&gateway.LastHeartbeat,
		&gateway.CreatedAt,
		&gateway.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gateway, nil
}

// GetByName retrieves a gateway by name.
func (r *GatewayRepository) GetByName(ctx context.Context, name string) (*Gateway, error) {
	query := `
		SELECT id, name, hostname, public_ip, vpn_port, vpn_protocol, token, public_key,
		       config, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE name = $1`

	var gateway Gateway
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&gateway.ID,
		&gateway.Name,
		&gateway.Hostname,
		&gateway.PublicIP,
		&gateway.VPNPort,
		&gateway.VPNProtocol,
		&gateway.Token,
		&gateway.PublicKey,
		&gateway.Config,
		&gateway.IsActive,
		&gateway.LastHeartbeat,
		&gateway.CreatedAt,
		&gateway.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gateway, nil
}

// ListActive returns all active gateways.
func (r *GatewayRepository) ListActive(ctx context.Context) ([]Gateway, error) {
	query := `
		SELECT id, name, hostname, public_ip, vpn_port, vpn_protocol, public_key,
		       config, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE is_active = TRUE
		ORDER BY name`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []Gateway
	for rows.Next() {
		var gateway Gateway
		err := rows.Scan(
			&gateway.ID,
			&gateway.Name,
			&gateway.Hostname,
			&gateway.PublicIP,
			&gateway.VPNPort,
			&gateway.VPNProtocol,
			&gateway.PublicKey,
			&gateway.Config,
			&gateway.IsActive,
			&gateway.LastHeartbeat,
			&gateway.CreatedAt,
			&gateway.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}
	return gateways, nil
}

// UpdateHeartbeat updates the last heartbeat timestamp.
func (r *GatewayRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE gateways SET last_heartbeat = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// ValidateToken checks if a gateway token is valid.
func (r *GatewayRepository) ValidateToken(ctx context.Context, tokenHash string) (*Gateway, error) {
	query := `
		SELECT id, name, hostname, public_ip, vpn_port, vpn_protocol, public_key,
		       config, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE token = $1 AND is_active = TRUE`

	var gateway Gateway
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&gateway.ID,
		&gateway.Name,
		&gateway.Hostname,
		&gateway.PublicIP,
		&gateway.VPNPort,
		&gateway.VPNProtocol,
		&gateway.PublicKey,
		&gateway.Config,
		&gateway.IsActive,
		&gateway.LastHeartbeat,
		&gateway.CreatedAt,
		&gateway.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gateway, nil
}

// AuditLogRepository handles audit log database operations.
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository creates a new audit log repository.
func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry.
func (r *AuditLogRepository) Create(ctx context.Context, log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, event, actor_id, actor_email, actor_ip, resource_type, resource_id, details, success)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING timestamp`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	return r.db.Pool.QueryRow(ctx, query,
		log.ID,
		log.Event,
		log.ActorID,
		log.ActorEmail,
		log.ActorIP,
		log.ResourceType,
		log.ResourceID,
		log.Details,
		log.Success,
	).Scan(&log.Timestamp)
}

// List retrieves audit logs with pagination.
func (r *AuditLogRepository) List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]AuditLog, int, error) {
	// Build dynamic query based on filters
	baseQuery := `FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if event, ok := filters["event"].(string); ok && event != "" {
		baseQuery += fmt.Sprintf(" AND event = $%d", argNum)
		args = append(args, event)
		argNum++
	}

	if actorID, ok := filters["actor_id"].(uuid.UUID); ok && actorID != uuid.Nil {
		baseQuery += fmt.Sprintf(" AND actor_id = $%d", argNum)
		args = append(args, actorID)
		argNum++
	}

	if resourceType, ok := filters["resource_type"].(string); ok && resourceType != "" {
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", argNum)
		args = append(args, resourceType)
		argNum++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch records
	selectQuery := `SELECT id, timestamp, event, actor_id, actor_email, actor_ip, resource_type, resource_id, details, success ` +
		baseQuery + fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Event,
			&log.ActorID,
			&log.ActorEmail,
			&log.ActorIP,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.Success,
		)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}
