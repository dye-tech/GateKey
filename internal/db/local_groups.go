package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// LocalGroupPrefix is prepended to local group names when returned to clients
const LocalGroupPrefix = "local:"

var (
	ErrLocalGroupNotFound = errors.New("local group not found")
	ErrLocalGroupExists   = errors.New("local group already exists")
	ErrMemberNotFound     = errors.New("member not found in group")
	ErrMemberExists       = errors.New("member already in group")
)

// LocalGroup represents an admin-managed local group
type LocalGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LocalGroupMember represents a user's membership in a local group
type LocalGroupMember struct {
	UserID     string    `json:"user_id"`
	MemberType string    `json:"member_type"` // "sso" or "local"
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

// LocalGroupStore handles local group persistence
type LocalGroupStore struct {
	db *DB
}

// NewLocalGroupStore creates a new local group store
func NewLocalGroupStore(db *DB) *LocalGroupStore {
	return &LocalGroupStore{db: db}
}

// CreateLocalGroup creates a new local group
func (s *LocalGroupStore) CreateLocalGroup(ctx context.Context, group *LocalGroup) error {
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO local_groups (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`, group.Name, group.Description).Scan(
		&group.ID, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrLocalGroupExists
	}
	return err
}

// GetLocalGroup retrieves a local group by ID
func (s *LocalGroupStore) GetLocalGroup(ctx context.Context, id string) (*LocalGroup, error) {
	var group LocalGroup
	err := s.db.Pool.QueryRow(ctx, `
		SELECT lg.id, lg.name, lg.description, lg.created_at, lg.updated_at,
		       (SELECT COUNT(*) FROM local_group_members WHERE group_id = lg.id) as member_count
		FROM local_groups lg
		WHERE lg.id = $1
	`, id).Scan(&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt, &group.MemberCount)
	if err == pgx.ErrNoRows {
		return nil, ErrLocalGroupNotFound
	}
	return &group, err
}

// GetLocalGroupByName retrieves a local group by name
func (s *LocalGroupStore) GetLocalGroupByName(ctx context.Context, name string) (*LocalGroup, error) {
	var group LocalGroup
	err := s.db.Pool.QueryRow(ctx, `
		SELECT lg.id, lg.name, lg.description, lg.created_at, lg.updated_at,
		       (SELECT COUNT(*) FROM local_group_members WHERE group_id = lg.id) as member_count
		FROM local_groups lg
		WHERE lg.name = $1
	`, name).Scan(&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt, &group.MemberCount)
	if err == pgx.ErrNoRows {
		return nil, ErrLocalGroupNotFound
	}
	return &group, err
}

// ListLocalGroups retrieves all local groups
func (s *LocalGroupStore) ListLocalGroups(ctx context.Context) ([]*LocalGroup, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT lg.id, lg.name, lg.description, lg.created_at, lg.updated_at,
		       (SELECT COUNT(*) FROM local_group_members WHERE group_id = lg.id) as member_count
		FROM local_groups lg
		ORDER BY lg.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*LocalGroup
	for rows.Next() {
		var g LocalGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount); err != nil {
			return nil, err
		}
		groups = append(groups, &g)
	}
	return groups, rows.Err()
}

// UpdateLocalGroup updates a local group
func (s *LocalGroupStore) UpdateLocalGroup(ctx context.Context, group *LocalGroup) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE local_groups SET name = $2, description = $3
		WHERE id = $1
	`, group.ID, group.Name, group.Description)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrLocalGroupExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLocalGroupNotFound
	}
	return nil
}

// DeleteLocalGroup deletes a local group
func (s *LocalGroupStore) DeleteLocalGroup(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM local_groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLocalGroupNotFound
	}
	return nil
}

// AddMember adds a user to a local group
func (s *LocalGroupStore) AddMember(ctx context.Context, groupID, userID, memberType string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO local_group_members (group_id, user_id, member_type)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, groupID, userID, memberType)
	return err
}

// RemoveMember removes a user from a local group
func (s *LocalGroupStore) RemoveMember(ctx context.Context, groupID, userID, memberType string) error {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM local_group_members
		WHERE group_id = $1 AND user_id = $2 AND member_type = $3
	`, groupID, userID, memberType)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// GetGroupMembers returns all members of a local group with user details
func (s *LocalGroupStore) GetGroupMembers(ctx context.Context, groupID string) ([]LocalGroupMember, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT lgm.user_id, lgm.member_type, lgm.created_at,
		       COALESCE(u.email, lu.email, '') as email,
		       COALESCE(u.name, lu.username, '') as name
		FROM local_group_members lgm
		LEFT JOIN users u ON lgm.member_type = 'sso' AND lgm.user_id = u.id::text
		LEFT JOIN local_users lu ON lgm.member_type = 'local' AND lgm.user_id = lu.id::text
		WHERE lgm.group_id = $1
		ORDER BY lgm.member_type, email
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []LocalGroupMember
	for rows.Next() {
		var m LocalGroupMember
		if err := rows.Scan(&m.UserID, &m.MemberType, &m.CreatedAt, &m.Email, &m.Name); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// GetUserLocalGroups returns all local groups a user belongs to
// Returns group names with the "local:" prefix
func (s *LocalGroupStore) GetUserLocalGroups(ctx context.Context, userID, memberType string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT lg.name
		FROM local_groups lg
		INNER JOIN local_group_members lgm ON lg.id = lgm.group_id
		WHERE lgm.user_id = $1 AND lgm.member_type = $2
		ORDER BY lg.name
	`, userID, memberType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		// Add the local: prefix to distinguish from IdP groups
		groups = append(groups, LocalGroupPrefix+name)
	}
	return groups, rows.Err()
}

// GetUserLocalGroupsRaw returns all local groups a user belongs to without prefix
// This is useful for internal operations where we need the raw group name
func (s *LocalGroupStore) GetUserLocalGroupsRaw(ctx context.Context, userID, memberType string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT lg.name
		FROM local_groups lg
		INNER JOIN local_group_members lgm ON lg.id = lgm.group_id
		WHERE lgm.user_id = $1 AND lgm.member_type = $2
		ORDER BY lg.name
	`, userID, memberType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		groups = append(groups, name)
	}
	return groups, rows.Err()
}

// GetLocalGroupMembersForGroup returns all members (SSO and local users) of a local group
// This is similar to GetGroupMembers but also returns SSO user info
func (s *LocalGroupStore) GetLocalGroupMembersForGroup(ctx context.Context, groupName string) ([]*SSOUser, []*LocalUser, error) {
	// First get the group by name
	group, err := s.GetLocalGroupByName(ctx, groupName)
	if err != nil {
		return nil, nil, err
	}

	// Get SSO members
	ssoRows, err := s.db.Pool.Query(ctx, `
		SELECT u.id, u.external_id, u.provider, u.email, u.name, u.groups, u.is_admin, u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		INNER JOIN local_group_members lgm ON lgm.user_id = u.id::text AND lgm.member_type = 'sso'
		WHERE lgm.group_id = $1
		ORDER BY u.email
	`, group.ID)
	if err != nil {
		return nil, nil, err
	}
	defer ssoRows.Close()

	var ssoUsers []*SSOUser
	for ssoRows.Next() {
		var u SSOUser
		var groupsJSON []byte
		if err := ssoRows.Scan(&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
			&groupsJSON, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, nil, err
		}
		ssoUsers = append(ssoUsers, &u)
	}

	// Get local members
	localRows, err := s.db.Pool.Query(ctx, `
		SELECT lu.id, lu.username, lu.email, lu.is_admin, lu.last_login_at, lu.created_at
		FROM local_users lu
		INNER JOIN local_group_members lgm ON lgm.user_id = lu.id::text AND lgm.member_type = 'local'
		WHERE lgm.group_id = $1
		ORDER BY lu.username
	`, group.ID)
	if err != nil {
		return nil, nil, err
	}
	defer localRows.Close()

	var localUsers []*LocalUser
	for localRows.Next() {
		var u LocalUser
		if err := localRows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.LastLoginAt, &u.CreatedAt); err != nil {
			return nil, nil, err
		}
		localUsers = append(localUsers, &u)
	}

	return ssoUsers, localUsers, nil
}
