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
