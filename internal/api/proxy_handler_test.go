package api

import (
	"net"
	"testing"
)

func TestIsPrivateIP_PrivateIPv4(t *testing.T) {
	privateIPs := []string{
		"127.0.0.1",      // Loopback
		"127.255.255.254",
		"10.0.0.1",       // Class A private
		"10.255.255.255",
		"172.16.0.1",     // Class B private
		"172.31.255.255",
		"192.168.0.1",    // Class C private
		"192.168.255.255",
		"169.254.169.254", // AWS metadata endpoint (link-local)
		"169.254.0.1",
	}

	for _, ipStr := range privateIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("Failed to parse IP: %s", ipStr)
		}
		if !isPrivateIP(ip) {
			t.Errorf("Expected %s to be private, but it wasn't", ipStr)
		}
	}
}

func TestIsPrivateIP_PublicIPv4(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8",        // Google DNS
		"1.1.1.1",        // Cloudflare DNS
		"142.250.80.46",  // google.com
		"151.101.1.140",  // reddit.com
		"13.107.42.14",   // microsoft.com
	}

	for _, ipStr := range publicIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("Failed to parse IP: %s", ipStr)
		}
		if isPrivateIP(ip) {
			t.Errorf("Expected %s to be public, but it was marked as private", ipStr)
		}
	}
}

func TestIsPrivateIP_PrivateIPv6(t *testing.T) {
	privateIPs := []string{
		"::1",                    // IPv6 loopback
		"fe80::1",                // Link-local
		"fc00::1",                // Unique local
		"fd00::1",                // Unique local
		"::ffff:192.168.1.1",     // IPv4-mapped IPv6 (private IPv4)
		"::ffff:10.0.0.1",        // IPv4-mapped IPv6 (private IPv4)
	}

	for _, ipStr := range privateIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("Failed to parse IP: %s", ipStr)
		}
		if !isPrivateIP(ip) {
			t.Errorf("Expected %s to be private, but it wasn't", ipStr)
		}
	}
}

func TestIsPrivateIP_PublicIPv6(t *testing.T) {
	publicIPs := []string{
		"2607:f8b0:4004:800::200e", // google.com IPv6
		"2606:4700::6810:84e5",     // Cloudflare
	}

	for _, ipStr := range publicIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("Failed to parse IP: %s", ipStr)
		}
		if isPrivateIP(ip) {
			t.Errorf("Expected %s to be public, but it was marked as private", ipStr)
		}
	}
}

func TestIsPrivateIP_Unspecified(t *testing.T) {
	// Unspecified addresses should be blocked
	unspecified := []string{
		"0.0.0.0",
		"::",
	}

	for _, ipStr := range unspecified {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("Failed to parse IP: %s", ipStr)
		}
		if !isPrivateIP(ip) {
			t.Errorf("Expected %s (unspecified) to be blocked, but it wasn't", ipStr)
		}
	}
}

func TestErrSSRFBlocked(t *testing.T) {
	// Verify the error exists and has the expected message
	if ErrSSRFBlocked.Error() != "SSRF: request to private/internal address blocked" {
		t.Errorf("Unexpected error message: %s", ErrSSRFBlocked.Error())
	}
}

func TestPrivateIPBlocks_Initialized(t *testing.T) {
	// Verify that the private IP blocks were initialized
	if len(privateIPBlocks) == 0 {
		t.Error("privateIPBlocks was not initialized")
	}

	// We expect at least 7 CIDR blocks (5 IPv4 + 3 IPv6 minus comments)
	expectedMinBlocks := 7
	if len(privateIPBlocks) < expectedMinBlocks {
		t.Errorf("Expected at least %d private IP blocks, got %d", expectedMinBlocks, len(privateIPBlocks))
	}
}
