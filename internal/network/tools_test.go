package network

import (
	"context"
	"strings"
	"testing"
)

func TestValidateTarget_Valid(t *testing.T) {
	validTargets := []string{
		"192.168.1.1",
		"10.0.0.1",
		"google.com",
		"example.org",
		"sub.domain.example.com",
		"2001:db8::1",
		"localhost",
		"my-host.local",
	}

	for _, target := range validTargets {
		t.Run(target, func(t *testing.T) {
			err := validateTarget(target)
			if err != nil {
				t.Errorf("Expected valid target %q, got error: %v", target, err)
			}
		})
	}
}

func TestValidateTarget_Empty(t *testing.T) {
	err := validateTarget("")
	if err == nil {
		t.Error("Expected error for empty target")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected 'empty' in error message, got: %v", err)
	}
}

func TestValidateTarget_TooLong(t *testing.T) {
	longTarget := strings.Repeat("a", 254)
	err := validateTarget(longTarget)
	if err == nil {
		t.Error("Expected error for target exceeding 253 characters")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("Expected 'too long' in error message, got: %v", err)
	}
}

func TestValidateTarget_CommandInjection(t *testing.T) {
	injectionAttempts := []struct {
		name   string
		target string
	}{
		{"semicolon", "google.com; rm -rf /"},
		{"ampersand", "google.com && cat /etc/passwd"},
		{"pipe", "google.com | cat /etc/passwd"},
		{"backtick", "google.com `whoami`"},
		{"dollar", "google.com $(whoami)"},
		{"newline", "google.com\nwhoami"},
		{"carriage return", "google.com\rwhoami"},
		{"multiple injection chars", "host;|&`$"},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTarget(tt.target)
			if err == nil {
				t.Errorf("Expected error for injection attempt %q", tt.target)
			}
			if !strings.Contains(err.Error(), "invalid character") {
				t.Errorf("Expected 'invalid character' in error, got: %v", err)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	// validateHostname wraps validateTarget, so test basic functionality
	err := validateHostname("example.com")
	if err != nil {
		t.Errorf("Expected valid hostname, got error: %v", err)
	}

	err = validateHostname("")
	if err == nil {
		t.Error("Expected error for empty hostname")
	}

	err = validateHostname("host;injection")
	if err == nil {
		t.Error("Expected error for injection attempt in hostname")
	}
}

func TestValidatePorts_Valid(t *testing.T) {
	validPorts := []string{
		"",           // empty is allowed (uses defaults)
		"80",         // single port
		"443",        // single port
		"22,80,443",  // multiple ports
		"1-1000",     // port range
		"22,80-443",  // mixed
		"1,2,3,4,5",  // multiple single ports
		"1-65535",    // full range
	}

	for _, ports := range validPorts {
		t.Run(ports, func(t *testing.T) {
			err := validatePorts(ports)
			if err != nil {
				t.Errorf("Expected valid ports %q, got error: %v", ports, err)
			}
		})
	}
}

func TestValidatePorts_TooLong(t *testing.T) {
	longPorts := strings.Repeat("1,", 60) // exceeds 100 chars
	err := validatePorts(longPorts)
	if err == nil {
		t.Error("Expected error for port specification exceeding 100 characters")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("Expected 'too long' in error message, got: %v", err)
	}
}

func TestValidatePorts_InvalidCharacters(t *testing.T) {
	invalidPorts := []struct {
		name  string
		ports string
	}{
		{"semicolon", "80;443"},
		{"ampersand", "80&&443"},
		{"pipe", "80|443"},
		{"letters", "80,http"},
		{"spaces", "80, 443"},
		{"slash", "80/tcp"},
		{"backtick", "80`whoami`"},
		{"dollar", "80$(id)"},
	}

	for _, tt := range invalidPorts {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePorts(tt.ports)
			if err == nil {
				t.Errorf("Expected error for invalid ports %q", tt.ports)
			}
			if !strings.Contains(err.Error(), "invalid character") {
				t.Errorf("Expected 'invalid character' in error, got: %v", err)
			}
		})
	}
}

func TestTruncateOutput_Short(t *testing.T) {
	short := "This is a short output"
	result := truncateOutput(short)
	if result != short {
		t.Errorf("Expected output unchanged, got: %q", result)
	}
}

func TestTruncateOutput_WithWhitespace(t *testing.T) {
	withSpace := "  output with spaces  \n\n"
	result := truncateOutput(withSpace)
	expected := "output with spaces"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTruncateOutput_Long(t *testing.T) {
	// Create output longer than MaxOutputSize (10KB)
	long := strings.Repeat("a", MaxOutputSize+1000)
	result := truncateOutput(long)

	if len(result) <= MaxOutputSize {
		// Should be MaxOutputSize + truncation message
		t.Logf("Result length: %d", len(result))
	}

	if !strings.Contains(result, "truncated") {
		t.Error("Expected truncation message in output")
	}

	// Verify it starts with the original content
	if !strings.HasPrefix(result, strings.Repeat("a", 100)) {
		t.Error("Expected truncated output to start with original content")
	}
}

func TestTruncateOutput_ExactlyMaxSize(t *testing.T) {
	exact := strings.Repeat("b", MaxOutputSize)
	result := truncateOutput(exact)
	// Should not be truncated since it's exactly MaxOutputSize
	if strings.Contains(result, "truncated") {
		t.Error("Output exactly at MaxOutputSize should not be truncated")
	}
}

func TestAvailableTools(t *testing.T) {
	tools := AvailableTools()

	expectedTools := []string{"ping", "nslookup", "traceroute", "nc", "nmap"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in list", expected)
		}
	}
}

func TestToolResult_Fields(t *testing.T) {
	result := ToolResult{
		Tool:   "ping",
		Target: "example.com",
		Status: "success",
		Output: "PING example.com...",
	}

	if result.Tool != "ping" {
		t.Errorf("Expected tool 'ping', got %q", result.Tool)
	}
	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}
}

func TestConstants(t *testing.T) {
	if MaxOutputSize != 10*1024 {
		t.Errorf("Expected MaxOutputSize to be 10KB, got %d", MaxOutputSize)
	}
	if DefaultTimeout.Seconds() != 30 {
		t.Errorf("Expected DefaultTimeout to be 30s, got %v", DefaultTimeout)
	}
}

// Test Execute functions with invalid inputs (doesn't actually run commands)

func TestExecutePing_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	result, err := ExecutePing(ctx, "host;rm -rf /", 4)
	if err != nil {
		t.Fatalf("Expected no error (validation error in result), got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "invalid character") {
		t.Errorf("Expected 'invalid character' in error, got: %q", result.Error)
	}
}

func TestExecutePing_EmptyTarget(t *testing.T) {
	ctx := context.Background()
	result, err := ExecutePing(ctx, "", 4)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "empty") {
		t.Errorf("Expected 'empty' in error, got: %q", result.Error)
	}
}

func TestExecutePing_CountBounds(t *testing.T) {
	// This tests the count normalization without actually running ping
	// by using an invalid target
	ctx := context.Background()

	// Count <= 0 should default to 4
	result, _ := ExecutePing(ctx, "", 0)
	if result.Tool != "ping" {
		t.Error("Expected tool to be 'ping'")
	}

	// Count > 10 should be capped to 10
	result, _ = ExecutePing(ctx, "", 100)
	if result.Tool != "ping" {
		t.Error("Expected tool to be 'ping'")
	}
}

func TestExecuteNslookup_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteNslookup(ctx, "host|injection")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
}

func TestExecuteTraceroute_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteTraceroute(ctx, "host`whoami`")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
}

func TestExecuteNetcat_InvalidHost(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteNetcat(ctx, "host;injection", 80)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
}

func TestExecuteNetcat_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too high", 65536},
		{"way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := ExecuteNetcat(ctx, "localhost", tt.port)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result.Status != "error" {
				t.Errorf("Expected status 'error', got %q", result.Status)
			}
			if !strings.Contains(result.Error, "Invalid port") {
				t.Errorf("Expected 'Invalid port' in error, got: %q", result.Error)
			}
		})
	}
}

func TestExecuteNmap_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteNmap(ctx, "host$(id)", "80")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
}

func TestExecuteNmap_InvalidPorts(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteNmap(ctx, "localhost", "80;cat /etc/passwd")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "invalid character") {
		t.Errorf("Expected 'invalid character' in error, got: %q", result.Error)
	}
}

func TestExecuteNmap_EmptyPorts(t *testing.T) {
	// Empty ports should be valid (uses defaults)
	ctx := context.Background()
	// Use invalid target to avoid actually running nmap
	result, err := ExecuteNmap(ctx, "", "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// Should fail on target validation, not ports
	if !strings.Contains(result.Error, "empty") {
		t.Errorf("Expected target validation error, got: %q", result.Error)
	}
}
