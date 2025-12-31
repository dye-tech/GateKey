# Security Scan Exclusions

This document explains the GoSec security scanner exclusions configured for the GateKey project.

## Overview

GateKey is a zero-trust VPN solution that by design must:
- Execute OpenVPN binaries to establish VPN connections
- Read/write configuration files from user-specified paths
- Act as a reverse proxy forwarding requests to configured backend URLs
- Support optional TLS verification bypass for internal/development environments

These legitimate use cases trigger false positives in security scanners.

## Excluded Rules

### G101: Hardcoded Credentials
**Reason**: False positive on OpenVPN hook type constants like `"auth-user-pass-verify"` which are OpenVPN protocol strings, not actual credentials.

### G107: URL Provided to HTTP Request as Taint Input
**Reason**: GateKey includes a reverse proxy feature that intentionally forwards HTTP requests to user-configured backend URLs. This is core functionality, not a vulnerability.

### G115: Integer Overflow Conversion
**Reason**: False positives on known-safe integer conversions:
- Database connection pool sizes (always small positive integers)
- Argon2 password hashing parameters (fixed small values)
- These values are validated at configuration time

### G204: Subprocess Launched with Variable
**Reason**: The VPN client must execute the `openvpn` binary to establish connections. The binary path is located via `exec.LookPath()` and arguments are constructed internally, not from untrusted user input.

### G304: File Inclusion via Variable
**Reason**: VPN clients must read configuration files from user-specified paths. The paths come from:
- CLI arguments provided by the user running the client
- Configuration files in the user's home directory
- These are trusted paths in the user's control

### G402: TLS InsecureSkipVerify
**Reason**: The reverse proxy feature supports an optional `insecure_skip_verify` setting for internal/development environments where backend services use self-signed certificates. This is:
- Disabled by default
- Controlled via explicit configuration
- Documented with security warnings

## File Permission Decisions

### Certificates (0644)
CA and server certificates are written with 0644 permissions because:
- OpenVPN runs as a separate user and needs to read these files
- Certificates are public data (only private keys need protection)
- Private keys are always written with 0600

### Log Files (0644)
VPN log files are created with 0644 so users can read connection logs even though OpenVPN runs with elevated privileges.

### Client State Files (0600)
Client connection state files use 0600 as they may contain session information.

## Remaining Scanned Categories

The following rule categories are NOT excluded and actively scanned:
- SQL injection (G201-G203)
- Command injection in server code
- Weak cryptography (G401-G405)
- File permission issues (with documented exceptions)
- HTTP security issues (G112)

## Updating Exclusions

If adding new exclusions:
1. Document the specific reason in this file
2. Update `.github/workflows/codeql.yml` with the exclusion
3. Consider if the code can be refactored to avoid the issue instead
