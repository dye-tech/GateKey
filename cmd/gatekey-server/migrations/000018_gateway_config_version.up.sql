-- Add config version tracking for push-based config updates
-- The config_version is a hash of all gateway configuration settings
-- When settings change, the version changes, signaling gateways to reprovision

ALTER TABLE gateways ADD COLUMN IF NOT EXISTS config_version VARCHAR(64);

COMMENT ON COLUMN gateways.config_version IS 'Hash of gateway config settings. Changes trigger gateway reprovision.';

-- Create function to compute config version hash
CREATE OR REPLACE FUNCTION compute_gateway_config_version(
    p_crypto_profile VARCHAR,
    p_vpn_port INTEGER,
    p_vpn_protocol VARCHAR,
    p_vpn_subnet CIDR,
    p_tls_auth_enabled BOOLEAN,
    p_tls_auth_key TEXT
) RETURNS VARCHAR(64) AS $$
BEGIN
    RETURN encode(
        sha256(
            (COALESCE(p_crypto_profile, '') || '|' ||
             COALESCE(p_vpn_port::TEXT, '') || '|' ||
             COALESCE(p_vpn_protocol, '') || '|' ||
             COALESCE(p_vpn_subnet::TEXT, '') || '|' ||
             COALESCE(p_tls_auth_enabled::TEXT, '') || '|' ||
             COALESCE(p_tls_auth_key, ''))::bytea
        ),
        'hex'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create trigger to auto-update config_version on changes
CREATE OR REPLACE FUNCTION update_gateway_config_version() RETURNS TRIGGER AS $$
BEGIN
    NEW.config_version := compute_gateway_config_version(
        NEW.crypto_profile,
        NEW.vpn_port,
        NEW.vpn_protocol,
        NEW.vpn_subnet,
        NEW.tls_auth_enabled,
        NEW.tls_auth_key
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_gateway_config_version ON gateways;
CREATE TRIGGER trigger_gateway_config_version
    BEFORE INSERT OR UPDATE ON gateways
    FOR EACH ROW
    EXECUTE FUNCTION update_gateway_config_version();

-- Update existing gateways with their config version
UPDATE gateways SET config_version = compute_gateway_config_version(
    crypto_profile, vpn_port, vpn_protocol, vpn_subnet, tls_auth_enabled, tls_auth_key
);
