-- Add WireGuard support to mesh networking tables
-- This migration adds gateway_type field to allow choosing between OpenVPN and WireGuard

-- Add WireGuard support to mesh_hubs table
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS gateway_type VARCHAR(20) NOT NULL DEFAULT 'openvpn';
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS wg_private_key TEXT;
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS wg_public_key TEXT;
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS wg_listen_port INTEGER DEFAULT 51820;

-- Add constraint to ensure valid gateway types for hubs
ALTER TABLE mesh_hubs ADD CONSTRAINT valid_mesh_hub_gateway_type
    CHECK (gateway_type IN ('openvpn', 'wireguard'));

-- Add index for filtering by gateway type
CREATE INDEX IF NOT EXISTS idx_mesh_hubs_gateway_type ON mesh_hubs(gateway_type);

-- Add WireGuard support to mesh_gateways (spokes) table
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS gateway_type VARCHAR(20) NOT NULL DEFAULT 'openvpn';
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS wg_private_key TEXT;
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS wg_public_key TEXT;
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS wg_preshared_key TEXT;

-- Add constraint to ensure valid gateway types for spokes
ALTER TABLE mesh_gateways ADD CONSTRAINT valid_mesh_spoke_gateway_type
    CHECK (gateway_type IN ('openvpn', 'wireguard'));

-- Add index for filtering by gateway type
CREATE INDEX IF NOT EXISTS idx_mesh_gateways_gateway_type ON mesh_gateways(gateway_type);

-- Comment: Spokes should inherit gateway_type from their parent hub
-- This is enforced at the application level, not DB level, to allow flexibility
