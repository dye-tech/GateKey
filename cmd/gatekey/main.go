// GateKey Client - User VPN client that wraps OpenVPN
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gatekey-project/gatekey/internal/client"
)

var (
	version   = "0.1.0"
	serverURL string
	cfgFile   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey",
		Short: "GateKey VPN Client",
		Long: `GateKey is a zero-trust VPN client that wraps OpenVPN.

It handles authentication via your browser and automatically manages
VPN configurations, making it easy to connect securely without manual setup.

Quick start:
  gatekey config init --server https://vpn.example.com
  gatekey login
  gatekey connect`,
	}

	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "GateKey server URL (e.g., https://vpn.example.com)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: ~/.gatekey/config.yaml)")

	// Commands
	rootCmd.AddCommand(
		loginCmd(),
		logoutCmd(),
		connectCmd(),
		disconnectCmd(),
		statusCmd(),
		listCmd(),
		configCmd(),
		versionCmd(),
		fipsCheckCmd(),
		meshCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loginCmd() *cobra.Command {
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GateKey server",
		Long: `Opens your browser to authenticate with your identity provider.
After successful authentication, your session is saved locally.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}

			if cfg.ServerURL == "" {
				return fmt.Errorf("server URL required. Use --server flag or set in config file")
			}

			auth := client.NewAuthManager(cfg)
			return auth.Login(cmd.Context(), noBrowser)
		},
	}

	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print login URL instead of opening browser")

	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear saved credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			auth := client.NewAuthManager(cfg)
			return auth.Logout()
		},
	}
}

func connectCmd() *cobra.Command {
	var gateway string
	var mesh string

	cmd := &cobra.Command{
		Use:   "connect [gateway]",
		Short: "Connect to VPN",
		Long: `Connects to a VPN gateway or mesh hub.

For gateways:
  gatekey connect <gateway>
  gatekey connect --gateway <gateway>

For mesh hubs:
  gatekey connect --mesh <hub>

If no gateway is specified and only one is available, it connects to that one.

This command:
1. Checks your authentication status
2. Downloads a fresh VPN configuration
3. Starts OpenVPN with the configuration`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}

			vpn := client.NewVPNManager(cfg)

			// If --mesh flag is provided, connect to mesh hub
			if mesh != "" {
				return vpn.ConnectMesh(cmd.Context(), mesh)
			}

			// Otherwise connect to gateway
			if len(args) > 0 {
				gateway = args[0]
			}

			return vpn.Connect(cmd.Context(), gateway)
		},
	}

	cmd.Flags().StringVarP(&gateway, "gateway", "g", "", "Gateway name to connect to")
	cmd.Flags().StringVarP(&mesh, "mesh", "m", "", "Mesh hub name to connect to")

	return cmd
}

func disconnectCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "disconnect [gateway]",
		Aliases: []string{"stop"},
		Short:   "Disconnect from VPN",
		Long: `Disconnect from a VPN gateway. If no gateway is specified and only one
gateway is connected, disconnects from that one. If multiple gateways are
connected, you must specify which one to disconnect or use --all.

Examples:
  gatekey disconnect           # Disconnect from single gateway or all if multiple
  gatekey disconnect prod-1    # Disconnect from specific gateway
  gatekey disconnect --all     # Disconnect from all gateways`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			vpn := client.NewVPNManager(cfg)

			gatewayName := ""
			if len(args) > 0 {
				gatewayName = args[0]
			}

			// If --all flag or no specific gateway, disconnect from all
			if all || gatewayName == "" {
				return vpn.Disconnect()
			}

			return vpn.DisconnectGateway(gatewayName)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Disconnect from all gateways")

	return cmd
}

func statusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show connection status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			vpn := client.NewVPNManager(cfg)
			return vpn.Status(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available gateways",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}

			vpn := client.NewVPNManager(cfg)
			return vpn.ListGateways(cmd.Context())
		},
	}
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage client configuration",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show current configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := client.LoadConfig(cfgFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				return cfg.Print()
			},
		},
		&cobra.Command{
			Use:   "set [key] [value]",
			Short: "Set a configuration value",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := client.LoadConfig(cfgFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				return cfg.Set(args[0], args[1])
			},
		},
		&cobra.Command{
			Use:   "init",
			Short: "Initialize configuration with server URL",
			RunE: func(cmd *cobra.Command, args []string) error {
				if serverURL == "" {
					return fmt.Errorf("--server flag required for init")
				}
				return client.InitConfig(cfgFile, serverURL)
			},
		},
	)

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("GateKey Client v%s\n", version)
		},
	}
}

func fipsCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fips-check",
		Short: "Check FIPS 140-3 compliance status",
		Long: `Checks the system for FIPS 140-3 compliance status.

This command verifies:
- OpenSSL FIPS mode status
- Available cryptographic ciphers
- System FIPS configuration
- OpenVPN cipher support

FIPS 140-3 is the current standard for cryptographic module validation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return client.CheckFIPSCompliance()
		},
	}
}

func meshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "Mesh VPN hub commands",
		Long: `Commands for connecting to mesh VPN hubs.

Mesh hubs allow you to access spoke networks through a central hub.
Routes are determined by your access rules (zero-trust model).

Use 'gatekey mesh list' to see available hubs.
Use 'gatekey connect --mesh <hub>' to connect (recommended).`,
	}

	cmd.AddCommand(
		meshListCmd(),
		meshConnectCmd(),
	)

	return cmd
}

func meshListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available mesh hubs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}

			vpn := client.NewVPNManager(cfg)
			return vpn.ListMeshHubs(cmd.Context())
		},
	}
}

func meshConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect [hub]",
		Short: "Connect to a mesh hub",
		Long: `Connects to a mesh VPN hub. If no hub is specified and only one
is available, it connects to that one. Otherwise, prompts for selection.

This command:
1. Checks your authentication status
2. Downloads a fresh VPN configuration for the mesh hub
3. Routes are based on your access rules (zero-trust)
4. Starts OpenVPN with the configuration

Note: You will only receive routes to networks you have explicit access
rules for. Contact your administrator if you cannot reach expected networks.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}

			hubName := ""
			if len(args) > 0 {
				hubName = args[0]
			}

			vpn := client.NewVPNManager(cfg)
			return vpn.ConnectMesh(cmd.Context(), hubName)
		},
	}

	return cmd
}
