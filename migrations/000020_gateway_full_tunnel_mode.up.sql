-- Add full_tunnel_mode column to gateways
-- When enabled, pushes redirect-gateway def1 bypass-dhcp to route all traffic through VPN
-- Default is false (split tunnel mode - only push routes for allowed networks)

ALTER TABLE gateways ADD COLUMN IF NOT EXISTS full_tunnel_mode BOOLEAN NOT NULL DEFAULT false;

-- Add DNS configuration columns
-- push_dns: When true, push DNS servers to VPN clients. Default is false (don't override client DNS)
-- dns_servers: Array of DNS server IPs to push to clients (e.g., ['1.1.1.1', '8.8.8.8'])
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS push_dns BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS dns_servers TEXT[] NOT NULL DEFAULT '{}';

-- Update config_version trigger to include full_tunnel_mode and DNS settings
CREATE OR REPLACE FUNCTION update_gateway_config_version()
RETURNS TRIGGER AS $$
BEGIN
    -- Compute config version based on settings that affect gateway config
    NEW.config_version := encode(
        sha256(
            (COALESCE(NEW.crypto_profile, '') || '|' ||
             COALESCE(NEW.vpn_port::text, '') || '|' ||
             COALESCE(NEW.vpn_protocol, '') || '|' ||
             COALESCE(NEW.vpn_subnet::text, '') || '|' ||
             COALESCE(NEW.tls_auth_enabled::text, '') || '|' ||
             COALESCE(NEW.tls_auth_key, '') || '|' ||
             COALESCE(NEW.full_tunnel_mode::text, '') || '|' ||
             COALESCE(NEW.push_dns::text, '') || '|' ||
             COALESCE(array_to_string(NEW.dns_servers, ','), ''))::bytea
        ),
        'hex'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON COLUMN gateways.full_tunnel_mode IS 'When true, all client traffic routes through VPN (redirect-gateway). When false, only routes for allowed networks are pushed.';
COMMENT ON COLUMN gateways.push_dns IS 'When true, push DNS servers to VPN clients. Default is false (don''t override client DNS).';
COMMENT ON COLUMN gateways.dns_servers IS 'Array of DNS server IPs to push to clients.';
