package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCANotFound = errors.New("CA not found in database")
)

// CA Status constants
const (
	CAStatusActive  = "active"  // Currently issuing certificates
	CAStatusPending = "pending" // Generated but not yet activated (for rotation)
	CAStatusRetired = "retired" // No longer issuing, but still trusted for verification
	CAStatusRevoked = "revoked" // Revoked, no longer trusted
)

// StoredCA represents a CA stored in the database.
type StoredCA struct {
	ID             string
	CertificatePEM string
	PrivateKeyPEM  string
	SerialNumber   string
	NotBefore      time.Time
	NotAfter       time.Time
	Status         string
	Fingerprint    string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CARotationEvent represents a CA rotation audit event.
type CARotationEvent struct {
	ID             string
	CAID           string
	EventType      string
	OldFingerprint string
	NewFingerprint string
	InitiatedBy    string
	Notes          string
	CreatedAt      time.Time
}

// PKIStore handles PKI persistence.
type PKIStore struct {
	db *DB
}

// NewPKIStore creates a new PKI store.
func NewPKIStore(db *DB) *PKIStore {
	return &PKIStore{db: db}
}

// GetCA retrieves the active CA from the database.
func (s *PKIStore) GetCA(ctx context.Context) (*StoredCA, error) {
	var ca StoredCA
	var status, fingerprint, description *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, certificate_pem, private_key_pem, serial_number, not_before, not_after,
		       COALESCE(status, 'active'), fingerprint, description, created_at, updated_at
		FROM pki_ca
		WHERE id = 'default' OR status = 'active'
		ORDER BY CASE WHEN status = 'active' THEN 0 ELSE 1 END
		LIMIT 1
	`).Scan(&ca.ID, &ca.CertificatePEM, &ca.PrivateKeyPEM, &ca.SerialNumber,
		&ca.NotBefore, &ca.NotAfter, &status, &fingerprint, &description, &ca.CreatedAt, &ca.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrCANotFound
	}
	if err != nil {
		return nil, err
	}
	if status != nil {
		ca.Status = *status
	}
	if fingerprint != nil {
		ca.Fingerprint = *fingerprint
	}
	if description != nil {
		ca.Description = *description
	}
	return &ca, nil
}

// GetCAByID retrieves a specific CA by ID.
func (s *PKIStore) GetCAByID(ctx context.Context, id string) (*StoredCA, error) {
	var ca StoredCA
	var status, fingerprint, description *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, certificate_pem, private_key_pem, serial_number, not_before, not_after,
		       status, fingerprint, description, created_at, updated_at
		FROM pki_ca
		WHERE id = $1
	`, id).Scan(&ca.ID, &ca.CertificatePEM, &ca.PrivateKeyPEM, &ca.SerialNumber,
		&ca.NotBefore, &ca.NotAfter, &status, &fingerprint, &description, &ca.CreatedAt, &ca.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrCANotFound
	}
	if err != nil {
		return nil, err
	}
	if status != nil {
		ca.Status = *status
	}
	if fingerprint != nil {
		ca.Fingerprint = *fingerprint
	}
	if description != nil {
		ca.Description = *description
	}
	return &ca, nil
}

// ListCAs retrieves all CAs from the database.
func (s *PKIStore) ListCAs(ctx context.Context) ([]*StoredCA, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, certificate_pem, private_key_pem, serial_number, not_before, not_after,
		       COALESCE(status, 'active'), fingerprint, description, created_at, updated_at
		FROM pki_ca
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cas []*StoredCA
	for rows.Next() {
		var ca StoredCA
		var status, fingerprint, description *string
		if err := rows.Scan(&ca.ID, &ca.CertificatePEM, &ca.PrivateKeyPEM, &ca.SerialNumber,
			&ca.NotBefore, &ca.NotAfter, &status, &fingerprint, &description, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, err
		}
		if status != nil {
			ca.Status = *status
		}
		if fingerprint != nil {
			ca.Fingerprint = *fingerprint
		}
		if description != nil {
			ca.Description = *description
		}
		cas = append(cas, &ca)
	}
	return cas, rows.Err()
}

// GetTrustedCAs returns all CAs that should be trusted (active + retired during grace period).
func (s *PKIStore) GetTrustedCAs(ctx context.Context) ([]*StoredCA, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, certificate_pem, private_key_pem, serial_number, not_before, not_after,
		       COALESCE(status, 'active'), fingerprint, description, created_at, updated_at
		FROM pki_ca
		WHERE status IN ('active', 'retired', 'pending')
		ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cas []*StoredCA
	for rows.Next() {
		var ca StoredCA
		var status, fingerprint, description *string
		if err := rows.Scan(&ca.ID, &ca.CertificatePEM, &ca.PrivateKeyPEM, &ca.SerialNumber,
			&ca.NotBefore, &ca.NotAfter, &status, &fingerprint, &description, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, err
		}
		if status != nil {
			ca.Status = *status
		}
		if fingerprint != nil {
			ca.Fingerprint = *fingerprint
		}
		if description != nil {
			ca.Description = *description
		}
		cas = append(cas, &ca)
	}
	return cas, rows.Err()
}

// SaveCA saves the CA to the database.
func (s *PKIStore) SaveCA(ctx context.Context, ca *StoredCA) error {
	// Calculate fingerprint from certificate PEM
	fingerprint := calculateFingerprint(ca.CertificatePEM)

	status := ca.Status
	if status == "" {
		status = CAStatusActive
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO pki_ca (id, certificate_pem, private_key_pem, serial_number, not_before, not_after, status, fingerprint, description)
		VALUES ('default', $1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			certificate_pem = EXCLUDED.certificate_pem,
			private_key_pem = EXCLUDED.private_key_pem,
			serial_number = EXCLUDED.serial_number,
			not_before = EXCLUDED.not_before,
			not_after = EXCLUDED.not_after,
			status = EXCLUDED.status,
			fingerprint = EXCLUDED.fingerprint,
			description = EXCLUDED.description,
			updated_at = NOW()
	`, ca.CertificatePEM, ca.PrivateKeyPEM, ca.SerialNumber, ca.NotBefore, ca.NotAfter, status, fingerprint, ca.Description)
	return err
}

// SaveCAWithID saves a CA with a specific ID (for rotation).
func (s *PKIStore) SaveCAWithID(ctx context.Context, ca *StoredCA) error {
	fingerprint := calculateFingerprint(ca.CertificatePEM)

	status := ca.Status
	if status == "" {
		status = CAStatusPending
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO pki_ca (id, certificate_pem, private_key_pem, serial_number, not_before, not_after, status, fingerprint, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			certificate_pem = EXCLUDED.certificate_pem,
			private_key_pem = EXCLUDED.private_key_pem,
			serial_number = EXCLUDED.serial_number,
			not_before = EXCLUDED.not_before,
			not_after = EXCLUDED.not_after,
			status = EXCLUDED.status,
			fingerprint = EXCLUDED.fingerprint,
			description = EXCLUDED.description,
			updated_at = NOW()
	`, ca.ID, ca.CertificatePEM, ca.PrivateKeyPEM, ca.SerialNumber, ca.NotBefore, ca.NotAfter, status, fingerprint, ca.Description)
	return err
}

// UpdateCAStatus updates the status of a CA.
func (s *PKIStore) UpdateCAStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE pki_ca SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	return err
}

// ActivateCA activates a pending CA and retires the current active CA.
func (s *PKIStore) ActivateCA(ctx context.Context, newCAID string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Get current active CA fingerprint for audit
	var oldFingerprint string
	err = tx.QueryRow(ctx, `SELECT COALESCE(fingerprint, '') FROM pki_ca WHERE status = 'active' LIMIT 1`).Scan(&oldFingerprint)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	// Retire currently active CA
	_, err = tx.Exec(ctx, `UPDATE pki_ca SET status = 'retired', updated_at = NOW() WHERE status = 'active'`)
	if err != nil {
		return err
	}

	// Activate the new CA
	_, err = tx.Exec(ctx, `UPDATE pki_ca SET status = 'active', updated_at = NOW() WHERE id = $1`, newCAID)
	if err != nil {
		return err
	}

	// Get new CA fingerprint for audit
	var newFingerprint string
	err = tx.QueryRow(ctx, `SELECT COALESCE(fingerprint, '') FROM pki_ca WHERE id = $1`, newCAID).Scan(&newFingerprint)
	if err != nil {
		return err
	}

	// Record rotation event
	_, err = tx.Exec(ctx, `
		INSERT INTO ca_rotation_events (ca_id, event_type, old_fingerprint, new_fingerprint, notes)
		VALUES ($1, 'activated', $2, $3, 'CA rotation completed')
	`, newCAID, oldFingerprint, newFingerprint)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RevokeCA marks a CA as revoked.
func (s *PKIStore) RevokeCA(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE pki_ca SET status = 'revoked', updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// RecordRotationEvent records a CA rotation event for audit.
func (s *PKIStore) RecordRotationEvent(ctx context.Context, event *CARotationEvent) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO ca_rotation_events (ca_id, event_type, old_fingerprint, new_fingerprint, initiated_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.CAID, event.EventType, event.OldFingerprint, event.NewFingerprint, event.InitiatedBy, event.Notes)
	return err
}

// GetCAFingerprint returns the fingerprint of the active CA.
func (s *PKIStore) GetCAFingerprint(ctx context.Context) (string, error) {
	var fingerprint string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(fingerprint, '') FROM pki_ca WHERE status = 'active' LIMIT 1
	`).Scan(&fingerprint)
	if err == pgx.ErrNoRows {
		return "", ErrCANotFound
	}
	return fingerprint, err
}

// calculateFingerprint calculates SHA256 fingerprint from PEM certificate.
func calculateFingerprint(certPEM string) string {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return ""
	}
	hash := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(hash[:])
}
