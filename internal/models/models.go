// Package models defines database models for GateKey.
package models

import (
	"database/sql"
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// User represents a user account synced from an identity provider.
type User struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ExternalID  string          `json:"external_id" db:"external_id"` // ID from the IdP
	Provider    string          `json:"provider" db:"provider"`       // OIDC provider name
	Email       string          `json:"email" db:"email"`
	Name        string          `json:"name" db:"name"`
	Groups      []string        `json:"groups" db:"groups"`         // Groups from IdP
	Attributes  json.RawMessage `json:"attributes" db:"attributes"` // Additional claims
	IsAdmin     bool            `json:"is_admin" db:"is_admin"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	LastLoginAt *time.Time      `json:"last_login_at" db:"last_login_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// Session represents an active user session.
type Session struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Token     string     `json:"-" db:"token"` // Hashed session token
	IPAddress string     `json:"ip_address" db:"ip_address"`
	UserAgent string     `json:"user_agent" db:"user_agent"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// Certificate represents an issued client certificate for tracking and revocation.
type Certificate struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	SessionID        uuid.UUID  `json:"session_id" db:"session_id"`
	SerialNumber     string     `json:"serial_number" db:"serial_number"`
	Subject          string     `json:"subject" db:"subject"`
	NotBefore        time.Time  `json:"not_before" db:"not_before"`
	NotAfter         time.Time  `json:"not_after" db:"not_after"`
	Fingerprint      string     `json:"fingerprint" db:"fingerprint"` // SHA256 fingerprint
	IsRevoked        bool       `json:"is_revoked" db:"is_revoked"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevocationReason string     `json:"revocation_reason,omitempty" db:"revocation_reason"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// Gateway represents a registered gateway node.
type Gateway struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	Name           string          `json:"name" db:"name"`
	Hostname       string          `json:"hostname" db:"hostname"`
	PublicIP       string          `json:"public_ip" db:"public_ip"`
	VPNPort        int             `json:"vpn_port" db:"vpn_port"`
	VPNProtocol    string          `json:"vpn_protocol" db:"vpn_protocol"`         // tcp or udp
	TLSAuthEnabled bool            `json:"tls_auth_enabled" db:"tls_auth_enabled"` // Enable TLS-Auth
	Token          string          `json:"-" db:"token"`                           // Hashed authentication token
	PublicKey      string          `json:"public_key" db:"public_key"`             // Gateway's TLS public key
	Config         json.RawMessage `json:"config" db:"config"`                     // Additional config
	IsActive       bool            `json:"is_active" db:"is_active"`
	LastHeartbeat  *time.Time      `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// Policy represents an access control policy.
type Policy struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Priority    int       `json:"priority" db:"priority"` // Lower = higher priority
	IsEnabled   bool      `json:"is_enabled" db:"is_enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
}

// PolicyRule represents a single rule within a policy.
type PolicyRule struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	PolicyID   uuid.UUID       `json:"policy_id" db:"policy_id"`
	Action     string          `json:"action" db:"action"`         // allow, deny
	Subject    json.RawMessage `json:"subject" db:"subject"`       // User/group matcher
	Resource   json.RawMessage `json:"resource" db:"resource"`     // Network/service matcher
	Conditions json.RawMessage `json:"conditions" db:"conditions"` // Additional conditions
	Priority   int             `json:"priority" db:"priority"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// PolicySubject defines who the policy applies to.
type PolicySubject struct {
	Users    []string `json:"users,omitempty"`    // User emails or IDs
	Groups   []string `json:"groups,omitempty"`   // Group names
	Everyone bool     `json:"everyone,omitempty"` // Applies to all users
}

// PolicyResource defines what resources the policy controls.
type PolicyResource struct {
	Gateways []string     `json:"gateways,omitempty"` // Gateway IDs or names
	Networks []string     `json:"networks,omitempty"` // CIDR ranges
	Ports    []PolicyPort `json:"ports,omitempty"`    // Port ranges
	Services []string     `json:"services,omitempty"` // Service names
}

// PolicyPort defines a port or port range.
type PolicyPort struct {
	Protocol string `json:"protocol"`            // tcp, udp, or both
	Port     int    `json:"port,omitempty"`      // Single port
	FromPort int    `json:"from_port,omitempty"` // Range start
	ToPort   int    `json:"to_port,omitempty"`   // Range end
}

// PolicyCondition defines additional conditions for a rule.
type PolicyCondition struct {
	TimeWindows []TimeWindow `json:"time_windows,omitempty"` // When the policy applies
	SourceIPs   []string     `json:"source_ips,omitempty"`   // Client source IP restrictions
}

// TimeWindow defines when a policy is active.
type TimeWindow struct {
	Days      []string `json:"days"`       // mon, tue, wed, thu, fri, sat, sun
	StartTime string   `json:"start_time"` // HH:MM format
	EndTime   string   `json:"end_time"`   // HH:MM format
	Timezone  string   `json:"timezone"`   // IANA timezone
}

// Connection represents an active VPN connection.
type Connection struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	SessionID        uuid.UUID  `json:"session_id" db:"session_id"`
	CertificateID    uuid.UUID  `json:"certificate_id" db:"certificate_id"`
	GatewayID        uuid.UUID  `json:"gateway_id" db:"gateway_id"`
	ClientIP         string     `json:"client_ip" db:"client_ip"`         // Real client IP
	VPNIPv4          string     `json:"vpn_ipv4" db:"vpn_ipv4"`           // Assigned VPN IPv4
	VPNIPv6          string     `json:"vpn_ipv6,omitempty" db:"vpn_ipv6"` // Assigned VPN IPv6
	BytesSent        int64      `json:"bytes_sent" db:"bytes_sent"`
	BytesReceived    int64      `json:"bytes_received" db:"bytes_received"`
	ConnectedAt      time.Time  `json:"connected_at" db:"connected_at"`
	DisconnectedAt   *time.Time `json:"disconnected_at,omitempty" db:"disconnected_at"`
	DisconnectReason string     `json:"disconnect_reason,omitempty" db:"disconnect_reason"`
}

// AuditLog represents an audit trail entry.
type AuditLog struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
	Event        string          `json:"event" db:"event"`                 // Event type
	ActorID      *uuid.UUID      `json:"actor_id,omitempty" db:"actor_id"` // User who performed action
	ActorEmail   string          `json:"actor_email,omitempty" db:"actor_email"`
	ActorIP      string          `json:"actor_ip" db:"actor_ip"`
	ResourceType string          `json:"resource_type" db:"resource_type"` // user, policy, gateway, etc.
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty" db:"resource_id"`
	Details      json.RawMessage `json:"details" db:"details"` // Event-specific data
	Success      bool            `json:"success" db:"success"`
}

// Config represents a generated OpenVPN configuration.
type Config struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	SessionID     uuid.UUID  `json:"session_id" db:"session_id"`
	CertificateID uuid.UUID  `json:"certificate_id" db:"certificate_id"`
	GatewayID     uuid.UUID  `json:"gateway_id" db:"gateway_id"`
	FileName      string     `json:"file_name" db:"file_name"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	DownloadedAt  *time.Time `json:"downloaded_at,omitempty" db:"downloaded_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// NullableIP handles nullable IP addresses in the database.
type NullableIP struct {
	IP    net.IP
	Valid bool
}

// Scan implements the sql.Scanner interface.
func (n *NullableIP) Scan(value interface{}) error {
	if value == nil {
		n.IP, n.Valid = nil, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case string:
		n.IP = net.ParseIP(v)
	case []byte:
		n.IP = net.ParseIP(string(v))
	}
	return nil
}

// StringArray handles PostgreSQL text arrays.
type StringArray []string

// Scan implements the sql.Scanner interface.
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	// PostgreSQL array format: {item1,item2,item3}
	// For simplicity, using JSON array in the database
	return json.Unmarshal(value.([]byte), a)
}

// NullTime is a nullable time.Time.
type NullTime struct {
	sql.NullTime
}

// MarshalJSON implements json.Marshaler.
func (t NullTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time)
}
