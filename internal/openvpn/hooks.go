// Package openvpn provides hook handlers for OpenVPN integration.
package openvpn

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// HookType represents the type of OpenVPN hook.
type HookType string

const (
	HookAuthUserPassVerify HookType = "auth-user-pass-verify"
	HookTLSVerify          HookType = "tls-verify"
	HookClientConnect      HookType = "client-connect"
	HookClientDisconnect   HookType = "client-disconnect"
)

// HookRequest represents a request from an OpenVPN hook.
type HookRequest struct {
	Token          string            `json:"token"`
	Type           HookType          `json:"type"`
	CommonName     string            `json:"common_name"`
	Username       string            `json:"username,omitempty"`
	Password       string            `json:"password,omitempty"` // auth-user-pass password (auth token)
	TrustedIP      string            `json:"trusted_ip"`
	UntrustedIP    string            `json:"untrusted_ip"`
	UntrustedPort  string            `json:"untrusted_port"`
	TLSSerial      string            `json:"tls_serial,omitempty"`
	TLSFingerprint string            `json:"tls_fingerprint,omitempty"`
	IFConfigLocal  string            `json:"ifconfig_local,omitempty"`
	IFConfigRemote string            `json:"ifconfig_remote,omitempty"`
	BytesReceived  int64             `json:"bytes_received,omitempty"`
	BytesSent      int64             `json:"bytes_sent,omitempty"`
	TimeConnected  int64             `json:"time_connected,omitempty"`
	Env            map[string]string `json:"env"`
}

// HookResponse represents a response to an OpenVPN hook.
type HookResponse struct {
	Allow       bool     `json:"allow"`
	Message     string   `json:"message,omitempty"`
	ClientConfig []string `json:"client_config,omitempty"`
}

// HookClient communicates with the GateKey control plane from hooks.
type HookClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewHookClient creates a new hook client.
func NewHookClient(baseURL, token string) *HookClient {
	return &HookClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Verify sends a verification request to the control plane.
func (c *HookClient) Verify(req HookRequest) (*HookResponse, error) {
	// Add token to request
	verifyReq := struct {
		Token        string `json:"token"`
		CommonName   string `json:"common_name"`
		Username     string `json:"username,omitempty"`
		Password     string `json:"password,omitempty"` // auth token
		SerialNumber string `json:"serial_number,omitempty"`
		ClientIP     string `json:"client_ip,omitempty"`
	}{
		Token:        c.token,
		CommonName:   req.CommonName,
		Username:     req.Username,
		Password:     req.Password,
		SerialNumber: req.TLSSerial,
		ClientIP:     req.UntrustedIP,
	}

	body, err := json.Marshal(verifyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/gateway/verify", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Allowed     bool   `json:"allowed"`
		Reason      string `json:"reason,omitempty"`
		GatewayID   string `json:"gateway_id,omitempty"`
		GatewayName string `json:"gateway_name,omitempty"`
		Error       string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &HookResponse{
		Allow:   apiResp.Allowed,
		Message: apiResp.Reason,
	}, nil
}

// Connect sends a connect notification to the control plane.
func (c *HookClient) Connect(req HookRequest) (*HookResponse, error) {
	connectReq := struct {
		Token        string `json:"token"`
		CommonName   string `json:"common_name"`
		ClientIP     string `json:"client_ip"`
		VPNIPv4      string `json:"vpn_ipv4,omitempty"`
		VPNIPv6      string `json:"vpn_ipv6,omitempty"`
		SerialNumber string `json:"serial_number,omitempty"`
	}{
		Token:        c.token,
		CommonName:   req.CommonName,
		ClientIP:     req.UntrustedIP,
		VPNIPv4:      req.IFConfigRemote,
		SerialNumber: req.TLSSerial,
	}

	body, err := json.Marshal(connectReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/gateway/connect", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var hookResp HookResponse
	if err := json.NewDecoder(resp.Body).Decode(&hookResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &hookResp, nil
}

// Disconnect sends a disconnect notification to the control plane.
func (c *HookClient) Disconnect(req HookRequest) error {
	disconnectReq := struct {
		Token      string `json:"token"`
		CommonName string `json:"common_name"`
		ClientIP   string `json:"client_ip,omitempty"`
		Duration   int64  `json:"duration_seconds,omitempty"`
		BytesSent  int64  `json:"bytes_sent,omitempty"`
		BytesRecv  int64  `json:"bytes_received,omitempty"`
	}{
		Token:      c.token,
		CommonName: req.CommonName,
		ClientIP:   req.UntrustedIP,
		Duration:   req.TimeConnected,
		BytesSent:  req.BytesSent,
		BytesRecv:  req.BytesReceived,
	}

	body, err := json.Marshal(disconnectReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/gateway/disconnect", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("disconnect failed with status: %d", resp.StatusCode)
	}

	return nil
}

// HeartbeatResponse contains the response from a heartbeat request.
type HeartbeatResponse struct {
	Status           string `json:"status"`
	GatewayID        string `json:"gateway_id"`
	GatewayName      string `json:"gateway_name"`
	ConfigVersion    string `json:"config_version"`
	NeedsReprovision bool   `json:"needs_reprovision"`
}

// Heartbeat sends a heartbeat to the control plane.
// Returns the server's config version and whether reprovision is needed.
func (c *HookClient) Heartbeat(publicIP string, activeClients int, openvpnRunning bool, configVersion string) (*HeartbeatResponse, error) {
	heartbeatReq := struct {
		Token          string `json:"token"`
		PublicIP       string `json:"public_ip,omitempty"`
		ActiveClients  int    `json:"active_clients"`
		OpenVPNRunning bool   `json:"openvpn_running"`
		ConfigVersion  string `json:"config_version,omitempty"`
	}{
		Token:          c.token,
		PublicIP:       publicIP,
		ActiveClients:  activeClients,
		OpenVPNRunning: openvpnRunning,
		ConfigVersion:  configVersion,
	}

	body, err := json.Marshal(heartbeatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/gateway/heartbeat", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	var result HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ProvisionResponse contains the response from a provision request.
type ProvisionResponse struct {
	GatewayID      string `json:"gateway_id"`
	GatewayName    string `json:"gateway_name"`
	CACert         string `json:"ca_cert"`
	ServerCert     string `json:"server_cert"`
	ServerKey      string `json:"server_key"`
	VPNSubnet      string `json:"vpn_subnet"`
	VPNNetwork     string `json:"vpn_network"`
	VPNNetmask     string `json:"vpn_netmask"`
	VPNPort        int    `json:"vpn_port"`
	VPNProtocol    string `json:"vpn_protocol"`
	CryptoProfile  string `json:"crypto_profile"`
	TLSAuthEnabled bool   `json:"tls_auth_enabled"`
	TLSAuthKey     string `json:"tls_auth_key,omitempty"`
}

// Provision requests new certificates and configuration from the control plane.
func (c *HookClient) Provision() (*ProvisionResponse, error) {
	provisionReq := struct {
		Token string `json:"token"`
	}{
		Token: c.token,
	}

	body, err := json.Marshal(provisionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/gateway/provision", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provision failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ParseEnvFile parses the environment file passed by OpenVPN's via-file method.
func ParseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return env, scanner.Err()
}

// GetOpenVPNEnv extracts relevant OpenVPN environment variables.
func GetOpenVPNEnv() map[string]string {
	env := make(map[string]string)

	envVars := []string{
		"common_name",
		"username",
		"password",
		"trusted_ip",
		"trusted_port",
		"untrusted_ip",
		"untrusted_port",
		"tls_serial_0",
		"tls_serial_hex_0",
		"tls_digest_0",
		"tls_digest_sha256_0",
		"ifconfig_local",
		"ifconfig_pool_remote_ip",
		"ifconfig_pool_netmask",
		"bytes_received",
		"bytes_sent",
		"time_duration",
		"script_type",
		"dev",
		"daemon",
		"daemon_log_redirect",
	}

	for _, v := range envVars {
		if val := os.Getenv(v); val != "" {
			env[v] = val
		}
	}

	return env
}

// BuildHookRequest builds a HookRequest from environment variables.
func BuildHookRequest(hookType HookType) HookRequest {
	env := GetOpenVPNEnv()

	return HookRequest{
		Type:           hookType,
		CommonName:     env["common_name"],
		Username:       env["username"],
		Password:       env["password"], // auth token from auth-user-pass
		TrustedIP:      env["trusted_ip"],
		UntrustedIP:    env["untrusted_ip"],
		UntrustedPort:  env["untrusted_port"],
		TLSSerial:      env["tls_serial_hex_0"],
		TLSFingerprint: env["tls_digest_sha256_0"],
		IFConfigLocal:  env["ifconfig_local"],
		IFConfigRemote: env["ifconfig_pool_remote_ip"],
		Env:            env,
	}
}

// WriteClientConfig writes the client configuration file for OpenVPN.
func WriteClientConfig(path string, config []string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create client config: %w", err)
	}
	defer file.Close()

	for _, line := range config {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write config line: %w", err)
		}
	}

	return nil
}
