package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrProxyAppNotFound = errors.New("proxy application not found")
	ErrProxyAppExists   = errors.New("proxy application with this slug already exists")
)

// ProxyApplication represents a web application accessible through the reverse proxy
type ProxyApplication struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Slug               string            `json:"slug"`
	Description        string            `json:"description"`
	InternalURL        string            `json:"internal_url"`
	IconURL            *string           `json:"icon_url"`
	IsActive           bool              `json:"is_active"`
	PreserveHostHeader bool              `json:"preserve_host_header"`
	StripPrefix        bool              `json:"strip_prefix"`
	InjectHeaders      map[string]string `json:"inject_headers"`
	AllowedHeaders     []string          `json:"allowed_headers"`
	WebsocketEnabled   bool              `json:"websocket_enabled"`
	TimeoutSeconds     int               `json:"timeout_seconds"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// ProxyAccessLog represents an access log entry for proxy requests
type ProxyAccessLog struct {
	ID             string    `json:"id"`
	ProxyAppID     string    `json:"proxy_app_id"`
	UserID         string    `json:"user_id"`
	UserEmail      string    `json:"user_email"`
	RequestMethod  string    `json:"request_method"`
	RequestPath    string    `json:"request_path"`
	ResponseStatus int       `json:"response_status"`
	ResponseTimeMs int       `json:"response_time_ms"`
	ClientIP       string    `json:"client_ip"`
	UserAgent      string    `json:"user_agent"`
	CreatedAt      time.Time `json:"created_at"`
}

// ProxyApplicationStore handles proxy application persistence
type ProxyApplicationStore struct {
	db *DB
}

// NewProxyApplicationStore creates a new proxy application store
func NewProxyApplicationStore(db *DB) *ProxyApplicationStore {
	return &ProxyApplicationStore{db: db}
}

// CreateProxyApplication creates a new proxy application
func (s *ProxyApplicationStore) CreateProxyApplication(ctx context.Context, app *ProxyApplication) error {
	injectHeaders, err := json.Marshal(app.InjectHeaders)
	if err != nil {
		return err
	}
	allowedHeaders, err := json.Marshal(app.AllowedHeaders)
	if err != nil {
		return err
	}

	err = s.db.Pool.QueryRow(ctx, `
		INSERT INTO proxy_applications (name, slug, description, internal_url, icon_url, is_active,
			preserve_host_header, strip_prefix, inject_headers, allowed_headers, websocket_enabled, timeout_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`, app.Name, app.Slug, app.Description, app.InternalURL, app.IconURL, app.IsActive,
		app.PreserveHostHeader, app.StripPrefix, injectHeaders, allowedHeaders,
		app.WebsocketEnabled, app.TimeoutSeconds).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)

	if err != nil && isUniqueViolation(err) {
		return ErrProxyAppExists
	}
	return err
}

// GetProxyApplication retrieves a proxy application by ID
func (s *ProxyApplicationStore) GetProxyApplication(ctx context.Context, id string) (*ProxyApplication, error) {
	return s.scanProxyApp(s.db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, description, internal_url, icon_url, is_active,
			preserve_host_header, strip_prefix, inject_headers, allowed_headers,
			websocket_enabled, timeout_seconds, created_at, updated_at
		FROM proxy_applications WHERE id = $1
	`, id))
}

// GetProxyApplicationBySlug retrieves a proxy application by slug
func (s *ProxyApplicationStore) GetProxyApplicationBySlug(ctx context.Context, slug string) (*ProxyApplication, error) {
	return s.scanProxyApp(s.db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, description, internal_url, icon_url, is_active,
			preserve_host_header, strip_prefix, inject_headers, allowed_headers,
			websocket_enabled, timeout_seconds, created_at, updated_at
		FROM proxy_applications WHERE slug = $1
	`, slug))
}

func (s *ProxyApplicationStore) scanProxyApp(row pgx.Row) (*ProxyApplication, error) {
	var app ProxyApplication
	var injectHeaders, allowedHeaders []byte

	err := row.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.InternalURL,
		&app.IconURL, &app.IsActive, &app.PreserveHostHeader, &app.StripPrefix,
		&injectHeaders, &allowedHeaders, &app.WebsocketEnabled, &app.TimeoutSeconds,
		&app.CreatedAt, &app.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrProxyAppNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(injectHeaders, &app.InjectHeaders); err != nil {
		app.InjectHeaders = make(map[string]string)
	}
	if err := json.Unmarshal(allowedHeaders, &app.AllowedHeaders); err != nil {
		app.AllowedHeaders = []string{"*"}
	}

	return &app, nil
}

// ListProxyApplications retrieves all proxy applications
func (s *ProxyApplicationStore) ListProxyApplications(ctx context.Context) ([]*ProxyApplication, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, slug, description, internal_url, icon_url, is_active,
			preserve_host_header, strip_prefix, inject_headers, allowed_headers,
			websocket_enabled, timeout_seconds, created_at, updated_at
		FROM proxy_applications ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanProxyApps(rows)
}

// ListActiveProxyApplications retrieves all active proxy applications
func (s *ProxyApplicationStore) ListActiveProxyApplications(ctx context.Context) ([]*ProxyApplication, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, slug, description, internal_url, icon_url, is_active,
			preserve_host_header, strip_prefix, inject_headers, allowed_headers,
			websocket_enabled, timeout_seconds, created_at, updated_at
		FROM proxy_applications WHERE is_active = true ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanProxyApps(rows)
}

func (s *ProxyApplicationStore) scanProxyApps(rows pgx.Rows) ([]*ProxyApplication, error) {
	var apps []*ProxyApplication
	for rows.Next() {
		var app ProxyApplication
		var injectHeaders, allowedHeaders []byte

		if err := rows.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.InternalURL,
			&app.IconURL, &app.IsActive, &app.PreserveHostHeader, &app.StripPrefix,
			&injectHeaders, &allowedHeaders, &app.WebsocketEnabled, &app.TimeoutSeconds,
			&app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(injectHeaders, &app.InjectHeaders); err != nil {
			app.InjectHeaders = make(map[string]string)
		}
		if err := json.Unmarshal(allowedHeaders, &app.AllowedHeaders); err != nil {
			app.AllowedHeaders = []string{"*"}
		}

		apps = append(apps, &app)
	}
	return apps, rows.Err()
}

// UpdateProxyApplication updates a proxy application
func (s *ProxyApplicationStore) UpdateProxyApplication(ctx context.Context, app *ProxyApplication) error {
	injectHeaders, err := json.Marshal(app.InjectHeaders)
	if err != nil {
		return err
	}
	allowedHeaders, err := json.Marshal(app.AllowedHeaders)
	if err != nil {
		return err
	}

	result, err := s.db.Pool.Exec(ctx, `
		UPDATE proxy_applications SET name = $2, slug = $3, description = $4, internal_url = $5,
			icon_url = $6, is_active = $7, preserve_host_header = $8, strip_prefix = $9,
			inject_headers = $10, allowed_headers = $11, websocket_enabled = $12, timeout_seconds = $13
		WHERE id = $1
	`, app.ID, app.Name, app.Slug, app.Description, app.InternalURL, app.IconURL, app.IsActive,
		app.PreserveHostHeader, app.StripPrefix, injectHeaders, allowedHeaders,
		app.WebsocketEnabled, app.TimeoutSeconds)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrProxyAppExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProxyAppNotFound
	}
	return nil
}

// DeleteProxyApplication deletes a proxy application
func (s *ProxyApplicationStore) DeleteProxyApplication(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM proxy_applications WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProxyAppNotFound
	}
	return nil
}

// ---- User Assignment Methods ----

// AssignAppToUser assigns a proxy application to a user
func (s *ProxyApplicationStore) AssignAppToUser(ctx context.Context, userID, appID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO user_proxy_applications (user_id, proxy_app_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, appID)
	return err
}

// RemoveAppFromUser removes a proxy application from a user
func (s *ProxyApplicationStore) RemoveAppFromUser(ctx context.Context, userID, appID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM user_proxy_applications WHERE user_id = $1 AND proxy_app_id = $2
	`, userID, appID)
	return err
}

// GetAppUsers returns all users assigned to a proxy application
func (s *ProxyApplicationStore) GetAppUsers(ctx context.Context, appID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT user_id FROM user_proxy_applications WHERE proxy_app_id = $1
	`, appID)
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

// ---- Group Assignment Methods ----

// AssignAppToGroup assigns a proxy application to a group
func (s *ProxyApplicationStore) AssignAppToGroup(ctx context.Context, groupName, appID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO group_proxy_applications (group_name, proxy_app_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupName, appID)
	return err
}

// RemoveAppFromGroup removes a proxy application from a group
func (s *ProxyApplicationStore) RemoveAppFromGroup(ctx context.Context, groupName, appID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM group_proxy_applications WHERE group_name = $1 AND proxy_app_id = $2
	`, groupName, appID)
	return err
}

// GetAppGroups returns all groups assigned to a proxy application
func (s *ProxyApplicationStore) GetAppGroups(ctx context.Context, appID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name FROM group_proxy_applications WHERE proxy_app_id = $1
	`, appID)
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

// ---- User Access Methods ----

// GetUserProxyApplications returns all proxy applications a user can access
// This checks: direct user assignment and group membership
func (s *ProxyApplicationStore) GetUserProxyApplications(ctx context.Context, userID string, groups []string) ([]*ProxyApplication, error) {
	query := `
		SELECT DISTINCT pa.id, pa.name, pa.slug, pa.description, pa.internal_url, pa.icon_url,
			pa.is_active, pa.preserve_host_header, pa.strip_prefix, pa.inject_headers,
			pa.allowed_headers, pa.websocket_enabled, pa.timeout_seconds, pa.created_at, pa.updated_at
		FROM proxy_applications pa
		LEFT JOIN user_proxy_applications upa ON pa.id = upa.proxy_app_id AND upa.user_id = $1
		LEFT JOIN group_proxy_applications gpa ON pa.id = gpa.proxy_app_id
		WHERE pa.is_active = true
			AND (
				upa.user_id IS NOT NULL
				OR gpa.group_name = ANY($2)
			)
		ORDER BY pa.name
	`

	rows, err := s.db.Pool.Query(ctx, query, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanProxyApps(rows)
}

// CanUserAccessApp checks if a user can access a specific proxy application by slug
// Returns the application if access is granted
func (s *ProxyApplicationStore) CanUserAccessApp(ctx context.Context, userID string, groups []string, slug string) (bool, *ProxyApplication, error) {
	query := `
		SELECT DISTINCT pa.id, pa.name, pa.slug, pa.description, pa.internal_url, pa.icon_url,
			pa.is_active, pa.preserve_host_header, pa.strip_prefix, pa.inject_headers,
			pa.allowed_headers, pa.websocket_enabled, pa.timeout_seconds, pa.created_at, pa.updated_at
		FROM proxy_applications pa
		LEFT JOIN user_proxy_applications upa ON pa.id = upa.proxy_app_id AND upa.user_id = $1
		LEFT JOIN group_proxy_applications gpa ON pa.id = gpa.proxy_app_id
		WHERE pa.slug = $3 AND pa.is_active = true
			AND (
				upa.user_id IS NOT NULL
				OR gpa.group_name = ANY($2)
			)
		LIMIT 1
	`

	app, err := s.scanProxyApp(s.db.Pool.QueryRow(ctx, query, userID, groups, slug))
	if err == ErrProxyAppNotFound {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	return true, app, nil
}

// ---- Audit Logging ----

// LogProxyAccess logs a proxy access event
func (s *ProxyApplicationStore) LogProxyAccess(ctx context.Context, log *ProxyAccessLog) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO proxy_access_logs (proxy_app_id, user_id, user_email, request_method,
			request_path, response_status, response_time_ms, client_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet, $9)
	`, log.ProxyAppID, log.UserID, log.UserEmail, log.RequestMethod, log.RequestPath,
		log.ResponseStatus, log.ResponseTimeMs, log.ClientIP, log.UserAgent)
	return err
}

// GetProxyAccessLogs retrieves access logs for a proxy application
func (s *ProxyApplicationStore) GetProxyAccessLogs(ctx context.Context, appID string, limit int) ([]*ProxyAccessLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, proxy_app_id, user_id, user_email, request_method, request_path,
			response_status, response_time_ms, host(client_ip), user_agent, created_at
		FROM proxy_access_logs
		WHERE proxy_app_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*ProxyAccessLog
	for rows.Next() {
		var log ProxyAccessLog
		var clientIP *string
		if err := rows.Scan(&log.ID, &log.ProxyAppID, &log.UserID, &log.UserEmail,
			&log.RequestMethod, &log.RequestPath, &log.ResponseStatus, &log.ResponseTimeMs,
			&clientIP, &log.UserAgent, &log.CreatedAt); err != nil {
			return nil, err
		}
		if clientIP != nil {
			log.ClientIP = *clientIP
		}
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}

// helper function to check for unique constraint violations
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
