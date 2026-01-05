-- Add WireGuard support to gateways table
-- Add gateway_type column (default 'openvpn' for backward compatibility)
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS gateway_type VARCHAR(20) NOT NULL DEFAULT 'openvpn';

-- Add WireGuard-specific columns
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS wg_private_key TEXT;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS wg_public_key TEXT;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS wg_listen_port INTEGER DEFAULT 51820;

-- Add index for gateway_type for filtering
CREATE INDEX IF NOT EXISTS idx_gateways_type ON gateways(gateway_type);

-- Add constraint for valid gateway types
ALTER TABLE gateways DROP CONSTRAINT IF EXISTS valid_gateway_type;
ALTER TABLE gateways ADD CONSTRAINT valid_gateway_type CHECK (gateway_type IN ('openvpn', 'wireguard'));
