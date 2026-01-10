package models

import (
	"database/sql"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNullableIP_Scan_Nil(t *testing.T) {
	var ip NullableIP
	err := ip.Scan(nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if ip.Valid {
		t.Error("Expected Valid to be false for nil value")
	}
	if ip.IP != nil {
		t.Error("Expected IP to be nil")
	}
}

func TestNullableIP_Scan_StringIPv4(t *testing.T) {
	var ip NullableIP
	err := ip.Scan("192.168.1.1")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !ip.Valid {
		t.Error("Expected Valid to be true")
	}
	expected := net.ParseIP("192.168.1.1")
	if !ip.IP.Equal(expected) {
		t.Errorf("Expected IP %v, got %v", expected, ip.IP)
	}
}

func TestNullableIP_Scan_StringIPv6(t *testing.T) {
	var ip NullableIP
	err := ip.Scan("2001:db8::1")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !ip.Valid {
		t.Error("Expected Valid to be true")
	}
	expected := net.ParseIP("2001:db8::1")
	if !ip.IP.Equal(expected) {
		t.Errorf("Expected IP %v, got %v", expected, ip.IP)
	}
}

func TestNullableIP_Scan_Bytes(t *testing.T) {
	var ip NullableIP
	err := ip.Scan([]byte("10.0.0.1"))
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !ip.Valid {
		t.Error("Expected Valid to be true")
	}
	expected := net.ParseIP("10.0.0.1")
	if !ip.IP.Equal(expected) {
		t.Errorf("Expected IP %v, got %v", expected, ip.IP)
	}
}

func TestNullableIP_Scan_InvalidIP(t *testing.T) {
	var ip NullableIP
	err := ip.Scan("not-an-ip")
	if err != nil {
		t.Errorf("Expected no error (ParseIP returns nil for invalid), got: %v", err)
	}
	if !ip.Valid {
		t.Error("Expected Valid to be true (parsing happened)")
	}
	if ip.IP != nil {
		t.Error("Expected IP to be nil for invalid IP string")
	}
}

func TestStringArray_Scan_Nil(t *testing.T) {
	var arr StringArray
	err := arr.Scan(nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if arr != nil {
		t.Error("Expected nil array for nil value")
	}
}

func TestStringArray_Scan_JSONArray(t *testing.T) {
	var arr StringArray
	err := arr.Scan([]byte(`["admin", "users", "developers"]`))
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}
	if arr[0] != "admin" {
		t.Errorf("Expected first element 'admin', got '%s'", arr[0])
	}
	if arr[1] != "users" {
		t.Errorf("Expected second element 'users', got '%s'", arr[1])
	}
}

func TestStringArray_Scan_EmptyArray(t *testing.T) {
	var arr StringArray
	err := arr.Scan([]byte(`[]`))
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(arr) != 0 {
		t.Errorf("Expected empty array, got %d elements", len(arr))
	}
}

func TestStringArray_Scan_InvalidJSON(t *testing.T) {
	var arr StringArray
	err := arr.Scan([]byte(`not valid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestNullTime_MarshalJSON_Valid(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	nt := NullTime{
		NullTime: sql.NullTime{
			Time:  now,
			Valid: true,
		},
	}

	data, err := nt.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Unmarshal to verify format
	var result time.Time
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal result: %v", err)
	}
	if !result.Equal(now) {
		t.Errorf("Expected time %v, got %v", now, result)
	}
}

func TestNullTime_MarshalJSON_Null(t *testing.T) {
	nt := NullTime{
		NullTime: sql.NullTime{
			Valid: false,
		},
	}

	data, err := nt.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("Expected 'null', got '%s'", string(data))
	}
}

func TestUser_JSONMarshalling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	user := User{
		ID:         uuid.New(),
		ExternalID: "ext-123",
		Provider:   "google",
		Email:      "test@example.com",
		Name:       "Test User",
		Groups:     []string{"admin", "developers"},
		IsAdmin:    true,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	var result User
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if result.Email != user.Email {
		t.Errorf("Expected email '%s', got '%s'", user.Email, result.Email)
	}
	if result.IsAdmin != user.IsAdmin {
		t.Errorf("Expected IsAdmin %v, got %v", user.IsAdmin, result.IsAdmin)
	}
}

func TestSession_TokenNotInJSON(t *testing.T) {
	session := Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "secret-token-hash",
		IPAddress: "192.168.1.1",
		UserAgent: "Test/1.0",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Token should not be in JSON output (has json:"-" tag)
	if _, exists := result["token"]; exists {
		t.Error("Token should not be in JSON output")
	}
}

func TestGateway_TokenNotInJSON(t *testing.T) {
	gateway := Gateway{
		ID:          uuid.New(),
		Name:        "gateway-1",
		Hostname:    "vpn.example.com",
		PublicIP:    "203.0.113.1",
		VPNPort:     1194,
		VPNProtocol: "udp",
		Token:       "secret-gateway-token",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(gateway)
	if err != nil {
		t.Fatalf("Failed to marshal gateway: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Token should not be in JSON output
	if _, exists := result["token"]; exists {
		t.Error("Token should not be in JSON output")
	}
}

func TestCertificate_Fields(t *testing.T) {
	cert := Certificate{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		SessionID:    uuid.New(),
		SerialNumber: "1234567890ABCDEF",
		Subject:      "CN=test@example.com",
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		Fingerprint:  "AA:BB:CC:DD:EE:FF",
		IsRevoked:    false,
		CreatedAt:    time.Now(),
	}

	if cert.SerialNumber != "1234567890ABCDEF" {
		t.Errorf("Expected serial number '1234567890ABCDEF', got '%s'", cert.SerialNumber)
	}
	if cert.IsRevoked {
		t.Error("Expected certificate not to be revoked")
	}
}

func TestPolicy_Priority(t *testing.T) {
	policy := Policy{
		ID:          uuid.New(),
		Name:        "deny-external",
		Description: "Deny all external access",
		Priority:    10,
		IsEnabled:   true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   uuid.New(),
	}

	if policy.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", policy.Priority)
	}
}

func TestPolicySubject_Everyone(t *testing.T) {
	subject := PolicySubject{
		Everyone: true,
	}

	data, err := json.Marshal(subject)
	if err != nil {
		t.Fatalf("Failed to marshal subject: %v", err)
	}

	var result PolicySubject
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal subject: %v", err)
	}

	if !result.Everyone {
		t.Error("Expected Everyone to be true")
	}
}

func TestPolicySubject_UsersAndGroups(t *testing.T) {
	subject := PolicySubject{
		Users:  []string{"user1@example.com", "user2@example.com"},
		Groups: []string{"developers", "admin"},
	}

	data, err := json.Marshal(subject)
	if err != nil {
		t.Fatalf("Failed to marshal subject: %v", err)
	}

	var result PolicySubject
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal subject: %v", err)
	}

	if len(result.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(result.Users))
	}
	if len(result.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(result.Groups))
	}
}

func TestPolicyResource_Networks(t *testing.T) {
	resource := PolicyResource{
		Networks: []string{"10.0.0.0/8", "192.168.0.0/16"},
		Ports: []PolicyPort{
			{Protocol: "tcp", Port: 80},
			{Protocol: "tcp", FromPort: 443, ToPort: 445},
		},
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	var result PolicyResource
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	if len(result.Networks) != 2 {
		t.Errorf("Expected 2 networks, got %d", len(result.Networks))
	}
	if len(result.Ports) != 2 {
		t.Errorf("Expected 2 port rules, got %d", len(result.Ports))
	}
}

func TestPolicyPort_SinglePort(t *testing.T) {
	port := PolicyPort{
		Protocol: "tcp",
		Port:     443,
	}

	if port.Protocol != "tcp" {
		t.Errorf("Expected protocol 'tcp', got '%s'", port.Protocol)
	}
	if port.Port != 443 {
		t.Errorf("Expected port 443, got %d", port.Port)
	}
}

func TestPolicyPort_Range(t *testing.T) {
	port := PolicyPort{
		Protocol: "udp",
		FromPort: 1024,
		ToPort:   65535,
	}

	if port.FromPort != 1024 {
		t.Errorf("Expected from port 1024, got %d", port.FromPort)
	}
	if port.ToPort != 65535 {
		t.Errorf("Expected to port 65535, got %d", port.ToPort)
	}
}

func TestTimeWindow_Fields(t *testing.T) {
	window := TimeWindow{
		Days:      []string{"mon", "tue", "wed", "thu", "fri"},
		StartTime: "09:00",
		EndTime:   "17:00",
		Timezone:  "America/New_York",
	}

	if len(window.Days) != 5 {
		t.Errorf("Expected 5 days, got %d", len(window.Days))
	}
	if window.StartTime != "09:00" {
		t.Errorf("Expected start time '09:00', got '%s'", window.StartTime)
	}
	if window.Timezone != "America/New_York" {
		t.Errorf("Expected timezone 'America/New_York', got '%s'", window.Timezone)
	}
}

func TestPolicyCondition_SourceIPs(t *testing.T) {
	condition := PolicyCondition{
		SourceIPs: []string{"10.0.0.0/8", "192.168.1.0/24"},
		TimeWindows: []TimeWindow{
			{
				Days:      []string{"mon", "fri"},
				StartTime: "08:00",
				EndTime:   "18:00",
				Timezone:  "UTC",
			},
		},
	}

	if len(condition.SourceIPs) != 2 {
		t.Errorf("Expected 2 source IPs, got %d", len(condition.SourceIPs))
	}
	if len(condition.TimeWindows) != 1 {
		t.Errorf("Expected 1 time window, got %d", len(condition.TimeWindows))
	}
}

func TestConnection_Fields(t *testing.T) {
	now := time.Now()
	conn := Connection{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		SessionID:     uuid.New(),
		CertificateID: uuid.New(),
		GatewayID:     uuid.New(),
		ClientIP:      "203.0.113.50",
		VPNIPv4:       "10.8.0.5",
		VPNIPv6:       "fd00::5",
		BytesSent:     1024000,
		BytesReceived: 2048000,
		ConnectedAt:   now,
	}

	if conn.VPNIPv4 != "10.8.0.5" {
		t.Errorf("Expected VPN IPv4 '10.8.0.5', got '%s'", conn.VPNIPv4)
	}
	if conn.VPNIPv6 != "fd00::5" {
		t.Errorf("Expected VPN IPv6 'fd00::5', got '%s'", conn.VPNIPv6)
	}
	if conn.BytesSent != 1024000 {
		t.Errorf("Expected bytes sent 1024000, got %d", conn.BytesSent)
	}
}

func TestAuditLog_Fields(t *testing.T) {
	actorID := uuid.New()
	resourceID := uuid.New()
	log := AuditLog{
		ID:           uuid.New(),
		Timestamp:    time.Now(),
		Event:        "user.login",
		ActorID:      &actorID,
		ActorEmail:   "admin@example.com",
		ActorIP:      "192.168.1.100",
		ResourceType: "user",
		ResourceID:   &resourceID,
		Details:      json.RawMessage(`{"method": "oidc"}`),
		Success:      true,
	}

	if log.Event != "user.login" {
		t.Errorf("Expected event 'user.login', got '%s'", log.Event)
	}
	if !log.Success {
		t.Error("Expected success to be true")
	}
}

func TestConfig_Fields(t *testing.T) {
	config := Config{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		SessionID:     uuid.New(),
		CertificateID: uuid.New(),
		GatewayID:     uuid.New(),
		FileName:      "user_gateway-1.ovpn",
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		CreatedAt:     time.Now(),
	}

	if config.FileName != "user_gateway-1.ovpn" {
		t.Errorf("Expected file name 'user_gateway-1.ovpn', got '%s'", config.FileName)
	}
	if config.DownloadedAt != nil {
		t.Error("Expected DownloadedAt to be nil initially")
	}
}
