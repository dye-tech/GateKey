package client

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DNSManager handles system DNS configuration
type DNSManager struct {
	originalResolv string // backup of /etc/resolv.conf content
	interfaceName  string // VPN interface name
	configured     bool
}

// NewDNSManager creates a new DNS manager
func NewDNSManager() *DNSManager {
	return &DNSManager{}
}

// ConfigureDNS sets system DNS servers and search domains for the VPN interface
func (d *DNSManager) ConfigureDNS(interfaceName string, dnsServers, searchDomains []string) error {
	if len(dnsServers) == 0 {
		return nil
	}

	d.interfaceName = interfaceName

	switch runtime.GOOS {
	case "linux":
		return d.configureLinux(interfaceName, dnsServers, searchDomains)
	case "darwin":
		return d.configureDarwin(interfaceName, dnsServers, searchDomains)
	default:
		return fmt.Errorf("DNS configuration not supported on %s", runtime.GOOS)
	}
}

// RestoreDNS restores the original DNS configuration
func (d *DNSManager) RestoreDNS() error {
	if !d.configured {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return d.restoreLinux()
	case "darwin":
		return d.restoreDarwin()
	default:
		return nil
	}
}

// configureLinux tries systemd-resolved first, falls back to /etc/resolv.conf
func (d *DNSManager) configureLinux(interfaceName string, dnsServers, searchDomains []string) error {
	// Try systemd-resolved first (most modern distros)
	if _, err := exec.LookPath("resolvectl"); err == nil {
		return d.configureSystemdResolved(interfaceName, dnsServers, searchDomains)
	}

	// Fallback to direct /etc/resolv.conf modification
	return d.configureResolvConf(dnsServers, searchDomains)
}

func (d *DNSManager) configureSystemdResolved(interfaceName string, dnsServers, searchDomains []string) error {
	// Set DNS servers for the interface
	args := append([]string{"dns", interfaceName}, dnsServers...)
	if output, err := exec.Command("resolvectl", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("resolvectl dns failed: %s: %w", output, err)
	}

	// Set search domains with routing-only flag (~)
	if len(searchDomains) > 0 {
		// Add ~ prefix for routing-only domains
		routingDomains := make([]string, len(searchDomains))
		for i, domain := range searchDomains {
			if !strings.HasPrefix(domain, "~") {
				routingDomains[i] = "~" + domain
			} else {
				routingDomains[i] = domain
			}
		}
		domainArgs := append([]string{"domain", interfaceName}, routingDomains...)
		if output, err := exec.Command("resolvectl", domainArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("resolvectl domain failed: %s: %w", output, err)
		}
	}

	d.configured = true
	fmt.Printf("DNS configured via systemd-resolved: servers=%v, domains=%v\n", dnsServers, searchDomains)
	return nil
}

func (d *DNSManager) configureResolvConf(dnsServers, searchDomains []string) error {
	// Backup existing resolv.conf
	existing, err := os.ReadFile("/etc/resolv.conf")
	if err == nil {
		d.originalResolv = string(existing)
	}

	// Build new resolv.conf content — prepend our servers, keep existing as fallback
	var lines []string
	if len(searchDomains) > 0 {
		lines = append(lines, "search "+strings.Join(searchDomains, " "))
	}
	for _, server := range dnsServers {
		lines = append(lines, "nameserver "+server)
	}
	// Keep existing nameservers as fallback
	for _, line := range strings.Split(d.originalResolv, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "nameserver") {
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile("/etc/resolv.conf", []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf: %w", err)
	}

	d.configured = true
	fmt.Printf("DNS configured via /etc/resolv.conf: servers=%v, domains=%v\n", dnsServers, searchDomains)
	return nil
}

// configureDarwin uses scutil to set DNS on macOS
func (d *DNSManager) configureDarwin(interfaceName string, dnsServers, searchDomains []string) error {
	// Create a DNS configuration using scutil
	var config strings.Builder
	config.WriteString("d.init\n")
	if len(dnsServers) > 0 {
		config.WriteString("d.add ServerAddresses * " + strings.Join(dnsServers, " ") + "\n")
	}
	if len(searchDomains) > 0 {
		config.WriteString("d.add SearchDomains * " + strings.Join(searchDomains, " ") + "\n")
		config.WriteString("d.add SupplementalMatchDomains * " + strings.Join(searchDomains, " ") + "\n")
	}
	fmt.Fprintf(&config, "set State:/Network/Service/GateKey-%s/DNS\n", interfaceName)

	cmd := exec.Command("scutil")
	cmd.Stdin = strings.NewReader(config.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("scutil failed: %s: %w", output, err)
	}

	d.configured = true
	fmt.Printf("DNS configured via scutil: servers=%v, domains=%v\n", dnsServers, searchDomains)
	return nil
}

func (d *DNSManager) restoreLinux() error {
	if _, err := exec.LookPath("resolvectl"); err == nil {
		// Revert systemd-resolved by resetting the interface
		if d.interfaceName != "" {
			exec.Command("resolvectl", "revert", d.interfaceName).Run()
		}
		d.configured = false
		return nil
	}

	// Restore /etc/resolv.conf
	if d.originalResolv != "" {
		if err := os.WriteFile("/etc/resolv.conf", []byte(d.originalResolv), 0644); err != nil {
			return fmt.Errorf("failed to restore /etc/resolv.conf: %w", err)
		}
	}
	d.configured = false
	return nil
}

func (d *DNSManager) restoreDarwin() error {
	if d.interfaceName != "" {
		config := fmt.Sprintf("remove State:/Network/Service/GateKey-%s/DNS\n", d.interfaceName)
		cmd := exec.Command("scutil")
		cmd.Stdin = strings.NewReader(config)
		cmd.Run()
	}
	d.configured = false
	return nil
}
