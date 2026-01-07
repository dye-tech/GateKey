// GateKey Admin CLI - Administrative command-line interface for GateKey
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gatekey-project/gatekey/internal/adminclient"
)

var (
	// Build information (set by ldflags)
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var (
	cfgFile      string
	serverURL    string
	apiKey       string
	outputFormat string
	client       *adminclient.Client
	cfg          *adminclient.Config
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-admin",
		Short: "GateKey Admin CLI - Zero Trust VPN Management",
		Long: `GateKey Admin CLI provides administrative functions for managing
gateways, networks, access rules, users, API keys, mesh VPN, and certificates.

Authentication:
  Use 'gatekey-admin login' to authenticate via browser (SSO)
  Use 'gatekey-admin login --api-key KEY' to authenticate with an API key

Configuration:
  Use 'gatekey-admin config init --server URL' to initialize configuration
  Configuration is stored in ~/.gatekey-admin/config.yaml`,
		PersistentPreRunE: initClient,
		SilenceUsage:      true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: ~/.gatekey-admin/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "GateKey server URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")

	// Add commands
	rootCmd.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newConfigCmd(),
		newGatewayCmd(),
		newNetworkCmd(),
		newAccessRuleCmd(),
		newUserCmd(),
		newLocalUserCmd(),
		newGroupCmd(),
		newLocalGroupCmd(),
		newAPIKeyCmd(),
		newMeshCmd(),
		newCACmd(),
		newAuditCmd(),
		newConnectionCmd(),
		newTroubleshootCmd(),
		newTopologyCmd(),
		newSessionCmd(),
		newProviderCmd(),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initClient(cmd *cobra.Command, args []string) error {
	// Skip for commands that don't need auth
	skipAuth := cmd.Name() == "init" || cmd.Name() == "show" || cmd.Name() == "set" ||
		cmd.Name() == "login" || cmd.Name() == "version" || cmd.Name() == "help"
	if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
		skipAuth = true
	}

	var err error
	cfg, err = adminclient.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override with flags
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}

	if cfg.ServerURL == "" && !skipAuth {
		return fmt.Errorf("server URL not configured. Run 'gatekey-admin config init --server URL' first")
	}

	client = adminclient.NewClient(cfg)
	return nil
}

// === Login Command ===

func newLoginCmd() *cobra.Command {
	var noBrowser bool
	var useAPIKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GateKey server",
		Long: `Authenticate with the GateKey server.

By default, opens a browser for SSO authentication.
Use --api-key to authenticate with an API key instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.ServerURL == "" {
				return fmt.Errorf("server URL not configured. Run 'gatekey-admin config init --server URL' first")
			}

			ctx := context.Background()
			auth := client.Auth()

			if useAPIKey != "" {
				return auth.LoginAPIKey(ctx, useAPIKey)
			}
			return auth.Login(ctx, noBrowser)
		},
	}

	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser, display URL instead")
	cmd.Flags().StringVar(&useAPIKey, "api-key", "", "Authenticate with API key")

	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear saved credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return client.Auth().Logout()
		},
	}
}

// === Config Command ===

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	// Init subcommand
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			if server == "" {
				return fmt.Errorf("--server is required")
			}
			return adminclient.InitConfig(cfgFile, server)
		},
	}
	initCmd.Flags().String("server", "", "GateKey server URL (required)")
	initCmd.MarkFlagRequired("server")

	// Show subcommand
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Print()
		},
	}

	// Set subcommand
	setCmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Set(args[0], args[1])
		},
	}

	cmd.AddCommand(initCmd, showCmd, setCmd)
	return cmd
}

// === Gateway Command ===

func newGatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"gw"},
		Short:   "Manage VPN gateways",
	}

	// List
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all gateways",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			gateways, err := client.ListGateways(ctx)
			if err != nil {
				return err
			}
			return outputResult(gateways, []string{"ID", "Name", "Endpoint", "Status", "Clients", "Mesh Hub"}, func(item interface{}) []string {
				gw := item.(adminclient.Gateway)
				meshHub := "No"
				if gw.IsMeshHub {
					meshHub = "Yes"
				}
				return []string{gw.ID, gw.Name, gw.Endpoint, gw.Status, fmt.Sprintf("%d", gw.ClientCount), meshHub}
			})
		},
	}

	// Get
	getCmd := &cobra.Command{
		Use:   "get ID",
		Short: "Get gateway details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			gw, err := client.GetGateway(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(gw)
		},
	}

	// Create
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			endpoint, _ := cmd.Flags().GetString("endpoint")
			port, _ := cmd.Flags().GetInt("port")
			protocol, _ := cmd.Flags().GetString("protocol")
			description, _ := cmd.Flags().GetString("description")
			sessionEnabled, _ := cmd.Flags().GetBool("session")

			if name == "" || endpoint == "" {
				return fmt.Errorf("--name and --endpoint are required")
			}

			ctx := context.Background()
			gw, err := client.CreateGateway(ctx, map[string]interface{}{
				"name":            name,
				"endpoint":        endpoint,
				"port":            port,
				"protocol":        protocol,
				"description":     description,
				"session_enabled": sessionEnabled,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Gateway created: %s (%s)\n", gw.Name, gw.ID)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Gateway name (required)")
	createCmd.Flags().String("endpoint", "", "Gateway endpoint/hostname (required)")
	createCmd.Flags().Int("port", 1194, "VPN port")
	createCmd.Flags().String("protocol", "udp", "Protocol (udp/tcp)")
	createCmd.Flags().String("description", "", "Description")
	createCmd.Flags().Bool("session", true, "Enable remote sessions (default: true)")

	// Update
	updateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a gateway",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				req["name"] = name
			}
			if endpoint, _ := cmd.Flags().GetString("endpoint"); endpoint != "" {
				req["endpoint"] = endpoint
			}
			if port, _ := cmd.Flags().GetInt("port"); port > 0 {
				req["port"] = port
			}
			if description, _ := cmd.Flags().GetString("description"); description != "" {
				req["description"] = description
			}
			if cmd.Flags().Changed("session") {
				session, _ := cmd.Flags().GetBool("session")
				req["session_enabled"] = session
			}

			ctx := context.Background()
			gw, err := client.UpdateGateway(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("Gateway updated: %s\n", gw.Name)
			return nil
		},
	}
	updateCmd.Flags().String("name", "", "Gateway name")
	updateCmd.Flags().String("endpoint", "", "Gateway endpoint")
	updateCmd.Flags().Int("port", 0, "VPN port")
	updateCmd.Flags().String("description", "", "Description")
	updateCmd.Flags().Bool("session", true, "Enable remote sessions")

	// Delete
	deleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a gateway",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteGateway(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Gateway deleted")
			return nil
		},
	}

	// Provision
	provisionCmd := &cobra.Command{
		Use:   "provision ID",
		Short: "Provision a gateway (generate config)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := client.ProvisionGateway(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Gateway provisioned: %s\n", result.Gateway.Name)
			if result.Token != "" {
				fmt.Printf("Token: %s\n", result.Token)
			}
			fmt.Println("\n--- Gateway Configuration ---")
			fmt.Println(result.Config)
			return nil
		},
	}

	// Reprovision
	reprovisionCmd := &cobra.Command{
		Use:   "reprovision ID",
		Short: "Reprovision a gateway (regenerate certificates)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := client.ReprovisionGateway(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Gateway reprovisioned: %s\n", result.Gateway.Name)
			fmt.Println("\n--- Gateway Configuration ---")
			fmt.Println(result.Config)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd, provisionCmd, reprovisionCmd)
	return cmd
}

// === Network Command ===

func newNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network",
		Aliases: []string{"net"},
		Short:   "Manage networks",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			networks, err := client.ListNetworks(ctx)
			if err != nil {
				return err
			}
			return outputResult(networks, []string{"ID", "Name", "CIDR", "Gateway", "Default"}, func(item interface{}) []string {
				n := item.(adminclient.Network)
				def := "No"
				if n.IsDefault {
					def = "Yes"
				}
				return []string{n.ID, n.Name, n.CIDR, n.GatewayName, def}
			})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get ID",
		Short: "Get network details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			net, err := client.GetNetwork(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(net)
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new network",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			cidr, _ := cmd.Flags().GetString("cidr")
			gatewayID, _ := cmd.Flags().GetString("gateway")
			description, _ := cmd.Flags().GetString("description")

			if name == "" || cidr == "" || gatewayID == "" {
				return fmt.Errorf("--name, --cidr, and --gateway are required")
			}

			ctx := context.Background()
			net, err := client.CreateNetwork(ctx, map[string]interface{}{
				"name":        name,
				"cidr":        cidr,
				"gateway_id":  gatewayID,
				"description": description,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Network created: %s (%s)\n", net.Name, net.ID)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Network name (required)")
	createCmd.Flags().String("cidr", "", "Network CIDR (required)")
	createCmd.Flags().String("gateway", "", "Gateway ID (required)")
	createCmd.Flags().String("description", "", "Description")

	updateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				req["name"] = name
			}
			if cidr, _ := cmd.Flags().GetString("cidr"); cidr != "" {
				req["cidr"] = cidr
			}
			if description, _ := cmd.Flags().GetString("description"); description != "" {
				req["description"] = description
			}

			ctx := context.Background()
			net, err := client.UpdateNetwork(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("Network updated: %s\n", net.Name)
			return nil
		},
	}
	updateCmd.Flags().String("name", "", "Network name")
	updateCmd.Flags().String("cidr", "", "Network CIDR")
	updateCmd.Flags().String("description", "", "Description")

	deleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteNetwork(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Network deleted")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return cmd
}

// === Access Rule Command ===

func newAccessRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access-rule",
		Aliases: []string{"rule", "ar"},
		Short:   "Manage access rules",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all access rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			rules, err := client.ListAccessRules(ctx)
			if err != nil {
				return err
			}
			return outputResult(rules, []string{"ID", "Name", "Network", "Groups", "Active", "Priority"}, func(item interface{}) []string {
				r := item.(adminclient.AccessRule)
				active := "No"
				if r.IsActive {
					active = "Yes"
				}
				return []string{r.ID, r.Name, r.NetworkName, strings.Join(r.Groups, ","), active, fmt.Sprintf("%d", r.Priority)}
			})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get ID",
		Short: "Get access rule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			rule, err := client.GetAccessRule(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(rule)
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new access rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			networkID, _ := cmd.Flags().GetString("network")
			groups, _ := cmd.Flags().GetStringSlice("groups")
			cidrs, _ := cmd.Flags().GetStringSlice("cidrs")
			priority, _ := cmd.Flags().GetInt("priority")
			description, _ := cmd.Flags().GetString("description")

			if name == "" || networkID == "" {
				return fmt.Errorf("--name and --network are required")
			}

			ctx := context.Background()
			rule, err := client.CreateAccessRule(ctx, map[string]interface{}{
				"name":          name,
				"network_id":    networkID,
				"groups":        groups,
				"allowed_cidrs": cidrs,
				"priority":      priority,
				"description":   description,
				"is_active":     true,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Access rule created: %s (%s)\n", rule.Name, rule.ID)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Rule name (required)")
	createCmd.Flags().String("network", "", "Network ID (required)")
	createCmd.Flags().StringSlice("groups", nil, "Allowed groups")
	createCmd.Flags().StringSlice("cidrs", nil, "Allowed CIDRs")
	createCmd.Flags().Int("priority", 100, "Priority")
	createCmd.Flags().String("description", "", "Description")

	updateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update an access rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				req["name"] = name
			}
			if groups, _ := cmd.Flags().GetStringSlice("groups"); len(groups) > 0 {
				req["groups"] = groups
			}
			if cidrs, _ := cmd.Flags().GetStringSlice("cidrs"); len(cidrs) > 0 {
				req["allowed_cidrs"] = cidrs
			}
			if priority, _ := cmd.Flags().GetInt("priority"); priority > 0 {
				req["priority"] = priority
			}
			if active, _ := cmd.Flags().GetBool("active"); cmd.Flags().Changed("active") {
				req["is_active"] = active
			}

			ctx := context.Background()
			rule, err := client.UpdateAccessRule(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("Access rule updated: %s\n", rule.Name)
			return nil
		},
	}
	updateCmd.Flags().String("name", "", "Rule name")
	updateCmd.Flags().StringSlice("groups", nil, "Allowed groups")
	updateCmd.Flags().StringSlice("cidrs", nil, "Allowed CIDRs")
	updateCmd.Flags().Int("priority", 0, "Priority")
	updateCmd.Flags().Bool("active", true, "Active status")

	deleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete an access rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteAccessRule(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Access rule deleted")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return cmd
}

// === User Command ===

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			users, err := client.ListUsers(ctx)
			if err != nil {
				return err
			}
			return outputResult(users, []string{"ID", "Email", "Name", "Provider", "Admin", "Active"}, func(item interface{}) []string {
				u := item.(adminclient.User)
				admin := "No"
				if u.IsAdmin {
					admin = "Yes"
				}
				active := "No"
				if u.IsActive {
					active = "Yes"
				}
				return []string{u.ID, u.Email, u.Name, u.Provider, admin, active}
			})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get ID",
		Short: "Get user details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			user, err := client.GetUser(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(user)
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if admin, _ := cmd.Flags().GetBool("admin"); cmd.Flags().Changed("admin") {
				req["is_admin"] = admin
			}
			if active, _ := cmd.Flags().GetBool("active"); cmd.Flags().Changed("active") {
				req["is_active"] = active
			}

			ctx := context.Background()
			user, err := client.UpdateUser(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("User updated: %s\n", user.Email)
			return nil
		},
	}
	updateCmd.Flags().Bool("admin", false, "Admin status")
	updateCmd.Flags().Bool("active", true, "Active status")

	revokeConfigsCmd := &cobra.Command{
		Use:   "revoke-configs ID",
		Short: "Revoke all VPN configs for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.RevokeUserConfigs(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("User configs revoked")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, updateCmd, revokeConfigsCmd)
	return cmd
}

// === Local User Command ===

func newLocalUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "local-user",
		Aliases: []string{"lu"},
		Short:   "Manage local users",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all local users",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			users, err := client.ListLocalUsers(ctx)
			if err != nil {
				return err
			}
			return outputResult(users, []string{"ID", "Username", "Email", "Admin", "Active"}, func(item interface{}) []string {
				u := item.(adminclient.LocalUser)
				admin := "No"
				if u.IsAdmin {
					admin = "Yes"
				}
				active := "No"
				if u.IsActive {
					active = "Yes"
				}
				return []string{u.ID, u.Username, u.Email, admin, active}
			})
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a local user",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			password, _ := cmd.Flags().GetString("password")
			admin, _ := cmd.Flags().GetBool("admin")

			if username == "" || email == "" || password == "" {
				return fmt.Errorf("--username, --email, and --password are required")
			}

			ctx := context.Background()
			user, err := client.CreateLocalUser(ctx, map[string]interface{}{
				"username": username,
				"email":    email,
				"password": password,
				"is_admin": admin,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Local user created: %s (%s)\n", user.Username, user.ID)
			return nil
		},
	}
	createCmd.Flags().String("username", "", "Username (required)")
	createCmd.Flags().String("email", "", "Email (required)")
	createCmd.Flags().String("password", "", "Password (required)")
	createCmd.Flags().Bool("admin", false, "Admin user")

	deleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a local user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteLocalUser(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Local user deleted")
			return nil
		},
	}

	resetPwdCmd := &cobra.Command{
		Use:   "reset-password ID",
		Short: "Reset local user password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			password, _ := cmd.Flags().GetString("password")
			if password == "" {
				return fmt.Errorf("--password is required")
			}

			ctx := context.Background()
			if err := client.ResetLocalUserPassword(ctx, args[0], map[string]string{"password": password}); err != nil {
				return err
			}
			fmt.Println("Password reset successfully")
			return nil
		},
	}
	resetPwdCmd.Flags().String("password", "", "New password (required)")

	cmd.AddCommand(listCmd, createCmd, deleteCmd, resetPwdCmd)
	return cmd
}

// === Group Command ===

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "View group information",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			groups, err := client.ListGroups(ctx)
			if err != nil {
				return err
			}
			return outputResult(groups, []string{"Name", "Members", "Rules"}, func(item interface{}) []string {
				g := item.(adminclient.Group)
				return []string{g.Name, fmt.Sprintf("%d", g.MemberCount), fmt.Sprintf("%d", g.RuleCount)}
			})
		},
	}

	membersCmd := &cobra.Command{
		Use:   "members NAME",
		Short: "List group members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			members, err := client.GetGroupMembers(ctx, args[0])
			if err != nil {
				return err
			}
			return outputResult(members, []string{"ID", "Email", "Name"}, func(item interface{}) []string {
				u := item.(adminclient.User)
				return []string{u.ID, u.Email, u.Name}
			})
		},
	}

	rulesCmd := &cobra.Command{
		Use:   "rules NAME",
		Short: "List group access rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			rules, err := client.GetGroupRules(ctx, args[0])
			if err != nil {
				return err
			}
			return outputResult(rules, []string{"ID", "Name", "Network", "Active"}, func(item interface{}) []string {
				r := item.(adminclient.AccessRule)
				active := "No"
				if r.IsActive {
					active = "Yes"
				}
				return []string{r.ID, r.Name, r.NetworkName, active}
			})
		},
	}

	cmd.AddCommand(listCmd, membersCmd, rulesCmd)
	return cmd
}

// === Local Group Command ===

func newLocalGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "local-group",
		Aliases: []string{"lg"},
		Short:   "Manage local groups",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all local groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			groups, err := client.ListLocalGroups(ctx)
			if err != nil {
				return err
			}
			return outputResult(groups, []string{"ID", "Name", "Description", "Members"}, func(item interface{}) []string {
				g := item.(adminclient.LocalGroup)
				return []string{g.ID, g.Name, g.Description, fmt.Sprintf("%d", g.MemberCount)}
			})
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a local group",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			ctx := context.Background()
			group, err := client.CreateLocalGroup(ctx, name, description)
			if err != nil {
				return err
			}
			fmt.Printf("Local group created: %s (%s)\n", group.Name, group.ID)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Group name (required)")
	createCmd.Flags().String("description", "", "Group description")

	updateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a local group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")

			ctx := context.Background()
			err := client.UpdateLocalGroup(ctx, args[0], name, description)
			if err != nil {
				return err
			}
			fmt.Println("Local group updated")
			return nil
		},
	}
	updateCmd.Flags().String("name", "", "New group name")
	updateCmd.Flags().String("description", "", "New group description")

	deleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a local group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			err := client.DeleteLocalGroup(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Println("Local group deleted")
			return nil
		},
	}

	membersCmd := &cobra.Command{
		Use:   "members ID",
		Short: "List members of a local group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			members, err := client.ListLocalGroupMembers(ctx, args[0])
			if err != nil {
				return err
			}
			return outputResult(members, []string{"User ID", "Type", "Email", "Name"}, func(item interface{}) []string {
				m := item.(adminclient.LocalGroupMember)
				return []string{m.UserID, m.MemberType, m.Email, m.Name}
			})
		},
	}

	addMemberCmd := &cobra.Command{
		Use:   "add-member GROUP_ID USER_ID",
		Short: "Add a member to a local group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			memberType, _ := cmd.Flags().GetString("type")
			if memberType != "sso" && memberType != "local" {
				return fmt.Errorf("--type must be 'sso' or 'local'")
			}

			ctx := context.Background()
			err := client.AddLocalGroupMember(ctx, args[0], args[1], memberType)
			if err != nil {
				return err
			}
			fmt.Println("Member added to local group")
			return nil
		},
	}
	addMemberCmd.Flags().String("type", "sso", "Member type (sso or local)")

	removeMemberCmd := &cobra.Command{
		Use:   "remove-member GROUP_ID USER_ID",
		Short: "Remove a member from a local group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			memberType, _ := cmd.Flags().GetString("type")
			if memberType != "sso" && memberType != "local" {
				return fmt.Errorf("--type must be 'sso' or 'local'")
			}

			ctx := context.Background()
			err := client.RemoveLocalGroupMember(ctx, args[0], args[1], memberType)
			if err != nil {
				return err
			}
			fmt.Println("Member removed from local group")
			return nil
		},
	}
	removeMemberCmd.Flags().String("type", "sso", "Member type (sso or local)")

	cmd.AddCommand(listCmd, createCmd, updateCmd, deleteCmd, membersCmd, addMemberCmd, removeMemberCmd)
	return cmd
}

// === API Key Command ===

func newAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "api-key",
		Aliases: []string{"key"},
		Short:   "Manage API keys",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("user")

			ctx := context.Background()
			var keys []adminclient.APIKey
			var err error

			if userID != "" {
				keys, err = client.ListUserAPIKeys(ctx, userID)
			} else {
				keys, err = client.ListAPIKeys(ctx)
			}
			if err != nil {
				return err
			}

			return outputResult(keys, []string{"ID", "Name", "User", "Prefix", "Scopes", "Last Used", "Revoked"}, func(item interface{}) []string {
				k := item.(adminclient.APIKey)
				lastUsed := "Never"
				if k.LastUsedAt != nil {
					lastUsed = k.LastUsedAt.Format("2006-01-02 15:04")
				}
				revoked := "No"
				if k.IsRevoked {
					revoked = "Yes"
				}
				return []string{k.ID, k.Name, k.UserEmail, k.KeyPrefix, strings.Join(k.Scopes, ","), lastUsed, revoked}
			})
		},
	}
	listCmd.Flags().String("user", "", "Filter by user ID")

	createCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("user")
			scopes, _ := cmd.Flags().GetStringSlice("scopes")
			expires, _ := cmd.Flags().GetString("expires")
			description, _ := cmd.Flags().GetString("description")

			req := map[string]interface{}{
				"name":        args[0],
				"scopes":      scopes,
				"description": description,
			}

			if expires != "" {
				duration, err := parseDuration(expires)
				if err != nil {
					return fmt.Errorf("invalid expires format: %w", err)
				}
				req["expires_at"] = time.Now().Add(duration).Format(time.RFC3339)
			}

			ctx := context.Background()
			var result *adminclient.APIKeyCreateResponse
			var err error

			if userID != "" {
				result, err = client.CreateUserAPIKey(ctx, userID, req)
			} else {
				result, err = client.CreateAPIKey(ctx, req)
			}
			if err != nil {
				return err
			}

			fmt.Println("API key created successfully!")
			fmt.Println("")
			fmt.Printf("Name:   %s\n", result.APIKey.Name)
			fmt.Printf("ID:     %s\n", result.APIKey.ID)
			fmt.Printf("Scopes: %s\n", strings.Join(result.APIKey.Scopes, ", "))
			fmt.Println("")
			fmt.Println("IMPORTANT: Save this API key now. You won't be able to see it again!")
			fmt.Printf("\n  %s\n\n", result.RawKey)
			return nil
		},
	}
	createCmd.Flags().String("user", "", "Create for specific user ID")
	createCmd.Flags().StringSlice("scopes", []string{"*"}, "Scopes (default: all)")
	createCmd.Flags().String("expires", "", "Expiration (e.g., 30d, 90d, 1y)")
	createCmd.Flags().String("description", "", "Description")

	revokeCmd := &cobra.Command{
		Use:   "revoke ID",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reason, _ := cmd.Flags().GetString("reason")
			ctx := context.Background()
			if err := client.RevokeAPIKey(ctx, args[0], reason); err != nil {
				return err
			}
			fmt.Println("API key revoked")
			return nil
		},
	}
	revokeCmd.Flags().String("reason", "", "Revocation reason")

	revokeAllCmd := &cobra.Command{
		Use:   "revoke-all",
		Short: "Revoke all API keys for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("user")
			reason, _ := cmd.Flags().GetString("reason")

			if userID == "" {
				return fmt.Errorf("--user is required")
			}

			ctx := context.Background()
			if err := client.RevokeUserAPIKeys(ctx, userID, reason); err != nil {
				return err
			}
			fmt.Println("All API keys revoked for user")
			return nil
		},
	}
	revokeAllCmd.Flags().String("user", "", "User ID (required)")
	revokeAllCmd.Flags().String("reason", "", "Revocation reason")

	cmd.AddCommand(listCmd, createCmd, revokeCmd, revokeAllCmd)
	return cmd
}

// === Mesh Command ===

func newMeshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "Manage mesh VPN topology",
	}

	// Hub subcommands
	hubCmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage mesh hubs",
	}

	hubListCmd := &cobra.Command{
		Use:   "list",
		Short: "List mesh hubs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			hubs, err := client.ListMeshHubs(ctx)
			if err != nil {
				return err
			}
			return outputResult(hubs, []string{"ID", "Gateway", "Hub Network", "Provisioned", "Spokes"}, func(item interface{}) []string {
				h := item.(adminclient.MeshHub)
				prov := "No"
				if h.IsProvisioned {
					prov = "Yes"
				}
				return []string{h.ID, h.GatewayName, h.HubNetwork, prov, fmt.Sprintf("%d", h.ConnectedSpokes)}
			})
		},
	}

	hubCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a mesh hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			gatewayID, _ := cmd.Flags().GetString("gateway")
			network, _ := cmd.Flags().GetString("network")
			sessionEnabled, _ := cmd.Flags().GetBool("session")

			if gatewayID == "" || network == "" {
				return fmt.Errorf("--gateway and --network are required")
			}

			ctx := context.Background()
			hub, err := client.CreateMeshHub(ctx, map[string]interface{}{
				"gateway_id":      gatewayID,
				"hub_network":     network,
				"session_enabled": sessionEnabled,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Mesh hub created: %s\n", hub.ID)
			return nil
		},
	}
	hubCreateCmd.Flags().String("gateway", "", "Gateway ID (required)")
	hubCreateCmd.Flags().String("network", "", "Hub network CIDR (required)")
	hubCreateCmd.Flags().Bool("session", true, "Enable remote sessions (default: true)")

	hubUpdateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a mesh hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if cmd.Flags().Changed("session") {
				session, _ := cmd.Flags().GetBool("session")
				req["session_enabled"] = session
			}
			if len(req) == 0 {
				return fmt.Errorf("no updates specified")
			}

			ctx := context.Background()
			hub, err := client.UpdateMeshHub(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("Mesh hub updated: %s\n", hub.ID)
			return nil
		},
	}
	hubUpdateCmd.Flags().Bool("session", true, "Enable remote sessions")

	hubDeleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a mesh hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteMeshHub(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Mesh hub deleted")
			return nil
		},
	}

	hubProvisionCmd := &cobra.Command{
		Use:   "provision ID",
		Short: "Provision a mesh hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := client.ProvisionMeshHub(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Println("Mesh hub provisioned")
			fmt.Println("\n--- Hub Configuration ---")
			fmt.Println(result.Config)
			return nil
		},
	}

	hubCmd.AddCommand(hubListCmd, hubCreateCmd, hubUpdateCmd, hubDeleteCmd, hubProvisionCmd)

	// Spoke subcommands
	spokeCmd := &cobra.Command{
		Use:   "spoke",
		Short: "Manage mesh spokes",
	}

	spokeListCmd := &cobra.Command{
		Use:   "list",
		Short: "List mesh spokes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			spokes, err := client.ListMeshSpokes(ctx)
			if err != nil {
				return err
			}
			return outputResult(spokes, []string{"ID", "Gateway", "Hub", "Spoke Network", "Provisioned", "Connected"}, func(item interface{}) []string {
				s := item.(adminclient.MeshSpoke)
				prov := "No"
				if s.IsProvisioned {
					prov = "Yes"
				}
				conn := "No"
				if s.IsConnected {
					conn = "Yes"
				}
				return []string{s.ID, s.GatewayName, s.HubName, s.SpokeNetwork, prov, conn}
			})
		},
	}

	spokeCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a mesh spoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			gatewayID, _ := cmd.Flags().GetString("gateway")
			hubID, _ := cmd.Flags().GetString("hub")
			network, _ := cmd.Flags().GetString("network")
			sessionEnabled, _ := cmd.Flags().GetBool("session")

			if gatewayID == "" || hubID == "" || network == "" {
				return fmt.Errorf("--gateway, --hub, and --network are required")
			}

			ctx := context.Background()
			spoke, err := client.CreateMeshSpoke(ctx, map[string]interface{}{
				"gateway_id":      gatewayID,
				"hub_id":          hubID,
				"spoke_network":   network,
				"session_enabled": sessionEnabled,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Mesh spoke created: %s\n", spoke.ID)
			return nil
		},
	}
	spokeCreateCmd.Flags().String("gateway", "", "Gateway ID (required)")
	spokeCreateCmd.Flags().String("hub", "", "Hub ID (required)")
	spokeCreateCmd.Flags().String("network", "", "Spoke network CIDR (required)")
	spokeCreateCmd.Flags().Bool("session", true, "Enable remote sessions (default: true)")

	spokeUpdateCmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a mesh spoke",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})
			if cmd.Flags().Changed("session") {
				session, _ := cmd.Flags().GetBool("session")
				req["session_enabled"] = session
			}
			if len(req) == 0 {
				return fmt.Errorf("no updates specified")
			}

			ctx := context.Background()
			spoke, err := client.UpdateMeshSpoke(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("Mesh spoke updated: %s\n", spoke.ID)
			return nil
		},
	}
	spokeUpdateCmd.Flags().Bool("session", true, "Enable remote sessions")

	spokeDeleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a mesh spoke",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteMeshSpoke(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Mesh spoke deleted")
			return nil
		},
	}

	spokeProvisionCmd := &cobra.Command{
		Use:   "provision ID",
		Short: "Provision a mesh spoke",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := client.ProvisionMeshSpoke(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Println("Mesh spoke provisioned")
			fmt.Println("\n--- Spoke Configuration ---")
			fmt.Println(result.Config)
			return nil
		},
	}

	spokeCmd.AddCommand(spokeListCmd, spokeCreateCmd, spokeUpdateCmd, spokeDeleteCmd, spokeProvisionCmd)

	cmd.AddCommand(hubCmd, spokeCmd)
	return cmd
}

// === CA Command ===

func newCACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ca",
		Short: "Manage Certificate Authority",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current CA information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			ca, err := client.GetCA(ctx)
			if err != nil {
				return err
			}
			return outputSingle(ca)
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all CAs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cas, err := client.ListCAs(ctx)
			if err != nil {
				return err
			}
			return outputResult(cas, []string{"ID", "Name", "Serial", "Active", "Current", "Valid Until"}, func(item interface{}) []string {
				ca := item.(adminclient.CA)
				active := "No"
				if ca.IsActive {
					active = "Yes"
				}
				current := "No"
				if ca.IsCurrent {
					current = "Yes"
				}
				return []string{ca.ID, ca.Name, ca.SerialNumber, active, current, ca.NotAfter.Format("2006-01-02")}
			})
		},
	}

	rotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate the CA certificate",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			years, _ := cmd.Flags().GetInt("years")

			req := map[string]interface{}{}
			if name != "" {
				req["name"] = name
			}
			if years > 0 {
				req["validity_years"] = years
			}

			ctx := context.Background()
			ca, err := client.RotateCA(ctx, req)
			if err != nil {
				return err
			}
			fmt.Printf("CA rotated successfully: %s\n", ca.Name)
			fmt.Printf("New CA valid until: %s\n", ca.NotAfter.Format("2006-01-02"))
			return nil
		},
	}
	rotateCmd.Flags().String("name", "", "New CA name")
	rotateCmd.Flags().Int("years", 10, "Validity in years")

	activateCmd := &cobra.Command{
		Use:   "activate ID",
		Short: "Activate a CA",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.ActivateCA(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("CA activated")
			return nil
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke ID",
		Short: "Revoke a CA",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.RevokeCA(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("CA revoked")
			return nil
		},
	}

	cmd.AddCommand(showCmd, listCmd, rotateCmd, activateCmd, revokeCmd)
	return cmd
}

// === Audit Command ===

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit logs",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")

			ctx := context.Background()
			logs, err := client.ListAuditLogs(ctx, limit)
			if err != nil {
				return err
			}
			return outputResult(logs, []string{"Time", "Action", "Resource", "User", "IP"}, func(item interface{}) []string {
				l := item.(adminclient.AuditLog)
				return []string{l.CreatedAt.Format("2006-01-02 15:04:05"), l.Action, l.Resource, l.UserEmail, l.IPAddress}
			})
		},
	}
	listCmd.Flags().Int("limit", 50, "Number of entries to show")

	cmd.AddCommand(listCmd)
	return cmd
}

// === Connection Command ===

func newConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection",
		Aliases: []string{"conn"},
		Short:   "Manage active VPN connections",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List active connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			conns, err := client.ListConnections(ctx)
			if err != nil {
				return err
			}
			return outputResult(conns, []string{"ID", "User", "Gateway", "Client IP", "VPN IP", "Connected"}, func(item interface{}) []string {
				c := item.(adminclient.Connection)
				return []string{c.ID, c.UserEmail, c.GatewayName, c.ClientIP, c.VPNAddress, c.ConnectedAt.Format("2006-01-02 15:04")}
			})
		},
	}

	disconnectCmd := &cobra.Command{
		Use:   "disconnect ID",
		Short: "Disconnect a client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DisconnectUser(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Client disconnected")
			return nil
		},
	}

	cmd.AddCommand(listCmd, disconnectCmd)
	return cmd
}

// === Troubleshoot Command ===

func newTroubleshootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "troubleshoot",
		Aliases: []string{"ts", "diag"},
		Short:   "Network troubleshooting tools",
		Long: `Run network diagnostic tools from the control plane or remote nodes.

Available tools: ping, nslookup, traceroute, nc (netcat), nmap

Examples:
  gatekey-admin troubleshoot ping 8.8.8.8
  gatekey-admin troubleshoot nslookup google.com
  gatekey-admin troubleshoot traceroute 10.0.0.1
  gatekey-admin troubleshoot nc 192.168.1.1 --port 443
  gatekey-admin troubleshoot nmap 10.0.0.0/24 --ports 22,80,443`,
	}

	// Ping
	pingCmd := &cobra.Command{
		Use:   "ping TARGET",
		Short: "Test ICMP connectivity to a host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location, _ := cmd.Flags().GetString("location")
			count, _ := cmd.Flags().GetInt("count")

			options := map[string]string{}
			if count > 0 {
				options["count"] = fmt.Sprintf("%d", count)
			}

			return executeNetworkTool("ping", args[0], 0, "", location, options)
		},
	}
	pingCmd.Flags().String("location", "control-plane", "Execution location (control-plane, gateway:<id>, hub:<id>, spoke:<id>)")
	pingCmd.Flags().Int("count", 4, "Number of ping packets")

	// Nslookup
	nslookupCmd := &cobra.Command{
		Use:   "nslookup TARGET",
		Short: "Perform DNS lookup for a hostname",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location, _ := cmd.Flags().GetString("location")
			return executeNetworkTool("nslookup", args[0], 0, "", location, nil)
		},
	}
	nslookupCmd.Flags().String("location", "control-plane", "Execution location")

	// Traceroute
	tracerouteCmd := &cobra.Command{
		Use:     "traceroute TARGET",
		Aliases: []string{"tracert"},
		Short:   "Trace the route to a host",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location, _ := cmd.Flags().GetString("location")
			return executeNetworkTool("traceroute", args[0], 0, "", location, nil)
		},
	}
	tracerouteCmd.Flags().String("location", "control-plane", "Execution location")

	// Netcat
	ncCmd := &cobra.Command{
		Use:     "nc TARGET",
		Aliases: []string{"netcat"},
		Short:   "Test TCP connectivity to a port",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location, _ := cmd.Flags().GetString("location")
			port, _ := cmd.Flags().GetInt("port")

			if port <= 0 {
				return fmt.Errorf("--port is required for netcat")
			}

			return executeNetworkTool("nc", args[0], port, "", location, nil)
		},
	}
	ncCmd.Flags().String("location", "control-plane", "Execution location")
	ncCmd.Flags().Int("port", 0, "Target port (required)")
	ncCmd.MarkFlagRequired("port")

	// Nmap
	nmapCmd := &cobra.Command{
		Use:   "nmap TARGET",
		Short: "Scan ports on a host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location, _ := cmd.Flags().GetString("location")
			ports, _ := cmd.Flags().GetString("ports")

			return executeNetworkTool("nmap", args[0], 0, ports, location, nil)
		},
	}
	nmapCmd.Flags().String("location", "control-plane", "Execution location")
	nmapCmd.Flags().String("ports", "", "Ports to scan (e.g., 22,80,443 or 1-1000)")

	// List tools
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available tools and execution locations",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			info, err := client.GetNetworkToolsInfo(ctx)
			if err != nil {
				return err
			}

			fmt.Println("Available Tools:")
			for _, tool := range info.Tools {
				fmt.Printf("  %-12s - %s\n", tool.Name, tool.Description)
			}

			fmt.Println("\nExecution Locations:")
			for _, loc := range info.Locations {
				fmt.Printf("  %-30s (%s)\n", loc["id"], loc["name"])
			}

			return nil
		},
	}

	cmd.AddCommand(pingCmd, nslookupCmd, tracerouteCmd, ncCmd, nmapCmd, listCmd)
	return cmd
}

func executeNetworkTool(tool, target string, port int, ports, location string, options map[string]string) error {
	ctx := context.Background()

	fmt.Printf("Executing %s to %s", tool, target)
	if port > 0 {
		fmt.Printf(":%d", port)
	}
	if location != "" && location != "control-plane" {
		fmt.Printf(" from %s", location)
	}
	fmt.Println("...")
	fmt.Println()

	result, err := client.ExecuteNetworkTool(ctx, &adminclient.NetworkToolRequest{
		Tool:     tool,
		Target:   target,
		Port:     port,
		Ports:    ports,
		Location: location,
		Options:  options,
	})
	if err != nil {
		return err
	}

	// Print output
	if result.Output != "" {
		fmt.Println(result.Output)
	}

	// Print status summary
	fmt.Println()
	fmt.Printf("Status:   %s\n", result.Status)
	fmt.Printf("Duration: %s\n", result.Duration)

	if result.Error != "" {
		fmt.Printf("Error:    %s\n", result.Error)
	}

	return nil
}

// === Topology Command ===

func newTopologyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topology",
		Short: "View network topology",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show network topology overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			topo, err := client.GetTopology(ctx)
			if err != nil {
				return err
			}

			fmt.Println("=== Network Topology ===")
			fmt.Println()

			// Gateways
			fmt.Printf("Gateways (%d):\n", len(topo.Gateways))
			for _, gw := range topo.Gateways {
				status := "inactive"
				if gw.IsActive {
					status = "active"
				}
				fmt.Printf("  - %s (%s) - %s:%d [%s]\n", gw.Name, gw.ID, gw.PublicIP, gw.VPNPort, status)
			}

			fmt.Println()

			// Mesh Hubs
			fmt.Printf("Mesh Hubs (%d):\n", len(topo.MeshHubs))
			for _, hub := range topo.MeshHubs {
				fmt.Printf("  - %s (%s) - %s [%s] - %d spokes, %d users\n",
					hub.Name, hub.ID, hub.VPNSubnet, hub.Status, hub.ConnectedSpokes, hub.ConnectedUsers)
			}

			fmt.Println()

			// Mesh Spokes
			fmt.Printf("Mesh Spokes (%d):\n", len(topo.MeshSpokes))
			for _, spoke := range topo.MeshSpokes {
				fmt.Printf("  - %s (%s) -> hub:%s - %s [%s]\n",
					spoke.Name, spoke.ID, spoke.HubID, spoke.TunnelIP, spoke.Status)
				if len(spoke.LocalNetworks) > 0 {
					fmt.Printf("      Networks: %s\n", strings.Join(spoke.LocalNetworks, ", "))
				}
			}

			fmt.Println()

			// Connections
			fmt.Printf("Connections (%d):\n", len(topo.Connections))
			for _, conn := range topo.Connections {
				fmt.Printf("  - %s -> %s [%s] (%s)\n", conn.Source, conn.Target, conn.Status, conn.Type)
			}

			return nil
		},
	}

	cmd.AddCommand(showCmd)
	return cmd
}

// === Session Command ===

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"remote", "shell"},
		Short:   "Remote session management for gateways, hubs, and spokes",
		Long: `Connect to and execute commands on remote gateways, mesh hubs, and spokes.

This feature requires agents to be connected to the control plane via the
remote session agent. Once connected, you can list available agents and
either run single commands or start an interactive shell session.

Examples:
  gatekey-admin session list
  gatekey-admin session exec hub-1 "ip addr"
  gatekey-admin session connect hub-1`,
	}

	// List connected agents
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List connected agents available for remote sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			agents, err := client.ListRemoteSessionAgents(ctx)
			if err != nil {
				return err
			}

			if len(agents) == 0 {
				fmt.Println("No agents currently connected")
				return nil
			}

			return outputResult(agents, []string{"Agent ID", "Type", "Node Name", "Connected"}, func(item interface{}) []string {
				a := item.(adminclient.RemoteSessionAgent)
				return []string{a.AgentID, a.NodeType, a.NodeName, a.ConnectedAt.Format("2006-01-02 15:04:05")}
			})
		},
	}

	// Execute single command
	execCmd := &cobra.Command{
		Use:   "exec AGENT_ID COMMAND",
		Short: "Execute a single command on an agent",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			command := strings.Join(args[1:], " ")

			return executeRemoteCommand(agentID, command)
		},
	}

	// Interactive session
	connectCmd := &cobra.Command{
		Use:   "connect AGENT_ID",
		Short: "Start an interactive shell session with an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			return startInteractiveSession(agentID)
		},
	}

	cmd.AddCommand(listCmd, execCmd, connectCmd)
	return cmd
}

// WebSocket message types for remote sessions
type wsMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	ID        string          `json:"id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

type connectAgentPayload struct {
	AgentID string `json:"agentId"`
}

type commandPayload struct {
	Command string `json:"command"`
}

type outputPayload struct {
	Output   string `json:"output"`
	IsStderr bool   `json:"is_stderr"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Done     bool   `json:"done"`
}

func executeRemoteCommand(agentID, command string) error {
	wsURL, err := client.GetWebSocketURL()
	if err != nil {
		return fmt.Errorf("failed to get WebSocket URL: %w", err)
	}

	// Get auth header
	authHeader, err := client.Auth().GetAuthHeader()
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}

	// Connect with auth
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	headers := http.Header{}
	headers.Set("Authorization", authHeader)

	conn, httpResp, err := dialer.Dial(wsURL, headers)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	defer conn.Close()

	// Read initial agent_list message (sent immediately on connect)
	var initialMsg wsMessage
	if err := conn.ReadJSON(&initialMsg); err != nil {
		return fmt.Errorf("failed to read initial message: %w", err)
	}
	// Ignore agent_list, we already know which agent we want

	// Connect to agent
	connectPayload, _ := json.Marshal(connectAgentPayload{AgentID: agentID})
	msg := wsMessage{
		Type:      "connect_agent",
		Payload:   connectPayload,
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}

	// Wait for agent_connected response
	var resp wsMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if resp.Type == "error" {
		var errPayload struct {
			Message string `json:"message"`
		}
		json.Unmarshal(resp.Payload, &errPayload)
		return fmt.Errorf("failed to connect to agent: %s", errPayload.Message)
	}
	if resp.Type != "agent_connected" {
		return fmt.Errorf("unexpected response: %s", resp.Type)
	}

	// Send command
	cmdPayload, _ := json.Marshal(commandPayload{Command: command})
	cmdMsg := wsMessage{
		Type:      "command",
		Payload:   cmdPayload,
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(cmdMsg); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Read output until done
	for {
		var outMsg wsMessage
		if err := conn.ReadJSON(&outMsg); err != nil {
			return fmt.Errorf("connection closed: %w", err)
		}

		if outMsg.Type == "output" {
			var out outputPayload
			if err := json.Unmarshal(outMsg.Payload, &out); err != nil {
				continue
			}

			if out.Output != "" {
				if out.IsStderr {
					fmt.Fprint(os.Stderr, out.Output)
				} else {
					fmt.Print(out.Output)
				}
			}

			if out.Done {
				if out.ExitCode != nil && *out.ExitCode != 0 {
					return fmt.Errorf("command exited with code %d", *out.ExitCode)
				}
				return nil
			}
		} else if outMsg.Type == "error" {
			var errPayload struct {
				Message string `json:"message"`
			}
			json.Unmarshal(outMsg.Payload, &errPayload)
			return fmt.Errorf("error: %s", errPayload.Message)
		}
	}
}

func startInteractiveSession(agentID string) error {
	wsURL, err := client.GetWebSocketURL()
	if err != nil {
		return fmt.Errorf("failed to get WebSocket URL: %w", err)
	}

	// Get auth header
	authHeader, err := client.Auth().GetAuthHeader()
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}

	// Connect with auth
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	headers := http.Header{}
	headers.Set("Authorization", authHeader)

	conn, httpResp, err := dialer.Dial(wsURL, headers)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	defer conn.Close()

	// Read initial agent_list message (sent immediately on connect)
	var initialMsg wsMessage
	if err := conn.ReadJSON(&initialMsg); err != nil {
		return fmt.Errorf("failed to read initial message: %w", err)
	}
	// Ignore agent_list, we already know which agent we want

	// Connect to agent
	connectPayload, _ := json.Marshal(connectAgentPayload{AgentID: agentID})
	msg := wsMessage{
		Type:      "connect_agent",
		Payload:   connectPayload,
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}

	// Wait for agent_connected response
	var resp wsMessage
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if resp.Type == "error" {
		var errPayload struct {
			Message string `json:"message"`
		}
		json.Unmarshal(resp.Payload, &errPayload)
		return fmt.Errorf("failed to connect to agent: %s", errPayload.Message)
	}
	if resp.Type != "agent_connected" {
		return fmt.Errorf("unexpected response: %s", resp.Type)
	}

	fmt.Printf("Connected to %s. Type 'exit' to disconnect.\n\n", agentID)

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start output reader goroutine
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		for {
			var outMsg wsMessage
			if err := conn.ReadJSON(&outMsg); err != nil {
				return
			}

			if outMsg.Type == "output" {
				var out outputPayload
				if err := json.Unmarshal(outMsg.Payload, &out); err != nil {
					continue
				}

				if out.Output != "" {
					if out.IsStderr {
						fmt.Fprint(os.Stderr, out.Output)
					} else {
						fmt.Print(out.Output)
					}
				}

				if out.Done && out.ExitCode != nil {
					fmt.Printf("\n[Exit code: %d]\n", *out.ExitCode)
				}
			} else if outMsg.Type == "error" {
				var errPayload struct {
					Message string `json:"message"`
				}
				json.Unmarshal(outMsg.Payload, &errPayload)
				fmt.Fprintf(os.Stderr, "Error: %s\n", errPayload.Message)
			}
		}
	}()

	// Read commands from stdin
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-sigChan:
			fmt.Println("\nDisconnecting...")
			disconnectMsg := wsMessage{
				Type:      "disconnect",
				Timestamp: time.Now(),
			}
			conn.WriteJSON(disconnectMsg)
			return nil
		case <-outputDone:
			return fmt.Errorf("connection closed")
		default:
		}

		fmt.Print("$ ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		command := strings.TrimSpace(line)
		if command == "" {
			continue
		}

		if command == "exit" || command == "quit" {
			fmt.Println("Disconnecting...")
			disconnectMsg := wsMessage{
				Type:      "disconnect",
				Timestamp: time.Now(),
			}
			conn.WriteJSON(disconnectMsg)
			return nil
		}

		// Send command
		cmdPayload, _ := json.Marshal(commandPayload{Command: command})
		cmdMsg := wsMessage{
			Type:      "command",
			Payload:   cmdPayload,
			Timestamp: time.Now(),
		}
		if err := conn.WriteJSON(cmdMsg); err != nil {
			return fmt.Errorf("failed to send command: %w", err)
		}

		// Wait for command to complete before prompting again
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// === Provider Command ===

func newProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage authentication providers (OIDC and SAML)",
	}

	cmd.AddCommand(newOIDCProviderCmd(), newSAMLProviderCmd())
	return cmd
}

func newOIDCProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oidc",
		Short: "Manage OIDC providers",
	}

	// List OIDC providers
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all OIDC providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			providers, err := client.ListOIDCProviders(ctx)
			if err != nil {
				return err
			}
			return outputResult(providers, []string{"Name", "Display Name", "Issuer", "Client ID", "Enabled"}, func(item interface{}) []string {
				p := item.(adminclient.OIDCProvider)
				enabled := "No"
				if p.Enabled {
					enabled = "Yes"
				}
				return []string{p.Name, p.DisplayName, p.Issuer, p.ClientID, enabled}
			})
		},
	}

	// Get OIDC provider
	getCmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get OIDC provider details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			provider, err := client.GetOIDCProvider(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(provider)
		},
	}

	// Create OIDC provider
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new OIDC provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			displayName, _ := cmd.Flags().GetString("display-name")
			issuer, _ := cmd.Flags().GetString("issuer")
			clientID, _ := cmd.Flags().GetString("client-id")
			clientSecret, _ := cmd.Flags().GetString("client-secret")
			redirectURL, _ := cmd.Flags().GetString("redirect-url")
			scopes, _ := cmd.Flags().GetStringSlice("scopes")
			adminGroup, _ := cmd.Flags().GetString("admin-group")
			enabled, _ := cmd.Flags().GetBool("enabled")

			// Claim mappings
			claimEmail, _ := cmd.Flags().GetString("claim-email")
			claimName, _ := cmd.Flags().GetString("claim-name")
			claimGivenName, _ := cmd.Flags().GetString("claim-given-name")
			claimFamilyName, _ := cmd.Flags().GetString("claim-family-name")
			claimGroups, _ := cmd.Flags().GetString("claim-groups")

			if name == "" || issuer == "" || clientID == "" || clientSecret == "" {
				return fmt.Errorf("--name, --issuer, --client-id, and --client-secret are required")
			}

			if displayName == "" {
				displayName = name
			}

			if len(scopes) == 0 {
				scopes = []string{"openid", "profile", "email"}
			}

			req := map[string]interface{}{
				"name":          name,
				"display_name":  displayName,
				"issuer":        issuer,
				"client_id":     clientID,
				"client_secret": clientSecret,
				"redirect_url":  redirectURL,
				"scopes":        scopes,
				"admin_group":   adminGroup,
				"enabled":       enabled,
				"claim_mappings": map[string]string{
					"email":       claimEmail,
					"name":        claimName,
					"given_name":  claimGivenName,
					"family_name": claimFamilyName,
					"groups":      claimGroups,
				},
			}

			ctx := context.Background()
			provider, err := client.CreateOIDCProvider(ctx, req)
			if err != nil {
				return err
			}
			fmt.Printf("OIDC provider created: %s\n", provider.Name)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Provider name (required)")
	createCmd.Flags().String("display-name", "", "Display name")
	createCmd.Flags().String("issuer", "", "OIDC issuer URL (required)")
	createCmd.Flags().String("client-id", "", "Client ID (required)")
	createCmd.Flags().String("client-secret", "", "Client secret (required)")
	createCmd.Flags().String("redirect-url", "", "Redirect URL")
	createCmd.Flags().StringSlice("scopes", []string{"openid", "profile", "email"}, "OAuth scopes")
	createCmd.Flags().String("admin-group", "", "Admin group name")
	createCmd.Flags().Bool("enabled", true, "Enable provider")
	createCmd.Flags().String("claim-email", "email", "Email claim name")
	createCmd.Flags().String("claim-name", "name", "Name claim name")
	createCmd.Flags().String("claim-given-name", "given_name", "Given name claim name")
	createCmd.Flags().String("claim-family-name", "family_name", "Family name claim name")
	createCmd.Flags().String("claim-groups", "groups", "Groups claim name")

	// Update OIDC provider
	updateCmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update an OIDC provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})

			if displayName, _ := cmd.Flags().GetString("display-name"); displayName != "" {
				req["display_name"] = displayName
			}
			if issuer, _ := cmd.Flags().GetString("issuer"); issuer != "" {
				req["issuer"] = issuer
			}
			if clientID, _ := cmd.Flags().GetString("client-id"); clientID != "" {
				req["client_id"] = clientID
			}
			if clientSecret, _ := cmd.Flags().GetString("client-secret"); clientSecret != "" {
				req["client_secret"] = clientSecret
			}
			if redirectURL, _ := cmd.Flags().GetString("redirect-url"); redirectURL != "" {
				req["redirect_url"] = redirectURL
			}
			if scopes, _ := cmd.Flags().GetStringSlice("scopes"); len(scopes) > 0 {
				req["scopes"] = scopes
			}
			if adminGroup, _ := cmd.Flags().GetString("admin-group"); cmd.Flags().Changed("admin-group") {
				req["admin_group"] = adminGroup
			}
			if cmd.Flags().Changed("enabled") {
				enabled, _ := cmd.Flags().GetBool("enabled")
				req["enabled"] = enabled
			}

			// Check if any claim mapping flags were changed
			claimMappings := make(map[string]string)
			if cmd.Flags().Changed("claim-email") {
				claimEmail, _ := cmd.Flags().GetString("claim-email")
				claimMappings["email"] = claimEmail
			}
			if cmd.Flags().Changed("claim-name") {
				claimName, _ := cmd.Flags().GetString("claim-name")
				claimMappings["name"] = claimName
			}
			if cmd.Flags().Changed("claim-given-name") {
				claimGivenName, _ := cmd.Flags().GetString("claim-given-name")
				claimMappings["given_name"] = claimGivenName
			}
			if cmd.Flags().Changed("claim-family-name") {
				claimFamilyName, _ := cmd.Flags().GetString("claim-family-name")
				claimMappings["family_name"] = claimFamilyName
			}
			if cmd.Flags().Changed("claim-groups") {
				claimGroups, _ := cmd.Flags().GetString("claim-groups")
				claimMappings["groups"] = claimGroups
			}
			if len(claimMappings) > 0 {
				req["claim_mappings"] = claimMappings
			}

			ctx := context.Background()
			provider, err := client.UpdateOIDCProvider(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("OIDC provider updated: %s\n", provider.Name)
			return nil
		},
	}
	updateCmd.Flags().String("display-name", "", "Display name")
	updateCmd.Flags().String("issuer", "", "OIDC issuer URL")
	updateCmd.Flags().String("client-id", "", "Client ID")
	updateCmd.Flags().String("client-secret", "", "Client secret")
	updateCmd.Flags().String("redirect-url", "", "Redirect URL")
	updateCmd.Flags().StringSlice("scopes", nil, "OAuth scopes")
	updateCmd.Flags().String("admin-group", "", "Admin group name")
	updateCmd.Flags().Bool("enabled", true, "Enable provider")
	updateCmd.Flags().String("claim-email", "", "Email claim name")
	updateCmd.Flags().String("claim-name", "", "Name claim name")
	updateCmd.Flags().String("claim-given-name", "", "Given name claim name")
	updateCmd.Flags().String("claim-family-name", "", "Family name claim name")
	updateCmd.Flags().String("claim-groups", "", "Groups claim name")

	// Delete OIDC provider
	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete an OIDC provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteOIDCProvider(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("OIDC provider deleted")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return cmd
}

func newSAMLProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saml",
		Short: "Manage SAML providers",
	}

	// List SAML providers
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all SAML providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			providers, err := client.ListSAMLProviders(ctx)
			if err != nil {
				return err
			}
			return outputResult(providers, []string{"Name", "Display Name", "Entity ID", "IDP Metadata URL", "Enabled"}, func(item interface{}) []string {
				p := item.(adminclient.SAMLProvider)
				enabled := "No"
				if p.Enabled {
					enabled = "Yes"
				}
				return []string{p.Name, p.DisplayName, p.EntityID, p.IDPMetadataURL, enabled}
			})
		},
	}

	// Get SAML provider
	getCmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get SAML provider details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			provider, err := client.GetSAMLProvider(ctx, args[0])
			if err != nil {
				return err
			}
			return outputSingle(provider)
		},
	}

	// Create SAML provider
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new SAML provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			displayName, _ := cmd.Flags().GetString("display-name")
			idpMetadataURL, _ := cmd.Flags().GetString("idp-metadata-url")
			entityID, _ := cmd.Flags().GetString("entity-id")
			acsURL, _ := cmd.Flags().GetString("acs-url")
			adminGroup, _ := cmd.Flags().GetString("admin-group")
			enabled, _ := cmd.Flags().GetBool("enabled")

			// Attribute mappings
			attrEmail, _ := cmd.Flags().GetString("attr-email")
			attrName, _ := cmd.Flags().GetString("attr-name")
			attrGivenName, _ := cmd.Flags().GetString("attr-given-name")
			attrFamilyName, _ := cmd.Flags().GetString("attr-family-name")
			attrGroups, _ := cmd.Flags().GetString("attr-groups")

			if name == "" || idpMetadataURL == "" || entityID == "" {
				return fmt.Errorf("--name, --idp-metadata-url, and --entity-id are required")
			}

			if displayName == "" {
				displayName = name
			}

			req := map[string]interface{}{
				"name":             name,
				"display_name":     displayName,
				"idp_metadata_url": idpMetadataURL,
				"entity_id":        entityID,
				"acs_url":          acsURL,
				"admin_group":      adminGroup,
				"enabled":          enabled,
				"attribute_mappings": map[string]string{
					"email":       attrEmail,
					"name":        attrName,
					"given_name":  attrGivenName,
					"family_name": attrFamilyName,
					"groups":      attrGroups,
				},
			}

			ctx := context.Background()
			provider, err := client.CreateSAMLProvider(ctx, req)
			if err != nil {
				return err
			}
			fmt.Printf("SAML provider created: %s\n", provider.Name)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Provider name (required)")
	createCmd.Flags().String("display-name", "", "Display name")
	createCmd.Flags().String("idp-metadata-url", "", "IDP Metadata URL (required)")
	createCmd.Flags().String("entity-id", "", "Entity ID (required)")
	createCmd.Flags().String("acs-url", "", "Assertion Consumer Service URL")
	createCmd.Flags().String("admin-group", "", "Admin group name")
	createCmd.Flags().Bool("enabled", true, "Enable provider")
	createCmd.Flags().String("attr-email", "email", "Email attribute name")
	createCmd.Flags().String("attr-name", "displayName", "Name attribute name")
	createCmd.Flags().String("attr-given-name", "givenName", "Given name attribute name")
	createCmd.Flags().String("attr-family-name", "sn", "Family name attribute name")
	createCmd.Flags().String("attr-groups", "groups", "Groups attribute name")

	// Update SAML provider
	updateCmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update a SAML provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := make(map[string]interface{})

			if displayName, _ := cmd.Flags().GetString("display-name"); displayName != "" {
				req["display_name"] = displayName
			}
			if idpMetadataURL, _ := cmd.Flags().GetString("idp-metadata-url"); idpMetadataURL != "" {
				req["idp_metadata_url"] = idpMetadataURL
			}
			if entityID, _ := cmd.Flags().GetString("entity-id"); entityID != "" {
				req["entity_id"] = entityID
			}
			if acsURL, _ := cmd.Flags().GetString("acs-url"); acsURL != "" {
				req["acs_url"] = acsURL
			}
			if adminGroup, _ := cmd.Flags().GetString("admin-group"); cmd.Flags().Changed("admin-group") {
				req["admin_group"] = adminGroup
			}
			if cmd.Flags().Changed("enabled") {
				enabled, _ := cmd.Flags().GetBool("enabled")
				req["enabled"] = enabled
			}

			// Check if any attribute mapping flags were changed
			attrMappings := make(map[string]string)
			if cmd.Flags().Changed("attr-email") {
				attrEmail, _ := cmd.Flags().GetString("attr-email")
				attrMappings["email"] = attrEmail
			}
			if cmd.Flags().Changed("attr-name") {
				attrName, _ := cmd.Flags().GetString("attr-name")
				attrMappings["name"] = attrName
			}
			if cmd.Flags().Changed("attr-given-name") {
				attrGivenName, _ := cmd.Flags().GetString("attr-given-name")
				attrMappings["given_name"] = attrGivenName
			}
			if cmd.Flags().Changed("attr-family-name") {
				attrFamilyName, _ := cmd.Flags().GetString("attr-family-name")
				attrMappings["family_name"] = attrFamilyName
			}
			if cmd.Flags().Changed("attr-groups") {
				attrGroups, _ := cmd.Flags().GetString("attr-groups")
				attrMappings["groups"] = attrGroups
			}
			if len(attrMappings) > 0 {
				req["attribute_mappings"] = attrMappings
			}

			ctx := context.Background()
			provider, err := client.UpdateSAMLProvider(ctx, args[0], req)
			if err != nil {
				return err
			}
			fmt.Printf("SAML provider updated: %s\n", provider.Name)
			return nil
		},
	}
	updateCmd.Flags().String("display-name", "", "Display name")
	updateCmd.Flags().String("idp-metadata-url", "", "IDP Metadata URL")
	updateCmd.Flags().String("entity-id", "", "Entity ID")
	updateCmd.Flags().String("acs-url", "", "Assertion Consumer Service URL")
	updateCmd.Flags().String("admin-group", "", "Admin group name")
	updateCmd.Flags().Bool("enabled", true, "Enable provider")
	updateCmd.Flags().String("attr-email", "", "Email attribute name")
	updateCmd.Flags().String("attr-name", "", "Name attribute name")
	updateCmd.Flags().String("attr-given-name", "", "Given name attribute name")
	updateCmd.Flags().String("attr-family-name", "", "Family name attribute name")
	updateCmd.Flags().String("attr-groups", "", "Groups attribute name")

	// Delete SAML provider
	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a SAML provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := client.DeleteSAMLProvider(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("SAML provider deleted")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return cmd
}

// === Version Command ===

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("GateKey Admin CLI\n")
			fmt.Printf("  Version:    %s\n", version)
			fmt.Printf("  Commit:     %s\n", commit)
			fmt.Printf("  Build Date: %s\n", buildDate)
		},
	}
}

// === Output Helpers ===

func outputResult[T any](items []T, headers []string, rowFunc func(interface{}) []string) error {
	switch outputFormat {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(items)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(items)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(headers, "\t"))
		for _, item := range items {
			fmt.Fprintln(w, strings.Join(rowFunc(item), "\t"))
		}
		return w.Flush()
	}
}

func outputSingle(item interface{}) error {
	switch outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(item)
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}
}

func parseDuration(s string) (time.Duration, error) {
	// Handle custom formats like "30d", "90d", "1y"
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 'd':
		multiplier = 24 * time.Hour
	case 'w':
		multiplier = 7 * 24 * time.Hour
	case 'm':
		multiplier = 30 * 24 * time.Hour
	case 'y':
		multiplier = 365 * 24 * time.Hour
	default:
		// Try standard duration parsing
		return time.ParseDuration(s)
	}

	var num int
	if _, err := fmt.Sscanf(value, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", value)
	}

	return time.Duration(num) * multiplier, nil
}
