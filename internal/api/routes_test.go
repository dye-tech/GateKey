package api

import (
	"testing"
)

func TestHasScope_SessionAuth(t *testing.T) {
	// Session-based auth (no API key) has full access
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: nil, // nil means session auth
	}

	if !user.HasScope(ScopeGatewaysRead) {
		t.Error("Session auth should have access to gateways:read")
	}
	if !user.HasScope(ScopeAdmin) {
		t.Error("Session auth should have access to admin scope")
	}
}

func TestHasScope_Wildcard(t *testing.T) {
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeAll},
	}

	if !user.HasScope(ScopeGatewaysRead) {
		t.Error("Wildcard scope should grant gateways:read")
	}
	if !user.HasScope(ScopeAdmin) {
		t.Error("Wildcard scope should grant admin")
	}
	if !user.HasScope(ScopeMeshWrite) {
		t.Error("Wildcard scope should grant mesh:write")
	}
}

func TestHasScope_ExactMatch(t *testing.T) {
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeGatewaysRead, ScopeUsersRead},
	}

	if !user.HasScope(ScopeGatewaysRead) {
		t.Error("Should have gateways:read scope")
	}
	if !user.HasScope(ScopeUsersRead) {
		t.Error("Should have users:read scope")
	}
	if user.HasScope(ScopeGatewaysWrite) {
		t.Error("Should not have gateways:write scope")
	}
	if user.HasScope(ScopeAdmin) {
		t.Error("Should not have admin scope")
	}
}

func TestHasScope_HierarchicalMatch(t *testing.T) {
	// Parent scope "gateways" should grant "gateways:read" and "gateways:write"
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{"gateways"},
	}

	if !user.HasScope(ScopeGatewaysRead) {
		t.Error("Parent scope 'gateways' should grant gateways:read")
	}
	if !user.HasScope(ScopeGatewaysWrite) {
		t.Error("Parent scope 'gateways' should grant gateways:write")
	}
	if user.HasScope(ScopeUsersRead) {
		t.Error("Parent scope 'gateways' should not grant users:read")
	}
}

func TestHasScope_ReadWriteHierarchy(t *testing.T) {
	// Write scope should grant read access
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeGatewaysWrite},
	}

	if !user.HasScope(ScopeGatewaysWrite) {
		t.Error("Should have gateways:write scope")
	}
	if !user.HasScope(ScopeGatewaysRead) {
		t.Error("gateways:write should also grant gateways:read")
	}
}

func TestHasAnyScope_WithMatch(t *testing.T) {
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeGatewaysRead},
	}

	if !user.HasAnyScope(ScopeGatewaysRead, ScopeUsersRead) {
		t.Error("Should match when one of the scopes is present")
	}
}

func TestHasAnyScope_NoMatch(t *testing.T) {
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeMeshRead},
	}

	if user.HasAnyScope(ScopeGatewaysRead, ScopeUsersRead) {
		t.Error("Should not match when none of the scopes are present")
	}
}

func TestHasAnyScope_AdminAccess(t *testing.T) {
	user := &authenticatedUser{
		UserID:       "user1",
		APIKeyScopes: []string{ScopeAdmin},
	}

	// Admin scope should grant access when admin is in the required list
	if !user.HasAnyScope(ScopeGatewaysRead, ScopeGatewaysWrite, ScopeAdmin) {
		t.Error("Admin scope should be recognized in HasAnyScope")
	}
}

func TestScopeConstants(t *testing.T) {
	// Verify scope constants are defined correctly
	tests := []struct {
		scope    string
		expected string
	}{
		{ScopeAll, "*"},
		{ScopeRead, "read"},
		{ScopeWrite, "write"},
		{ScopeAdmin, "admin"},
		{ScopeGatewaysRead, "gateways:read"},
		{ScopeGatewaysWrite, "gateways:write"},
		{ScopeUsersRead, "users:read"},
		{ScopeUsersWrite, "users:write"},
		{ScopeMeshRead, "mesh:read"},
		{ScopeMeshWrite, "mesh:write"},
		{ScopeConfigRead, "config:read"},
		{ScopeConfigWrite, "config:write"},
	}

	for _, tt := range tests {
		if tt.scope != tt.expected {
			t.Errorf("Scope constant mismatch: got %q, expected %q", tt.scope, tt.expected)
		}
	}
}
