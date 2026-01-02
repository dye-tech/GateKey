// Package network provides network diagnostic tools
package network

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ToolResult represents the result of a network tool execution
type ToolResult struct {
	Tool      string        `json:"tool"`
	Target    string        `json:"target"`
	Status    string        `json:"status"` // success, error, timeout
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	StartedAt time.Time     `json:"startedAt"`
	Duration  time.Duration `json:"duration"`
}

// MaxOutputSize is the maximum output size in bytes
const MaxOutputSize = 10 * 1024 // 10KB

// DefaultTimeout is the default execution timeout
const DefaultTimeout = 30 * time.Second

// ExecutePing runs a ping command
func ExecutePing(ctx context.Context, target string, count int) (*ToolResult, error) {
	if count <= 0 {
		count = 4
	}
	if count > 10 {
		count = 10
	}

	result := &ToolResult{
		Tool:      "ping",
		Target:    target,
		StartedAt: time.Now(),
	}

	// Validate target
	if err := validateTarget(target); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", fmt.Sprintf("%d", count), target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(result.StartedAt)

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Output = truncateOutput(stdout.String())
		result.Error = "Command timed out"
		return result, nil
	}

	if err != nil {
		result.Status = "error"
		result.Output = truncateOutput(stdout.String())
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, nil
	}

	result.Status = "success"
	result.Output = truncateOutput(stdout.String())
	return result, nil
}

// ExecuteNslookup runs a DNS lookup
func ExecuteNslookup(ctx context.Context, target string) (*ToolResult, error) {
	result := &ToolResult{
		Tool:      "nslookup",
		Target:    target,
		StartedAt: time.Now(),
	}

	// Validate target
	if err := validateHostname(target); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nslookup", target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(result.StartedAt)

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Output = truncateOutput(stdout.String())
		result.Error = "Command timed out"
		return result, nil
	}

	if err != nil {
		result.Status = "error"
		result.Output = truncateOutput(stdout.String())
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, nil
	}

	result.Status = "success"
	result.Output = truncateOutput(stdout.String())
	return result, nil
}

// ExecuteTraceroute runs a traceroute command
func ExecuteTraceroute(ctx context.Context, target string) (*ToolResult, error) {
	result := &ToolResult{
		Tool:      "traceroute",
		Target:    target,
		StartedAt: time.Now(),
	}

	// Validate target
	if err := validateTarget(target); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	// Use longer timeout for traceroute
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Try traceroute first, fall back to tracepath
	cmd := exec.CommandContext(ctx, "traceroute", "-m", "15", target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(result.StartedAt)

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Output = truncateOutput(stdout.String())
		result.Error = "Command timed out"
		return result, nil
	}

	if err != nil {
		// Try tracepath as fallback
		cmd = exec.CommandContext(ctx, "tracepath", target)
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		result.Duration = time.Since(result.StartedAt)
	}

	if err != nil {
		result.Status = "error"
		result.Output = truncateOutput(stdout.String())
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, nil
	}

	result.Status = "success"
	result.Output = truncateOutput(stdout.String())
	return result, nil
}

// ExecuteNetcat tests TCP connectivity to a host:port
func ExecuteNetcat(ctx context.Context, host string, port int) (*ToolResult, error) {
	result := &ToolResult{
		Tool:      "nc",
		Target:    fmt.Sprintf("%s:%d", host, port),
		StartedAt: time.Now(),
	}

	// Validate inputs
	if err := validateTarget(host); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}
	if port < 1 || port > 65535 {
		result.Status = "error"
		result.Error = "Invalid port number"
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use nc with zero I/O mode and 5s timeout
	cmd := exec.CommandContext(ctx, "nc", "-zv", "-w", "5", host, fmt.Sprintf("%d", port))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(result.StartedAt)

	// nc outputs to stderr typically
	output := stderr.String()
	if stdout.Len() > 0 {
		output = stdout.String() + "\n" + output
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Output = truncateOutput(output)
		result.Error = "Connection timed out"
		return result, nil
	}

	if err != nil {
		result.Status = "error"
		result.Output = truncateOutput(output)
		result.Error = fmt.Sprintf("Port %d is not open", port)
		return result, nil
	}

	result.Status = "success"
	result.Output = truncateOutput(output)
	return result, nil
}

// ExecuteNmap runs a limited nmap scan (TCP connect only)
func ExecuteNmap(ctx context.Context, target string, ports string) (*ToolResult, error) {
	result := &ToolResult{
		Tool:      "nmap",
		Target:    target,
		StartedAt: time.Now(),
	}

	// Validate target
	if err := validateTarget(target); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	// Validate ports
	if err := validatePorts(ports); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(result.StartedAt)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Use TCP connect scan (-sT) which doesn't require root
	// Limit to specified ports only
	args := []string{"-sT", "-Pn", "--max-retries", "2"}
	if ports != "" {
		args = append(args, "-p", ports)
	} else {
		// Default to common ports
		args = append(args, "-p", "22,80,443,3306,5432,6379,8080")
	}
	args = append(args, target)

	cmd := exec.CommandContext(ctx, "nmap", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(result.StartedAt)

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Output = truncateOutput(stdout.String())
		result.Error = "Scan timed out"
		return result, nil
	}

	if err != nil {
		result.Status = "error"
		result.Output = truncateOutput(stdout.String())
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, nil
	}

	result.Status = "success"
	result.Output = truncateOutput(stdout.String())
	return result, nil
}

// validateTarget validates a target host/IP
func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if len(target) > 253 {
		return fmt.Errorf("target too long")
	}
	// Block command injection
	for _, c := range target {
		if c == ';' || c == '&' || c == '|' || c == '`' || c == '$' || c == '\n' || c == '\r' {
			return fmt.Errorf("invalid character in target")
		}
	}
	return nil
}

// validateHostname validates a hostname for DNS lookup
func validateHostname(hostname string) error {
	if err := validateTarget(hostname); err != nil {
		return err
	}
	// Additional hostname validation could go here
	return nil
}

// validatePorts validates a port specification
func validatePorts(ports string) error {
	if ports == "" {
		return nil
	}
	if len(ports) > 100 {
		return fmt.Errorf("port specification too long")
	}
	// Block command injection
	for _, c := range ports {
		if !((c >= '0' && c <= '9') || c == ',' || c == '-') {
			return fmt.Errorf("invalid character in ports")
		}
	}
	return nil
}

// truncateOutput limits output size
func truncateOutput(output string) string {
	if len(output) > MaxOutputSize {
		return output[:MaxOutputSize] + "\n... (output truncated)"
	}
	return strings.TrimSpace(output)
}

// AvailableTools returns a list of available network tools
func AvailableTools() []string {
	return []string{"ping", "nslookup", "traceroute", "nc", "nmap"}
}
