-- IPv6 Support Migration
-- Adds IPv6 columns to networks, gateways, and mesh tables

-- Add IPv6 CIDR to networks table
ALTER TABLE networks ADD COLUMN IF NOT EXISTS cidr_v6 CIDR;
COMMENT ON COLUMN networks.cidr_v6 IS 'IPv6 network CIDR (optional, for dual-stack support)';

-- Add IPv6 columns to gateways table
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS public_ip_v6 INET;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS internal_ip_v6 INET;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS vpn_subnet_v6 CIDR;
COMMENT ON COLUMN gateways.public_ip_v6 IS 'Public IPv6 address of the gateway';
COMMENT ON COLUMN gateways.internal_ip_v6 IS 'Internal IPv6 address of the gateway';
COMMENT ON COLUMN gateways.vpn_subnet_v6 IS 'IPv6 VPN subnet for client IP allocation';

-- Add IPv6 VPN subnet to mesh_hubs table
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS vpn_subnet_v6 CIDR;
COMMENT ON COLUMN mesh_hubs.vpn_subnet_v6 IS 'IPv6 mesh network subnet for dual-stack support';

-- Add IPv6 local networks to mesh_gateways (spokes) table
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS local_networks_v6 TEXT[] NOT NULL DEFAULT '{}';
COMMENT ON COLUMN mesh_gateways.local_networks_v6 IS 'Array of IPv6 CIDRs for networks behind this gateway';

-- Add IPv6 tunnel IP to mesh_gateways
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS tunnel_ip_v6 INET;
COMMENT ON COLUMN mesh_gateways.tunnel_ip_v6 IS 'IPv6 tunnel IP assigned by hub';

-- Add IPv6 to mesh connections
ALTER TABLE mesh_connections ADD COLUMN IF NOT EXISTS client_ip_v6 INET;
ALTER TABLE mesh_connections ADD COLUMN IF NOT EXISTS tunnel_ip_v6 INET;
COMMENT ON COLUMN mesh_connections.client_ip_v6 IS 'Client IPv6 address';
COMMENT ON COLUMN mesh_connections.tunnel_ip_v6 IS 'Tunnel IPv6 address assigned to client';

-- Create indexes for IPv6 columns
CREATE INDEX IF NOT EXISTS idx_networks_cidr_v6 ON networks USING gist (cidr_v6 inet_ops) WHERE cidr_v6 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gateways_public_ip_v6 ON gateways (public_ip_v6) WHERE public_ip_v6 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gateways_vpn_subnet_v6 ON gateways USING gist (vpn_subnet_v6 inet_ops) WHERE vpn_subnet_v6 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_mesh_hubs_vpn_subnet_v6 ON mesh_hubs USING gist (vpn_subnet_v6 inet_ops) WHERE vpn_subnet_v6 IS NOT NULL;
