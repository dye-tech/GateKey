// Package api implements the HTTP API server for GateKey.
package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/k8s"
	"github.com/gatekey-project/gatekey/internal/metrics"
	"github.com/gatekey-project/gatekey/internal/openvpn"
	"github.com/gatekey-project/gatekey/internal/pki"
	"github.com/gatekey-project/gatekey/internal/session"
	"github.com/gatekey-project/gatekey/internal/telemetry"
)

// Server represents the HTTP API server.
type Server struct {
	config          *config.Config
	logger          *zap.Logger
	router          *gin.Engine
	httpServer      *http.Server
	db              *db.DB
	userStore       *db.UserStore
	providerStore   *db.ProviderStore
	stateStore      *db.StateStore
	configStore     *db.ConfigStore
	gatewayStore    *db.GatewayStore
	networkStore    *db.NetworkStore
	accessRuleStore *db.AccessRuleStore
	settingsStore   *db.SettingsStore
	pkiStore        *db.PKIStore
	proxyAppStore   *db.ProxyApplicationStore
	loginLogStore   *db.LoginLogStore
	meshStore       *db.MeshStore
	meshConfigStore *db.MeshConfigStore
	apiKeyStore     *db.APIKeyStore
	localGroupStore *db.LocalGroupStore
	geoFenceStore   *db.GeoFenceStore
	wgConfigStore   *db.WireGuardConfigStore // WireGuard config store
	wgPeerStore     *db.WireGuardPeerStore   // WireGuard peer store
	jitStore        *db.JITAccessStore       // JIT access store
	ca              *pki.CA
	configGen       *openvpn.ConfigGenerator
	adminPassword   string              // Initial admin password (shown once at startup)
	bgCancel        context.CancelFunc  // Cancel function for background tasks
	sessionMgr      *session.Manager    // Remote session manager
	authRateLimiter *rateLimiter        // Rate limiter for auth endpoints
	telemetry       *telemetry.Provider // OpenTelemetry provider
}

// NewServer creates a new API server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(zapLogger(logger))
	// Request body size limit (10MB default) to prevent DoS attacks
	router.Use(requestBodyLimitMiddleware(10 * 1024 * 1024))

	// Add Prometheus metrics middleware
	if cfg.Metrics.Enabled {
		router.Use(metrics.GinMiddleware())
	}

	// Initialize OpenTelemetry tracing
	var telemetryProvider *telemetry.Provider
	if cfg.Telemetry.Enabled {
		telemetryCfg := telemetry.Config{
			Enabled:        cfg.Telemetry.Enabled,
			ServiceName:    cfg.Telemetry.ServiceName,
			ServiceVersion: "1.0.0",
			Environment:    cfg.Telemetry.Environment,
			OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
			OTLPProtocol:   cfg.Telemetry.OTLPProtocol,
			OTLPInsecure:   cfg.Telemetry.OTLPInsecure,
			SampleRate:     cfg.Telemetry.SampleRate,
		}
		var err error
		telemetryProvider, err = telemetry.NewProvider(telemetryCfg)
		if err != nil {
			logger.Warn("Failed to initialize OpenTelemetry tracing", zap.Error(err))
		} else {
			logger.Info("OpenTelemetry tracing enabled",
				zap.String("endpoint", cfg.Telemetry.OTLPEndpoint),
				zap.String("service_name", cfg.Telemetry.ServiceName),
				zap.Float64("sample_rate", cfg.Telemetry.SampleRate))
			// Add tracing middleware
			serviceName := cfg.Telemetry.ServiceName
			if serviceName == "" {
				serviceName = "gatekey"
			}
			router.Use(telemetry.GinMiddleware(serviceName))
		}
	}

	// Configure CORS with exception for SAML ACS endpoint
	// SAML ACS receives form POSTs from IdPs (not XHR/fetch) so doesn't need CORS headers.
	// The security comes from cryptographic signature validation, not browser origin checks.
	if len(cfg.Server.CORSOrigins) > 0 {
		corsMiddleware := cors.New(cors.Config{
			AllowOrigins:     cfg.Server.CORSOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		})
		router.Use(func(c *gin.Context) {
			// SAML ACS uses browser form POST (not XHR), so no CORS headers needed.
			// Adding CORS headers would only enable JavaScript from any origin to POST,
			// which is unnecessary and potentially a security concern.
			// SAML security is based on cryptographic signature validation of the assertion.
			if c.Request.URL.Path == "/api/v1/auth/saml/acs" {
				// Only allow POST (SAML assertion) and OPTIONS (preflight, though unlikely)
				if c.Request.Method == "OPTIONS" {
					// Minimal preflight response - don't allow arbitrary origins
					c.AbortWithStatus(204)
					return
				}
				// For POST, proceed without adding CORS headers
				// The SAML handler validates the assertion cryptographically
				c.Next()
				return
			}
			// Apply normal CORS for all other endpoints
			corsMiddleware(c)
		})
	}

	// Configure trusted proxies
	if len(cfg.Server.TrustedProxies) > 0 {
		_ = router.SetTrustedProxies(cfg.Server.TrustedProxies) // Ignore error, will use defaults
	}

	// Add security headers middleware (multi-tenant aware)
	if cfg.Security.SecurityHeadersEnabled {
		router.Use(securityHeadersMiddleware(cfg))
	}

	// Initialize database connection with TLS configuration
	ctx := context.Background()
	tlsCfg := &db.TLSConfig{
		SSLMode:     cfg.Database.SSLMode,
		SSLRootCert: cfg.Database.SSLRootCert,
		SSLCert:     cfg.Database.SSLCert,
		SSLKey:      cfg.Database.SSLKey,
	}
	database, err := db.NewWithTLS(ctx, cfg.Database.URL, tlsCfg)
	if err != nil {
		return nil, err
	}

	// Security warnings for TLS configuration
	switch cfg.Database.SSLMode {
	case "disable":
		logger.Warn("SECURITY WARNING: Database SSL is DISABLED - connections are unencrypted. " +
			"This is a critical security risk in production. Set database.ssl_mode to 'verify-full' or 'verify-ca'")
	case "require":
		logger.Warn("Database SSL mode is 'require' - connections are encrypted but server certificate is NOT verified. " +
			"For production, use 'verify-full' to prevent MITM attacks")
	case "verify-ca":
		logger.Info("Database SSL mode is 'verify-ca' - server certificate verified against CA. " +
			"Consider 'verify-full' for additional hostname verification")
	case "verify-full":
		logger.Info("Database SSL configured with full verification (recommended)")
	}

	// Warn if proxy TLS verification is disabled
	if !cfg.Security.ProxyTLSVerify {
		logger.Warn("SECURITY WARNING: Proxy TLS verification is DISABLED - proxy targets are not verified. " +
			"This allows MITM attacks on proxied connections. Set security.proxy_tls_verify to true for production")
	}

	// Warn about deprecated RSA-2048 key algorithm
	if cfg.PKI.KeyAlgorithm == "rsa2048" {
		logger.Warn("DEPRECATION WARNING: RSA-2048 key algorithm is deprecated and will be removed in a future version. " +
			"RSA-2048 provides only 112 bits of security. Migrate to ecdsa256 (128 bits), ecdsa384 (192 bits), or rsa4096 (140 bits)")
	}

	// Initialize stores
	userStore := db.NewUserStore(database)
	providerStore := db.NewProviderStore(database)
	stateStore := db.NewStateStore(database)
	configStore := db.NewConfigStore(database)
	gatewayStore := db.NewGatewayStore(database)
	networkStore := db.NewNetworkStore(database)
	accessRuleStore := db.NewAccessRuleStore(database)
	settingsStore := db.NewSettingsStore(database)

	// Create PKI store with encryption - key is auto-generated and persisted if not provided
	// CA private keys are encrypted at rest using AES-256-GCM
	var pkiStore *db.PKIStore
	encryptionKey := cfg.Security.CAKeyEncryptionKey

	// If no key in config, check database settings
	if encryptionKey == "" {
		storedKey := settingsStore.GetString(ctx, db.SettingCAKeyEncryptionKey, "")
		if storedKey != "" {
			encryptionKey = storedKey
			logger.Info("Using CA key encryption key from database settings")
		}
	}

	// If still no key, auto-generate and persist to database
	if encryptionKey == "" {
		keyBytes := make([]byte, 32) // 256-bit key for AES-256-GCM
		if _, err := rand.Read(keyBytes); err != nil {
			return nil, fmt.Errorf("failed to generate CA encryption key: %w", err)
		}
		encryptionKey = base64.StdEncoding.EncodeToString(keyBytes)

		// Persist to database for cluster-wide consistency
		if err := settingsStore.Set(ctx, db.SettingCAKeyEncryptionKey, encryptionKey); err != nil {
			return nil, fmt.Errorf("failed to persist CA encryption key: %w", err)
		}
		logger.Info("Generated and stored new CA key encryption key in database")
	}

	// Initialize PKI store with encryption
	pkiStore, err = db.NewPKIStoreWithEncryption(database, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PKI store with encryption: %w", err)
	}
	logger.Info("PKI store initialized with CA private key encryption (AES-256-GCM)")
	proxyAppStore := db.NewProxyApplicationStore(database)
	loginLogStore := db.NewLoginLogStore(database)
	meshStore := db.NewMeshStore(database)
	meshConfigStore := db.NewMeshConfigStore(database)
	apiKeyStore := db.NewAPIKeyStore(database)
	localGroupStore := db.NewLocalGroupStore(database)
	geoFenceStore := db.NewGeoFenceStore(database)
	wgConfigStore := db.NewWireGuardConfigStore(database)
	wgPeerStore := db.NewWireGuardPeerStore(database)
	jitStore := db.NewJITAccessStore(database)

	// Initialize PKI with database store for CA persistence
	// This ensures all pods share the same CA
	ca, err := pki.NewCAWithStore(cfg.PKI, &pkiStoreAdapter{pkiStore})
	if err != nil {
		logger.Warn("Failed to initialize CA, config generation will be unavailable", zap.Error(err))
	}

	// Initialize OpenVPN config generator
	var configGen *openvpn.ConfigGenerator
	if ca != nil {
		configGen, err = openvpn.NewConfigGenerator(ca, nil)
		if err != nil {
			logger.Warn("Failed to initialize config generator", zap.Error(err))
		}
	}

	// Create default admin if no users exist
	adminPassword, created, err := userStore.InitDefaultAdmin(ctx)
	if err != nil {
		logger.Warn("Failed to check/create default admin", zap.Error(err))
	}

	srv := &Server{
		config:          cfg,
		logger:          logger,
		router:          router,
		db:              database,
		userStore:       userStore,
		providerStore:   providerStore,
		stateStore:      stateStore,
		configStore:     configStore,
		gatewayStore:    gatewayStore,
		networkStore:    networkStore,
		accessRuleStore: accessRuleStore,
		settingsStore:   settingsStore,
		pkiStore:        pkiStore,
		proxyAppStore:   proxyAppStore,
		loginLogStore:   loginLogStore,
		meshStore:       meshStore,
		meshConfigStore: meshConfigStore,
		apiKeyStore:     apiKeyStore,
		localGroupStore: localGroupStore,
		geoFenceStore:   geoFenceStore,
		wgConfigStore:   wgConfigStore,
		wgPeerStore:     wgPeerStore,
		jitStore:        jitStore,
		ca:              ca,
		configGen:       configGen,
		adminPassword:   adminPassword,
		telemetry:       telemetryProvider,
	}

	// Save admin password to Kubernetes secret if created
	if created {
		logger.Info("==============================================")
		logger.Info("Initial admin account created")
		logger.Info("Username: admin")

		// Try to save to Kubernetes secret
		secretMgr, err := k8s.NewSecretManager()
		if err != nil {
			logger.Warn("Failed to initialize Kubernetes secret manager", zap.Error(err))
		}

		if secretMgr != nil {
			if err := secretMgr.SaveAdminPassword(ctx, adminPassword); err != nil {
				logger.Warn("Failed to save admin password to Kubernetes secret", zap.Error(err))
				logger.Info("Admin password (save this, it won't be shown again): " + adminPassword)
			} else {
				logger.Info("Admin password saved to Kubernetes secret: " + k8s.AdminPasswordSecretName)
				logger.Info("Retrieve with: kubectl get secret " + k8s.AdminPasswordSecretName + " -o jsonpath='{.data.admin-password}' | base64 -d")
			}
		} else {
			// Not running in Kubernetes, log the password
			logger.Info("Admin password (save this, it won't be shown again): " + adminPassword)
		}

		logger.Info("Please change this password after first login!")
		logger.Info("==============================================")
	}

	// Initialize session manager for remote sessions
	srv.sessionMgr = session.NewManager(logger)
	srv.sessionMgr.ValidateAgentToken = srv.validateAgentToken
	// Configure allowed origins for admin WebSocket connections
	if len(cfg.Server.CORSOrigins) > 0 {
		srv.sessionMgr.SetAllowedOrigins(cfg.Server.CORSOrigins)
	}

	// Create background context for all background tasks
	bgCtx, bgCancel := context.WithCancel(context.Background())
	srv.bgCancel = bgCancel

	// Initialize rate limiter for auth endpoints (50 attempts per minute per IP)
	srv.authRateLimiter = newRateLimiter(bgCtx, 50, time.Minute)

	// Setup routes
	srv.setupRoutes()

	// Start background tasks
	go srv.runGatewayHealthCheck(bgCtx)
	go srv.runConfigCleanup(bgCtx)
	go srv.runLoginLogCleanup(bgCtx)
	go srv.runMeshHealthCheck(bgCtx)
	go srv.runJITCleanup(bgCtx)

	return srv, nil
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health check endpoints for Kubernetes and monitoring
	s.router.GET("/health", s.healthCheck) // Comprehensive health with dependency checks
	s.router.GET("/ready", s.readyCheck)   // Kubernetes readiness probe
	s.router.GET("/live", s.liveCheck)     // Kubernetes liveness probe

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	v1.Use(s.csrfMiddleware())
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			// OIDC - rate limited to prevent abuse
			auth.GET("/oidc/login", s.authRateLimitMiddleware(), s.handleOIDCLogin)
			auth.GET("/oidc/callback", s.authRateLimitMiddleware(), s.handleOIDCCallback)

			// SAML - rate limited to prevent abuse
			auth.GET("/saml/login", s.authRateLimitMiddleware(), s.handleSAMLLogin)
			auth.POST("/saml/acs", s.authRateLimitMiddleware(), s.handleSAMLACS)
			auth.OPTIONS("/saml/acs", s.handleSAMLACSOptions) // CORS preflight for IdP POST
			auth.GET("/saml/metadata", s.handleSAMLMetadata)  // Metadata doesn't need rate limiting

			// CLI authentication (browser-based flow for CLI client) - rate limited
			auth.GET("/cli/login", s.authRateLimitMiddleware(), s.handleCLILogin)
			auth.GET("/cli/complete", s.handleCLIComplete)
			auth.GET("/cli/callback", s.handleCLICallback)
			auth.POST("/refresh", s.handleTokenRefresh)

			// Local authentication (for initial setup) - rate limited
			auth.POST("/local/login", s.authRateLimitMiddleware(), s.handleLocalLogin)
			auth.POST("/local/change-password", s.authRateLimitMiddleware(), s.handleChangePassword)

			// Session management
			auth.POST("/logout", s.handleLogout)
			auth.GET("/session", s.handleGetSession)
			auth.GET("/providers", s.handleGetProviders)
		}

		// Admin settings routes (requires admin auth)
		settings := v1.Group("/admin/settings")
		settings.Use(s.adminAuthMiddleware())
		{
			settings.GET("", s.handleGetSettings)
			settings.PUT("", s.handleUpdateSettings)
			settings.GET("/oidc", s.handleGetOIDCProvidersDynamic)
			settings.POST("/oidc", s.handleCreateOIDCProviderDynamic)
			settings.PUT("/oidc/:name", s.handleUpdateOIDCProviderDynamic)
			settings.DELETE("/oidc/:name", s.handleDeleteOIDCProviderDynamic)
			settings.GET("/saml", s.handleGetSAMLProvidersDynamic)
			settings.POST("/saml", s.handleCreateSAMLProviderDynamic)
			settings.PUT("/saml/:name", s.handleUpdateSAMLProviderDynamic)
			settings.DELETE("/saml/:name", s.handleDeleteSAMLProviderDynamic)
			// CA management
			settings.GET("/ca", s.handleGetCA)
			settings.POST("/ca/rotate", s.handleRotateCA)
			settings.PUT("/ca", s.handleUpdateCA)
			// Graceful CA rotation
			settings.GET("/ca/list", s.handleListCAs)
			settings.POST("/ca/prepare-rotation", s.handlePrepareCARotation)
			settings.POST("/ca/activate/:id", s.handleActivateCA)
			settings.POST("/ca/revoke/:id", s.handleRevokeCA)
			settings.GET("/ca/fingerprint", s.handleGetCAFingerprint)
		}

		// Config generation routes
		configs := v1.Group("/configs")
		{
			configs.GET("", s.handleListUserConfigs) // List user's configs
			configs.POST("/generate", s.handleGenerateConfig)
			configs.GET("/download/:id", s.handleDownloadConfig)
			configs.GET("/:id", s.handleGetConfigMetadata)    // Get config metadata (for CLI polling)
			configs.GET("/:id/raw", s.handleGetConfigRaw)     // Get raw config content (for CLI)
			configs.POST("/:id/revoke", s.handleRevokeConfig) // Revoke user's own config
		}

		// Gateway routes (internal - OpenVPN)
		gateway := v1.Group("/gateway")
		{
			gateway.POST("/verify", s.handleGatewayVerify)
			gateway.POST("/connect", s.handleGatewayConnect)
			gateway.POST("/disconnect", s.handleGatewayDisconnect)
			gateway.POST("/heartbeat", s.handleGatewayHeartbeat)
			gateway.POST("/provision", s.handleGatewayProvision)
			gateway.POST("/client-rules", s.handleGatewayClientRules)
			gateway.POST("/all-rules", s.handleGatewayAllRules)
			gateway.POST("/client-stats", s.handleGatewayClientStats)           // Periodic client stats sync
			gateway.POST("/pending-disconnects", s.handleGetPendingDisconnects) // Poll for pending user disconnects
			gateway.POST("/ack-disconnect", s.handleAckDisconnect)              // Acknowledge disconnect executed
		}

		// WireGuard Gateway routes (internal - WireGuard gateway agents)
		wgGateway := v1.Group("/wireguard-gateway")
		{
			wgGateway.POST("/provision", s.handleWireGuardGatewayProvision)
			wgGateway.POST("/heartbeat", s.handleWireGuardGatewayHeartbeat)
			wgGateway.POST("/sync-peers", s.handleWireGuardGatewaySyncPeers)
			wgGateway.POST("/peer-stats", s.handleWireGuardGatewayPeerStats)
		}

		// Mesh Hub internal routes (hub → control plane communication)
		meshHub := v1.Group("/mesh-hub")
		{
			meshHub.POST("/heartbeat", s.handleMeshHubHeartbeat)
			meshHub.POST("/provision", s.handleMeshHubProvisionRequest)
			meshHub.GET("/routes", s.handleMeshHubGetRoutes)
			meshHub.GET("/spokes", s.handleMeshHubGetSpokes)
			meshHub.POST("/spoke-connected", s.handleMeshSpokeConnected)
			meshHub.POST("/spoke-disconnected", s.handleMeshSpokeDisconnected)
			meshHub.POST("/client-connected", s.handleMeshClientConnected)
			meshHub.POST("/client-disconnected", s.handleMeshClientDisconnected)
			meshHub.POST("/client-rules", s.handleMeshClientRules)        // Get access rules for a client
			meshHub.POST("/all-client-rules", s.handleMeshAllClientRules) // Get access rules for all clients
			meshHub.POST("/client-stats", s.handleMeshClientStats)        // Periodic client stats sync
		}

		// Mesh Spoke internal routes (spoke → control plane for initial setup)
		meshSpoke := v1.Group("/mesh-spoke")
		{
			meshSpoke.POST("/provision", s.handleMeshSpokeProvisionRequest)
			meshSpoke.POST("/heartbeat", s.handleMeshSpokeHeartbeat)
		}

		// Mesh Gateway alias (binary uses mesh-gateway, routes to same handlers)
		meshGateway := v1.Group("/mesh-gateway")
		{
			meshGateway.POST("/provision", s.handleMeshSpokeProvisionRequest)
			meshGateway.POST("/heartbeat", s.handleMeshSpokeHeartbeat)
		}

		// WireGuard Mesh Hub internal routes (WG hub → control plane communication)
		wgMeshHub := v1.Group("/wg-mesh-hub")
		{
			wgMeshHub.POST("/provision", s.handleWireGuardMeshHubProvision)
			wgMeshHub.POST("/heartbeat", s.handleWireGuardMeshHubHeartbeat)
			wgMeshHub.POST("/sync-peers", s.handleWireGuardMeshHubSyncPeers)
			wgMeshHub.POST("/peer-stats", s.handleWireGuardMeshHubPeerStats)
		}

		// WireGuard Mesh Spoke internal routes (WG spoke → control plane)
		wgMeshSpoke := v1.Group("/wg-mesh-spoke")
		{
			wgMeshSpoke.POST("/provision", s.handleWireGuardMeshSpokeProvision)
			wgMeshSpoke.POST("/heartbeat", s.handleWireGuardMeshSpokeHeartbeat)
		}

		// Gateway listing for authenticated users
		v1.GET("/gateways", s.handleListUserGateways)

		// Server info for clients (includes FIPS requirements)
		v1.GET("/server/info", s.handleGetServerInfo)

		// Admin routes (requires admin auth)
		admin := v1.Group("/admin")
		admin.Use(s.adminAuthMiddleware())
		{
			// Gateway routes with scope enforcement
			gatewaysRead := admin.Group("")
			gatewaysRead.Use(s.requireAnyScope(ScopeGatewaysRead, ScopeGatewaysWrite, ScopeAdmin))
			{
				gatewaysRead.GET("/gateways", s.handleListGateways)
				gatewaysRead.GET("/gateways/:id/networks", s.handleGetGatewayNetworks)
				gatewaysRead.GET("/gateways/:id/users", s.handleGetGatewayUsers)
				gatewaysRead.GET("/gateways/:id/groups", s.handleGetGatewayGroups)
			}
			gatewaysWrite := admin.Group("")
			gatewaysWrite.Use(s.requireAnyScope(ScopeGatewaysWrite, ScopeAdmin))
			{
				gatewaysWrite.POST("/gateways", s.handleRegisterGateway)
				gatewaysWrite.PUT("/gateways/:id", s.handleUpdateGateway)
				gatewaysWrite.DELETE("/gateways/:id", s.handleDeleteGateway)
				gatewaysWrite.POST("/gateways/:id/rotate-token", s.handleRotateGatewayToken)
				gatewaysWrite.POST("/gateways/:id/reprovision", s.handleReprovisionGateway)
				gatewaysWrite.POST("/gateways/:id/networks", s.handleAssignGatewayNetwork)
				gatewaysWrite.DELETE("/gateways/:id/networks/:networkId", s.handleRemoveGatewayNetwork)
				gatewaysWrite.POST("/gateways/:id/users", s.handleAssignGatewayUser)
				gatewaysWrite.DELETE("/gateways/:id/users/:userId", s.handleRemoveGatewayUser)
				gatewaysWrite.POST("/gateways/:id/groups", s.handleAssignGatewayGroup)
				gatewaysWrite.DELETE("/gateways/:id/groups/:groupName", s.handleRemoveGatewayGroup)
			}
			// Monitoring routes (admin scope required)
			monitorRead := admin.Group("")
			monitorRead.Use(s.requireScope(ScopeAdmin))
			{
				monitorRead.GET("/connections", s.handleListConnections)
				monitorRead.GET("/audit", s.handleGetAuditLogs)
			}

			// Network management with scope enforcement
			configRead := admin.Group("")
			configRead.Use(s.requireAnyScope(ScopeConfigRead, ScopeConfigWrite, ScopeAdmin))
			{
				configRead.GET("/networks", s.handleListNetworks)
				configRead.GET("/networks/:id", s.handleGetNetwork)
				configRead.GET("/networks/:id/gateways", s.handleGetNetworkGateways)
				configRead.GET("/networks/:id/access-rules", s.handleGetNetworkAccessRules)
				configRead.GET("/networks/:id/mesh-hubs", s.handleGetNetworkMeshHubs)
				configRead.GET("/access-rules", s.handleListAccessRules)
				configRead.GET("/access-rules/:id", s.handleGetAccessRule)
			}
			configWrite := admin.Group("")
			configWrite.Use(s.requireAnyScope(ScopeConfigWrite, ScopeAdmin))
			{
				configWrite.POST("/networks", s.handleCreateNetwork)
				configWrite.PUT("/networks/:id", s.handleUpdateNetwork)
				configWrite.DELETE("/networks/:id", s.handleDeleteNetwork)
				configWrite.POST("/access-rules", s.handleCreateAccessRule)
				configWrite.PUT("/access-rules/:id", s.handleUpdateAccessRule)
				configWrite.DELETE("/access-rules/:id", s.handleDeleteAccessRule)
				configWrite.POST("/access-rules/:id/users", s.handleAssignRuleToUser)
				configWrite.DELETE("/access-rules/:id/users/:userId", s.handleRemoveRuleFromUser)
				configWrite.POST("/access-rules/:id/groups", s.handleAssignRuleToGroup)
				configWrite.DELETE("/access-rules/:id/groups/:groupName", s.handleRemoveRuleFromGroup)
				configWrite.POST("/configs/:id/revoke", s.handleAdminRevokeConfig)
			}

			// User management with scope enforcement
			usersRead := admin.Group("")
			usersRead.Use(s.requireAnyScope(ScopeUsersRead, ScopeUsersWrite, ScopeAdmin))
			{
				usersRead.GET("/users", s.handleListUsers)
				usersRead.GET("/users/:id", s.handleGetUser)
				usersRead.GET("/users/:id/access-rules", s.handleGetUserAccessRules)
				usersRead.GET("/users/:id/gateways", s.handleGetUserGateways)
				usersRead.GET("/users/:id/configs", s.handleAdminListUserConfigs)
				usersRead.GET("/users/:id/mesh-configs", s.handleAdminListUserMeshConfigs)
				usersRead.GET("/groups", s.handleListGroups)
				usersRead.GET("/groups/:name/members", s.handleGetGroupMembers)
				usersRead.GET("/groups/:name/access-rules", s.handleGetGroupAccessRules)
				usersRead.GET("/local-users", s.handleListLocalUsers)
				usersRead.GET("/local-groups", s.handleListLocalGroups)
				usersRead.GET("/local-groups/:id", s.handleGetLocalGroup)
				usersRead.GET("/local-groups/:id/members", s.handleListLocalGroupMembers)
			}
			usersWrite := admin.Group("")
			usersWrite.Use(s.requireAnyScope(ScopeUsersWrite, ScopeAdmin))
			{
				usersWrite.POST("/users/:id/gateways", s.handleAssignUserGateway)
				usersWrite.DELETE("/users/:id/gateways/:gatewayId", s.handleRemoveUserGateway)
				usersWrite.POST("/users/:id/revoke-configs", s.handleAdminRevokeUserConfigs)
				usersWrite.POST("/local-users", s.handleCreateLocalUser)
				usersWrite.DELETE("/local-users/:id", s.handleDeleteLocalUser)
				usersWrite.POST("/local-groups", s.handleCreateLocalGroup)
				usersWrite.PUT("/local-groups/:id", s.handleUpdateLocalGroup)
				usersWrite.DELETE("/local-groups/:id", s.handleDeleteLocalGroup)
				usersWrite.POST("/local-groups/:id/members", s.handleAddLocalGroupMember)
				usersWrite.DELETE("/local-groups/:id/members/:userId/:memberType", s.handleRemoveLocalGroupMember)
			}

			// Proxy application management with scope enforcement
			proxyRead := admin.Group("")
			proxyRead.Use(s.requireAnyScope(ScopeConfigRead, ScopeConfigWrite, ScopeAdmin))
			{
				proxyRead.GET("/proxy-apps", s.handleListProxyApps)
				proxyRead.GET("/proxy-apps/:id", s.handleGetProxyApp)
				proxyRead.GET("/proxy-apps/:id/users", s.handleGetProxyAppUsers)
				proxyRead.GET("/proxy-apps/:id/groups", s.handleGetProxyAppGroups)
				proxyRead.GET("/proxy-apps/:id/logs", s.handleGetProxyAppLogs)
			}
			proxyWrite := admin.Group("")
			proxyWrite.Use(s.requireAnyScope(ScopeConfigWrite, ScopeAdmin))
			{
				proxyWrite.POST("/proxy-apps", s.handleCreateProxyApp)
				proxyWrite.PUT("/proxy-apps/:id", s.handleUpdateProxyApp)
				proxyWrite.DELETE("/proxy-apps/:id", s.handleDeleteProxyApp)
				proxyWrite.POST("/proxy-apps/:id/users", s.handleAssignProxyAppToUser)
				proxyWrite.DELETE("/proxy-apps/:id/users/:userId", s.handleRemoveProxyAppFromUser)
				proxyWrite.POST("/proxy-apps/:id/groups", s.handleAssignProxyAppToGroup)
				proxyWrite.DELETE("/proxy-apps/:id/groups/:groupName", s.handleRemoveProxyAppFromGroup)
			}

			// Login logs / monitoring (admin scope - sensitive data)
			logsRead := admin.Group("")
			logsRead.Use(s.requireScope(ScopeAdmin))
			{
				logsRead.GET("/login-logs", s.handleListLoginLogs)
				logsRead.GET("/login-logs/stats", s.handleGetLoginLogStats)
				logsRead.GET("/login-logs/retention", s.handleGetLoginLogRetention)
			}
			logsWrite := admin.Group("")
			logsWrite.Use(s.requireScope(ScopeAdmin))
			{
				logsWrite.DELETE("/login-logs", s.handlePurgeLoginLogs)
				logsWrite.PUT("/login-logs/retention", s.handleSetLoginLogRetention)
			}

			// Mesh Hub management with scope enforcement
			meshRead := admin.Group("")
			meshRead.Use(s.requireAnyScope(ScopeMeshRead, ScopeMeshWrite, ScopeAdmin))
			{
				meshRead.GET("/mesh/hubs", s.handleListMeshHubs)
				meshRead.GET("/mesh/hubs/:id", s.handleGetMeshHub)
				meshRead.GET("/mesh/hubs/:id/install-script", s.handleMeshHubInstallScript)
				meshRead.GET("/mesh/hubs/:id/users", s.handleGetMeshHubUsers)
				meshRead.GET("/mesh/hubs/:id/groups", s.handleGetMeshHubGroups)
				meshRead.GET("/mesh/hubs/:id/networks", s.handleGetMeshHubNetworks)
				meshRead.GET("/mesh/hubs/:id/spokes", s.handleListMeshSpokes)
				meshRead.GET("/mesh/spokes/:id", s.handleGetMeshSpoke)
				meshRead.GET("/mesh/spokes/:id/install-script", s.handleMeshSpokeInstallScript)
				meshRead.GET("/mesh/spokes/:id/users", s.handleGetMeshSpokeUsers)
				meshRead.GET("/mesh/spokes/:id/groups", s.handleGetMeshSpokeGroups)
			}
			meshWrite := admin.Group("")
			meshWrite.Use(s.requireAnyScope(ScopeMeshWrite, ScopeAdmin))
			{
				meshWrite.POST("/mesh/hubs", s.handleCreateMeshHub)
				meshWrite.PUT("/mesh/hubs/:id", s.handleUpdateMeshHub)
				meshWrite.DELETE("/mesh/hubs/:id", s.handleDeleteMeshHub)
				meshWrite.POST("/mesh/hubs/:id/rotate-token", s.handleRotateMeshHubToken)
				meshWrite.POST("/mesh/hubs/:id/provision", s.handleProvisionMeshHub)
				meshWrite.POST("/mesh/hubs/:id/users", s.handleAssignMeshHubUser)
				meshWrite.DELETE("/mesh/hubs/:id/users/:userId", s.handleRemoveMeshHubUser)
				meshWrite.POST("/mesh/hubs/:id/groups", s.handleAssignMeshHubGroup)
				meshWrite.DELETE("/mesh/hubs/:id/groups/:groupName", s.handleRemoveMeshHubGroup)
				meshWrite.POST("/mesh/hubs/:id/networks", s.handleAssignMeshHubNetwork)
				meshWrite.DELETE("/mesh/hubs/:id/networks/:networkId", s.handleRemoveMeshHubNetwork)
				meshWrite.POST("/mesh/hubs/:id/spokes", s.handleCreateMeshSpoke)
				meshWrite.PUT("/mesh/spokes/:id", s.handleUpdateMeshSpoke)
				meshWrite.DELETE("/mesh/spokes/:id", s.handleDeleteMeshSpoke)
				meshWrite.POST("/mesh/spokes/:id/rotate-token", s.handleRotateMeshSpokeToken)
				meshWrite.POST("/mesh/spokes/:id/provision", s.handleProvisionMeshSpoke)
				meshWrite.POST("/mesh/spokes/:id/users", s.handleAssignMeshSpokeUser)
				meshWrite.DELETE("/mesh/spokes/:id/users/:userId", s.handleRemoveMeshSpokeUser)
				meshWrite.POST("/mesh/spokes/:id/groups", s.handleAssignMeshSpokeGroup)
				meshWrite.DELETE("/mesh/spokes/:id/groups/:groupName", s.handleRemoveMeshSpokeGroup)
			}

			// Admin config management with scope enforcement
			adminConfigRead := admin.Group("")
			adminConfigRead.Use(s.requireAnyScope(ScopeConfigRead, ScopeConfigWrite, ScopeAdmin))
			{
				adminConfigRead.GET("/configs", s.handleAdminListAllConfigs)
				adminConfigRead.GET("/mesh-configs", s.handleAdminListMeshConfigs)
				adminConfigRead.GET("/wireguard/configs", s.handleAdminListWireGuardConfigs)
				adminConfigRead.GET("/wireguard/peers", s.handleAdminListWireGuardPeers)
			}
			adminConfigWrite := admin.Group("")
			adminConfigWrite.Use(s.requireAnyScope(ScopeConfigWrite, ScopeAdmin))
			{
				adminConfigWrite.POST("/mesh-configs/:id/revoke", s.handleAdminRevokeMeshConfig)
				adminConfigWrite.POST("/users/:id/revoke-mesh-configs", s.handleAdminRevokeMeshUserConfigs)
				adminConfigWrite.POST("/wireguard/configs/:id/revoke", s.handleAdminRevokeWireGuardConfig)
			}

			// API key management (admin scope - sensitive auth operations)
			apiKeyRoutes := admin.Group("")
			apiKeyRoutes.Use(s.requireScope(ScopeAdmin))
			{
				apiKeyRoutes.GET("/api-keys", s.handleAdminListAPIKeys)
				apiKeyRoutes.POST("/api-keys", s.handleAdminCreateAPIKey)
				apiKeyRoutes.GET("/api-keys/:id", s.handleAdminGetAPIKey)
				apiKeyRoutes.DELETE("/api-keys/:id", s.handleAdminRevokeAPIKey)
				apiKeyRoutes.GET("/users/:id/api-keys", s.handleAdminListUserAPIKeys)
				apiKeyRoutes.POST("/users/:id/api-keys", s.handleAdminCreateUserAPIKey)
				apiKeyRoutes.DELETE("/users/:id/api-keys", s.handleAdminRevokeUserAPIKeys)
				apiKeyRoutes.DELETE("/users/:id/api-keys/all", s.handleAdminDeleteUserAPIKeys)
			}

			// Geo-fencing management with scope enforcement
			geoRead := admin.Group("")
			geoRead.Use(s.requireAnyScope(ScopeConfigRead, ScopeConfigWrite, ScopeAdmin))
			{
				geoRead.GET("/geo-fence/settings", s.handleGetGeoFenceSettings)
				geoRead.GET("/geo-fence/rules", s.handleListGeoFenceRules)
				geoRead.GET("/geo-fence/rules/:id", s.handleGetGeoFenceRule)
				geoRead.GET("/geo-fence/global", s.handleListGlobalGeoRules)
				geoRead.GET("/geo-fence/users/:userId/rules", s.handleListUserGeoRules)
				geoRead.GET("/geo-fence/groups/:groupName/rules", s.handleListGroupGeoRules)
			}
			geoWrite := admin.Group("")
			geoWrite.Use(s.requireAnyScope(ScopeConfigWrite, ScopeAdmin))
			{
				geoWrite.PUT("/geo-fence/settings", s.handleUpdateGeoFenceSettings)
				geoWrite.POST("/geo-fence/rules", s.handleCreateGeoFenceRule)
				geoWrite.PUT("/geo-fence/rules/:id", s.handleUpdateGeoFenceRule)
				geoWrite.DELETE("/geo-fence/rules/:id", s.handleDeleteGeoFenceRule)
				geoWrite.POST("/geo-fence/global", s.handleAddGlobalGeoRule)
				geoWrite.DELETE("/geo-fence/global/:ruleId", s.handleRemoveGlobalGeoRule)
				geoWrite.POST("/geo-fence/users/:userId/rules", s.handleAddUserGeoRule)
				geoWrite.DELETE("/geo-fence/users/:userId/rules/:ruleId", s.handleRemoveUserGeoRule)
				geoWrite.POST("/geo-fence/groups/:groupName/rules", s.handleAddGroupGeoRule)
				geoWrite.DELETE("/geo-fence/groups/:groupName/rules/:ruleId", s.handleRemoveGroupGeoRule)
			}

			// JIT access management with scope enforcement
			jitRead := admin.Group("")
			jitRead.Use(s.requireAnyScope(ScopeConfigRead, ScopeConfigWrite, ScopeAdmin))
			{
				jitRead.GET("/jit/requests", s.handleListAllJITRequests)
				jitRead.GET("/jit/stats", s.handleGetJITStats)
				jitRead.GET("/jit/stats/detailed", s.handleGetJITDetailedStats)
				jitRead.GET("/jit/policies", s.handleListJITPolicies)
			}
			jitWrite := admin.Group("")
			jitWrite.Use(s.requireAnyScope(ScopeConfigWrite, ScopeAdmin))
			{
				jitWrite.POST("/jit/requests/:id/approve", s.handleApproveJITRequest)
				jitWrite.POST("/jit/requests/:id/deny", s.handleDenyJITRequest)
				jitWrite.POST("/jit/grants/:id/revoke", s.handleRevokeJITGrant)
				jitWrite.POST("/jit/policies", s.handleCreateJITPolicy)
				jitWrite.PUT("/jit/policies/:id", s.handleUpdateJITPolicy)
				jitWrite.DELETE("/jit/policies/:id", s.handleDeleteJITPolicy)
			}

			// Topology and network tools (admin scope - powerful operations)
			toolsRoutes := admin.Group("")
			toolsRoutes.Use(s.requireScope(ScopeAdmin))
			{
				toolsRoutes.GET("/topology", s.handleGetTopology)
				toolsRoutes.GET("/sessions/active", s.handleGetActiveSessions)
				toolsRoutes.POST("/sessions/:sessionId/disconnect", s.handleDisconnectSession)
				toolsRoutes.GET("/network-tools", s.handleListNetworkTools)
				toolsRoutes.POST("/network-tools/execute", s.handleExecuteNetworkTool)
				toolsRoutes.GET("/remote-session/agents", s.handleGetConnectedAgents)

				// User management (disconnect, disable, status)
				toolsRoutes.POST("/users/:id/disconnect", s.handleDisconnectUser)
				toolsRoutes.POST("/users/:id/disable", s.handleDisableUser)
				toolsRoutes.POST("/users/:id/enable", s.handleEnableUser)
				toolsRoutes.GET("/users/:id/status", s.handleGetUserStatus)
				toolsRoutes.GET("/users/:id/sessions", s.handleGetUserActiveSessions)
			}

			// PKI management (admin scope - certificate revocation)
			pkiRoutes := admin.Group("/pki")
			pkiRoutes.Use(s.requireScope(ScopeAdmin))
			{
				pkiRoutes.POST("/revoke", s.handleRevokeCertificate)
				pkiRoutes.GET("/revocations", s.handleListRevocations)
				pkiRoutes.DELETE("/revocations/:serial", s.handleUnrevokeCertificate)
			}
		}

		// User API key management
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.GET("", s.handleListUserAPIKeys)
			apiKeys.POST("", s.handleCreateUserAPIKey)
			apiKeys.GET("/:id", s.handleGetUserAPIKey)
			apiKeys.DELETE("/:id", s.handleRevokeUserAPIKey)
		}

		// API key validation (for CLI login)
		v1.GET("/auth/api-key/validate", s.handleValidateAPIKey)

		// User proxy applications portal
		v1.GET("/proxy-apps", s.handleListUserProxyApps)

		// JIT access - user endpoints
		v1.GET("/jit/resources", s.handleListJITResources)
		v1.POST("/jit/requests", s.handleCreateJITRequest)
		v1.GET("/jit/requests", s.handleListMyJITRequests)
		v1.POST("/jit/requests/:id/cancel", s.handleCancelJITRequest)
		v1.GET("/jit/grants", s.handleListMyJITGrants)

		// User mesh hub access
		v1.GET("/mesh/hubs", s.handleListUserMeshHubs)
		v1.POST("/mesh/generate-config", s.handleGenerateMeshClientConfig)
		v1.POST("/mesh/wireguard/generate-config", s.handleGenerateWireGuardMeshClientConfig)

		// User mesh config management
		v1.GET("/mesh-configs", s.handleListUserMeshConfigs)
		v1.GET("/mesh-configs/:id/download", s.handleDownloadMeshConfig)
		v1.POST("/mesh-configs/:id/revoke", s.handleRevokeMeshConfig)

		// User WireGuard config management
		wgConfigs := v1.Group("/wireguard/configs")
		{
			wgConfigs.POST("/generate", s.handleGenerateWireGuardConfig)
			wgConfigs.GET("", s.handleListUserWireGuardConfigs)
			wgConfigs.GET("/download/:id", s.handleDownloadWireGuardConfig)
			wgConfigs.POST("/:id/revoke", s.handleRevokeWireGuardConfig)
		}
	}

	// WebSocket endpoints for remote sessions (outside API group - no JSON middleware)
	s.router.GET("/ws/agent", s.handleAgentWebSocket)
	s.router.GET("/ws/admin/session", s.handleAdminSessionWebSocket)

	// Metrics endpoint using Prometheus handler
	if s.config.Metrics.Enabled {
		// Set server info metric
		metrics.ServerInfo.WithLabelValues("1.0.0", "go1.22").Set(1)
		// Use the standard Prometheus HTTP handler
		s.router.GET(s.config.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// Reverse proxy routes (outside API group, handles /proxy/{slug}/*)
	s.router.Any("/proxy/:slug", s.handleProxyRequest)
	s.router.Any("/proxy/:slug/*path", s.handleProxyRequest)

	// NoRoute handler to redirect unhandled paths to proxy context
	// This catches JavaScript-generated absolute path requests (e.g., /api2/..., /pve2/...)
	// and redirects them to the correct proxy based on Referer or cookie
	s.router.NoRoute(s.handleProxyContextRedirect)

	// Scripts endpoint for installers
	s.router.GET("/scripts/install-gateway.sh", s.handleInstallScript)
	s.router.GET("/scripts/install-client.sh", s.handleClientInstallScript)
	s.router.GET("/scripts/install-hub.sh", s.handleHubInstallScript)
	s.router.GET("/scripts/install-mesh-spoke.sh", s.handleMeshSpokeGenericInstallScript)
	s.router.GET("/scripts/install-wireguard-gateway.sh", s.handleWireGuardGatewayInstallScript)
	s.router.GET("/scripts/install-wireguard-hub.sh", s.handleWireGuardHubInstallScript)
	s.router.GET("/scripts/install-wireguard-mesh-spoke.sh", s.handleWireGuardMeshSpokeInstallScript)
	s.router.GET("/install.sh", s.handleInstallScript) // Alias for easy curl install

	// Downloads endpoints
	s.router.GET("/downloads", s.handleDownloadsPage)
	s.router.GET("/downloads/:filename", s.handleDownloadBinary)
	s.router.GET("/bin/:filename", s.handleDownloadBinary) // Alias for /downloads

	// Public PKI endpoints (no authentication required)
	pki := s.router.Group("/pki")
	{
		pki.GET("/crl", s.handleGetCRL)                    // CRL in DER format
		pki.GET("/crl.pem", s.handleGetCRLPEM)             // CRL in PEM format
		pki.GET("/check/:serial", s.handleCheckRevocation) // Check if certificate is revoked
	}
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// ListenAndServeTLS starts the HTTPS server.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Cancel background tasks
	if s.bgCancel != nil {
		s.bgCancel()
	}

	// Shutdown telemetry provider to flush traces
	if s.telemetry != nil {
		if err := s.telemetry.Shutdown(ctx); err != nil {
			s.logger.Warn("Failed to shutdown telemetry provider", zap.Error(err))
		}
	}

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// runGatewayHealthCheck periodically marks gateways as inactive if they haven't sent a heartbeat
func (s *Server) runGatewayHealthCheck(ctx context.Context) {
	// Gateway heartbeat interval is 30s, so check every 30s and mark inactive after 2 minutes
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	threshold := 2 * time.Minute

	s.logger.Info("Started gateway health check background task",
		zap.Duration("interval", 30*time.Second),
		zap.Duration("threshold", threshold))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Gateway health check stopped")
			return
		case <-ticker.C:
			count, err := s.gatewayStore.MarkInactiveGateways(ctx, threshold)
			if err != nil {
				s.logger.Error("Failed to mark inactive gateways", zap.Error(err))
			} else if count > 0 {
				s.logger.Info("Marked gateways as inactive", zap.Int64("count", count))
			}
		}
	}
}

// runConfigCleanup periodically deletes expired VPN configs
func (s *Server) runConfigCleanup(ctx context.Context) {
	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	s.logger.Info("Started config cleanup background task", zap.Duration("interval", 1*time.Hour))

	// Run once at startup
	s.cleanupExpiredConfigs(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Config cleanup stopped")
			return
		case <-ticker.C:
			s.cleanupExpiredConfigs(ctx)
		}
	}
}

// cleanupExpiredConfigs deletes configs that have expired + 1 hour buffer
func (s *Server) cleanupExpiredConfigs(ctx context.Context) {
	// Get VPN cert validity from settings (default 24 hours)
	validityHours := s.settingsStore.GetInt(ctx, db.SettingVPNCertValidityHours, 24)

	// Delete configs that expired more than 1 hour ago
	// The config already has an expiry based on validity, so we add 1 hour buffer
	olderThan := time.Duration(1) * time.Hour

	// Clean up gateway configs
	count, err := s.configStore.DeleteExpiredConfigs(ctx, olderThan)
	if err != nil {
		s.logger.Error("Failed to cleanup expired gateway configs", zap.Error(err))
	} else if count > 0 {
		s.logger.Info("Cleaned up expired gateway configs",
			zap.Int64("deleted", count),
			zap.Int("validity_hours", validityHours),
			zap.Duration("buffer", olderThan))
	}

	// Clean up mesh configs
	meshCount, err := s.meshConfigStore.DeleteExpiredConfigs(ctx, olderThan)
	if err != nil {
		s.logger.Error("Failed to cleanup expired mesh configs", zap.Error(err))
	} else if meshCount > 0 {
		s.logger.Info("Cleaned up expired mesh configs",
			zap.Int64("deleted", meshCount),
			zap.Int("validity_hours", validityHours),
			zap.Duration("buffer", olderThan))
	}

	// Clean up WireGuard configs
	wgCount, err := s.wgConfigStore.DeleteExpired(ctx, olderThan)
	if err != nil {
		s.logger.Error("Failed to cleanup expired WireGuard configs", zap.Error(err))
	} else if wgCount > 0 {
		s.logger.Info("Cleaned up expired WireGuard configs",
			zap.Int64("deleted", wgCount),
			zap.Int("validity_hours", validityHours),
			zap.Duration("buffer", olderThan))
	}

	// Clean up stale WireGuard peers (no handshake in 5 minutes)
	stalePeers, err := s.wgPeerStore.CleanupStale(ctx, 5*time.Minute)
	if err != nil {
		s.logger.Error("Failed to cleanup stale WireGuard peers", zap.Error(err))
	} else if stalePeers > 0 {
		s.logger.Info("Cleaned up stale WireGuard peers", zap.Int64("marked_disconnected", stalePeers))
	}

	// Clean up revoked API keys (delete after 24 hours)
	revokedKeysCount, err := s.apiKeyStore.DeleteRevokedKeys(ctx)
	if err != nil {
		s.logger.Error("Failed to cleanup revoked API keys", zap.Error(err))
	} else if revokedKeysCount > 0 {
		s.logger.Info("Cleaned up revoked API keys",
			zap.Int64("deleted", revokedKeysCount))
	}

	// Clean up expired API keys (delete after 30 days)
	expiredKeysCount, err := s.apiKeyStore.DeleteExpiredKeys(ctx)
	if err != nil {
		s.logger.Error("Failed to cleanup expired API keys", zap.Error(err))
	} else if expiredKeysCount > 0 {
		s.logger.Info("Cleaned up expired API keys",
			zap.Int64("deleted", expiredKeysCount))
	}
}

// runLoginLogCleanup periodically deletes old login logs based on retention setting
func (s *Server) runLoginLogCleanup(ctx context.Context) {
	// Run cleanup every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	s.logger.Info("Started login log cleanup background task", zap.Duration("interval", 6*time.Hour))

	// Run once at startup
	s.cleanupOldLoginLogs(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Login log cleanup stopped")
			return
		case <-ticker.C:
			s.cleanupOldLoginLogs(ctx)
		}
	}
}

// cleanupOldLoginLogs deletes login logs older than the retention setting
func (s *Server) cleanupOldLoginLogs(ctx context.Context) {
	// Get retention setting (default 30 days, 0 = forever)
	retentionDays := s.settingsStore.GetInt(ctx, db.SettingLoginLogRetentionDays, 30)

	// Skip cleanup if retention is 0 (keep forever)
	if retentionDays <= 0 {
		return
	}

	count, err := s.loginLogStore.DeleteOlderThan(ctx, retentionDays)
	if err != nil {
		s.logger.Error("Failed to cleanup old login logs", zap.Error(err))
		return
	}

	if count > 0 {
		s.logger.Info("Cleaned up old login logs",
			zap.Int64("deleted", count),
			zap.Int("retention_days", retentionDays))
	}
}

// runJITCleanup periodically revokes expired JIT grants and expires pending requests
func (s *Server) runJITCleanup(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	s.logger.Info("Started JIT access cleanup background task", zap.Duration("interval", 60*time.Second))
	s.cleanupJITAccess(ctx)
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("JIT access cleanup stopped")
			return
		case <-ticker.C:
			s.cleanupJITAccess(ctx)
		}
	}
}

func (s *Server) cleanupJITAccess(ctx context.Context) {
	revoked, err := s.jitStore.RevokeExpiredGrants(ctx)
	if err != nil {
		s.logger.Error("Failed to revoke expired JIT grants", zap.Error(err))
	} else if revoked > 0 {
		s.logger.Info("Revoked expired JIT grants", zap.Int("count", revoked))
	}
	expired, err := s.jitStore.ExpirePendingRequests(ctx)
	if err != nil {
		s.logger.Error("Failed to expire pending JIT requests", zap.Error(err))
	} else if expired > 0 {
		s.logger.Info("Expired pending JIT requests", zap.Int("count", expired))
	}
}

// runMeshHealthCheck periodically marks mesh spokes as inactive and syncs hub connection counts
func (s *Server) runMeshHealthCheck(ctx context.Context) {
	// Run every minute to keep spoke status and counts accurate
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	threshold := 2 * time.Minute // Mark offline after 2 minutes of no heartbeat

	s.logger.Info("Started mesh health check background task",
		zap.Duration("interval", 1*time.Minute),
		zap.Duration("threshold", threshold))

	// Run once at startup
	s.syncMeshSpokeHealth(ctx, threshold)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Mesh health check stopped")
			return
		case <-ticker.C:
			s.syncMeshSpokeHealth(ctx, threshold)
		}
	}
}

// syncMeshSpokeHealth marks inactive spokes and syncs hub connection counts
func (s *Server) syncMeshSpokeHealth(ctx context.Context, threshold time.Duration) {
	// Mark inactive spokes
	inactiveCount, err := s.meshStore.MarkInactiveMeshSpokes(ctx, threshold)
	if err != nil {
		s.logger.Error("Failed to mark inactive mesh spokes", zap.Error(err))
	} else if inactiveCount > 0 {
		s.logger.Info("Marked mesh spokes as offline", zap.Int64("count", inactiveCount))
	}

	// Sync hub connection counts
	syncCount, err := s.meshStore.SyncHubConnectionCounts(ctx)
	if err != nil {
		s.logger.Error("Failed to sync hub connection counts", zap.Error(err))
	} else if syncCount > 0 {
		s.logger.Info("Synced hub connection counts", zap.Int64("hubs_updated", syncCount))
	}
}

// adminAuthMiddleware returns middleware that requires admin authentication.
// It validates the session/API key and checks for admin privileges.
func (s *Server) adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := s.getAuthenticatedUser(c)
		if err != nil || user == nil {
			s.logger.Warn("Admin auth failed - no user",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		if !user.IsAdmin {
			s.logger.Warn("Admin auth failed - not admin",
				zap.String("email", user.Email),
				zap.Bool("isAdmin", user.IsAdmin),
				zap.String("provider", user.Provider),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
			})
			return
		}

		// Store user info in context for handlers
		c.Set("authenticated_user", user)
		c.Next()
	}
}

// requireScope returns middleware that requires specific API key scopes.
// Session-authenticated users are allowed through without scope checking.
// API key users must have the required scope or the wildcard "*" scope.
func (s *Server) requireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user was already authenticated by adminAuthMiddleware
		var user *authenticatedUser
		if cachedUser, exists := c.Get("authenticated_user"); exists {
			user = cachedUser.(*authenticatedUser)
		} else {
			var err error
			user, err = s.getAuthenticatedUser(c)
			if err != nil || user == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
				})
				return
			}
			c.Set("authenticated_user", user)
		}

		if !user.HasScope(requiredScope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient scope",
				"required_scope": requiredScope,
			})
			return
		}

		c.Next()
	}
}

// requireAnyScope returns middleware that requires any of the specified scopes.
func (s *Server) requireAnyScope(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user was already authenticated by adminAuthMiddleware
		var user *authenticatedUser
		if cachedUser, exists := c.Get("authenticated_user"); exists {
			user = cachedUser.(*authenticatedUser)
		} else {
			var err error
			user, err = s.getAuthenticatedUser(c)
			if err != nil || user == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
				})
				return
			}
			c.Set("authenticated_user", user)
		}

		if !user.HasAnyScope(requiredScopes...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":           "Insufficient scope",
				"required_scopes": requiredScopes,
			})
			return
		}

		c.Next()
	}
}

// resolveUserID resolves a user identifier (UUID, username, or email) to a UUID.
// If the input already looks like a UUID, it's returned as-is. Otherwise, the user
// is looked up by username or email in both SSO and local user stores.
func (s *Server) resolveUserID(ctx context.Context, userIdentifier string) (string, error) {
	// If it looks like a UUID already, return as-is
	if len(userIdentifier) == 36 && strings.Count(userIdentifier, "-") == 4 {
		return userIdentifier, nil
	}

	// Try as local username
	if localUser, err := s.userStore.GetLocalUserByUsername(ctx, userIdentifier); err == nil && localUser != nil {
		return localUser.ID, nil
	}

	// Try as email (local users)
	if localUser, err := s.userStore.GetLocalUserByEmail(ctx, userIdentifier); err == nil && localUser != nil {
		return localUser.ID, nil
	}

	// Try as email (SSO users)
	if ssoUser, err := s.userStore.GetSSOUserByEmail(ctx, userIdentifier); err == nil && ssoUser != nil {
		return ssoUser.ID, nil
	}

	return "", fmt.Errorf("user not found: %s", userIdentifier)
}

const (
	csrfCookieName = "gatekey_csrf"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenLen   = 32
)

// csrfMiddleware provides CSRF protection using the double-submit cookie pattern.
// It sets a CSRF token in a cookie and requires the same token in a header for
// state-changing requests (POST, PUT, DELETE, PATCH).
// Requests using API key authentication (Authorization: Bearer gk_*) are exempt
// since they're not vulnerable to CSRF attacks.
func (s *Server) csrfMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			// Ensure CSRF cookie is set for subsequent requests
			s.ensureCSRFCookie(c)
			c.Next()
			return
		}

		// Skip CSRF for Bearer token authentication (not vulnerable to CSRF)
		// This includes API keys (gk_*) and JWT tokens from CLI/mobile clients
		// CSRF attacks can only exploit session cookies, not Authorization headers
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// Skip CSRF for gateway-to-server communication
		// Check for gateway token header
		if c.GetHeader("X-Gateway-Token") != "" || strings.HasPrefix(authHeader, "Gateway ") {
			c.Next()
			return
		}
		// Skip CSRF for gateway internal routes (they use body-based token auth)
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/gateway/") ||
			strings.HasPrefix(path, "/api/v1/wireguard-gateway/") ||
			strings.HasPrefix(path, "/api/v1/mesh-hub/") ||
			strings.HasPrefix(path, "/api/v1/mesh-spoke/") ||
			strings.HasPrefix(path, "/api/v1/mesh-gateway/") ||
			strings.HasPrefix(path, "/api/v1/wg-mesh-hub/") ||
			strings.HasPrefix(path, "/api/v1/wg-mesh-spoke/") {
			c.Next()
			return
		}

		// Skip CSRF for SAML ACS (handled by signature validation)
		if c.Request.URL.Path == "/api/v1/auth/saml/acs" {
			c.Next()
			return
		}

		// Skip CSRF for local login (JSON body with credentials, not vulnerable to CSRF)
		if c.Request.URL.Path == "/api/v1/auth/local/login" {
			c.Next()
			return
		}

		// Validate CSRF token for state-changing requests with session auth (cookie-based)
		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			s.logger.Warn("CSRF validation failed - cookie missing",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing",
			})
			return
		}

		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" {
			s.logger.Warn("CSRF validation failed - header missing",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token header missing",
			})
			return
		}

		// Validate tokens match
		if headerToken != cookieToken {
			s.logger.Warn("CSRF validation failed - token mismatch",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token mismatch",
			})
			return
		}

		c.Next()
	}
}

// ensureCSRFCookie sets a CSRF token cookie if not present.
func (s *Server) ensureCSRFCookie(c *gin.Context) {
	if _, err := c.Cookie(csrfCookieName); err != nil {
		token, err := generateCSRFToken()
		if err != nil {
			// Critical security failure - log and don't set cookie
			// The middleware will reject state-changing requests without a valid token
			s.logger.Error("Failed to generate CSRF token", zap.Error(err))
			return
		}
		// Set cookie with SameSite=Lax for CSRF protection
		// HttpOnly=false because JavaScript needs to read the token for header inclusion
		// Secure=true when TLS is enabled
		cookie := &http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			MaxAge:   86400,
			Path:     "/",
			Secure:   s.config.Server.TLSEnabled,
			HttpOnly: false, // Must be false so JS can read and include in header
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(c.Writer, cookie)
		s.logger.Debug("CSRF cookie set",
			zap.String("path", c.Request.URL.Path),
			zap.Bool("secure", s.config.Server.TLSEnabled),
		)
	}
}

// generateCSRFToken creates a cryptographically secure random token.
// Returns an error if random generation fails - this is a critical security function.
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// rateLimiter implements a sliding window rate limiter for auth endpoints.
type rateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int           // Max requests per window
	window   time.Duration // Time window
}

// newRateLimiter creates a rate limiter with the specified limit and window.
// The context is used to stop the cleanup goroutine on shutdown.
func newRateLimiter(ctx context.Context, limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup(ctx)
	return rl
}

// allow checks if a request from the given key is allowed.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get existing requests for this key
	requests := rl.requests[key]

	// Filter to only requests within the window
	validRequests := make([]time.Time, 0, len(requests))
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.limit {
		rl.requests[key] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	return true
}

// cleanup periodically removes old entries to prevent memory leaks.
// It stops when the context is canceled.
func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			windowStart := now.Add(-rl.window)

			for key, requests := range rl.requests {
				validRequests := make([]time.Time, 0)
				for _, t := range requests {
					if t.After(windowStart) {
						validRequests = append(validRequests, t)
					}
				}
				if len(validRequests) == 0 {
					delete(rl.requests, key)
				} else {
					rl.requests[key] = validRequests
				}
			}
			rl.mu.Unlock()
		}
	}
}

// authRateLimitMiddleware provides rate limiting for authentication endpoints.
// It limits requests per IP to prevent brute force attacks.
func (s *Server) authRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use client IP as rate limit key
		key := c.ClientIP()

		if !s.authRateLimiter.allow(key) {
			s.logger.Warn("Rate limit exceeded for auth endpoint",
				zap.String("ip", key),
				zap.String("path", c.Request.URL.Path))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many authentication attempts. Please try again later.",
			})
			return
		}

		c.Next()
	}
}

// securityHeadersMiddleware returns a Gin middleware that adds security headers to all responses.
// It is multi-tenant aware and can dynamically set frame-ancestors based on the Host header.
// NOTE: Security headers are NOT applied to /proxy/* routes to avoid breaking proxied applications.
// Proxied apps have their own security headers that should not be overwritten.
func securityHeadersMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip security headers for proxy routes - proxied apps have their own headers
		// and our CSP/X-Content-Type-Options can break them
		if strings.HasPrefix(c.Request.URL.Path, "/proxy/") {
			c.Next()
			return
		}

		// X-Content-Type-Options: Prevents MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: Legacy clickjacking protection (CSP frame-ancestors is preferred)
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// X-XSS-Protection: Legacy XSS protection (modern browsers use CSP)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: Controls how much referrer info is sent
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy: Restrict browser features
		if cfg.Security.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", cfg.Security.PermissionsPolicy)
		}

		// HSTS: HTTP Strict Transport Security (only when TLS is enabled)
		if cfg.Security.HSTSEnabled && cfg.Server.TLSEnabled {
			hstsValue := buildHSTSHeader(cfg.Security.HSTSMaxAge, cfg.Security.HSTSIncludeSubdomains, cfg.Security.HSTSPreload)
			c.Header("Strict-Transport-Security", hstsValue)
		}

		// Content-Security-Policy: Main defense against XSS and injection attacks
		csp := cfg.Security.ContentSecurityPolicy
		if csp == "" {
			// Build dynamic CSP for multi-tenant deployments
			frameAncestors := "'self'"
			if cfg.Security.FrameAncestors == "dynamic" {
				// Use the Host header for multi-tenant frame-ancestors
				// This allows vpn.companya.com to frame itself but not others
				host := c.Request.Host
				if host != "" {
					// Extract just the hostname without port
					if idx := strings.Index(host, ":"); idx != -1 {
						host = host[:idx]
					}
					frameAncestors = "'self' https://" + host
				}
			} else if cfg.Security.FrameAncestors != "" && cfg.Security.FrameAncestors != "self" {
				frameAncestors = cfg.Security.FrameAncestors
			}

			// Secure default CSP - balanced for React production builds
			// Note: 'unsafe-inline' required for React/Tailwind CSS-in-JS patterns
			// 'unsafe-eval' removed - not needed for production React builds
			csp = "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline'; " + // unsafe-inline for React; eval not needed in production
				"style-src 'self' 'unsafe-inline'; " + // Inline styles for React/Tailwind
				"img-src 'self' data: https:; " + // Allow data URIs and HTTPS images
				"font-src 'self' data:; " + // Allow data URI fonts
				"connect-src 'self' wss: https:; " + // WebSocket and API connections
				"frame-ancestors " + frameAncestors + "; " +
				"base-uri 'self'; " +
				"form-action 'self'"
		}
		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}

// buildHSTSHeader constructs the HSTS header value
func buildHSTSHeader(maxAge int, includeSubdomains, preload bool) string {
	header := "max-age=" + strconv.Itoa(maxAge)
	if includeSubdomains {
		header += "; includeSubDomains"
	}
	if preload {
		header += "; preload"
	}
	return header
}

// requestBodyLimitMiddleware limits the size of request bodies to prevent DoS attacks.
// Default limit is 10MB which is sufficient for most API operations.
func requestBodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// zapLogger returns a Gin middleware that logs requests using zap.
func zapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if query != "" {
			path = path + "?" + query
		}

		logger.Info("request",
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
		)
	}
}

// Health check handlers

// healthCheck returns overall system health including all dependencies.
// Use this for comprehensive health monitoring dashboards.
func (s *Server) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	status := "healthy"
	httpStatus := http.StatusOK

	checks := make(map[string]interface{})

	// Check database connectivity
	if err := s.db.Ping(ctx); err != nil {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
		checks["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		checks["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check CA availability
	if s.ca == nil {
		checks["ca"] = map[string]interface{}{
			"status": "unavailable",
		}
	} else {
		checks["ca"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"checks":  checks,
		"version": "1.0.0",
	})
}

// readyCheck indicates if the server is ready to accept traffic.
// Use this for Kubernetes readiness probes.
func (s *Server) readyCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database connectivity - required for readiness
	if err := s.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "database unavailable",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// liveCheck indicates if the server process is alive.
// Use this for Kubernetes liveness probes. Should always return 200 if the process is running.
func (s *Server) liveCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// pkiStoreAdapter adapts db.PKIStore to pki.CAStore interface.
type pkiStoreAdapter struct {
	store *db.PKIStore
}

func (a *pkiStoreAdapter) GetCA(ctx context.Context) (*pki.StoredCA, error) {
	dbCA, err := a.store.GetCA(ctx)
	if err != nil {
		return nil, err
	}
	return &pki.StoredCA{
		CertificatePEM: dbCA.CertificatePEM,
		PrivateKeyPEM:  dbCA.PrivateKeyPEM,
		SerialNumber:   dbCA.SerialNumber,
		NotBefore:      dbCA.NotBefore,
		NotAfter:       dbCA.NotAfter,
	}, nil
}

func (a *pkiStoreAdapter) SaveCA(ctx context.Context, ca *pki.StoredCA) error {
	dbCA := &db.StoredCA{
		CertificatePEM: ca.CertificatePEM,
		PrivateKeyPEM:  ca.PrivateKeyPEM,
		SerialNumber:   ca.SerialNumber,
		NotBefore:      ca.NotBefore,
		NotAfter:       ca.NotAfter,
	}
	return a.store.SaveCA(ctx, dbCA)
}

// Security settings helpers - check database first, fall back to config

// GetProxySSRFProtection returns whether SSRF protection is enabled for proxied requests.
// Checks database setting first, falls back to config file setting.
func (s *Server) GetProxySSRFProtection(ctx context.Context) bool {
	return s.settingsStore.GetBool(ctx, db.SettingProxySSRFProtection, s.config.Security.ProxySSRFProtection)
}

// GetProxyDNSRebindingProtection returns whether DNS rebinding protection is enabled.
// Checks database setting first, falls back to config file setting.
func (s *Server) GetProxyDNSRebindingProtection(ctx context.Context) bool {
	return s.settingsStore.GetBool(ctx, db.SettingProxyDNSRebindingProtection, s.config.Security.ProxyDNSRebindingProtection)
}

// GetProxyTLSVerify returns whether TLS certificate verification is enabled for proxied requests.
// Checks database setting first, falls back to config file setting.
func (s *Server) GetProxyTLSVerify(ctx context.Context) bool {
	return s.settingsStore.GetBool(ctx, db.SettingProxyTLSVerify, s.config.Security.ProxyTLSVerify)
}
