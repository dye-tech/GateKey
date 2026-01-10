-- Add IPv6 support to gateways table
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS public_ip_v6 INET;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS vpn_subnet_v6 CIDR;

-- Add IPv6 support to mesh_hubs table
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS vpn_subnet_v6 CIDR;

-- Add IPv6 support to mesh_gateways (spokes) table
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS tunnel_ip_v6 INET;

-- Add IPv6 support to connections table (if not already present)
-- Note: vpn_ipv6 should exist from initial migration, but add if missing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'connections' AND column_name = 'vpn_ipv6') THEN
        ALTER TABLE connections ADD COLUMN vpn_ipv6 INET;
    END IF;
END $$;

-- Add IPv6 support to wg_mesh_peers table
ALTER TABLE wg_mesh_peers ADD COLUMN IF NOT EXISTS assigned_ip_v6 INET;
