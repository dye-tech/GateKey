package openvpn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHookType_Constants(t *testing.T) {
	if HookAuthUserPassVerify != "auth-user-pass-verify" {
		t.Errorf("Expected 'auth-user-pass-verify', got '%s'", HookAuthUserPassVerify)
	}
	if HookTLSVerify != "tls-verify" {
		t.Errorf("Expected 'tls-verify', got '%s'", HookTLSVerify)
	}
	if HookClientConnect != "client-connect" {
		t.Errorf("Expected 'client-connect', got '%s'", HookClientConnect)
	}
	if HookClientDisconnect != "client-disconnect" {
		t.Errorf("Expected 'client-disconnect', got '%s'", HookClientDisconnect)
	}
}

func TestNewHookClient(t *testing.T) {
	client := NewHookClient("https://api.example.com/", "secret-token")

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL without trailing slash, got '%s'", client.baseURL)
	}
	if client.token != "secret-token" {
		t.Errorf("Expected token 'secret-token', got '%s'", client.token)
	}
	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestNewHookClient_NoTrailingSlash(t *testing.T) {
	client := NewHookClient("https://api.example.com", "token")
	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}
}

func TestHookRequest_Fields(t *testing.T) {
	req := HookRequest{
		Token:          "token-123",
		Type:           HookClientConnect,
		CommonName:     "user@example.com",
		Username:       "user",
		Password:       "auth-token",
		TrustedIP:      "10.0.0.1",
		UntrustedIP:    "203.0.113.50",
		UntrustedPort:  "51234",
		TLSSerial:      "1234567890",
		TLSFingerprint: "AA:BB:CC:DD",
		IFConfigLocal:  "10.8.0.1",
		IFConfigRemote: "10.8.0.5",
		BytesReceived:  1024,
		BytesSent:      2048,
		TimeConnected:  3600,
		Env:            map[string]string{"key": "value"},
	}

	if req.CommonName != "user@example.com" {
		t.Errorf("Expected CommonName 'user@example.com', got '%s'", req.CommonName)
	}
	if req.BytesSent != 2048 {
		t.Errorf("Expected BytesSent 2048, got %d", req.BytesSent)
	}
}

func TestHookResponse_Fields(t *testing.T) {
	resp := HookResponse{
		Allow:        true,
		Message:      "Access granted",
		ClientConfig: []string{"push \"route 10.0.0.0 255.0.0.0\"", "push \"dhcp-option DNS 8.8.8.8\""},
	}

	if !resp.Allow {
		t.Error("Expected Allow to be true")
	}
	if len(resp.ClientConfig) != 2 {
		t.Errorf("Expected 2 client config lines, got %d", len(resp.ClientConfig))
	}
}

func TestHookClient_Verify(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateway/verify" {
			t.Errorf("Expected path '/api/v1/gateway/verify', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": true,
			"reason":  "Certificate valid",
		})
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	resp, err := client.Verify(HookRequest{
		CommonName:  "user@example.com",
		TLSSerial:   "123456",
		UntrustedIP: "192.168.1.100",
	})

	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !resp.Allow {
		t.Error("Expected Allow to be true")
	}
	if resp.Message != "Certificate valid" {
		t.Errorf("Expected message 'Certificate valid', got '%s'", resp.Message)
	}
}

func TestHookClient_Verify_Denied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": false,
			"reason":  "Certificate revoked",
		})
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	resp, err := client.Verify(HookRequest{CommonName: "user@example.com"})

	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if resp.Allow {
		t.Error("Expected Allow to be false")
	}
	if resp.Message != "Certificate revoked" {
		t.Errorf("Expected message 'Certificate revoked', got '%s'", resp.Message)
	}
}

func TestHookClient_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateway/connect" {
			t.Errorf("Expected path '/api/v1/gateway/connect', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HookResponse{
			Allow:        true,
			Message:      "Connected",
			ClientConfig: []string{"push \"route 10.0.0.0 255.0.0.0\""},
		})
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	resp, err := client.Connect(HookRequest{
		CommonName:     "user@example.com",
		UntrustedIP:    "192.168.1.100",
		IFConfigRemote: "10.8.0.5",
	})

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !resp.Allow {
		t.Error("Expected Allow to be true")
	}
	if len(resp.ClientConfig) != 1 {
		t.Errorf("Expected 1 client config, got %d", len(resp.ClientConfig))
	}
}

func TestHookClient_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateway/disconnect" {
			t.Errorf("Expected path '/api/v1/gateway/disconnect', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	err := client.Disconnect(HookRequest{
		CommonName:    "user@example.com",
		UntrustedIP:   "192.168.1.100",
		BytesSent:     1000,
		BytesReceived: 2000,
		TimeConnected: 3600,
	})

	if err != nil {
		t.Errorf("Disconnect failed: %v", err)
	}
}

func TestHookClient_Disconnect_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	err := client.Disconnect(HookRequest{CommonName: "user@example.com"})

	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestHookClient_Heartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateway/heartbeat" {
			t.Errorf("Expected path '/api/v1/gateway/heartbeat', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HeartbeatResponse{
			Status:           "ok",
			GatewayID:        "gw-123",
			GatewayName:      "gateway-1",
			ConfigVersion:    "v2",
			NeedsReprovision: false,
		})
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	resp, err := client.Heartbeat("203.0.113.1", 5, true, "v1")

	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
	if resp.GatewayID != "gw-123" {
		t.Errorf("Expected gateway ID 'gw-123', got '%s'", resp.GatewayID)
	}
}

func TestHookClient_Heartbeat_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "invalid-token")
	_, err := client.Heartbeat("203.0.113.1", 0, false, "")

	if err == nil {
		t.Error("Expected error for 401 response")
	}
}

func TestHookClient_Provision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateway/provision" {
			t.Errorf("Expected path '/api/v1/gateway/provision', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ProvisionResponse{
			GatewayID:      "gw-123",
			GatewayName:    "gateway-1",
			CACert:         "-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----",
			ServerCert:     "-----BEGIN CERTIFICATE-----\nSERVER\n-----END CERTIFICATE-----",
			ServerKey:      "-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----",
			VPNSubnet:      "10.8.0.0/24",
			VPNNetwork:     "10.8.0.0",
			VPNNetmask:     "255.255.255.0",
			VPNPort:        1194,
			VPNProtocol:    "udp",
			CryptoProfile:  "aes-256-gcm",
			TLSAuthEnabled: true,
			TLSAuthKey:     "tls-auth-key-data",
		})
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	resp, err := client.Provision()

	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}
	if resp.GatewayID != "gw-123" {
		t.Errorf("Expected gateway ID 'gw-123', got '%s'", resp.GatewayID)
	}
	if resp.VPNPort != 1194 {
		t.Errorf("Expected VPN port 1194, got %d", resp.VPNPort)
	}
	if !resp.TLSAuthEnabled {
		t.Error("Expected TLSAuthEnabled to be true")
	}
}

func TestHookClient_Provision_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	client := NewHookClient(server.URL, "test-token")
	_, err := client.Provision()

	if err == nil {
		t.Error("Expected error for 403 response")
	}
}

func TestParseEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env")

	content := `common_name=user@example.com
trusted_ip=10.0.0.1
untrusted_ip=192.168.1.100
bytes_sent=1024
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	env, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}

	if env["common_name"] != "user@example.com" {
		t.Errorf("Expected common_name 'user@example.com', got '%s'", env["common_name"])
	}
	if env["bytes_sent"] != "1024" {
		t.Errorf("Expected bytes_sent '1024', got '%s'", env["bytes_sent"])
	}
}

func TestParseEnvFile_NonExistent(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParseEnvFile_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env")

	content := `key1=value1

key2=value2
malformed_line
key3=value3
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	env, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}

	if env["key1"] != "value1" {
		t.Errorf("Expected key1 'value1', got '%s'", env["key1"])
	}
	if env["key2"] != "value2" {
		t.Errorf("Expected key2 'value2', got '%s'", env["key2"])
	}
	if env["key3"] != "value3" {
		t.Errorf("Expected key3 'value3', got '%s'", env["key3"])
	}
}

func TestWriteClientConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "client.conf")

	config := []string{
		"push \"route 10.0.0.0 255.0.0.0\"",
		"push \"dhcp-option DNS 8.8.8.8\"",
		"ifconfig-push 10.8.0.5 255.255.255.0",
	}

	err := WriteClientConfig(configFile, config)
	if err != nil {
		t.Fatalf("WriteClientConfig failed: %v", err)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expected := "push \"route 10.0.0.0 255.0.0.0\"\npush \"dhcp-option DNS 8.8.8.8\"\nifconfig-push 10.8.0.5 255.255.255.0\n"
	if string(content) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(content))
	}
}

func TestWriteClientConfig_InvalidPath(t *testing.T) {
	err := WriteClientConfig("/nonexistent/directory/client.conf", []string{"test"})
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestHeartbeatResponse_Fields(t *testing.T) {
	resp := HeartbeatResponse{
		Status:           "ok",
		GatewayID:        "gw-123",
		GatewayName:      "gateway-prod-1",
		ConfigVersion:    "v5",
		NeedsReprovision: true,
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
	if !resp.NeedsReprovision {
		t.Error("Expected NeedsReprovision to be true")
	}
}

func TestProvisionResponse_Fields(t *testing.T) {
	resp := ProvisionResponse{
		GatewayID:      "gw-456",
		GatewayName:    "gateway-eu-1",
		CACert:         "ca-cert-pem",
		ServerCert:     "server-cert-pem",
		ServerKey:      "server-key-pem",
		VPNSubnet:      "10.8.0.0/24",
		VPNNetwork:     "10.8.0.0",
		VPNNetmask:     "255.255.255.0",
		VPNPort:        443,
		VPNProtocol:    "tcp",
		CryptoProfile:  "chacha20-poly1305",
		TLSAuthEnabled: false,
		TLSAuthKey:     "",
	}

	if resp.VPNProtocol != "tcp" {
		t.Errorf("Expected VPNProtocol 'tcp', got '%s'", resp.VPNProtocol)
	}
	if resp.VPNPort != 443 {
		t.Errorf("Expected VPNPort 443, got %d", resp.VPNPort)
	}
}

func TestGetOpenVPNEnv(t *testing.T) {
	// Save and restore original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"common_name", "username", "password", "trusted_ip",
		"untrusted_ip", "untrusted_port", "ifconfig_pool_remote_ip",
	}
	for _, v := range envVars {
		originalEnv[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("common_name", "test@example.com")
	os.Setenv("username", "testuser")
	os.Setenv("trusted_ip", "10.0.0.1")
	os.Setenv("untrusted_ip", "192.168.1.100")
	os.Setenv("untrusted_port", "51234")
	os.Setenv("ifconfig_pool_remote_ip", "10.8.0.5")

	env := GetOpenVPNEnv()

	if env["common_name"] != "test@example.com" {
		t.Errorf("Expected common_name 'test@example.com', got '%s'", env["common_name"])
	}
	if env["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", env["username"])
	}
	if env["trusted_ip"] != "10.0.0.1" {
		t.Errorf("Expected trusted_ip '10.0.0.1', got '%s'", env["trusted_ip"])
	}
	if env["untrusted_ip"] != "192.168.1.100" {
		t.Errorf("Expected untrusted_ip '192.168.1.100', got '%s'", env["untrusted_ip"])
	}
	if env["ifconfig_pool_remote_ip"] != "10.8.0.5" {
		t.Errorf("Expected ifconfig_pool_remote_ip '10.8.0.5', got '%s'", env["ifconfig_pool_remote_ip"])
	}

	// Empty env vars should not be included
	os.Unsetenv("password")
	env = GetOpenVPNEnv()
	if _, exists := env["password"]; exists {
		t.Error("Expected password to not be in env map when not set")
	}
}

func TestGetOpenVPNEnv_Empty(t *testing.T) {
	// Save and restore original environment
	envVars := []string{
		"common_name", "username", "password", "trusted_ip",
		"untrusted_ip", "untrusted_port",
	}
	originalEnv := make(map[string]string)
	for _, v := range envVars {
		originalEnv[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	env := GetOpenVPNEnv()

	// Should return empty map when no env vars are set
	if len(env) != 0 {
		t.Errorf("Expected empty env map, got %d entries", len(env))
	}
}

func TestBuildHookRequest(t *testing.T) {
	// Save and restore original environment
	envVars := []string{
		"common_name", "username", "password", "trusted_ip",
		"untrusted_ip", "untrusted_port", "tls_serial_hex_0",
		"tls_digest_sha256_0", "ifconfig_local", "ifconfig_pool_remote_ip",
	}
	originalEnv := make(map[string]string)
	for _, v := range envVars {
		originalEnv[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("common_name", "user@example.com")
	os.Setenv("username", "testuser")
	os.Setenv("password", "auth-token-123")
	os.Setenv("trusted_ip", "10.0.0.1")
	os.Setenv("untrusted_ip", "192.168.1.100")
	os.Setenv("untrusted_port", "51234")
	os.Setenv("tls_serial_hex_0", "AABBCCDD")
	os.Setenv("tls_digest_sha256_0", "sha256fingerprint")
	os.Setenv("ifconfig_local", "10.8.0.1")
	os.Setenv("ifconfig_pool_remote_ip", "10.8.0.5")

	req := BuildHookRequest(HookClientConnect)

	if req.Type != HookClientConnect {
		t.Errorf("Expected Type '%s', got '%s'", HookClientConnect, req.Type)
	}
	if req.CommonName != "user@example.com" {
		t.Errorf("Expected CommonName 'user@example.com', got '%s'", req.CommonName)
	}
	if req.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", req.Username)
	}
	if req.Password != "auth-token-123" {
		t.Errorf("Expected Password 'auth-token-123', got '%s'", req.Password)
	}
	if req.TrustedIP != "10.0.0.1" {
		t.Errorf("Expected TrustedIP '10.0.0.1', got '%s'", req.TrustedIP)
	}
	if req.UntrustedIP != "192.168.1.100" {
		t.Errorf("Expected UntrustedIP '192.168.1.100', got '%s'", req.UntrustedIP)
	}
	if req.UntrustedPort != "51234" {
		t.Errorf("Expected UntrustedPort '51234', got '%s'", req.UntrustedPort)
	}
	if req.TLSSerial != "AABBCCDD" {
		t.Errorf("Expected TLSSerial 'AABBCCDD', got '%s'", req.TLSSerial)
	}
	if req.TLSFingerprint != "sha256fingerprint" {
		t.Errorf("Expected TLSFingerprint 'sha256fingerprint', got '%s'", req.TLSFingerprint)
	}
	if req.IFConfigLocal != "10.8.0.1" {
		t.Errorf("Expected IFConfigLocal '10.8.0.1', got '%s'", req.IFConfigLocal)
	}
	if req.IFConfigRemote != "10.8.0.5" {
		t.Errorf("Expected IFConfigRemote '10.8.0.5', got '%s'", req.IFConfigRemote)
	}

	// Verify Env map is populated
	if req.Env == nil {
		t.Error("Expected Env map to be populated")
	}
	if req.Env["common_name"] != "user@example.com" {
		t.Errorf("Expected Env[common_name] 'user@example.com', got '%s'", req.Env["common_name"])
	}
}

func TestBuildHookRequest_AllHookTypes(t *testing.T) {
	hookTypes := []HookType{
		HookAuthUserPassVerify,
		HookTLSVerify,
		HookClientConnect,
		HookClientDisconnect,
	}

	for _, hookType := range hookTypes {
		t.Run(string(hookType), func(t *testing.T) {
			req := BuildHookRequest(hookType)
			if req.Type != hookType {
				t.Errorf("Expected Type '%s', got '%s'", hookType, req.Type)
			}
		})
	}
}
