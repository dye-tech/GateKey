package db

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrGeoFenceRuleNotFound = errors.New("geo-fence rule not found")
)

// GeoFenceRule represents an allowed IP range for geo-fencing
type GeoFenceRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IPRange     string    `json:"ipRange"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GeoFenceStore handles geo-fence rule persistence
type GeoFenceStore struct {
	db *DB
}

// NewGeoFenceStore creates a new geo-fence store
func NewGeoFenceStore(db *DB) *GeoFenceStore {
	return &GeoFenceStore{db: db}
}

// CreateRule creates a new geo-fence rule
func (s *GeoFenceStore) CreateRule(ctx context.Context, name, description, ipRange string) (*GeoFenceRule, error) {
	var rule GeoFenceRule
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO geo_fence_rules (name, description, ip_range, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id, name, description, ip_range::text, is_active, created_at, updated_at
	`, name, description, ipRange).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.IPRange, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return &rule, err
}

// GetRule retrieves a geo-fence rule by ID
func (s *GeoFenceStore) GetRule(ctx context.Context, id string) (*GeoFenceRule, error) {
	var rule GeoFenceRule
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, ip_range::text, is_active, created_at, updated_at
		FROM geo_fence_rules WHERE id = $1
	`, id).Scan(&rule.ID, &rule.Name, &rule.Description, &rule.IPRange, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrGeoFenceRuleNotFound
	}
	return &rule, err
}

// ListRules retrieves all geo-fence rules
func (s *GeoFenceStore) ListRules(ctx context.Context) ([]*GeoFenceRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description, ip_range::text, is_active, created_at, updated_at
		FROM geo_fence_rules ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*GeoFenceRule
	for rows.Next() {
		var r GeoFenceRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.IPRange, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// UpdateRule updates a geo-fence rule
func (s *GeoFenceStore) UpdateRule(ctx context.Context, id, name, description, ipRange string, isActive bool) (*GeoFenceRule, error) {
	var rule GeoFenceRule
	err := s.db.Pool.QueryRow(ctx, `
		UPDATE geo_fence_rules
		SET name = $2, description = $3, ip_range = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, ip_range::text, is_active, created_at, updated_at
	`, id, name, description, ipRange, isActive).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.IPRange, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrGeoFenceRuleNotFound
	}
	return &rule, err
}

// DeleteRule deletes a geo-fence rule
func (s *GeoFenceStore) DeleteRule(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM geo_fence_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGeoFenceRuleNotFound
	}
	return nil
}

// --- Global Rule Assignments ---

// AddGlobalRule adds a rule to global geo-fencing
func (s *GeoFenceStore) AddGlobalRule(ctx context.Context, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO geo_fence_global (rule_id) VALUES ($1)
		ON CONFLICT (rule_id) DO NOTHING
	`, ruleID)
	return err
}

// RemoveGlobalRule removes a rule from global geo-fencing
func (s *GeoFenceStore) RemoveGlobalRule(ctx context.Context, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM geo_fence_global WHERE rule_id = $1`, ruleID)
	return err
}

// ListGlobalRules lists all global geo-fence rules
func (s *GeoFenceStore) ListGlobalRules(ctx context.Context) ([]*GeoFenceRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT r.id, r.name, r.description, r.ip_range::text, r.is_active, r.created_at, r.updated_at
		FROM geo_fence_rules r
		JOIN geo_fence_global g ON r.id = g.rule_id
		ORDER BY r.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*GeoFenceRule
	for rows.Next() {
		var r GeoFenceRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.IPRange, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// --- User Rule Assignments ---

// AddUserRule adds a geo-fence rule to a user
func (s *GeoFenceStore) AddUserRule(ctx context.Context, userID, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO user_geo_fence_rules (user_id, rule_id) VALUES ($1, $2)
		ON CONFLICT (user_id, rule_id) DO NOTHING
	`, userID, ruleID)
	return err
}

// RemoveUserRule removes a geo-fence rule from a user
func (s *GeoFenceStore) RemoveUserRule(ctx context.Context, userID, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM user_geo_fence_rules WHERE user_id = $1 AND rule_id = $2`, userID, ruleID)
	return err
}

// ListUserRules lists all geo-fence rules for a user
func (s *GeoFenceStore) ListUserRules(ctx context.Context, userID string) ([]*GeoFenceRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT r.id, r.name, r.description, r.ip_range::text, r.is_active, r.created_at, r.updated_at
		FROM geo_fence_rules r
		JOIN user_geo_fence_rules u ON r.id = u.rule_id
		WHERE u.user_id = $1
		ORDER BY r.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*GeoFenceRule
	for rows.Next() {
		var r GeoFenceRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.IPRange, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// --- Group Rule Assignments ---

// AddGroupRule adds a geo-fence rule to a group
func (s *GeoFenceStore) AddGroupRule(ctx context.Context, groupName, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO group_geo_fence_rules (group_name, rule_id) VALUES ($1, $2)
		ON CONFLICT (group_name, rule_id) DO NOTHING
	`, groupName, ruleID)
	return err
}

// RemoveGroupRule removes a geo-fence rule from a group
func (s *GeoFenceStore) RemoveGroupRule(ctx context.Context, groupName, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM group_geo_fence_rules WHERE group_name = $1 AND rule_id = $2`, groupName, ruleID)
	return err
}

// ListGroupRules lists all geo-fence rules for a group
func (s *GeoFenceStore) ListGroupRules(ctx context.Context, groupName string) ([]*GeoFenceRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT r.id, r.name, r.description, r.ip_range::text, r.is_active, r.created_at, r.updated_at
		FROM geo_fence_rules r
		JOIN group_geo_fence_rules g ON r.id = g.rule_id
		WHERE g.group_name = $1
		ORDER BY r.name
	`, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*GeoFenceRule
	for rows.Next() {
		var r GeoFenceRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.IPRange, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// --- Core Geo-Fencing Logic ---

// GetEffectiveRules returns the effective geo-fence rules for a user
// Priority: User rules > Group rules > Global rules
func (s *GeoFenceStore) GetEffectiveRules(ctx context.Context, userID string, groups []string) ([]*GeoFenceRule, error) {
	// 1. Check user-specific rules
	userRules, err := s.ListUserRules(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(userRules) > 0 {
		return userRules, nil
	}

	// 2. Check group rules
	var groupRules []*GeoFenceRule
	seen := make(map[string]bool)
	for _, group := range groups {
		rules, err := s.ListGroupRules(ctx, group)
		if err != nil {
			return nil, err
		}
		for _, r := range rules {
			if !seen[r.ID] {
				seen[r.ID] = true
				groupRules = append(groupRules, r)
			}
		}
	}
	if len(groupRules) > 0 {
		return groupRules, nil
	}

	// 3. Fall back to global rules
	return s.ListGlobalRules(ctx)
}

// IsIPAllowed checks if a client IP is allowed based on geo-fence rules
// Returns true if the IP matches any active rule, false otherwise
// If no rules exist, returns false (whitelist model - deny by default)
func (s *GeoFenceStore) IsIPAllowed(ctx context.Context, clientIP, userID string, groups []string) (bool, error) {
	rules, err := s.GetEffectiveRules(ctx, userID, groups)
	if err != nil {
		return false, err
	}

	// No rules = no access (whitelist model)
	if len(rules) == 0 {
		return false, nil
	}

	// Parse client IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false, errors.New("invalid client IP address")
	}

	// Check if IP matches any allowed range
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		// Support multiple comma-separated CIDRs per rule
		cidrs := strings.Split(rule.IPRange, ",")
		for _, cidr := range cidrs {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if ipNet.Contains(ip) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetRuleAssignmentCounts returns usage counts for a rule
func (s *GeoFenceStore) GetRuleAssignmentCounts(ctx context.Context, ruleID string) (global bool, userCount, groupCount int, err error) {
	// Check global
	var globalCount int
	err = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM geo_fence_global WHERE rule_id = $1`, ruleID).Scan(&globalCount)
	if err != nil {
		return
	}
	global = globalCount > 0

	// Count users
	err = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_geo_fence_rules WHERE rule_id = $1`, ruleID).Scan(&userCount)
	if err != nil {
		return
	}

	// Count groups
	err = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_geo_fence_rules WHERE rule_id = $1`, ruleID).Scan(&groupCount)
	return
}
