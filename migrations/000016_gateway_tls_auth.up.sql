-- Add TLS Auth toggle for gateways
-- When disabled, TLS-Auth key won't be included in client configs

ALTER TABLE gateways ADD COLUMN IF NOT EXISTS tls_auth_enabled BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN gateways.tls_auth_enabled IS 'Enable TLS-Auth for additional security. Disable for simpler direct IP connections.';
