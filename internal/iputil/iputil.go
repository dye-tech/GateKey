// Package iputil provides IP address validation and utility functions.
package iputil

import (
	"fmt"
	"net"
	"strings"
)

// IsIPv4 checks if the given IP address string is a valid IPv4 address.
func IsIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() != nil
}

// IsIPv6 checks if the given IP address string is a valid IPv6 address.
func IsIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() == nil
}

// IsValidIP checks if the given string is a valid IP address (IPv4 or IPv6).
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidCIDR checks if the given string is a valid CIDR notation (IPv4 or IPv6).
func IsValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// IsIPv4CIDR checks if the given CIDR is a valid IPv4 CIDR.
func IsIPv4CIDR(cidr string) bool {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6CIDR checks if the given CIDR is a valid IPv6 CIDR.
func IsIPv6CIDR(cidr string) bool {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ip.To4() == nil
}

// ParseCIDRList parses a comma-separated list of CIDRs and returns them as a slice.
// Returns an error if any CIDR is invalid.
func ParseCIDRList(cidrList string) ([]string, error) {
	if cidrList == "" {
		return nil, nil
	}

	parts := strings.Split(cidrList, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		cidr := strings.TrimSpace(part)
		if cidr == "" {
			continue
		}
		if !IsValidCIDR(cidr) {
			return nil, fmt.Errorf("invalid CIDR: %s", cidr)
		}
		result = append(result, cidr)
	}

	return result, nil
}

// SplitCIDRsByVersion separates a list of CIDRs into IPv4 and IPv6 lists.
func SplitCIDRsByVersion(cidrs []string) (ipv4 []string, ipv6 []string) {
	for _, cidr := range cidrs {
		if IsIPv4CIDR(cidr) {
			ipv4 = append(ipv4, cidr)
		} else if IsIPv6CIDR(cidr) {
			ipv6 = append(ipv6, cidr)
		}
	}
	return
}

// CIDRContainsIP checks if a CIDR network contains the given IP address.
func CIDRContainsIP(cidr, ip string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return network.Contains(parsedIP)
}

// NormalizeCIDR normalizes a CIDR to its canonical form.
// For example, "192.168.1.5/24" becomes "192.168.1.0/24".
func NormalizeCIDR(cidr string) (string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}
	return network.String(), nil
}

// IPToHostCIDR converts an IP address to a host CIDR (/32 for IPv4, /128 for IPv6).
func IPToHostCIDR(ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}
	if parsed.To4() != nil {
		return ip + "/32", nil
	}
	return ip + "/128", nil
}

// ExpandIPv6 expands an abbreviated IPv6 address to its full form.
// For example, "::1" becomes "0000:0000:0000:0000:0000:0000:0000:0001".
func ExpandIPv6(ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}
	if parsed.To4() != nil {
		return "", fmt.Errorf("not an IPv6 address: %s", ip)
	}

	// Get the 16-byte representation
	ipv6 := parsed.To16()
	if ipv6 == nil {
		return "", fmt.Errorf("failed to convert to IPv6: %s", ip)
	}

	// Format as full IPv6
	return fmt.Sprintf("%02x%02x:%02x%02x:%02x%02x:%02x%02x:%02x%02x:%02x%02x:%02x%02x:%02x%02x",
		ipv6[0], ipv6[1], ipv6[2], ipv6[3],
		ipv6[4], ipv6[5], ipv6[6], ipv6[7],
		ipv6[8], ipv6[9], ipv6[10], ipv6[11],
		ipv6[12], ipv6[13], ipv6[14], ipv6[15]), nil
}

// CompressIPv6 compresses an IPv6 address to its shortest form.
// For example, "2001:0db8:0000:0000:0000:0000:0000:0001" becomes "2001:db8::1".
func CompressIPv6(ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}
	if parsed.To4() != nil {
		return "", fmt.Errorf("not an IPv6 address: %s", ip)
	}
	return parsed.String(), nil
}

// IsPrivateIP checks if the given IP address is a private/internal address.
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsPrivate()
}

// IsLoopbackIP checks if the given IP address is a loopback address.
func IsLoopbackIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback()
}

// IsLinkLocalIP checks if the given IP address is a link-local address.
func IsLinkLocalIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast()
}

// GetIPVersion returns 4 for IPv4, 6 for IPv6, or 0 for invalid.
func GetIPVersion(ip string) int {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return 0
	}
	if parsed.To4() != nil {
		return 4
	}
	return 6
}

// ValidateIPOrCIDR validates that a string is either a valid IP or CIDR.
func ValidateIPOrCIDR(value string) error {
	if IsValidIP(value) {
		return nil
	}
	if IsValidCIDR(value) {
		return nil
	}
	return fmt.Errorf("invalid IP address or CIDR: %s", value)
}

// ParseIPRange parses an IP range in the format "start-end" or as a single IP/CIDR.
// Returns the start IP, end IP, and any error.
func ParseIPRange(rangeStr string) (start, end net.IP, err error) {
	// Check if it's a range (contains "-")
	if strings.Contains(rangeStr, "-") {
		parts := strings.SplitN(rangeStr, "-", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid IP range format: %s", rangeStr)
		}
		start = net.ParseIP(strings.TrimSpace(parts[0]))
		end = net.ParseIP(strings.TrimSpace(parts[1]))
		if start == nil || end == nil {
			return nil, nil, fmt.Errorf("invalid IP addresses in range: %s", rangeStr)
		}
		return start, end, nil
	}

	// Check if it's a CIDR
	if strings.Contains(rangeStr, "/") {
		ip, network, err := net.ParseCIDR(rangeStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid CIDR: %w", err)
		}
		// Start is the network address
		start = network.IP
		// End is the broadcast address (all host bits set to 1)
		end = make(net.IP, len(network.IP))
		copy(end, network.IP)
		for i := range end {
			end[i] |= ^network.Mask[i]
		}
		_ = ip // unused
		return start, end, nil
	}

	// Single IP
	start = net.ParseIP(rangeStr)
	if start == nil {
		return nil, nil, fmt.Errorf("invalid IP address: %s", rangeStr)
	}
	return start, start, nil
}

// IPInRange checks if an IP is within the given range (inclusive).
func IPInRange(ip, start, end net.IP) bool {
	if ip == nil || start == nil || end == nil {
		return false
	}

	// Normalize all to 16-byte representation for consistent comparison
	ip = ip.To16()
	start = start.To16()
	end = end.To16()

	if ip == nil || start == nil || end == nil {
		return false
	}

	// Compare bytes
	for i := 0; i < 16; i++ {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > end[i] {
			return false
		}
		if ip[i] > start[i] && ip[i] < end[i] {
			return true
		}
	}
	return true
}
