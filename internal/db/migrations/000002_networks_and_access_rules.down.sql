-- Drop triggers
DROP TRIGGER IF EXISTS access_rules_updated_at ON access_rules;
DROP TRIGGER IF EXISTS networks_updated_at ON networks;
DROP FUNCTION IF EXISTS update_networks_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_group_access_rules_group;
DROP INDEX IF EXISTS idx_user_access_rules_user;
DROP INDEX IF EXISTS idx_access_rules_type;
DROP INDEX IF EXISTS idx_access_rules_network;
DROP INDEX IF EXISTS idx_gateway_networks_network;
DROP INDEX IF EXISTS idx_gateway_networks_gateway;
DROP INDEX IF EXISTS idx_networks_cidr;

-- Drop tables
DROP TABLE IF EXISTS group_access_rules;
DROP TABLE IF EXISTS user_access_rules;
DROP TABLE IF EXISTS access_rules;
DROP TABLE IF EXISTS gateway_networks;
DROP TABLE IF EXISTS networks;
