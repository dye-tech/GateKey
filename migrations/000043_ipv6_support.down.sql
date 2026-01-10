-- IPv6 Support Migration Rollback
-- Removes IPv6 columns from networks, gateways, and mesh tables

-- Drop indexes first
DROP INDEX IF EXISTS idx_mesh_hubs_vpn_subnet_v6;
DROP INDEX IF EXISTS idx_gateways_vpn_subnet_v6;
DROP INDEX IF EXISTS idx_gateways_public_ip_v6;
DROP INDEX IF EXISTS idx_networks_cidr_v6;

-- Remove IPv6 columns from mesh_connections
ALTER TABLE mesh_connections DROP COLUMN IF EXISTS tunnel_ip_v6;
ALTER TABLE mesh_connections DROP COLUMN IF EXISTS client_ip_v6;

-- Remove IPv6 columns from mesh_gateways
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS tunnel_ip_v6;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS local_networks_v6;

-- Remove IPv6 VPN subnet from mesh_hubs
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS vpn_subnet_v6;

-- Remove IPv6 columns from gateways
ALTER TABLE gateways DROP COLUMN IF EXISTS vpn_subnet_v6;
ALTER TABLE gateways DROP COLUMN IF EXISTS internal_ip_v6;
ALTER TABLE gateways DROP COLUMN IF EXISTS public_ip_v6;

-- Remove IPv6 CIDR from networks
ALTER TABLE networks DROP COLUMN IF EXISTS cidr_v6;
