-- Add FIPS requirement setting
INSERT INTO system_settings (key, value, description) VALUES
    ('require_fips', 'false', 'Require FIPS 140-2 compliance for VPN connections')
ON CONFLICT (key) DO NOTHING;
