// Package openvpn provides OpenVPN configuration generation and integration.
package openvpn

import (
	"bytes"
	cryptoRand "crypto/rand"
	"fmt"
	"net"
	"strings"
	"text/template"
	"time"

	"github.com/gatekey-project/gatekey/internal/models"
	"github.com/gatekey-project/gatekey/internal/pki"
)

// isIPAddress checks if a string is an IP address (not a hostname)
func isIPAddress(addr string) bool {
	return net.ParseIP(addr) != nil
}

// ConfigGenerator generates OpenVPN configuration files.
type ConfigGenerator struct {
	caPEM    []byte
	tlsAuth  []byte // Optional TLS-Auth key
	template *template.Template
}

// NewConfigGenerator creates a new OpenVPN configuration generator.
func NewConfigGenerator(ca *pki.CA, tlsAuthKey []byte) (*ConfigGenerator, error) {
	tmpl, err := template.New("ovpn").Parse(ovpnTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &ConfigGenerator{
		caPEM:    ca.CertificatePEM(),
		tlsAuth:  tlsAuthKey,
		template: tmpl,
	}, nil
}

// CryptoProfile constants
const (
	CryptoProfileModern     = "modern"     // Modern secure defaults
	CryptoProfileFIPS       = "fips"       // FIPS 140-3 compliant
	CryptoProfileCompatible = "compatible" // Maximum compatibility
)

// GenerateRequest contains parameters for generating an OpenVPN config.
type GenerateRequest struct {
	Gateway       *models.Gateway
	User          *models.User
	Certificate   *pki.IssuedCertificate
	ExpiresAt     time.Time
	Routes        []Route
	DNS           []string
	Options       map[string]string
	CryptoProfile string // "modern", "fips", or "compatible"
	TLSAuthKey    string // Gateway-specific TLS-Auth key (overrides generator's default)
	AuthToken     string // Unique token for password authentication (embedded in config)
}

// Route represents a route to push to the client.
type Route struct {
	Network string
	Netmask string
}

// GeneratedConfig contains the generated OpenVPN configuration.
type GeneratedConfig struct {
	Content   []byte
	FileName  string
	ExpiresAt time.Time
}

// CryptoSettings contains crypto-specific configuration for each profile.
type CryptoSettings struct {
	Cipher        string
	Auth          string
	TLSVersionMin string
	TLSCipher     string
	DataCiphers   string // OpenVPN 2.5+ data-ciphers directive
	CryptoProfile string // For display in config
}

// GetCryptoSettings returns the crypto settings for a given profile.
func GetCryptoSettings(profile string) CryptoSettings {
	switch profile {
	case CryptoProfileFIPS:
		// FIPS 140-3 compliant settings
		// Uses only FIPS-approved algorithms: AES-GCM, SHA-256/384/512, TLS 1.2+
		return CryptoSettings{
			Cipher:        "AES-256-GCM",
			Auth:          "SHA384",
			TLSVersionMin: "1.2",
			TLSCipher:     "TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-RSA-WITH-AES-256-GCM-SHA384",
			DataCiphers:   "AES-256-GCM:AES-128-GCM",
			CryptoProfile: "FIPS 140-3 Compliant",
		}
	case CryptoProfileCompatible:
		// Maximum compatibility with older clients
		return CryptoSettings{
			Cipher:        "AES-256-CBC",
			Auth:          "SHA256",
			TLSVersionMin: "1.0",
			TLSCipher:     "", // Let OpenVPN negotiate
			DataCiphers:   "AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC",
			CryptoProfile: "Compatible (Legacy Support)",
		}
	default: // CryptoProfileModern
		// Modern secure defaults (ECDSA preferred, strong ciphers only)
		return CryptoSettings{
			Cipher:        "AES-256-GCM",
			Auth:          "SHA256",
			TLSVersionMin: "1.2",
			TLSCipher:     "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-ECDSA-WITH-CHACHA20-POLY1305-SHA256:TLS-ECDHE-RSA-WITH-CHACHA20-POLY1305-SHA256",
			DataCiphers:   "AES-256-GCM:CHACHA20-POLY1305",
			CryptoProfile: "Modern (Secure Defaults)",
		}
	}
}

// configData contains data for the template.
type configData struct {
	GatewayHostname  string
	GatewayPort      int
	Protocol         string
	CACert           string
	ClientCert       string
	ClientKey        string
	TLSAuth          string
	TLSAuthDirection string
	AuthUsername     string // Username for auth-user-pass (user email)
	AuthPassword     string // Password for auth-user-pass (auth token)
	Routes           []Route
	DNS              []string
	ExpiresAt        string
	UserEmail        string
	GatewayName      string
	Options          map[string]string
	Crypto           CryptoSettings
}

// Generate generates an OpenVPN configuration file.
func (g *ConfigGenerator) Generate(req GenerateRequest) (*GeneratedConfig, error) {
	protocol := strings.ToLower(req.Gateway.VPNProtocol)
	if protocol == "" {
		protocol = "udp"
	}

	// Use hostname if available, otherwise fall back to public IP
	gatewayAddress := req.Gateway.Hostname
	if gatewayAddress == "" {
		gatewayAddress = req.Gateway.PublicIP
	}

	// Get crypto settings based on profile
	cryptoProfile := req.CryptoProfile
	if cryptoProfile == "" {
		cryptoProfile = CryptoProfileModern
	}
	crypto := GetCryptoSettings(cryptoProfile)

	data := configData{
		GatewayHostname: gatewayAddress,
		GatewayPort:     req.Gateway.VPNPort,
		Protocol:        protocol,
		CACert:          string(g.caPEM),
		ClientCert:      string(req.Certificate.CertificatePEM),
		ClientKey:       string(req.Certificate.PrivateKeyPEM),
		AuthUsername:    req.User.Email, // Use email as username
		AuthPassword:    req.AuthToken,  // Use unique token as password
		Routes:          req.Routes,
		DNS:             req.DNS,
		ExpiresAt:       req.ExpiresAt.UTC().Format(time.RFC3339),
		UserEmail:       req.User.Email,
		GatewayName:     req.Gateway.Name,
		Options:         req.Options,
		Crypto:          crypto,
	}

	// Only include TLS-Auth if enabled for this gateway
	// Use gateway-specific key from request, fall back to generator's default
	if req.Gateway.TLSAuthEnabled {
		tlsKey := req.TLSAuthKey
		if tlsKey == "" && len(g.tlsAuth) > 0 {
			tlsKey = string(g.tlsAuth)
		}
		if tlsKey != "" {
			data.TLSAuth = tlsKey
			data.TLSAuthDirection = "1" // Client direction
		}
	}

	var buf bytes.Buffer
	if err := g.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Generate filename
	fileName := fmt.Sprintf("gatekey-%s-%s.ovpn",
		sanitizeFileName(req.Gateway.Name),
		req.ExpiresAt.Format("20060102-1504"))

	return &GeneratedConfig{
		Content:   buf.Bytes(),
		FileName:  fileName,
		ExpiresAt: req.ExpiresAt,
	}, nil
}

// sanitizeFileName removes unsafe characters from a filename.
func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return replacer.Replace(name)
}

// OpenVPN configuration template
const ovpnTemplate = `# GateKey OpenVPN Configuration
# Generated: {{ .ExpiresAt }}
# Gateway: {{ .GatewayName }}
# User: {{ .UserEmail }}
# Crypto Profile: {{ .Crypto.CryptoProfile }}
#
# This configuration expires at {{ .ExpiresAt }}
# After expiration, you must generate a new configuration.

client
dev tun
proto {{ .Protocol }}
remote {{ .GatewayHostname }} {{ .GatewayPort }}
resolv-retry infinite
nobind
persist-key
persist-tun

# Security settings ({{ .Crypto.CryptoProfile }})
remote-cert-tls server
auth-user-pass
cipher {{ .Crypto.Cipher }}
{{- if .Crypto.DataCiphers }}
data-ciphers {{ .Crypto.DataCiphers }}
{{- end }}
auth {{ .Crypto.Auth }}
tls-version-min {{ .Crypto.TLSVersionMin }}
{{- if .Crypto.TLSCipher }}
tls-cipher {{ .Crypto.TLSCipher }}
{{- end }}

# Connection settings
connect-retry 5 30
connect-timeout 30
server-poll-timeout 10

# Logging
verb 3
mute 10

{{- if .Options.keepalive }}
keepalive {{ .Options.keepalive }}
{{- else }}
keepalive 10 60
{{- end }}

{{- range .Routes }}
route {{ .Network }} {{ .Netmask }}
{{- end }}

{{- range .DNS }}
dhcp-option DNS {{ . }}
{{- end }}

{{- if .Options.compLzo }}
comp-lzo {{ .Options.compLzo }}
{{- end }}

{{- if .Options.mtu }}
tun-mtu {{ .Options.mtu }}
{{- end }}

# Embedded CA Certificate
<ca>
{{ .CACert -}}
</ca>

# Embedded Client Certificate
<cert>
{{ .ClientCert -}}
</cert>

# Embedded Client Private Key
<key>
{{ .ClientKey -}}
</key>

{{- if .TLSAuth }}

# TLS Authentication
key-direction {{ .TLSAuthDirection }}
<tls-auth>
{{ .TLSAuth -}}
</tls-auth>
{{- end }}

{{- if .AuthPassword }}

# Embedded Authentication Credentials
# Username: {{ .AuthUsername }}
# This token is unique to this config and can be revoked
<auth-user-pass>
{{ .AuthUsername }}
{{ .AuthPassword }}
</auth-user-pass>
{{- end }}
`

// GenerateTLSAuthKey generates a new TLS-Auth static key.
func GenerateTLSAuthKey() ([]byte, error) {
	// TLS-Auth key format:
	// -----BEGIN OpenVPN Static key V1-----
	// [16 lines of 32 hex chars each = 256 bytes]
	// -----END OpenVPN Static key V1-----

	key := make([]byte, 256)
	if _, err := cryptoRand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("#\n# 2048 bit OpenVPN static key\n#\n")
	buf.WriteString("-----BEGIN OpenVPN Static key V1-----\n")

	for i := 0; i < 256; i += 16 {
		buf.WriteString(fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x\n",
			key[i], key[i+1], key[i+2], key[i+3],
			key[i+4], key[i+5], key[i+6], key[i+7],
			key[i+8], key[i+9], key[i+10], key[i+11],
			key[i+12], key[i+13], key[i+14], key[i+15]))
	}

	buf.WriteString("-----END OpenVPN Static key V1-----\n")
	return buf.Bytes(), nil
}

// ServerConfig represents OpenVPN server configuration.
type ServerConfig struct {
	Port            int
	Protocol        string
	Device          string
	ServerNetwork   string
	ServerNetmask   string
	CACertPath      string
	ServerCertPath  string
	ServerKeyPath   string
	DHPath          string
	TLSAuthPath     string
	CRLPath         string
	StatusLog       string
	ClientConfigDir string
	ManagementAddr  string
	PushOptions     []string
	Scripts         ScriptPaths
}

// ScriptPaths contains paths to hook scripts.
type ScriptPaths struct {
	AuthUserPassVerify string
	TLSVerify          string
	ClientConnect      string
	ClientDisconnect   string
}

// GenerateServerConfig generates an OpenVPN server configuration.
func GenerateServerConfig(cfg ServerConfig) ([]byte, error) {
	tmpl, err := template.New("server").Parse(serverConfigTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

const serverConfigTemplate = `# GateKey OpenVPN Server Configuration
# Generated by GateKey

port {{ .Port }}
proto {{ .Protocol }}
dev {{ .Device }}

server {{ .ServerNetwork }} {{ .ServerNetmask }}

ca {{ .CACertPath }}
cert {{ .ServerCertPath }}
key {{ .ServerKeyPath }}
dh {{ .DHPath }}

{{- if .TLSAuthPath }}
tls-auth {{ .TLSAuthPath }} 0
{{- end }}

{{- if .CRLPath }}
crl-verify {{ .CRLPath }}
{{- end }}

# Security
cipher AES-256-GCM
auth SHA256
tls-version-min 1.2

# Connection
keepalive 10 60
persist-key
persist-tun

# Logging
status {{ .StatusLog }} 10
verb 3
mute 20

{{- if .ClientConfigDir }}
client-config-dir {{ .ClientConfigDir }}
{{- end }}

{{- if .ManagementAddr }}
management {{ .ManagementAddr }} 7505
{{- end }}

# Push options to clients
{{- range .PushOptions }}
push "{{ . }}"
{{- end }}

# Hook scripts
{{- if .Scripts.AuthUserPassVerify }}
auth-user-pass-verify {{ .Scripts.AuthUserPassVerify }} via-file
script-security 2
{{- end }}

{{- if .Scripts.TLSVerify }}
tls-verify {{ .Scripts.TLSVerify }}
{{- end }}

{{- if .Scripts.ClientConnect }}
client-connect {{ .Scripts.ClientConnect }}
{{- end }}

{{- if .Scripts.ClientDisconnect }}
client-disconnect {{ .Scripts.ClientDisconnect }}
{{- end }}

# Enable username/password in addition to certificates
verify-client-cert require

# Topology
topology subnet
`
