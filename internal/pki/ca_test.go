package pki

import (
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
