-- Rollback WireGuard support from gateways table
ALTER TABLE gateways DROP CONSTRAINT IF EXISTS valid_gateway_type;
DROP INDEX IF EXISTS idx_gateways_type;
ALTER TABLE gateways DROP COLUMN IF EXISTS wg_listen_port;
ALTER TABLE gateways DROP COLUMN IF EXISTS wg_public_key;
ALTER TABLE gateways DROP COLUMN IF EXISTS wg_private_key;
ALTER TABLE gateways DROP COLUMN IF EXISTS gateway_type;
