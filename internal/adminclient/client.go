package adminclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client provides admin API access.
type Client struct {
	config     *Config
	auth       *AuthManager
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new admin API client.
func NewClient(config *Config) *Client {
	return &Client{
		config:     config,
		auth:       NewAuthManager(config),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    config.ServerURL,
	}
}

// Auth returns the authentication manager.
func (c *Client) Auth() *AuthManager {
	return c.auth
}

// request makes an authenticated HTTP request.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	fullURL.Path = path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	authHeader, err := c.auth.GetAuthHeader()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// doJSON makes a request and decodes JSON response.
func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	resp, err := c.request(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.handleError(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// handleError extracts error message from response.
func (c *Client) handleError(resp *http.Response) error {
	var errBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &errBody)

	msg := errBody.Error
	if msg == "" {
		msg = errBody.Message
	}
	if msg == "" {
		msg = string(body)
	}
	if msg == "" {
		msg = resp.Status
	}

	return fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
}

// === Gateway Operations ===

type Gateway struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	Endpoint            string     `json:"endpoint"`
	PublicIP            string     `json:"public_ip"`
	InternalIP          string     `json:"internal_ip,omitempty"`
	Port                int        `json:"port"`
	Protocol            string     `json:"protocol"`
	Status              string     `json:"status"`
	LastHeartbeat       *time.Time `json:"last_heartbeat,omitempty"`
	ClientCount         int        `json:"client_count"`
	IsMeshHub           bool       `json:"is_mesh_hub"`
	IsProvisioned       bool       `json:"is_provisioned"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (c *Client) ListGateways(ctx context.Context) ([]Gateway, error) {
	var result struct {
		Gateways []Gateway `json:"gateways"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/gateways", nil, &result)
	return result.Gateways, err
}

func (c *Client) GetGateway(ctx context.Context, id string) (*Gateway, error) {
	var gw Gateway
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/gateways/"+id, nil, &gw)
	return &gw, err
}

func (c *Client) CreateGateway(ctx context.Context, req interface{}) (*Gateway, error) {
	var gw Gateway
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/gateways", req, &gw)
	return &gw, err
}

func (c *Client) UpdateGateway(ctx context.Context, id string, req interface{}) (*Gateway, error) {
	var gw Gateway
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/gateways/"+id, req, &gw)
	return &gw, err
}

func (c *Client) DeleteGateway(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/gateways/"+id, nil, nil)
}

type ProvisionResponse struct {
	Gateway   Gateway `json:"gateway"`
	Config    string  `json:"config"`
	Token     string  `json:"token,omitempty"`
	Message   string  `json:"message"`
}

func (c *Client) ProvisionGateway(ctx context.Context, id string) (*ProvisionResponse, error) {
	var result ProvisionResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/gateways/"+id+"/provision", nil, &result)
	return &result, err
}

func (c *Client) ReprovisionGateway(ctx context.Context, id string) (*ProvisionResponse, error) {
	var result ProvisionResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/gateways/"+id+"/reprovision", nil, &result)
	return &result, err
}

// === Network Operations ===

type Network struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CIDR        string    `json:"cidr"`
	GatewayID   string    `json:"gateway_id"`
	GatewayName string    `json:"gateway_name,omitempty"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	var result struct {
		Networks []Network `json:"networks"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/networks", nil, &result)
	return result.Networks, err
}

func (c *Client) GetNetwork(ctx context.Context, id string) (*Network, error) {
	var net Network
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/networks/"+id, nil, &net)
	return &net, err
}

func (c *Client) CreateNetwork(ctx context.Context, req interface{}) (*Network, error) {
	var net Network
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/networks", req, &net)
	return &net, err
}

func (c *Client) UpdateNetwork(ctx context.Context, id string, req interface{}) (*Network, error) {
	var net Network
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/networks/"+id, req, &net)
	return &net, err
}

func (c *Client) DeleteNetwork(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/networks/"+id, nil, nil)
}

// === Access Rule Operations ===

type AccessRule struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	NetworkID     string    `json:"network_id"`
	NetworkName   string    `json:"network_name,omitempty"`
	Groups        []string  `json:"groups"`
	AllowedCIDRs  []string  `json:"allowed_cidrs"`
	IsActive      bool      `json:"is_active"`
	Priority      int       `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c *Client) ListAccessRules(ctx context.Context) ([]AccessRule, error) {
	var result struct {
		AccessRules []AccessRule `json:"access_rules"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/access-rules", nil, &result)
	return result.AccessRules, err
}

func (c *Client) GetAccessRule(ctx context.Context, id string) (*AccessRule, error) {
	var rule AccessRule
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/access-rules/"+id, nil, &rule)
	return &rule, err
}

func (c *Client) CreateAccessRule(ctx context.Context, req interface{}) (*AccessRule, error) {
	var rule AccessRule
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/access-rules", req, &rule)
	return &rule, err
}

func (c *Client) UpdateAccessRule(ctx context.Context, id string, req interface{}) (*AccessRule, error) {
	var rule AccessRule
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/access-rules/"+id, req, &rule)
	return &rule, err
}

func (c *Client) DeleteAccessRule(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/access-rules/"+id, nil, nil)
}

// === User Operations ===

type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Provider    string     `json:"provider"`
	Groups      []string   `json:"groups"`
	IsAdmin     bool       `json:"is_admin"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var result struct {
		Users []User `json:"users"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/users", nil, &result)
	return result.Users, err
}

func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var user User
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/users/"+id, nil, &user)
	return &user, err
}

func (c *Client) UpdateUser(ctx context.Context, id string, req interface{}) (*User, error) {
	var user User
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/users/"+id, req, &user)
	return &user, err
}

func (c *Client) RevokeUserConfigs(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/admin/users/"+id+"/revoke-configs", nil, nil)
}

// === Local User Operations ===

type LocalUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	IsAdmin      bool      `json:"is_admin"`
	IsActive     bool      `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (c *Client) ListLocalUsers(ctx context.Context) ([]LocalUser, error) {
	var result struct {
		Users []LocalUser `json:"users"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/local-users", nil, &result)
	return result.Users, err
}

func (c *Client) CreateLocalUser(ctx context.Context, req interface{}) (*LocalUser, error) {
	var user LocalUser
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/local-users", req, &user)
	return &user, err
}

func (c *Client) DeleteLocalUser(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/local-users/"+id, nil, nil)
}

func (c *Client) ResetLocalUserPassword(ctx context.Context, id string, req interface{}) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/admin/local-users/"+id+"/reset-password", req, nil)
}

// === Group Operations ===

type Group struct {
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
	RuleCount   int    `json:"rule_count"`
}

func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var result struct {
		Groups []Group `json:"groups"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/groups", nil, &result)
	return result.Groups, err
}

func (c *Client) GetGroupMembers(ctx context.Context, name string) ([]User, error) {
	var result struct {
		Members []User `json:"members"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/groups/"+url.PathEscape(name)+"/members", nil, &result)
	return result.Members, err
}

func (c *Client) GetGroupRules(ctx context.Context, name string) ([]AccessRule, error) {
	var result struct {
		Rules []AccessRule `json:"rules"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/groups/"+url.PathEscape(name)+"/rules", nil, &result)
	return result.Rules, err
}

// === API Key Operations ===

type APIKey struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	UserEmail          string     `json:"user_email,omitempty"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	KeyPrefix          string     `json:"key_prefix"`
	Scopes             []string   `json:"scopes"`
	IsAdminProvisioned bool       `json:"is_admin_provisioned"`
	ProvisionedBy      *string    `json:"provisioned_by,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP         *string    `json:"last_used_ip,omitempty"`
	IsRevoked          bool       `json:"is_revoked"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type APIKeyCreateResponse struct {
	APIKey APIKey `json:"api_key"`
	RawKey string `json:"key"`
}

func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var result struct {
		APIKeys []APIKey `json:"api_keys"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/api-keys", nil, &result)
	return result.APIKeys, err
}

func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	var key APIKey
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/api-keys/"+id, nil, &key)
	return &key, err
}

func (c *Client) CreateAPIKey(ctx context.Context, req interface{}) (*APIKeyCreateResponse, error) {
	var result APIKeyCreateResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/api-keys", req, &result)
	return &result, err
}

func (c *Client) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/api-keys/"+id, map[string]string{"reason": reason}, nil)
}

func (c *Client) ListUserAPIKeys(ctx context.Context, userID string) ([]APIKey, error) {
	var result struct {
		APIKeys []APIKey `json:"api_keys"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/users/"+userID+"/api-keys", nil, &result)
	return result.APIKeys, err
}

func (c *Client) CreateUserAPIKey(ctx context.Context, userID string, req interface{}) (*APIKeyCreateResponse, error) {
	var result APIKeyCreateResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/users/"+userID+"/api-keys", req, &result)
	return &result, err
}

func (c *Client) RevokeUserAPIKeys(ctx context.Context, userID string, reason string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/users/"+userID+"/api-keys", map[string]string{"reason": reason}, nil)
}

// === Mesh Hub Operations ===

type MeshHub struct {
	ID                string    `json:"id"`
	GatewayID         string    `json:"gateway_id"`
	GatewayName       string    `json:"gateway_name,omitempty"`
	HubNetwork        string    `json:"hub_network"`
	IsProvisioned     bool      `json:"is_provisioned"`
	ConnectedSpokes   int       `json:"connected_spokes"`
	CreatedAt         time.Time `json:"created_at"`
}

func (c *Client) ListMeshHubs(ctx context.Context) ([]MeshHub, error) {
	var result struct {
		Hubs []MeshHub `json:"hubs"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/mesh/hubs", nil, &result)
	return result.Hubs, err
}

func (c *Client) GetMeshHub(ctx context.Context, id string) (*MeshHub, error) {
	var hub MeshHub
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/mesh/hubs/"+id, nil, &hub)
	return &hub, err
}

func (c *Client) CreateMeshHub(ctx context.Context, req interface{}) (*MeshHub, error) {
	var hub MeshHub
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/mesh/hubs", req, &hub)
	return &hub, err
}

func (c *Client) UpdateMeshHub(ctx context.Context, id string, req interface{}) (*MeshHub, error) {
	var hub MeshHub
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/mesh/hubs/"+id, req, &hub)
	return &hub, err
}

func (c *Client) DeleteMeshHub(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/mesh/hubs/"+id, nil, nil)
}

func (c *Client) ProvisionMeshHub(ctx context.Context, id string) (*ProvisionResponse, error) {
	var result ProvisionResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/mesh/hubs/"+id+"/provision", nil, &result)
	return &result, err
}

// === Mesh Spoke Operations ===

type MeshSpoke struct {
	ID            string    `json:"id"`
	GatewayID     string    `json:"gateway_id"`
	GatewayName   string    `json:"gateway_name,omitempty"`
	HubID         string    `json:"hub_id"`
	HubName       string    `json:"hub_name,omitempty"`
	SpokeNetwork  string    `json:"spoke_network"`
	IsProvisioned bool      `json:"is_provisioned"`
	IsConnected   bool      `json:"is_connected"`
	CreatedAt     time.Time `json:"created_at"`
}

func (c *Client) ListMeshSpokes(ctx context.Context) ([]MeshSpoke, error) {
	var result struct {
		Spokes []MeshSpoke `json:"spokes"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/mesh/spokes", nil, &result)
	return result.Spokes, err
}

func (c *Client) GetMeshSpoke(ctx context.Context, id string) (*MeshSpoke, error) {
	var spoke MeshSpoke
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/mesh/spokes/"+id, nil, &spoke)
	return &spoke, err
}

func (c *Client) CreateMeshSpoke(ctx context.Context, req interface{}) (*MeshSpoke, error) {
	var spoke MeshSpoke
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/mesh/spokes", req, &spoke)
	return &spoke, err
}

func (c *Client) UpdateMeshSpoke(ctx context.Context, id string, req interface{}) (*MeshSpoke, error) {
	var spoke MeshSpoke
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/admin/mesh/spokes/"+id, req, &spoke)
	return &spoke, err
}

func (c *Client) DeleteMeshSpoke(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/admin/mesh/spokes/"+id, nil, nil)
}

func (c *Client) ProvisionMeshSpoke(ctx context.Context, id string) (*ProvisionResponse, error) {
	var result ProvisionResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/mesh/spokes/"+id+"/provision", nil, &result)
	return &result, err
}

// === CA Operations ===

type CA struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	IsActive      bool       `json:"is_active"`
	IsCurrent     bool       `json:"is_current"`
	SerialNumber  string     `json:"serial_number"`
	NotBefore     time.Time  `json:"not_before"`
	NotAfter      time.Time  `json:"not_after"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (c *Client) GetCA(ctx context.Context) (*CA, error) {
	var ca CA
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/ca", nil, &ca)
	return &ca, err
}

func (c *Client) ListCAs(ctx context.Context) ([]CA, error) {
	var result struct {
		CAs []CA `json:"cas"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/ca/list", nil, &result)
	return result.CAs, err
}

func (c *Client) RotateCA(ctx context.Context, req interface{}) (*CA, error) {
	var ca CA
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/ca/rotate", req, &ca)
	return &ca, err
}

func (c *Client) ActivateCA(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/admin/ca/"+id+"/activate", nil, nil)
}

func (c *Client) RevokeCA(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/admin/ca/"+id+"/revoke", nil, nil)
}

// === Audit Log Operations ===

type AuditLog struct {
	ID         string                 `json:"id"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	UserEmail  string                 `json:"user_email,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

func (c *Client) ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	path := "/api/v1/admin/audit"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	var result struct {
		Logs []AuditLog `json:"logs"`
	}
	err := c.doJSON(ctx, http.MethodGet, path, nil, &result)
	return result.Logs, err
}

// === Connection Operations ===

type Connection struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	GatewayID   string    `json:"gateway_id"`
	GatewayName string    `json:"gateway_name"`
	ClientIP    string    `json:"client_ip"`
	VPNAddress  string    `json:"vpn_address"`
	ConnectedAt time.Time `json:"connected_at"`
	BytesSent   int64     `json:"bytes_sent"`
	BytesRecv   int64     `json:"bytes_recv"`
}

func (c *Client) ListConnections(ctx context.Context) ([]Connection, error) {
	var result struct {
		Connections []Connection `json:"connections"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/connections", nil, &result)
	return result.Connections, err
}

func (c *Client) DisconnectUser(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/admin/connections/"+id+"/disconnect", nil, nil)
}

// === System Info ===

type SystemInfo struct {
	Version       string `json:"version"`
	BuildTime     string `json:"build_time"`
	GoVersion     string `json:"go_version"`
	Uptime        string `json:"uptime"`
	TotalGateways int    `json:"total_gateways"`
	TotalUsers    int    `json:"total_users"`
	TotalNetworks int    `json:"total_networks"`
}

func (c *Client) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	var info SystemInfo
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/system/info", nil, &info)
	return &info, err
}

// === Topology Operations ===

type TopologyGateway struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Hostname      string     `json:"hostname"`
	PublicIP      string     `json:"publicIp"`
	VPNPort       int        `json:"vpnPort"`
	VPNProtocol   string     `json:"vpnProtocol"`
	IsActive      bool       `json:"isActive"`
	LastHeartbeat *time.Time `json:"lastHeartbeat"`
	ClientCount   int        `json:"clientCount"`
}

type TopologyMeshHub struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	PublicEndpoint  string     `json:"publicEndpoint"`
	VPNSubnet       string     `json:"vpnSubnet"`
	Status          string     `json:"status"`
	LastHeartbeat   *time.Time `json:"lastHeartbeat"`
	ConnectedSpokes int        `json:"connectedSpokes"`
	ConnectedUsers  int        `json:"connectedUsers"`
}

type TopologyMeshSpoke struct {
	ID            string     `json:"id"`
	HubID         string     `json:"hubId"`
	Name          string     `json:"name"`
	LocalNetworks []string   `json:"localNetworks"`
	TunnelIP      string     `json:"tunnelIp"`
	Status        string     `json:"status"`
	LastSeen      *time.Time `json:"lastSeen"`
	RemoteIP      string     `json:"remoteIp"`
}

type TopologyConnection struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type TopologyResponse struct {
	Gateways    []TopologyGateway    `json:"gateways"`
	MeshHubs    []TopologyMeshHub    `json:"meshHubs"`
	MeshSpokes  []TopologyMeshSpoke  `json:"meshSpokes"`
	Connections []TopologyConnection `json:"connections"`
}

func (c *Client) GetTopology(ctx context.Context) (*TopologyResponse, error) {
	var result TopologyResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/topology", nil, &result)
	return &result, err
}

// === Network Tools Operations ===

type NetworkToolRequest struct {
	Tool     string            `json:"tool"`
	Target   string            `json:"target"`
	Port     int               `json:"port,omitempty"`
	Ports    string            `json:"ports,omitempty"`
	Location string            `json:"location,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type NetworkToolResult struct {
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
	Location  string `json:"location"`
	StartedAt string `json:"startedAt"`
}

type NetworkToolInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
	Required    []string `json:"required,omitempty"`
}

type NetworkToolsInfoResponse struct {
	Tools     []NetworkToolInfo        `json:"tools"`
	Locations []map[string]string `json:"locations"`
}

func (c *Client) GetNetworkToolsInfo(ctx context.Context) (*NetworkToolsInfoResponse, error) {
	var result NetworkToolsInfoResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/network-tools", nil, &result)
	return &result, err
}

func (c *Client) ExecuteNetworkTool(ctx context.Context, req *NetworkToolRequest) (*NetworkToolResult, error) {
	var result NetworkToolResult
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/admin/network-tools/execute", req, &result)
	return &result, err
}

// === Remote Session Operations ===

type RemoteSessionAgent struct {
	AgentID     string    `json:"agentId"`
	NodeType    string    `json:"nodeType"`
	NodeID      string    `json:"nodeId"`
	NodeName    string    `json:"nodeName"`
	ConnectedAt time.Time `json:"connected"`
}

func (c *Client) ListRemoteSessionAgents(ctx context.Context) ([]RemoteSessionAgent, error) {
	var result struct {
		Agents []RemoteSessionAgent `json:"agents"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/admin/remote-session/agents", nil, &result)
	return result.Agents, err
}

// GetWebSocketURL returns the WebSocket URL for remote sessions
func (c *Client) GetWebSocketURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert http(s) to ws(s)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = "/ws/admin/session"

	return u.String(), nil
}
