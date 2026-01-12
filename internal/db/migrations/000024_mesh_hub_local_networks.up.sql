-- Add local_networks to mesh_hubs for routes the hub can directly reach
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS local_networks text[] NOT NULL DEFAULT '{}';

-- Add settings for spokes: full_tunnel_mode, push_dns, dns_servers
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS full_tunnel_mode boolean NOT NULL DEFAULT false;
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS push_dns boolean NOT NULL DEFAULT false;
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS dns_servers text[] NOT NULL DEFAULT '{}';
