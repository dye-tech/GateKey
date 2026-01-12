-- Remove config version tracking
DROP TRIGGER IF EXISTS trigger_gateway_config_version ON gateways;
DROP FUNCTION IF EXISTS update_gateway_config_version();
DROP FUNCTION IF EXISTS compute_gateway_config_version(VARCHAR, INTEGER, VARCHAR, CIDR, BOOLEAN, TEXT);
ALTER TABLE gateways DROP COLUMN IF EXISTS config_version;
