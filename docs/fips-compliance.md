# FIPS Compliance Guide

## Overview

GateKey is designed with FIPS 140-2/140-3 compliance in mind. This document outlines the cryptographic controls and configuration options for achieving FIPS compliance.

## Cryptographic Algorithms

### Approved Algorithms

GateKey uses only FIPS-approved cryptographic algorithms:

| Category | Algorithm | FIPS Status |
|----------|-----------|-------------|
| Key Exchange | ECDH P-256, P-384 | Approved |
| Digital Signatures | ECDSA P-256, P-384 | Approved |
| Digital Signatures | RSA-2048, RSA-3072, RSA-4096 | Approved |
| Hashing | SHA-256, SHA-384, SHA-512 | Approved |
| Symmetric Encryption | AES-128-GCM, AES-256-GCM | Approved |
| Key Derivation | HKDF, PBKDF2 | Approved |
| TLS | TLS 1.2, TLS 1.3 | Approved |

### Disallowed Algorithms

The following algorithms are NOT used:

- MD5, SHA-1 (deprecated hash functions)
- DES, 3DES (deprecated symmetric ciphers)
- RSA < 2048 bits (insufficient key length)
- EC curves other than P-256, P-384 (unapproved curves)

## Configuration

### Enabling FIPS Mode

#### Operating System Level

Enable FIPS mode at the OS level. Instructions vary by distribution.

---

#### Fedora 41+

Fedora 41 and newer use `update-crypto-policies` instead of the deprecated `fips-mode-setup`:

```bash
# Enable FIPS crypto policy
sudo update-crypto-policies --set FIPS

# Enable kernel FIPS mode (required for full compliance)
sudo grubby --update-kernel=ALL --args="fips=1"

# Reboot to apply changes
sudo reboot
```

**Verify FIPS mode:**
```bash
# Check crypto policy
update-crypto-policies --show
# Expected output: FIPS

# Check kernel FIPS mode
cat /proc/sys/crypto/fips_enabled
# Expected output: 1
```

**Note:** If you only need application-level FIPS (not kernel), you can skip the `grubby` command, but full FIPS 140-3 compliance requires kernel FIPS mode.

---

#### RHEL / CentOS / Rocky Linux / AlmaLinux

RHEL-based distributions use the `fips-mode-setup` command:

```bash
# Install FIPS packages (if not already installed)
sudo dnf install crypto-policies-scripts

# Enable FIPS mode
sudo fips-mode-setup --enable

# Reboot to apply changes
sudo reboot
```

**Verify FIPS mode:**
```bash
# Check FIPS status
fips-mode-setup --check
# Expected output: FIPS mode is enabled.

# Or check directly
cat /proc/sys/crypto/fips_enabled
# Expected output: 1
```

**For RHEL 9+**, you can also use:
```bash
sudo update-crypto-policies --set FIPS
sudo grubby --update-kernel=ALL --args="fips=1"
sudo reboot
```

---

#### Ubuntu

Ubuntu requires **Ubuntu Pro** subscription for official FIPS support. FIPS packages are available for Ubuntu 18.04 LTS, 20.04 LTS, and 22.04 LTS.

**With Ubuntu Pro (Recommended for production):**
```bash
# Attach Ubuntu Pro subscription
sudo pro attach <your-token>

# Enable FIPS
sudo pro enable fips-updates

# Reboot to apply changes
sudo reboot
```

**Verify FIPS mode:**
```bash
cat /proc/sys/crypto/fips_enabled
# Expected output: 1

# Check Ubuntu Pro FIPS status
sudo pro status
```

**Without Ubuntu Pro (testing only):**

For testing purposes only, you can enable FIPS-like crypto policies:
```bash
# Install FIPS OpenSSL
sudo apt update
sudo apt install openssl libssl3

# Configure OpenSSL to prefer FIPS algorithms
# Edit /etc/ssl/openssl.cnf (see OpenSSL FIPS Provider section below)
```

**Note:** Without Ubuntu Pro, the kernel and OpenSSL are not validated FIPS modules. This configuration is for testing only and does not provide FIPS 140-3 compliance.

---

#### Debian

Debian does not provide official FIPS-validated packages. For FIPS compliance on Debian-based systems, consider:

1. Using Ubuntu with Ubuntu Pro instead
2. Using a RHEL-based distribution
3. Building OpenSSL with FIPS module from source (advanced)

---

#### macOS

macOS does not have a system-wide FIPS mode. FIPS compliance on macOS requires:

**Option 1: Use FIPS-validated OpenSSL**
```bash
# Install OpenSSL via Homebrew (not FIPS-validated, but uses same algorithms)
brew install openssl@3

# Verify OpenSSL version
/opt/homebrew/opt/openssl@3/bin/openssl version
```

**Option 2: Apple's Cryptographic Libraries**

Apple's CoreCrypto module has FIPS 140-2 validation (certificate #3856). Applications using Apple's native crypto APIs may achieve compliance. However, OpenVPN on macOS typically uses OpenSSL.

**Checking GateKey FIPS Compatibility on macOS:**
```bash
# Run GateKey FIPS check
gatekey fips-check
```

**Limitations on macOS:**
- No kernel FIPS mode
- OpenSSL from Homebrew is not FIPS-validated
- For strict FIPS compliance, use a Linux-based system

**Recommendation:** For environments requiring strict FIPS 140-3 compliance, deploy VPN gateways on RHEL/Ubuntu Pro and use macOS clients with the understanding that client-side crypto may not be FIPS-validated.

---

#### Windows

Windows has FIPS mode available through Group Policy:

**Enable via Group Policy:**
1. Open `gpedit.msc`
2. Navigate to: Computer Configuration → Windows Settings → Security Settings → Local Policies → Security Options
3. Enable: "System cryptography: Use FIPS compliant algorithms"
4. Restart the computer

**Enable via Registry:**
```powershell
# Run as Administrator
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\FipsAlgorithmPolicy" -Name "Enabled" -Value 1
Restart-Computer
```

**Verify FIPS mode:**
```powershell
(Get-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\FipsAlgorithmPolicy").Enabled
# Expected output: 1
```

---

#### Verifying FIPS Mode (All Platforms)

Use the GateKey client to check FIPS compliance:

```bash
gatekey fips-check
```

Example output:
```
GateKey FIPS 140-3 Compliance Check
====================================

Component Status:
-----------------
✓ System FIPS Mode:  Enabled
  Kernel FIPS mode is active
✓ OpenSSL FIPS:      FIPS Provider Loaded
  OpenSSL FIPS provider is active (OpenSSL 3.0.9)
✓ OpenVPN Ciphers:   FIPS Ciphers Available
  Found: AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC
✓ Go Crypto:         Standard
  Go 1.21.0 - standard crypto library

FIPS 140-3 Approved Ciphers for VPN:
------------------------------------
  AES-256-GCM (recommended)
  AES-128-GCM
  AES-256-CBC + SHA256/SHA384/SHA512
  AES-128-CBC + SHA256/SHA384/SHA512

Summary:
--------
✓ System appears to be FIPS 140-3 compliant
  VPN connections will use FIPS-approved algorithms
```

#### Go Runtime

Build with FIPS-compliant crypto:

```bash
# Using BoringCrypto
CGO_ENABLED=1 GOEXPERIMENT=boringcrypto go build ./...
```

#### OpenSSL FIPS Provider

Configure OpenSSL to use FIPS provider:

```bash
# /etc/ssl/openssl.cnf
openssl_conf = openssl_init

[openssl_init]
providers = provider_sect

[provider_sect]
fips = fips_sect
base = base_sect

[fips_sect]
activate = 1

[base_sect]
activate = 1
```

### GateKey Configuration

Configure FIPS-approved algorithms:

```yaml
pki:
  # Use FIPS-approved key algorithm
  key_algorithm: "ecdsa256"  # or ecdsa384, rsa2048, rsa3072, rsa4096

  # Minimum 2048-bit RSA or 256-bit EC
  organization: "Example Corp"
```

### Server-Enforced FIPS Mode

GateKey can enforce FIPS 140-3 compliance from the server side. When enabled:

1. **VPN configurations** use only FIPS-approved ciphers (AES-256-GCM, AES-128-GCM)
2. **Clients are blocked** from connecting if their system is not FIPS-compliant
3. **All gateways** use the FIPS crypto profile regardless of individual gateway settings

**Enabling Server-Enforced FIPS:**

1. Navigate to **Admin > Settings > General Settings**
2. Enable **Require FIPS 140-3 Compliance**
3. Click **Save Settings**

**What happens when FIPS is enforced:**

- Clients must have FIPS mode enabled on their system to connect
- VPN configs are generated with FIPS-only ciphers:
  - `cipher AES-256-GCM`
  - `data-ciphers AES-256-GCM:AES-128-GCM`
  - `auth SHA384`
  - `tls-version-min 1.2`
- Non-FIPS ciphers like CHACHA20-POLY1305 are excluded
- Clients see an error message with instructions if not compliant

**Client-Side Enforcement:**

When a client tries to connect to a FIPS-enforced server:

```bash
$ gatekey connect
Checking server requirements...
✗ FIPS 140-3 compliance required but system is not compliant

The GateKey server requires FIPS 140-3 compliance.
Your system does not meet the requirements.

To enable FIPS mode on your system:
  Linux (Fedora):  sudo update-crypto-policies --set FIPS && sudo reboot
  Linux (RHEL):    sudo fips-mode-setup --enable && sudo reboot
  macOS:           FIPS mode not available (see documentation)

Run 'gatekey fips-check' for detailed compliance status.
```

### Gateway Crypto Profiles

GateKey provides three crypto profiles for OpenVPN gateways:

| Profile | Data Ciphers | Auth | TLS Version | Use Case |
|---------|--------------|------|-------------|----------|
| **Modern** | AES-256-GCM, CHACHA20-POLY1305 | SHA256 | 1.2+ | Best performance |
| **FIPS** | AES-256-GCM only | SHA256 | 1.2+ | FIPS compliance |
| **Compatible** | AES-256-GCM, AES-256-CBC | SHA256 | 1.2+ | Legacy clients |

#### Setting Crypto Profile

When registering or editing a gateway in the web UI:

1. Navigate to **Admin > Gateways**
2. Click **Add Gateway** or **Edit** on an existing gateway
3. Select the appropriate **Crypto Profile**
4. Save the gateway

The gateway install script will automatically configure OpenVPN with the correct ciphers.

### OpenVPN Configuration

Configure OpenVPN with FIPS-approved ciphers:

```
# Server configuration (FIPS profile)
cipher AES-256-GCM
data-ciphers AES-256-GCM
auth SHA256
tls-version-min 1.2
tls-cipher TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384

# For OpenVPN 2.4.x, use ncp-ciphers instead of data-ciphers
ncp-ciphers AES-256-GCM
```

## Certificate Requirements

### Key Sizes

| Algorithm | Minimum Key Size | Recommended |
|-----------|-----------------|-------------|
| RSA | 2048 bits | 3072 bits |
| ECDSA | P-256 | P-384 |

### Certificate Validity

For FIPS compliance, consider:

- Short validity periods (24 hours for client certs)
- Regular key rotation
- Automated certificate renewal

### Certificate Generation

Example FIPS-compliant certificate generation:

```go
// Use P-256 or P-384 curves
key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

// Use SHA-384 for signing
template := x509.Certificate{
    SignatureAlgorithm: x509.ECDSAWithSHA384,
    // ...
}
```

## TLS Configuration

### Server TLS

```yaml
server:
  tls_enabled: true
  tls_cert: "/etc/gatekey/certs/server.crt"
  tls_key: "/etc/gatekey/certs/server.key"
```

### Cipher Suites

GateKey enforces FIPS-approved TLS cipher suites:

```go
tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
    CurvePreferences: []tls.CurveID{
        tls.CurveP384,
        tls.CurveP256,
    },
}
```

## Database Security

### PostgreSQL

Configure PostgreSQL for FIPS:

```sql
-- Use scram-sha-256 for password hashing
password_encryption = scram-sha-256

-- Require SSL
ssl = on
ssl_ciphers = 'HIGH:!aNULL:!MD5'
ssl_min_protocol_version = 'TLSv1.2'
```

### Connection String

```yaml
database:
  url: "postgres://user:pass@localhost/gatekey?sslmode=verify-full"
```

## Audit and Logging

### Audit Events

Enable comprehensive audit logging:

```yaml
audit:
  enabled: true
  events:
    - "auth.login"
    - "auth.logout"
    - "auth.failed"
    - "cert.issue"
    - "cert.revoke"
    - "policy.change"
```

### Log Protection

- Store logs securely
- Use append-only storage
- Implement log rotation with preservation

## Compliance Verification

### Self-Check

Run FIPS compliance check:

```bash
./bin/gatekey fips-check
```

Output:
```
FIPS Compliance Check
=====================
Operating System FIPS Mode: ENABLED
OpenSSL FIPS Provider: ENABLED
Go BoringCrypto: ENABLED

Cryptographic Algorithms:
  [PASS] Key Algorithm: ECDSA P-384
  [PASS] Hash Algorithm: SHA-384
  [PASS] TLS Cipher: TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384

Overall: COMPLIANT
```

### Documentation

Maintain documentation for:

1. FIPS mode configuration
2. Key management procedures
3. Certificate lifecycle management
4. Audit log retention policies
5. Incident response procedures

## Post-Quantum Considerations

### Future Readiness

GateKey is designed for post-quantum readiness:

1. **Modular Crypto**: Crypto operations abstracted behind interfaces
2. **Hybrid Support**: Architecture supports hybrid classical+PQC schemes
3. **Algorithm Agility**: Easy to add new algorithms

### NIST PQC Algorithms

When NIST finalizes PQC standards, GateKey will support:

- ML-KEM (CRYSTALS-Kyber) for key encapsulation
- ML-DSA (CRYSTALS-Dilithium) for signatures
- SLH-DSA (SPHINCS+) for hash-based signatures

## References

- [NIST FIPS 140-3](https://csrc.nist.gov/publications/detail/fips/140/3/final)
- [NIST SP 800-131A Rev 2](https://csrc.nist.gov/publications/detail/sp/800-131a/rev-2/final)
- [NIST SP 800-52 Rev 2](https://csrc.nist.gov/publications/detail/sp/800-52/rev-2/final)
- [OpenSSL FIPS Module](https://www.openssl.org/docs/man3.0/man7/fips_module.html)
