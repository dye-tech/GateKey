-- Remove TLS verification settings from proxy applications
ALTER TABLE proxy_applications DROP COLUMN IF EXISTS custom_ca_cert;
ALTER TABLE proxy_applications DROP COLUMN IF EXISTS skip_tls_verify;
