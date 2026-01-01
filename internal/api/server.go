// Package api implements the HTTP API server for GateKey.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/k8s"
	"github.com/gatekey-project/gatekey/internal/openvpn"
	"github.com/gatekey-project/gatekey/internal/pki"
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
	apiKeyStore     *db.APIKeyStore
	ca              *pki.CA
	configGen       *openvpn.ConfigGenerator
	adminPassword   string             // Initial admin password (shown once at startup)
	bgCancel        context.CancelFunc // Cancel function for background tasks
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

	// Configure CORS
	if len(cfg.Server.CORSOrigins) > 0 {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.Server.CORSOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	// Configure trusted proxies
	if len(cfg.Server.TrustedProxies) > 0 {
		_ = router.SetTrustedProxies(cfg.Server.TrustedProxies) // Ignore error, will use defaults
	}

	// Initialize database connection
	ctx := context.Background()
	database, err := db.New(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
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
	pkiStore := db.NewPKIStore(database)
	proxyAppStore := db.NewProxyApplicationStore(database)
	loginLogStore := db.NewLoginLogStore(database)
	meshStore := db.NewMeshStore(database)
	apiKeyStore := db.NewAPIKeyStore(database)

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
		apiKeyStore:     apiKeyStore,
		ca:              ca,
		configGen:       configGen,
		adminPassword:   adminPassword,
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

	// Setup routes
	srv.setupRoutes()

	// Start background tasks
	bgCtx, bgCancel := context.WithCancel(context.Background())
	srv.bgCancel = bgCancel
	go srv.runGatewayHealthCheck(bgCtx)
	go srv.runConfigCleanup(bgCtx)
	go srv.runLoginLogCleanup(bgCtx)

	return srv, nil
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/ready", s.readyCheck)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			// OIDC
			auth.GET("/oidc/login", s.handleOIDCLogin)
			auth.GET("/oidc/callback", s.handleOIDCCallback)

			// SAML
			auth.GET("/saml/login", s.handleSAMLLogin)
			auth.POST("/saml/acs", s.handleSAMLACS)
			auth.GET("/saml/metadata", s.handleSAMLMetadata)

			// CLI authentication (browser-based flow for CLI client)
			auth.GET("/cli/login", s.handleCLILogin)
			auth.GET("/cli/complete", s.handleCLIComplete)
			auth.GET("/cli/callback", s.handleCLICallback)
			auth.POST("/refresh", s.handleTokenRefresh)

			// Local authentication (for initial setup)
			auth.POST("/local/login", s.handleLocalLogin)
			auth.POST("/local/change-password", s.handleChangePassword)

			// Session management
			auth.POST("/logout", s.handleLogout)
			auth.GET("/session", s.handleGetSession)
			auth.GET("/providers", s.handleGetProviders)
		}

		// Admin settings routes (requires admin auth)
		settings := v1.Group("/admin/settings")
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

		// Certificate routes
		certs := v1.Group("/certs")
		{
			certs.GET("/ca", s.handleGetCACert)
			certs.POST("/revoke", s.handleRevokeCert)
		}

		// Policy routes (admin only)
		policies := v1.Group("/policies")
		{
			policies.GET("", s.handleListPolicies)
			policies.POST("", s.handleCreatePolicy)
			policies.GET("/:id", s.handleGetPolicy)
			policies.PUT("/:id", s.handleUpdatePolicy)
			policies.DELETE("/:id", s.handleDeletePolicy)
		}

		// Gateway routes (internal)
		gateway := v1.Group("/gateway")
		{
			gateway.POST("/verify", s.handleGatewayVerify)
			gateway.POST("/connect", s.handleGatewayConnect)
			gateway.POST("/disconnect", s.handleGatewayDisconnect)
			gateway.POST("/heartbeat", s.handleGatewayHeartbeat)
			gateway.POST("/provision", s.handleGatewayProvision)
			gateway.POST("/client-rules", s.handleGatewayClientRules)
			gateway.POST("/all-rules", s.handleGatewayAllRules)
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

		// User routes
		users := v1.Group("/users")
		{
			users.GET("/me", s.handleGetCurrentUser)
			users.GET("/me/connections", s.handleGetUserConnections)
		}

		// Gateway listing for authenticated users
		v1.GET("/gateways", s.handleListUserGateways)

		// Server info for clients (includes FIPS requirements)
		v1.GET("/server/info", s.handleGetServerInfo)

		// Admin routes
		admin := v1.Group("/admin")
		{
			admin.GET("/gateways", s.handleListGateways)
			admin.POST("/gateways", s.handleRegisterGateway)
			admin.PUT("/gateways/:id", s.handleUpdateGateway)
			admin.DELETE("/gateways/:id", s.handleDeleteGateway)
			admin.POST("/gateways/:id/reprovision", s.handleReprovisionGateway)
			admin.GET("/gateways/:id/networks", s.handleGetGatewayNetworks)
			admin.POST("/gateways/:id/networks", s.handleAssignGatewayNetwork)
			admin.DELETE("/gateways/:id/networks/:networkId", s.handleRemoveGatewayNetwork)
			admin.GET("/gateways/:id/users", s.handleGetGatewayUsers)
			admin.POST("/gateways/:id/users", s.handleAssignGatewayUser)
			admin.DELETE("/gateways/:id/users/:userId", s.handleRemoveGatewayUser)
			admin.GET("/gateways/:id/groups", s.handleGetGatewayGroups)
			admin.POST("/gateways/:id/groups", s.handleAssignGatewayGroup)
			admin.DELETE("/gateways/:id/groups/:groupName", s.handleRemoveGatewayGroup)
			admin.GET("/connections", s.handleListConnections)
			admin.GET("/audit", s.handleGetAuditLogs)

			// Network management
			admin.GET("/networks", s.handleListNetworks)
			admin.POST("/networks", s.handleCreateNetwork)
			admin.GET("/networks/:id", s.handleGetNetwork)
			admin.PUT("/networks/:id", s.handleUpdateNetwork)
			admin.DELETE("/networks/:id", s.handleDeleteNetwork)
			admin.GET("/networks/:id/gateways", s.handleGetNetworkGateways)
			admin.GET("/networks/:id/access-rules", s.handleGetNetworkAccessRules)

			// Access rules management
			admin.GET("/access-rules", s.handleListAccessRules)
			admin.POST("/access-rules", s.handleCreateAccessRule)
			admin.GET("/access-rules/:id", s.handleGetAccessRule)
			admin.PUT("/access-rules/:id", s.handleUpdateAccessRule)
			admin.DELETE("/access-rules/:id", s.handleDeleteAccessRule)
			admin.POST("/access-rules/:id/users", s.handleAssignRuleToUser)
			admin.DELETE("/access-rules/:id/users/:userId", s.handleRemoveRuleFromUser)
			admin.POST("/access-rules/:id/groups", s.handleAssignRuleToGroup)
			admin.DELETE("/access-rules/:id/groups/:groupName", s.handleRemoveRuleFromGroup)

			// User management
			admin.GET("/users", s.handleListUsers)
			admin.GET("/users/:id", s.handleGetUser)
			admin.GET("/users/:id/access-rules", s.handleGetUserAccessRules)
			admin.GET("/users/:id/gateways", s.handleGetUserGateways)
			admin.POST("/users/:id/gateways", s.handleAssignUserGateway)
			admin.DELETE("/users/:id/gateways/:gatewayId", s.handleRemoveUserGateway)
			admin.POST("/users/:id/revoke-configs", s.handleAdminRevokeUserConfigs)

			// Config management (admin)
			admin.POST("/configs/:id/revoke", s.handleAdminRevokeConfig)
			admin.GET("/local-users", s.handleListLocalUsers)
			admin.POST("/local-users", s.handleCreateLocalUser)
			admin.DELETE("/local-users/:id", s.handleDeleteLocalUser)

			// Group management
			admin.GET("/groups", s.handleListGroups)
			admin.GET("/groups/:name/members", s.handleGetGroupMembers)
			admin.GET("/groups/:name/access-rules", s.handleGetGroupAccessRules)

			// Proxy application management
			admin.GET("/proxy-apps", s.handleListProxyApps)
			admin.POST("/proxy-apps", s.handleCreateProxyApp)
			admin.GET("/proxy-apps/:id", s.handleGetProxyApp)
			admin.PUT("/proxy-apps/:id", s.handleUpdateProxyApp)
			admin.DELETE("/proxy-apps/:id", s.handleDeleteProxyApp)
			admin.GET("/proxy-apps/:id/users", s.handleGetProxyAppUsers)
			admin.POST("/proxy-apps/:id/users", s.handleAssignProxyAppToUser)
			admin.DELETE("/proxy-apps/:id/users/:userId", s.handleRemoveProxyAppFromUser)
			admin.GET("/proxy-apps/:id/groups", s.handleGetProxyAppGroups)
			admin.POST("/proxy-apps/:id/groups", s.handleAssignProxyAppToGroup)
			admin.DELETE("/proxy-apps/:id/groups/:groupName", s.handleRemoveProxyAppFromGroup)
			admin.GET("/proxy-apps/:id/logs", s.handleGetProxyAppLogs)

			// Login logs / monitoring
			admin.GET("/login-logs", s.handleListLoginLogs)
			admin.GET("/login-logs/stats", s.handleGetLoginLogStats)
			admin.DELETE("/login-logs", s.handlePurgeLoginLogs)
			admin.GET("/login-logs/retention", s.handleGetLoginLogRetention)
			admin.PUT("/login-logs/retention", s.handleSetLoginLogRetention)

			// Mesh Hub management
			admin.GET("/mesh/hubs", s.handleListMeshHubs)
			admin.POST("/mesh/hubs", s.handleCreateMeshHub)
			admin.GET("/mesh/hubs/:id", s.handleGetMeshHub)
			admin.PUT("/mesh/hubs/:id", s.handleUpdateMeshHub)
			admin.DELETE("/mesh/hubs/:id", s.handleDeleteMeshHub)
			admin.POST("/mesh/hubs/:id/provision", s.handleProvisionMeshHub)
			admin.GET("/mesh/hubs/:id/install-script", s.handleMeshHubInstallScript)
			admin.GET("/mesh/hubs/:id/users", s.handleGetMeshHubUsers)
			admin.POST("/mesh/hubs/:id/users", s.handleAssignMeshHubUser)
			admin.DELETE("/mesh/hubs/:id/users/:userId", s.handleRemoveMeshHubUser)
			admin.GET("/mesh/hubs/:id/groups", s.handleGetMeshHubGroups)
			admin.POST("/mesh/hubs/:id/groups", s.handleAssignMeshHubGroup)
			admin.DELETE("/mesh/hubs/:id/groups/:groupName", s.handleRemoveMeshHubGroup)
			admin.GET("/mesh/hubs/:id/networks", s.handleGetMeshHubNetworks)
			admin.POST("/mesh/hubs/:id/networks", s.handleAssignMeshHubNetwork)
			admin.DELETE("/mesh/hubs/:id/networks/:networkId", s.handleRemoveMeshHubNetwork)

			// Mesh Spoke management
			admin.GET("/mesh/hubs/:id/spokes", s.handleListMeshSpokes)
			admin.POST("/mesh/hubs/:id/spokes", s.handleCreateMeshSpoke)
			admin.GET("/mesh/spokes/:id", s.handleGetMeshSpoke)
			admin.PUT("/mesh/spokes/:id", s.handleUpdateMeshSpoke)
			admin.DELETE("/mesh/spokes/:id", s.handleDeleteMeshSpoke)
			admin.POST("/mesh/spokes/:id/provision", s.handleProvisionMeshSpoke)
			admin.GET("/mesh/spokes/:id/install-script", s.handleMeshSpokeInstallScript)
			admin.GET("/mesh/spokes/:id/users", s.handleGetMeshSpokeUsers)
			admin.POST("/mesh/spokes/:id/users", s.handleAssignMeshSpokeUser)
			admin.DELETE("/mesh/spokes/:id/users/:userId", s.handleRemoveMeshSpokeUser)
			admin.GET("/mesh/spokes/:id/groups", s.handleGetMeshSpokeGroups)
			admin.POST("/mesh/spokes/:id/groups", s.handleAssignMeshSpokeGroup)
			admin.DELETE("/mesh/spokes/:id/groups/:groupName", s.handleRemoveMeshSpokeGroup)

			// API key management (admin)
			admin.GET("/api-keys", s.handleAdminListAPIKeys)
			admin.POST("/api-keys", s.handleAdminCreateAPIKey)
			admin.GET("/api-keys/:id", s.handleAdminGetAPIKey)
			admin.DELETE("/api-keys/:id", s.handleAdminRevokeAPIKey)
			admin.GET("/users/:id/api-keys", s.handleAdminListUserAPIKeys)
			admin.POST("/users/:id/api-keys", s.handleAdminCreateUserAPIKey)
			admin.DELETE("/users/:id/api-keys", s.handleAdminRevokeUserAPIKeys)
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

		// User mesh hub access
		v1.GET("/mesh/hubs", s.handleListUserMeshHubs)
		v1.POST("/mesh/generate-config", s.handleGenerateMeshClientConfig)
	}

	// Metrics endpoint
	if s.config.Metrics.Enabled {
		s.router.GET(s.config.Metrics.Path, s.handleMetrics)
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
	s.router.GET("/install.sh", s.handleInstallScript) // Alias for easy curl install

	// Downloads endpoints
	s.router.GET("/downloads", s.handleDownloadsPage)
	s.router.GET("/downloads/:filename", s.handleDownloadBinary)
	s.router.GET("/bin/:filename", s.handleDownloadBinary) // Alias for /downloads
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

	count, err := s.configStore.DeleteExpiredConfigs(ctx, olderThan)
	if err != nil {
		s.logger.Error("Failed to cleanup expired configs", zap.Error(err))
		return
	}

	if count > 0 {
		s.logger.Info("Cleaned up expired configs",
			zap.Int64("deleted", count),
			zap.Int("validity_hours", validityHours),
			zap.Duration("buffer", olderThan))
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
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) readyCheck(c *gin.Context) {
	// TODO: Check database connectivity
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
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
