// GateKey Server - Software Defined Perimeter Control Plane
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gatekey-project/gatekey/internal/api"
	"github.com/gatekey-project/gatekey/internal/config"
)

var (
	configPath string
	logger     *zap.Logger
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-server",
		Short: "GateKey Control Plane Server",
		Long: `GateKey is a software-defined perimeter solution that wraps OpenVPN
to provide zero-trust VPN capabilities with OIDC/SAML authentication,
short-lived certificates, and policy-based access control.`,
		RunE: run,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger, err = initLogger(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting GateKey server",
		zap.String("version", "0.1.0"),
		zap.Bool("tls_enabled", cfg.Server.TLSEnabled),
	)

	// Create server
	srv, err := api.NewServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		var listenErr error
		if cfg.Server.TLSEnabled {
			logger.Info("Starting HTTPS server", zap.String("address", cfg.Server.TLSAddress))
			listenErr = srv.ListenAndServeTLS(cfg.Server.TLSAddress, cfg.Server.TLSCert, cfg.Server.TLSKey)
		} else {
			logger.Info("Starting HTTP server", zap.String("address", cfg.Server.Address))
			listenErr = srv.ListenAndServe(cfg.Server.Address)
		}
		if listenErr != nil && listenErr != http.ErrServerClosed {
			errChan <- listenErr
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutdown signal received")
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
		return err
	}

	logger.Info("Server stopped")
	return nil
}

func initLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	var zapCfg zap.Config
	if cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	zapCfg.Level = zap.NewAtomicLevelAt(level)

	if cfg.Output != "stdout" && cfg.Output != "stderr" {
		zapCfg.OutputPaths = []string{cfg.Output}
	}

	return zapCfg.Build()
}
