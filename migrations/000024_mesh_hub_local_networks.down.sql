-- Remove local_networks from mesh_hubs
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS local_networks;

-- Remove settings from mesh_gateways
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS full_tunnel_mode;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS push_dns;
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS dns_servers;
