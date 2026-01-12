-- Remove TLS Auth toggle
ALTER TABLE gateways DROP COLUMN IF EXISTS tls_auth_enabled;
