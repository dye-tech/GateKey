-- Rollback WireGuard mesh client support

-- Drop wg_mesh_peers table
DROP TABLE IF EXISTS wg_mesh_peers;

-- Remove columns from mesh_generated_configs
ALTER TABLE mesh_generated_configs DROP COLUMN IF EXISTS assigned_ip;
ALTER TABLE mesh_generated_configs DROP COLUMN IF EXISTS is_wireguard;
ALTER TABLE mesh_generated_configs DROP COLUMN IF EXISTS public_key;
ALTER TABLE mesh_generated_configs DROP COLUMN IF EXISTS preshared_key;
