// Package pki provides certificate generation for GateKey.
package pki

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"time"
)

// CertificateRequest contains parameters for generating a client certificate.
type CertificateRequest struct {
	CommonName   string
	Email        string
	Organization string
	ValidFor     time.Duration
	DNSNames     []string
	IPAddresses  []string
}

// IssuedCertificate contains the issued certificate and private key.
type IssuedCertificate struct {
	Certificate    *x509.Certificate
	PrivateKey     crypto.Signer
	CertificatePEM []byte
	PrivateKeyPEM  []byte
	SerialNumber   string
	Fingerprint    string
	NotBefore      time.Time
	NotAfter       time.Time
}

// IssueClientCertificate issues a new client certificate.
func (ca *CA) IssueClientCertificate(req CertificateRequest) (*IssuedCertificate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	// Generate private key for client
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client private key: %w", err)
	}

	// Generate serial number
	serial, err := ca.nextSerial()
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	validity := req.ValidFor
	if validity == 0 {
		validity = ca.config.CertValidity
	}

	org := req.Organization
	if org == "" {
		org = ca.config.Organization
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{org},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Add email to SAN
	if req.Email != "" {
		template.EmailAddresses = []string{req.Email}
	}

	// Add DNS names
	if len(req.DNSNames) > 0 {
		template.DNSNames = req.DNSNames
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, publicKey(key), ca.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM, err := encodePrivateKeyPEM(key)
	if err != nil {
		return nil, fmt.Errorf("failed to encode private key: %w", err)
	}

	return &IssuedCertificate{
		Certificate:    cert,
		PrivateKey:     key,
		CertificatePEM: certPEM,
		PrivateKeyPEM:  keyPEM,
		SerialNumber:   serial.Text(16),
		Fingerprint:    Fingerprint(cert),
		NotBefore:      now,
		NotAfter:       now.Add(validity),
	}, nil
}

// IssueServerCertificate issues a new server certificate.
func (ca *CA) IssueServerCertificate(req CertificateRequest) (*IssuedCertificate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	// Generate private key for server
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Generate serial number
	serial, err := ca.nextSerial()
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	validity := req.ValidFor
	if validity == 0 {
		validity = ca.config.CertValidity
	}

	org := req.Organization
	if org == "" {
		org = ca.config.Organization
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{org},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Add DNS names
	if len(req.DNSNames) > 0 {
		template.DNSNames = req.DNSNames
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, publicKey(key), ca.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM, err := encodePrivateKeyPEM(key)
	if err != nil {
		return nil, fmt.Errorf("failed to encode private key: %w", err)
	}

	return &IssuedCertificate{
		Certificate:    cert,
		PrivateKey:     key,
		CertificatePEM: certPEM,
		PrivateKeyPEM:  keyPEM,
		SerialNumber:   serial.Text(16),
		Fingerprint:    Fingerprint(cert),
		NotBefore:      now,
		NotAfter:       now.Add(validity),
	}, nil
}

// VerifyCertificate verifies a certificate against the CA.
func (ca *CA) VerifyCertificate(cert *x509.Certificate) error {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	roots := x509.NewCertPool()
	roots.AddCert(ca.certificate)

	opts := x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	_, err := cert.Verify(opts)
	return err
}

// ParseCertificatePEM parses a PEM-encoded certificate.
func ParseCertificatePEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("PEM block is not a certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

// encodePrivateKeyPEM encodes a private key to PEM format.
func encodePrivateKeyPEM(key crypto.Signer) ([]byte, error) {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}), nil
}
