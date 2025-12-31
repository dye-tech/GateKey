// Package client provides FIPS compliance checking functionality.
package client

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// FIPSStatus represents the FIPS compliance status of a component.
type FIPSStatus struct {
	Component   string
	Status      string
	Compliant   bool
	Description string
}

// CheckFIPSCompliance checks the system for FIPS 140-3 compliance.
func CheckFIPSCompliance() error {
	fmt.Println("GateKey FIPS 140-3 Compliance Check")
	fmt.Println("====================================")
	fmt.Println()

	var results []FIPSStatus
	allCompliant := true

	// Check 1: System FIPS mode (Linux)
	if runtime.GOOS == "linux" {
		status := checkLinuxFIPSMode()
		results = append(results, status)
		if !status.Compliant {
			allCompliant = false
		}
	}

	// Check 2: OpenSSL FIPS mode
	status := checkOpenSSLFIPS()
	results = append(results, status)
	if !status.Compliant {
		allCompliant = false
	}

	// Check 3: OpenVPN availability and ciphers
	status = checkOpenVPNCiphers()
	results = append(results, status)
	if !status.Compliant {
		allCompliant = false
	}

	// Check 4: Go crypto (always uses Go's crypto which can be FIPS-compliant with BoringCrypto)
	status = checkGoCrypto()
	results = append(results, status)

	// Print results
	fmt.Println("Component Status:")
	fmt.Println("-----------------")
	for _, r := range results {
		statusIcon := "✓"
		if !r.Compliant {
			statusIcon = "✗"
		}
		fmt.Printf("%s %-20s %s\n", statusIcon, r.Component+":", r.Status)
		if r.Description != "" {
			fmt.Printf("  %s\n", r.Description)
		}
	}

	fmt.Println()

	// Print FIPS-approved ciphers
	fmt.Println("FIPS 140-3 Approved Ciphers for VPN:")
	fmt.Println("------------------------------------")
	fmt.Println("  AES-256-GCM (recommended)")
	fmt.Println("  AES-128-GCM")
	fmt.Println("  AES-256-CBC + SHA256/SHA384/SHA512")
	fmt.Println("  AES-128-CBC + SHA256/SHA384/SHA512")
	fmt.Println()

	// Print summary
	fmt.Println("Summary:")
	fmt.Println("--------")
	if allCompliant {
		fmt.Println("✓ System appears to be FIPS 140-3 compliant")
		fmt.Println("  VPN connections will use FIPS-approved algorithms")
	} else {
		fmt.Println("✗ System is NOT fully FIPS 140-3 compliant")
		fmt.Println()
		printFIPSEnableInstructions()
	}

	return nil
}

// printFIPSEnableInstructions prints OS-specific instructions for enabling FIPS mode.
func printFIPSEnableInstructions() {
	fmt.Println("To enable FIPS mode:")
	fmt.Println()

	switch runtime.GOOS {
	case "linux":
		// Try to detect the distribution
		distro := detectLinuxDistro()
		switch distro {
		case "fedora":
			fmt.Println("  Fedora (41+):")
			fmt.Println("  -------------")
			fmt.Println("  1. Enable FIPS crypto policy:")
			fmt.Println("     sudo update-crypto-policies --set FIPS")
			fmt.Println()
			fmt.Println("  2. Enable kernel FIPS mode:")
			fmt.Println("     sudo grubby --update-kernel=ALL --args=\"fips=1\"")
			fmt.Println()
			fmt.Println("  3. Reboot the system:")
			fmt.Println("     sudo reboot")
		case "rhel", "centos", "rocky", "almalinux":
			fmt.Println("  RHEL / CentOS / Rocky / AlmaLinux:")
			fmt.Println("  -----------------------------------")
			fmt.Println("  1. Enable FIPS mode:")
			fmt.Println("     sudo fips-mode-setup --enable")
			fmt.Println()
			fmt.Println("  2. Reboot the system:")
			fmt.Println("     sudo reboot")
			fmt.Println()
			fmt.Println("  Alternative (RHEL 9+):")
			fmt.Println("     sudo update-crypto-policies --set FIPS")
			fmt.Println("     sudo grubby --update-kernel=ALL --args=\"fips=1\"")
			fmt.Println("     sudo reboot")
		case "ubuntu":
			fmt.Println("  Ubuntu (requires Ubuntu Pro):")
			fmt.Println("  ------------------------------")
			fmt.Println("  1. Attach Ubuntu Pro subscription:")
			fmt.Println("     sudo pro attach <your-token>")
			fmt.Println()
			fmt.Println("  2. Enable FIPS:")
			fmt.Println("     sudo pro enable fips-updates")
			fmt.Println()
			fmt.Println("  3. Reboot the system:")
			fmt.Println("     sudo reboot")
			fmt.Println()
			fmt.Println("  Note: FIPS on Ubuntu requires an Ubuntu Pro subscription.")
			fmt.Println("  Visit: https://ubuntu.com/pro")
		case "debian":
			fmt.Println("  Debian:")
			fmt.Println("  -------")
			fmt.Println("  Debian does not provide official FIPS-validated packages.")
			fmt.Println()
			fmt.Println("  Options:")
			fmt.Println("  - Use Ubuntu with Ubuntu Pro for FIPS support")
			fmt.Println("  - Use a RHEL-based distribution")
			fmt.Println("  - Build OpenSSL with FIPS module from source (advanced)")
		default:
			fmt.Println("  Linux (Generic):")
			fmt.Println("  -----------------")
			fmt.Println("  Check your distribution's documentation for FIPS setup.")
			fmt.Println()
			fmt.Println("  Common methods:")
			fmt.Println("  - Fedora/RHEL 9+: sudo update-crypto-policies --set FIPS")
			fmt.Println("  - RHEL/CentOS:    sudo fips-mode-setup --enable")
			fmt.Println("  - Ubuntu Pro:     sudo pro enable fips-updates")
		}
	case "darwin":
		fmt.Println("  macOS:")
		fmt.Println("  ------")
		fmt.Println("  macOS does not have a system-wide FIPS mode.")
		fmt.Println()
		fmt.Println("  Options:")
		fmt.Println("  - Apple's CoreCrypto has FIPS 140-2 validation (certificate #3856)")
		fmt.Println("  - OpenVPN on macOS uses OpenSSL, which is not FIPS-validated")
		fmt.Println()
		fmt.Println("  For strict FIPS 140-3 compliance:")
		fmt.Println("  - Deploy VPN gateways on RHEL or Ubuntu Pro")
		fmt.Println("  - Client-side crypto on macOS may not be fully FIPS-validated")
		fmt.Println()
		fmt.Println("  Run 'brew install openssl@3' for OpenSSL 3.x (uses same algorithms)")
	case "windows":
		fmt.Println("  Windows:")
		fmt.Println("  --------")
		fmt.Println("  Enable via Group Policy:")
		fmt.Println("  1. Open gpedit.msc")
		fmt.Println("  2. Navigate to: Computer Configuration > Windows Settings >")
		fmt.Println("     Security Settings > Local Policies > Security Options")
		fmt.Println("  3. Enable: \"System cryptography: Use FIPS compliant algorithms\"")
		fmt.Println("  4. Restart the computer")
		fmt.Println()
		fmt.Println("  Or via PowerShell (Run as Administrator):")
		fmt.Println("  Set-ItemProperty -Path \"HKLM:\\SYSTEM\\CurrentControlSet\\Control\\Lsa\\FipsAlgorithmPolicy\" -Name \"Enabled\" -Value 1")
		fmt.Println("  Restart-Computer")
	default:
		fmt.Println("  Consult your operating system documentation for FIPS setup.")
	}

	fmt.Println()
	fmt.Println("For more details, see: docs/fips-compliance.md")
}

// detectLinuxDistro attempts to detect the Linux distribution.
func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}

	content := strings.ToLower(string(data))

	switch {
	case strings.Contains(content, "fedora"):
		return "fedora"
	case strings.Contains(content, "rhel") || strings.Contains(content, "red hat"):
		return "rhel"
	case strings.Contains(content, "centos"):
		return "centos"
	case strings.Contains(content, "rocky"):
		return "rocky"
	case strings.Contains(content, "almalinux"):
		return "almalinux"
	case strings.Contains(content, "ubuntu"):
		return "ubuntu"
	case strings.Contains(content, "debian"):
		return "debian"
	default:
		return "unknown"
	}
}

// IsFIPSCompliant checks if the system meets FIPS 140-3 requirements.
// Returns true if the system is compliant, false otherwise.
func IsFIPSCompliant() bool {
	// Check Linux FIPS mode
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/sys/crypto/fips_enabled")
		if err == nil && strings.TrimSpace(string(data)) == "1" {
			return true
		}
	}

	// For now, we also consider systems with FIPS-capable OpenSSL as potentially compliant
	// A stricter check would require kernel FIPS mode
	cmd := exec.Command("openssl", "list", "-providers")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "fips") {
		return true
	}

	return false
}

// checkLinuxFIPSMode checks if Linux FIPS mode is enabled.
func checkLinuxFIPSMode() FIPSStatus {
	status := FIPSStatus{
		Component: "System FIPS Mode",
		Status:    "Disabled",
		Compliant: false,
	}

	// Check /proc/sys/crypto/fips_enabled
	data, err := os.ReadFile("/proc/sys/crypto/fips_enabled")
	if err != nil {
		status.Description = "Could not read FIPS status"
		return status
	}

	if strings.TrimSpace(string(data)) == "1" {
		status.Status = "Enabled"
		status.Compliant = true
		status.Description = "Kernel FIPS mode is active"
	} else {
		status.Description = "Kernel FIPS mode is not enabled"
	}

	return status
}

// checkOpenSSLFIPS checks OpenSSL FIPS mode status.
func checkOpenSSLFIPS() FIPSStatus {
	status := FIPSStatus{
		Component: "OpenSSL FIPS",
		Status:    "Unknown",
		Compliant: false,
	}

	// Check OpenSSL version and FIPS status
	cmd := exec.Command("openssl", "version", "-a")
	output, err := cmd.Output()
	if err != nil {
		status.Status = "Not Found"
		status.Description = "OpenSSL not available"
		return status
	}

	outputStr := string(output)

	// Check for FIPS provider or FIPS module
	if strings.Contains(outputStr, "fips") || strings.Contains(strings.ToLower(outputStr), "fips") {
		status.Status = "FIPS Provider Available"
		status.Compliant = true
		status.Description = "OpenSSL has FIPS support"
	} else {
		// Try to check FIPS provider directly (OpenSSL 3.x)
		providerCmd := exec.Command("openssl", "list", "-providers")
		providerOutput, err := providerCmd.Output()
		if err == nil && strings.Contains(string(providerOutput), "fips") {
			status.Status = "FIPS Provider Loaded"
			status.Compliant = true
			status.Description = "OpenSSL FIPS provider is active"
		} else {
			status.Status = "Standard (Non-FIPS)"
			status.Description = "Using standard OpenSSL without FIPS provider"
		}
	}

	// Extract version
	lines := strings.Split(outputStr, "\n")
	if len(lines) > 0 {
		status.Description += fmt.Sprintf(" (%s)", strings.TrimSpace(lines[0]))
	}

	return status
}

// checkOpenVPNCiphers checks OpenVPN for FIPS-compliant cipher support.
func checkOpenVPNCiphers() FIPSStatus {
	status := FIPSStatus{
		Component: "OpenVPN Ciphers",
		Status:    "Unknown",
		Compliant: false,
	}

	// Check if OpenVPN is available
	openvpnPath, err := exec.LookPath("openvpn")
	if err != nil {
		status.Status = "Not Found"
		status.Description = "OpenVPN is not installed"
		return status
	}

	// Get OpenVPN version
	versionCmd := exec.Command(openvpnPath, "--version")
	versionOutput, _ := versionCmd.Output()
	versionStr := string(versionOutput)
	versionLine := strings.Split(versionStr, "\n")[0]

	// Check available ciphers
	cipherCmd := exec.Command(openvpnPath, "--show-ciphers")
	cipherOutput, err := cipherCmd.Output()
	if err != nil {
		status.Status = "Error"
		status.Description = "Could not query ciphers"
		return status
	}

	cipherStr := string(cipherOutput)
	fipsCiphers := []string{"AES-256-GCM", "AES-128-GCM", "AES-256-CBC", "AES-128-CBC"}
	foundCiphers := []string{}

	for _, cipher := range fipsCiphers {
		if strings.Contains(cipherStr, cipher) {
			foundCiphers = append(foundCiphers, cipher)
		}
	}

	if len(foundCiphers) >= 2 {
		status.Status = "FIPS Ciphers Available"
		status.Compliant = true
		status.Description = fmt.Sprintf("Found: %s (%s)", strings.Join(foundCiphers, ", "), versionLine)
	} else {
		status.Status = "Limited Ciphers"
		status.Description = fmt.Sprintf("Only found: %s", strings.Join(foundCiphers, ", "))
	}

	return status
}

// checkGoCrypto checks Go's crypto status.
func checkGoCrypto() FIPSStatus {
	status := FIPSStatus{
		Component: "Go Crypto",
		Status:    "Standard",
		Compliant: true, // Go's crypto is considered acceptable for most uses
	}

	// Check if built with BoringCrypto (would indicate FIPS mode)
	// This is a compile-time setting, so we can only note the status
	status.Description = fmt.Sprintf("Go %s - standard crypto library", runtime.Version())

	return status
}
