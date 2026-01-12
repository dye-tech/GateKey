-- Add crypto_profile column to gateways table
-- Allows selection of cryptographic profiles including FIPS-compliant options

ALTER TABLE gateways ADD COLUMN IF NOT EXISTS crypto_profile VARCHAR(50) NOT NULL DEFAULT 'modern';

-- Valid profiles:
-- 'modern'     - Modern secure defaults (AES-256-GCM, SHA256, ECDH)
-- 'fips'       - FIPS 140-2 compliant (AES-256-GCM, SHA256, approved TLS suites)
-- 'compatible' - Maximum compatibility with older clients

COMMENT ON COLUMN gateways.crypto_profile IS 'Cryptographic profile: modern (default), fips (FIPS 140-2 compliant), compatible (legacy support)';
