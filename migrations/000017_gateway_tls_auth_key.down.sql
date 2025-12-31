-- Remove TLS Auth key column
ALTER TABLE gateways DROP COLUMN IF EXISTS tls_auth_key;
