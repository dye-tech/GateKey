package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCANotFound = errors.New("CA not found in database")
)

// StoredCA represents a CA stored in the database.
type StoredCA struct {
	ID             string
	CertificatePEM string
	PrivateKeyPEM  string
	SerialNumber   string
	NotBefore      time.Time
	NotAfter       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PKIStore handles PKI persistence.
type PKIStore struct {
	db *DB
}

// NewPKIStore creates a new PKI store.
func NewPKIStore(db *DB) *PKIStore {
	return &PKIStore{db: db}
}

// GetCA retrieves the CA from the database.
func (s *PKIStore) GetCA(ctx context.Context) (*StoredCA, error) {
	var ca StoredCA
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, certificate_pem, private_key_pem, serial_number, not_before, not_after, created_at, updated_at
		FROM pki_ca
		WHERE id = 'default'
	`).Scan(&ca.ID, &ca.CertificatePEM, &ca.PrivateKeyPEM, &ca.SerialNumber,
		&ca.NotBefore, &ca.NotAfter, &ca.CreatedAt, &ca.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrCANotFound
	}
	if err != nil {
		return nil, err
	}
	return &ca, nil
}

// SaveCA saves the CA to the database.
func (s *PKIStore) SaveCA(ctx context.Context, ca *StoredCA) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO pki_ca (id, certificate_pem, private_key_pem, serial_number, not_before, not_after)
		VALUES ('default', $1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			certificate_pem = EXCLUDED.certificate_pem,
			private_key_pem = EXCLUDED.private_key_pem,
			serial_number = EXCLUDED.serial_number,
			not_before = EXCLUDED.not_before,
			not_after = EXCLUDED.not_after,
			updated_at = NOW()
	`, ca.CertificatePEM, ca.PrivateKeyPEM, ca.SerialNumber, ca.NotBefore, ca.NotAfter)
	return err
}
