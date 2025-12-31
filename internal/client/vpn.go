package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// VPNManager handles OpenVPN process management.
type VPNManager struct {
	config *Config
	auth   *AuthManager
}

// ConnectionState holds the current VPN connection state.
type ConnectionState struct {
	Connected    bool      `json:"connected"`
	Gateway      string    `json:"gateway,omitempty"`
	GatewayID    string    `json:"gateway_id,omitempty"`
	ConnectedAt  time.Time `json:"connected_at,omitempty"`
	LocalIP      string    `json:"local_ip,omitempty"`
	RemoteIP     string    `json:"remote_ip,omitempty"`
	BytesIn      int64     `json:"bytes_in,omitempty"`
	BytesOut     int64     `json:"bytes_out,omitempty"`
	PID          int       `json:"pid,omitempty"`
	TunInterface string    `json:"tun_interface,omitempty"`
}

// MultiConnectionState holds multiple VPN connection states.
type MultiConnectionState struct {
	Connections map[string]*ConnectionState `json:"connections"`
}

// Gateway represents a VPN gateway from the server.
type Gateway struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Status      string `json:"status"`
}

// NewVPNManager creates a new VPN manager.
func NewVPNManager(config *Config) *VPNManager {
	return &VPNManager{
		config: config,
		auth:   NewAuthManager(config),
	}
}

// Connect connects to a VPN gateway. Multiple gateways can be connected simultaneously.
func (v *VPNManager) Connect(ctx context.Context, gatewayName string) error {
	// Load existing multi-connection state
	multiState := v.loadMultiState()

	// Check if already connected to this specific gateway
	if gatewayName != "" {
		if conn, exists := multiState.Connections[gatewayName]; exists && conn.Connected {
			if v.isProcessRunning(conn.PID) {
				return fmt.Errorf("already connected to %s. Run 'gatekey disconnect %s' first", gatewayName, gatewayName)
			}
			// Process died, clean up stale connection
			delete(multiState.Connections, gatewayName)
		}
	}

	// Clean up any stale connections (processes that died)
	v.cleanupStaleConnections(multiState)

	// Ensure we're logged in
	token, err := v.auth.GetToken()
	if err != nil {
		return fmt.Errorf("authentication required: %w\nRun 'gatekey login' to authenticate", err)
	}

	// Check server FIPS requirements
	if err := v.checkServerFIPSRequirement(ctx, token); err != nil {
		return err
	}

	// Get available gateways
	gateways, err := v.fetchGateways(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to fetch gateways: %w", err)
	}

	if len(gateways) == 0 {
		return fmt.Errorf("no gateways available")
	}

	// Select gateway
	var selectedGateway *Gateway
	if gatewayName == "" {
		if len(gateways) == 1 {
			selectedGateway = &gateways[0]
		} else {
			return v.promptGatewaySelection(gateways)
		}
	} else {
		for i := range gateways {
			if strings.EqualFold(gateways[i].Name, gatewayName) || gateways[i].ID == gatewayName {
				selectedGateway = &gateways[i]
				break
			}
		}
		if selectedGateway == nil {
			return fmt.Errorf("gateway '%s' not found", gatewayName)
		}
	}

	// Check if already connected to this gateway (by name match after selection)
	if conn, exists := multiState.Connections[selectedGateway.Name]; exists && conn.Connected {
		if v.isProcessRunning(conn.PID) {
			return fmt.Errorf("already connected to %s", selectedGateway.Name)
		}
	}

	fmt.Printf("Connecting to %s...\n", selectedGateway.Name)

	// Find an available tun interface number
	tunNum := v.findAvailableTunNumber(multiState)
	tunInterface := fmt.Sprintf("tun%d", tunNum)

	// Download VPN configuration to gateway-specific path
	configPath, err := v.downloadConfigForGateway(ctx, token, selectedGateway.ID, selectedGateway.Name)
	if err != nil {
		return fmt.Errorf("failed to download VPN configuration: %w", err)
	}

	// Start OpenVPN with specific tun interface
	pid, err := v.startOpenVPNForGateway(configPath, selectedGateway.Name, tunInterface)
	if err != nil {
		return fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	// Save connection state
	conn := &ConnectionState{
		Connected:    true,
		Gateway:      selectedGateway.Name,
		GatewayID:    selectedGateway.ID,
		ConnectedAt:  time.Now(),
		PID:          pid,
		TunInterface: tunInterface,
	}
	multiState.Connections[selectedGateway.Name] = conn

	if err := v.saveMultiState(multiState); err != nil {
		// Kill the process if we can't save state
		if proc, err := os.FindProcess(pid); err == nil {
			proc.Kill()
		}
		return fmt.Errorf("failed to save connection state: %w", err)
	}

	fmt.Printf("Connected to %s (PID: %d, Interface: %s)\n", selectedGateway.Name, pid, tunInterface)
	fmt.Println("VPN connection established. Use 'gatekey status' to check connection.")
	return nil
}

// findAvailableTunNumber finds the next available tun interface number.
func (v *VPNManager) findAvailableTunNumber(multiState *MultiConnectionState) int {
	used := make(map[int]bool)

	// Mark tun numbers used by active connections
	for _, conn := range multiState.Connections {
		if conn.Connected && conn.TunInterface != "" {
			var num int
			if _, err := fmt.Sscanf(conn.TunInterface, "tun%d", &num); err == nil {
				used[num] = true
			}
		}
	}

	// Also check what tun interfaces actually exist on the system
	if data, err := os.ReadFile("/proc/net/dev"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "tun") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) >= 1 {
					var num int
					if _, err := fmt.Sscanf(parts[0], "tun%d", &num); err == nil {
						used[num] = true
					}
				}
			}
		}
	}

	// Find the first available number
	for i := 0; i < 100; i++ {
		if !used[i] {
			return i
		}
	}
	return 0 // Fallback
}

// cleanupStaleConnections removes connections whose processes have died.
func (v *VPNManager) cleanupStaleConnections(multiState *MultiConnectionState) {
	for name, conn := range multiState.Connections {
		if conn.Connected && !v.isProcessRunning(conn.PID) {
			// Process died, clean up
			if conn.TunInterface != "" {
				exec.Command("sudo", "ip", "link", "delete", conn.TunInterface).Run()
			}
			delete(multiState.Connections, name)
		}
	}
}

// Disconnect disconnects from a VPN gateway. If gatewayName is empty, disconnects from all.
func (v *VPNManager) Disconnect() error {
	return v.DisconnectGateway("")
}

// DisconnectGateway disconnects from a specific gateway or all if gatewayName is empty.
func (v *VPNManager) DisconnectGateway(gatewayName string) error {
	multiState := v.loadMultiState()

	if len(multiState.Connections) == 0 {
		return fmt.Errorf("not connected to any gateway")
	}

	// If no gateway specified, disconnect from all
	if gatewayName == "" {
		return v.disconnectAll(multiState)
	}

	// Find the specific connection
	conn, exists := multiState.Connections[gatewayName]
	if !exists {
		// Try case-insensitive match
		for name, c := range multiState.Connections {
			if strings.EqualFold(name, gatewayName) {
				gatewayName = name
				conn = c
				exists = true
				break
			}
		}
	}

	if !exists {
		return fmt.Errorf("not connected to gateway '%s'", gatewayName)
	}

	// Disconnect this specific gateway
	v.disconnectSingle(conn, gatewayName)

	// Remove from state
	delete(multiState.Connections, gatewayName)
	v.saveMultiState(multiState)

	fmt.Printf("Disconnected from %s\n", gatewayName)
	return nil
}

// disconnectAll disconnects from all gateways.
func (v *VPNManager) disconnectAll(multiState *MultiConnectionState) error {
	if len(multiState.Connections) == 0 {
		return fmt.Errorf("not connected to any gateway")
	}

	var disconnected []string
	for name, conn := range multiState.Connections {
		v.disconnectSingle(conn, name)
		disconnected = append(disconnected, name)
	}

	// Clear all connections
	multiState.Connections = make(map[string]*ConnectionState)
	v.saveMultiState(multiState)

	if len(disconnected) == 1 {
		fmt.Printf("Disconnected from %s\n", disconnected[0])
	} else {
		fmt.Printf("Disconnected from %d gateways: %s\n", len(disconnected), strings.Join(disconnected, ", "))
	}
	return nil
}

// disconnectSingle disconnects from a single gateway.
func (v *VPNManager) disconnectSingle(conn *ConnectionState, gatewayName string) {
	// Kill the OpenVPN process
	if conn.PID > 0 {
		proc, err := os.FindProcess(conn.PID)
		if err == nil {
			// Send SIGTERM first for graceful shutdown
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				if !strings.Contains(err.Error(), "process already finished") {
					proc.Kill()
				}
			} else {
				time.Sleep(1 * time.Second)
				if v.isProcessRunning(conn.PID) {
					proc.Kill()
				}
			}
		}
	}

	// Also try to kill by gateway-specific PID file
	pidPath := v.config.GatewayPidPath(gatewayName)
	if pidData, err := os.ReadFile(pidPath); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				proc.Signal(syscall.SIGTERM)
			}
		}
	}

	// Wait for OpenVPN to clean up
	time.Sleep(500 * time.Millisecond)

	// Clean up the specific tun interface
	if conn.TunInterface != "" {
		exec.Command("sudo", "ip", "link", "delete", conn.TunInterface).Run()
	}

	// Clean up gateway-specific files
	os.Remove(v.config.GatewayPidPath(gatewayName))
	os.Remove(v.config.GatewayConfigPath(gatewayName))
	// Keep the log file for debugging
}

// cleanupTunInterfaces removes stale tun interfaces, preserving those used by active connections.
func (v *VPNManager) cleanupTunInterfaces() {
	// Load current connections to know which interfaces to preserve
	multiState := v.loadMultiState()
	activeInterfaces := make(map[string]bool)
	for _, conn := range multiState.Connections {
		if conn.Connected && conn.TunInterface != "" && v.isProcessRunning(conn.PID) {
			activeInterfaces[conn.TunInterface] = true
		}
	}

	// Read network interfaces
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "tun") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 1 {
				ifName := strings.TrimSpace(parts[0])
				// Only delete if not used by an active connection
				if !activeInterfaces[ifName] {
					exec.Command("sudo", "ip", "link", "delete", ifName).Run()
				}
			}
		}
	}
}

// loadMultiState loads the multi-connection state from disk.
func (v *VPNManager) loadMultiState() *MultiConnectionState {
	multiState := &MultiConnectionState{
		Connections: make(map[string]*ConnectionState),
	}

	data, err := os.ReadFile(v.config.StateFilePath())
	if err != nil {
		return multiState
	}

	// Try to load as multi-connection state first
	if err := json.Unmarshal(data, multiState); err == nil && len(multiState.Connections) > 0 {
		return multiState
	}

	// Fall back to legacy single connection state
	// Reset connections map since the unmarshal above may have modified it
	multiState.Connections = make(map[string]*ConnectionState)
	var legacyState ConnectionState
	if err := json.Unmarshal(data, &legacyState); err == nil && legacyState.Connected && legacyState.Gateway != "" {
		multiState.Connections[legacyState.Gateway] = &legacyState
	}

	return multiState
}

// saveMultiState saves the multi-connection state to disk.
func (v *VPNManager) saveMultiState(state *MultiConnectionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(v.config.StateFilePath(), data, 0600)
}

// downloadConfigForGateway downloads the VPN config to a gateway-specific path.
func (v *VPNManager) downloadConfigForGateway(ctx context.Context, token *TokenData, gatewayID, gatewayName string) (string, error) {
	configPath := v.config.GatewayConfigPath(gatewayName)
	client := &http.Client{Timeout: 60 * time.Second}

	// Step 1: Generate config and get download URL
	reqURL := fmt.Sprintf("%s/api/v1/configs/generate", v.config.ServerURL)
	reqBody := fmt.Sprintf(`{"gateway_id": "%s"}`, gatewayID)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var configResp struct {
		ID          string `json:"id"`
		DownloadURL string `json:"downloadUrl"`
		FileName    string `json:"fileName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Step 2: Download the actual config file
	downloadURL := fmt.Sprintf("%s%s", v.config.ServerURL, configResp.DownloadURL)
	downloadReq, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", err
	}
	downloadReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		return "", err
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		return "", fmt.Errorf("download failed with %d: %s", downloadResp.StatusCode, string(body))
	}

	configData, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	return configPath, nil
}

// checkServerFIPSRequirement checks if the server requires FIPS mode and verifies compliance.
func (v *VPNManager) checkServerFIPSRequirement(ctx context.Context, token *TokenData) error {
	reqURL := fmt.Sprintf("%s/api/v1/server/info", v.config.ServerURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil // Don't fail if we can't check
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil // Don't fail if server doesn't support this endpoint
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // Old server version, skip check
	}

	var serverInfo struct {
		RequireFIPS bool   `json:"require_fips"`
		Version     string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&serverInfo); err != nil {
		return nil // Skip if we can't parse
	}

	if serverInfo.RequireFIPS {
		if !IsFIPSCompliant() {
			fmt.Println()
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println("  FIPS 140-3 COMPLIANCE REQUIRED")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("  This server requires FIPS 140-3 compliance for all")
			fmt.Println("  VPN connections, but your system is not FIPS-compliant.")
			fmt.Println()
			fmt.Println("  To enable FIPS mode on your system:")
			fmt.Println("    1. Install FIPS packages for your distribution")
			fmt.Println("    2. Run: sudo fips-mode-setup --enable")
			fmt.Println("    3. Reboot your system")
			fmt.Println()
			fmt.Println("  Run 'gatekey fips-check' for detailed compliance status.")
			fmt.Println()
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			return fmt.Errorf("FIPS 140-3 compliance required but system is not compliant")
		}
		fmt.Println("✓ FIPS 140-3 compliance verified")
	}

	return nil
}

// startOpenVPNForGateway starts OpenVPN for a specific gateway with a specific tun interface.
func (v *VPNManager) startOpenVPNForGateway(configPath, gatewayName, tunInterface string) (int, error) {
	openvpnPath, err := exec.LookPath(v.config.OpenVPNBinary)
	if err != nil {
		return 0, fmt.Errorf("OpenVPN not found. Please install OpenVPN and ensure it's in your PATH")
	}

	logPath := v.config.GatewayLogPath(gatewayName)
	pidPath := v.config.GatewayPidPath(gatewayName)

	// Pre-create log file with readable permissions
	if err := os.WriteFile(logPath, []byte{}, 0644); err != nil {
		// Not fatal
	}

	args := []string{
		"--config", configPath,
		"--daemon",
		"--writepid", pidPath,
		"--log-append", logPath,
		"--dev", tunInterface,
		"--verb", "3",
	}

	needsSudo := os.Geteuid() != 0

	var cmd *exec.Cmd
	if needsSudo {
		fmt.Println("OpenVPN requires root privileges. You may be prompted for your password.")
		sudoArgs := append([]string{openvpnPath}, args...)
		cmd = exec.Command("sudo", sudoArgs...)
	} else {
		cmd = exec.Command(openvpnPath, args...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if logData, readErr := os.ReadFile(logPath); readErr == nil && len(logData) > 0 {
			return 0, fmt.Errorf("OpenVPN failed: %s", string(logData))
		}
		return 0, fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	time.Sleep(1 * time.Second)

	pidData, err := os.ReadFile(pidPath)
	if err != nil {
		if logData, readErr := os.ReadFile(logPath); readErr == nil && len(logData) > 0 {
			lines := strings.Split(string(logData), "\n")
			lastLines := lines
			if len(lines) > 5 {
				lastLines = lines[len(lines)-5:]
			}
			return 0, fmt.Errorf("OpenVPN failed to start. Log:\n%s", strings.Join(lastLines, "\n"))
		}
		return 0, fmt.Errorf("OpenVPN started but couldn't determine PID")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	// Make log file readable
	if needsSudo {
		exec.Command("sudo", "chmod", "644", logPath).Run()
	}

	return pid, nil
}

// Status shows the current connection status for all gateways.
func (v *VPNManager) Status(jsonOutput bool) error {
	multiState := v.loadMultiState()
	v.cleanupStaleConnections(multiState)

	// Count active connections
	activeCount := 0
	for _, conn := range multiState.Connections {
		if conn.Connected && v.isProcessRunning(conn.PID) {
			activeCount++
		}
	}

	if activeCount == 0 {
		if jsonOutput {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"connected":   false,
				"connections": []interface{}{},
			})
		} else {
			fmt.Println("Status: Disconnected")
		}
		return nil
	}

	if jsonOutput {
		connections := make([]*ConnectionState, 0)
		for _, conn := range multiState.Connections {
			if conn.Connected && v.isProcessRunning(conn.PID) {
				v.updateStateFromLogForGateway(conn)
				connections = append(connections, conn)
			}
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"connected":   true,
			"connections": connections,
		})
	}

	// Text output
	if activeCount == 1 {
		// Single connection - show detailed view
		for _, conn := range multiState.Connections {
			if conn.Connected && v.isProcessRunning(conn.PID) {
				v.showSingleConnectionStatus(conn)
				break
			}
		}
	} else {
		// Multiple connections - show summary
		fmt.Printf("Status: Connected to %d gateways\n\n", activeCount)
		for name, conn := range multiState.Connections {
			if conn.Connected && v.isProcessRunning(conn.PID) {
				v.updateStateFromLogForGateway(conn)
				tunnelStatus := v.checkTunnelStatusForGateway(name)
				statusStr := "Connected"
				if tunnelStatus == "connecting" {
					statusStr = "Connecting"
				} else if tunnelStatus == "failed" {
					statusStr = "Failed"
				}
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    Status:    %s\n", statusStr)
				fmt.Printf("    Interface: %s\n", conn.TunInterface)
				if conn.LocalIP != "" {
					fmt.Printf("    Local IP:  %s\n", conn.LocalIP)
				}
				fmt.Printf("    Duration:  %s\n", time.Since(conn.ConnectedAt).Round(time.Second))
				fmt.Printf("    PID:       %d\n", conn.PID)
				fmt.Println()
			}
		}
	}

	return nil
}

// showSingleConnectionStatus shows detailed status for a single connection.
func (v *VPNManager) showSingleConnectionStatus(conn *ConnectionState) {
	v.updateStateFromLogForGateway(conn)
	tunnelStatus := v.checkTunnelStatusForGateway(conn.Gateway)

	switch tunnelStatus {
	case "connected":
		fmt.Println("Status: Connected")
	case "connecting":
		fmt.Println("Status: Connecting (tunnel not yet established)")
		fmt.Printf("Gateway:      %s\n", conn.Gateway)
		fmt.Printf("Interface:    %s\n", conn.TunInterface)
		fmt.Printf("PID:          %d\n", conn.PID)
		logPath := v.config.GatewayLogPath(conn.Gateway)
		fmt.Printf("\nCheck logs: sudo tail -f %s\n", logPath)
		return
	case "failed":
		fmt.Println("Status: Connection Failed")
		fmt.Printf("Gateway:      %s\n", conn.Gateway)
		fmt.Printf("Interface:    %s\n", conn.TunInterface)
		fmt.Printf("PID:          %d\n", conn.PID)
		logPath := v.config.GatewayLogPath(conn.Gateway)
		fmt.Printf("\nCheck logs: sudo cat %s | tail -20\n", logPath)
		return
	}

	fmt.Printf("Gateway:      %s\n", conn.Gateway)
	fmt.Printf("Interface:    %s\n", conn.TunInterface)
	fmt.Printf("Connected at: %s\n", conn.ConnectedAt.Format(time.RFC3339))
	fmt.Printf("Duration:     %s\n", time.Since(conn.ConnectedAt).Round(time.Second))
	if conn.LocalIP != "" {
		fmt.Printf("Local IP:     %s\n", conn.LocalIP)
	}
	if conn.RemoteIP != "" {
		fmt.Printf("Remote IP:    %s\n", conn.RemoteIP)
	}
	fmt.Printf("PID:          %d\n", conn.PID)

	routes := v.getRoutesFromGatewayConfig(conn.Gateway)
	if len(routes) > 0 {
		fmt.Println("\nRoutes:")
		for _, route := range routes {
			fmt.Printf("  %s\n", route)
		}
	}
}

// updateStateFromLogForGateway updates state from gateway-specific log.
func (v *VPNManager) updateStateFromLogForGateway(conn *ConnectionState) {
	logPath := v.config.GatewayLogPath(conn.Gateway)
	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "ifconfig") && strings.Contains(line, conn.TunInterface) {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "ifconfig" && i+1 < len(parts) {
					conn.LocalIP = parts[i+1]
				}
			}
		}
	}
}

// checkTunnelStatusForGateway checks tunnel status for a specific gateway.
func (v *VPNManager) checkTunnelStatusForGateway(gatewayName string) string {
	logPath := v.config.GatewayLogPath(gatewayName)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "connecting"
	}

	logContent := string(data)

	if strings.Contains(logContent, "Initialization Sequence Completed") {
		lines := strings.Split(logContent, "\n")
		lastInitIndex := -1
		lastErrorIndex := -1

		for i, line := range lines {
			if strings.Contains(line, "Initialization Sequence Completed") {
				lastInitIndex = i
			}
			if strings.Contains(line, "SIGTERM") ||
				strings.Contains(line, "SIGUSR1") ||
				strings.Contains(line, "Connection reset") ||
				strings.Contains(line, "TLS Error") ||
				strings.Contains(line, "AUTH_FAILED") {
				lastErrorIndex = i
			}
		}

		if lastInitIndex > lastErrorIndex {
			return "connected"
		}
	}

	if strings.Contains(logContent, "EHOSTUNREACH") ||
		strings.Contains(logContent, "Connection refused") ||
		strings.Contains(logContent, "Connection timed out") ||
		strings.Contains(logContent, "No route to host") ||
		strings.Contains(logContent, "AUTH_FAILED") ||
		strings.Contains(logContent, "TLS Error") {
		lines := strings.Split(logContent, "\n")
		recentLines := lines
		if len(lines) > 20 {
			recentLines = lines[len(lines)-20:]
		}
		for _, line := range recentLines {
			if strings.Contains(line, "EHOSTUNREACH") ||
				strings.Contains(line, "No route to host") ||
				strings.Contains(line, "Connection refused") ||
				strings.Contains(line, "AUTH_FAILED") {
				return "failed"
			}
		}
	}

	return "connecting"
}

// getRoutesFromGatewayConfig extracts routes from a gateway-specific config.
func (v *VPNManager) getRoutesFromGatewayConfig(gatewayName string) []string {
	configPath := v.config.GatewayConfigPath(gatewayName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var routes []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "route ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				cidr := netmaskToCIDR(parts[2])
				routes = append(routes, fmt.Sprintf("%s/%s", parts[1], cidr))
			}
		}
	}
	return routes
}

// getRoutesFromConfig extracts route lines from the current OpenVPN config
func (v *VPNManager) getRoutesFromConfig() []string {
	configPath := v.config.CurrentConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var routes []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "route ") {
			// Parse route line: "route 192.168.50.0 255.255.255.0"
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Convert netmask to CIDR
				cidr := netmaskToCIDR(parts[2])
				routes = append(routes, fmt.Sprintf("%s/%s", parts[1], cidr))
			}
		}
	}
	return routes
}

// netmaskToCIDR converts dotted decimal netmask to CIDR notation
func netmaskToCIDR(netmask string) string {
	parts := strings.Split(netmask, ".")
	if len(parts) != 4 {
		return netmask
	}

	bits := 0
	for _, part := range parts {
		var n int
		fmt.Sscanf(part, "%d", &n)
		for n > 0 {
			bits += n & 1
			n >>= 1
		}
	}
	return fmt.Sprintf("%d", bits)
}

// ListGateways lists available gateways.
func (v *VPNManager) ListGateways(ctx context.Context) error {
	token, err := v.auth.GetToken()
	if err != nil {
		return fmt.Errorf("authentication required: %w\nRun 'gatekey login' to authenticate", err)
	}

	gateways, err := v.fetchGateways(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to fetch gateways: %w", err)
	}

	if len(gateways) == 0 {
		fmt.Println("No gateways available.")
		return nil
	}

	fmt.Println("Available Gateways:")
	fmt.Println("-------------------")
	for _, gw := range gateways {
		statusIcon := "✓"
		if gw.Status != "online" {
			statusIcon = "✗"
		}
		fmt.Printf("%s %s\n", statusIcon, gw.Name)
		if gw.Description != "" {
			fmt.Printf("  Description: %s\n", gw.Description)
		}
		if gw.Location != "" {
			fmt.Printf("  Location:    %s\n", gw.Location)
		}
		fmt.Printf("  Hostname:    %s\n", gw.Hostname)
		fmt.Printf("  Status:      %s\n", gw.Status)
		fmt.Println()
	}

	return nil
}

// fetchGateways retrieves the list of available gateways from the server.
func (v *VPNManager) fetchGateways(ctx context.Context, token *TokenData) ([]Gateway, error) {
	gatewaysURL, err := url.Parse(v.config.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}
	gatewaysURL.Path = "/api/v1/gateways"

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewaysURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication expired. Run 'gatekey login' to re-authenticate")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Gateways []Gateway `json:"gateways"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Gateways, nil
}

// downloadConfig downloads the VPN configuration for a gateway.
func (v *VPNManager) downloadConfig(ctx context.Context, token *TokenData, gatewayID string) (string, error) {
	configURL, err := url.Parse(v.config.ServerURL)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}
	configURL.Path = "/api/v1/configs/generate"

	client := &http.Client{Timeout: 60 * time.Second}

	// Step 1: Generate config and get metadata
	body := fmt.Sprintf(`{"gateway_id":"%s"}`, gatewayID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, configURL.String(), strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("authentication expired. Run 'gatekey login' to re-authenticate")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response to get download URL
	var configResp struct {
		ID          string `json:"id"`
		DownloadURL string `json:"downloadUrl"`
		FileName    string `json:"fileName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return "", fmt.Errorf("failed to parse config response: %w", err)
	}

	// Step 2: Download the actual config file
	downloadURL, err := url.Parse(v.config.ServerURL)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}
	downloadURL.Path = configResp.DownloadURL

	downloadReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	downloadReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		return "", fmt.Errorf("download request failed: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		return "", fmt.Errorf("download failed with %d: %s", downloadResp.StatusCode, string(body))
	}

	// Read config content
	configData, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	// Save to file
	configPath := v.config.CurrentConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	return configPath, nil
}

// startOpenVPN starts the OpenVPN process with the given configuration.
func (v *VPNManager) startOpenVPN(configPath string) (int, error) {
	// Check if OpenVPN is installed
	openvpnPath, err := exec.LookPath(v.config.OpenVPNBinary)
	if err != nil {
		return 0, fmt.Errorf("OpenVPN not found. Please install OpenVPN and ensure it's in your PATH")
	}

	// Pre-create log file with readable permissions before OpenVPN starts
	// This ensures the user can read it even though OpenVPN runs as root
	logPath := v.config.LogPath()
	if err := os.WriteFile(logPath, []byte{}, 0644); err != nil {
		// Not fatal, OpenVPN will create it
	}

	// Build command arguments
	args := []string{
		"--config", configPath,
		"--daemon",
		"--writepid", v.config.PidPath(),
		"--log-append", logPath, // Use --log-append instead of --log to preserve our permissions
		"--verb", "3",
	}

	// Check if we need sudo
	needsSudo := os.Geteuid() != 0

	var cmd *exec.Cmd
	if needsSudo {
		fmt.Println("OpenVPN requires root privileges. You may be prompted for your password.")
		sudoArgs := append([]string{openvpnPath}, args...)
		cmd = exec.Command("sudo", sudoArgs...)
	} else {
		cmd = exec.Command(openvpnPath, args...)
	}

	// Connect all standard streams for sudo password prompt
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	// Wait for the command to complete (it will daemonize itself with --daemon flag)
	if err := cmd.Wait(); err != nil {
		// Check if there's an error in the log
		if logData, readErr := os.ReadFile(v.config.LogPath()); readErr == nil && len(logData) > 0 {
			return 0, fmt.Errorf("OpenVPN failed: %s", string(logData))
		}
		return 0, fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	// Wait a moment for the daemon to write PID
	time.Sleep(1 * time.Second)

	// Read the PID from the file
	pidData, err := os.ReadFile(v.config.PidPath())
	if err != nil {
		// Check log for errors
		if logData, readErr := os.ReadFile(v.config.LogPath()); readErr == nil && len(logData) > 0 {
			lines := strings.Split(string(logData), "\n")
			lastLines := lines
			if len(lines) > 5 {
				lastLines = lines[len(lines)-5:]
			}
			return 0, fmt.Errorf("OpenVPN failed to start. Log:\n%s", strings.Join(lastLines, "\n"))
		}
		return 0, fmt.Errorf("OpenVPN started but couldn't determine PID")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	// Make log file readable by user so status command works without sudo
	// This runs silently - if it fails, status will just require sudo
	if needsSudo {
		exec.Command("sudo", "chmod", "644", logPath).Run()
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running.
func (v *VPNManager) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Check if /proc/PID exists and contains openvpn in cmdline
	// This works even for root-owned processes without needing signal permissions
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "openvpn")
}

// checkTunnelStatus checks if the OpenVPN tunnel is actually established.
// Returns "connected", "connecting", or "failed".
func (v *VPNManager) checkTunnelStatus() string {
	// Read the OpenVPN log to determine actual connection status
	logPath := v.config.LogPath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		// Can't read log, assume connecting
		return "connecting"
	}

	logContent := string(data)

	// Check for successful connection (this message appears when tunnel is up)
	if strings.Contains(logContent, "Initialization Sequence Completed") {
		// Make sure it's not followed by a disconnect/error
		lines := strings.Split(logContent, "\n")
		lastInitIndex := -1
		lastErrorIndex := -1

		for i, line := range lines {
			if strings.Contains(line, "Initialization Sequence Completed") {
				lastInitIndex = i
			}
			if strings.Contains(line, "SIGTERM") ||
				strings.Contains(line, "SIGUSR1") ||
				strings.Contains(line, "Connection reset") ||
				strings.Contains(line, "TLS Error") ||
				strings.Contains(line, "AUTH_FAILED") {
				lastErrorIndex = i
			}
		}

		// Connected if last init is after last error (or no error)
		if lastInitIndex > lastErrorIndex {
			return "connected"
		}
	}

	// Check for common failure patterns
	if strings.Contains(logContent, "EHOSTUNREACH") ||
		strings.Contains(logContent, "Connection refused") ||
		strings.Contains(logContent, "Connection timed out") ||
		strings.Contains(logContent, "No route to host") ||
		strings.Contains(logContent, "AUTH_FAILED") ||
		strings.Contains(logContent, "TLS Error") {
		// Check if these are recent (in the last few lines)
		lines := strings.Split(logContent, "\n")
		recentLines := lines
		if len(lines) > 20 {
			recentLines = lines[len(lines)-20:]
		}
		for _, line := range recentLines {
			if strings.Contains(line, "EHOSTUNREACH") ||
				strings.Contains(line, "No route to host") ||
				strings.Contains(line, "Connection refused") ||
				strings.Contains(line, "AUTH_FAILED") {
				return "failed"
			}
		}
	}

	return "connecting"
}

// loadState loads the connection state from disk.
func (v *VPNManager) loadState() (*ConnectionState, error) {
	data, err := os.ReadFile(v.config.StateFilePath())
	if err != nil {
		return nil, err
	}

	var state ConnectionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState saves the connection state to disk.
func (v *VPNManager) saveState(state *ConnectionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(v.config.StateFilePath(), data, 0600)
}

// updateStateFromLog tries to extract additional info from the OpenVPN log.
func (v *VPNManager) updateStateFromLog(state *ConnectionState) {
	file, err := os.Open(v.config.LogPath())
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for local IP assignment
		if strings.Contains(line, "ip addr add dev") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "add" && i+1 < len(parts) {
					ip := strings.Split(parts[i+1], "/")[0]
					state.LocalIP = ip
				}
			}
		}

		// Look for PUSH: Received control message
		if strings.Contains(line, "PUSH_REPLY") {
			// Parse pushed options
			if strings.Contains(line, "ifconfig") {
				start := strings.Index(line, "ifconfig ")
				if start != -1 {
					rest := line[start+9:]
					parts := strings.Fields(rest)
					if len(parts) >= 2 {
						state.LocalIP = parts[0]
						state.RemoteIP = parts[1]
					}
				}
			}
		}
	}
}

// promptGatewaySelection prompts the user to select a gateway.
func (v *VPNManager) promptGatewaySelection(gateways []Gateway) error {
	fmt.Println("Multiple gateways available. Please specify one:")
	fmt.Println()
	for i, gw := range gateways {
		fmt.Printf("  %d. %s", i+1, gw.Name)
		if gw.Location != "" {
			fmt.Printf(" (%s)", gw.Location)
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("Run: gatekey connect <gateway-name>")
	return nil
}
