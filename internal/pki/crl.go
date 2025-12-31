// Package pki provides CRL (Certificate Revocation List) operations for GateKey.
package pki

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// RevokedCertificate represents a revoked certificate entry.
type RevokedCertificate struct {
	SerialNumber string
	RevokedAt    time.Time
	Reason       int // CRL reason code
}

// CRL revocation reason codes (RFC 5280)
const (
	ReasonUnspecified          = 0
	ReasonKeyCompromise        = 1
	ReasonCACompromise         = 2
	ReasonAffiliationChanged   = 3
	ReasonSuperseded           = 4
	ReasonCessationOfOperation = 5
	ReasonCertificateHold      = 6
	ReasonRemoveFromCRL        = 8
	ReasonPrivilegeWithdrawn   = 9
	ReasonAACompromise         = 10
)

// GenerateCRL generates a Certificate Revocation List.
func (ca *CA) GenerateCRL(revokedCerts []RevokedCertificate, validity time.Duration) ([]byte, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	now := time.Now()
	nextUpdate := now.Add(validity)

	// Convert to pkix format
	revokedList := make([]pkix.RevokedCertificate, len(revokedCerts))
	for i, cert := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(cert.SerialNumber, 16)

		revokedList[i] = pkix.RevokedCertificate{
			SerialNumber:   serial,
			RevocationTime: cert.RevokedAt,
			Extensions: []pkix.Extension{
				{
					Id:       []int{2, 5, 29, 21}, // CRL Reason Code OID
					Critical: false,
					Value:    []byte{0x0a, 0x01, byte(cert.Reason)}, // ENUMERATED reason
				},
			},
		}
	}

	crlTemplate := &x509.RevocationList{
		RevokedCertificateEntries: make([]x509.RevocationListEntry, len(revokedCerts)),
		Number:                    big.NewInt(time.Now().UnixNano()),
		ThisUpdate:                now,
		NextUpdate:                nextUpdate,
	}

	for i, cert := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(cert.SerialNumber, 16)

		crlTemplate.RevokedCertificateEntries[i] = x509.RevocationListEntry{
			SerialNumber:   serial,
			RevocationTime: cert.RevokedAt,
			ReasonCode:     cert.Reason,
		}
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, crlTemplate, ca.certificate, ca.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRL: %w", err)
	}

	// Encode to PEM
	crlPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crlDER,
	})

	return crlPEM, nil
}

// GenerateCRLDER generates a CRL in DER format.
func (ca *CA) GenerateCRLDER(revokedCerts []RevokedCertificate, validity time.Duration) ([]byte, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	now := time.Now()
	nextUpdate := now.Add(validity)

	crlTemplate := &x509.RevocationList{
		RevokedCertificateEntries: make([]x509.RevocationListEntry, len(revokedCerts)),
		Number:                    big.NewInt(time.Now().UnixNano()),
		ThisUpdate:                now,
		NextUpdate:                nextUpdate,
	}

	for i, cert := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(cert.SerialNumber, 16)

		crlTemplate.RevokedCertificateEntries[i] = x509.RevocationListEntry{
			SerialNumber:   serial,
			RevocationTime: cert.RevokedAt,
			ReasonCode:     cert.Reason,
		}
	}

	return x509.CreateRevocationList(rand.Reader, crlTemplate, ca.certificate, ca.privateKey)
}

// ParseCRL parses a PEM-encoded CRL.
func ParseCRL(pemData []byte) (*x509.RevocationList, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	if block.Type != "X509 CRL" {
		return nil, fmt.Errorf("PEM block is not a CRL")
	}
	return x509.ParseRevocationList(block.Bytes)
}

// IsCertificateRevoked checks if a certificate is in the CRL.
func IsCertificateRevoked(crl *x509.RevocationList, cert *x509.Certificate) bool {
	for _, entry := range crl.RevokedCertificateEntries {
		if entry.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			return true
		}
	}
	return false
}
