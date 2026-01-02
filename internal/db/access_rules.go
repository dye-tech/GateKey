package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrAccessRuleNotFound = errors.New("access rule not found")
)

// AccessRuleType defines the type of access rule
type AccessRuleType string

const (
	AccessRuleTypeIP               AccessRuleType = "ip"
	AccessRuleTypeCIDR             AccessRuleType = "cidr"
	AccessRuleTypeHostname         AccessRuleType = "hostname"
	AccessRuleTypeHostnameWildcard AccessRuleType = "hostname_wildcard"
)

// AccessRule represents an IP address or hostname whitelist rule
type AccessRule struct {
	ID          string
	Name        string
	Description string
	RuleType    AccessRuleType
	Value       string  // IP, CIDR, or hostname
	PortRange   *string // Optional: "80", "443", "8000-9000", "*"
	Protocol    *string // Optional: tcp, udp, icmp, *
	NetworkID   *string // Optional: restrict to specific network
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AccessRuleStore handles access rule persistence
type AccessRuleStore struct {
	db *DB
}

// NewAccessRuleStore creates a new access rule store
func NewAccessRuleStore(db *DB) *AccessRuleStore {
	return &AccessRuleStore{db: db}
}

// CreateAccessRule creates a new access rule
func (s *AccessRuleStore) CreateAccessRule(ctx context.Context, rule *AccessRule) error {
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO access_rules (name, description, rule_type, value, port_range, protocol, network_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, rule.Name, rule.Description, rule.RuleType, rule.Value, rule.PortRange, rule.Protocol, rule.NetworkID, rule.IsActive).Scan(
		&rule.ID, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return err
}

// GetAccessRule retrieves an access rule by ID
func (s *AccessRuleStore) GetAccessRule(ctx context.Context, id string) (*AccessRule, error) {
	var rule AccessRule
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, rule_type, value, port_range, protocol, network_id, is_active, created_at, updated_at
		FROM access_rules WHERE id = $1
	`, id).Scan(&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Value,
		&rule.PortRange, &rule.Protocol, &rule.NetworkID, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrAccessRuleNotFound
	}
	return &rule, err
}

// ListAccessRules retrieves all access rules
func (s *AccessRuleStore) ListAccessRules(ctx context.Context) ([]*AccessRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description, rule_type, value, port_range, protocol, network_id, is_active, created_at, updated_at
		FROM access_rules ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AccessRule
	for rows.Next() {
		var r AccessRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Value,
			&r.PortRange, &r.Protocol, &r.NetworkID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// ListAccessRulesByNetwork retrieves access rules for a specific network
func (s *AccessRuleStore) ListAccessRulesByNetwork(ctx context.Context, networkID string) ([]*AccessRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description, rule_type, value, port_range, protocol, network_id, is_active, created_at, updated_at
		FROM access_rules WHERE network_id = $1 OR network_id IS NULL
		ORDER BY name
	`, networkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AccessRule
	for rows.Next() {
		var r AccessRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Value,
			&r.PortRange, &r.Protocol, &r.NetworkID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// UpdateAccessRule updates an access rule
func (s *AccessRuleStore) UpdateAccessRule(ctx context.Context, rule *AccessRule) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE access_rules SET name = $2, description = $3, rule_type = $4, value = $5,
		       port_range = $6, protocol = $7, network_id = $8, is_active = $9
		WHERE id = $1
	`, rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Value,
		rule.PortRange, rule.Protocol, rule.NetworkID, rule.IsActive)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAccessRuleNotFound
	}
	return nil
}

// DeleteAccessRule deletes an access rule
func (s *AccessRuleStore) DeleteAccessRule(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM access_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAccessRuleNotFound
	}
	return nil
}

// AssignRuleToUser assigns an access rule to a user
func (s *AccessRuleStore) AssignRuleToUser(ctx context.Context, userID, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO user_access_rules (user_id, access_rule_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, ruleID)
	return err
}

// RemoveRuleFromUser removes an access rule from a user
func (s *AccessRuleStore) RemoveRuleFromUser(ctx context.Context, userID, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM user_access_rules WHERE user_id = $1 AND access_rule_id = $2
	`, userID, ruleID)
	return err
}

// AssignRuleToGroup assigns an access rule to a group
func (s *AccessRuleStore) AssignRuleToGroup(ctx context.Context, groupName, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO group_access_rules (group_name, access_rule_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupName, ruleID)
	return err
}

// RemoveRuleFromGroup removes an access rule from a group
func (s *AccessRuleStore) RemoveRuleFromGroup(ctx context.Context, groupName, ruleID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM group_access_rules WHERE group_name = $1 AND access_rule_id = $2
	`, groupName, ruleID)
	return err
}

// GetUserAccessRules gets all access rules assigned to a user (directly or via groups)
func (s *AccessRuleStore) GetUserAccessRules(ctx context.Context, userID string, groups []string) ([]*AccessRule, error) {
	// Get rules assigned directly to the user and via their groups
	query := `
		SELECT DISTINCT ar.id, ar.name, ar.description, ar.rule_type, ar.value,
		       ar.port_range, ar.protocol, ar.network_id, ar.is_active, ar.created_at, ar.updated_at
		FROM access_rules ar
		LEFT JOIN user_access_rules uar ON ar.id = uar.access_rule_id AND uar.user_id = $1
		LEFT JOIN group_access_rules gar ON ar.id = gar.access_rule_id
		WHERE ar.is_active = true AND (uar.user_id IS NOT NULL OR gar.group_name = ANY($2))
		ORDER BY ar.name
	`
	rows, err := s.db.Pool.Query(ctx, query, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AccessRule
	for rows.Next() {
		var r AccessRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Value,
			&r.PortRange, &r.Protocol, &r.NetworkID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// GetRuleUsers gets all users assigned to an access rule
func (s *AccessRuleStore) GetRuleUsers(ctx context.Context, ruleID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT user_id FROM user_access_rules WHERE access_rule_id = $1
	`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

// GetRuleGroups gets all groups assigned to an access rule
func (s *AccessRuleStore) GetRuleGroups(ctx context.Context, ruleID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name FROM group_access_rules WHERE access_rule_id = $1
	`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupName string
		if err := rows.Scan(&groupName); err != nil {
			return nil, err
		}
		groups = append(groups, groupName)
	}
	return groups, rows.Err()
}

// GetAllUserAccessRuleAssignments returns all user-to-rule assignments as map[userID][]ruleID
func (s *AccessRuleStore) GetAllUserAccessRuleAssignments(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT user_id, access_rule_id FROM user_access_rules
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var userID, ruleID string
		if err := rows.Scan(&userID, &ruleID); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], ruleID)
	}
	return result, rows.Err()
}

// GetAllGroupAccessRuleAssignments returns all group-to-rule assignments as map[groupName][]ruleID
func (s *AccessRuleStore) GetAllGroupAccessRuleAssignments(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name, access_rule_id FROM group_access_rules
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var groupName, ruleID string
		if err := rows.Scan(&groupName, &ruleID); err != nil {
			return nil, err
		}
		result[groupName] = append(result[groupName], ruleID)
	}
	return result, rows.Err()
}

// GetUserAccessRulesForGateway gets access rules for a user that are associated with networks
// assigned to the specified gateway. This ensures only relevant routes are pushed to clients.
func (s *AccessRuleStore) GetUserAccessRulesForGateway(ctx context.Context, userID string, groups []string, gatewayID string) ([]*AccessRule, error) {
	// Get rules assigned to the user (directly or via groups) that belong to networks
	// assigned to this gateway via gateway_networks
	query := `
		SELECT DISTINCT ar.id, ar.name, ar.description, ar.rule_type, ar.value,
		       ar.port_range, ar.protocol, ar.network_id, ar.is_active, ar.created_at, ar.updated_at
		FROM access_rules ar
		JOIN gateway_networks gn ON ar.network_id = gn.network_id
		LEFT JOIN user_access_rules uar ON ar.id = uar.access_rule_id AND uar.user_id = $1
		LEFT JOIN group_access_rules gar ON ar.id = gar.access_rule_id
		WHERE ar.is_active = true
		AND gn.gateway_id = $3
		AND (uar.user_id IS NOT NULL OR gar.group_name = ANY($2))
		ORDER BY ar.name
	`
	rows, err := s.db.Pool.Query(ctx, query, userID, groups, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AccessRule
	for rows.Next() {
		var r AccessRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Value,
			&r.PortRange, &r.Protocol, &r.NetworkID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}
