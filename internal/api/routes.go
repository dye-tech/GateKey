// Package api contains route handler implementations.
package api

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/models"
	"github.com/gatekey-project/gatekey/internal/openvpn"
	"github.com/gatekey-project/gatekey/internal/pki"
)

// cliCallbackStore stores CLI callback URLs by state
var cliCallbackStore sync.Map

// cidrToRoute converts a CIDR notation (e.g., "192.168.50.0/23") to OpenVPN route format (e.g., "route 192.168.50.0 255.255.254.0")
func cidrToRoute(cidr string) string {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ""
	}
	// Convert net.IPMask to dotted decimal format
	// Use "push" directive for client-connect scripts
	mask := ipNet.Mask
	if len(mask) == 4 {
		return fmt.Sprintf("push \"route %s %d.%d.%d.%d\"", ipNet.IP.String(), mask[0], mask[1], mask[2], mask[3])
	}
	return ""
}

// Authentication handlers

func (s *Server) handleOIDCLogin(c *gin.Context) {
	providerName := c.Query("provider")
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider parameter required"})
		return
	}

	// Get provider config from database
	providerConfig, err := s.providerStore.GetOIDCProvider(c.Request.Context(), providerName)
	if err != nil {
		s.logger.Error("Failed to get OIDC provider", zap.String("provider", providerName), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	if !providerConfig.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is disabled"})
		return
	}

	// Create OIDC provider
	ctx := context.Background()
	issuerURL := strings.TrimSpace(providerConfig.Issuer)
	s.logger.Info("Connecting to OIDC provider", zap.String("provider", providerName), zap.String("issuer", issuerURL))

	oidcProvider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		s.logger.Error("Failed to create OIDC provider",
			zap.String("provider", providerName),
			zap.String("issuer", issuerURL),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to identity provider: " + err.Error()})
		return
	}

	// Determine scopes
	scopes := providerConfig.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     strings.TrimSpace(providerConfig.ClientID),
		ClientSecret: strings.TrimSpace(providerConfig.ClientSecret),
		RedirectURL:  strings.TrimSpace(providerConfig.RedirectURL),
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       scopes,
	}

	// Generate state and nonce
	state, err := generateState()
	if err != nil {
		s.logger.Error("Failed to generate state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	nonce, err := generateState()
	if err != nil {
		s.logger.Error("Failed to generate nonce", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	// Check for CLI state (to redirect back to CLI after auth)
	cliState := c.Query("cli_state")
	var cliCallbackURL string
	if cliState != "" {
		s.logger.Info("OIDC login with CLI state", zap.String("cli_state", cliState))
		// Look up the CLI callback URL from database
		callbackURL, err := s.stateStore.GetCLICallback(c.Request.Context(), cliState)
		if err != nil {
			s.logger.Warn("CLI callback URL not found", zap.String("cli_state", cliState), zap.Error(err))
		} else {
			cliCallbackURL = callbackURL
			s.logger.Info("Found CLI callback URL", zap.String("callback_url", cliCallbackURL))
		}
	}

	// Store state data in database for validation (expires in 10 minutes)
	oauthState := &db.OAuthState{
		State:          state,
		Provider:       providerName,
		ProviderType:   "oidc",
		Nonce:          nonce,
		RelayState:     cliState,
		CLICallbackURL: cliCallbackURL, // Store CLI callback URL for redirect after auth
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	if err := s.stateStore.SaveState(c.Request.Context(), oauthState); err != nil {
		s.logger.Error("Failed to save state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save state"})
		return
	}

	// Redirect to authorization URL
	authURL := oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce))
	c.Redirect(http.StatusFound, authURL)
}

func (s *Server) handleOIDCCallback(c *gin.Context) {
	// Get state and code from query params
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		// Check for error response from IdP
		if errMsg := c.Query("error"); errMsg != "" {
			errDesc := c.Query("error_description")
			s.logger.Error("OIDC error from IdP", zap.String("error", errMsg), zap.String("description", errDesc))
			c.Redirect(http.StatusFound, "/login?error="+errMsg)
			return
		}
		c.Redirect(http.StatusFound, "/login?error=invalid_callback")
		return
	}

	// Validate and retrieve state data from database
	stateData, err := s.stateStore.GetState(c.Request.Context(), state)
	if err != nil {
		s.logger.Error("Invalid or expired state", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=invalid_state")
		return
	}

	// Get provider config from database
	providerConfig, err := s.providerStore.GetOIDCProvider(c.Request.Context(), stateData.Provider)
	if err != nil {
		s.logger.Error("Failed to get OIDC provider", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=provider_not_found")
		return
	}

	// Create OIDC provider
	ctx := context.Background()
	issuerURL := strings.TrimSpace(providerConfig.Issuer)
	oidcProvider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		s.logger.Error("Failed to create OIDC provider", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=provider_error")
		return
	}

	// Create OAuth2 config
	scopes := providerConfig.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     strings.TrimSpace(providerConfig.ClientID),
		ClientSecret: strings.TrimSpace(providerConfig.ClientSecret),
		RedirectURL:  strings.TrimSpace(providerConfig.RedirectURL),
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       scopes,
	}

	// Exchange code for token
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		s.logger.Error("Failed to exchange code for token", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=token_exchange_failed")
		return
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		s.logger.Error("No id_token in token response")
		c.Redirect(http.StatusFound, "/login?error=no_id_token")
		return
	}

	// Verify ID token
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: providerConfig.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.logger.Error("Failed to verify ID token", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=token_verification_failed")
		return
	}

	// Verify nonce
	if idToken.Nonce != stateData.Nonce {
		s.logger.Error("Nonce mismatch")
		c.Redirect(http.StatusFound, "/login?error=nonce_mismatch")
		return
	}

	// Extract claims
	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		PreferredUser string   `json:"preferred_username"`
		Groups        []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		s.logger.Error("Failed to parse claims", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=claims_error")
		return
	}

	// Use preferred_username or email as identifier
	username := claims.PreferredUser
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = idToken.Subject
	}

	email := claims.Email
	if email == "" {
		email = username + "@" + stateData.Provider
	}

	name := claims.Name
	if name == "" {
		name = username
	}

	// Generate a session token
	tokenBytes := make([]byte, 32)
	if _, err := cryptoRand.Read(tokenBytes); err != nil {
		s.logger.Error("Failed to generate session token", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=session_error")
		return
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create or update user in database and create session
	// For SSO users, we'll create a session directly using a special user ID
	// In a full implementation, you'd sync users to the database
	userID := "oidc:" + stateData.Provider + ":" + idToken.Subject

	expiresAt := time.Now().Add(s.config.Auth.Session.Validity)
	ipAddress := getRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	// Store SSO session (using a modified approach since SSO users aren't in local_users)
	// For now, we'll store it in admin_sessions with a synthetic user_id
	// A better approach would be to have a separate sso_sessions table
	if err := s.createSSOSession(c.Request.Context(), userID, token, expiresAt, ipAddress, userAgent, username, email, name, claims.Groups); err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=session_error")
		return
	}

	// Set session cookie
	c.SetCookie(
		s.config.Auth.Session.CookieName,
		token,
		int(s.config.Auth.Session.Validity.Seconds()),
		"/",
		"",
		s.config.Auth.Session.Secure,
		true, // httpOnly
	)

	s.logger.Info("OIDC login successful",
		zap.String("provider", stateData.Provider),
		zap.String("user", username),
		zap.String("email", email),
	)

	// Log the successful login
	s.logUserLogin(c.Request.Context(), userID, email, name, "oidc", stateData.Provider, ipAddress, userAgent, token, true, "")

	// Check if this is a CLI login flow
	if stateData.CLICallbackURL != "" {
		s.logger.Info("OIDC callback with CLI callback URL", zap.String("callback_url", stateData.CLICallbackURL))
		// Redirect to CLI callback with token
		redirectURL := stateData.CLICallbackURL + "?token=" + token + "&email=" + url.QueryEscape(email) + "&name=" + url.QueryEscape(name) + "&expires_in=86400"
		s.logger.Info("Redirecting to CLI", zap.String("redirect_url", redirectURL))
		c.Redirect(http.StatusFound, redirectURL)
		return
	} else {
		s.logger.Info("OIDC callback without CLI callback URL (normal web login)")
	}

	// Redirect to dashboard for normal web login
	c.Redirect(http.StatusFound, "/")
}

func (s *Server) handleSAMLLogin(c *gin.Context) {
	providerName := c.Query("provider")
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider parameter required"})
		return
	}

	// Get provider config from database
	providerConfig, err := s.providerStore.GetSAMLProvider(c.Request.Context(), providerName)
	if err != nil {
		s.logger.Error("Failed to get SAML provider", zap.String("provider", providerName), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	if !providerConfig.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is disabled"})
		return
	}

	// Fetch IdP metadata
	idpMetadataURL, err := url.Parse(providerConfig.IDPMetadataURL)
	if err != nil {
		s.logger.Error("Invalid IdP metadata URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid IdP metadata URL"})
		return
	}

	idpMetadata, err := samlsp.FetchMetadata(c.Request.Context(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		s.logger.Error("Failed to fetch IdP metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch IdP metadata"})
		return
	}

	// Parse ACS URL
	acsURL, err := url.Parse(providerConfig.ACSURL)
	if err != nil {
		s.logger.Error("Invalid ACS URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid ACS URL"})
		return
	}

	// Create a minimal SP (we don't need signing for the AuthnRequest in most cases)
	sp := &saml.ServiceProvider{
		EntityID:          providerConfig.EntityID,
		AcsURL:            *acsURL,
		IDPMetadata:       idpMetadata,
		AllowIDPInitiated: true,
	}

	// Generate a relay state for CSRF protection
	relayState, err := generateState()
	if err != nil {
		s.logger.Error("Failed to generate relay state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// Store state data in database for validation (expires in 10 minutes)
	oauthState := &db.OAuthState{
		State:        relayState,
		Provider:     providerName,
		ProviderType: "saml",
		RelayState:   relayState,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	if err := s.stateStore.SaveState(c.Request.Context(), oauthState); err != nil {
		s.logger.Error("Failed to save state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save state"})
		return
	}

	// Create AuthnRequest
	authnRequest, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		s.logger.Error("Failed to create SAML AuthnRequest", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create authentication request"})
		return
	}

	// Get redirect URL
	redirectURL, err := authnRequest.Redirect(relayState, sp)
	if err != nil {
		s.logger.Error("Failed to create redirect URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create redirect"})
		return
	}

	c.Redirect(http.StatusFound, redirectURL.String())
}

func (s *Server) handleSAMLACS(c *gin.Context) {
	// Get provider from relay state
	relayState := c.PostForm("RelayState")
	if relayState == "" {
		relayState = c.Query("RelayState")
	}

	// Validate relay state from database
	stateData, err := s.stateStore.GetState(c.Request.Context(), relayState)
	if err != nil {
		s.logger.Error("Invalid or expired relay state", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=invalid_state")
		return
	}

	// Get provider config from database
	providerConfig, err := s.providerStore.GetSAMLProvider(c.Request.Context(), stateData.Provider)
	if err != nil {
		s.logger.Error("Failed to get SAML provider", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=provider_not_found")
		return
	}

	// Fetch IdP metadata
	idpMetadataURL, err := url.Parse(providerConfig.IDPMetadataURL)
	if err != nil {
		s.logger.Error("Invalid IdP metadata URL", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=config_error")
		return
	}

	idpMetadata, err := samlsp.FetchMetadata(c.Request.Context(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		s.logger.Error("Failed to fetch IdP metadata", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=metadata_error")
		return
	}

	// Parse ACS URL
	acsURL, err := url.Parse(providerConfig.ACSURL)
	if err != nil {
		s.logger.Error("Invalid ACS URL", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=config_error")
		return
	}

	// Create SP
	sp := &saml.ServiceProvider{
		EntityID:          providerConfig.EntityID,
		AcsURL:            *acsURL,
		IDPMetadata:       idpMetadata,
		AllowIDPInitiated: true,
	}

	// Get SAML response
	samlResponse := c.PostForm("SAMLResponse")
	if samlResponse == "" {
		s.logger.Error("No SAMLResponse in request")
		c.Redirect(http.StatusFound, "/login?error=no_response")
		return
	}

	// Parse and validate the assertion
	assertion, err := sp.ParseResponse(c.Request, []string{})
	if err != nil {
		s.logger.Error("Failed to parse SAML response", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=invalid_response")
		return
	}

	// Extract user info from assertion
	username := ""
	email := ""
	name := ""
	nameID := ""
	var groups []string

	// Get NameID as primary identifier
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		nameID = assertion.Subject.NameID.Value
		username = nameID
	}

	// Extract attributes from all attribute statements
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			if len(attr.Values) == 0 {
				continue
			}
			// Check both Name and FriendlyName
			attrName := attr.Name
			friendlyName := attr.FriendlyName

			switch {
			case attrName == "email" || friendlyName == "email" ||
				attrName == "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
				email = attr.Values[0].Value
			case attrName == "name" || friendlyName == "name" || attrName == "displayName" ||
				attrName == "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name":
				name = attr.Values[0].Value
			case attrName == "username" || friendlyName == "username" ||
				attrName == "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn":
				if username == "" || username == nameID {
					username = attr.Values[0].Value
				}
			case attrName == "groups" || friendlyName == "groups" ||
				attrName == "http://schemas.xmlsoap.org/claims/Group" || attrName == "memberOf":
				for _, v := range attr.Values {
					groups = append(groups, v.Value)
				}
			}
		}
	}

	// Use email as username if no username found
	if username == "" && email != "" {
		username = email
	}
	if username == "" {
		username = "unknown"
	}
	if nameID == "" {
		nameID = username
	}

	if email == "" {
		email = username + "@" + stateData.Provider
	}

	if name == "" {
		name = username
	}

	// Generate a session token
	tokenBytes := make([]byte, 32)
	if _, err := cryptoRand.Read(tokenBytes); err != nil {
		s.logger.Error("Failed to generate session token", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=session_error")
		return
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create session
	userID := "saml:" + stateData.Provider + ":" + nameID
	expiresAt := time.Now().Add(s.config.Auth.Session.Validity)
	ipAddress := getRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	if err := s.createSSOSession(c.Request.Context(), userID, token, expiresAt, ipAddress, userAgent, username, email, name, groups); err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?error=session_error")
		return
	}

	// Set session cookie
	c.SetCookie(
		s.config.Auth.Session.CookieName,
		token,
		int(s.config.Auth.Session.Validity.Seconds()),
		"/",
		"",
		s.config.Auth.Session.Secure,
		true,
	)

	s.logger.Info("SAML login successful",
		zap.String("provider", stateData.Provider),
		zap.String("user", username),
		zap.String("email", email),
	)

	// Log the successful login
	s.logUserLogin(c.Request.Context(), userID, email, name, "saml", stateData.Provider, ipAddress, userAgent, token, true, "")

	// Redirect to dashboard
	c.Redirect(http.StatusFound, "/")
}

func (s *Server) handleSAMLMetadata(c *gin.Context) {
	providerName := c.Query("provider")
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider parameter required"})
		return
	}

	// Get provider config from database
	providerConfig, err := s.providerStore.GetSAMLProvider(c.Request.Context(), providerName)
	if err != nil {
		s.logger.Error("Failed to get SAML provider", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Parse ACS URL
	acsURL, err := url.Parse(providerConfig.ACSURL)
	if err != nil {
		s.logger.Error("Invalid ACS URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid ACS URL"})
		return
	}

	// Create SP metadata
	sp := &saml.ServiceProvider{
		EntityID: providerConfig.EntityID,
		AcsURL:   *acsURL,
	}

	metadata := sp.Metadata()

	// Return metadata as XML
	c.Header("Content-Type", "application/samlmetadata+xml")
	xmlData, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate metadata"})
		return
	}
	c.String(http.StatusOK, xml.Header+string(xmlData))
}

// Unused imports prevention
var (
	_ = rsa.PublicKey{}
	_ = x509.Certificate{}
)

func (s *Server) handleLogout(c *gin.Context) {
	// Get session token from cookie
	sessionCookie, err := c.Cookie(s.config.Auth.Session.CookieName)
	if err == nil && sessionCookie != "" {
		// Delete from SSO session database (best effort cleanup)
		_ = s.stateStore.DeleteSSOSession(c.Request.Context(), sessionCookie)
		// Delete from local session database (best effort cleanup)
		_ = s.userStore.DeleteSession(c.Request.Context(), sessionCookie)
	}

	// Clear session cookie
	c.SetCookie(
		s.config.Auth.Session.CookieName,
		"",
		-1,
		"/",
		"",
		s.config.Auth.Session.Secure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (s *Server) handleGetSession(c *gin.Context) {
	// Check for session cookie or Authorization header
	token := ""
	sessionCookie, err := c.Cookie(s.config.Auth.Session.CookieName)
	if err == nil && sessionCookie != "" {
		token = sessionCookie
	} else {
		// Also check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusOK, gin.H{"user": nil, "authenticated": false})
		return
	}

	// First, check SSO session in database
	if ssoSession, err := s.stateStore.GetSSOSession(c.Request.Context(), token); err == nil {
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":       ssoSession.UserID,
				"email":    ssoSession.Email,
				"name":     ssoSession.Name,
				"groups":   ssoSession.Groups,
				"isAdmin":  ssoSession.IsAdmin,
				"provider": ssoSession.Provider,
			},
			"authenticated": true,
		})
		return
	}

	// Fall back to local user session from database
	_, user, err := s.userStore.GetSession(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user": nil, "authenticated": false})
		return
	}

	// Return user info
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.Username,
			"email":   user.Email,
			"name":    user.Username,
			"groups":  []string{},
			"isAdmin": user.IsAdmin,
		},
		"authenticated": true,
	})
}

// createSSOSession stores an SSO session in the database
func (s *Server) createSSOSession(ctx context.Context, userID, token string, expiresAt time.Time, ipAddress, userAgent, username, email, name string, groups []string) error {
	// Determine provider and external ID from userID (format: "oidc:provider:subject" or "saml:provider:subject")
	providerType := ""
	providerName := ""
	externalID := ""
	parts := splitN(userID, ":", 3)
	if len(parts) >= 2 {
		providerType = parts[0]
		providerName = parts[1]
	}
	if len(parts) >= 3 {
		externalID = parts[2]
	}

	// Check if user should be admin based on provider's admin_group setting
	isAdmin := false
	s.logger.Info("Checking admin status for SSO user",
		zap.String("email", email),
		zap.String("providerType", providerType),
		zap.String("providerName", providerName),
		zap.Strings("groups", groups))

	if providerType == "oidc" && providerName != "" {
		oidcProvider, err := s.providerStore.GetOIDCProvider(ctx, providerName)
		if err != nil {
			s.logger.Warn("Failed to get OIDC provider for admin check",
				zap.String("provider", providerName),
				zap.Error(err))
		} else if oidcProvider.AdminGroup == "" {
			s.logger.Info("No admin group configured for OIDC provider",
				zap.String("provider", providerName))
		} else {
			s.logger.Info("Checking OIDC admin group membership",
				zap.String("email", email),
				zap.String("adminGroup", oidcProvider.AdminGroup),
				zap.Strings("userGroups", groups))
			// Check if user is in the admin group
			for _, group := range groups {
				if group == oidcProvider.AdminGroup {
					isAdmin = true
					s.logger.Info("User granted admin via OIDC group membership",
						zap.String("email", email),
						zap.String("group", oidcProvider.AdminGroup))
					break
				}
			}
			if !isAdmin {
				s.logger.Info("User NOT in OIDC admin group",
					zap.String("email", email),
					zap.String("adminGroup", oidcProvider.AdminGroup),
					zap.Strings("userGroups", groups))
			}
		}
	} else if providerType == "saml" && providerName != "" {
		samlProvider, err := s.providerStore.GetSAMLProvider(ctx, providerName)
		if err != nil {
			s.logger.Warn("Failed to get SAML provider for admin check",
				zap.String("provider", providerName),
				zap.Error(err))
		} else if samlProvider.AdminGroup == "" {
			s.logger.Info("No admin group configured for SAML provider",
				zap.String("provider", providerName))
		} else {
			s.logger.Info("Checking SAML admin group membership",
				zap.String("email", email),
				zap.String("adminGroup", samlProvider.AdminGroup),
				zap.Strings("userGroups", groups))
			// Check if user is in the admin group
			for _, group := range groups {
				if group == samlProvider.AdminGroup {
					isAdmin = true
					s.logger.Info("User granted admin via SAML group membership",
						zap.String("email", email),
						zap.String("group", samlProvider.AdminGroup))
					break
				}
			}
			if !isAdmin {
				s.logger.Info("User NOT in SAML admin group",
					zap.String("email", email),
					zap.String("adminGroup", samlProvider.AdminGroup),
					zap.Strings("userGroups", groups))
			}
		}
	}

	// Persist the user to the database (upsert on each login)
	// Use the actual database UUID as the session UserID for consistent access checks
	actualUserID := userID // fallback to compound ID
	if externalID != "" && providerName != "" {
		ssoUser, err := s.userStore.UpsertSSOUser(ctx, externalID, providerName, email, name, groups, isAdmin)
		if err != nil {
			s.logger.Warn("Failed to persist SSO user", zap.Error(err), zap.String("email", email))
			// Continue anyway - session can still be created
		} else if ssoUser != nil {
			actualUserID = ssoUser.ID // Use the database UUID
		}
	}

	session := &db.SSOSession{
		Token:     token,
		UserID:    actualUserID,
		Username:  username,
		Email:     email,
		Name:      name,
		Groups:    groups,
		Provider:  providerName,
		IsAdmin:   isAdmin,
		ExpiresAt: expiresAt,
	}

	return s.stateStore.SaveSSOSession(ctx, session)
}

// splitN splits a string by separator into at most n parts
func splitN(s, sep string, n int) []string {
	result := []string{}
	for i := 0; i < n-1; i++ {
		idx := -1
		for j := 0; j < len(s); j++ {
			if j+len(sep) <= len(s) && s[j:j+len(sep)] == sep {
				idx = j
				break
			}
		}
		if idx == -1 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func (s *Server) handleGetProviders(c *gin.Context) {
	// Return list of available auth providers
	providers := []gin.H{}

	// Only include dynamically configured OIDC providers from database
	oidcProviders, _ := s.providerStore.GetOIDCProviders(c.Request.Context())
	for _, p := range oidcProviders {
		if p.Enabled {
			providers = append(providers, gin.H{
				"type":         "oidc",
				"name":         p.Name,
				"display_name": p.DisplayName,
				"login_url":    "/api/v1/auth/oidc/login?provider=" + p.Name,
			})
		}
	}

	// Only include dynamically configured SAML providers from database
	samlProviders, _ := s.providerStore.GetSAMLProviders(c.Request.Context())
	for _, p := range samlProviders {
		if p.Enabled {
			providers = append(providers, gin.H{
				"type":         "saml",
				"name":         p.Name,
				"display_name": p.DisplayName,
				"login_url":    "/api/v1/auth/saml/login?provider=" + p.Name,
			})
		}
	}

	// Always include local auth for admin access
	providers = append(providers, gin.H{
		"type":         "local",
		"name":         "local",
		"display_name": "Local Admin",
		"login_url":    "/api/v1/auth/local/login",
	})

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// Local authentication handlers

func (s *Server) handleLocalLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}

	ipAddress := getRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	user, err := s.userStore.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		// Log failed login attempt
		s.logUserLogin(c.Request.Context(), "", req.Username, "", "local", "", ipAddress, userAgent, "", false, "invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate a session token
	tokenBytes := make([]byte, 32)
	if _, err := cryptoRand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate session"})
		return
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store session in database
	expiresAt := time.Now().Add(s.config.Auth.Session.Validity)
	if err := s.userStore.CreateSession(c.Request.Context(), user.ID, token, expiresAt, ipAddress, userAgent); err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set session cookie
	c.SetCookie(
		s.config.Auth.Session.CookieName,
		token,
		int(s.config.Auth.Session.Validity.Seconds()),
		"/",
		"",
		s.config.Auth.Session.Secure,
		true, // httpOnly
	)

	// Log successful login
	s.logUserLogin(c.Request.Context(), user.ID, user.Email, user.Username, "local", "", ipAddress, userAgent, token, true, "")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
		"token": token,
	})
}

func (s *Server) handleChangePassword(c *gin.Context) {
	var req struct {
		Username        string `json:"username" binding:"required"`
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Verify current password
	_, err := s.userStore.Authenticate(c.Request.Context(), req.Username, req.CurrentPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	// Update password
	if err := s.userStore.UpdatePassword(c.Request.Context(), req.Username, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

// CLI authentication handlers

func (s *Server) handleCLILogin(c *gin.Context) {
	callbackURL := c.Query("callback")
	if callbackURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "callback parameter required"})
		return
	}

	// Store the CLI callback URL in a session/state
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// Store callback URL in database (works across replicas)
	if err := s.stateStore.SaveCLICallback(c.Request.Context(), state, callbackURL); err != nil {
		s.logger.Error("Failed to save CLI callback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save state"})
		return
	}

	s.logger.Info("CLI login initiated", zap.String("state", state), zap.String("callback", callbackURL))

	// Redirect to the login page with the CLI flow indicator
	loginPageURL := "/login?cli=true&state=" + state
	c.Redirect(http.StatusFound, loginPageURL)
}

// handleCLIComplete completes CLI login for users who are already logged in
func (s *Server) handleCLIComplete(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state parameter required"})
		return
	}

	// Get session from cookie (use configured cookie name)
	token, err := c.Cookie(s.config.Auth.Session.CookieName)
	if err != nil || token == "" {
		s.logger.Warn("CLI complete: no session cookie", zap.String("cookie_name", s.config.Auth.Session.CookieName), zap.Error(err))
		c.Redirect(http.StatusFound, "/login?cli=true&state="+url.QueryEscape(state)+"&error=not_logged_in")
		return
	}

	// Validate session
	session, err := s.stateStore.GetSSOSession(c.Request.Context(), token)
	if err != nil {
		s.logger.Warn("CLI complete: invalid session", zap.Error(err))
		c.Redirect(http.StatusFound, "/login?cli=true&state="+url.QueryEscape(state)+"&error=session_expired")
		return
	}

	// Get CLI callback URL from database
	callbackURL, err := s.stateStore.GetCLICallback(c.Request.Context(), state)
	if err != nil {
		s.logger.Warn("CLI complete: callback URL not found", zap.String("state", state), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
		return
	}

	s.logger.Info("CLI complete: redirecting with existing session",
		zap.String("state", state),
		zap.String("email", session.Email),
		zap.Bool("is_admin", session.IsAdmin),
		zap.String("callback_url", callbackURL))

	// Redirect to CLI callback with token (include is_admin flag)
	redirectURL := callbackURL + "?token=" + session.Token + "&email=" + url.QueryEscape(session.Email) + "&name=" + url.QueryEscape(session.Name) + "&expires_in=86400"
	if session.IsAdmin {
		redirectURL += "&is_admin=true"
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (s *Server) handleCLICallback(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state parameter required"})
		return
	}

	// Retrieve the CLI callback URL
	callbackURLInterface, ok := cliCallbackStore.Load(state)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
		return
	}
	callbackURL := callbackURLInterface.(string)
	cliCallbackStore.Delete(state) // Clean up

	// Get user info from session (set by OIDC/SAML callback)
	userEmail, _ := c.Get("user_email")
	userName, _ := c.Get("user_name")
	token, _ := c.Get("access_token")

	if token == nil || token == "" {
		// Generate a new token for the CLI
		token = "cli-token-placeholder" // TODO: Generate proper JWT
	}

	// Build callback URL with token
	redirectURL := callbackURL + "?token=" + token.(string)
	if userEmail != nil {
		redirectURL += "&email=" + userEmail.(string)
	}
	if userName != nil {
		redirectURL += "&name=" + userName.(string)
	}
	redirectURL += "&expires_in=86400" // 24 hours

	c.Redirect(http.StatusFound, redirectURL)
}

func (s *Server) handleTokenRefresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// TODO: Validate refresh token and issue new access token
	c.JSON(http.StatusNotImplemented, gin.H{"error": "token refresh not yet implemented"})
}

// generateState creates a secure random state string
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := cryptoRand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Config generation handlers

func (s *Server) handleGenerateConfig(c *gin.Context) {
	// Check if config generation is available
	if s.ca == nil || s.configGen == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "config generation not available"})
		return
	}

	// Get authenticated user from session
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Parse request
	var req struct {
		GatewayID      string `json:"gateway_id" binding:"required"`
		CLICallbackURL string `json:"cli_callback_url"` // Optional: for CLI auto-download
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway_id is required"})
		return
	}

	// Get gateway info
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGateway(ctx, req.GatewayID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
		return
	}

	// Check if gateway is active
	if !gateway.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "gateway is not active"})
		return
	}

	// Check if user has access to this gateway (user must be assigned directly or via group)
	hasAccess, err := s.gatewayStore.UserHasGatewayAccess(ctx, user.UserID, gateway.ID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to check gateway access", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check access"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this gateway"})
		return
	}

	// Generate client certificate (valid for configured duration or 24h default)
	certValidity := s.config.PKI.CertValidity
	if certValidity == 0 {
		certValidity = 24 * time.Hour
	}

	certReq := pki.CertificateRequest{
		CommonName: user.Email,
		Email:      user.Email,
		ValidFor:   certValidity,
	}

	cert, err := s.ca.IssueClientCertificate(certReq)
	if err != nil {
		s.logger.Error("Failed to issue client certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate certificate"})
		return
	}

	// Create models for config generation
	modelGateway := &models.Gateway{
		Name:           gateway.Name,
		Hostname:       gateway.Hostname,
		PublicIP:       gateway.PublicIP,
		VPNPort:        gateway.VPNPort,
		VPNProtocol:    gateway.VPNProtocol,
		TLSAuthEnabled: gateway.TLSAuthEnabled,
	}

	modelUser := &models.User{
		Email: user.Email,
		Name:  user.Name,
	}

	// Get networks assigned to this gateway for routes
	networks, err := s.networkStore.GetGatewayNetworks(ctx, gateway.ID)
	if err != nil {
		s.logger.Warn("Failed to get gateway networks", zap.Error(err))
		// Continue without routes - not a fatal error
	}

	// Convert networks to routes
	var routes []openvpn.Route
	for _, network := range networks {
		if network.IsActive && network.CIDR != "" {
			netIP, netmask, err := cidrToNetmask(network.CIDR)
			if err != nil {
				s.logger.Warn("Invalid network CIDR", zap.String("cidr", network.CIDR), zap.Error(err))
				continue
			}
			routes = append(routes, openvpn.Route{
				Network: netIP,
				Netmask: netmask,
			})
		}
	}

	// Determine crypto profile - enforce FIPS if server requires it
	cryptoProfile := gateway.CryptoProfile
	requireFIPS := s.settingsStore.GetBool(ctx, db.SettingRequireFIPS, false)
	if requireFIPS {
		cryptoProfile = openvpn.CryptoProfileFIPS
		s.logger.Info("FIPS mode enforced by server settings", zap.String("gateway", gateway.Name))
	}

	// Generate unique config ID and auth token
	configID := generateConfigID()
	authToken := generateAuthToken()

	// Generate OpenVPN config
	genReq := openvpn.GenerateRequest{
		Gateway:       modelGateway,
		User:          modelUser,
		Certificate:   cert,
		ExpiresAt:     cert.NotAfter,
		Routes:        routes,
		CryptoProfile: cryptoProfile,
		TLSAuthKey:    gateway.TLSAuthKey, // Use gateway-specific TLS-Auth key
		AuthToken:     authToken,          // Unique token for password authentication
	}

	vpnConfig, err := s.configGen.Generate(genReq)
	if err != nil {
		s.logger.Error("Failed to generate config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate config"})
		return
	}

	// Store config in database
	dbConfig := &db.GeneratedConfig{
		ID:             configID,
		UserID:         user.UserID,
		GatewayID:      req.GatewayID,
		GatewayName:    gateway.Name,
		FileName:       vpnConfig.FileName,
		ConfigData:     vpnConfig.Content,
		SerialNumber:   cert.SerialNumber,
		Fingerprint:    cert.Fingerprint,
		CLICallbackURL: req.CLICallbackURL,
		AuthToken:      authToken, // Store token for gateway verification
		ExpiresAt:      vpnConfig.ExpiresAt,
	}

	if err := s.configStore.SaveConfig(c.Request.Context(), dbConfig); err != nil {
		s.logger.Error("Failed to save config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config"})
		return
	}

	s.logger.Info("Config generated",
		zap.String("config_id", configID),
		zap.String("user", user.Email),
		zap.String("gateway", gateway.Name),
	)

	// Return config metadata
	c.JSON(http.StatusOK, gin.H{
		"id":          configID,
		"fileName":    vpnConfig.FileName,
		"gatewayName": gateway.Name,
		"expiresAt":   vpnConfig.ExpiresAt.Format(time.RFC3339),
		"downloadUrl": "/api/v1/configs/download/" + configID,
		"cliCallback": req.CLICallbackURL != "",
	})
}

func (s *Server) handleDownloadConfig(c *gin.Context) {
	configID := c.Param("id")

	// Get config from database
	vpnConfig, err := s.configStore.GetConfig(c.Request.Context(), configID)
	if err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		if err == db.ErrConfigExpired {
			c.JSON(http.StatusGone, gin.H{"error": "config expired"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Check if this is a CLI callback request
	cliRedirect := c.Query("cli_redirect")
	if cliRedirect == "true" && vpnConfig.CLICallbackURL != "" {
		// Redirect to CLI with config data encoded
		_ = s.configStore.MarkDownloaded(c.Request.Context(), configID) // Best effort
		redirectURL := vpnConfig.CLICallbackURL + "?config_id=" + configID
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	// Mark as downloaded (best effort, don't fail download if this fails)
	_ = s.configStore.MarkDownloaded(c.Request.Context(), configID)

	// Return config file
	c.Header("Content-Disposition", "attachment; filename="+vpnConfig.FileName)
	c.Header("Content-Type", "application/x-openvpn-profile")
	c.Data(http.StatusOK, "application/x-openvpn-profile", vpnConfig.ConfigData)
}

// Helper function to get authenticated user from session or API key
func (s *Server) getAuthenticatedUser(c *gin.Context) (*authenticatedUser, error) {
	token := ""
	sessionCookie, err := c.Cookie(s.config.Auth.Session.CookieName)
	if err == nil && sessionCookie != "" {
		token = sessionCookie
	} else {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		return nil, errors.New("no session token")
	}

	// Check if it's an API key (starts with gk_)
	if strings.HasPrefix(token, "gk_") {
		keyHash := db.HashAPIKey(token)
		apiKey, ssoUser, err := s.apiKeyStore.ValidateKey(c.Request.Context(), keyHash)
		if err != nil {
			return nil, err
		}
		if apiKey == nil || ssoUser == nil {
			return nil, errors.New("invalid API key")
		}

		// Update last used (async)
		go func() { _ = s.apiKeyStore.UpdateLastUsed(context.Background(), apiKey.ID, c.ClientIP()) }()

		return &authenticatedUser{
			UserID:   ssoUser.ID,
			Email:    ssoUser.Email,
			Name:     ssoUser.Name,
			Groups:   ssoUser.Groups,
			IsAdmin:  ssoUser.IsAdmin,
			Provider: "api_key",
		}, nil
	}

	// Check SSO session
	if ssoSession, err := s.stateStore.GetSSOSession(c.Request.Context(), token); err == nil {
		return &authenticatedUser{
			UserID:   ssoSession.UserID,
			Email:    ssoSession.Email,
			Name:     ssoSession.Name,
			Groups:   ssoSession.Groups,
			IsAdmin:  ssoSession.IsAdmin,
			Provider: ssoSession.Provider,
		}, nil
	}

	// Check local session
	_, localUser, err := s.userStore.GetSession(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}

	return &authenticatedUser{
		UserID:  localUser.ID,
		Email:   localUser.Email,
		Name:    localUser.Username,
		Groups:  []string{}, // Local users don't have IdP groups
		IsAdmin: localUser.IsAdmin,
	}, nil
}

type authenticatedUser struct {
	UserID   string
	Email    string
	Name     string
	Groups   []string
	IsAdmin  bool
	Provider string
}

func generateConfigID() string {
	return uuid.New().String()
}

// generateAuthToken generates a cryptographically secure random token for config authentication
func generateAuthToken() string {
	b := make([]byte, 32) // 256-bit token
	if _, err := cryptoRand.Read(b); err != nil {
		// Fallback to UUID if crypto rand fails
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

// cidrToNetmask converts a CIDR notation (e.g., "10.0.0.0/24") to network and netmask
func cidrToNetmask(cidr string) (string, string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	// Get network address
	network := ipNet.IP.String()

	// Convert mask to dotted decimal
	mask := ipNet.Mask
	netmask := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])

	return network, netmask, nil
}

func (s *Server) handleGetConfigMetadata(c *gin.Context) {
	configID := c.Param("id")

	// Get config from database
	vpnConfig, err := s.configStore.GetConfig(c.Request.Context(), configID)
	if err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		if err == db.ErrConfigExpired {
			c.JSON(http.StatusGone, gin.H{"error": "config expired"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Return metadata only (not the actual config data)
	c.JSON(http.StatusOK, gin.H{
		"id":          vpnConfig.ID,
		"fileName":    vpnConfig.FileName,
		"gatewayName": vpnConfig.GatewayName,
		"expiresAt":   vpnConfig.ExpiresAt.Format(time.RFC3339),
		"createdAt":   vpnConfig.CreatedAt.Format(time.RFC3339),
		"downloaded":  vpnConfig.DownloadedAt != nil,
	})
}

func (s *Server) handleGetConfigRaw(c *gin.Context) {
	configID := c.Param("id")

	// Get config from database
	vpnConfig, err := s.configStore.GetConfig(c.Request.Context(), configID)
	if err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		if err == db.ErrConfigExpired {
			c.JSON(http.StatusGone, gin.H{"error": "config expired"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Mark as downloaded (best effort)
	_ = s.configStore.MarkDownloaded(c.Request.Context(), configID)

	// Return raw config content as text
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, string(vpnConfig.ConfigData))
}

// handleRevokeConfig allows users to revoke their own config
func (s *Server) handleRevokeConfig(c *gin.Context) {
	configID := c.Param("id")

	// Get user from session
	userID, _, err := s.getCurrentUserInfo(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get config to verify ownership
	config, err := s.configStore.GetConfig(c.Request.Context(), configID)
	if err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		s.logger.Error("Failed to get config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	// Verify ownership (user can only revoke their own configs)
	if config.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only revoke your own configs"})
		return
	}

	// Revoke the config
	if err := s.configStore.RevokeConfig(c.Request.Context(), configID, "revoked by user"); err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found or already revoked"})
			return
		}
		s.logger.Error("Failed to revoke config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke config"})
		return
	}

	s.logger.Info("Config revoked by user",
		zap.String("config_id", configID),
		zap.String("user_id", userID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Config revoked successfully",
	})
}

// handleListUserConfigs returns all configs for the current user
func (s *Server) handleListUserConfigs(c *gin.Context) {
	// Get user from session
	userID, _, err := s.getCurrentUserInfo(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get all configs for user
	configs, err := s.configStore.GetUserConfigs(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to get user configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get configs"})
		return
	}

	// Convert to response format
	result := make([]gin.H, len(configs))
	for i, cfg := range configs {
		result[i] = gin.H{
			"id":          cfg.ID,
			"gatewayId":   cfg.GatewayID,
			"gatewayName": cfg.GatewayName,
			"fileName":    cfg.FileName,
			"expiresAt":   cfg.ExpiresAt.Format(time.RFC3339),
			"createdAt":   cfg.CreatedAt.Format(time.RFC3339),
			"isRevoked":   cfg.IsRevoked,
			"revokedAt":   nil,
			"downloaded":  cfg.DownloadedAt != nil,
		}
		if cfg.RevokedAt != nil {
			result[i]["revokedAt"] = cfg.RevokedAt.Format(time.RFC3339)
		}
	}

	c.JSON(http.StatusOK, gin.H{"configs": result})
}

// handleAdminListAllConfigs returns all gateway configs with user info (admin only)
func (s *Server) handleAdminListAllConfigs(c *gin.Context) {
	limit := 100
	offset := 0

	configs, total, err := s.configStore.GetAllConfigs(c.Request.Context(), limit, offset)
	if err != nil {
		s.logger.Error("Failed to list all configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs"})
		return
	}

	result := make([]gin.H, len(configs))
	for i, cfg := range configs {
		result[i] = gin.H{
			"id":           cfg.ID,
			"userId":       cfg.UserID,
			"userEmail":    cfg.UserEmail,
			"userName":     cfg.UserName,
			"gatewayId":    cfg.GatewayID,
			"gatewayName":  cfg.GatewayName,
			"fileName":     cfg.FileName,
			"serialNumber": cfg.SerialNumber,
			"fingerprint":  cfg.Fingerprint,
			"expiresAt":    cfg.ExpiresAt.Format(time.RFC3339),
			"createdAt":    cfg.CreatedAt.Format(time.RFC3339),
			"isRevoked":    cfg.IsRevoked,
			"revokedAt":    nil,
			"downloaded":   cfg.DownloadedAt != nil,
		}
		if cfg.RevokedAt != nil {
			result[i]["revokedAt"] = cfg.RevokedAt.Format(time.RFC3339)
		}
		if cfg.RevokedReason != "" {
			result[i]["revokedReason"] = cfg.RevokedReason
		}
	}

	c.JSON(http.StatusOK, gin.H{"configs": result, "total": total})
}

// handleAdminRevokeConfig allows admins to revoke any config
func (s *Server) handleAdminRevokeConfig(c *gin.Context) {
	configID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "revoked by admin"
	}

	// Revoke the config
	if err := s.configStore.RevokeConfig(c.Request.Context(), configID, req.Reason); err != nil {
		if err == db.ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found or already revoked"})
			return
		}
		s.logger.Error("Failed to revoke config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke config"})
		return
	}

	s.logger.Info("Config revoked by admin", zap.String("config_id", configID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Config revoked successfully",
	})
}

// handleAdminRevokeUserConfigs allows admins to revoke all configs for a user
func (s *Server) handleAdminRevokeUserConfigs(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "all configs revoked by admin"
	}

	// Revoke all configs for user
	count, err := s.configStore.RevokeUserConfigs(c.Request.Context(), userID, req.Reason)
	if err != nil {
		s.logger.Error("Failed to revoke user configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke configs"})
		return
	}

	s.logger.Info("User configs revoked by admin",
		zap.String("user_id", userID),
		zap.Int64("count", count))

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "User configs revoked successfully",
		"revokedCount": count,
	})
}

// handleAdminListUserConfigs returns all gateway configs for a specific user
func (s *Server) handleAdminListUserConfigs(c *gin.Context) {
	userID := c.Param("id")

	configs, err := s.configStore.GetUserConfigs(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to get user configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user configs"})
		return
	}

	type configResponse struct {
		ID          string  `json:"id"`
		GatewayID   string  `json:"gatewayId"`
		GatewayName string  `json:"gatewayName"`
		FileName    string  `json:"fileName"`
		ExpiresAt   string  `json:"expiresAt"`
		CreatedAt   string  `json:"createdAt"`
		IsRevoked   bool    `json:"isRevoked"`
		RevokedAt   *string `json:"revokedAt"`
		Downloaded  bool    `json:"downloaded"`
	}

	var response []configResponse
	for _, cfg := range configs {
		resp := configResponse{
			ID:          cfg.ID,
			GatewayID:   cfg.GatewayID,
			GatewayName: cfg.GatewayName,
			FileName:    cfg.FileName,
			ExpiresAt:   cfg.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt:   cfg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			IsRevoked:   cfg.IsRevoked,
			Downloaded:  cfg.DownloadedAt != nil,
		}
		if cfg.RevokedAt != nil {
			revokedAt := cfg.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
			resp.RevokedAt = &revokedAt
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// handleAdminListUserMeshConfigs returns all mesh configs for a specific user
func (s *Server) handleAdminListUserMeshConfigs(c *gin.Context) {
	userID := c.Param("id")

	configs, err := s.meshConfigStore.GetUserConfigs(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to get user mesh configs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user mesh configs"})
		return
	}

	type configResponse struct {
		ID         string  `json:"id"`
		HubID      string  `json:"hubId"`
		HubName    string  `json:"hubName"`
		FileName   string  `json:"fileName"`
		ExpiresAt  string  `json:"expiresAt"`
		CreatedAt  string  `json:"createdAt"`
		IsRevoked  bool    `json:"isRevoked"`
		RevokedAt  *string `json:"revokedAt"`
		Downloaded bool    `json:"downloaded"`
	}

	var response []configResponse
	for _, cfg := range configs {
		resp := configResponse{
			ID:         cfg.ID,
			HubID:      cfg.HubID,
			HubName:    cfg.HubName,
			FileName:   cfg.FileName,
			ExpiresAt:  cfg.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt:  cfg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			IsRevoked:  cfg.IsRevoked,
			Downloaded: cfg.DownloadedAt != nil,
		}
		if cfg.RevokedAt != nil {
			revokedAt := cfg.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
			resp.RevokedAt = &revokedAt
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// Certificate handlers

func (s *Server) handleGetCACert(c *gin.Context) {
	// TODO: Return CA certificate
	c.JSON(http.StatusNotImplemented, gin.H{"error": "get CA cert not yet implemented"})
}

func (s *Server) handleRevokeCert(c *gin.Context) {
	// TODO: Revoke a certificate
	c.JSON(http.StatusNotImplemented, gin.H{"error": "revoke cert not yet implemented"})
}

// Policy handlers

func (s *Server) handleListPolicies(c *gin.Context) {
	// TODO: List all policies
	c.JSON(http.StatusNotImplemented, gin.H{"error": "list policies not yet implemented"})
}

func (s *Server) handleCreatePolicy(c *gin.Context) {
	// TODO: Create a new policy
	c.JSON(http.StatusNotImplemented, gin.H{"error": "create policy not yet implemented"})
}

func (s *Server) handleGetPolicy(c *gin.Context) {
	policyID := c.Param("id")
	// TODO: Get a specific policy
	c.JSON(http.StatusNotImplemented, gin.H{"error": "get policy not yet implemented", "policy_id": policyID})
}

func (s *Server) handleUpdatePolicy(c *gin.Context) {
	policyID := c.Param("id")
	// TODO: Update a policy
	c.JSON(http.StatusNotImplemented, gin.H{"error": "update policy not yet implemented", "policy_id": policyID})
}

func (s *Server) handleDeletePolicy(c *gin.Context) {
	policyID := c.Param("id")
	// TODO: Delete a policy
	c.JSON(http.StatusNotImplemented, gin.H{"error": "delete policy not yet implemented", "policy_id": policyID})
}

// Gateway handlers (internal API for gateways)

func (s *Server) handleGatewayVerify(c *gin.Context) {
	// Verify a client connection request from gateway agent
	var req struct {
		Token        string `json:"token" binding:"required"`
		CommonName   string `json:"common_name" binding:"required"`
		Username     string `json:"username"` // auth-user-pass username (email)
		Password     string `json:"password"` // auth-user-pass password (auth token)
		SerialNumber string `json:"serial_number"`
		ClientIP     string `json:"client_ip"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		s.logger.Warn("Gateway verify: invalid token", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token", "allowed": false})
		return
	}

	// Verify auth token (password) if provided - this is the primary authentication method
	var config *db.GeneratedConfig
	if req.Password != "" {
		var err error
		config, err = s.configStore.GetConfigByAuthToken(ctx, req.Password)
		if err != nil {
			if err == db.ErrConfigRevoked {
				s.logger.Warn("Gateway verify: config revoked",
					zap.String("username", req.Username))
				c.JSON(http.StatusOK, gin.H{
					"allowed": false,
					"reason":  "access revoked",
				})
				return
			}
			if err == db.ErrConfigExpired {
				s.logger.Warn("Gateway verify: config expired",
					zap.String("username", req.Username))
				c.JSON(http.StatusOK, gin.H{
					"allowed": false,
					"reason":  "config expired",
				})
				return
			}
			s.logger.Warn("Gateway verify: invalid auth token",
				zap.String("username", req.Username),
				zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "invalid credentials",
			})
			return
		}

		// Verify the username matches the config's user (email)
		// Username in auth-user-pass should be the user's email
		if req.Username != "" && config.UserID != "" {
			// Get user to verify email matches
			user, err := s.userStore.GetSSOUser(ctx, config.UserID)
			if err == nil && user.Email != req.Username {
				s.logger.Warn("Gateway verify: username mismatch",
					zap.String("provided", req.Username),
					zap.String("expected", user.Email))
				c.JSON(http.StatusOK, gin.H{
					"allowed": false,
					"reason":  "username mismatch",
				})
				return
			}
		}

		// Check if the config was issued for this gateway
		if config.GatewayID != gateway.ID {
			s.logger.Warn("Gateway verify: config not for this gateway",
				zap.String("config_gateway", config.GatewayID),
				zap.String("request_gateway", gateway.ID))
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "config not valid for this gateway",
			})
			return
		}

		s.logger.Info("Gateway verify: auth token valid",
			zap.String("config_id", config.ID),
			zap.String("user_id", config.UserID),
			zap.String("gateway", gateway.Name))
	} else if req.SerialNumber != "" {
		// Fallback to certificate verification if no password provided
		var err error
		config, err = s.configStore.GetConfigBySerial(ctx, req.SerialNumber)
		if err != nil {
			s.logger.Warn("Gateway verify: certificate not found",
				zap.String("serial", req.SerialNumber),
				zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "certificate not found or revoked",
			})
			return
		}

		// Check if config is revoked
		if config.IsRevoked {
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "access revoked",
			})
			return
		}

		// Check if certificate has expired
		if time.Now().After(config.ExpiresAt) {
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "certificate expired",
			})
			return
		}

		// Check if the config was issued for this gateway
		if config.GatewayID != gateway.ID {
			s.logger.Warn("Gateway verify: config not for this gateway",
				zap.String("config_gateway", config.GatewayID),
				zap.String("request_gateway", gateway.ID))
			c.JSON(http.StatusOK, gin.H{
				"allowed": false,
				"reason":  "certificate not valid for this gateway",
			})
			return
		}
	}

	// Look up the user by email (common_name is the email)
	user, err := s.userStore.GetSSOUserByEmail(ctx, req.CommonName)
	if err != nil {
		s.logger.Warn("Gateway verify: user not found",
			zap.String("common_name", req.CommonName),
			zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"allowed": false,
			"reason":  "user not found",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"allowed": false,
			"reason":  "user account is disabled",
		})
		return
	}

	// Check if user has access to this gateway
	hasAccess, err := s.gatewayStore.UserHasGatewayAccess(ctx, user.ID, gateway.ID, user.Groups)
	if err != nil {
		s.logger.Error("Gateway verify: failed to check access", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"allowed": false,
			"reason":  "access check failed",
		})
		return
	}
	if !hasAccess {
		s.logger.Warn("Gateway verify: user does not have gateway access",
			zap.String("user", user.Email),
			zap.String("gateway", gateway.Name))
		c.JSON(http.StatusOK, gin.H{
			"allowed": false,
			"reason":  "user does not have access to this gateway",
		})
		return
	}

	s.logger.Info("Gateway verify: connection allowed",
		zap.String("gateway", gateway.Name),
		zap.String("user", user.Email),
		zap.String("client_ip", req.ClientIP))

	c.JSON(http.StatusOK, gin.H{
		"allowed":      true,
		"gateway_id":   gateway.ID,
		"gateway_name": gateway.Name,
		"user_id":      user.ID,
		"user_email":   user.Email,
	})
}

func (s *Server) handleGatewayConnect(c *gin.Context) {
	// Record a client connection from gateway agent
	var req struct {
		Token        string `json:"token" binding:"required"`
		CommonName   string `json:"common_name" binding:"required"`
		ClientIP     string `json:"client_ip" binding:"required"`
		VPNIPv4      string `json:"vpn_ipv4"`
		VPNIPv6      string `json:"vpn_ipv6"`
		SerialNumber string `json:"serial_number"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	// Look up the user by email (common_name is the email)
	user, err := s.userStore.GetSSOUserByEmail(ctx, req.CommonName)
	if err != nil {
		s.logger.Warn("Gateway connect: user not found",
			zap.String("common_name", req.CommonName),
			zap.Error(err))
		c.JSON(http.StatusForbidden, gin.H{"error": "user not found"})
		return
	}

	// Check if user has access to this gateway (defense in depth)
	hasAccess, err := s.gatewayStore.UserHasGatewayAccess(ctx, user.ID, gateway.ID, user.Groups)
	if err != nil || !hasAccess {
		s.logger.Warn("Gateway connect: access denied",
			zap.String("user", user.Email),
			zap.String("gateway", gateway.Name))
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Get the user's access rules for firewall enforcement
	// Only get rules for networks assigned to this specific gateway
	accessRules, err := s.accessRuleStore.GetUserAccessRulesForGateway(ctx, user.ID, user.Groups, gateway.ID)
	if err != nil {
		s.logger.Error("Gateway connect: failed to get access rules", zap.Error(err))
		// Continue but with empty rules (default deny)
		accessRules = nil
	}

	// Build firewall rules from access rules
	// Default: DENY ALL
	firewallRules := []gin.H{}
	clientConfig := []string{}

	// If full tunnel mode is enabled, push default route for all traffic
	if gateway.FullTunnelMode {
		clientConfig = append(clientConfig, "push \"redirect-gateway def1 bypass-dhcp\"")
	}

	// Push DNS servers if enabled
	if gateway.PushDNS {
		if len(gateway.DNSServers) > 0 {
			// Use custom DNS servers configured for this gateway
			for _, dns := range gateway.DNSServers {
				clientConfig = append(clientConfig, fmt.Sprintf("push \"dhcp-option DNS %s\"", dns))
			}
		} else {
			// Fallback to public DNS if push_dns is enabled but no servers configured
			clientConfig = append(clientConfig, "push \"dhcp-option DNS 1.1.1.1\"")
			clientConfig = append(clientConfig, "push \"dhcp-option DNS 8.8.8.8\"")
		}
	}

	for _, rule := range accessRules {
		if !rule.IsActive {
			continue
		}
		fwRule := gin.H{
			"action":    "allow",
			"rule_type": rule.RuleType,
			"value":     rule.Value,
		}
		if rule.PortRange != nil {
			fwRule["port_range"] = *rule.PortRange
		}
		if rule.Protocol != nil {
			fwRule["protocol"] = *rule.Protocol
		}
		firewallRules = append(firewallRules, fwRule)

		// For split tunnel mode, push routes for CIDR and IP rules
		if !gateway.FullTunnelMode {
			var route string
			switch rule.RuleType {
			case db.AccessRuleTypeCIDR:
				// Convert CIDR to OpenVPN route format (network netmask)
				route = cidrToRoute(rule.Value)
			case db.AccessRuleTypeIP:
				// Single IP is a /32 CIDR
				route = cidrToRoute(rule.Value + "/32")
			case db.AccessRuleTypeHostname, db.AccessRuleTypeHostnameWildcard:
				// Hostname rules don't generate routes
			}
			if route != "" {
				clientConfig = append(clientConfig, route)
			}
		}
	}

	s.logger.Info("Gateway connect: client connected with rules",
		zap.String("gateway", gateway.Name),
		zap.String("user", user.Email),
		zap.String("vpn_ipv4", req.VPNIPv4),
		zap.Int("rule_count", len(firewallRules)),
		zap.Bool("full_tunnel", gateway.FullTunnelMode),
		zap.Int("route_count", len(clientConfig)))

	c.JSON(http.StatusOK, gin.H{
		"allow":          true,
		"status":         "connected",
		"gateway_id":     gateway.ID,
		"gateway_name":   gateway.Name,
		"user_id":        user.ID,
		"user_email":     user.Email,
		"default_policy": "deny",
		"firewall_rules": firewallRules,
		"client_config":  clientConfig,
	})
}

func (s *Server) handleGatewayDisconnect(c *gin.Context) {
	// Record a client disconnection from gateway agent
	var req struct {
		Token      string `json:"token" binding:"required"`
		CommonName string `json:"common_name" binding:"required"`
		ClientIP   string `json:"client_ip"`
		Duration   int64  `json:"duration_seconds"`
		BytesSent  int64  `json:"bytes_sent"`
		BytesRecv  int64  `json:"bytes_received"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	s.logger.Info("Gateway disconnect: client disconnected",
		zap.String("gateway", gateway.Name),
		zap.String("common_name", req.CommonName),
		zap.Int64("duration_seconds", req.Duration),
		zap.Int64("bytes_sent", req.BytesSent),
		zap.Int64("bytes_received", req.BytesRecv))

	// TODO: Update connection record in database and remove firewall rules

	c.JSON(http.StatusOK, gin.H{
		"status":       "disconnected",
		"gateway_id":   gateway.ID,
		"gateway_name": gateway.Name,
	})
}

func (s *Server) handleGatewayHeartbeat(c *gin.Context) {
	// Process gateway heartbeat
	var req struct {
		Token          string  `json:"token" binding:"required"`
		PublicIP       string  `json:"public_ip"`
		ActiveClients  int     `json:"active_clients"`
		CPUUsage       float64 `json:"cpu_usage"`
		MemoryUsage    float64 `json:"memory_usage"`
		OpenVPNRunning bool    `json:"openvpn_running"`
		ConfigVersion  string  `json:"config_version"` // Gateway's current config version
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify and update gateway
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	// Update gateway heartbeat and status
	if req.PublicIP != "" {
		err = s.gatewayStore.UpdateGatewayStatus(ctx, gateway.ID, req.PublicIP)
	} else {
		err = s.gatewayStore.UpdateHeartbeat(ctx, gateway.ID)
	}

	if err != nil {
		s.logger.Error("Failed to update gateway heartbeat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update heartbeat"})
		return
	}

	// Check if gateway needs to reprovision
	// Trigger reprovision if:
	// 1. Gateway sends empty version AND server has a version (new/reset gateway needs initial provision)
	// 2. Gateway version doesn't match server version (config changed)
	needsReprovision := false
	if gateway.ConfigVersion != "" {
		if req.ConfigVersion == "" {
			needsReprovision = true
			s.logger.Info("Gateway has no config version, signaling initial reprovision",
				zap.String("gateway", gateway.Name),
				zap.String("server_version", gateway.ConfigVersion))
		} else if req.ConfigVersion != gateway.ConfigVersion {
			needsReprovision = true
			s.logger.Info("Gateway config version mismatch, signaling reprovision",
				zap.String("gateway", gateway.Name),
				zap.String("gateway_version", req.ConfigVersion),
				zap.String("server_version", gateway.ConfigVersion))
		}
	}

	// Get CA fingerprint for rotation detection
	caFingerprint := ""
	if s.ca != nil && s.ca.Certificate() != nil {
		caFingerprint = pki.Fingerprint(s.ca.Certificate())
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "ok",
		"gateway_id":        gateway.ID,
		"gateway_name":      gateway.Name,
		"config_version":    gateway.ConfigVersion,
		"needs_reprovision": needsReprovision,
		"ca_fingerprint":    caFingerprint,
	})
}

// handleGatewayProvision provisions certificates for a gateway to run OpenVPN server
func (s *Server) handleGatewayProvision(c *gin.Context) {
	if s.ca == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PKI not configured"})
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	// Issue server certificate for this gateway
	certReq := pki.CertificateRequest{
		CommonName: fmt.Sprintf("gateway-%s", gateway.Name),
		ValidFor:   365 * 24 * time.Hour, // 1 year validity for server certs
	}

	cert, err := s.ca.IssueServerCertificate(certReq)
	if err != nil {
		s.logger.Error("Failed to issue gateway server certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue certificate"})
		return
	}

	// Generate DH parameters (or use pre-generated ones)
	// For simplicity, we'll use ECDH which doesn't need DH params
	// OpenVPN 2.4+ supports this with "dh none" and ecdh-curve

	// Use gateway's VPN subnet or default
	vpnSubnet := gateway.VPNSubnet
	if vpnSubnet == "" {
		vpnSubnet = db.DefaultVPNSubnet
	}

	// Parse subnet to get network and netmask
	vpnNetwork, vpnNetmask := parseSubnetToNetworkMask(vpnSubnet)

	// Get or generate TLS-Auth key if enabled for this gateway
	var tlsAuthKey string
	if gateway.TLSAuthEnabled {
		// First check if gateway already has a TLS-Auth key in database
		if gateway.TLSAuthKey != "" {
			tlsAuthKey = gateway.TLSAuthKey
			s.logger.Info("Using existing TLS-Auth key from database", zap.String("gateway", gateway.Name))
		} else {
			// Generate new TLS-Auth key
			tlsAuthKeyBytes, err := openvpn.GenerateTLSAuthKey()
			if err != nil {
				s.logger.Error("Failed to generate TLS-Auth key", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TLS-Auth key"})
				return
			}
			tlsAuthKey = string(tlsAuthKeyBytes)

			// Store TLS-Auth key in database for client config generation
			if err := s.gatewayStore.SetTLSAuthKey(ctx, gateway.ID, tlsAuthKey); err != nil {
				s.logger.Error("Failed to store TLS-Auth key", zap.Error(err))
				// Non-fatal - continue with provisioning
			}
			s.logger.Info("Generated new TLS-Auth key", zap.String("gateway", gateway.Name))
		}
	}

	s.logger.Info("Gateway provisioned",
		zap.String("gateway", gateway.Name),
		zap.String("serial", cert.SerialNumber),
		zap.String("vpn_subnet", vpnSubnet),
		zap.Bool("tls_auth_enabled", gateway.TLSAuthEnabled))

	response := gin.H{
		"gateway_id":       gateway.ID,
		"gateway_name":     gateway.Name,
		"ca_cert":          string(s.ca.CertificatePEM()),
		"server_cert":      string(cert.CertificatePEM),
		"server_key":       string(cert.PrivateKeyPEM),
		"vpn_subnet":       vpnSubnet,
		"vpn_network":      vpnNetwork,
		"vpn_netmask":      vpnNetmask,
		"vpn_port":         gateway.VPNPort,
		"vpn_protocol":     gateway.VPNProtocol,
		"crypto_profile":   gateway.CryptoProfile,
		"tls_auth_enabled": gateway.TLSAuthEnabled,
	}

	// Only include TLS-Auth key if enabled
	if gateway.TLSAuthEnabled && tlsAuthKey != "" {
		response["tls_auth_key"] = tlsAuthKey
	}

	c.JSON(http.StatusOK, response)
}

// parseSubnetToNetworkMask converts CIDR (e.g., "10.8.0.0/24") to network and netmask
func parseSubnetToNetworkMask(cidr string) (string, string) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Default fallback
		return "10.8.0.0", "255.255.255.0"
	}
	network := ipNet.IP.String()
	mask := ipNet.Mask
	netmask := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
	return network, netmask
}

// handleGatewayClientRules returns access rules for a specific client
// Called by gateway agent when a client connects to determine allowed destinations
func (s *Server) handleGatewayClientRules(c *gin.Context) {
	var req struct {
		Token      string   `json:"token" binding:"required"`
		UserID     string   `json:"user_id" binding:"required"`
		UserEmail  string   `json:"user_email"`
		UserGroups []string `json:"user_groups"`
		ClientIP   string   `json:"client_ip"` // VPN IP assigned to client
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	// The UserID from the certificate common name is actually the user's email
	// We need to look up the user by email to get their UUID for rule lookup
	var userID string
	var userGroups []string

	// Try to find user by email (the common name in the cert is the email)
	ssoUser, err := s.userStore.GetSSOUserByEmail(ctx, req.UserID)
	if err == nil && ssoUser != nil {
		userID = ssoUser.ID
		userGroups = ssoUser.Groups
	} else if localUser, localErr := s.userStore.GetLocalUserByEmail(ctx, req.UserID); localErr == nil && localUser != nil {
		// Check if it's a local user
		userID = localUser.ID
		userGroups = []string{} // Local users don't have groups
	} else {
		// If no user found, return empty rules (deny all)
		s.logger.Warn("User not found for access rules", zap.String("user_id", req.UserID))
		c.JSON(http.StatusOK, gin.H{
			"user_id":   req.UserID,
			"client_ip": req.ClientIP,
			"allowed":   []interface{}{},
			"default":   "deny",
		})
		return
	}

	// Get access rules for this user
	// Rules come from: user_access_rules + group_access_rules
	rules, err := s.accessRuleStore.GetUserAccessRules(ctx, userID, userGroups)
	if err != nil {
		s.logger.Error("Failed to get user access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access rules"})
		return
	}

	// Convert rules to firewall-friendly format
	type AllowedDestination struct {
		Type     string `json:"type"`     // "ip", "cidr", "hostname"
		Value    string `json:"value"`    // IP address, CIDR, or hostname
		Port     string `json:"port"`     // Port or port range (empty = all)
		Protocol string `json:"protocol"` // tcp, udp, or empty for both
	}

	allowed := make([]AllowedDestination, 0)
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		port := ""
		if rule.PortRange != nil {
			port = *rule.PortRange
		}
		protocol := ""
		if rule.Protocol != nil {
			protocol = *rule.Protocol
		}
		dest := AllowedDestination{
			Type:     string(rule.RuleType),
			Value:    rule.Value,
			Port:     port,
			Protocol: protocol,
		}
		allowed = append(allowed, dest)
	}

	s.logger.Info("Client rules requested",
		zap.String("gateway", gateway.Name),
		zap.String("user_id", req.UserID),
		zap.String("client_ip", req.ClientIP),
		zap.Int("rules_count", len(allowed)))

	c.JSON(http.StatusOK, gin.H{
		"user_id":     req.UserID,
		"client_ip":   req.ClientIP,
		"allowed":     allowed,
		"default":     "deny", // Default policy is deny
		"last_update": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGatewayAllRules returns all active access rules for periodic refresh
// Gateway can poll this to detect changes
func (s *Server) handleGatewayAllRules(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify gateway token
	ctx := c.Request.Context()
	gateway, err := s.gatewayStore.GetGatewayByToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway token"})
		return
	}

	// Get all active access rules
	rules, err := s.accessRuleStore.ListAccessRules(ctx)
	if err != nil {
		s.logger.Error("Failed to list access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
		return
	}

	// Get all user-rule and group-rule assignments
	userRules, err := s.accessRuleStore.GetAllUserAccessRuleAssignments(ctx)
	if err != nil {
		s.logger.Warn("Failed to get user rule assignments", zap.Error(err))
		userRules = make(map[string][]string) // Continue with empty
	}

	groupRules, err := s.accessRuleStore.GetAllGroupAccessRuleAssignments(ctx)
	if err != nil {
		s.logger.Warn("Failed to get group rule assignments", zap.Error(err))
		groupRules = make(map[string][]string) // Continue with empty
	}

	// Build response
	type RuleInfo struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Value    string `json:"value"`
		Port     string `json:"port"`
		Protocol string `json:"protocol"`
		IsActive bool   `json:"is_active"`
	}

	ruleList := make([]RuleInfo, 0, len(rules))
	for _, r := range rules {
		port := ""
		if r.PortRange != nil {
			port = *r.PortRange
		}
		protocol := ""
		if r.Protocol != nil {
			protocol = *r.Protocol
		}
		ruleList = append(ruleList, RuleInfo{
			ID:       r.ID,
			Name:     r.Name,
			Type:     string(r.RuleType),
			Value:    r.Value,
			Port:     port,
			Protocol: protocol,
			IsActive: r.IsActive,
		})
	}

	// Generate a hash of the rules for change detection
	rulesHash := fmt.Sprintf("%d-%d", len(rules), time.Now().Unix()/60) // Changes every minute if rules change

	s.logger.Debug("All rules requested", zap.String("gateway", gateway.Name))

	c.JSON(http.StatusOK, gin.H{
		"rules":       ruleList,
		"user_rules":  userRules,  // map[userID][]ruleID
		"group_rules": groupRules, // map[groupName][]ruleID
		"hash":        rulesHash,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

// User handlers

func (s *Server) handleGetCurrentUser(c *gin.Context) {
	// TODO: Get current user info
	c.JSON(http.StatusNotImplemented, gin.H{"error": "get current user not yet implemented"})
}

func (s *Server) handleGetUserConnections(c *gin.Context) {
	// TODO: Get current user's connections
	c.JSON(http.StatusNotImplemented, gin.H{"error": "get user connections not yet implemented"})
}

// User gateway handlers

func (s *Server) handleListUserGateways(c *gin.Context) {
	// List gateways available to the authenticated user
	// Users only see gateways they're assigned to (directly or via group)
	ctx := c.Request.Context()

	// Get user info from session
	userID, groups, err := s.getCurrentUserInfo(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get gateways user has access to
	gateways, err := s.gatewayStore.ListUserGateways(ctx, userID, groups)
	if err != nil {
		s.logger.Error("Failed to list user gateways", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list gateways"})
		return
	}

	// Convert to API response format
	// Only include gateways that are truly active (heartbeat within last 2 minutes)
	result := make([]gin.H, 0, len(gateways))
	activeThreshold := 2 * time.Minute
	now := time.Now()

	for _, gw := range gateways {
		// Gateway is active only if it has sent a heartbeat within the threshold
		isActive := gw.LastHeartbeat != nil && now.Sub(*gw.LastHeartbeat) < activeThreshold

		// Show all gateways (both online and offline) so users know what's available
		gwData := gin.H{
			"id":          gw.ID,
			"name":        gw.Name,
			"hostname":    gw.Hostname,
			"publicIp":    gw.PublicIP,
			"vpnPort":     gw.VPNPort,
			"vpnProtocol": gw.VPNProtocol,
			"isActive":    isActive,
		}
		if gw.LastHeartbeat != nil {
			gwData["lastHeartbeat"] = gw.LastHeartbeat.Format(time.RFC3339)
		}
		result = append(result, gwData)
	}

	c.JSON(http.StatusOK, gin.H{"gateways": result})
}

// getCurrentUserInfo extracts user ID and groups from the session
func (s *Server) getCurrentUserInfo(c *gin.Context) (string, []string, error) {
	// Check for session cookie or Authorization header
	token := ""
	sessionCookie, err := c.Cookie(s.config.Auth.Session.CookieName)
	if err == nil && sessionCookie != "" {
		token = sessionCookie
	} else {
		// Also check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		return "", nil, errors.New("no session token")
	}

	// Check if it's an API key (starts with gk_)
	if strings.HasPrefix(token, "gk_") {
		keyHash := db.HashAPIKey(token)
		apiKey, ssoUser, err := s.apiKeyStore.ValidateKey(c.Request.Context(), keyHash)
		if err != nil {
			return "", nil, fmt.Errorf("invalid API key: %w", err)
		}
		if apiKey == nil || ssoUser == nil {
			return "", nil, errors.New("invalid API key")
		}
		// Update last used in background
		go func() { _ = s.apiKeyStore.UpdateLastUsed(c.Request.Context(), apiKey.ID, c.ClientIP()) }()
		return ssoUser.ID, ssoUser.Groups, nil
	}

	// First, check SSO session
	if ssoSession, err := s.stateStore.GetSSOSession(c.Request.Context(), token); err == nil {
		return ssoSession.UserID, ssoSession.Groups, nil
	}

	// Fall back to local user session
	_, user, err := s.userStore.GetSession(c.Request.Context(), token)
	if err != nil {
		return "", nil, err
	}

	return user.ID, []string{}, nil
}

// Admin handlers

func (s *Server) handleListGateways(c *gin.Context) {
	// List all registered gateways (admin only)
	ctx := c.Request.Context()

	gateways, err := s.gatewayStore.ListGateways(ctx)
	if err != nil {
		s.logger.Error("Failed to list gateways", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list gateways"})
		return
	}

	// Convert to API response format (include token for admin)
	// Compute isActive dynamically based on last_heartbeat (active if heartbeat within last 2 minutes)
	result := make([]gin.H, 0, len(gateways))
	activeThreshold := 2 * time.Minute
	now := time.Now()

	for _, gw := range gateways {
		// Gateway is active only if it has sent a heartbeat within the threshold
		isActive := gw.LastHeartbeat != nil && now.Sub(*gw.LastHeartbeat) < activeThreshold

		gwData := gin.H{
			"id":             gw.ID,
			"name":           gw.Name,
			"hostname":       gw.Hostname,
			"publicIp":       gw.PublicIP,
			"vpnPort":        gw.VPNPort,
			"vpnProtocol":    gw.VPNProtocol,
			"cryptoProfile":  gw.CryptoProfile,
			"vpnSubnet":      gw.VPNSubnet,
			"tlsAuthEnabled": gw.TLSAuthEnabled,
			"fullTunnelMode": gw.FullTunnelMode,
			"pushDns":        gw.PushDNS,
			"dnsServers":     gw.DNSServers,
			"isActive":       isActive,
			"createdAt":      gw.CreatedAt.Format(time.RFC3339),
			"updatedAt":      gw.UpdatedAt.Format(time.RFC3339),
		}
		if gw.LastHeartbeat != nil {
			gwData["lastHeartbeat"] = gw.LastHeartbeat.Format(time.RFC3339)
		}
		result = append(result, gwData)
	}

	c.JSON(http.StatusOK, gin.H{"gateways": result})
}

func (s *Server) handleRegisterGateway(c *gin.Context) {
	// Register a new gateway (admin only)
	var req struct {
		Name           string   `json:"name" binding:"required"`
		Hostname       string   `json:"hostname"`
		PublicIP       string   `json:"public_ip"`
		VPNPort        int      `json:"vpn_port"`
		VPNProtocol    string   `json:"vpn_protocol"`
		CryptoProfile  string   `json:"crypto_profile"`   // modern, fips, or compatible
		VPNSubnet      string   `json:"vpn_subnet"`       // VPN client subnet (e.g., "10.8.0.0/24")
		TLSAuthEnabled *bool    `json:"tls_auth_enabled"` // Enable TLS-Auth (default: true)
		FullTunnelMode *bool    `json:"full_tunnel_mode"` // Route all traffic through VPN (default: false)
		PushDNS        *bool    `json:"push_dns"`         // Push DNS servers to clients (default: false)
		DNSServers     []string `json:"dns_servers"`      // DNS server IPs to push
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that at least one of hostname or public_ip is provided
	if req.Hostname == "" && req.PublicIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either hostname or public_ip is required"})
		return
	}

	// Default values
	if req.VPNPort == 0 {
		req.VPNPort = 1194
	}
	if req.VPNProtocol == "" {
		req.VPNProtocol = "udp"
	}
	if req.CryptoProfile == "" {
		req.CryptoProfile = db.CryptoProfileModern
	}
	if req.VPNSubnet == "" {
		req.VPNSubnet = db.DefaultVPNSubnet
	}
	if req.DNSServers == nil {
		req.DNSServers = []string{}
	}
	// Validate crypto profile is valid
	switch req.CryptoProfile {
	case db.CryptoProfileModern, db.CryptoProfileFIPS, db.CryptoProfileCompatible:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid crypto_profile: must be 'modern', 'fips', or 'compatible'"})
		return
	}

	// Validate crypto profile is allowed by system settings
	ctx := c.Request.Context()
	if err := s.validateCryptoProfileAllowed(ctx, req.CryptoProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate authentication token
	token, err := db.GenerateToken()
	if err != nil {
		s.logger.Error("Failed to generate gateway token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Default TLS Auth to true if not specified
	tlsAuthEnabled := true
	if req.TLSAuthEnabled != nil {
		tlsAuthEnabled = *req.TLSAuthEnabled
	}

	// Default Full Tunnel Mode to false if not specified
	fullTunnelMode := false
	if req.FullTunnelMode != nil {
		fullTunnelMode = *req.FullTunnelMode
	}

	// Default Push DNS to false if not specified
	pushDNS := false
	if req.PushDNS != nil {
		pushDNS = *req.PushDNS
	}

	gateway := &db.Gateway{
		Name:           req.Name,
		Hostname:       req.Hostname,
		PublicIP:       req.PublicIP,
		VPNPort:        req.VPNPort,
		VPNProtocol:    req.VPNProtocol,
		CryptoProfile:  req.CryptoProfile,
		VPNSubnet:      req.VPNSubnet,
		TLSAuthEnabled: tlsAuthEnabled,
		FullTunnelMode: fullTunnelMode,
		PushDNS:        pushDNS,
		DNSServers:     req.DNSServers,
		Token:          token,
	}

	if err := s.gatewayStore.CreateGateway(ctx, gateway); err != nil {
		if err == db.ErrGatewayExists {
			c.JSON(http.StatusConflict, gin.H{"error": "gateway with this name already exists"})
			return
		}
		s.logger.Error("Failed to create gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create gateway"})
		return
	}

	// Fetch the created gateway to get its ID
	createdGateway, err := s.gatewayStore.GetGatewayByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to fetch created gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gateway created but failed to fetch details"})
		return
	}

	s.logger.Info("Gateway registered",
		zap.String("name", req.Name),
		zap.String("hostname", req.Hostname))

	c.JSON(http.StatusCreated, gin.H{
		"id":             createdGateway.ID,
		"name":           createdGateway.Name,
		"hostname":       createdGateway.Hostname,
		"vpnPort":        createdGateway.VPNPort,
		"vpnProtocol":    createdGateway.VPNProtocol,
		"cryptoProfile":  createdGateway.CryptoProfile,
		"tlsAuthEnabled": createdGateway.TLSAuthEnabled,
		"fullTunnelMode": createdGateway.FullTunnelMode,
		"pushDns":        createdGateway.PushDNS,
		"dnsServers":     createdGateway.DNSServers,
		"token":          token, // Only returned on creation
		"message":        "Gateway registered successfully. Save the token - it will not be shown again.",
	})
}

func (s *Server) handleDeleteGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	ctx := c.Request.Context()
	if err := s.gatewayStore.DeleteGateway(ctx, gatewayID); err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
			return
		}
		s.logger.Error("Failed to delete gateway", zap.Error(err), zap.String("id", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete gateway"})
		return
	}

	s.logger.Info("Gateway deleted", zap.String("id", gatewayID))
	c.JSON(http.StatusOK, gin.H{"message": "gateway deleted successfully"})
}

// handleReprovisionGateway forces a gateway to re-provision its certificates
// This is useful after CA rotation or to regenerate TLS-Auth keys
func (s *Server) handleReprovisionGateway(c *gin.Context) {
	gatewayID := c.Param("id")
	ctx := c.Request.Context()

	// Get current gateway
	gateway, err := s.gatewayStore.GetGateway(ctx, gatewayID)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err), zap.String("id", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway"})
		return
	}

	// Generate a new config version to trigger reprovision on next heartbeat
	// Using a UUID ensures it will never match the gateway's current version
	newConfigVersion := fmt.Sprintf("reprovision-%d", time.Now().UnixNano())

	if err := s.gatewayStore.UpdateGatewayConfigVersion(ctx, gatewayID, newConfigVersion); err != nil {
		s.logger.Error("Failed to update gateway config version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger reprovision"})
		return
	}

	s.logger.Info("Gateway reprovision triggered",
		zap.String("id", gatewayID),
		zap.String("gateway", gateway.Name),
		zap.String("new_config_version", newConfigVersion))

	c.JSON(http.StatusOK, gin.H{
		"message":        "reprovision triggered - gateway will reprovision on next heartbeat",
		"config_version": newConfigVersion,
	})
}

func (s *Server) handleUpdateGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	var req struct {
		Name           string   `json:"name" binding:"required"`
		Hostname       string   `json:"hostname"`
		PublicIP       string   `json:"public_ip"`
		VPNPort        int      `json:"vpn_port"`
		VPNProtocol    string   `json:"vpn_protocol"`
		CryptoProfile  string   `json:"crypto_profile"`   // modern, fips, or compatible
		VPNSubnet      string   `json:"vpn_subnet"`       // VPN client subnet (e.g., "10.8.0.0/24")
		TLSAuthEnabled *bool    `json:"tls_auth_enabled"` // Enable TLS-Auth
		FullTunnelMode *bool    `json:"full_tunnel_mode"` // Route all traffic through VPN
		PushDNS        *bool    `json:"push_dns"`         // Push DNS servers to clients
		DNSServers     []string `json:"dns_servers"`      // DNS server IPs to push
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that at least one of hostname or public_ip is provided
	if req.Hostname == "" && req.PublicIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either hostname or public_ip is required"})
		return
	}

	// Default values
	if req.VPNPort == 0 {
		req.VPNPort = 1194
	}
	if req.VPNProtocol == "" {
		req.VPNProtocol = "udp"
	}
	if req.CryptoProfile == "" {
		req.CryptoProfile = db.CryptoProfileModern
	}
	if req.VPNSubnet == "" {
		req.VPNSubnet = db.DefaultVPNSubnet
	}
	// Validate crypto profile is valid
	switch req.CryptoProfile {
	case db.CryptoProfileModern, db.CryptoProfileFIPS, db.CryptoProfileCompatible:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid crypto_profile: must be 'modern', 'fips', or 'compatible'"})
		return
	}

	// Validate crypto profile is allowed by system settings
	ctx := c.Request.Context()
	if err := s.validateCryptoProfileAllowed(ctx, req.CryptoProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing gateway to preserve TLSAuthEnabled if not specified
	existingGw, err := s.gatewayStore.GetGateway(ctx, gatewayID)
	if err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
			return
		}
		s.logger.Error("Failed to get gateway", zap.Error(err), zap.String("id", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway"})
		return
	}

	// Use existing TLSAuthEnabled if not specified in request
	tlsAuthEnabled := existingGw.TLSAuthEnabled
	if req.TLSAuthEnabled != nil {
		tlsAuthEnabled = *req.TLSAuthEnabled
	}

	// Use existing FullTunnelMode if not specified in request
	fullTunnelMode := existingGw.FullTunnelMode
	if req.FullTunnelMode != nil {
		fullTunnelMode = *req.FullTunnelMode
	}

	// Use existing PushDNS if not specified in request
	pushDNS := existingGw.PushDNS
	if req.PushDNS != nil {
		pushDNS = *req.PushDNS
	}

	// Use request DNSServers if provided, otherwise keep existing
	dnsServers := existingGw.DNSServers
	if req.DNSServers != nil {
		dnsServers = req.DNSServers
	}

	gw := &db.Gateway{
		ID:             gatewayID,
		Name:           req.Name,
		Hostname:       req.Hostname,
		PublicIP:       req.PublicIP,
		VPNPort:        req.VPNPort,
		VPNProtocol:    req.VPNProtocol,
		CryptoProfile:  req.CryptoProfile,
		VPNSubnet:      req.VPNSubnet,
		TLSAuthEnabled: tlsAuthEnabled,
		FullTunnelMode: fullTunnelMode,
		PushDNS:        pushDNS,
		DNSServers:     dnsServers,
	}

	if err := s.gatewayStore.UpdateGateway(ctx, gw); err != nil {
		if err == db.ErrGatewayNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
			return
		}
		if err == db.ErrGatewayExists {
			c.JSON(http.StatusConflict, gin.H{"error": "gateway with this name already exists"})
			return
		}
		s.logger.Error("Failed to update gateway", zap.Error(err), zap.String("id", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update gateway"})
		return
	}

	s.logger.Info("Gateway updated", zap.String("id", gatewayID), zap.String("name", req.Name))
	c.JSON(http.StatusOK, gin.H{"message": "gateway updated successfully"})
}

func (s *Server) handleGetGatewayUsers(c *gin.Context) {
	gatewayID := c.Param("id")
	ctx := c.Request.Context()

	users, err := s.gatewayStore.GetGatewayUsers(ctx, gatewayID)
	if err != nil {
		s.logger.Error("Failed to get gateway users", zap.Error(err), zap.String("gatewayId", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway users"})
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, u := range users {
		result = append(result, gin.H{
			"userId":    u.UserID,
			"email":     u.Email,
			"name":      u.Name,
			"createdAt": u.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": result})
}

func (s *Server) handleAssignGatewayUser(c *gin.Context) {
	gatewayID := c.Param("id")

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Resolve email/username to actual user UUID
	resolvedUserID := req.UserID
	if strings.Contains(req.UserID, "@") {
		// Looks like an email, try to find the user
		if ssoUser, err := s.userStore.GetSSOUserByEmail(ctx, req.UserID); err == nil {
			resolvedUserID = ssoUser.ID
		} else if localUser, err := s.userStore.GetLocalUserByEmail(ctx, req.UserID); err == nil {
			resolvedUserID = localUser.ID
		}
	} else {
		// Try to find by username (for local users)
		if localUser, err := s.userStore.GetLocalUserByUsername(ctx, req.UserID); err == nil {
			resolvedUserID = localUser.ID
		}
		// If not found, assume it's already a UUID
	}

	if err := s.gatewayStore.AssignUserToGateway(ctx, resolvedUserID, gatewayID); err != nil {
		s.logger.Error("Failed to assign user to gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign user to gateway"})
		return
	}

	s.logger.Info("User assigned to gateway", zap.String("userId", resolvedUserID), zap.String("gatewayId", gatewayID))
	c.JSON(http.StatusOK, gin.H{"message": "user assigned to gateway"})
}

func (s *Server) handleRemoveGatewayUser(c *gin.Context) {
	gatewayID := c.Param("id")
	userID := c.Param("userId")

	ctx := c.Request.Context()
	if err := s.gatewayStore.RemoveUserFromGateway(ctx, userID, gatewayID); err != nil {
		s.logger.Error("Failed to remove user from gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user from gateway"})
		return
	}

	s.logger.Info("User removed from gateway", zap.String("userId", userID), zap.String("gatewayId", gatewayID))
	c.JSON(http.StatusOK, gin.H{"message": "user removed from gateway"})
}

func (s *Server) handleGetGatewayGroups(c *gin.Context) {
	gatewayID := c.Param("id")
	ctx := c.Request.Context()

	groups, err := s.gatewayStore.GetGatewayGroups(ctx, gatewayID)
	if err != nil {
		s.logger.Error("Failed to get gateway groups", zap.Error(err), zap.String("gatewayId", gatewayID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway groups"})
		return
	}

	result := make([]gin.H, 0, len(groups))
	for _, g := range groups {
		result = append(result, gin.H{
			"groupName": g.GroupName,
			"createdAt": g.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"groups": result})
}

func (s *Server) handleAssignGatewayGroup(c *gin.Context) {
	gatewayID := c.Param("id")

	var req struct {
		GroupName string `json:"group_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.gatewayStore.AssignGroupToGateway(ctx, req.GroupName, gatewayID); err != nil {
		s.logger.Error("Failed to assign group to gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign group to gateway"})
		return
	}

	s.logger.Info("Group assigned to gateway", zap.String("groupName", req.GroupName), zap.String("gatewayId", gatewayID))
	c.JSON(http.StatusOK, gin.H{"message": "group assigned to gateway"})
}

func (s *Server) handleRemoveGatewayGroup(c *gin.Context) {
	gatewayID := c.Param("id")
	groupName := c.Param("groupName")

	ctx := c.Request.Context()
	if err := s.gatewayStore.RemoveGroupFromGateway(ctx, groupName, gatewayID); err != nil {
		s.logger.Error("Failed to remove group from gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove group from gateway"})
		return
	}

	s.logger.Info("Group removed from gateway", zap.String("groupName", groupName), zap.String("gatewayId", gatewayID))
	c.JSON(http.StatusOK, gin.H{"message": "group removed from gateway"})
}

func (s *Server) handleListConnections(c *gin.Context) {
	// TODO: List all active connections
	c.JSON(http.StatusNotImplemented, gin.H{"error": "list connections not yet implemented"})
}

func (s *Server) handleGetAuditLogs(c *gin.Context) {
	// TODO: Get audit logs
	c.JSON(http.StatusNotImplemented, gin.H{"error": "get audit logs not yet implemented"})
}

// Network handlers

func (s *Server) handleListNetworks(c *gin.Context) {
	ctx := c.Request.Context()
	networks, err := s.networkStore.ListNetworks(ctx)
	if err != nil {
		s.logger.Error("Failed to list networks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list networks"})
		return
	}

	result := make([]gin.H, 0, len(networks))
	for _, n := range networks {
		result = append(result, gin.H{
			"id":          n.ID,
			"name":        n.Name,
			"description": n.Description,
			"cidr":        n.CIDR,
			"isActive":    n.IsActive,
			"createdAt":   n.CreatedAt.Format(time.RFC3339),
			"updatedAt":   n.UpdatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"networks": result})
}

func (s *Server) handleCreateNetwork(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		CIDR        string `json:"cidr" binding:"required"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	network := &db.Network{
		Name:        req.Name,
		Description: req.Description,
		CIDR:        req.CIDR,
		IsActive:    isActive,
	}

	ctx := c.Request.Context()
	if err := s.networkStore.CreateNetwork(ctx, network); err != nil {
		if err == db.ErrNetworkExists {
			c.JSON(http.StatusConflict, gin.H{"error": "network with this name already exists"})
			return
		}
		s.logger.Error("Failed to create network", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create network"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          network.ID,
		"name":        network.Name,
		"description": network.Description,
		"cidr":        network.CIDR,
		"isActive":    network.IsActive,
		"createdAt":   network.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleGetNetwork(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	network, err := s.networkStore.GetNetwork(ctx, id)
	if err != nil {
		if err == db.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          network.ID,
		"name":        network.Name,
		"description": network.Description,
		"cidr":        network.CIDR,
		"isActive":    network.IsActive,
		"createdAt":   network.CreatedAt.Format(time.RFC3339),
		"updatedAt":   network.UpdatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleUpdateNetwork(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		CIDR        string `json:"cidr" binding:"required"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	network, err := s.networkStore.GetNetwork(ctx, id)
	if err != nil {
		if err == db.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get network"})
		return
	}

	network.Name = req.Name
	network.Description = req.Description
	network.CIDR = req.CIDR
	if req.IsActive != nil {
		network.IsActive = *req.IsActive
	}

	if err := s.networkStore.UpdateNetwork(ctx, network); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "network updated successfully"})
}

func (s *Server) handleDeleteNetwork(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	if err := s.networkStore.DeleteNetwork(ctx, id); err != nil {
		if err == db.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "network deleted successfully"})
}

func (s *Server) handleGetNetworkGateways(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	gateways, err := s.networkStore.GetNetworkGateways(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get network gateways"})
		return
	}

	result := make([]gin.H, 0, len(gateways))
	for _, g := range gateways {
		gw := gin.H{
			"id":          g.ID,
			"name":        g.Name,
			"hostname":    g.Hostname,
			"publicIp":    g.PublicIP,
			"vpnPort":     g.VPNPort,
			"vpnProtocol": g.VPNProtocol,
			"isActive":    g.IsActive,
		}
		if g.LastHeartbeat != nil {
			gw["lastHeartbeat"] = g.LastHeartbeat.Format(time.RFC3339)
		}
		result = append(result, gw)
	}
	c.JSON(http.StatusOK, gin.H{"gateways": result})
}

func (s *Server) handleGetNetworkAccessRules(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	rules, err := s.accessRuleStore.ListAccessRulesByNetwork(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get network access rules"})
		return
	}

	result := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		// Get users and groups for this rule
		users, _ := s.accessRuleStore.GetRuleUsers(ctx, r.ID)
		groups, _ := s.accessRuleStore.GetRuleGroups(ctx, r.ID)

		rule := gin.H{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"rule_type":   r.RuleType,
			"value":       r.Value,
			"port_range":  r.PortRange,
			"protocol":    r.Protocol,
			"network_id":  r.NetworkID,
			"is_active":   r.IsActive,
			"users":       users,
			"groups":      groups,
		}
		result = append(result, rule)
	}
	c.JSON(http.StatusOK, gin.H{"access_rules": result})
}

func (s *Server) handleGetGatewayNetworks(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	networks, err := s.networkStore.GetGatewayNetworks(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get gateway networks"})
		return
	}

	result := make([]gin.H, 0, len(networks))
	for _, n := range networks {
		result = append(result, gin.H{
			"id":          n.ID,
			"name":        n.Name,
			"description": n.Description,
			"cidr":        n.CIDR,
			"isActive":    n.IsActive,
		})
	}
	c.JSON(http.StatusOK, gin.H{"networks": result})
}

func (s *Server) handleAssignGatewayNetwork(c *gin.Context) {
	gatewayID := c.Param("id")
	var req struct {
		NetworkID string `json:"network_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.networkStore.AssignGatewayToNetwork(ctx, gatewayID, req.NetworkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign gateway to network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway assigned to network"})
}

func (s *Server) handleRemoveGatewayNetwork(c *gin.Context) {
	gatewayID := c.Param("id")
	networkID := c.Param("networkId")
	ctx := c.Request.Context()

	if err := s.networkStore.RemoveGatewayFromNetwork(ctx, gatewayID, networkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove gateway from network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway removed from network"})
}

// Access Rule handlers

func (s *Server) handleListAccessRules(c *gin.Context) {
	ctx := c.Request.Context()
	rules, err := s.accessRuleStore.ListAccessRules(ctx)
	if err != nil {
		s.logger.Error("Failed to list access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list access rules"})
		return
	}

	result := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		rule := gin.H{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"ruleType":    r.RuleType,
			"value":       r.Value,
			"isActive":    r.IsActive,
			"createdAt":   r.CreatedAt.Format(time.RFC3339),
			"updatedAt":   r.UpdatedAt.Format(time.RFC3339),
		}
		if r.PortRange != nil {
			rule["portRange"] = *r.PortRange
		}
		if r.Protocol != nil {
			rule["protocol"] = *r.Protocol
		}
		if r.NetworkID != nil {
			rule["networkId"] = *r.NetworkID
		}
		result = append(result, rule)
	}
	c.JSON(http.StatusOK, gin.H{"accessRules": result})
}

func (s *Server) handleCreateAccessRule(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		RuleType    string  `json:"rule_type" binding:"required"`
		Value       string  `json:"value" binding:"required"`
		PortRange   *string `json:"port_range"`
		Protocol    *string `json:"protocol"`
		NetworkID   *string `json:"network_id"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate rule type
	validTypes := map[string]bool{"ip": true, "cidr": true, "hostname": true, "hostname_wildcard": true}
	if !validTypes[req.RuleType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule_type, must be: ip, cidr, hostname, or hostname_wildcard"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule := &db.AccessRule{
		Name:        req.Name,
		Description: req.Description,
		RuleType:    db.AccessRuleType(req.RuleType),
		Value:       req.Value,
		PortRange:   req.PortRange,
		Protocol:    req.Protocol,
		NetworkID:   req.NetworkID,
		IsActive:    isActive,
	}

	ctx := c.Request.Context()
	if err := s.accessRuleStore.CreateAccessRule(ctx, rule); err != nil {
		s.logger.Error("Failed to create access rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        rule.ID,
		"name":      rule.Name,
		"ruleType":  rule.RuleType,
		"value":     rule.Value,
		"isActive":  rule.IsActive,
		"createdAt": rule.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleGetAccessRule(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	rule, err := s.accessRuleStore.GetAccessRule(ctx, id)
	if err != nil {
		if err == db.ErrAccessRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "access rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access rule"})
		return
	}

	// Get assigned users and groups
	users, _ := s.accessRuleStore.GetRuleUsers(ctx, id)
	groups, _ := s.accessRuleStore.GetRuleGroups(ctx, id)

	result := gin.H{
		"id":          rule.ID,
		"name":        rule.Name,
		"description": rule.Description,
		"ruleType":    rule.RuleType,
		"value":       rule.Value,
		"isActive":    rule.IsActive,
		"createdAt":   rule.CreatedAt.Format(time.RFC3339),
		"updatedAt":   rule.UpdatedAt.Format(time.RFC3339),
		"users":       users,
		"groups":      groups,
	}
	if rule.PortRange != nil {
		result["portRange"] = *rule.PortRange
	}
	if rule.Protocol != nil {
		result["protocol"] = *rule.Protocol
	}
	if rule.NetworkID != nil {
		result["networkId"] = *rule.NetworkID
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleUpdateAccessRule(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		RuleType    string  `json:"rule_type" binding:"required"`
		Value       string  `json:"value" binding:"required"`
		PortRange   *string `json:"port_range"`
		Protocol    *string `json:"protocol"`
		NetworkID   *string `json:"network_id"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	rule, err := s.accessRuleStore.GetAccessRule(ctx, id)
	if err != nil {
		if err == db.ErrAccessRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "access rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access rule"})
		return
	}

	rule.Name = req.Name
	rule.Description = req.Description
	rule.RuleType = db.AccessRuleType(req.RuleType)
	rule.Value = req.Value
	rule.PortRange = req.PortRange
	rule.Protocol = req.Protocol
	rule.NetworkID = req.NetworkID
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if err := s.accessRuleStore.UpdateAccessRule(ctx, rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update access rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "access rule updated successfully"})
}

func (s *Server) handleDeleteAccessRule(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	if err := s.accessRuleStore.DeleteAccessRule(ctx, id); err != nil {
		if err == db.ErrAccessRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "access rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete access rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "access rule deleted successfully"})
}

func (s *Server) handleAssignRuleToUser(c *gin.Context) {
	ruleID := c.Param("id")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.accessRuleStore.AssignRuleToUser(ctx, req.UserID, ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign rule to user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule assigned to user"})
}

func (s *Server) handleRemoveRuleFromUser(c *gin.Context) {
	ruleID := c.Param("id")
	userID := c.Param("userId")
	ctx := c.Request.Context()

	if err := s.accessRuleStore.RemoveRuleFromUser(ctx, userID, ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove rule from user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule removed from user"})
}

func (s *Server) handleAssignRuleToGroup(c *gin.Context) {
	ruleID := c.Param("id")
	var req struct {
		GroupName string `json:"group_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.accessRuleStore.AssignRuleToGroup(ctx, req.GroupName, ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign rule to group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule assigned to group"})
}

func (s *Server) handleRemoveRuleFromGroup(c *gin.Context) {
	ruleID := c.Param("id")
	groupName := c.Param("groupName")
	ctx := c.Request.Context()

	if err := s.accessRuleStore.RemoveRuleFromGroup(ctx, groupName, ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove rule from group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule removed from group"})
}

// Metrics handler

func (s *Server) handleMetrics(c *gin.Context) {
	// TODO: Implement Prometheus metrics
	c.String(http.StatusOK, "# HELP gatekey_info GateKey server info\n# TYPE gatekey_info gauge\ngatekey_info{version=\"0.1.0\"} 1\n")
}

// Server info handler - returns server requirements for clients
func (s *Server) handleGetServerInfo(c *gin.Context) {
	ctx := c.Request.Context()

	// Get FIPS requirement from settings
	requireFIPS := s.settingsStore.GetBool(ctx, db.SettingRequireFIPS, false)

	c.JSON(http.StatusOK, gin.H{
		"require_fips": requireFIPS,
		"version":      "0.1.0",
	})
}

// Admin settings handlers

func (s *Server) handleGetSettings(c *gin.Context) {
	ctx := c.Request.Context()

	// Get settings from database
	settings, err := s.settingsStore.GetAll(ctx)
	if err != nil {
		s.logger.Error("Failed to get settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	// Convert to map for easier frontend consumption
	settingsMap := make(map[string]string)
	for _, setting := range settings {
		settingsMap[setting.Key] = setting.Value
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsMap})
}

func (s *Server) handleUpdateSettings(c *gin.Context) {
	ctx := c.Request.Context()

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate allowed settings
	allowedSettings := map[string]bool{
		db.SettingSessionDurationHours:  true,
		db.SettingSecureCookies:         true,
		db.SettingVPNCertValidityHours:  true,
		db.SettingRequireFIPS:           true,
		db.SettingAllowedCryptoProfiles: true,
		db.SettingMinTLSVersion:         true,
		db.SettingAllowedCiphers:        true,
	}

	for key, value := range req {
		if !allowedSettings[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid setting key: " + key})
			return
		}
		if err := s.settingsStore.Set(ctx, key, value); err != nil {
			s.logger.Error("Failed to update setting", zap.String("key", key), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
			return
		}
	}

	s.logger.Info("Settings updated", zap.Any("settings", req))
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

// Install script and binary download handlers

func (s *Server) handleInstallScript(c *gin.Context) {
	// Serve the gateway installer script from file
	// Try multiple paths for development and production
	scriptPaths := []string{
		"/app/scripts/install-gateway.sh", // Docker container
		"scripts/install-gateway.sh",      // Local development
		"../scripts/install-gateway.sh",   // Running from cmd/
	}

	var script []byte
	var err error
	for _, path := range scriptPaths {
		script, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		s.logger.Error("Failed to read install script", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "install script not found"})
		return
	}

	c.Header("Content-Type", "text/x-shellscript")
	c.Header("Content-Disposition", "attachment; filename=install-gateway.sh")
	c.String(http.StatusOK, string(script))
}

func (s *Server) handleHubInstallScript(c *gin.Context) {
	// Serve the hub installer script from file
	scriptPaths := []string{
		"/app/scripts/install-hub.sh",
		"scripts/install-hub.sh",
		"../scripts/install-hub.sh",
	}

	var script []byte
	var err error
	for _, path := range scriptPaths {
		script, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		s.logger.Error("Failed to read hub install script", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hub install script not found"})
		return
	}

	c.Header("Content-Type", "text/x-shellscript")
	c.Header("Content-Disposition", "attachment; filename=install-hub.sh")
	c.String(http.StatusOK, string(script))
}

func (s *Server) handleMeshSpokeGenericInstallScript(c *gin.Context) {
	// Serve the mesh spoke installer script from file
	scriptPaths := []string{
		"/app/scripts/install-mesh-spoke.sh",
		"scripts/install-mesh-spoke.sh",
		"../scripts/install-mesh-spoke.sh",
	}

	var script []byte
	var err error
	for _, path := range scriptPaths {
		script, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		s.logger.Error("Failed to read mesh spoke install script", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mesh spoke install script not found"})
		return
	}

	c.Header("Content-Type", "text/x-shellscript")
	c.Header("Content-Disposition", "attachment; filename=install-mesh-spoke.sh")
	c.String(http.StatusOK, string(script))
}

func (s *Server) handleDownloadBinary(c *gin.Context) {
	filename := c.Param("filename")

	// Map filename to GitHub release asset
	allowedBinaries := map[string]bool{
		// Gateway binaries
		"gatekey-gateway-linux-amd64":       true,
		"gatekey-gateway-linux-arm64":       true,
		"gatekey-gateway-darwin-amd64":      true,
		"gatekey-gateway-darwin-arm64":      true,
		"gatekey-gateway-windows-amd64.exe": true,
		// Client binaries
		"gatekey-linux-amd64":       true,
		"gatekey-linux-arm64":       true,
		"gatekey-darwin-amd64":      true,
		"gatekey-darwin-arm64":      true,
		"gatekey-windows-amd64.exe": true,
		// Admin CLI binaries
		"gatekey-admin-linux-amd64":  true,
		"gatekey-admin-linux-arm64":  true,
		"gatekey-admin-darwin-amd64": true,
		"gatekey-admin-darwin-arm64": true,
		// Hub binaries
		"gatekey-hub-linux-amd64":  true,
		"gatekey-hub-linux-arm64":  true,
		"gatekey-hub-darwin-amd64": true,
		"gatekey-hub-darwin-arm64": true,
		// Mesh spoke binaries
		"gatekey-mesh-spoke-linux-amd64":  true,
		"gatekey-mesh-spoke-linux-arm64":  true,
		"gatekey-mesh-spoke-darwin-amd64": true,
		"gatekey-mesh-spoke-darwin-arm64": true,
	}

	if !allowedBinaries[filename] {
		c.JSON(http.StatusNotFound, gin.H{"error": "binary not found"})
		return
	}

	// Map download names to actual file names (mesh-spoke -> mesh-gateway)
	actualFilename := filename
	filenameMapping := map[string]string{
		"gatekey-mesh-spoke-linux-amd64":  "gatekey-mesh-gateway-linux-amd64",
		"gatekey-mesh-spoke-linux-arm64":  "gatekey-mesh-gateway-linux-arm64",
		"gatekey-mesh-spoke-darwin-amd64": "gatekey-mesh-gateway-darwin-amd64",
		"gatekey-mesh-spoke-darwin-arm64": "gatekey-mesh-gateway-darwin-arm64",
	}
	if mapped, ok := filenameMapping[filename]; ok {
		actualFilename = mapped
	}

	// Try multiple paths for development and production
	binPaths := []string{
		"/app/bin/" + actualFilename, // Docker container
		"./bin/" + actualFilename,    // Local development
		"../bin/" + actualFilename,   // Running from cmd/
	}

	for _, binPath := range binPaths {
		if _, err := os.Stat(binPath); err == nil {
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", "attachment; filename="+filename)
			c.File(binPath)
			return
		}
	}

	// Redirect to GitHub releases for production deployments
	githubReleasesURL := "https://github.com/dye-tech/GateKey/releases/latest/download/" + filename
	c.Redirect(http.StatusTemporaryRedirect, githubReleasesURL)
}

func (s *Server) handleDownloadsPage(c *gin.Context) {
	// Return a simple HTML page listing available downloads
	serverURL := c.Request.Host
	// Check X-Forwarded-Proto header first (for reverse proxy/Istio)
	protocol := c.GetHeader("X-Forwarded-Proto")
	if protocol == "" {
		if c.Request.TLS != nil {
			protocol = "https"
		} else {
			protocol = "http"
		}
	}
	baseURL := protocol + "://" + serverURL

	html := `<!DOCTYPE html>
<html>
<head>
    <title>GateKey Downloads</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #1a56db; }
        .section { margin: 30px 0; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px; }
        .section h2 { margin-top: 0; color: #374151; }
        code { background: #f3f4f6; padding: 2px 6px; border-radius: 4px; }
        pre { background: #1f2937; color: #f9fafb; padding: 15px; border-radius: 8px; overflow-x: auto; }
        a { color: #1a56db; }
        ul { line-height: 2; }
    </style>
</head>
<body>
    <h1>GateKey Downloads</h1>

    <div class="section">
        <h2>VPN Client (gatekey)</h2>
        <p>The GateKey client allows you to connect to VPN gateways.</p>
        <h3>Quick Install (Linux/macOS)</h3>
        <pre>curl -sSL ` + baseURL + `/scripts/install-client.sh | bash</pre>
        <h3>Manual Downloads</h3>
        <ul>
            <li><a href="/downloads/gatekey-linux-amd64">Linux (x86_64)</a></li>
            <li><a href="/downloads/gatekey-linux-arm64">Linux (ARM64)</a></li>
            <li><a href="/downloads/gatekey-darwin-amd64">macOS (Intel)</a></li>
            <li><a href="/downloads/gatekey-darwin-arm64">macOS (Apple Silicon)</a></li>
            <li><a href="/downloads/gatekey-windows-amd64.exe">Windows (x86_64)</a></li>
        </ul>
    </div>

    <div class="section">
        <h2>Gateway Agent (gatekey-gateway)</h2>
        <p>The gateway agent runs alongside OpenVPN to provide zero-trust access control.</p>
        <h3>Quick Install (Linux)</h3>
        <pre>curl -sSL ` + baseURL + `/scripts/install-gateway.sh | bash -s -- \
  --server ` + baseURL + ` \
  --token YOUR_GATEWAY_TOKEN \
  --name my-gateway</pre>
        <h3>Manual Downloads</h3>
        <ul>
            <li><a href="/downloads/gatekey-gateway-linux-amd64">Linux (x86_64)</a></li>
            <li><a href="/downloads/gatekey-gateway-linux-arm64">Linux (ARM64)</a></li>
            <li><a href="/downloads/gatekey-gateway-darwin-amd64">macOS (Intel)</a></li>
            <li><a href="/downloads/gatekey-gateway-darwin-arm64">macOS (Apple Silicon)</a></li>
        </ul>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleClientInstallScript(c *gin.Context) {
	serverURL := c.Request.Host
	// Check X-Forwarded-Proto header first (for reverse proxy/Istio)
	protocol := c.GetHeader("X-Forwarded-Proto")
	if protocol == "" {
		if c.Request.TLS != nil {
			protocol = "https"
		} else {
			protocol = "http"
		}
	}

	script := `#!/bin/bash
# GateKey Client Installer
# This script installs the GateKey VPN client.
#
# Usage:
#   curl -sSL ` + protocol + `://` + serverURL + `/scripts/install-client.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

GATEKEY_SERVER="` + protocol + `://` + serverURL + `"
INSTALL_DIR="/usr/local/bin"

echo -e "${GREEN}GateKey Client Installer${NC}"
echo "========================="
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    linux|darwin)
        BINARY_NAME="gatekey-${OS}-${ARCH}"
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

DOWNLOAD_URL="${GATEKEY_SERVER}/downloads/${BINARY_NAME}"

echo "Detected: $OS ($ARCH)"
echo "Downloading from: $DOWNLOAD_URL"
echo ""

# Check for curl or wget
if command -v curl &> /dev/null; then
    DOWNLOADER="curl -fsSL -o"
elif command -v wget &> /dev/null; then
    DOWNLOADER="wget -q -O"
else
    echo -e "${RED}Error: Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
fi

# Create temp file
TMP_FILE=$(mktemp)
trap "rm -f $TMP_FILE" EXIT

echo -e "${YELLOW}Downloading GateKey client...${NC}"
$DOWNLOADER "$TMP_FILE" "$DOWNLOAD_URL"

if [ ! -s "$TMP_FILE" ]; then
    echo -e "${RED}Error: Download failed or file is empty${NC}"
    exit 1
fi

# Install binary
echo -e "${YELLOW}Installing to $INSTALL_DIR/gatekey...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/gatekey"
    chmod +x "$INSTALL_DIR/gatekey"
else
    echo "Root permissions required to install to $INSTALL_DIR"
    sudo mv "$TMP_FILE" "$INSTALL_DIR/gatekey"
    sudo chmod +x "$INSTALL_DIR/gatekey"
fi

echo ""
echo -e "${GREEN}GateKey client installed successfully!${NC}"
echo ""
echo "Getting started:"
echo "  1. Configure the client:"
echo "     gatekey config init --server $GATEKEY_SERVER"
echo ""
echo "  2. Login to your account:"
echo "     gatekey login"
echo ""
echo "  3. Connect to VPN:"
echo "     gatekey connect"
echo ""
`

	c.Header("Content-Type", "text/x-shellscript")
	c.String(http.StatusOK, script)
}

// User management handlers

func (s *Server) handleListUsers(c *gin.Context) {
	ctx := c.Request.Context()
	users, err := s.userStore.ListSSOUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	// Build response with user details
	response := make([]gin.H, 0, len(users))
	for _, u := range users {
		response = append(response, gin.H{
			"id":            u.ID,
			"external_id":   u.ExternalID,
			"provider":      u.Provider,
			"email":         u.Email,
			"name":          u.Name,
			"groups":        u.Groups,
			"is_admin":      u.IsAdmin,
			"is_active":     u.IsActive,
			"last_login_at": u.LastLoginAt,
			"created_at":    u.CreatedAt,
			"updated_at":    u.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

func (s *Server) handleGetUser(c *gin.Context) {
	userID := c.Param("id")
	ctx := c.Request.Context()

	user, err := s.userStore.GetSSOUser(ctx, userID)
	if err != nil {
		if err == db.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"external_id":   user.ExternalID,
		"provider":      user.Provider,
		"email":         user.Email,
		"name":          user.Name,
		"groups":        user.Groups,
		"is_admin":      user.IsAdmin,
		"is_active":     user.IsActive,
		"last_login_at": user.LastLoginAt,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	})
}

func (s *Server) handleGetUserAccessRules(c *gin.Context) {
	userID := c.Param("id")
	ctx := c.Request.Context()

	// First get the user to get their groups
	user, err := s.userStore.GetSSOUser(ctx, userID)
	if err != nil {
		if err == db.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Get access rules for this user (direct + via groups)
	rules, err := s.accessRuleStore.GetUserAccessRules(ctx, userID, user.Groups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user access rules"})
		return
	}

	response := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		response = append(response, gin.H{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"rule_type":   r.RuleType,
			"value":       r.Value,
			"port_range":  r.PortRange,
			"protocol":    r.Protocol,
			"network_id":  r.NetworkID,
			"is_active":   r.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{"access_rules": response})
}

func (s *Server) handleGetUserGateways(c *gin.Context) {
	userID := c.Param("id")
	ctx := c.Request.Context()

	gateways, err := s.gatewayStore.GetGatewaysForUser(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user gateways"})
		return
	}

	response := make([]gin.H, 0, len(gateways))
	for _, g := range gateways {
		response = append(response, gin.H{
			"id":             g.ID,
			"name":           g.Name,
			"hostname":       g.Hostname,
			"public_ip":      g.PublicIP,
			"vpn_port":       g.VPNPort,
			"vpn_protocol":   g.VPNProtocol,
			"is_active":      g.IsActive,
			"last_heartbeat": g.LastHeartbeat,
		})
	}

	c.JSON(http.StatusOK, gin.H{"gateways": response})
}

func (s *Server) handleAssignUserGateway(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		GatewayID string `json:"gateway_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.gatewayStore.AssignUserToGateway(ctx, userID, req.GatewayID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign gateway"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway assigned successfully"})
}

func (s *Server) handleRemoveUserGateway(c *gin.Context) {
	userID := c.Param("id")
	gatewayID := c.Param("gatewayId")
	ctx := c.Request.Context()

	if err := s.gatewayStore.RemoveUserFromGateway(ctx, userID, gatewayID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove gateway"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway removed successfully"})
}

func (s *Server) handleListLocalUsers(c *gin.Context) {
	ctx := c.Request.Context()
	users, err := s.userStore.ListLocalUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list local users"})
		return
	}

	response := make([]gin.H, 0, len(users))
	for _, u := range users {
		response = append(response, gin.H{
			"id":            u.ID,
			"username":      u.Username,
			"email":         u.Email,
			"is_admin":      u.IsAdmin,
			"last_login_at": u.LastLoginAt,
			"created_at":    u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

func (s *Server) handleCreateLocalUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := s.userStore.CreateUser(ctx, req.Username, req.Password, req.Email, req.IsAdmin); err != nil {
		if err == db.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created successfully"})
}

func (s *Server) handleDeleteLocalUser(c *gin.Context) {
	userID := c.Param("id")
	ctx := c.Request.Context()

	// Get the user first to check if it's the admin account
	user, err := s.userStore.GetUserByID(ctx, userID)
	if err != nil {
		if err == db.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Prevent deletion of the default admin account
	if user.Username == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete the default admin account"})
		return
	}

	if err := s.userStore.DeleteLocalUser(ctx, userID); err != nil {
		if err == db.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// Group management handlers

func (s *Server) handleListGroups(c *gin.Context) {
	ctx := c.Request.Context()
	groups, err := s.userStore.ListAllGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}

	// For each group, get member count
	response := make([]gin.H, 0, len(groups))
	for _, g := range groups {
		members, _ := s.userStore.GetGroupMembers(ctx, g)
		response = append(response, gin.H{
			"name":         g,
			"member_count": len(members),
		})
	}

	c.JSON(http.StatusOK, gin.H{"groups": response})
}

func (s *Server) handleGetGroupMembers(c *gin.Context) {
	groupName := c.Param("name")
	ctx := c.Request.Context()

	members, err := s.userStore.GetGroupMembers(ctx, groupName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get group members"})
		return
	}

	response := make([]gin.H, 0, len(members))
	for _, u := range members {
		response = append(response, gin.H{
			"id":       u.ID,
			"email":    u.Email,
			"name":     u.Name,
			"provider": u.Provider,
		})
	}

	c.JSON(http.StatusOK, gin.H{"members": response, "group": groupName})
}

func (s *Server) handleGetGroupAccessRules(c *gin.Context) {
	groupName := c.Param("name")
	ctx := c.Request.Context()

	// Get all access rules that are assigned to this group
	allRules, err := s.accessRuleStore.ListAccessRules(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list access rules"})
		return
	}

	// Filter rules that have this group assigned
	var groupRules []*db.AccessRule
	for _, rule := range allRules {
		groups, err := s.accessRuleStore.GetRuleGroups(ctx, rule.ID)
		if err != nil {
			continue
		}
		for _, g := range groups {
			if g == groupName {
				groupRules = append(groupRules, rule)
				break
			}
		}
	}

	response := make([]gin.H, 0, len(groupRules))
	for _, r := range groupRules {
		response = append(response, gin.H{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"rule_type":   r.RuleType,
			"value":       r.Value,
			"port_range":  r.PortRange,
			"protocol":    r.Protocol,
			"network_id":  r.NetworkID,
			"is_active":   r.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{"access_rules": response, "group": groupName})
}

// CA management handlers

func (s *Server) handleGetCA(c *gin.Context) {
	if s.ca == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA not initialized"})
		return
	}

	cert := s.ca.Certificate()
	if cert == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA certificate not available"})
		return
	}

	// Return CA info (but NOT the private key)
	c.JSON(http.StatusOK, gin.H{
		"serial_number": cert.SerialNumber.Text(16),
		"subject":       cert.Subject.String(),
		"issuer":        cert.Issuer.String(),
		"not_before":    cert.NotBefore,
		"not_after":     cert.NotAfter,
		"is_ca":         cert.IsCA,
		"fingerprint":   pki.Fingerprint(cert),
		"certificate":   string(s.ca.CertificatePEM()),
	})
}

func (s *Server) handleRotateCA(c *gin.Context) {
	if s.ca == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA not initialized"})
		return
	}

	ctx := c.Request.Context()

	// Generate new CA
	if err := s.ca.Rotate(ctx); err != nil {
		s.logger.Error("Failed to rotate CA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate CA: " + err.Error()})
		return
	}

	// Update config generator with new CA
	if s.configGen != nil {
		newConfigGen, err := openvpn.NewConfigGenerator(s.ca, nil)
		if err != nil {
			s.logger.Error("Failed to reinitialize config generator", zap.Error(err))
		} else {
			s.configGen = newConfigGen
		}
	}

	cert := s.ca.Certificate()
	s.logger.Info("CA rotated successfully",
		zap.String("serial", cert.SerialNumber.Text(16)),
		zap.Time("not_after", cert.NotAfter))

	c.JSON(http.StatusOK, gin.H{
		"message":       "CA rotated successfully",
		"serial_number": cert.SerialNumber.Text(16),
		"not_before":    cert.NotBefore,
		"not_after":     cert.NotAfter,
		"fingerprint":   pki.Fingerprint(cert),
	})
}

func (s *Server) handleUpdateCA(c *gin.Context) {
	if s.ca == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA not initialized"})
		return
	}

	var req struct {
		Certificate string `json:"certificate" binding:"required"`
		PrivateKey  string `json:"private_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: certificate and private_key required"})
		return
	}

	ctx := c.Request.Context()

	// Update CA with custom cert/key
	if err := s.ca.UpdateFromPEM(ctx, req.Certificate, req.PrivateKey); err != nil {
		s.logger.Error("Failed to update CA", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to update CA: " + err.Error()})
		return
	}

	// Update config generator with new CA
	if s.configGen != nil {
		newConfigGen, err := openvpn.NewConfigGenerator(s.ca, nil)
		if err != nil {
			s.logger.Error("Failed to reinitialize config generator", zap.Error(err))
		} else {
			s.configGen = newConfigGen
		}
	}

	cert := s.ca.Certificate()
	s.logger.Info("CA updated with custom certificate",
		zap.String("serial", cert.SerialNumber.Text(16)),
		zap.Time("not_after", cert.NotAfter))

	c.JSON(http.StatusOK, gin.H{
		"message":       "CA updated successfully",
		"serial_number": cert.SerialNumber.Text(16),
		"subject":       cert.Subject.String(),
		"not_before":    cert.NotBefore,
		"not_after":     cert.NotAfter,
		"fingerprint":   pki.Fingerprint(cert),
	})
}

// handleListCAs returns all CAs with their statuses
func (s *Server) handleListCAs(c *gin.Context) {
	ctx := c.Request.Context()

	cas, err := s.pkiStore.ListCAs(ctx)
	if err != nil {
		s.logger.Error("Failed to list CAs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list CAs"})
		return
	}

	result := make([]gin.H, len(cas))
	for i, ca := range cas {
		result[i] = gin.H{
			"id":            ca.ID,
			"status":        ca.Status,
			"serial_number": ca.SerialNumber,
			"not_before":    ca.NotBefore,
			"not_after":     ca.NotAfter,
			"fingerprint":   ca.Fingerprint,
			"description":   ca.Description,
			"created_at":    ca.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"cas": result})
}

// handlePrepareCARotation generates a new pending CA for rotation
func (s *Server) handlePrepareCARotation(c *gin.Context) {
	if s.ca == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA not initialized"})
		return
	}

	var req struct {
		Description string `json:"description"`
	}
	_ = c.ShouldBindJSON(&req) // Optional, description can be empty

	ctx := c.Request.Context()

	// Generate a new CA certificate
	newCA, err := pki.NewCA(s.config.PKI)
	if err != nil {
		s.logger.Error("Failed to generate new CA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate new CA"})
		return
	}

	// Generate unique ID for the new CA
	newCAID := fmt.Sprintf("ca-%d", time.Now().Unix())

	// Save as pending
	storedCA := &db.StoredCA{
		ID:             newCAID,
		CertificatePEM: string(newCA.CertificatePEM()),
		PrivateKeyPEM:  string(newCA.PrivateKeyPEM()),
		SerialNumber:   newCA.Certificate().SerialNumber.String(),
		NotBefore:      newCA.Certificate().NotBefore,
		NotAfter:       newCA.Certificate().NotAfter,
		Status:         db.CAStatusPending,
		Description:    req.Description,
	}

	if err := s.pkiStore.SaveCAWithID(ctx, storedCA); err != nil {
		s.logger.Error("Failed to save pending CA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pending CA"})
		return
	}

	// Record rotation event
	oldFingerprint, _ := s.pkiStore.GetCAFingerprint(ctx)
	event := &db.CARotationEvent{
		CAID:           newCAID,
		EventType:      "initiated",
		OldFingerprint: oldFingerprint,
		NewFingerprint: pki.Fingerprint(newCA.Certificate()),
		Notes:          "CA rotation prepared",
	}
	_ = s.pkiStore.RecordRotationEvent(ctx, event) // Best effort

	s.logger.Info("Pending CA prepared for rotation",
		zap.String("id", newCAID),
		zap.String("fingerprint", pki.Fingerprint(newCA.Certificate())))

	c.JSON(http.StatusOK, gin.H{
		"message":       "Pending CA prepared for rotation",
		"id":            newCAID,
		"status":        "pending",
		"serial_number": newCA.Certificate().SerialNumber.Text(16),
		"not_before":    newCA.Certificate().NotBefore,
		"not_after":     newCA.Certificate().NotAfter,
		"fingerprint":   pki.Fingerprint(newCA.Certificate()),
		"next_steps": []string{
			"1. Wait for all gateways/hubs/spokes to download the new CA via heartbeat",
			"2. Call POST /api/v1/admin/settings/ca/activate/" + newCAID + " to complete rotation",
			"3. Old CA will be retired (still trusted) for a grace period",
		},
	})
}

// handleActivateCA activates a pending CA and retires the current active CA
func (s *Server) handleActivateCA(c *gin.Context) {
	caID := c.Param("id")
	if caID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CA ID required"})
		return
	}

	ctx := c.Request.Context()

	// Verify the CA exists and is pending
	pendingCA, err := s.pkiStore.GetCAByID(ctx, caID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA not found"})
		return
	}
	if pendingCA.Status != db.CAStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CA is not in pending status"})
		return
	}

	// Activate the CA (this also retires the current active CA)
	if err := s.pkiStore.ActivateCA(ctx, caID); err != nil {
		s.logger.Error("Failed to activate CA", zap.Error(err), zap.String("ca_id", caID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate CA"})
		return
	}

	// Reload the CA in memory
	if err := s.reloadActiveCA(ctx); err != nil {
		s.logger.Error("Failed to reload CA in memory", zap.Error(err))
		// Don't fail the request - the database is updated
	}

	s.logger.Info("CA activated", zap.String("ca_id", caID))

	c.JSON(http.StatusOK, gin.H{
		"message":     "CA activated successfully",
		"id":          caID,
		"status":      "active",
		"fingerprint": pendingCA.Fingerprint,
		"note":        "Previous CA has been retired but remains trusted. Gateways/Hubs/Spokes will reprovision automatically.",
	})
}

// handleRevokeCA revokes a retired CA (removes from trust)
func (s *Server) handleRevokeCA(c *gin.Context) {
	caID := c.Param("id")
	if caID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CA ID required"})
		return
	}

	ctx := c.Request.Context()

	// Verify the CA exists and is retired
	ca, err := s.pkiStore.GetCAByID(ctx, caID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA not found"})
		return
	}
	if ca.Status == db.CAStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot revoke active CA - activate a new CA first"})
		return
	}

	if err := s.pkiStore.RevokeCA(ctx, caID); err != nil {
		s.logger.Error("Failed to revoke CA", zap.Error(err), zap.String("ca_id", caID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke CA"})
		return
	}

	s.logger.Info("CA revoked", zap.String("ca_id", caID))

	c.JSON(http.StatusOK, gin.H{
		"message": "CA revoked successfully",
		"id":      caID,
		"status":  "revoked",
		"warning": "Components still using this CA will no longer be able to connect",
	})
}

// handleGetCAFingerprint returns the fingerprint of the active CA
func (s *Server) handleGetCAFingerprint(c *gin.Context) {
	ctx := c.Request.Context()

	fingerprint, err := s.pkiStore.GetCAFingerprint(ctx)
	if err != nil {
		// Fallback to in-memory CA
		if s.ca != nil && s.ca.Certificate() != nil {
			fingerprint = pki.Fingerprint(s.ca.Certificate())
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CA not available"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fingerprint": fingerprint,
	})
}

// reloadActiveCA reloads the active CA from database into memory
func (s *Server) reloadActiveCA(ctx context.Context) error {
	storedCA, err := s.pkiStore.GetCA(ctx)
	if err != nil {
		return err
	}

	// Update the CA in memory
	if err := s.ca.UpdateFromPEM(ctx, storedCA.CertificatePEM, storedCA.PrivateKeyPEM); err != nil {
		return err
	}

	// Reinitialize config generator
	if s.configGen != nil {
		newConfigGen, err := openvpn.NewConfigGenerator(s.ca, nil)
		if err != nil {
			s.logger.Error("Failed to reinitialize config generator", zap.Error(err))
		} else {
			s.configGen = newConfigGen
		}
	}

	return nil
}

// validateCryptoProfileAllowed checks if the given crypto profile is allowed by system settings
func (s *Server) validateCryptoProfileAllowed(ctx context.Context, profile string) error {
	// Get allowed profiles from settings
	setting, err := s.settingsStore.Get(ctx, db.SettingAllowedCryptoProfiles)
	if err != nil {
		// If setting doesn't exist, allow all profiles (default behavior)
		return nil
	}

	allowedProfiles := strings.Split(setting.Value, ",")
	for _, p := range allowedProfiles {
		if strings.TrimSpace(p) == profile {
			return nil
		}
	}

	return fmt.Errorf("crypto profile '%s' is not allowed by system policy. Allowed profiles: %s", profile, setting.Value)
}

// Login Log handlers

func (s *Server) handleListLoginLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse filter parameters
	filter := &db.LoginLogFilter{
		UserEmail: c.Query("email"),
		UserID:    c.Query("user_id"),
		IPAddress: c.Query("ip"),
		Provider:  c.Query("provider"),
		Limit:     50,
		Offset:    0,
	}

	// Parse success filter
	if successStr := c.Query("success"); successStr != "" {
		success := successStr == "true"
		filter.Success = &success
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Parse time filters
	if startStr := c.Query("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &start
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &end
		}
	}

	logs, total, err := s.loginLogStore.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list login logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list login logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (s *Server) handleGetLoginLogStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Default to 30 days if not specified
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	stats, err := s.loginLogStore.GetStats(ctx, days)
	if err != nil {
		s.logger.Error("Failed to get login log stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get login log stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) handlePurgeLoginLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse days parameter (required)
	daysStr := c.Query("days")
	if daysStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "days parameter is required"})
		return
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days parameter"})
		return
	}

	deleted, err := s.loginLogStore.DeleteOlderThan(ctx, days)
	if err != nil {
		s.logger.Error("Failed to purge login logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to purge login logs"})
		return
	}

	s.logger.Info("Purged login logs", zap.Int("days", days), zap.Int64("deleted", deleted))
	c.JSON(http.StatusOK, gin.H{
		"message": "login logs purged",
		"deleted": deleted,
	})
}

func (s *Server) handleGetLoginLogRetention(c *gin.Context) {
	ctx := c.Request.Context()

	days := s.settingsStore.GetInt(ctx, db.SettingLoginLogRetentionDays, 30)

	c.JSON(http.StatusOK, gin.H{
		"retention_days": days,
	})
}

func (s *Server) handleSetLoginLogRetention(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate retention days (0 = forever, otherwise 1-365)
	if req.RetentionDays < 0 || req.RetentionDays > 365 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "retention_days must be between 0 and 365 (0 = forever)"})
		return
	}

	if err := s.settingsStore.Set(ctx, db.SettingLoginLogRetentionDays, strconv.Itoa(req.RetentionDays)); err != nil {
		s.logger.Error("Failed to set login log retention", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set retention"})
		return
	}

	s.logger.Info("Login log retention updated", zap.Int("days", req.RetentionDays))
	c.JSON(http.StatusOK, gin.H{
		"message":        "retention setting updated",
		"retention_days": req.RetentionDays,
	})
}

// geoIPResult holds the result from IP geolocation lookup
type geoIPResult struct {
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
}

// lookupGeoIP performs a geolocation lookup for an IP address using ip-api.com
// Returns country, countryCode, and city, or empty strings on error
func lookupGeoIP(ip string) (country, countryCode, city string) {
	// Skip lookup for private/local IPs
	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.2") || strings.HasPrefix(ip, "172.30.") ||
		strings.HasPrefix(ip, "172.31.") || strings.HasPrefix(ip, "127.") ||
		ip == "::1" || ip == "localhost" {
		return "", "", ""
	}

	// Use ip-api.com (free, no API key required, 45 requests/minute limit)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=country,countryCode,city")
	if err != nil {
		return "", "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", ""
	}

	var result geoIPResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", ""
	}

	return result.Country, result.CountryCode, result.City
}

// getRealClientIP extracts the real client IP from headers or falls back to c.ClientIP()
// This handles cases where requests come through load balancers, ingress controllers, or proxies
func getRealClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (standard for proxies, may contain multiple IPs)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// The first one is the original client IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	// Check X-Real-IP header (used by nginx)
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := c.GetHeader("CF-Connecting-IP"); cfip != "" {
		return strings.TrimSpace(cfip)
	}

	// Check True-Client-IP (Akamai, Cloudflare Enterprise)
	if tcip := c.GetHeader("True-Client-IP"); tcip != "" {
		return strings.TrimSpace(tcip)
	}

	// Fall back to Gin's ClientIP which respects trusted proxies
	return c.ClientIP()
}

// logUserLogin creates a login log entry (helper for auth handlers)
func (s *Server) logUserLogin(ctx context.Context, userID, userEmail, userName, provider, providerName, ipAddress, userAgent, sessionID string, success bool, failureReason string) {
	// Look up geolocation for the IP address
	country, countryCode, city := lookupGeoIP(ipAddress)

	log := &db.LoginLog{
		UserID:        userID,
		UserEmail:     userEmail,
		UserName:      userName,
		Provider:      provider,
		ProviderName:  providerName,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Country:       country,
		CountryCode:   countryCode,
		City:          city,
		Success:       success,
		FailureReason: failureReason,
		SessionID:     sessionID,
	}

	if err := s.loginLogStore.Create(ctx, log); err != nil {
		s.logger.Error("Failed to create login log", zap.Error(err), zap.String("user_email", userEmail))
	}
}
