package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrDNSRuleNotFound   = errors.New("dns rule not found")
	ErrDNSRecordNotFound = errors.New("dns record not found")
)

// DNSRule represents DNS configuration for a network
type DNSRule struct {
	ID            string
	NetworkID     string
	DNSServers    []string
	SearchDomains []string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DNSRecord represents a static DNS record for a network
type DNSRecord struct {
	ID          string
	NetworkID   string
	Hostname    string
	IPAddress   string
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

// DNSStore handles DNS rule and record persistence
type DNSStore struct {
	db *DB
}

// NewDNSStore creates a new DNS store
func NewDNSStore(db *DB) *DNSStore {
	return &DNSStore{db: db}
}

// GetRuleByNetwork retrieves the DNS rule for a network
func (s *DNSStore) GetRuleByNetwork(ctx context.Context, networkID string) (*DNSRule, error) {
	var rule DNSRule
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, network_id, dns_servers, search_domains, is_active, created_at, updated_at
		FROM dns_rules WHERE network_id = $1
	`, networkID).Scan(
		&rule.ID, &rule.NetworkID, &rule.DNSServers, &rule.SearchDomains,
		&rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrDNSRuleNotFound
	}
	return &rule, err
}

// UpsertRule inserts or updates a DNS rule for a network
func (s *DNSStore) UpsertRule(ctx context.Context, rule *DNSRule) error {
	return s.db.Pool.QueryRow(ctx, `
		INSERT INTO dns_rules (network_id, dns_servers, search_domains, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (network_id) DO UPDATE SET
			dns_servers = EXCLUDED.dns_servers,
			search_domains = EXCLUDED.search_domains,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`, rule.NetworkID, rule.DNSServers, rule.SearchDomains, rule.IsActive).Scan(
		&rule.ID, &rule.CreatedAt, &rule.UpdatedAt,
	)
}

// DeleteRule deletes the DNS rule for a network
func (s *DNSStore) DeleteRule(ctx context.Context, networkID string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM dns_rules WHERE network_id = $1`, networkID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDNSRuleNotFound
	}
	return nil
}

// GetRulesForNetworks retrieves DNS rules for multiple networks
func (s *DNSStore) GetRulesForNetworks(ctx context.Context, networkIDs []string) ([]*DNSRule, error) {
	if len(networkIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, network_id, dns_servers, search_domains, is_active, created_at, updated_at
		FROM dns_rules
		WHERE network_id = ANY($1) AND is_active = true
	`, networkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*DNSRule
	for rows.Next() {
		var r DNSRule
		if err := rows.Scan(&r.ID, &r.NetworkID, &r.DNSServers, &r.SearchDomains,
			&r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// ListRecords lists DNS records for a network
func (s *DNSStore) ListRecords(ctx context.Context, networkID string) ([]*DNSRecord, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, network_id, hostname, host(ip_address), description, is_active, created_at
		FROM dns_records WHERE network_id = $1
		ORDER BY hostname
	`, networkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*DNSRecord
	for rows.Next() {
		var r DNSRecord
		var desc *string
		if err := rows.Scan(&r.ID, &r.NetworkID, &r.Hostname, &r.IPAddress,
			&desc, &r.IsActive, &r.CreatedAt); err != nil {
			return nil, err
		}
		if desc != nil {
			r.Description = *desc
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

// CreateRecord creates a new DNS record
func (s *DNSStore) CreateRecord(ctx context.Context, record *DNSRecord) error {
	return s.db.Pool.QueryRow(ctx, `
		INSERT INTO dns_records (network_id, hostname, ip_address, description, is_active)
		VALUES ($1, $2, $3::inet, $4, $5)
		RETURNING id, created_at
	`, record.NetworkID, record.Hostname, record.IPAddress, nilIfEmpty(record.Description), record.IsActive).Scan(
		&record.ID, &record.CreatedAt,
	)
}

// UpdateRecord updates a DNS record
func (s *DNSStore) UpdateRecord(ctx context.Context, record *DNSRecord) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE dns_records SET hostname = $2, ip_address = $3::inet, description = $4, is_active = $5
		WHERE id = $1
	`, record.ID, record.Hostname, record.IPAddress, nilIfEmpty(record.Description), record.IsActive)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDNSRecordNotFound
	}
	return nil
}

// DeleteRecord deletes a DNS record
func (s *DNSStore) DeleteRecord(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM dns_records WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDNSRecordNotFound
	}
	return nil
}

// GetRecordsForNetworks retrieves all active DNS records for multiple networks
func (s *DNSStore) GetRecordsForNetworks(ctx context.Context, networkIDs []string) ([]*DNSRecord, error) {
	if len(networkIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, network_id, hostname, host(ip_address), description, is_active, created_at
		FROM dns_records
		WHERE network_id = ANY($1) AND is_active = true
		ORDER BY hostname
	`, networkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*DNSRecord
	for rows.Next() {
		var r DNSRecord
		var desc *string
		if err := rows.Scan(&r.ID, &r.NetworkID, &r.Hostname, &r.IPAddress,
			&desc, &r.IsActive, &r.CreatedAt); err != nil {
			return nil, err
		}
		if desc != nil {
			r.Description = *desc
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

// nilIfEmpty returns nil if s is empty, otherwise returns a pointer to s
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
