# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of GateKey seriously. If you believe you have found a security vulnerability, please report it to us as described below.

**Please do not report security vulnerabilities through public GitHub issues.**

### How to Report

1. **Email**: Send an email to security@dye.tech with:
   - A description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact of the vulnerability
   - Any possible mitigations you've identified

2. **Expected Response Time**:
   - Initial response: Within 48 hours
   - Status update: Within 7 days
   - Resolution timeline: Depends on severity

### What to Include

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the issue
- Location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Safe Harbor

We support safe harbor for security researchers who:

- Make a good faith effort to avoid privacy violations, destruction of data, and interruption or degradation of our services
- Only interact with accounts you own or with explicit permission of the account holder
- Do not exploit a security issue for purposes other than verification
- Report vulnerabilities promptly and do not publicly disclose issues until we've had reasonable time to address them

## Security Best Practices

When deploying GateKey, we recommend:

### Server Security

- Run behind a reverse proxy with TLS termination
- Use strong, unique secrets for all configuration values
- Enable audit logging and monitor for suspicious activity
- Keep the server and all dependencies up to date
- Use network segmentation to isolate the control plane

### Database Security

- Use a dedicated database user with minimal required permissions
- Enable TLS for database connections
- Regularly backup the database
- Enable audit logging on the database

### Gateway Security

- Deploy gateways in isolated network segments
- Use the FIPS crypto profile for compliance requirements
- Enable TLS-Auth for additional security layer
- Regularly rotate gateway tokens
- Monitor gateway heartbeats for anomalies

### Client Security

- VPN configurations expire after 24 hours by default
- Users should not share their .ovpn files
- Revoke compromised configurations immediately
- Use the CLI client for better security (no file handling)

## Security Features

GateKey includes several security features:

- **Zero Trust Architecture**: All access is denied by default
- **Short-Lived Certificates**: Client certificates expire after 24 hours
- **Per-Identity Firewall Rules**: Each user gets isolated firewall rules
- **Real-Time Rule Enforcement**: Access changes take effect within 10 seconds
- **Audit Logging**: All access attempts are logged
- **FIPS 140-2 Compliance**: Optional FIPS-compliant crypto profile
- **TLS-Auth**: Additional HMAC authentication layer
- **Config Revocation**: Instant revocation of compromised configs

## Acknowledgments

We appreciate the security research community's efforts in helping keep GateKey and our users safe. Reporters of valid security issues will be acknowledged (with permission) in our release notes.
