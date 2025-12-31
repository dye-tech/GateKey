-- Add vpn_subnet column to gateways table for configurable VPN client addressing
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS vpn_subnet CIDR NOT NULL DEFAULT '10.8.0.0/24';

COMMENT ON COLUMN gateways.vpn_subnet IS 'VPN subnet for client IP allocation (e.g., 10.8.0.0/24)';
