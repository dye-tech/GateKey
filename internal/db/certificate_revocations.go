package db

import (
	"context"
	"time"
)

// CertificateRevocation represents a revoked certificate entry.
type CertificateRevocation struct {
	ID           string    `json:"id"`
	SerialNumber string    `json:"serial_number"`
	CAID         string    `json:"ca_id"`
	CommonName   string    `json:"common_name"`
	RevokedAt    time.Time `json:"revoked_at"`
	Reason       int       `json:"reason"` // RFC 5280 reason code
	RevokedBy    string    `json:"revoked_by"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

// CRLCache represents a cached CRL.
type CRLCache struct {
	ID         string    `json:"id"`
	CAID       string    `json:"ca_id"`
	CRLDER     []byte    `json:"-"`
	ThisUpdate time.Time `json:"this_update"`
	NextUpdate time.Time `json:"next_update"`
	CRLNumber  int64     `json:"crl_number"`
	CreatedAt  time.Time `json:"created_at"`
}

// CertificateRevocationStore handles certificate revocation persistence.
type CertificateRevocationStore struct {
	db *DB
}

// NewCertificateRevocationStore creates a new certificate revocation store.
func NewCertificateRevocationStore(db *DB) *CertificateRevocationStore {
	return &CertificateRevocationStore{db: db}
}

// RevokeCertificate adds a certificate to the revocation list.
func (s *CertificateRevocationStore) RevokeCertificate(ctx context.Context, rev *CertificateRevocation) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO certificate_revocations (serial_number, ca_id, common_name, revoked_at, reason, revoked_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (serial_number) DO UPDATE SET
			revoked_at = EXCLUDED.revoked_at,
			reason = EXCLUDED.reason,
			revoked_by = EXCLUDED.revoked_by,
			notes = EXCLUDED.notes
	`, rev.SerialNumber, rev.CAID, rev.CommonName, rev.RevokedAt, rev.Reason, rev.RevokedBy, rev.Notes)
	return err
}

// GetRevocation retrieves a specific certificate revocation by serial number.
func (s *CertificateRevocationStore) GetRevocation(ctx context.Context, serialNumber string) (*CertificateRevocation, error) {
	var rev CertificateRevocation
	var commonName, revokedBy, notes *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, serial_number, ca_id, common_name, revoked_at, reason, revoked_by, notes, created_at
		FROM certificate_revocations
		WHERE serial_number = $1
	`, serialNumber).Scan(&rev.ID, &rev.SerialNumber, &rev.CAID, &commonName, &rev.RevokedAt,
		&rev.Reason, &revokedBy, &notes, &rev.CreatedAt)
	if err != nil {
		return nil, err
	}
	if commonName != nil {
		rev.CommonName = *commonName
	}
	if revokedBy != nil {
		rev.RevokedBy = *revokedBy
	}
	if notes != nil {
		rev.Notes = *notes
	}
	return &rev, nil
}

// ListRevocations retrieves all revoked certificates for a CA.
func (s *CertificateRevocationStore) ListRevocations(ctx context.Context, caID string) ([]*CertificateRevocation, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, serial_number, ca_id, common_name, revoked_at, reason, revoked_by, notes, created_at
		FROM certificate_revocations
		WHERE ca_id = $1
		ORDER BY revoked_at DESC
	`, caID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revocations []*CertificateRevocation
	for rows.Next() {
		var rev CertificateRevocation
		var commonName, revokedBy, notes *string
		if err := rows.Scan(&rev.ID, &rev.SerialNumber, &rev.CAID, &commonName, &rev.RevokedAt,
			&rev.Reason, &revokedBy, &notes, &rev.CreatedAt); err != nil {
			return nil, err
		}
		if commonName != nil {
			rev.CommonName = *commonName
		}
		if revokedBy != nil {
			rev.RevokedBy = *revokedBy
		}
		if notes != nil {
			rev.Notes = *notes
		}
		revocations = append(revocations, &rev)
	}
	return revocations, rows.Err()
}

// ListAllRevocations retrieves all revoked certificates across all CAs.
func (s *CertificateRevocationStore) ListAllRevocations(ctx context.Context) ([]*CertificateRevocation, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, serial_number, ca_id, common_name, revoked_at, reason, revoked_by, notes, created_at
		FROM certificate_revocations
		ORDER BY revoked_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revocations []*CertificateRevocation
	for rows.Next() {
		var rev CertificateRevocation
		var commonName, revokedBy, notes *string
		if err := rows.Scan(&rev.ID, &rev.SerialNumber, &rev.CAID, &commonName, &rev.RevokedAt,
			&rev.Reason, &revokedBy, &notes, &rev.CreatedAt); err != nil {
			return nil, err
		}
		if commonName != nil {
			rev.CommonName = *commonName
		}
		if revokedBy != nil {
			rev.RevokedBy = *revokedBy
		}
		if notes != nil {
			rev.Notes = *notes
		}
		revocations = append(revocations, &rev)
	}
	return revocations, rows.Err()
}

// DeleteRevocation removes a certificate from the revocation list.
func (s *CertificateRevocationStore) DeleteRevocation(ctx context.Context, serialNumber string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM certificate_revocations WHERE serial_number = $1
	`, serialNumber)
	return err
}

// IsCertificateRevoked checks if a certificate is revoked.
func (s *CertificateRevocationStore) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM certificate_revocations WHERE serial_number = $1
	`, serialNumber).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SaveCRLCache saves a generated CRL to the cache.
func (s *CertificateRevocationStore) SaveCRLCache(ctx context.Context, cache *CRLCache) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO crl_cache (ca_id, crl_der, this_update, next_update, crl_number)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (ca_id) DO UPDATE SET
			crl_der = EXCLUDED.crl_der,
			this_update = EXCLUDED.this_update,
			next_update = EXCLUDED.next_update,
			crl_number = EXCLUDED.crl_number,
			created_at = NOW()
	`, cache.CAID, cache.CRLDER, cache.ThisUpdate, cache.NextUpdate, cache.CRLNumber)
	return err
}

// GetCRLCache retrieves a cached CRL if it's still valid.
func (s *CertificateRevocationStore) GetCRLCache(ctx context.Context, caID string) (*CRLCache, error) {
	var cache CRLCache
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, ca_id, crl_der, this_update, next_update, crl_number, created_at
		FROM crl_cache
		WHERE ca_id = $1 AND next_update > NOW()
	`, caID).Scan(&cache.ID, &cache.CAID, &cache.CRLDER, &cache.ThisUpdate,
		&cache.NextUpdate, &cache.CRLNumber, &cache.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

// InvalidateCRLCache invalidates the CRL cache for a CA.
func (s *CertificateRevocationStore) InvalidateCRLCache(ctx context.Context, caID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM crl_cache WHERE ca_id = $1`, caID)
	return err
}
