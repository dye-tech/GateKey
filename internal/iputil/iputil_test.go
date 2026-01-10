// Package iputil provides IP address validation and utility functions.
package iputil

import (
	"net"
	"testing"
)

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 zeros", "0.0.0.0", true},
		{"valid IPv4 broadcast", "255.255.255.255", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"IPv6 address", "::1", false},
		{"IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"IPv6 compressed", "2001:db8::1", false},
		{"invalid IP", "not-an-ip", false},
		{"empty string", "", false},
		{"partial IPv4", "192.168.1", false},
		{"IPv4 with extra octet", "192.168.1.1.1", false},
		{"IPv4 with out of range", "256.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIPv4(tt.ip)
			if result != tt.expected {
				t.Errorf("IsIPv4(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv6 loopback", "::1", true},
		{"IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"IPv6 compressed", "2001:db8::1", true},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 all zeros", "::", true},
		{"IPv4 address", "192.168.1.1", false},
		{"invalid IP", "not-an-ip", false},
		{"empty string", "", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", false}, // Go treats IPv4-mapped IPv6 as IPv4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIPv6(tt.ip)
			if result != tt.expected {
				t.Errorf("IsIPv6(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:db8::1", true},
		{"IPv6 loopback", "::1", true},
		{"invalid", "not-an-ip", false},
		{"empty", "", false},
		{"CIDR notation", "192.168.1.0/24", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsValidCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected bool
	}{
		{"valid IPv4 CIDR", "192.168.1.0/24", true},
		{"valid IPv4 CIDR /32", "192.168.1.1/32", true},
		{"valid IPv4 CIDR /0", "0.0.0.0/0", true},
		{"valid IPv6 CIDR", "2001:db8::/32", true},
		{"valid IPv6 CIDR /128", "::1/128", true},
		{"valid IPv6 CIDR /0", "::/0", true},
		{"plain IP", "192.168.1.1", false},
		{"invalid CIDR", "not-a-cidr", false},
		{"empty", "", false},
		{"invalid prefix length", "192.168.1.0/33", false},
		{"invalid IPv6 prefix", "2001:db8::/129", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidCIDR(tt.cidr)
			if result != tt.expected {
				t.Errorf("IsValidCIDR(%q) = %v, want %v", tt.cidr, result, tt.expected)
			}
		})
	}
}

func TestIsIPv4CIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected bool
	}{
		{"valid IPv4 CIDR", "192.168.1.0/24", true},
		{"valid IPv4 CIDR /32", "10.0.0.1/32", true},
		{"IPv6 CIDR", "2001:db8::/32", false},
		{"invalid CIDR", "not-a-cidr", false},
		{"plain IPv4", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIPv4CIDR(tt.cidr)
			if result != tt.expected {
				t.Errorf("IsIPv4CIDR(%q) = %v, want %v", tt.cidr, result, tt.expected)
			}
		})
	}
}

func TestIsIPv6CIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected bool
	}{
		{"valid IPv6 CIDR", "2001:db8::/32", true},
		{"valid IPv6 CIDR /64", "fd00::/64", true},
		{"valid IPv6 CIDR /128", "::1/128", true},
		{"IPv4 CIDR", "192.168.1.0/24", false},
		{"invalid CIDR", "not-a-cidr", false},
		{"plain IPv6", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIPv6CIDR(tt.cidr)
			if result != tt.expected {
				t.Errorf("IsIPv6CIDR(%q) = %v, want %v", tt.cidr, result, tt.expected)
			}
		})
	}
}

func TestParseCIDRList(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []string
		expectError bool
	}{
		{
			name:        "single IPv4 CIDR",
			input:       "192.168.1.0/24",
			expected:    []string{"192.168.1.0/24"},
			expectError: false,
		},
		{
			name:        "multiple IPv4 CIDRs",
			input:       "192.168.1.0/24,10.0.0.0/8,172.16.0.0/12",
			expected:    []string{"192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"},
			expectError: false,
		},
		{
			name:        "mixed IPv4 and IPv6",
			input:       "192.168.1.0/24,2001:db8::/32",
			expected:    []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError: false,
		},
		{
			name:        "with spaces",
			input:       " 192.168.1.0/24 , 10.0.0.0/8 ",
			expected:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: false,
		},
		{
			name:        "empty parts",
			input:       "192.168.1.0/24,,10.0.0.0/8",
			expected:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectError: false,
		},
		{
			name:        "invalid CIDR",
			input:       "192.168.1.0/24,invalid,10.0.0.0/8",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCIDRList(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseCIDRList(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseCIDRList(%q) unexpected error: %v", tt.input, err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ParseCIDRList(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ParseCIDRList(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSplitCIDRsByVersion(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		expectedIPv4 []string
		expectedIPv6 []string
	}{
		{
			name:         "mixed CIDRs",
			input:        []string{"192.168.1.0/24", "2001:db8::/32", "10.0.0.0/8", "fd00::/64"},
			expectedIPv4: []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectedIPv6: []string{"2001:db8::/32", "fd00::/64"},
		},
		{
			name:         "only IPv4",
			input:        []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectedIPv4: []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectedIPv6: nil,
		},
		{
			name:         "only IPv6",
			input:        []string{"2001:db8::/32", "fd00::/64"},
			expectedIPv4: nil,
			expectedIPv6: []string{"2001:db8::/32", "fd00::/64"},
		},
		{
			name:         "empty input",
			input:        []string{},
			expectedIPv4: nil,
			expectedIPv6: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv4, ipv6 := SplitCIDRsByVersion(tt.input)
			if len(ipv4) != len(tt.expectedIPv4) {
				t.Errorf("SplitCIDRsByVersion IPv4 = %v, want %v", ipv4, tt.expectedIPv4)
			}
			if len(ipv6) != len(tt.expectedIPv6) {
				t.Errorf("SplitCIDRsByVersion IPv6 = %v, want %v", ipv6, tt.expectedIPv6)
			}
		})
	}
}

func TestCIDRContainsIP(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		ip       string
		expected bool
	}{
		{"IPv4 in range", "192.168.1.0/24", "192.168.1.100", true},
		{"IPv4 first address", "192.168.1.0/24", "192.168.1.0", true},
		{"IPv4 last address", "192.168.1.0/24", "192.168.1.255", true},
		{"IPv4 out of range", "192.168.1.0/24", "192.168.2.1", false},
		{"IPv6 in range", "2001:db8::/32", "2001:db8::1", true},
		{"IPv6 out of range", "2001:db8::/32", "2001:db9::1", false},
		{"invalid CIDR", "invalid", "192.168.1.1", false},
		{"invalid IP", "192.168.1.0/24", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CIDRContainsIP(tt.cidr, tt.ip)
			if result != tt.expected {
				t.Errorf("CIDRContainsIP(%q, %q) = %v, want %v", tt.cidr, tt.ip, result, tt.expected)
			}
		})
	}
}

func TestNormalizeCIDR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"already normalized IPv4", "192.168.1.0/24", "192.168.1.0/24", false},
		{"non-normalized IPv4", "192.168.1.5/24", "192.168.1.0/24", false},
		{"IPv4 host", "192.168.1.1/32", "192.168.1.1/32", false},
		{"IPv6 normalized", "2001:db8::/32", "2001:db8::/32", false},
		{"IPv6 non-normalized", "2001:db8::1/32", "2001:db8::/32", false},
		{"invalid CIDR", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeCIDR(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("NormalizeCIDR(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeCIDR(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeCIDR(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIPToHostCIDR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"IPv4 address", "192.168.1.1", "192.168.1.1/32", false},
		{"IPv6 address", "2001:db8::1", "2001:db8::1/128", false},
		{"IPv6 loopback", "::1", "::1/128", false},
		{"invalid IP", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IPToHostCIDR(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("IPToHostCIDR(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("IPToHostCIDR(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("IPToHostCIDR(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandIPv6(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"loopback", "::1", "0000:0000:0000:0000:0000:0000:0000:0001", false},
		{"full address", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"compressed", "2001:db8::1", "2001:0db8:0000:0000:0000:0000:0000:0001", false},
		{"all zeros", "::", "0000:0000:0000:0000:0000:0000:0000:0000", false},
		{"IPv4 address", "192.168.1.1", "", true},
		{"invalid IP", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandIPv6(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ExpandIPv6(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ExpandIPv6(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandIPv6(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompressIPv6(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"already compressed", "2001:db8::1", "2001:db8::1", false},
		{"full address", "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:db8::1", false},
		{"loopback", "0000:0000:0000:0000:0000:0000:0000:0001", "::1", false},
		{"all zeros", "0000:0000:0000:0000:0000:0000:0000:0000", "::", false},
		{"IPv4 address", "192.168.1.1", "", true},
		{"invalid IP", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompressIPv6(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("CompressIPv6(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("CompressIPv6(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("CompressIPv6(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"private 10.x.x.x", "10.0.0.1", true},
		{"private 172.16.x.x", "172.16.0.1", true},
		{"private 192.168.x.x", "192.168.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"loopback", "127.0.0.1", false},
		{"IPv6 private ULA", "fd00::1", true},
		{"IPv6 public", "2001:db8::1", false},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsLoopbackIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback range", "127.255.255.255", true},
		{"IPv6 loopback", "::1", true},
		{"private IP", "192.168.1.1", false},
		{"public IP", "8.8.8.8", false},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLoopbackIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsLoopbackIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsLinkLocalIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 link-local", "169.254.1.1", true},
		{"IPv6 link-local unicast", "fe80::1", true},
		{"private IP", "192.168.1.1", false},
		{"public IP", "8.8.8.8", false},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLinkLocalIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsLinkLocalIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestGetIPVersion(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected int
	}{
		{"IPv4 address", "192.168.1.1", 4},
		{"IPv6 address", "2001:db8::1", 6},
		{"IPv6 loopback", "::1", 6},
		{"IPv4 loopback", "127.0.0.1", 4},
		{"invalid IP", "invalid", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIPVersion(tt.ip)
			if result != tt.expected {
				t.Errorf("GetIPVersion(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateIPOrCIDR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"valid IPv4 IP", "192.168.1.1", false},
		{"valid IPv6 IP", "2001:db8::1", false},
		{"valid IPv4 CIDR", "192.168.1.0/24", false},
		{"valid IPv6 CIDR", "2001:db8::/32", false},
		{"invalid", "not-valid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPOrCIDR(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateIPOrCIDR(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateIPOrCIDR(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

func TestParseIPRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectStart string
		expectEnd   string
		expectError bool
	}{
		{
			name:        "IP range",
			input:       "192.168.1.1-192.168.1.10",
			expectStart: "192.168.1.1",
			expectEnd:   "192.168.1.10",
			expectError: false,
		},
		{
			name:        "IPv6 range",
			input:       "2001:db8::1-2001:db8::ff",
			expectStart: "2001:db8::1",
			expectEnd:   "2001:db8::ff",
			expectError: false,
		},
		{
			name:        "CIDR /24",
			input:       "192.168.1.0/24",
			expectStart: "192.168.1.0",
			expectEnd:   "192.168.1.255",
			expectError: false,
		},
		{
			name:        "single IP",
			input:       "192.168.1.1",
			expectStart: "192.168.1.1",
			expectEnd:   "192.168.1.1",
			expectError: false,
		},
		{
			name:        "invalid range start",
			input:       "invalid-192.168.1.10",
			expectError: true,
		},
		{
			name:        "invalid CIDR",
			input:       "192.168.1.0/33",
			expectError: true,
		},
		{
			name:        "invalid single IP",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseIPRange(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseIPRange(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseIPRange(%q) unexpected error: %v", tt.input, err)
				return
			}
			if start.String() != tt.expectStart {
				t.Errorf("ParseIPRange(%q) start = %v, want %v", tt.input, start.String(), tt.expectStart)
			}
			if end.String() != tt.expectEnd {
				t.Errorf("ParseIPRange(%q) end = %v, want %v", tt.input, end.String(), tt.expectEnd)
			}
		})
	}
}

func TestIPInRange(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		start    string
		end      string
		expected bool
	}{
		{"IP in range", "192.168.1.5", "192.168.1.1", "192.168.1.10", true},
		{"IP at start", "192.168.1.1", "192.168.1.1", "192.168.1.10", true},
		{"IP at end", "192.168.1.10", "192.168.1.1", "192.168.1.10", true},
		{"IP before range", "192.168.1.0", "192.168.1.1", "192.168.1.10", false},
		{"IP after range", "192.168.1.11", "192.168.1.1", "192.168.1.10", false},
		{"IPv6 in range", "2001:db8::5", "2001:db8::1", "2001:db8::ff", true},
		{"IPv6 before range", "2001:db8::0", "2001:db8::1", "2001:db8::ff", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			start := net.ParseIP(tt.start)
			end := net.ParseIP(tt.end)
			result := IPInRange(ip, start, end)
			if result != tt.expected {
				t.Errorf("IPInRange(%q, %q, %q) = %v, want %v", tt.ip, tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestIPInRangeNilInputs(t *testing.T) {
	ip := net.ParseIP("192.168.1.5")
	start := net.ParseIP("192.168.1.1")
	end := net.ParseIP("192.168.1.10")

	tests := []struct {
		name     string
		ip       net.IP
		start    net.IP
		end      net.IP
		expected bool
	}{
		{"nil IP", nil, start, end, false},
		{"nil start", ip, nil, end, false},
		{"nil end", ip, start, nil, false},
		{"all nil", nil, nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IPInRange(tt.ip, tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("IPInRange with nil inputs = %v, want %v", result, tt.expected)
			}
		})
	}
}
