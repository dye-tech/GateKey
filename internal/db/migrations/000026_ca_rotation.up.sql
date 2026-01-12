-- Add CA rotation support with multi-CA trust
-- This allows graceful CA rotation with a dual-trust period

-- Add new columns to pki_ca table
ALTER TABLE pki_ca ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';
ALTER TABLE pki_ca ADD COLUMN IF NOT EXISTS fingerprint VARCHAR(64);
ALTER TABLE pki_ca ADD COLUMN IF NOT EXISTS description TEXT;

-- Remove the singleton constraint to allow multiple CAs
DROP INDEX IF EXISTS idx_pki_ca_singleton;

-- Update existing CA to have 'active' status
UPDATE pki_ca SET status = 'active' WHERE id = 'default';

-- Create index for looking up active CAs
CREATE INDEX IF NOT EXISTS idx_pki_ca_status ON pki_ca(status);

-- Add trusted_ca_chain to mesh_hubs for dual-trust during rotation
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS trusted_ca_fingerprints TEXT[];

-- Add ca_fingerprint to gateways for detecting CA changes
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS ca_fingerprint VARCHAR(64);

-- Add ca_fingerprint to mesh_gateways (spokes) for detecting CA changes
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS ca_fingerprint VARCHAR(64);

-- Create table for CA rotation events/audit
CREATE TABLE IF NOT EXISTS ca_rotation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ca_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'initiated', 'activated', 'retired', 'revoked'
    old_fingerprint VARCHAR(64),
    new_fingerprint VARCHAR(64),
    initiated_by VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ca_rotation_events_created ON ca_rotation_events(created_at DESC);
