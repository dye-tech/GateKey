-- Revert CA rotation support

DROP TABLE IF EXISTS ca_rotation_events;

ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS ca_fingerprint;
ALTER TABLE gateways DROP COLUMN IF EXISTS ca_fingerprint;
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS trusted_ca_fingerprints;

-- Restore singleton constraint
CREATE UNIQUE INDEX IF NOT EXISTS idx_pki_ca_singleton ON pki_ca ((true));

ALTER TABLE pki_ca DROP COLUMN IF EXISTS description;
ALTER TABLE pki_ca DROP COLUMN IF EXISTS fingerprint;
ALTER TABLE pki_ca DROP COLUMN IF EXISTS status;
