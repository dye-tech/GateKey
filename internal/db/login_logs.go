package db

import (
	"context"
	"strconv"
	"time"
)

// LoginLog represents a user login event
type LoginLog struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	UserEmail     string    `json:"user_email"`
	UserName      string    `json:"user_name,omitempty"`
	Provider      string    `json:"provider"`      // 'oidc', 'saml', 'local'
	ProviderName  string    `json:"provider_name"` // specific provider name
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent,omitempty"`
	Country       string    `json:"country,omitempty"`
	CountryCode   string    `json:"country_code,omitempty"` // ISO 3166-1 alpha-2 code
	City          string    `json:"city,omitempty"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failure_reason,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LoginLogStats provides aggregated statistics
type LoginLogStats struct {
	TotalLogins      int            `json:"total_logins"`
	SuccessfulLogins int            `json:"successful_logins"`
	FailedLogins     int            `json:"failed_logins"`
	UniqueUsers      int            `json:"unique_users"`
	UniqueIPs        int            `json:"unique_ips"`
	LoginsByProvider map[string]int `json:"logins_by_provider"`
	LoginsByCountry  map[string]int `json:"logins_by_country,omitempty"`
	RecentFailures   []*LoginLog    `json:"recent_failures,omitempty"`
}

// LoginLogFilter provides filtering options for queries
type LoginLogFilter struct {
	UserEmail string
	UserID    string
	IPAddress string
	Provider  string
	Success   *bool
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// LoginLogStore handles login log persistence
type LoginLogStore struct {
	db *DB
}

// NewLoginLogStore creates a new login log store
func NewLoginLogStore(db *DB) *LoginLogStore {
	return &LoginLogStore{db: db}
}

// Create inserts a new login log entry
func (s *LoginLogStore) Create(ctx context.Context, log *LoginLog) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO login_logs (
			user_id, user_email, user_name, provider, provider_name,
			ip_address, user_agent, country, country_code, city, success, failure_reason, session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, log.UserID, log.UserEmail, log.UserName, log.Provider, log.ProviderName,
		log.IPAddress, log.UserAgent, log.Country, log.CountryCode, log.City, log.Success, log.FailureReason, log.SessionID)
	return err
}

// List retrieves login logs with optional filtering
func (s *LoginLogStore) List(ctx context.Context, filter *LoginLogFilter) ([]*LoginLog, int, error) {
	// Build query with filters
	baseQuery := `
		SELECT id, user_id, user_email, COALESCE(user_name, ''), provider, COALESCE(provider_name, ''),
		       host(ip_address), COALESCE(user_agent, ''), COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''),
		       success, COALESCE(failure_reason, ''), COALESCE(session_id, ''), created_at
		FROM login_logs
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM login_logs WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if filter.UserEmail != "" {
		baseQuery += ` AND user_email ILIKE $` + itoa(argNum)
		countQuery += ` AND user_email ILIKE $` + itoa(argNum)
		args = append(args, "%"+filter.UserEmail+"%")
		argNum++
	}
	if filter.UserID != "" {
		baseQuery += ` AND user_id = $` + itoa(argNum)
		countQuery += ` AND user_id = $` + itoa(argNum)
		args = append(args, filter.UserID)
		argNum++
	}
	if filter.IPAddress != "" {
		baseQuery += ` AND host(ip_address) LIKE $` + itoa(argNum)
		countQuery += ` AND host(ip_address) LIKE $` + itoa(argNum)
		args = append(args, "%"+filter.IPAddress+"%")
		argNum++
	}
	if filter.Provider != "" {
		baseQuery += ` AND provider = $` + itoa(argNum)
		countQuery += ` AND provider = $` + itoa(argNum)
		args = append(args, filter.Provider)
		argNum++
	}
	if filter.Success != nil {
		baseQuery += ` AND success = $` + itoa(argNum)
		countQuery += ` AND success = $` + itoa(argNum)
		args = append(args, *filter.Success)
		argNum++
	}
	if filter.StartTime != nil {
		baseQuery += ` AND created_at >= $` + itoa(argNum)
		countQuery += ` AND created_at >= $` + itoa(argNum)
		args = append(args, *filter.StartTime)
		argNum++
	}
	if filter.EndTime != nil {
		baseQuery += ` AND created_at <= $` + itoa(argNum)
		countQuery += ` AND created_at <= $` + itoa(argNum)
		args = append(args, *filter.EndTime)
		argNum++
	}

	// Get total count
	var total int
	err := s.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add ordering and pagination
	baseQuery += ` ORDER BY created_at DESC`
	if filter.Limit > 0 {
		baseQuery += ` LIMIT $` + itoa(argNum)
		args = append(args, filter.Limit)
		argNum++
	}
	if filter.Offset > 0 {
		baseQuery += ` OFFSET $` + itoa(argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*LoginLog
	for rows.Next() {
		var log LoginLog
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.UserEmail, &log.UserName, &log.Provider, &log.ProviderName,
			&log.IPAddress, &log.UserAgent, &log.Country, &log.CountryCode, &log.City,
			&log.Success, &log.FailureReason, &log.SessionID, &log.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}
	return logs, total, rows.Err()
}

// GetStats retrieves aggregated login statistics
func (s *LoginLogStore) GetStats(ctx context.Context, days int) (*LoginLogStats, error) {
	stats := &LoginLogStats{
		LoginsByProvider: make(map[string]int),
		LoginsByCountry:  make(map[string]int),
	}

	since := time.Now().AddDate(0, 0, -days)

	// Get total counts
	err := s.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE success = true) as successful,
			COUNT(*) FILTER (WHERE success = false) as failed,
			COUNT(DISTINCT user_email) as unique_users,
			COUNT(DISTINCT ip_address) as unique_ips
		FROM login_logs
		WHERE created_at >= $1
	`, since).Scan(&stats.TotalLogins, &stats.SuccessfulLogins, &stats.FailedLogins, &stats.UniqueUsers, &stats.UniqueIPs)
	if err != nil {
		return nil, err
	}

	// Get logins by provider
	rows, err := s.db.Pool.Query(ctx, `
		SELECT provider, COUNT(*)
		FROM login_logs
		WHERE created_at >= $1
		GROUP BY provider
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			return nil, err
		}
		stats.LoginsByProvider[provider] = count
	}

	// Get logins by country (top 10)
	rows, err = s.db.Pool.Query(ctx, `
		SELECT COALESCE(country, 'Unknown'), COUNT(*)
		FROM login_logs
		WHERE created_at >= $1
		GROUP BY country
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var country string
		var count int
		if err := rows.Scan(&country, &count); err != nil {
			return nil, err
		}
		stats.LoginsByCountry[country] = count
	}

	// Get recent failures
	failRows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, user_email, COALESCE(user_name, ''), provider, COALESCE(provider_name, ''),
		       host(ip_address), COALESCE(user_agent, ''), COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''),
		       success, COALESCE(failure_reason, ''), COALESCE(session_id, ''), created_at
		FROM login_logs
		WHERE success = false AND created_at >= $1
		ORDER BY created_at DESC
		LIMIT 10
	`, since)
	if err != nil {
		return nil, err
	}
	defer failRows.Close()

	for failRows.Next() {
		var log LoginLog
		if err := failRows.Scan(
			&log.ID, &log.UserID, &log.UserEmail, &log.UserName, &log.Provider, &log.ProviderName,
			&log.IPAddress, &log.UserAgent, &log.Country, &log.CountryCode, &log.City,
			&log.Success, &log.FailureReason, &log.SessionID, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		stats.RecentFailures = append(stats.RecentFailures, &log)
	}

	return stats, nil
}

// DeleteOlderThan removes login logs older than the specified duration
func (s *LoginLogStore) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM login_logs WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetUserLoginHistory retrieves login history for a specific user
func (s *LoginLogStore) GetUserLoginHistory(ctx context.Context, userEmail string, limit int) ([]*LoginLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, user_email, COALESCE(user_name, ''), provider, COALESCE(provider_name, ''),
		       host(ip_address), COALESCE(user_agent, ''), COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''),
		       success, COALESCE(failure_reason, ''), COALESCE(session_id, ''), created_at
		FROM login_logs
		WHERE user_email = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userEmail, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*LoginLog
	for rows.Next() {
		var log LoginLog
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.UserEmail, &log.UserName, &log.Provider, &log.ProviderName,
			&log.IPAddress, &log.UserAgent, &log.Country, &log.CountryCode, &log.City,
			&log.Success, &log.FailureReason, &log.SessionID, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}

// helper function to convert int to string for query building
func itoa(i int) string {
	return strconv.Itoa(i)
}

// Setting key for login log retention
const SettingLoginLogRetentionDays = "login_log_retention_days"
