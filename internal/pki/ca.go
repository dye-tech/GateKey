// Package pki provides certificate authority and PKI operations for GateKey.
package pki

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/gatekey-project/gatekey/internal/config"
)

// CAStore defines the interface for persisting CA data.
type CAStore interface {
	GetCA(ctx context.Context) (*StoredCA, error)
	SaveCA(ctx context.Context, ca *StoredCA) error
}

// StoredCA represents a CA stored in a persistence layer.
type StoredCA struct {
	CertificatePEM string
	PrivateKeyPEM  string
	SerialNumber   string
	NotBefore      time.Time
	NotAfter       time.Time
}

// CA represents the Certificate Authority.
type CA struct {
	config      config.PKIConfig
	store       CAStore
	certificate *x509.Certificate
	privateKey  crypto.Signer
	mu          sync.RWMutex
	serialMu    sync.Mutex
	lastSerial  *big.Int
}

// NewCA creates a new Certificate Authority.
// If CA cert/key files are specified and exist, they are loaded.
// Otherwise, a new self-signed CA is generated.
func NewCA(cfg config.PKIConfig) (*CA, error) {
	ca := &CA{
		config:     cfg,
		lastSerial: big.NewInt(0),
	}

	// Try to load existing CA
	if cfg.CACert != "" && cfg.CAKey != "" {
		if err := ca.loadFromFiles(cfg.CACert, cfg.CAKey); err == nil {
			return ca, nil
		}
		// If loading fails, generate new CA
	}

	// Generate new CA
	if err := ca.generateSelfSigned(); err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	// Save to files if paths are specified
	if cfg.CACert != "" && cfg.CAKey != "" {
		if err := ca.saveToFiles(cfg.CACert, cfg.CAKey); err != nil {
			return nil, fmt.Errorf("failed to save CA: %w", err)
		}
	}

	return ca, nil
}

// NewCAWithStore creates a new Certificate Authority using a database store.
// This ensures all pods share the same CA.
func NewCAWithStore(cfg config.PKIConfig, store CAStore) (*CA, error) {
	ca := &CA{
		config:     cfg,
		store:      store,
		lastSerial: big.NewInt(0),
	}

	ctx := context.Background()

	// Try to load from database first
	storedCA, err := store.GetCA(ctx)
	if err == nil {
		// Load from stored data
		if err := ca.loadFromPEM(storedCA.CertificatePEM, storedCA.PrivateKeyPEM); err != nil {
			return nil, fmt.Errorf("failed to load CA from database: %w", err)
		}
		return ca, nil
	}

	// No CA in database, try loading from files as fallback
	if cfg.CACert != "" && cfg.CAKey != "" {
		if err := ca.loadFromFiles(cfg.CACert, cfg.CAKey); err == nil {
			// Save loaded CA to database for other pods
			if err := ca.saveToStore(ctx); err != nil {
				return nil, fmt.Errorf("failed to save CA to database: %w", err)
			}
			return ca, nil
		}
	}

	// Generate new CA
	if err := ca.generateSelfSigned(); err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	// Save to database
	if err := ca.saveToStore(ctx); err != nil {
		return nil, fmt.Errorf("failed to save CA to database: %w", err)
	}

	// Also save to files if paths are specified
	if cfg.CACert != "" && cfg.CAKey != "" {
		if err := ca.saveToFiles(cfg.CACert, cfg.CAKey); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to save CA to files: %v\n", err)
		}
	}

	return ca, nil
}

// loadFromPEM loads the CA from PEM strings.
func (ca *CA) loadFromPEM(certPEM, keyPEM string) error {
	// Parse certificate
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse private key
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	var key crypto.Signer
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		parsedKey, parseErr := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if parseErr != nil {
			return fmt.Errorf("failed to parse PKCS8 private key: %w", parseErr)
		}
		var ok bool
		key, ok = parsedKey.(crypto.Signer)
		if !ok {
			return fmt.Errorf("private key is not a signer")
		}
		err = nil
	default:
		return fmt.Errorf("unsupported private key type: %s", keyBlock.Type)
	}
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	ca.certificate = cert
	ca.privateKey = key
	return nil
}

// saveToStore saves the CA to the database store.
func (ca *CA) saveToStore(ctx context.Context) error {
	if ca.store == nil {
		return nil
	}

	certPEM := string(ca.CertificatePEM())
	keyPEM := string(ca.PrivateKeyPEM())

	storedCA := &StoredCA{
		CertificatePEM: certPEM,
		PrivateKeyPEM:  keyPEM,
		SerialNumber:   ca.certificate.SerialNumber.String(),
		NotBefore:      ca.certificate.NotBefore,
		NotAfter:       ca.certificate.NotAfter,
	}

	return ca.store.SaveCA(ctx, storedCA)
}

// PrivateKeyPEM returns the CA private key in PEM format.
func (ca *CA) PrivateKeyPEM() []byte {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var keyBytes []byte
	var keyType string
	switch k := ca.privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(k)
		keyType = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		keyBytes, _ = x509.MarshalECPrivateKey(k)
		keyType = "EC PRIVATE KEY"
	default:
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	})
}

// loadFromFiles loads the CA certificate and private key from files.
func (ca *CA) loadFromFiles(certPath, keyPath string) error {
	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	var key crypto.Signer
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		parsedKey, parseErr := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if parseErr != nil {
			return fmt.Errorf("failed to parse PKCS8 private key: %w", parseErr)
		}
		var ok bool
		key, ok = parsedKey.(crypto.Signer)
		if !ok {
			return fmt.Errorf("private key is not a signer")
		}
		err = nil
	default:
		return fmt.Errorf("unsupported private key type: %s", keyBlock.Type)
	}
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	ca.certificate = cert
	ca.privateKey = key
	return nil
}

// saveToFiles saves the CA certificate and private key to files.
func (ca *CA) saveToFiles(certPath, keyPath string) error {
	// Save certificate (needs to be readable by clients that verify certs)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.certificate.Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Save private key
	var keyBytes []byte
	var keyType string
	switch k := ca.privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(k)
		keyType = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		var err error
		keyBytes, err = x509.MarshalECPrivateKey(k)
		if err != nil {
			return fmt.Errorf("failed to marshal EC private key: %w", err)
		}
		keyType = "EC PRIVATE KEY"
	default:
		return fmt.Errorf("unsupported private key type")
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write CA private key: %w", err)
	}

	return nil
}

// generateSelfSigned generates a new self-signed CA certificate.
func (ca *CA) generateSelfSigned() error {
	// Generate private key
	key, err := generatePrivateKey(ca.config.KeyAlgorithm)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serial, err := generateSerialNumber()
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{ca.config.Organization},
			CommonName:   ca.config.Organization + " Root CA",
		},
		NotBefore:             now,
		NotAfter:              now.Add(ca.config.CAValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey(key), key)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	ca.certificate = cert
	ca.privateKey = key
	return nil
}

// Certificate returns the CA certificate.
func (ca *CA) Certificate() *x509.Certificate {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.certificate
}

// CertificatePEM returns the CA certificate in PEM format.
func (ca *CA) CertificatePEM() []byte {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.certificate.Raw,
	})
}

// Rotate generates a new CA certificate and saves it to the store.
func (ca *CA) Rotate(ctx context.Context) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Generate new CA
	if err := ca.generateSelfSigned(); err != nil {
		return fmt.Errorf("failed to generate new CA: %w", err)
	}

	// Save to store if configured
	if ca.store != nil {
		if err := ca.saveToStore(ctx); err != nil {
			return fmt.Errorf("failed to save CA to store: %w", err)
		}
	}

	// Save to files if configured
	if ca.config.CACert != "" && ca.config.CAKey != "" {
		if err := ca.saveToFiles(ca.config.CACert, ca.config.CAKey); err != nil {
			return fmt.Errorf("failed to save CA to files: %w", err)
		}
	}

	return nil
}

// UpdateFromPEM updates the CA with a custom certificate and private key.
func (ca *CA) UpdateFromPEM(ctx context.Context, certPEM, keyPEM string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Parse and validate the certificate and key
	if err := ca.loadFromPEM(certPEM, keyPEM); err != nil {
		return fmt.Errorf("invalid certificate/key: %w", err)
	}

	// Verify it's a CA certificate
	if !ca.certificate.IsCA {
		return fmt.Errorf("certificate is not a CA certificate")
	}

	// Save to store if configured
	if ca.store != nil {
		if err := ca.saveToStore(ctx); err != nil {
			return fmt.Errorf("failed to save CA to store: %w", err)
		}
	}

	// Save to files if configured
	if ca.config.CACert != "" && ca.config.CAKey != "" {
		if err := ca.saveToFiles(ca.config.CACert, ca.config.CAKey); err != nil {
			return fmt.Errorf("failed to save CA to files: %w", err)
		}
	}

	return nil
}

// nextSerial generates the next serial number.
func (ca *CA) nextSerial() (*big.Int, error) {
	ca.serialMu.Lock()
	defer ca.serialMu.Unlock()

	// Generate a random serial number
	serial, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}

	ca.lastSerial = serial
	return serial, nil
}

// generatePrivateKey generates a private key based on the algorithm.
func generatePrivateKey(algorithm string) (crypto.Signer, error) {
	switch algorithm {
	case "rsa2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	case "ecdsa256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ecdsa384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	default:
		return nil, fmt.Errorf("unsupported key algorithm: %s", algorithm)
	}
}

// GenerateECDSAKey generates an ECDSA P-256 private key
func GenerateECDSAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// generateSerialNumber generates a random serial number.
func generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

// publicKey extracts the public key from a private key.
func publicKey(priv crypto.Signer) crypto.PublicKey {
	return priv.Public()
}

// Fingerprint calculates the SHA256 fingerprint of a certificate.
func Fingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}
