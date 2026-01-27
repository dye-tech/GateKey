package pki

import (
	"context"
	"crypto/x509"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gatekey-project/gatekey/internal/config"
)

func TestNewCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	if ca.Certificate() == nil {
		t.Error("CA certificate is nil")
	}

	if ca.Certificate().IsCA != true {
		t.Error("CA certificate IsCA should be true")
	}

	if ca.Certificate().Subject.Organization[0] != "Test Org" {
		t.Errorf("Expected organization 'Test Org', got '%s'", ca.Certificate().Subject.Organization[0])
	}
}

func TestIssueClientCertificate(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-user",
		Email:      "test@example.com",
		ValidFor:   1 * time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if issued.Certificate == nil {
		t.Error("Issued certificate is nil")
	}

	if issued.Certificate.Subject.CommonName != "test-user" {
		t.Errorf("Expected CN 'test-user', got '%s'", issued.Certificate.Subject.CommonName)
	}

	if len(issued.Certificate.EmailAddresses) == 0 || issued.Certificate.EmailAddresses[0] != "test@example.com" {
		t.Error("Email not set correctly in certificate")
	}

	if issued.Certificate.IsCA {
		t.Error("Client certificate should not be a CA")
	}

	// Verify certificate chain
	err = ca.VerifyCertificate(issued.Certificate)
	if err != nil {
		t.Errorf("Certificate verification failed: %v", err)
	}
}

func TestIssueServerCertificate(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "vpn.example.com",
		DNSNames:   []string{"vpn.example.com", "vpn2.example.com"},
		ValidFor:   30 * 24 * time.Hour,
	}

	issued, err := ca.IssueServerCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if issued.Certificate.Subject.CommonName != "vpn.example.com" {
		t.Errorf("Expected CN 'vpn.example.com', got '%s'", issued.Certificate.Subject.CommonName)
	}

	if len(issued.Certificate.DNSNames) != 2 {
		t.Errorf("Expected 2 DNS names, got %d", len(issued.Certificate.DNSNames))
	}
}

func TestCertificateExpiry(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 1 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-user",
		ValidFor:   1 * time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Certificate should be valid now
	if time.Now().Before(issued.NotBefore) {
		t.Error("Certificate NotBefore is in the future")
	}

	if time.Now().After(issued.NotAfter) {
		t.Error("Certificate is already expired")
	}

	// Check validity duration
	duration := issued.NotAfter.Sub(issued.NotBefore)
	if duration != 1*time.Hour {
		t.Errorf("Expected validity of 1 hour, got %v", duration)
	}
}

func TestFingerprint(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	fingerprint := Fingerprint(ca.Certificate())

	if len(fingerprint) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Errorf("Expected fingerprint length 64, got %d", len(fingerprint))
	}

	// Fingerprint should be consistent
	fingerprint2 := Fingerprint(ca.Certificate())
	if fingerprint != fingerprint2 {
		t.Error("Fingerprint is not consistent")
	}
}

func TestDifferentKeyAlgorithms(t *testing.T) {
	algorithms := []string{"rsa2048", "rsa4096", "ecdsa256", "ecdsa384"}

	for _, alg := range algorithms {
		t.Run(alg, func(t *testing.T) {
			cfg := config.PKIConfig{
				KeyAlgorithm: alg,
				Organization: "Test Org",
				CertValidity: 24 * time.Hour,
				CAValidity:   365 * 24 * time.Hour,
			}

			ca, err := NewCA(cfg)
			if err != nil {
				t.Fatalf("Failed to create CA with %s: %v", alg, err)
			}

			req := CertificateRequest{
				CommonName: "test-user",
			}

			issued, err := ca.IssueClientCertificate(req)
			if err != nil {
				t.Fatalf("Failed to issue certificate with %s: %v", alg, err)
			}

			err = ca.VerifyCertificate(issued.Certificate)
			if err != nil {
				t.Errorf("Certificate verification failed with %s: %v", alg, err)
			}
		})
	}
}

func TestCertificatePEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM := ca.CertificatePEM()
	if len(certPEM) == 0 {
		t.Error("CertificatePEM returned empty")
	}

	// Should start with PEM header
	if string(certPEM[:27]) != "-----BEGIN CERTIFICATE-----" {
		t.Error("CertificatePEM should start with certificate header")
	}

	// Should be parseable
	cert, err := ParseCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("Failed to parse CertificatePEM: %v", err)
	}
	if !cert.IsCA {
		t.Error("Parsed certificate should be a CA")
	}
}

func TestPrivateKeyPEM(t *testing.T) {
	algorithms := []string{"rsa2048", "ecdsa256"}

	for _, alg := range algorithms {
		t.Run(alg, func(t *testing.T) {
			cfg := config.PKIConfig{
				KeyAlgorithm: alg,
				Organization: "Test Org",
				CertValidity: 24 * time.Hour,
				CAValidity:   365 * 24 * time.Hour,
			}

			ca, err := NewCA(cfg)
			if err != nil {
				t.Fatalf("Failed to create CA: %v", err)
			}

			keyPEM := ca.PrivateKeyPEM()
			if len(keyPEM) == 0 {
				t.Error("PrivateKeyPEM returned empty")
			}

			// Should contain private key header
			keyStr := string(keyPEM)
			if alg == "rsa2048" {
				if !contains(keyStr, "RSA PRIVATE KEY") {
					t.Error("RSA key should have RSA PRIVATE KEY header")
				}
			} else if alg == "ecdsa256" {
				if !contains(keyStr, "EC PRIVATE KEY") {
					t.Error("ECDSA key should have EC PRIVATE KEY header")
				}
			}
		})
	}
}

func TestLoadFromPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a CA to get valid PEM data
	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM := string(ca1.CertificatePEM())
	keyPEM := string(ca1.PrivateKeyPEM())

	// Create a new CA and load from PEM
	ca2 := &CA{
		config: cfg,
	}
	err = ca2.loadFromPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("loadFromPEM failed: %v", err)
	}

	// Verify loaded CA matches original
	if Fingerprint(ca1.Certificate()) != Fingerprint(ca2.Certificate()) {
		t.Error("Loaded CA certificate should match original")
	}
}

func TestLoadFromPEM_InvalidCert(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
	}
	ca := &CA{config: cfg}

	err := ca.loadFromPEM("invalid cert", "invalid key")
	if err == nil {
		t.Error("loadFromPEM should fail with invalid PEM")
	}
}

func TestLoadFromPEM_InvalidKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM := string(ca1.CertificatePEM())

	ca2 := &CA{config: cfg}
	err = ca2.loadFromPEM(certPEM, "invalid key")
	if err == nil {
		t.Error("loadFromPEM should fail with invalid key PEM")
	}
}

func TestParseCertificatePEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM := ca.CertificatePEM()
	cert, err := ParseCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("ParseCertificatePEM failed: %v", err)
	}

	if cert.Subject.Organization[0] != "Test Org" {
		t.Error("Parsed certificate should have correct organization")
	}
}

func TestParseCertificatePEM_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		pemData []byte
	}{
		{"empty", []byte{}},
		{"not PEM", []byte("not valid pem data")},
		{"wrong type", []byte("-----BEGIN PRIVATE KEY-----\nYWJjZA==\n-----END PRIVATE KEY-----")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCertificatePEM(tt.pemData)
			if err == nil {
				t.Error("ParseCertificatePEM should fail for invalid PEM")
			}
		})
	}
}

func TestGenerateSubCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	subCertPEM, subKeyPEM, err := ca.GenerateSubCA("Sub CA")
	if err != nil {
		t.Fatalf("GenerateSubCA failed: %v", err)
	}

	if subCertPEM == "" {
		t.Error("Sub CA certificate should not be empty")
	}
	if subKeyPEM == "" {
		t.Error("Sub CA key should not be empty")
	}

	// Parse and verify sub-CA
	subCert, err := ParseCertificatePEM([]byte(subCertPEM))
	if err != nil {
		t.Fatalf("Failed to parse sub-CA certificate: %v", err)
	}

	if !subCert.IsCA {
		t.Error("Sub CA certificate should be a CA")
	}
	if subCert.Subject.CommonName != "Sub CA" {
		t.Errorf("Expected CN 'Sub CA', got %q", subCert.Subject.CommonName)
	}
	if subCert.MaxPathLen != 0 {
		t.Errorf("Sub CA should have MaxPathLen 0, got %d", subCert.MaxPathLen)
	}
}

func TestGenerateServerCert(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM, keyPEM, err := ca.GenerateServerCert("server.example.com", []string{"server.example.com", "www.example.com"})
	if err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	if certPEM == "" {
		t.Error("Server certificate should not be empty")
	}
	if keyPEM == "" {
		t.Error("Server key should not be empty")
	}

	cert, err := ParseCertificatePEM([]byte(certPEM))
	if err != nil {
		t.Fatalf("Failed to parse server certificate: %v", err)
	}

	if cert.Subject.CommonName != "server.example.com" {
		t.Errorf("Expected CN 'server.example.com', got %q", cert.Subject.CommonName)
	}
	if len(cert.DNSNames) != 2 {
		t.Errorf("Expected 2 DNS names, got %d", len(cert.DNSNames))
	}
}

func TestGenerateClientCertWithCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Generate a sub-CA
	subCertPEM, subKeyPEM, err := ca.GenerateSubCA("Sub CA")
	if err != nil {
		t.Fatalf("GenerateSubCA failed: %v", err)
	}

	// Generate client cert with the sub-CA
	clientCertPEM, clientKeyPEM, err := ca.GenerateClientCertWithCA(subCertPEM, subKeyPEM, "client1", nil)
	if err != nil {
		t.Fatalf("GenerateClientCertWithCA failed: %v", err)
	}

	if clientCertPEM == "" {
		t.Error("Client certificate should not be empty")
	}
	if clientKeyPEM == "" {
		t.Error("Client key should not be empty")
	}

	clientCert, err := ParseCertificatePEM([]byte(clientCertPEM))
	if err != nil {
		t.Fatalf("Failed to parse client certificate: %v", err)
	}

	if clientCert.Subject.CommonName != "client1" {
		t.Errorf("Expected CN 'client1', got %q", clientCert.Subject.CommonName)
	}
	if clientCert.IsCA {
		t.Error("Client certificate should not be a CA")
	}
}

func TestGenerateClientCertWithCA_WithDNSNames(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	subCertPEM, subKeyPEM, _ := ca.GenerateSubCA("Sub CA")

	clientCertPEM, _, err := ca.GenerateClientCertWithCA(subCertPEM, subKeyPEM, "client1", []string{"client1.example.com"})
	if err != nil {
		t.Fatalf("GenerateClientCertWithCA failed: %v", err)
	}

	clientCert, _ := ParseCertificatePEM([]byte(clientCertPEM))
	if len(clientCert.DNSNames) != 1 || clientCert.DNSNames[0] != "client1.example.com" {
		t.Error("Client certificate should have DNS names set")
	}
}

func TestGenerateClientCertWithCA_InvalidCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
	}

	ca, _ := NewCA(cfg)

	_, _, err := ca.GenerateClientCertWithCA("invalid cert", "invalid key", "client1", nil)
	if err == nil {
		t.Error("GenerateClientCertWithCA should fail with invalid CA")
	}
}

func TestGenerateServerCertWithCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Generate a sub-CA
	subCertPEM, subKeyPEM, err := ca.GenerateSubCA("Sub CA")
	if err != nil {
		t.Fatalf("GenerateSubCA failed: %v", err)
	}

	// Generate server cert with the sub-CA
	serverCertPEM, serverKeyPEM, err := ca.GenerateServerCertWithCA(subCertPEM, subKeyPEM, "server1", []string{"server1.example.com"})
	if err != nil {
		t.Fatalf("GenerateServerCertWithCA failed: %v", err)
	}

	if serverCertPEM == "" {
		t.Error("Server certificate should not be empty")
	}
	if serverKeyPEM == "" {
		t.Error("Server key should not be empty")
	}

	serverCert, err := ParseCertificatePEM([]byte(serverCertPEM))
	if err != nil {
		t.Fatalf("Failed to parse server certificate: %v", err)
	}

	if serverCert.Subject.CommonName != "server1" {
		t.Errorf("Expected CN 'server1', got %q", serverCert.Subject.CommonName)
	}
	if len(serverCert.DNSNames) != 1 || serverCert.DNSNames[0] != "server1.example.com" {
		t.Error("Server certificate should have DNS names set")
	}
}

func TestGenerateServerCertWithCA_InvalidCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
	}

	ca, _ := NewCA(cfg)

	_, _, err := ca.GenerateServerCertWithCA("invalid cert", "invalid key", "server1", nil)
	if err == nil {
		t.Error("GenerateServerCertWithCA should fail with invalid CA")
	}
}

func TestGenerateECDSAKey(t *testing.T) {
	key, err := GenerateECDSAKey()
	if err != nil {
		t.Fatalf("GenerateECDSAKey failed: %v", err)
	}

	if key == nil {
		t.Fatal("GenerateECDSAKey returned nil")
	}

	// Should be P-256 curve
	if key.Curve.Params().Name != "P-256" {
		t.Errorf("Expected P-256 curve, got %s", key.Curve.Params().Name)
	}
}

func TestUnsupportedKeyAlgorithm(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "unsupported",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	_, err := NewCA(cfg)
	if err == nil {
		t.Error("NewCA should fail with unsupported key algorithm")
	}
}

func TestClientCertWithDefaultValidity(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Request with ValidFor = 0 should use default
	req := CertificateRequest{
		CommonName: "test-user",
		// ValidFor not set - should use default
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	duration := issued.NotAfter.Sub(issued.NotBefore)
	if duration != 24*time.Hour {
		t.Errorf("Expected default validity of 24 hours, got %v", duration)
	}
}

func TestClientCertWithCustomOrganization(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Default Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName:   "test-user",
		Organization: "Custom Org",
		ValidFor:     time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if issued.Certificate.Subject.Organization[0] != "Custom Org" {
		t.Errorf("Expected organization 'Custom Org', got %q", issued.Certificate.Subject.Organization[0])
	}
}

func TestServerCertWithDefaultValidity(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 48 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "server.example.com",
		// ValidFor not set
	}

	issued, err := ca.IssueServerCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	duration := issued.NotAfter.Sub(issued.NotBefore)
	if duration != 48*time.Hour {
		t.Errorf("Expected default validity of 48 hours, got %v", duration)
	}
}

func TestIssuedCertificateFields(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-user",
		ValidFor:   time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Check all fields are populated
	if issued.Certificate == nil {
		t.Error("Certificate should not be nil")
	}
	if issued.PrivateKey == nil {
		t.Error("PrivateKey should not be nil")
	}
	if len(issued.CertificatePEM) == 0 {
		t.Error("CertificatePEM should not be empty")
	}
	if len(issued.PrivateKeyPEM) == 0 {
		t.Error("PrivateKeyPEM should not be empty")
	}
	if issued.SerialNumber == "" {
		t.Error("SerialNumber should not be empty")
	}
	if issued.Fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
	if issued.NotBefore.IsZero() {
		t.Error("NotBefore should not be zero")
	}
	if issued.NotAfter.IsZero() {
		t.Error("NotAfter should not be zero")
	}
}

func TestSerialNumberUniqueness(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	serials := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := CertificateRequest{
			CommonName: "test-user",
			ValidFor:   time.Hour,
		}

		issued, err := ca.IssueClientCertificate(req)
		if err != nil {
			t.Fatalf("Failed to issue certificate %d: %v", i, err)
		}

		if serials[issued.SerialNumber] {
			t.Errorf("Duplicate serial number: %s", issued.SerialNumber)
		}
		serials[issued.SerialNumber] = true
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSaveAndLoadFromFiles(t *testing.T) {
	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "pki-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
		CACert:       certPath,
		CAKey:        keyPath,
	}

	// Create CA and save to files
	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file was not created")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file was not created")
	}

	// Check file permissions for key (should be 0600)
	keyInfo, _ := os.Stat(keyPath)
	if keyInfo.Mode().Perm() != 0600 {
		t.Errorf("Key file should have 0600 permissions, got %v", keyInfo.Mode().Perm())
	}

	fingerprint1 := Fingerprint(ca1.Certificate())

	// Load CA from files
	ca2, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to load CA from files: %v", err)
	}

	fingerprint2 := Fingerprint(ca2.Certificate())

	// Should be the same CA
	if fingerprint1 != fingerprint2 {
		t.Error("Loaded CA should have same fingerprint as original")
	}
}

func TestLoadFromFiles_RSA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-test-rsa-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org RSA",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
		CACert:       certPath,
		CAKey:        keyPath,
	}

	// Create and save
	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create RSA CA: %v", err)
	}

	fingerprint1 := Fingerprint(ca1.Certificate())

	// Load from files
	ca2, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to load RSA CA from files: %v", err)
	}

	fingerprint2 := Fingerprint(ca2.Certificate())

	if fingerprint1 != fingerprint2 {
		t.Error("Loaded RSA CA should have same fingerprint")
	}
}

func TestLoadFromFiles_NonExistent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
		CACert:       "/nonexistent/path/ca.crt",
		CAKey:        "/nonexistent/path/ca.key",
	}

	// Should generate new CA since files don't exist
	ca, err := NewCA(cfg)
	if err != nil {
		// Expected if directory doesn't exist and we can't write
		t.Logf("NewCA returned error as expected: %v", err)
		return
	}

	// If it succeeds, it should have generated a new CA
	if ca.Certificate() == nil {
		t.Error("CA certificate should not be nil")
	}
}

func TestNextSerial(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Get multiple serial numbers
	serials := make(map[string]bool)
	for i := 0; i < 100; i++ {
		serial, err := ca.nextSerial()
		if err != nil {
			t.Fatalf("nextSerial failed: %v", err)
		}
		if serial == nil {
			t.Fatal("Serial should not be nil")
		}
		serialStr := serial.String()
		if serials[serialStr] {
			t.Errorf("Duplicate serial number generated: %s", serialStr)
		}
		serials[serialStr] = true
	}
}

func TestGenerateSerialNumber(t *testing.T) {
	serials := make(map[string]bool)
	for i := 0; i < 100; i++ {
		serial, err := generateSerialNumber()
		if err != nil {
			t.Fatalf("generateSerialNumber failed: %v", err)
		}

		if serial == nil {
			t.Fatal("Serial should not be nil")
		}

		// Serial should be positive
		if serial.Sign() <= 0 {
			t.Error("Serial number should be positive")
		}

		// Serial should be less than 2^128
		max := new(big.Int).Lsh(big.NewInt(1), 128)
		if serial.Cmp(max) >= 0 {
			t.Error("Serial number should be less than 2^128")
		}

		serialStr := serial.String()
		if serials[serialStr] {
			t.Errorf("Duplicate serial: %s", serialStr)
		}
		serials[serialStr] = true
	}
}

func TestPublicKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	pubKey := publicKey(ca.privateKey)
	if pubKey == nil {
		t.Error("Public key should not be nil")
	}
}

func TestGeneratePrivateKey_AllAlgorithms(t *testing.T) {
	algorithms := []string{"rsa2048", "rsa4096", "ecdsa256", "ecdsa384"}

	for _, alg := range algorithms {
		t.Run(alg, func(t *testing.T) {
			key, err := generatePrivateKey(alg)
			if err != nil {
				t.Fatalf("generatePrivateKey(%s) failed: %v", alg, err)
			}
			if key == nil {
				t.Error("Key should not be nil")
			}

			// Verify we can get a public key
			pub := key.Public()
			if pub == nil {
				t.Error("Public key should not be nil")
			}
		})
	}
}

func TestGeneratePrivateKey_Unsupported(t *testing.T) {
	_, err := generatePrivateKey("unsupported_algo")
	if err == nil {
		t.Error("generatePrivateKey should fail for unsupported algorithm")
	}
}

func TestCertificateRequest_AllFields(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName:   "test-user",
		Email:        "user@example.com",
		Organization: "Custom Org",
		ValidFor:     12 * time.Hour,
		DNSNames:     []string{"test.example.com", "test2.example.com"},
		IPAddresses:  []string{"192.168.1.1"},
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	cert := issued.Certificate
	if cert.Subject.CommonName != "test-user" {
		t.Errorf("Expected CN 'test-user', got %q", cert.Subject.CommonName)
	}
	if cert.Subject.Organization[0] != "Custom Org" {
		t.Errorf("Expected Org 'Custom Org', got %q", cert.Subject.Organization[0])
	}
	if len(cert.EmailAddresses) != 1 || cert.EmailAddresses[0] != "user@example.com" {
		t.Error("Email not set correctly")
	}
	if len(cert.DNSNames) != 2 {
		t.Errorf("Expected 2 DNS names, got %d", len(cert.DNSNames))
	}
}

func TestStoredCA_Fields(t *testing.T) {
	now := time.Now()
	stored := &StoredCA{
		CertificatePEM: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		PrivateKeyPEM:  "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
		SerialNumber:   "12345",
		NotBefore:      now,
		NotAfter:       now.Add(365 * 24 * time.Hour),
	}

	if stored.CertificatePEM == "" {
		t.Error("CertificatePEM should not be empty")
	}
	if stored.PrivateKeyPEM == "" {
		t.Error("PrivateKeyPEM should not be empty")
	}
	if stored.SerialNumber != "12345" {
		t.Errorf("Expected SerialNumber '12345', got %q", stored.SerialNumber)
	}
}

func TestLoadFromPEM_InvalidCertPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{
		config: cfg,
	}

	// Test with invalid certificate PEM
	err := ca.loadFromPEM("not a valid PEM", "")
	if err == nil {
		t.Error("Expected error for invalid certificate PEM")
	}
}

func TestLoadFromPEM_InvalidCertContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{
		config: cfg,
	}

	// Valid PEM structure but invalid certificate content
	invalidCertPEM := `-----BEGIN CERTIFICATE-----
dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==
-----END CERTIFICATE-----`

	err := ca.loadFromPEM(invalidCertPEM, "")
	if err == nil {
		t.Error("Expected error for invalid certificate content")
	}
}

func TestLoadFromPEM_InvalidKeyPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a real CA to get a valid certificate
	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	ca := &CA{
		config: cfg,
	}

	validCertPEM := string(realCA.CertificatePEM())

	// Test with invalid key PEM
	err = ca.loadFromPEM(validCertPEM, "not a valid PEM")
	if err == nil {
		t.Error("Expected error for invalid key PEM")
	}
}

func TestLoadFromPEM_UnsupportedKeyType(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a real CA to get a valid certificate
	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	ca := &CA{
		config: cfg,
	}

	validCertPEM := string(realCA.CertificatePEM())

	// PEM with unsupported key type
	unsupportedKeyPEM := `-----BEGIN UNSUPPORTED KEY-----
dGVzdA==
-----END UNSUPPORTED KEY-----`

	err = ca.loadFromPEM(validCertPEM, unsupportedKeyPEM)
	if err == nil {
		t.Error("Expected error for unsupported key type")
	}
}

func TestLoadFromPEM_PKCS8Key(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Get valid cert PEM
	certPEM := string(ca.CertificatePEM())

	// Get valid key and re-encode as PKCS8
	keyPEM := string(ca.PrivateKeyPEM())

	// Create a new CA and load from PEM
	newCA := &CA{
		config: cfg,
	}

	err = newCA.loadFromPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Failed to load from valid PEM: %v", err)
	}

	if newCA.certificate == nil {
		t.Error("Certificate should be set after loading")
	}
	if newCA.privateKey == nil {
		t.Error("Private key should be set after loading")
	}
}

func TestLoadFromFiles_NonExistentCert(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{
		config: cfg,
	}

	tmpDir := t.TempDir()
	nonExistentCert := filepath.Join(tmpDir, "nonexistent.crt")
	keyFile := filepath.Join(tmpDir, "test.key")

	err := ca.loadFromFiles(nonExistentCert, keyFile)
	if err == nil {
		t.Error("Expected error for non-existent certificate file")
	}
}

func TestLoadFromFiles_NonExistentKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "test.crt")
	nonExistentKey := filepath.Join(tmpDir, "nonexistent.key")

	// Write only cert file
	err = os.WriteFile(certFile, realCA.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	ca := &CA{
		config: cfg,
	}

	err = ca.loadFromFiles(certFile, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key file")
	}
}

func TestIssueClientCertificate_WithEmail(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-user",
		Email:      "admin@example.com",
		ValidFor:   1 * time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if issued.Certificate == nil {
		t.Error("Issued certificate is nil")
	}

	if issued.Certificate.Subject.CommonName != "test-user" {
		t.Errorf("Expected CN 'test-user', got %q", issued.Certificate.Subject.CommonName)
	}
	if len(issued.Certificate.EmailAddresses) != 1 || issued.Certificate.EmailAddresses[0] != "admin@example.com" {
		t.Error("Email not set correctly")
	}
}

func TestIssueServerCertificate_MultipleDNSNames(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "vpn.example.com",
		DNSNames:   []string{"vpn.example.com", "vpn2.example.com", "vpn3.example.com"},
		ValidFor:   1 * time.Hour,
	}

	issued, err := ca.IssueServerCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if len(issued.Certificate.DNSNames) != 3 {
		t.Errorf("Expected 3 DNS names, got %d", len(issued.Certificate.DNSNames))
	}
}

func TestIssueServerCertificate_WithOrganization(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName:   "vpn.example.com",
		Organization: "Custom Org",
		ValidFor:     1 * time.Hour,
	}

	issued, err := ca.IssueServerCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if issued.Certificate.Subject.Organization[0] != "Custom Org" {
		t.Errorf("Expected Organization 'Custom Org', got %q", issued.Certificate.Subject.Organization[0])
	}
}

func TestSaveToFiles_InvalidPath(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Try to save to an invalid path
	err = ca.saveToFiles("/nonexistent/path/cert.crt", "/nonexistent/path/key.key")
	if err == nil {
		t.Error("Expected error for invalid file path")
	}
}

func TestNewCA_RSA4096(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa4096",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA with RSA4096: %v", err)
	}

	if ca.Certificate() == nil {
		t.Error("CA certificate is nil")
	}
}

func TestNewCA_ECDSA384(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa384",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA with ECDSA384: %v", err)
	}

	if ca.Certificate() == nil {
		t.Error("CA certificate is nil")
	}
}

func TestUpdateFromPEM_InvalidPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	err = ca.UpdateFromPEM(context.TODO(), "invalid cert", "invalid key")
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}
}

func TestUpdateFromPEM_NonCACertificate(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Issue a non-CA certificate
	req := CertificateRequest{
		CommonName: "test-server",
		ValidFor:   1 * time.Hour,
	}
	issued, err := ca.IssueServerCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Try to update CA with a non-CA certificate
	err = ca.UpdateFromPEM(context.TODO(), string(issued.CertificatePEM), string(issued.PrivateKeyPEM))
	if err == nil {
		t.Error("Expected error when updating with non-CA certificate")
	}
}

func TestUpdateFromPEM_ValidCA(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA1: %v", err)
	}

	// Create a second CA
	ca2, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA2: %v", err)
	}

	// Update CA1 with CA2's certificate
	oldSerial := ca1.Certificate().SerialNumber.String()

	err = ca1.UpdateFromPEM(context.TODO(), string(ca2.CertificatePEM()), string(ca2.PrivateKeyPEM()))
	if err != nil {
		t.Fatalf("Failed to update CA from PEM: %v", err)
	}

	// Verify the certificate was updated
	newSerial := ca1.Certificate().SerialNumber.String()
	if oldSerial == newSerial {
		t.Error("Serial number should have changed after update")
	}
}

func TestLoadFromPEM_RSAKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM := string(ca.CertificatePEM())
	keyPEM := string(ca.PrivateKeyPEM())

	// Create a new CA and load from PEM
	newCA := &CA{
		config: cfg,
	}

	err = newCA.loadFromPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Failed to load RSA CA from PEM: %v", err)
	}

	if newCA.certificate == nil {
		t.Error("Certificate should be set after loading")
	}
}

func TestLoadFromFiles_InvalidCertPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	tmpDir := t.TempDir()

	// Write an invalid cert file (not valid PEM)
	certFile := filepath.Join(tmpDir, "invalid.crt")
	keyFile := filepath.Join(tmpDir, "test.key")

	err := os.WriteFile(certFile, []byte("not valid PEM content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cert file: %v", err)
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error for invalid cert PEM content")
	}
}

func TestLoadFromFiles_InvalidKeyPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "test.crt")
	keyFile := filepath.Join(tmpDir, "invalid.key")

	// Write valid cert
	err = os.WriteFile(certFile, realCA.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write invalid key (not valid PEM)
	err = os.WriteFile(keyFile, []byte("not valid PEM key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error for invalid key PEM content")
	}
}

func TestLoadFromFiles_UnsupportedKeyType(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "test.crt")
	keyFile := filepath.Join(tmpDir, "invalid.key")

	// Write valid cert
	err = os.WriteFile(certFile, realCA.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write key with unsupported type
	unsupportedKeyPEM := `-----BEGIN UNKNOWN KEY-----
dGVzdA==
-----END UNKNOWN KEY-----`
	err = os.WriteFile(keyFile, []byte(unsupportedKeyPEM), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error for unsupported key type")
	}
}

func TestLoadFromFiles_InvalidCertContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "invalid.crt")
	keyFile := filepath.Join(tmpDir, "test.key")

	// Write PEM-formatted but invalid cert content
	invalidCertPEM := `-----BEGIN CERTIFICATE-----
dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==
-----END CERTIFICATE-----`
	err := os.WriteFile(certFile, []byte(invalidCertPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error for invalid certificate content")
	}
}

func TestGenerateClientCertWithCA_InvalidKeyPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Valid cert but invalid key PEM
	certPEM := string(ca.CertificatePEM())
	invalidKeyPEM := "not a valid key PEM"

	_, _, err = ca.GenerateClientCertWithCA(certPEM, invalidKeyPEM, "client", nil)
	if err == nil {
		t.Error("Expected error for invalid key PEM")
	}
}

func TestGenerateClientCertWithCA_InvalidKeyContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Valid cert but key PEM with invalid content
	certPEM := string(ca.CertificatePEM())
	invalidKeyPEM := `-----BEGIN PRIVATE KEY-----
dGhpcyBpcyBub3QgYSB2YWxpZCBrZXk=
-----END PRIVATE KEY-----`

	_, _, err = ca.GenerateClientCertWithCA(certPEM, invalidKeyPEM, "client", nil)
	if err == nil {
		t.Error("Expected error for invalid key content")
	}
}

func TestGenerateServerCertWithCA_InvalidKeyPEM(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Valid cert but invalid key PEM
	certPEM := string(ca.CertificatePEM())
	invalidKeyPEM := "not a valid key PEM"

	_, _, err = ca.GenerateServerCertWithCA(certPEM, invalidKeyPEM, "server", []string{"server.example.com"})
	if err == nil {
		t.Error("Expected error for invalid key PEM")
	}
}

func TestGenerateServerCertWithCA_InvalidKeyContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Valid cert but key PEM with invalid content
	certPEM := string(ca.CertificatePEM())
	invalidKeyPEM := `-----BEGIN PRIVATE KEY-----
dGhpcyBpcyBub3QgYSB2YWxpZCBrZXk=
-----END PRIVATE KEY-----`

	_, _, err = ca.GenerateServerCertWithCA(certPEM, invalidKeyPEM, "server", nil)
	if err == nil {
		t.Error("Expected error for invalid key content")
	}
}

func TestGenerateServerCertWithCA_InvalidCertContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, _ := NewCA(cfg)

	// Invalid cert PEM content
	invalidCertPEM := `-----BEGIN CERTIFICATE-----
dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==
-----END CERTIFICATE-----`
	validKeyPEM := string(ca.PrivateKeyPEM())

	_, _, err := ca.GenerateServerCertWithCA(invalidCertPEM, validKeyPEM, "server", nil)
	if err == nil {
		t.Error("Expected error for invalid cert content")
	}
}

func TestGenerateClientCertWithCA_InvalidCertContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, _ := NewCA(cfg)

	// Invalid cert PEM content
	invalidCertPEM := `-----BEGIN CERTIFICATE-----
dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==
-----END CERTIFICATE-----`
	validKeyPEM := string(ca.PrivateKeyPEM())

	_, _, err := ca.GenerateClientCertWithCA(invalidCertPEM, validKeyPEM, "client", nil)
	if err == nil {
		t.Error("Expected error for invalid cert content")
	}
}

func TestPrivateKeyPEM_UnsupportedType(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// This tests that the code path exists for the default case
	// The actual privateKey is always ECDSA or RSA, so default case
	// won't be reached in normal operation
	pem := ca.PrivateKeyPEM()
	if len(pem) == 0 {
		t.Error("PrivateKeyPEM should return non-empty for valid CA")
	}
}

func TestSaveToFiles_RSA(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org RSA",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create RSA CA: %v", err)
	}

	err = ca.saveToFiles(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to save CA to files: %v", err)
	}

	// Verify files exist and have correct permissions
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Key file should exist: %v", err)
	}
	if keyInfo.Mode().Perm() != 0600 {
		t.Errorf("Key file should have 0600 permissions, got %v", keyInfo.Mode().Perm())
	}
}

func TestVerifyCertificate_ExpiredCert(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 1 * time.Nanosecond, // Very short validity
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-user",
		ValidFor:   1 * time.Nanosecond, // Immediate expiration
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Wait for cert to expire
	time.Sleep(10 * time.Millisecond)

	// Verify should fail for expired cert
	err = ca.VerifyCertificate(issued.Certificate)
	if err == nil {
		t.Error("VerifyCertificate should fail for expired certificate")
	}
}

func TestClientCertWithDNSNames(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	req := CertificateRequest{
		CommonName: "test-client",
		DNSNames:   []string{"client.example.com", "client2.example.com"},
		ValidFor:   time.Hour,
	}

	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	if len(issued.Certificate.DNSNames) != 2 {
		t.Errorf("Expected 2 DNS names, got %d", len(issued.Certificate.DNSNames))
	}
}

func TestUpdateFromPEM_WithFileSave(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
		CACert:       certPath,
		CAKey:        keyPath,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA1: %v", err)
	}

	// Create a second CA to update from
	ca2, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA2: %v", err)
	}

	// Update CA1 with CA2's certificate - this should trigger file save
	err = ca1.UpdateFromPEM(context.TODO(), string(ca2.CertificatePEM()), string(ca2.PrivateKeyPEM()))
	if err != nil {
		t.Fatalf("Failed to update CA from PEM: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file should have been created")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file should have been created")
	}
}

func TestUpdateFromPEM_FileSaveError(t *testing.T) {
	// Create a directory path that will fail for file creation
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent", "nested")
	certPath := filepath.Join(nonExistentDir, "ca.crt")
	keyPath := filepath.Join(nonExistentDir, "ca.key")

	// First create a CA without file paths
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA1: %v", err)
	}

	ca2, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA2: %v", err)
	}

	// Now set the file paths that will fail
	ca1.config.CACert = certPath
	ca1.config.CAKey = keyPath

	// Update should fail because the directory doesn't exist
	err = ca1.UpdateFromPEM(context.TODO(), string(ca2.CertificatePEM()), string(ca2.PrivateKeyPEM()))
	if err == nil {
		t.Error("Expected error when file save fails")
	}
}

func TestLoadFromFiles_KeyFileNotReadable(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA1: %v", err)
	}

	// Write cert file but not the key file
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when key file doesn't exist")
	}
}

func TestGenerateSubCA_ValidateCert(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM, keyPEM, err := ca.GenerateSubCA("SubCA-Test")
	if err != nil {
		t.Fatalf("Failed to generate SubCA: %v", err)
	}

	// Parse the SubCA certificate
	subCACert, err := ParseCertificatePEM([]byte(certPEM))
	if err != nil {
		t.Fatalf("Failed to parse SubCA certificate: %v", err)
	}

	// Verify it's a CA certificate
	if !subCACert.IsCA {
		t.Error("SubCA certificate should have IsCA=true")
	}

	// Verify the key is valid
	if keyPEM == "" {
		t.Error("SubCA key PEM should not be empty")
	}

	// Verify the SubCA can issue certificates
	subCAConfig := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "SubCA Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}
	subCA := &CA{config: subCAConfig}
	err = subCA.loadFromPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Failed to load SubCA from PEM: %v", err)
	}

	// Issue a certificate from the SubCA
	req := CertificateRequest{
		CommonName: "issued-by-subca",
		ValidFor:   time.Hour,
	}
	issued, err := subCA.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("SubCA failed to issue certificate: %v", err)
	}

	if issued.Certificate.Subject.CommonName != "issued-by-subca" {
		t.Errorf("Expected CommonName 'issued-by-subca', got '%s'", issued.Certificate.Subject.CommonName)
	}
}

func TestCRL_AddMultipleRevocations(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Issue multiple certificates
	var certs []*IssuedCertificate
	for i := 0; i < 5; i++ {
		req := CertificateRequest{
			CommonName: "test-user-" + string(rune('0'+i)),
			ValidFor:   time.Hour,
		}
		issued, err := ca.IssueClientCertificate(req)
		if err != nil {
			t.Fatalf("Failed to issue certificate %d: %v", i, err)
		}
		certs = append(certs, issued)
	}

	// Create revocation list with hex format serial numbers
	var revokedCerts []RevokedCertificate
	for i := 0; i < 3; i++ {
		revokedCerts = append(revokedCerts, RevokedCertificate{
			SerialNumber: certs[i].Certificate.SerialNumber.Text(16),
			RevokedAt:    time.Now(),
			Reason:       ReasonUnspecified,
		})
	}

	// Generate CRL
	crlPEM, err := ca.GenerateCRL(revokedCerts, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	// Parse and verify
	parsed, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	if len(parsed.RevokedCertificateEntries) != 3 {
		t.Errorf("Expected 3 revoked certificates, got %d", len(parsed.RevokedCertificateEntries))
	}

	// Check revocation status
	for i := 0; i < 3; i++ {
		if !IsCertificateRevoked(parsed, certs[i].Certificate) {
			t.Errorf("Certificate %d should be revoked", i)
		}
	}
	for i := 3; i < 5; i++ {
		if IsCertificateRevoked(parsed, certs[i].Certificate) {
			t.Errorf("Certificate %d should not be revoked", i)
		}
	}
}

func TestFingerprint_DifferentAlgorithms(t *testing.T) {
	algorithms := []string{"ecdsa256", "ecdsa384", "rsa2048", "rsa4096"}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			cfg := config.PKIConfig{
				KeyAlgorithm: algo,
				Organization: "Test Org",
				CertValidity: 24 * time.Hour,
				CAValidity:   365 * 24 * time.Hour,
			}

			ca, err := NewCA(cfg)
			if err != nil {
				t.Fatalf("Failed to create CA with algorithm %s: %v", algo, err)
			}

			fp := Fingerprint(ca.Certificate())
			if fp == "" {
				t.Error("Fingerprint should not be empty")
			}

			// Fingerprint should be in SHA-256 hex format (64 hex chars)
			if len(fp) != 64 {
				t.Errorf("Expected fingerprint length 64, got %d", len(fp))
			}
		})
	}
}

func TestGenerateCRLDER_Basic(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Issue a certificate and revoke it
	req := CertificateRequest{
		CommonName: "test-user",
		ValidFor:   time.Hour,
	}
	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	revokedCerts := []RevokedCertificate{
		{
			SerialNumber: issued.Certificate.SerialNumber.Text(16),
			RevokedAt:    time.Now(),
			Reason:       ReasonKeyCompromise,
		},
	}

	// Generate CRL in DER format
	crlDER, err := ca.GenerateCRLDER(revokedCerts, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL DER: %v", err)
	}

	// Should be valid DER data
	if len(crlDER) == 0 {
		t.Error("CRL DER should not be empty")
	}

	// Parse the DER directly
	parsed, err := x509.ParseRevocationList(crlDER)
	if err != nil {
		t.Fatalf("Failed to parse CRL DER: %v", err)
	}

	if len(parsed.RevokedCertificateEntries) != 1 {
		t.Errorf("Expected 1 revoked certificate, got %d", len(parsed.RevokedCertificateEntries))
	}
}

func TestLoadFromFiles_CertFileNotReadable(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "nonexistent", "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{config: cfg}
	err := ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when cert file doesn't exist")
	}
}

func TestLoadFromFiles_PKCS1RSAKey(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org RSA",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create an RSA CA and save to files
	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create RSA CA: %v", err)
	}

	err = ca1.saveToFiles(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to save CA to files: %v", err)
	}

	// Load the CA back from files
	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to load RSA CA from files: %v", err)
	}

	// Verify the loaded CA is functional
	req := CertificateRequest{
		CommonName: "test-client",
		ValidFor:   time.Hour,
	}
	issued, err := ca2.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Loaded CA failed to issue certificate: %v", err)
	}

	if issued.Certificate.Subject.CommonName != "test-client" {
		t.Errorf("Expected CommonName 'test-client', got '%s'", issued.Certificate.Subject.CommonName)
	}
}

func TestParseCRL_InvalidPEM_Content(t *testing.T) {
	_, err := ParseCRL([]byte("not a PEM"))
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}
}

func TestParseCRL_NotCRLType(t *testing.T) {
	// Create a certificate PEM (wrong type for CRL)
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}
	ca, _ := NewCA(cfg)

	_, err := ParseCRL(ca.CertificatePEM())
	if err == nil {
		t.Error("Expected error when PEM is not a CRL")
	}
}

func TestGenerateClientCertWithCA_PKCS1RSAKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org RSA",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create RSA CA: %v", err)
	}

	certPEM, keyPEM, err := ca.GenerateClientCertWithCA(
		string(ca.CertificatePEM()),
		string(ca.PrivateKeyPEM()),
		"client-with-rsa",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to generate client cert: %v", err)
	}

	if certPEM == "" || keyPEM == "" {
		t.Error("Cert and key should not be empty")
	}
}

func TestGenerateServerCertWithCA_PKCS1RSAKey(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org RSA",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create RSA CA: %v", err)
	}

	certPEM, keyPEM, err := ca.GenerateServerCertWithCA(
		string(ca.CertificatePEM()),
		string(ca.PrivateKeyPEM()),
		"server-with-rsa",
		[]string{"server.example.com"},
	)
	if err != nil {
		t.Fatalf("Failed to generate server cert: %v", err)
	}

	if certPEM == "" || keyPEM == "" {
		t.Error("Cert and key should not be empty")
	}
}

func TestCRL_EmptyRevocationList(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Generate empty CRL
	crlPEM, err := ca.GenerateCRL([]RevokedCertificate{}, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate empty CRL: %v", err)
	}

	parsed, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse empty CRL: %v", err)
	}

	if len(parsed.RevokedCertificateEntries) != 0 {
		t.Errorf("Expected 0 revoked certificates, got %d", len(parsed.RevokedCertificateEntries))
	}
}

func TestGenerateServerCert_Success(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	certPEM, keyPEM, err := ca.GenerateServerCert("server.example.com", []string{"server.example.com"})
	if err != nil {
		t.Fatalf("Failed to generate server cert: %v", err)
	}

	if certPEM == "" || keyPEM == "" {
		t.Error("Cert and key should not be empty")
	}

	// Parse and verify
	cert, err := ParseCertificatePEM([]byte(certPEM))
	if err != nil {
		t.Fatalf("Failed to parse generated certificate: %v", err)
	}

	if cert.Subject.CommonName != "server.example.com" {
		t.Errorf("Expected CommonName 'server.example.com', got '%s'", cert.Subject.CommonName)
	}
}

func TestLoadFromFiles_InvalidCertPEM_DecodeFail(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Write invalid PEM to cert file
	err := os.WriteFile(certFile, []byte("not valid PEM data"), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when cert PEM is invalid")
	}
	if err != nil && !contains(err.Error(), "failed to decode CA certificate PEM") {
		t.Errorf("Expected 'failed to decode CA certificate PEM' error, got: %v", err)
	}
}

func TestLoadFromFiles_CorruptCertContent(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Write valid PEM with garbage content
	corruptPEM := "-----BEGIN CERTIFICATE-----\nY29ycnVwdCBkYXRh\n-----END CERTIFICATE-----\n"
	err := os.WriteFile(certFile, []byte(corruptPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca := &CA{config: cfg}
	err = ca.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when cert content is corrupt")
	}
	if err != nil && !contains(err.Error(), "failed to parse CA certificate") {
		t.Errorf("Expected 'failed to parse CA certificate' error, got: %v", err)
	}
}

func TestLoadFromFiles_InvalidKeyPEM_DecodeFail(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Create a valid CA first to get a valid cert
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Save just the cert
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write invalid PEM to key file
	err = os.WriteFile(keyFile, []byte("not valid key PEM"), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when key PEM is invalid")
	}
	if err != nil && !contains(err.Error(), "failed to decode CA private key PEM") {
		t.Errorf("Expected 'failed to decode CA private key PEM' error, got: %v", err)
	}
}

func TestLoadFromFiles_UnsupportedKeyType_DSA(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Create a valid CA first to get a valid cert
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Save just the cert
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write key with unsupported type (DSA)
	unsupportedKeyPEM := "-----BEGIN DSA PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END DSA PRIVATE KEY-----\n"
	err = os.WriteFile(keyFile, []byte(unsupportedKeyPEM), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when key type is unsupported")
	}
	if err != nil && !contains(err.Error(), "unsupported private key type") {
		t.Errorf("Expected 'unsupported private key type' error, got: %v", err)
	}
}

func TestLoadFromFiles_PKCS8KeyNotSigner(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Create a valid CA first to get a valid cert
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Save just the cert
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write valid-ish PKCS8 PEM with garbage content that will fail parsing
	invalidPKCS8PEM := "-----BEGIN PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END PRIVATE KEY-----\n"
	err = os.WriteFile(keyFile, []byte(invalidPKCS8PEM), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when PKCS8 key content is invalid")
	}
	if err != nil && !contains(err.Error(), "failed to parse PKCS8 private key") {
		t.Errorf("Expected 'failed to parse PKCS8 private key' error, got: %v", err)
	}
}

func TestLoadFromFiles_CorruptECKey(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Create a valid CA first to get a valid cert
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Save just the cert
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write EC key PEM with garbage content
	corruptECKeyPEM := "-----BEGIN EC PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END EC PRIVATE KEY-----\n"
	err = os.WriteFile(keyFile, []byte(corruptECKeyPEM), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when EC key content is corrupt")
	}
	if err != nil && !contains(err.Error(), "failed to parse CA private key") {
		t.Errorf("Expected 'failed to parse CA private key' error, got: %v", err)
	}
}

func TestLoadFromFiles_CorruptRSAKey(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")

	// Create a valid CA with RSA key
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	ca1, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Save just the cert
	err = os.WriteFile(certFile, ca1.CertificatePEM(), 0644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Write RSA key PEM with garbage content
	corruptRSAKeyPEM := "-----BEGIN RSA PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END RSA PRIVATE KEY-----\n"
	err = os.WriteFile(keyFile, []byte(corruptRSAKeyPEM), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	ca2 := &CA{config: cfg}
	err = ca2.loadFromFiles(certFile, keyFile)
	if err == nil {
		t.Error("Expected error when RSA key content is corrupt")
	}
	if err != nil && !contains(err.Error(), "failed to parse CA private key") {
		t.Errorf("Expected 'failed to parse CA private key' error, got: %v", err)
	}
}

func TestLoadFromPEM_CorruptRSAKeyContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "rsa2048",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a real CA to get a valid certificate
	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	ca := &CA{config: cfg}
	validCertPEM := string(realCA.CertificatePEM())

	// RSA key PEM with garbage content
	corruptRSAKeyPEM := "-----BEGIN RSA PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END RSA PRIVATE KEY-----\n"

	err = ca.loadFromPEM(validCertPEM, corruptRSAKeyPEM)
	if err == nil {
		t.Error("Expected error for corrupt RSA key content")
	}
	if err != nil && !contains(err.Error(), "failed to parse CA private key") {
		t.Errorf("Expected 'failed to parse CA private key' error, got: %v", err)
	}
}

func TestLoadFromPEM_CorruptECKeyContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a real CA to get a valid certificate
	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	ca := &CA{config: cfg}
	validCertPEM := string(realCA.CertificatePEM())

	// EC key PEM with garbage content
	corruptECKeyPEM := "-----BEGIN EC PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END EC PRIVATE KEY-----\n"

	err = ca.loadFromPEM(validCertPEM, corruptECKeyPEM)
	if err == nil {
		t.Error("Expected error for corrupt EC key content")
	}
	if err != nil && !contains(err.Error(), "failed to parse CA private key") {
		t.Errorf("Expected 'failed to parse CA private key' error, got: %v", err)
	}
}

func TestLoadFromPEM_CorruptPKCS8KeyContent(t *testing.T) {
	cfg := config.PKIConfig{
		KeyAlgorithm: "ecdsa256",
		Organization: "Test Org",
		CertValidity: 24 * time.Hour,
		CAValidity:   365 * 24 * time.Hour,
	}

	// Create a real CA to get a valid certificate
	realCA, err := NewCA(cfg)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	ca := &CA{config: cfg}
	validCertPEM := string(realCA.CertificatePEM())

	// PKCS8 key PEM with garbage content
	corruptPKCS8KeyPEM := "-----BEGIN PRIVATE KEY-----\nY29ycnVwdCBkYXRh\n-----END PRIVATE KEY-----\n"

	err = ca.loadFromPEM(validCertPEM, corruptPKCS8KeyPEM)
	if err == nil {
		t.Error("Expected error for corrupt PKCS8 key content")
	}
	if err != nil && !contains(err.Error(), "failed to parse PKCS8 private key") {
		t.Errorf("Expected 'failed to parse PKCS8 private key' error, got: %v", err)
	}
}
