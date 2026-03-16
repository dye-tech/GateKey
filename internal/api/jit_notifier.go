package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// sendJITRequestNotification sends a webhook notification when a new JIT request is created
func (s *Server) sendJITRequestNotification(req *db.JITAccessRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	enabled := s.settingsStore.GetString(ctx, db.SettingJITWebhookEnabled, "false")
	if enabled != "true" {
		return
	}

	webhookURL := s.settingsStore.GetString(ctx, db.SettingJITWebhookURL, "")
	if webhookURL == "" {
		return
	}

	appURL := s.settingsStore.GetString(ctx, db.SettingJITAppURL, "")
	approvalLink := ""
	if appURL != "" {
		approvalLink = fmt.Sprintf("%s/admin/jit-access", appURL)
	}

	// Build Slack-compatible webhook payload
	text := fmt.Sprintf("*New JIT Access Request*\n*Requester:* %s\n*Resource:* %s (%s)\n*Duration:* %d minutes\n*Justification:* %s",
		req.RequesterEmail,
		req.ResourceName,
		req.ResourceType,
		req.RequestedDurationMin,
		req.Justification,
	)

	if approvalLink != "" {
		text += fmt.Sprintf("\n<%s|Review in GateKey>", approvalLink)
	}

	payload := map[string]interface{}{
		"text": text,
	}

	s.sendWebhook(webhookURL, payload)
}

// sendJITApprovalNotification sends a webhook notification when a JIT request is approved
func (s *Server) sendJITApprovalNotification(req *db.JITAccessRequest, grant *db.JITAccessGrant, approverEmail string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	enabled := s.settingsStore.GetString(ctx, db.SettingJITWebhookEnabled, "false")
	if enabled != "true" {
		return
	}

	webhookURL := s.settingsStore.GetString(ctx, db.SettingJITWebhookURL, "")
	if webhookURL == "" {
		return
	}

	text := fmt.Sprintf("*JIT Access Approved*\n*Requester:* %s\n*Resource:* %s (%s)\n*Approved by:* %s\n*Expires:* %s",
		req.RequesterEmail,
		req.ResourceName,
		req.ResourceType,
		approverEmail,
		grant.ExpiresAt.Format(time.RFC3339),
	)

	payload := map[string]interface{}{
		"text": text,
	}

	s.sendWebhook(webhookURL, payload)
}

// sendJITDenialNotification sends a webhook notification when a JIT request is denied
func (s *Server) sendJITDenialNotification(req *db.JITAccessRequest, approverEmail, note string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	enabled := s.settingsStore.GetString(ctx, db.SettingJITWebhookEnabled, "false")
	if enabled != "true" {
		return
	}

	webhookURL := s.settingsStore.GetString(ctx, db.SettingJITWebhookURL, "")
	if webhookURL == "" {
		return
	}

	text := fmt.Sprintf("*JIT Access Denied*\n*Requester:* %s\n*Resource:* %s (%s)\n*Denied by:* %s",
		req.RequesterEmail,
		req.ResourceName,
		req.ResourceType,
		approverEmail,
	)
	if note != "" {
		text += fmt.Sprintf("\n*Reason:* %s", note)
	}

	payload := map[string]interface{}{
		"text": text,
	}

	s.sendWebhook(webhookURL, payload)
}

// sendWebhook sends a JSON payload to a webhook URL
func (s *Server) sendWebhook(url string, payload map[string]interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal webhook payload", zap.Error(err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		s.logger.Error("Failed to create webhook request", zap.Error(err))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		s.logger.Warn("Failed to send JIT webhook notification", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s.logger.Warn("JIT webhook returned error", zap.Int("status", resp.StatusCode))
	}
}
