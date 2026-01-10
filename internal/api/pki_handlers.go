package api

import (
	"net/http"
	"time"

	"github.com/gatekey-project/gatekey/internal/db"
	"github.com/gatekey-project/gatekey/internal/pki"
	"github.com/gin-gonic/gin"
)

// CRL validity duration (how often CRL should be regenerated)
const crlValidity = 24 * time.Hour

// handleGetCRL returns the Certificate Revocation List in DER format.
// GET /pki/crl
func (s *Server) handleGetCRL(c *gin.Context) {
	// Check if we have a valid cached CRL
	store := db.NewCertificateRevocationStore(s.db)
	cached, err := store.GetCRLCache(c.Request.Context(), "default")
	if err == nil && cached != nil {
		// Return cached CRL
		c.Header("Content-Type", "application/pkix-crl")
		c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		c.Data(http.StatusOK, "application/pkix-crl", cached.CRLDER)
		return
	}

	// Generate new CRL
	crlDER, err := s.generateCRL(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/pkix-crl")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/pkix-crl", crlDER)
}

// handleGetCRLPEM returns the Certificate Revocation List in PEM format.
// GET /pki/crl.pem
func (s *Server) handleGetCRLPEM(c *gin.Context) {
	// Get revoked certificates from database
	store := db.NewCertificateRevocationStore(s.db)
	revocations, err := store.ListAllRevocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list revocations"})
		return
	}

	// Convert to pki.RevokedCertificate format
	revokedCerts := make([]pki.RevokedCertificate, len(revocations))
	for i, rev := range revocations {
		revokedCerts[i] = pki.RevokedCertificate{
			SerialNumber: rev.SerialNumber,
			RevokedAt:    rev.RevokedAt,
			Reason:       rev.Reason,
		}
	}

	// Generate CRL
	crlPEM, err := s.ca.GenerateCRL(revokedCerts, crlValidity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate CRL"})
		return
	}

	c.Header("Content-Type", "application/x-pem-file")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/x-pem-file", crlPEM)
}

// generateCRL generates a new CRL and caches it.
func (s *Server) generateCRL(c *gin.Context) ([]byte, error) {
	store := db.NewCertificateRevocationStore(s.db)

	// Get revoked certificates from database
	revocations, err := store.ListAllRevocations(c.Request.Context())
	if err != nil {
		return nil, err
	}

	// Convert to pki.RevokedCertificate format
	revokedCerts := make([]pki.RevokedCertificate, len(revocations))
	for i, rev := range revocations {
		revokedCerts[i] = pki.RevokedCertificate{
			SerialNumber: rev.SerialNumber,
			RevokedAt:    rev.RevokedAt,
			Reason:       rev.Reason,
		}
	}

	// Generate CRL in DER format
	crlDER, err := s.ca.GenerateCRLDER(revokedCerts, crlValidity)
	if err != nil {
		return nil, err
	}

	// Cache the CRL
	now := time.Now()
	cache := &db.CRLCache{
		CAID:       "default",
		CRLDER:     crlDER,
		ThisUpdate: now,
		NextUpdate: now.Add(crlValidity),
		CRLNumber:  now.UnixNano(),
	}
	// Best effort caching - don't fail if cache write fails
	_ = store.SaveCRLCache(c.Request.Context(), cache)

	return crlDER, nil
}

// RevokeCertificateRequest is the request body for revoking a certificate.
type RevokeCertificateRequest struct {
	SerialNumber string `json:"serial_number" binding:"required"`
	CommonName   string `json:"common_name"`
	Reason       int    `json:"reason"` // RFC 5280 reason code (default 0 = unspecified)
	Notes        string `json:"notes"`
}

// handleRevokeCertificate revokes a certificate.
// POST /admin/pki/revoke
func (s *Server) handleRevokeCertificate(c *gin.Context) {
	var req RevokeCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the user who is revoking
	revokedBy := ""
	if userEmail, exists := c.Get("user_email"); exists {
		if email, ok := userEmail.(string); ok {
			revokedBy = email
		}
	}

	store := db.NewCertificateRevocationStore(s.db)

	// Add to revocation list
	rev := &db.CertificateRevocation{
		SerialNumber: req.SerialNumber,
		CAID:         "default",
		CommonName:   req.CommonName,
		RevokedAt:    time.Now(),
		Reason:       req.Reason,
		RevokedBy:    revokedBy,
		Notes:        req.Notes,
	}

	if err := store.RevokeCertificate(c.Request.Context(), rev); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke certificate"})
		return
	}

	// Invalidate CRL cache so it gets regenerated
	_ = store.InvalidateCRLCache(c.Request.Context(), "default")

	c.JSON(http.StatusOK, gin.H{
		"message":       "certificate revoked",
		"serial_number": req.SerialNumber,
	})
}

// handleListRevocations lists all revoked certificates.
// GET /admin/pki/revocations
func (s *Server) handleListRevocations(c *gin.Context) {
	store := db.NewCertificateRevocationStore(s.db)
	revocations, err := store.ListAllRevocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list revocations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"revocations": revocations,
		"count":       len(revocations),
	})
}

// handleUnrevokeCertificate removes a certificate from the revocation list.
// DELETE /admin/pki/revocations/:serial
func (s *Server) handleUnrevokeCertificate(c *gin.Context) {
	serialNumber := c.Param("serial")
	if serialNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serial number required"})
		return
	}

	store := db.NewCertificateRevocationStore(s.db)
	if err := store.DeleteRevocation(c.Request.Context(), serialNumber); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unrevoke certificate"})
		return
	}

	// Invalidate CRL cache
	_ = store.InvalidateCRLCache(c.Request.Context(), "default")

	c.JSON(http.StatusOK, gin.H{
		"message":       "certificate unrevoked",
		"serial_number": serialNumber,
	})
}

// handleCheckRevocation checks if a certificate is revoked.
// GET /pki/check/:serial
func (s *Server) handleCheckRevocation(c *gin.Context) {
	serialNumber := c.Param("serial")
	if serialNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serial number required"})
		return
	}

	store := db.NewCertificateRevocationStore(s.db)
	revoked, err := store.IsCertificateRevoked(c.Request.Context(), serialNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check revocation status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"serial_number": serialNumber,
		"revoked":       revoked,
	})
}
