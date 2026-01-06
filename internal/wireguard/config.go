package wireguard

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// ClientConfigRequest contains parameters for generating a client config.
type ClientConfigRequest struct {
	GatewayName         string
	GatewayEndpoint     string // hostname:port or ip:port
	GatewayPublicKey    string
	ClientPrivateKey    string
	ClientAddress       string   // e.g., "10.0.0.2/32"
	AllowedIPs          []string // Networks client can access
	DNS                 []string // Optional DNS servers
	PresharedKey        string   // Optional
	PersistentKeepalive int      // Optional, in seconds (0 = disabled)
	ExpiresAt           time.Time
	UserEmail           string
}

// GeneratedConfig contains the generated WireGuard configuration.
type GeneratedConfig struct {
	Content   []byte
	FileName  string
	ExpiresAt time.Time
}

// ServerConfigRequest contains parameters for generating server-side config.
type ServerConfigRequest struct {
	InterfaceName string
	PrivateKey    string
	Address       string // Server VPN address with CIDR (e.g., "10.0.0.1/24")
	ListenPort    int
	PostUp        string // Optional post-up script
	PostDown      string // Optional post-down script
}

// PeerConfig represents a peer configuration for the server.
type PeerConfig struct {
	PublicKey           string
	PresharedKey        string   // Optional
	AllowedIPs          []string // Client's allowed IPs
	Comment             string   // Optional comment (e.g., user email)
	Endpoint            string   // Optional endpoint (for client-mode connections)
	PersistentKeepalive int      // Optional keepalive interval in seconds (for NAT traversal)
}

const clientTemplate = `# GateKey WireGuard Configuration
# Gateway: {{ .GatewayName }}
# User: {{ .UserEmail }}
# Generated: {{ .GeneratedAt }}
# Expires: {{ .ExpiresAt }}
#
# WARNING: This configuration expires at {{ .ExpiresAt }}
# After expiration, you must generate a new configuration.

[Interface]
PrivateKey = {{ .ClientPrivateKey }}
Address = {{ .ClientAddress }}
{{- if .DNS }}
DNS = {{ joinComma .DNS }}
{{- end }}

[Peer]
PublicKey = {{ .GatewayPublicKey }}
Endpoint = {{ .GatewayEndpoint }}
AllowedIPs = {{ joinComma .AllowedIPs }}
{{- if .PresharedKey }}
PresharedKey = {{ .PresharedKey }}
{{- end }}
{{- if gt .PersistentKeepalive 0 }}
PersistentKeepalive = {{ .PersistentKeepalive }}
{{- end }}
`

const serverTemplate = `# GateKey WireGuard Server Configuration
# Generated: {{ .GeneratedAt }}

[Interface]
PrivateKey = {{ .PrivateKey }}
Address = {{ .Address }}
ListenPort = {{ .ListenPort }}
{{- if .PostUp }}
PostUp = {{ .PostUp }}
{{- end }}
{{- if .PostDown }}
PostDown = {{ .PostDown }}
{{- end }}

{{ range .Peers }}
# {{ .Comment }}
[Peer]
PublicKey = {{ .PublicKey }}
AllowedIPs = {{ joinComma .AllowedIPs }}
{{- if .PresharedKey }}
PresharedKey = {{ .PresharedKey }}
{{- end }}
{{ end }}
`

var templateFuncs = template.FuncMap{
	"joinComma": func(items []string) string {
		return strings.Join(items, ", ")
	},
}

// GenerateClientConfig generates a WireGuard .conf file for a client.
func GenerateClientConfig(req ClientConfigRequest) (*GeneratedConfig, error) {
	tmpl, err := template.New("wgconf").Funcs(templateFuncs).Parse(clientTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure AllowedIPs is not empty
	if len(req.AllowedIPs) == 0 {
		req.AllowedIPs = []string{"0.0.0.0/0", "::/0"} // Full tunnel if no specific routes
	}

	data := struct {
		ClientConfigRequest
		GeneratedAt string
	}{
		ClientConfigRequest: req,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	fileName := fmt.Sprintf("gatekey-wg-%s-%s.conf",
		sanitizeFileName(req.GatewayName),
		req.ExpiresAt.Format("20060102-1504"))

	return &GeneratedConfig{
		Content:   buf.Bytes(),
		FileName:  fileName,
		ExpiresAt: req.ExpiresAt,
	}, nil
}

// GenerateServerConfig generates a WireGuard server configuration.
func GenerateServerConfig(req ServerConfigRequest, peers []PeerConfig) ([]byte, error) {
	tmpl, err := template.New("wgserver").Funcs(templateFuncs).Parse(serverTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ServerConfigRequest
		Peers       []PeerConfig
		GeneratedAt string
	}{
		ServerConfigRequest: req,
		Peers:               peers,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// sanitizeFileName removes or replaces characters that are unsafe in filenames.
func sanitizeFileName(name string) string {
	// Replace spaces and special characters
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
}

// DefaultPersistentKeepalive is the default keepalive interval for peers behind NAT.
const DefaultPersistentKeepalive = 25

// DefaultListenPort is the default WireGuard listen port.
const DefaultListenPort = 51820
