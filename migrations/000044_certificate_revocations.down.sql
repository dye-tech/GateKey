-- Drop CRL cache table
DROP TABLE IF EXISTS crl_cache;

-- Drop certificate revocations table
DROP INDEX IF EXISTS idx_certificate_revocations_serial;
DROP INDEX IF EXISTS idx_certificate_revocations_revoked_at;
DROP INDEX IF EXISTS idx_certificate_revocations_ca_id;
DROP TABLE IF EXISTS certificate_revocations;
