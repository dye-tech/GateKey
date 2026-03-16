package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// DNS Rule Handlers

// handleGetDNSRule returns the DNS rule for a network
func (s *Server) handleGetDNSRule(c *gin.Context) {
	networkID := c.Param("id")
	ctx := c.Request.Context()

	rule, err := s.dnsStore.GetRuleByNetwork(ctx, networkID)
	if err != nil {
		if err == db.ErrDNSRuleNotFound {
			c.JSON(http.StatusOK, gin.H{"rule": nil})
			return
		}
		s.logger.Error("Failed to get DNS rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get DNS rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rule": gin.H{
			"id":             rule.ID,
			"network_id":     rule.NetworkID,
			"dns_servers":    rule.DNSServers,
			"search_domains": rule.SearchDomains,
			"is_active":      rule.IsActive,
			"created_at":     rule.CreatedAt,
			"updated_at":     rule.UpdatedAt,
		},
	})
}

// handleUpsertDNSRule creates or updates the DNS rule for a network
func (s *Server) handleUpsertDNSRule(c *gin.Context) {
	networkID := c.Param("id")
	ctx := c.Request.Context()

	var req struct {
		DNSServers    []string `json:"dns_servers"`
		SearchDomains []string `json:"search_domains"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify network exists
	if _, err := s.networkStore.GetNetwork(ctx, networkID); err != nil {
		if err == db.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
			return
		}
		s.logger.Error("Failed to get network", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify network"})
		return
	}

	if req.DNSServers == nil {
		req.DNSServers = []string{}
	}
	if req.SearchDomains == nil {
		req.SearchDomains = []string{}
	}

	rule := &db.DNSRule{
		NetworkID:     networkID,
		DNSServers:    req.DNSServers,
		SearchDomains: req.SearchDomains,
		IsActive:      true,
	}

	if err := s.dnsStore.UpsertRule(ctx, rule); err != nil {
		s.logger.Error("Failed to upsert DNS rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save DNS rule"})
		return
	}

	s.logger.Info("DNS rule upserted",
		zap.String("network_id", networkID),
		zap.Strings("dns_servers", req.DNSServers),
		zap.Strings("search_domains", req.SearchDomains))

	c.JSON(http.StatusOK, gin.H{
		"rule": gin.H{
			"id":             rule.ID,
			"network_id":     rule.NetworkID,
			"dns_servers":    rule.DNSServers,
			"search_domains": rule.SearchDomains,
			"is_active":      rule.IsActive,
			"created_at":     rule.CreatedAt,
			"updated_at":     rule.UpdatedAt,
		},
	})
}

// handleDeleteDNSRule deletes the DNS rule for a network
func (s *Server) handleDeleteDNSRule(c *gin.Context) {
	networkID := c.Param("id")
	ctx := c.Request.Context()

	if err := s.dnsStore.DeleteRule(ctx, networkID); err != nil {
		if err == db.ErrDNSRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "DNS rule not found"})
			return
		}
		s.logger.Error("Failed to delete DNS rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete DNS rule"})
		return
	}

	s.logger.Info("DNS rule deleted", zap.String("network_id", networkID))
	c.JSON(http.StatusOK, gin.H{"message": "DNS rule deleted"})
}

// DNS Record Handlers

// handleListDNSRecords lists DNS records for a network
func (s *Server) handleListDNSRecords(c *gin.Context) {
	networkID := c.Param("id")
	ctx := c.Request.Context()

	records, err := s.dnsStore.ListRecords(ctx, networkID)
	if err != nil {
		s.logger.Error("Failed to list DNS records", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list DNS records"})
		return
	}

	result := make([]gin.H, 0, len(records))
	for _, r := range records {
		result = append(result, gin.H{
			"id":          r.ID,
			"network_id":  r.NetworkID,
			"hostname":    r.Hostname,
			"ip_address":  r.IPAddress,
			"description": r.Description,
			"is_active":   r.IsActive,
			"created_at":  r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"records": result})
}

// handleCreateDNSRecord creates a new DNS record for a network
func (s *Server) handleCreateDNSRecord(c *gin.Context) {
	networkID := c.Param("id")
	ctx := c.Request.Context()

	var req struct {
		Hostname    string `json:"hostname" binding:"required"`
		IPAddress   string `json:"ip_address" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify network exists
	if _, err := s.networkStore.GetNetwork(ctx, networkID); err != nil {
		if err == db.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
			return
		}
		s.logger.Error("Failed to get network", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify network"})
		return
	}

	record := &db.DNSRecord{
		NetworkID:   networkID,
		Hostname:    req.Hostname,
		IPAddress:   req.IPAddress,
		Description: req.Description,
		IsActive:    true,
	}

	if err := s.dnsStore.CreateRecord(ctx, record); err != nil {
		s.logger.Error("Failed to create DNS record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DNS record"})
		return
	}

	s.logger.Info("DNS record created",
		zap.String("network_id", networkID),
		zap.String("hostname", req.Hostname),
		zap.String("ip_address", req.IPAddress))

	c.JSON(http.StatusCreated, gin.H{
		"record": gin.H{
			"id":          record.ID,
			"network_id":  record.NetworkID,
			"hostname":    record.Hostname,
			"ip_address":  record.IPAddress,
			"description": record.Description,
			"is_active":   record.IsActive,
			"created_at":  record.CreatedAt,
		},
	})
}

// handleUpdateDNSRecord updates a DNS record
func (s *Server) handleUpdateDNSRecord(c *gin.Context) {
	recordID := c.Param("id")
	ctx := c.Request.Context()

	var req struct {
		Hostname    *string `json:"hostname"`
		IPAddress   *string `json:"ip_address"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Hostname == nil || req.IPAddress == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostname and ip_address are required"})
		return
	}

	record := &db.DNSRecord{
		ID:        recordID,
		Hostname:  *req.Hostname,
		IPAddress: *req.IPAddress,
	}
	if req.Description != nil {
		record.Description = *req.Description
	}
	record.IsActive = true
	if req.IsActive != nil {
		record.IsActive = *req.IsActive
	}

	if err := s.dnsStore.UpdateRecord(ctx, record); err != nil {
		if err == db.ErrDNSRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "DNS record not found"})
			return
		}
		s.logger.Error("Failed to update DNS record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update DNS record"})
		return
	}

	s.logger.Info("DNS record updated", zap.String("record_id", recordID))
	c.JSON(http.StatusOK, gin.H{"message": "DNS record updated"})
}

// handleDeleteDNSRecord deletes a DNS record
func (s *Server) handleDeleteDNSRecord(c *gin.Context) {
	recordID := c.Param("id")
	ctx := c.Request.Context()

	if err := s.dnsStore.DeleteRecord(ctx, recordID); err != nil {
		if err == db.ErrDNSRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "DNS record not found"})
			return
		}
		s.logger.Error("Failed to delete DNS record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete DNS record"})
		return
	}

	s.logger.Info("DNS record deleted", zap.String("record_id", recordID))
	c.JSON(http.StatusOK, gin.H{"message": "DNS record deleted"})
}

// User DNS Config Handler

// handleGetUserDNSConfig returns the aggregated DNS config for the authenticated user
func (s *Server) handleGetUserDNSConfig(c *gin.Context) {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Get user's access rules
	rules, err := s.accessRuleStore.GetUserAccessRules(ctx, user.UserID, user.Groups)
	if err != nil {
		s.logger.Error("Failed to get user access rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access rules"})
		return
	}

	// Collect unique network IDs from access rules
	networkIDSet := make(map[string]bool)
	for _, rule := range rules {
		if rule.NetworkID != nil && *rule.NetworkID != "" {
			networkIDSet[*rule.NetworkID] = true
		}
	}

	networkIDs := make([]string, 0, len(networkIDSet))
	for id := range networkIDSet {
		networkIDs = append(networkIDs, id)
	}

	// Get DNS rules for those networks
	dnsRules, err := s.dnsStore.GetRulesForNetworks(ctx, networkIDs)
	if err != nil {
		s.logger.Error("Failed to get DNS rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get DNS rules"})
		return
	}

	// Get DNS records for those networks
	dnsRecords, err := s.dnsStore.GetRecordsForNetworks(ctx, networkIDs)
	if err != nil {
		s.logger.Error("Failed to get DNS records", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get DNS records"})
		return
	}

	// Aggregate DNS servers and search domains (deduplicated)
	serverSet := make(map[string]bool)
	domainSet := make(map[string]bool)
	var dnsServers []string
	var searchDomains []string

	for _, rule := range dnsRules {
		for _, srv := range rule.DNSServers {
			if !serverSet[srv] {
				serverSet[srv] = true
				dnsServers = append(dnsServers, srv)
			}
		}
		for _, dom := range rule.SearchDomains {
			if !domainSet[dom] {
				domainSet[dom] = true
				searchDomains = append(searchDomains, dom)
			}
		}
	}

	// Build records response
	records := make([]gin.H, 0, len(dnsRecords))
	for _, r := range dnsRecords {
		records = append(records, gin.H{
			"id":          r.ID,
			"network_id":  r.NetworkID,
			"hostname":    r.Hostname,
			"ip_address":  r.IPAddress,
			"description": r.Description,
			"is_active":   r.IsActive,
			"created_at":  r.CreatedAt,
		})
	}

	if dnsServers == nil {
		dnsServers = []string{}
	}
	if searchDomains == nil {
		searchDomains = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"dns_servers":    dnsServers,
		"search_domains": searchDomains,
		"records":        records,
	})
}
