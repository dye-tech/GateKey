-- Remove full_tunnel_mode and DNS columns from gateways
ALTER TABLE gateways DROP COLUMN IF EXISTS full_tunnel_mode;
ALTER TABLE gateways DROP COLUMN IF EXISTS push_dns;
ALTER TABLE gateways DROP COLUMN IF EXISTS dns_servers;

-- Restore original config_version trigger without full_tunnel_mode and DNS
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
             COALESCE(NEW.tls_auth_key, ''))::bytea
        ),
        'hex'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
