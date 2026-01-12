-- Remove mesh hub networks
DROP TABLE IF EXISTS mesh_hub_networks;

-- Remove hub VPN settings
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS full_tunnel_mode;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS push_dns;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS dns_servers;
