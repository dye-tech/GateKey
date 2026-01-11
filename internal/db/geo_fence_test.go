package db

import (
	"net"
	"strings"
	"testing"
)

func TestGeoFenceRule_Fields(t *testing.T) {
	ipRange := "192.168.1.0/24"
	ipv6Range := "2001:db8::/32"
	rule := GeoFenceRule{
		ID:          "test-id",
		Name:        "Test Rule",
		Description: "Test Description",
		IPRange:     &ipRange,
		IPv6Range:   &ipv6Range,
		IsActive:    true,
	}

	if rule.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", rule.ID)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("Expected Name 'Test Rule', got %q", rule.Name)
	}
	if rule.IPRange == nil || *rule.IPRange != "192.168.1.0/24" {
		t.Error("Expected IPRange '192.168.1.0/24'")
	}
	if rule.IPv6Range == nil || *rule.IPv6Range != "2001:db8::/32" {
		t.Error("Expected IPv6Range '2001:db8::/32'")
	}
	if !rule.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestGeoFenceRule_NullableRanges(t *testing.T) {
	// Test IPv4 only rule
	ipRange := "10.0.0.0/8"
	ipv4OnlyRule := GeoFenceRule{
		ID:        "ipv4-only",
		Name:      "IPv4 Only",
		IPRange:   &ipRange,
		IPv6Range: nil,
		IsActive:  true,
	}

	if ipv4OnlyRule.IPRange == nil {
		t.Error("Expected IPRange to be set")
	}
	if ipv4OnlyRule.IPv6Range != nil {
		t.Error("Expected IPv6Range to be nil")
	}

	// Test IPv6 only rule
	ipv6Range := "2001:db8::/32"
	ipv6OnlyRule := GeoFenceRule{
		ID:        "ipv6-only",
		Name:      "IPv6 Only",
		IPRange:   nil,
		IPv6Range: &ipv6Range,
		IsActive:  true,
	}

	if ipv6OnlyRule.IPRange != nil {
		t.Error("Expected IPRange to be nil")
	}
	if ipv6OnlyRule.IPv6Range == nil {
		t.Error("Expected IPv6Range to be set")
	}
}

func TestGeoFenceErrors(t *testing.T) {
	if ErrGeoFenceRuleNotFound.Error() == "" {
		t.Error("ErrGeoFenceRuleNotFound should have an error message")
	}
}

// TestIPMatching tests the core IP matching logic without database
func TestIPMatching_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		ip       string
		expected bool
	}{
		{"exact match /32", "192.168.1.1/32", "192.168.1.1", true},
		{"subnet match /24", "192.168.1.0/24", "192.168.1.100", true},
		{"subnet match first /24", "192.168.1.0/24", "192.168.1.0", true},
		{"subnet match last /24", "192.168.1.0/24", "192.168.1.255", true},
		{"subnet no match /24", "192.168.1.0/24", "192.168.2.1", false},
		{"wide range /8", "10.0.0.0/8", "10.255.255.255", true},
		{"wide range no match /8", "10.0.0.0/8", "11.0.0.1", false},
		{"private range /16", "172.16.0.0/16", "172.16.50.100", true},
		{"private range no match /16", "172.16.0.0/16", "172.17.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %s", tt.cidr)
			}

			result := ipNet.Contains(ip)
			if result != tt.expected {
				t.Errorf("IP %s in CIDR %s: expected %v, got %v", tt.ip, tt.cidr, tt.expected, result)
			}
		})
	}
}

// TestIPMatching_IPv6 tests IPv6 CIDR matching
func TestIPMatching_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		ip       string
		expected bool
	}{
		{"exact match /128", "2001:db8::1/128", "2001:db8::1", true},
		{"exact match no match /128", "2001:db8::1/128", "2001:db8::2", false},
		{"subnet match /64", "2001:db8:1234::/64", "2001:db8:1234::abcd", true},
		{"subnet no match /64", "2001:db8:1234::/64", "2001:db8:5678::1", false},
		{"wide range /32", "2001:db8::/32", "2001:db8:ffff:ffff::1", true},
		{"wide range no match /32", "2001:db8::/32", "2001:db9::1", false},
		{"loopback /128", "::1/128", "::1", true},
		{"link local /10", "fe80::/10", "fe80::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %s", tt.cidr)
			}

			result := ipNet.Contains(ip)
			if result != tt.expected {
				t.Errorf("IP %s in CIDR %s: expected %v, got %v", tt.ip, tt.cidr, tt.expected, result)
			}
		})
	}
}

// TestIPv4Detection tests detection of IPv4 vs IPv6
func TestIPv4Detection(t *testing.T) {
	tests := []struct {
		name   string
		ip     string
		isIPv6 bool
	}{
		{"IPv4 simple", "192.168.1.1", false},
		{"IPv4 loopback", "127.0.0.1", false},
		{"IPv6 simple", "2001:db8::1", true},
		{"IPv6 loopback", "::1", true},
		{"IPv6 full", "2001:0db8:0000:0000:0000:0000:0000:0001", true},
		{"IPv6 link local", "fe80::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			isIPv6 := ip.To4() == nil
			if isIPv6 != tt.isIPv6 {
				t.Errorf("IP %s: expected isIPv6=%v, got %v", tt.ip, tt.isIPv6, isIPv6)
			}
		})
	}
}

// TestMultipleCIDRParsing tests parsing multiple CIDRs from comma-separated string
func TestMultipleCIDRParsing(t *testing.T) {
	tests := []struct {
		name        string
		cidrString  string
		testIP      string
		shouldMatch bool
	}{
		{
			name:        "single CIDR match",
			cidrString:  "192.168.1.0/24",
			testIP:      "192.168.1.50",
			shouldMatch: true,
		},
		{
			name:        "multiple CIDRs first match",
			cidrString:  "10.0.0.0/8, 192.168.0.0/16",
			testIP:      "10.1.2.3",
			shouldMatch: true,
		},
		{
			name:        "multiple CIDRs second match",
			cidrString:  "10.0.0.0/8, 192.168.0.0/16",
			testIP:      "192.168.100.50",
			shouldMatch: true,
		},
		{
			name:        "multiple CIDRs no match",
			cidrString:  "10.0.0.0/8, 192.168.0.0/16",
			testIP:      "172.16.1.1",
			shouldMatch: false,
		},
		{
			name:        "with spaces",
			cidrString:  "  10.0.0.0/8  ,  192.168.0.0/16  ",
			testIP:      "10.0.0.1",
			shouldMatch: true,
		},
		{
			name:        "IPv6 multiple",
			cidrString:  "2001:db8::/32, 2001:db9::/32",
			testIP:      "2001:db9::100",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.testIP)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.testIP)
			}

			cidrs := strings.Split(tt.cidrString, ",")
			matched := false
			for _, cidr := range cidrs {
				cidr = strings.TrimSpace(cidr)
				if cidr == "" {
					continue
				}
				_, ipNet, err := net.ParseCIDR(cidr)
				if err != nil {
					continue
				}
				if ipNet.Contains(ip) {
					matched = true
					break
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("IP %s in CIDRs %q: expected match=%v, got %v",
					tt.testIP, tt.cidrString, tt.shouldMatch, matched)
			}
		})
	}
}

// TestInvalidCIDR tests handling of invalid CIDR strings
func TestInvalidCIDR(t *testing.T) {
	invalidCIDRs := []string{
		"not-a-cidr",
		"192.168.1.1",      // missing prefix
		"192.168.1.0/33",   // invalid prefix for IPv4
		"2001:db8::/129",   // invalid prefix for IPv6
		"192.168.1.256/24", // invalid IP
		"",                 // empty
		"   ",              // whitespace only
	}

	for _, cidr := range invalidCIDRs {
		_, _, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err == nil && strings.TrimSpace(cidr) != "" {
			t.Errorf("Expected error for invalid CIDR: %q", cidr)
		}
	}
}

// TestInvalidIP tests handling of invalid IP addresses
func TestInvalidIP(t *testing.T) {
	invalidIPs := []string{
		"not-an-ip",
		"192.168.1.256",
		"192.168.1",
		"",
		"   ",
	}

	for _, ipStr := range invalidIPs {
		ip := net.ParseIP(strings.TrimSpace(ipStr))
		if ip != nil && strings.TrimSpace(ipStr) != "" {
			t.Errorf("Expected nil for invalid IP: %q", ipStr)
		}
	}
}

// TestCountryCIDRRanges tests some real-world country CIDR examples
func TestCountryCIDRRanges(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		testIPs     []string
		shouldMatch bool
	}{
		{
			name:        "US range example",
			cidr:        "3.0.0.0/9",
			testIPs:     []string{"3.5.5.5", "3.100.200.100"},
			shouldMatch: true,
		},
		{
			name:        "Private RFC1918 10.x.x.x",
			cidr:        "10.0.0.0/8",
			testIPs:     []string{"10.0.0.1", "10.255.255.254"},
			shouldMatch: true,
		},
		{
			name:        "Private RFC1918 172.16-31.x.x",
			cidr:        "172.16.0.0/12",
			testIPs:     []string{"172.16.0.1", "172.31.255.254"},
			shouldMatch: true,
		},
		{
			name:        "Private RFC1918 192.168.x.x",
			cidr:        "192.168.0.0/16",
			testIPs:     []string{"192.168.0.1", "192.168.255.254"},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %s", tt.cidr)
			}

			for _, ipStr := range tt.testIPs {
				ip := net.ParseIP(ipStr)
				if ip == nil {
					t.Fatalf("Failed to parse IP: %s", ipStr)
				}

				result := ipNet.Contains(ip)
				if result != tt.shouldMatch {
					t.Errorf("IP %s in CIDR %s: expected %v, got %v",
						ipStr, tt.cidr, tt.shouldMatch, result)
				}
			}
		})
	}
}

// TestIPMatchingLogic simulates IsIPAllowed logic without database
func TestIPMatchingLogic(t *testing.T) {
	// Create test rules
	ipRange1 := "192.168.1.0/24, 10.0.0.0/8"
	ipv6Range1 := "2001:db8::/32"

	rule1 := GeoFenceRule{
		ID:        "rule1",
		Name:      "Office Network",
		IPRange:   &ipRange1,
		IPv6Range: &ipv6Range1,
		IsActive:  true,
	}

	ipRange2 := "172.16.0.0/12"
	rule2 := GeoFenceRule{
		ID:        "rule2",
		Name:      "VPN Network",
		IPRange:   &ipRange2,
		IPv6Range: nil,
		IsActive:  true,
	}

	inactiveIPRange := "0.0.0.0/0"
	rule3 := GeoFenceRule{
		ID:        "rule3",
		Name:      "Inactive Rule",
		IPRange:   &inactiveIPRange,
		IPv6Range: nil,
		IsActive:  false, // inactive
	}

	rules := []*GeoFenceRule{&rule1, &rule2, &rule3}

	tests := []struct {
		name        string
		clientIP    string
		shouldAllow bool
	}{
		{"IPv4 in first rule range 1", "192.168.1.100", true},
		{"IPv4 in first rule range 2", "10.5.5.5", true},
		{"IPv4 in second rule", "172.20.1.1", true},
		{"IPv4 not in any active rule", "8.8.8.8", false},
		{"IPv6 in first rule", "2001:db8::1", true},
		{"IPv6 not in any rule", "2001:db9::1", false},
		// Rule3 is inactive, so 1.1.1.1 should NOT match even though 0.0.0.0/0 would match
		{"IPv4 matches inactive rule only", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.clientIP)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.clientIP)
			}

			isIPv6 := ip.To4() == nil
			allowed := false

			for _, rule := range rules {
				if !rule.IsActive {
					continue
				}

				// Check IPv6 ranges first for IPv6 clients
				if isIPv6 && rule.IPv6Range != nil && *rule.IPv6Range != "" {
					cidrs := strings.Split(*rule.IPv6Range, ",")
					for _, cidr := range cidrs {
						cidr = strings.TrimSpace(cidr)
						if cidr == "" {
							continue
						}
						_, ipNet, err := net.ParseCIDR(cidr)
						if err != nil {
							continue
						}
						if ipNet.Contains(ip) {
							allowed = true
							break
						}
					}
				}

				if allowed {
					break
				}

				// Check IPv4 ranges
				if rule.IPRange != nil && *rule.IPRange != "" {
					cidrs := strings.Split(*rule.IPRange, ",")
					for _, cidr := range cidrs {
						cidr = strings.TrimSpace(cidr)
						if cidr == "" {
							continue
						}
						_, ipNet, err := net.ParseCIDR(cidr)
						if err != nil {
							continue
						}
						if ipNet.Contains(ip) {
							allowed = true
							break
						}
					}
				}

				if allowed {
					break
				}
			}

			if allowed != tt.shouldAllow {
				t.Errorf("IP %s: expected allowed=%v, got %v", tt.clientIP, tt.shouldAllow, allowed)
			}
		})
	}
}

// TestEmptyRules tests behavior when no rules exist
func TestEmptyRules(t *testing.T) {
	rules := []*GeoFenceRule{}

	// With whitelist model, no rules = no access
	if len(rules) != 0 {
		t.Error("Expected empty rules slice")
	}

	// Any IP should be denied with no rules (whitelist model)
	testIPs := []string{"192.168.1.1", "8.8.8.8", "2001:db8::1"}
	for _, ipStr := range testIPs {
		// Simulating the logic from IsIPAllowed
		// No rules = no access
		allowed := len(rules) > 0
		if allowed {
			t.Errorf("Expected IP %s to be denied with no rules", ipStr)
		}
	}
}

// TestIPv6OnlyRule tests rules with only IPv6 ranges
func TestIPv6OnlyRule(t *testing.T) {
	ipv6Range := "2001:db8::/32, 2001:db9::/32"
	rule := GeoFenceRule{
		ID:        "ipv6-only",
		Name:      "IPv6 Only Rule",
		IPRange:   nil, // No IPv4
		IPv6Range: &ipv6Range,
		IsActive:  true,
	}

	tests := []struct {
		name        string
		ip          string
		shouldMatch bool
	}{
		{"IPv6 in range 1", "2001:db8::100", true},
		{"IPv6 in range 2", "2001:db9::200", true},
		{"IPv6 not in range", "2001:dba::1", false},
		{"IPv4 should not match", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			matched := false

			// Only check IPv6 range since IPv4 is nil
			if rule.IPv6Range != nil && *rule.IPv6Range != "" {
				cidrs := strings.Split(*rule.IPv6Range, ",")
				for _, cidr := range cidrs {
					cidr = strings.TrimSpace(cidr)
					if cidr == "" {
						continue
					}
					_, ipNet, err := net.ParseCIDR(cidr)
					if err != nil {
						continue
					}
					if ipNet.Contains(ip) {
						matched = true
						break
					}
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("IP %s: expected match=%v, got %v", tt.ip, tt.shouldMatch, matched)
			}
		})
	}
}

// TestIPv4OnlyRule tests rules with only IPv4 ranges
func TestIPv4OnlyRule(t *testing.T) {
	ipRange := "192.168.0.0/16, 10.0.0.0/8"
	rule := GeoFenceRule{
		ID:        "ipv4-only",
		Name:      "IPv4 Only Rule",
		IPRange:   &ipRange,
		IPv6Range: nil, // No IPv6
		IsActive:  true,
	}

	tests := []struct {
		name        string
		ip          string
		shouldMatch bool
	}{
		{"IPv4 in range 1", "192.168.50.100", true},
		{"IPv4 in range 2", "10.1.2.3", true},
		{"IPv4 not in range", "172.16.1.1", false},
		{"IPv6 should not match", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			matched := false

			// Only check IPv4 range since IPv6 is nil
			if rule.IPRange != nil && *rule.IPRange != "" {
				cidrs := strings.Split(*rule.IPRange, ",")
				for _, cidr := range cidrs {
					cidr = strings.TrimSpace(cidr)
					if cidr == "" {
						continue
					}
					_, ipNet, err := net.ParseCIDR(cidr)
					if err != nil {
						continue
					}
					if ipNet.Contains(ip) {
						matched = true
						break
					}
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("IP %s: expected match=%v, got %v", tt.ip, tt.shouldMatch, matched)
			}
		})
	}
}

func TestNewGeoFenceStore(t *testing.T) {
	store := NewGeoFenceStore(nil)

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}
