-- Add TLS verification settings to proxy applications
ALTER TABLE proxy_applications ADD COLUMN IF NOT EXISTS skip_tls_verify BOOLEAN DEFAULT false;
ALTER TABLE proxy_applications ADD COLUMN IF NOT EXISTS custom_ca_cert TEXT;

COMMENT ON COLUMN proxy_applications.skip_tls_verify IS 'Skip TLS certificate verification for this app (overrides global setting)';
COMMENT ON COLUMN proxy_applications.custom_ca_cert IS 'Custom CA certificate PEM for this app (optional)';
