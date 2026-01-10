-- Remove IPv6 columns from gateways table
ALTER TABLE gateways DROP COLUMN IF EXISTS public_ip_v6;
ALTER TABLE gateways DROP COLUMN IF EXISTS vpn_subnet_v6;

-- Remove IPv6 columns from mesh_hubs table
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS vpn_subnet_v6;

-- Remove IPv6 columns from mesh_gateways (spokes) table
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS tunnel_ip_v6;

-- Remove IPv6 columns from wg_mesh_peers table
ALTER TABLE wg_mesh_peers DROP COLUMN IF EXISTS assigned_ip_v6;
