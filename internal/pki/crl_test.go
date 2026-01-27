package pki

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/gatekey-project/gatekey/internal/config"
)

func TestGenerateCRL_Empty(t *testing.T) {
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

	// Generate CRL with no revoked certificates
	crlPEM, err := ca.GenerateCRL(nil, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	if len(crlPEM) == 0 {
		t.Error("CRL should not be empty")
	}

	// Parse and verify the CRL
	crl, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	if len(crl.RevokedCertificateEntries) != 0 {
		t.Errorf("Expected 0 revoked certificates, got %d", len(crl.RevokedCertificateEntries))
	}
}

func TestGenerateCRL_WithRevokedCerts(t *testing.T) {
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

	revokedCerts := []RevokedCertificate{
		{
			SerialNumber: "1234567890abcdef",
			RevokedAt:    time.Now().Add(-1 * time.Hour),
			Reason:       ReasonKeyCompromise,
		},
		{
			SerialNumber: "fedcba0987654321",
			RevokedAt:    time.Now().Add(-30 * time.Minute),
			Reason:       ReasonCessationOfOperation,
		},
	}

	crlPEM, err := ca.GenerateCRL(revokedCerts, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	crl, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	if len(crl.RevokedCertificateEntries) != 2 {
		t.Errorf("Expected 2 revoked certificates, got %d", len(crl.RevokedCertificateEntries))
	}
}

func TestGenerateCRL_DifferentReasons(t *testing.T) {
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

	reasons := []int{
		ReasonUnspecified,
		ReasonKeyCompromise,
		ReasonCACompromise,
		ReasonAffiliationChanged,
		ReasonSuperseded,
		ReasonCessationOfOperation,
		ReasonCertificateHold,
		ReasonPrivilegeWithdrawn,
		ReasonAACompromise,
	}

	for i, reason := range reasons {
		t.Run(reasonName(reason), func(t *testing.T) {
			revokedCerts := []RevokedCertificate{
				{
					SerialNumber: "abcd1234" + string(rune('0'+i)),
					RevokedAt:    time.Now(),
					Reason:       reason,
				},
			}

			_, err := ca.GenerateCRL(revokedCerts, 24*time.Hour)
			if err != nil {
				t.Errorf("Failed to generate CRL with reason %d: %v", reason, err)
			}
		})
	}
}

func TestGenerateCRLDER(t *testing.T) {
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

	revokedCerts := []RevokedCertificate{
		{
			SerialNumber: "abc123",
			RevokedAt:    time.Now(),
			Reason:       ReasonSuperseded,
		},
	}

	crlDER, err := ca.GenerateCRLDER(revokedCerts, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL DER: %v", err)
	}

	if len(crlDER) == 0 {
		t.Error("CRL DER should not be empty")
	}

	// Verify it can be parsed as DER
	crl, err := x509.ParseRevocationList(crlDER)
	if err != nil {
		t.Fatalf("Failed to parse CRL DER: %v", err)
	}

	if len(crl.RevokedCertificateEntries) != 1 {
		t.Errorf("Expected 1 revoked certificate, got %d", len(crl.RevokedCertificateEntries))
	}
}

func TestParseCRL_InvalidPEM(t *testing.T) {
	tests := []struct {
		name    string
		pemData []byte
	}{
		{
			name:    "empty",
			pemData: []byte{},
		},
		{
			name:    "not PEM",
			pemData: []byte("this is not PEM data"),
		},
		{
			name:    "wrong PEM type",
			pemData: []byte("-----BEGIN CERTIFICATE-----\nYWJjZA==\n-----END CERTIFICATE-----"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCRL(tt.pemData)
			if err == nil {
				t.Error("Expected error parsing invalid CRL")
			}
		})
	}
}

func TestIsCertificateRevoked(t *testing.T) {
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

	// Issue two certificates
	req1 := CertificateRequest{CommonName: "user1", ValidFor: time.Hour}
	issued1, err := ca.IssueClientCertificate(req1)
	if err != nil {
		t.Fatalf("Failed to issue certificate 1: %v", err)
	}

	req2 := CertificateRequest{CommonName: "user2", ValidFor: time.Hour}
	issued2, err := ca.IssueClientCertificate(req2)
	if err != nil {
		t.Fatalf("Failed to issue certificate 2: %v", err)
	}

	// Revoke only the first certificate
	revokedCerts := []RevokedCertificate{
		{
			SerialNumber: issued1.Certificate.SerialNumber.Text(16),
			RevokedAt:    time.Now(),
			Reason:       ReasonKeyCompromise,
		},
	}

	crlPEM, err := ca.GenerateCRL(revokedCerts, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	crl, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	// First certificate should be revoked
	if !IsCertificateRevoked(crl, issued1.Certificate) {
		t.Error("Certificate 1 should be revoked")
	}

	// Second certificate should not be revoked
	if IsCertificateRevoked(crl, issued2.Certificate) {
		t.Error("Certificate 2 should not be revoked")
	}
}

func TestIsCertificateRevoked_EmptyCRL(t *testing.T) {
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

	req := CertificateRequest{CommonName: "user", ValidFor: time.Hour}
	issued, err := ca.IssueClientCertificate(req)
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Generate empty CRL
	crlPEM, err := ca.GenerateCRL(nil, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	crl, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	// Certificate should not be revoked in empty CRL
	if IsCertificateRevoked(crl, issued.Certificate) {
		t.Error("Certificate should not be revoked in empty CRL")
	}
}

func TestCRLValidity(t *testing.T) {
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

	validity := 12 * time.Hour
	crlPEM, err := ca.GenerateCRL(nil, validity)
	if err != nil {
		t.Fatalf("Failed to generate CRL: %v", err)
	}

	crl, err := ParseCRL(crlPEM)
	if err != nil {
		t.Fatalf("Failed to parse CRL: %v", err)
	}

	// Check that NextUpdate is approximately validity hours from now
	expectedNextUpdate := time.Now().Add(validity)
	timeDiff := crl.NextUpdate.Sub(expectedNextUpdate)
	if timeDiff < -time.Minute || timeDiff > time.Minute {
		t.Errorf("NextUpdate %v is not within 1 minute of expected %v", crl.NextUpdate, expectedNextUpdate)
	}

	// ThisUpdate should be close to now
	timeDiff = time.Since(crl.ThisUpdate)
	if timeDiff < 0 || timeDiff > time.Minute {
		t.Errorf("ThisUpdate %v is not within 1 minute of now", crl.ThisUpdate)
	}
}

func TestCRL_DifferentKeyAlgorithms(t *testing.T) {
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

			revokedCerts := []RevokedCertificate{
				{
					SerialNumber: "test123",
					RevokedAt:    time.Now(),
					Reason:       ReasonUnspecified,
				},
			}

			crlPEM, err := ca.GenerateCRL(revokedCerts, 24*time.Hour)
			if err != nil {
				t.Fatalf("Failed to generate CRL with %s: %v", alg, err)
			}

			_, err = ParseCRL(crlPEM)
			if err != nil {
				t.Errorf("Failed to parse CRL with %s: %v", alg, err)
			}
		})
	}
}

// reasonName returns a human-readable name for a CRL reason code.
func reasonName(reason int) string {
	names := map[int]string{
		ReasonUnspecified:          "Unspecified",
		ReasonKeyCompromise:        "KeyCompromise",
		ReasonCACompromise:         "CACompromise",
		ReasonAffiliationChanged:   "AffiliationChanged",
		ReasonSuperseded:           "Superseded",
		ReasonCessationOfOperation: "CessationOfOperation",
		ReasonCertificateHold:      "CertificateHold",
		ReasonRemoveFromCRL:        "RemoveFromCRL",
		ReasonPrivilegeWithdrawn:   "PrivilegeWithdrawn",
		ReasonAACompromise:         "AACompromise",
	}
	if name, ok := names[reason]; ok {
		return name
	}
	return "Unknown"
}
