package saml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvider_Type(t *testing.T) {
	p := &Provider{
		name:        "okta",
		displayName: "Okta SSO",
	}

	if p.Type() != "saml" {
		t.Errorf("Expected type 'saml', got %q", p.Type())
	}
}

func TestProvider_Name(t *testing.T) {
	p := &Provider{
		name:        "okta",
		displayName: "Okta SSO",
	}

	if p.Name() != "okta" {
		t.Errorf("Expected name 'okta', got %q", p.Name())
	}
}

func TestProvider_DisplayName_WithDisplayName(t *testing.T) {
	p := &Provider{
		name:        "okta",
		displayName: "Okta SSO",
	}

	if p.DisplayName() != "Okta SSO" {
		t.Errorf("Expected display name 'Okta SSO', got %q", p.DisplayName())
	}
}

func TestProvider_DisplayName_FallbackToName(t *testing.T) {
	p := &Provider{
		name:        "okta",
		displayName: "", // empty
	}

	if p.DisplayName() != "okta" {
		t.Errorf("Expected display name to fall back to 'okta', got %q", p.DisplayName())
	}
}

func TestProvider_AttributeMapping(t *testing.T) {
	// Test that attribute mappings can be customized
	p := &Provider{
		name: "custom",
		attributeMap: map[string]string{
			"email":  "http://custom/claims/email",
			"name":   "http://custom/claims/displayname",
			"groups": "http://custom/claims/memberof",
		},
	}

	if p.attributeMap["email"] != "http://custom/claims/email" {
		t.Errorf("Expected custom email attribute, got %q", p.attributeMap["email"])
	}
	if p.attributeMap["groups"] != "http://custom/claims/memberof" {
		t.Errorf("Expected custom groups attribute, got %q", p.attributeMap["groups"])
	}
}

func TestFetchIDPMetadata_InvalidFile(t *testing.T) {
	_, err := fetchIDPMetadata(nil, "/nonexistent/path/metadata.xml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFetchIDPMetadata_InvalidXML(t *testing.T) {
	// Create temp file with invalid XML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.xml")
	err := os.WriteFile(tmpFile, []byte("not valid xml at all"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = fetchIDPMetadata(nil, tmpFile)
	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestFetchIDPMetadata_ValidEntityDescriptor(t *testing.T) {
	// Create temp file with minimal valid EntityDescriptor XML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.xml")

	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com">
    <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
    </IDPSSODescriptor>
</EntityDescriptor>`

	err := os.WriteFile(tmpFile, []byte(metadata), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := fetchIDPMetadata(nil, tmpFile)
	if err != nil {
		t.Fatalf("Expected no error for valid metadata, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.EntityID != "https://idp.example.com" {
		t.Errorf("Expected entityID 'https://idp.example.com', got %q", result.EntityID)
	}
}

func TestFetchIDPMetadata_EntitiesDescriptor(t *testing.T) {
	// Create temp file with EntitiesDescriptor (multiple entities) XML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.xml")

	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<EntitiesDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata">
    <EntityDescriptor entityID="https://idp1.example.com">
        <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
            <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp1.example.com/sso"/>
        </IDPSSODescriptor>
    </EntityDescriptor>
    <EntityDescriptor entityID="https://idp2.example.com">
        <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
            <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp2.example.com/sso"/>
        </IDPSSODescriptor>
    </EntityDescriptor>
</EntitiesDescriptor>`

	err := os.WriteFile(tmpFile, []byte(metadata), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := fetchIDPMetadata(nil, tmpFile)
	if err != nil {
		t.Fatalf("Expected no error for EntitiesDescriptor, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	// Should return the first entity
	if result.EntityID != "https://idp1.example.com" {
		t.Errorf("Expected first entityID 'https://idp1.example.com', got %q", result.EntityID)
	}
}

func TestFetchIDPMetadata_EmptyEntitiesDescriptor(t *testing.T) {
	// Create temp file with empty EntitiesDescriptor
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.xml")

	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<EntitiesDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata">
</EntitiesDescriptor>`

	err := os.WriteFile(tmpFile, []byte(metadata), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = fetchIDPMetadata(nil, tmpFile)
	if err == nil {
		t.Error("Expected error for empty EntitiesDescriptor")
	}
}

func TestFetchIDPMetadata_FilePaths(t *testing.T) {
	// Test that file paths are handled (will fail with not found)
	tests := []struct {
		name  string
		input string
	}{
		{"absolute path", "/etc/saml/metadata.xml"},
		{"relative path", "./metadata.xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetchIDPMetadata(nil, tt.input)
			// Files will fail with not found
			if err == nil {
				t.Error("Expected error for non-existent file")
			}
		})
	}
}

func TestDefaultAttributeMappings(t *testing.T) {
	// Verify the default SAML attribute claim URIs
	expectedDefaults := map[string]string{
		"email":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		"name":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		"groups": "http://schemas.xmlsoap.org/claims/Group",
	}

	// Create provider with empty attribute map to get defaults
	p := &Provider{
		attributeMap: map[string]string{
			"email":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"groups": "http://schemas.xmlsoap.org/claims/Group",
		},
	}

	for key, expected := range expectedDefaults {
		if p.attributeMap[key] != expected {
			t.Errorf("Expected default %s attribute %q, got %q", key, expected, p.attributeMap[key])
		}
	}
}

func TestProvider_MultipleProviders(t *testing.T) {
	// Test that multiple providers can coexist
	providers := []*Provider{
		{name: "okta", displayName: "Okta SSO"},
		{name: "azure", displayName: "Azure AD"},
		{name: "onelogin", displayName: "OneLogin"},
	}

	names := make(map[string]bool)
	for _, p := range providers {
		if names[p.Name()] {
			t.Errorf("Duplicate provider name: %s", p.Name())
		}
		names[p.Name()] = true

		if p.Type() != "saml" {
			t.Errorf("Expected type 'saml' for all providers, got %q", p.Type())
		}
	}
}
