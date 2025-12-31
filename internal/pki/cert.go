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

// GenerateSubCA creates an intermediate CA certificate signed by this CA.
// Returns the CA certificate PEM and private key PEM.
func (ca *CA) GenerateSubCA(commonName string) (string, string, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	// Generate private key for sub-CA
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate sub-CA private key: %w", err)
	}

	// Generate serial number
	serial, err := ca.nextSerial()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	validity := ca.config.CAValidity
	if validity < 10*365*24*time.Hour {
		validity = 10 * 365 * 24 * time.Hour // Default 10 years for sub-CAs
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{ca.config.Organization},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Sign with parent CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, publicKey(key), ca.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create sub-CA certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM, err := encodePrivateKeyPEM(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	return string(certPEM), string(keyPEM), nil
}

// GenerateServerCert generates a server certificate and returns PEM strings.
// This is a convenience wrapper around IssueServerCertificate.
func (ca *CA) GenerateServerCert(commonName string, dnsNames []string) (string, string, error) {
	issued, err := ca.IssueServerCertificate(CertificateRequest{
		CommonName: commonName,
		DNSNames:   dnsNames,
		ValidFor:   365 * 24 * time.Hour, // 1 year
	})
	if err != nil {
		return "", "", err
	}
	return string(issued.CertificatePEM), string(issued.PrivateKeyPEM), nil
}

// GenerateClientCertWithCA issues a client certificate using a provided CA certificate and key.
// The CA cert and key are provided as PEM strings.
func (ca *CA) GenerateClientCertWithCA(caCertPEM, caKeyPEM, commonName string, dnsNames []string) (string, string, error) {
	// Parse the CA certificate
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse the CA private key
	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA private key PEM")
	}
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		// Try parsing as PKCS1 (RSA)
		caKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
		if err != nil {
			// Try parsing as EC key
			caKey, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
			if err != nil {
				return "", "", fmt.Errorf("failed to parse CA private key: %w", err)
			}
		}
	}

	caKeySigner, ok := caKey.(crypto.Signer)
	if !ok {
		return "", "", fmt.Errorf("CA key is not a valid signer")
	}

	// Generate private key for client
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client private key: %w", err)
	}

	// Generate serial number
	serial, err := ca.nextSerial()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	validity := 365 * 24 * time.Hour // 1 year

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{ca.config.Organization},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	if len(dnsNames) > 0 {
		template.DNSNames = dnsNames
	}

	// Sign with provided CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, publicKey(key), caKeySigner)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM, err := encodePrivateKeyPEM(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	return string(certPEM), string(keyPEM), nil
}

// GenerateServerCertWithCA issues a server certificate using a provided CA certificate and key.
// The CA cert and key are provided as PEM strings.
func (ca *CA) GenerateServerCertWithCA(caCertPEM, caKeyPEM, commonName string, dnsNames []string) (string, string, error) {
	// Parse the CA certificate
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse the CA private key
	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA private key PEM")
	}
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		// Try parsing as PKCS1 (RSA)
		caKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
		if err != nil {
			// Try parsing as EC key
			caKey, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
			if err != nil {
				return "", "", fmt.Errorf("failed to parse CA private key: %w", err)
			}
		}
	}

	caKeySigner, ok := caKey.(crypto.Signer)
	if !ok {
		return "", "", fmt.Errorf("CA key is not a valid signer")
	}

	// Generate private key for server
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Generate serial number
	serial, err := ca.nextSerial()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	validity := 365 * 24 * time.Hour // 1 year

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{ca.config.Organization},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	if len(dnsNames) > 0 {
		template.DNSNames = dnsNames
	}

	// Sign with provided CA
	serverCertDER, err := x509.CreateCertificate(rand.Reader, template, caCert, publicKey(key), caKeySigner)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	serverCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertDER,
	})

	// Encode private key to PEM
	serverKeyPEM, err := encodePrivateKeyPEM(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	return string(serverCertPEM), string(serverKeyPEM), nil
}
