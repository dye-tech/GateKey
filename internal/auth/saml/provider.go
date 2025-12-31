// Package saml implements SAML authentication for GateKey.
package saml

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	"github.com/gatekey-project/gatekey/internal/auth"
	"github.com/gatekey-project/gatekey/internal/config"
)

// Provider implements SAML authentication.
type Provider struct {
	name         string
	displayName  string
	config       config.SAMLProvider
	sp           *saml.ServiceProvider
	attributeMap map[string]string
}

// NewProvider creates a new SAML provider.
func NewProvider(ctx context.Context, cfg config.SAMLProvider) (*Provider, error) {
	// Load SP certificate and key
	keyPair, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load SP certificate: %w", err)
	}

	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP certificate: %w", err)
	}

	// Fetch IdP metadata
	idpMetadata, err := fetchIDPMetadata(ctx, cfg.IDPMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IdP metadata: %w", err)
	}

	acsURL, err := url.Parse(cfg.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ACS URL: %w", err)
	}

	entityID, err := url.Parse(cfg.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entity ID: %w", err)
	}

	sp := &saml.ServiceProvider{
		Key:         keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate: keyPair.Leaf,
		MetadataURL: *entityID,
		AcsURL:      *acsURL,
		IDPMetadata: idpMetadata,
	}

	// Default attribute mappings
	attributeMap := map[string]string{
		"email":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		"name":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		"groups": "http://schemas.xmlsoap.org/claims/Group",
	}
	// Override with configured mappings
	for k, v := range cfg.Attributes {
		attributeMap[k] = v
	}

	return &Provider{
		name:         cfg.Name,
		displayName:  cfg.DisplayName,
		config:       cfg,
		sp:           sp,
		attributeMap: attributeMap,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return p.name
}

// Type returns the provider type.
func (p *Provider) Type() string {
	return "saml"
}

// DisplayName returns the display name for UI.
func (p *Provider) DisplayName() string {
	if p.displayName != "" {
		return p.displayName
	}
	return p.name
}

// LoginURL returns the URL to redirect users for login.
func (p *Provider) LoginURL(state string) (string, error) {
	authnRequest, err := p.sp.MakeAuthenticationRequest(
		p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create SAML authn request: %w", err)
	}

	// Store state in relay state
	redirectURL, err := authnRequest.Redirect(state, p.sp)
	if err != nil {
		return "", fmt.Errorf("failed to create redirect URL: %w", err)
	}

	return redirectURL.String(), nil
}

// HandleCallback processes the SAML assertion and returns user info.
func (p *Provider) HandleCallback(ctx context.Context, r *http.Request) (*auth.UserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return nil, fmt.Errorf("no SAMLResponse in request")
	}

	// Decode and parse the SAML response
	rawResponse, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAMLResponse: %w", err)
	}

	assertion, err := p.sp.ParseResponse(r, []string{""})
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAML response: %w", err)
	}

	_ = rawResponse // Used for debugging if needed

	// Extract user info from assertion
	userInfo := &auth.UserInfo{
		Provider:   fmt.Sprintf("saml:%s", p.name),
		Attributes: make(map[string]interface{}),
	}

	// Get NameID
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		userInfo.ExternalID = assertion.Subject.NameID.Value
	}

	// Extract attributes from assertion
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) > 0 {
				if len(attr.Values) == 1 {
					userInfo.Attributes[attr.Name] = attr.Values[0].Value
				} else {
					values := make([]string, len(attr.Values))
					for i, v := range attr.Values {
						values[i] = v.Value
					}
					userInfo.Attributes[attr.Name] = values
				}
			}
		}
	}

	// Extract email
	if emailAttr := p.attributeMap["email"]; emailAttr != "" {
		if email, ok := userInfo.Attributes[emailAttr].(string); ok {
			userInfo.Email = email
		}
	}

	// Extract name
	if nameAttr := p.attributeMap["name"]; nameAttr != "" {
		if name, ok := userInfo.Attributes[nameAttr].(string); ok {
			userInfo.Name = name
		}
	}

	// Extract groups
	if groupsAttr := p.attributeMap["groups"]; groupsAttr != "" {
		switch v := userInfo.Attributes[groupsAttr].(type) {
		case []string:
			userInfo.Groups = v
		case string:
			userInfo.Groups = []string{v}
		}
	}

	return userInfo, nil
}

// Metadata returns the SP metadata XML.
func (p *Provider) Metadata() ([]byte, error) {
	metadata := p.sp.Metadata()
	return xml.MarshalIndent(metadata, "", "  ")
}

// fetchIDPMetadata fetches IdP metadata from a URL or file.
func fetchIDPMetadata(ctx context.Context, urlOrPath string) (*saml.EntityDescriptor, error) {
	if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
		// Fetch from URL
		idpMetadataURL, err := url.Parse(urlOrPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IdP metadata URL: %w", err)
		}
		return samlsp.FetchMetadata(ctx, http.DefaultClient, *idpMetadataURL)
	}

	// Read from file
	data, err := os.ReadFile(urlOrPath)
	if err != nil {
		return nil, err
	}

	metadata := &saml.EntityDescriptor{}
	if err := xml.Unmarshal(data, metadata); err != nil {
		// Try parsing as EntitiesDescriptor (multiple entities)
		entities := &saml.EntitiesDescriptor{}
		if err := xml.Unmarshal(data, entities); err != nil {
			return nil, fmt.Errorf("failed to parse IdP metadata: %w", err)
		}
		if len(entities.EntityDescriptors) == 0 {
			return nil, fmt.Errorf("no entity descriptors in metadata")
		}
		metadata = &entities.EntityDescriptors[0]
	}

	return metadata, nil
}
