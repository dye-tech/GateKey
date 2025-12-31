package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// OIDC Provider HTTP Handlers

func (s *Server) handleGetOIDCProvidersDynamic(c *gin.Context) {
	providers, err := s.providerStore.GetOIDCProviders(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to get OIDC providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"enabled":   len(providers) > 0,
	})
}

func (s *Server) handleCreateOIDCProviderDynamic(c *gin.Context) {
	var provider db.OIDCProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if provider.Name == "" || provider.Issuer == "" || provider.ClientID == "" || provider.ClientSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, issuer, client_id, and client_secret are required"})
		return
	}

	if err := s.providerStore.CreateOIDCProvider(c.Request.Context(), &provider); err != nil {
		if err == db.ErrProviderExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to create OIDC provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "provider created", "name": provider.Name})
}

func (s *Server) handleUpdateOIDCProviderDynamic(c *gin.Context) {
	name := c.Param("name")

	var provider db.OIDCProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := s.providerStore.UpdateOIDCProvider(c.Request.Context(), name, &provider); err != nil {
		if err == db.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to update OIDC provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider updated", "name": name})
}

func (s *Server) handleDeleteOIDCProviderDynamic(c *gin.Context) {
	name := c.Param("name")

	if err := s.providerStore.DeleteOIDCProvider(c.Request.Context(), name); err != nil {
		if err == db.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to delete OIDC provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider deleted", "name": name})
}

// SAML Provider HTTP Handlers

func (s *Server) handleGetSAMLProvidersDynamic(c *gin.Context) {
	providers, err := s.providerStore.GetSAMLProviders(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to get SAML providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"enabled":   len(providers) > 0,
	})
}

func (s *Server) handleCreateSAMLProviderDynamic(c *gin.Context) {
	var provider db.SAMLProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if provider.Name == "" || provider.IDPMetadataURL == "" || provider.EntityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, idp_metadata_url, and entity_id are required"})
		return
	}

	if err := s.providerStore.CreateSAMLProvider(c.Request.Context(), &provider); err != nil {
		if err == db.ErrProviderExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to create SAML provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "provider created", "name": provider.Name})
}

func (s *Server) handleUpdateSAMLProviderDynamic(c *gin.Context) {
	name := c.Param("name")

	var provider db.SAMLProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := s.providerStore.UpdateSAMLProvider(c.Request.Context(), name, &provider); err != nil {
		if err == db.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to update SAML provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider updated", "name": name})
}

func (s *Server) handleDeleteSAMLProviderDynamic(c *gin.Context) {
	name := c.Param("name")

	if err := s.providerStore.DeleteSAMLProvider(c.Request.Context(), name); err != nil {
		if err == db.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error("Failed to delete SAML provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider deleted", "name": name})
}
