-- Rollback WireGuard support from mesh networking tables

-- Remove indexes first
DROP INDEX IF EXISTS idx_mesh_hubs_gateway_type;
DROP INDEX IF EXISTS idx_mesh_gateways_gateway_type;

-- Remove constraints
ALTER TABLE mesh_hubs DROP CONSTRAINT IF EXISTS valid_mesh_hub_gateway_type;
ALTER TABLE mesh_gateways DROP CONSTRAINT IF EXISTS valid_mesh_spoke_gateway_type;

-- Remove WireGuard columns from mesh_gateways (spokes)
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS gateway_type;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS wg_private_key;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS wg_public_key;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS wg_preshared_key;

-- Remove WireGuard columns from mesh_hubs
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS gateway_type;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS wg_private_key;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS wg_public_key;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS wg_listen_port;
