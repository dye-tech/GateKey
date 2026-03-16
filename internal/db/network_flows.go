package db

import (
	"context"
	"fmt"
	"time"
)

// NetworkFlowLog represents a network flow log entry
type NetworkFlowLog struct {
	ID            string
	GatewayID     string
	GatewayName   string
	UserID        string
	UserEmail     string
	SourceIP      string
	DestIP        string
	DestPort      int
	Protocol      string
	BytesSent     int64
	BytesReceived int64
	FlowStart     time.Time
	FlowEnd       *time.Time
	CreatedAt     time.Time
}

// NetworkFlowFilter defines filtering options for flow log queries
type NetworkFlowFilter struct {
	GatewayID string
	UserID    string
	UserEmail string
	DestIP    string
	Protocol  string
	Limit     int
	Offset    int
}

// NetworkFlowStore handles network flow log persistence
type NetworkFlowStore struct {
	db *DB
}

// NewNetworkFlowStore creates a new network flow store
func NewNetworkFlowStore(db *DB) *NetworkFlowStore {
	return &NetworkFlowStore{db: db}
}

// BatchInsert inserts multiple flow log records
func (s *NetworkFlowStore) BatchInsert(ctx context.Context, flows []*NetworkFlowLog) error {
	for _, f := range flows {
		_, err := s.db.Pool.Exec(ctx, `
			INSERT INTO network_flow_logs (gateway_id, gateway_name, user_id, user_email, source_ip, dest_ip, dest_port, protocol, bytes_sent, bytes_received, flow_start, flow_end)
			VALUES ($1, $2, $3, $4, $5::inet, $6::inet, $7, $8, $9, $10, $11, $12)
		`, f.GatewayID, f.GatewayName, f.UserID, f.UserEmail, f.SourceIP, f.DestIP, f.DestPort, f.Protocol, f.BytesSent, f.BytesReceived, f.FlowStart, f.FlowEnd)
		if err != nil {
			return fmt.Errorf("failed to insert flow log: %w", err)
		}
	}
	return nil
}

// List returns flow logs matching the given filter
func (s *NetworkFlowStore) List(ctx context.Context, filter NetworkFlowFilter) ([]*NetworkFlowLog, error) {
	query := `SELECT id, gateway_id, gateway_name, user_id, user_email, source_ip::text, dest_ip::text, dest_port, protocol, bytes_sent, bytes_received, flow_start, flow_end, created_at FROM network_flow_logs WHERE 1=1`
	var args []interface{}
	paramNum := 1

	if filter.GatewayID != "" {
		query += fmt.Sprintf(" AND gateway_id = $%d", paramNum)
		args = append(args, filter.GatewayID)
		paramNum++
	}
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", paramNum)
		args = append(args, filter.UserID)
		paramNum++
	}
	if filter.UserEmail != "" {
		query += fmt.Sprintf(" AND user_email = $%d", paramNum)
		args = append(args, filter.UserEmail)
		paramNum++
	}
	if filter.DestIP != "" {
		query += fmt.Sprintf(" AND dest_ip = $%d::inet", paramNum)
		args = append(args, filter.DestIP)
		paramNum++
	}
	if filter.Protocol != "" {
		query += fmt.Sprintf(" AND protocol = $%d", paramNum)
		args = append(args, filter.Protocol)
		paramNum++
	}

	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	query += fmt.Sprintf(" LIMIT $%d", paramNum)
	args = append(args, limit)
	paramNum++

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query flow logs: %w", err)
	}
	defer rows.Close()

	var flows []*NetworkFlowLog
	for rows.Next() {
		var f NetworkFlowLog
		if err := rows.Scan(&f.ID, &f.GatewayID, &f.GatewayName, &f.UserID, &f.UserEmail, &f.SourceIP, &f.DestIP, &f.DestPort, &f.Protocol, &f.BytesSent, &f.BytesReceived, &f.FlowStart, &f.FlowEnd, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan flow log: %w", err)
		}
		flows = append(flows, &f)
	}
	return flows, rows.Err()
}

// GetUserActivity returns flow logs for a specific user
func (s *NetworkFlowStore) GetUserActivity(ctx context.Context, userID string, limit int) ([]*NetworkFlowLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, gateway_id, gateway_name, user_id, user_email, source_ip::text, dest_ip::text, dest_port, protocol, bytes_sent, bytes_received, flow_start, flow_end, created_at
		FROM network_flow_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user flow activity: %w", err)
	}
	defer rows.Close()

	var flows []*NetworkFlowLog
	for rows.Next() {
		var f NetworkFlowLog
		if err := rows.Scan(&f.ID, &f.GatewayID, &f.GatewayName, &f.UserID, &f.UserEmail, &f.SourceIP, &f.DestIP, &f.DestPort, &f.Protocol, &f.BytesSent, &f.BytesReceived, &f.FlowStart, &f.FlowEnd, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan flow log: %w", err)
		}
		flows = append(flows, &f)
	}
	return flows, rows.Err()
}

// GetTopDestinations returns the most accessed destinations
func (s *NetworkFlowStore) GetTopDestinations(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT dest_ip::text, COUNT(*) as flow_count, COALESCE(SUM(bytes_sent + bytes_received), 0) as total_bytes
		FROM network_flow_logs
		GROUP BY dest_ip
		ORDER BY flow_count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top destinations: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var destIP string
		var flowCount int64
		var totalBytes int64
		if err := rows.Scan(&destIP, &flowCount, &totalBytes); err != nil {
			return nil, fmt.Errorf("failed to scan top destination: %w", err)
		}
		results = append(results, map[string]interface{}{
			"dest_ip":     destIP,
			"flow_count":  flowCount,
			"total_bytes": totalBytes,
		})
	}
	return results, rows.Err()
}

// GetStats returns overall flow log statistics
func (s *NetworkFlowStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	var totalFlows int64
	var totalBytes int64
	var uniqueUsers int64
	var uniqueDests int64
	var flowsToday int64

	err := s.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(bytes_sent + bytes_received), 0),
			COUNT(DISTINCT user_id),
			COUNT(DISTINCT dest_ip)
		FROM network_flow_logs
	`).Scan(&totalFlows, &totalBytes, &uniqueUsers, &uniqueDests)
	if err != nil {
		return nil, fmt.Errorf("failed to query flow stats: %w", err)
	}

	err = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM network_flow_logs WHERE created_at >= CURRENT_DATE
	`).Scan(&flowsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to query flows today: %w", err)
	}

	return map[string]interface{}{
		"total_flows":         totalFlows,
		"total_bytes":         totalBytes,
		"unique_users":        uniqueUsers,
		"unique_destinations": uniqueDests,
		"flows_today":         flowsToday,
	}, nil
}

// DeleteOlderThan deletes flow logs older than the specified number of days
func (s *NetworkFlowStore) DeleteOlderThan(ctx context.Context, days int) (int, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM network_flow_logs WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`, days)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old flow logs: %w", err)
	}
	return int(result.RowsAffected()), nil
}
