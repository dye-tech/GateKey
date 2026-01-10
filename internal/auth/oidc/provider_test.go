package oidc

import (
	"encoding/json"
	"testing"
)

func TestExtractGroups_StringSlice(t *testing.T) {
	input := []string{"admin", "users", "developers"}
	groups := extractGroups(input)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groups))
	}
	if groups[0] != "admin" {
		t.Errorf("Expected first group 'admin', got %q", groups[0])
	}
	if groups[1] != "users" {
		t.Errorf("Expected second group 'users', got %q", groups[1])
	}
	if groups[2] != "developers" {
		t.Errorf("Expected third group 'developers', got %q", groups[2])
	}
}

func TestExtractGroups_InterfaceSlice(t *testing.T) {
	input := []interface{}{"admin", "users", "developers"}
	groups := extractGroups(input)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groups))
	}
	if groups[0] != "admin" {
		t.Errorf("Expected first group 'admin', got %q", groups[0])
	}
}

func TestExtractGroups_InterfaceSlice_WithNonStrings(t *testing.T) {
	// Some items are not strings - they should be skipped
	input := []interface{}{"admin", 123, "users", nil, "developers"}
	groups := extractGroups(input)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups (non-strings filtered), got %d", len(groups))
	}
	if groups[0] != "admin" {
		t.Errorf("Expected first group 'admin', got %q", groups[0])
	}
	if groups[1] != "users" {
		t.Errorf("Expected second group 'users', got %q", groups[1])
	}
	if groups[2] != "developers" {
		t.Errorf("Expected third group 'developers', got %q", groups[2])
	}
}

func TestExtractGroups_JSONString(t *testing.T) {
	// Groups encoded as JSON array in a string
	input := `["admin","users","developers"]`
	groups := extractGroups(input)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups from JSON string, got %d", len(groups))
	}
	if groups[0] != "admin" {
		t.Errorf("Expected first group 'admin', got %q", groups[0])
	}
}

func TestExtractGroups_PlainString(t *testing.T) {
	// Single group as plain string
	input := "admin"
	groups := extractGroups(input)

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group from plain string, got %d", len(groups))
	}
	if groups[0] != "admin" {
		t.Errorf("Expected group 'admin', got %q", groups[0])
	}
}

func TestExtractGroups_InvalidJSONString(t *testing.T) {
	// Invalid JSON should fall back to treating as single group
	input := "not-json-array"
	groups := extractGroups(input)

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group for non-JSON string, got %d", len(groups))
	}
	if groups[0] != "not-json-array" {
		t.Errorf("Expected group 'not-json-array', got %q", groups[0])
	}
}

func TestExtractGroups_Nil(t *testing.T) {
	groups := extractGroups(nil)
	if groups != nil {
		t.Errorf("Expected nil for nil input, got %v", groups)
	}
}

func TestExtractGroups_UnsupportedType(t *testing.T) {
	// Integer input should return nil
	groups := extractGroups(12345)
	if groups != nil {
		t.Errorf("Expected nil for unsupported type int, got %v", groups)
	}

	// Map input should return nil
	groups = extractGroups(map[string]string{"key": "value"})
	if groups != nil {
		t.Errorf("Expected nil for unsupported type map, got %v", groups)
	}
}

func TestExtractGroups_EmptySlice(t *testing.T) {
	input := []interface{}{}
	groups := extractGroups(input)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty slice, got %d", len(groups))
	}
}

func TestExtractGroups_EmptyStringSlice(t *testing.T) {
	input := []string{}
	groups := extractGroups(input)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty string slice, got %d", len(groups))
	}
}

func TestExtractGroups_EmptyJSONArray(t *testing.T) {
	input := "[]"
	groups := extractGroups(input)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty JSON array, got %d", len(groups))
	}
}

func TestExtractGroups_FromRealOIDCClaims(t *testing.T) {
	// Simulate how groups might appear in real OIDC claims
	claims := map[string]interface{}{
		"sub":    "user-123",
		"email":  "user@example.com",
		"groups": []interface{}{"Engineering", "VPN-Users", "Admin"},
	}

	groups := extractGroups(claims["groups"])
	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups from OIDC claims, got %d", len(groups))
	}
}

func TestExtractGroups_AzureADFormat(t *testing.T) {
	// Azure AD sometimes returns groups as array of strings in JSON
	jsonClaims := `{"groups": ["group-uuid-1", "group-uuid-2"]}`
	var claims map[string]interface{}
	if err := json.Unmarshal([]byte(jsonClaims), &claims); err != nil {
		t.Fatalf("Failed to unmarshal test claims: %v", err)
	}

	groups := extractGroups(claims["groups"])
	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups from Azure AD format, got %d", len(groups))
	}
}

func TestExtractGroups_OktaFormat(t *testing.T) {
	// Okta typically returns groups as array
	jsonClaims := `{"groups": ["Everyone", "Developers", "VPN-Access"]}`
	var claims map[string]interface{}
	if err := json.Unmarshal([]byte(jsonClaims), &claims); err != nil {
		t.Fatalf("Failed to unmarshal test claims: %v", err)
	}

	groups := extractGroups(claims["groups"])
	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups from Okta format, got %d", len(groups))
	}
	if groups[0] != "Everyone" {
		t.Errorf("Expected first group 'Everyone', got %q", groups[0])
	}
}

func TestProvider_Type(t *testing.T) {
	p := &Provider{
		name:        "google",
		displayName: "Google Workspace",
	}

	if p.Type() != "oidc" {
		t.Errorf("Expected type 'oidc', got %q", p.Type())
	}
}

func TestProvider_Name(t *testing.T) {
	p := &Provider{
		name:        "google",
		displayName: "Google Workspace",
	}

	if p.Name() != "google" {
		t.Errorf("Expected name 'google', got %q", p.Name())
	}
}

func TestProvider_DisplayName_WithDisplayName(t *testing.T) {
	p := &Provider{
		name:        "google",
		displayName: "Google Workspace",
	}

	if p.DisplayName() != "Google Workspace" {
		t.Errorf("Expected display name 'Google Workspace', got %q", p.DisplayName())
	}
}

func TestProvider_DisplayName_FallbackToName(t *testing.T) {
	p := &Provider{
		name:        "google",
		displayName: "", // empty
	}

	if p.DisplayName() != "google" {
		t.Errorf("Expected display name to fall back to 'google', got %q", p.DisplayName())
	}
}

func TestProvider_ClaimMapping(t *testing.T) {
	// Test that claim mappings can be customized
	p := &Provider{
		name: "custom",
		claimMap: map[string]string{
			"email":  "preferred_email",
			"name":   "full_name",
			"groups": "roles",
		},
	}

	if p.claimMap["email"] != "preferred_email" {
		t.Errorf("Expected email claim mapping 'preferred_email', got %q", p.claimMap["email"])
	}
	if p.claimMap["groups"] != "roles" {
		t.Errorf("Expected groups claim mapping 'roles', got %q", p.claimMap["groups"])
	}
}
