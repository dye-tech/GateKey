// Package oidc implements OIDC authentication for GateKey.
package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gatekey-project/gatekey/internal/auth"
	"github.com/gatekey-project/gatekey/internal/config"
	"golang.org/x/oauth2"
)

// Provider implements OIDC authentication.
type Provider struct {
	name        string
	displayName string
	config      config.OIDCProvider
	oauth2Cfg   *oauth2.Config
	verifier    *oidc.IDTokenVerifier
	provider    *oidc.Provider
	claimMap    map[string]string
}

// NewProvider creates a new OIDC provider.
func NewProvider(ctx context.Context, cfg config.OIDCProvider) (*Provider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	// Default claim mappings
	claimMap := map[string]string{
		"email":  "email",
		"name":   "name",
		"groups": "groups",
	}
	// Override with configured mappings
	for k, v := range cfg.Claims {
		claimMap[k] = v
	}

	return &Provider{
		name:        cfg.Name,
		displayName: cfg.DisplayName,
		config:      cfg,
		oauth2Cfg:   oauth2Cfg,
		verifier:    verifier,
		provider:    provider,
		claimMap:    claimMap,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return p.name
}

// Type returns the provider type.
func (p *Provider) Type() string {
	return "oidc"
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
	return p.oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// HandleCallback processes the authentication callback and returns user info.
func (p *Provider) HandleCallback(ctx context.Context, r *http.Request) (*auth.UserInfo, error) {
	// Get authorization code from callback
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("no authorization code in callback")
	}

	// Exchange code for tokens
	token, err := p.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Verify ID token
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Map claims to user info
	userInfo := &auth.UserInfo{
		ExternalID: idToken.Subject,
		Provider:   fmt.Sprintf("oidc:%s", p.name),
		Attributes: claims,
	}

	// Extract email
	if emailClaim, ok := claims[p.claimMap["email"]]; ok {
		if email, ok := emailClaim.(string); ok {
			userInfo.Email = email
		}
	}

	// Extract name
	if nameClaim, ok := claims[p.claimMap["name"]]; ok {
		if name, ok := nameClaim.(string); ok {
			userInfo.Name = name
		}
	}

	// Extract groups
	if groupsClaim, ok := claims[p.claimMap["groups"]]; ok {
		userInfo.Groups = extractGroups(groupsClaim)
	}

	return userInfo, nil
}

// extractGroups extracts groups from various claim formats.
func extractGroups(claim interface{}) []string {
	switch v := claim.(type) {
	case []interface{}:
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case []string:
		return v
	case string:
		// Try JSON array
		var groups []string
		if err := json.Unmarshal([]byte(v), &groups); err == nil {
			return groups
		}
		return []string{v}
	default:
		return nil
	}
}
