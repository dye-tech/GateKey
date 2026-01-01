-- Add TLS Auth key storage for gateways
-- The key is generated during provisioning and stored for client config generation
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS tls_auth_key TEXT;

COMMENT ON COLUMN gateways.tls_auth_key IS 'TLS-Auth static key for additional security. Generated during provisioning.';
