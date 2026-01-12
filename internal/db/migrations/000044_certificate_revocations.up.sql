-- Certificate Revocations table for CRL support
CREATE TABLE IF NOT EXISTS certificate_revocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number VARCHAR(64) NOT NULL UNIQUE,  -- Certificate serial number (hex)
    ca_id VARCHAR(255) NOT NULL,                -- Which CA issued this certificate
    common_name VARCHAR(255),                   -- Subject CN of the revoked certificate
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    reason INTEGER NOT NULL DEFAULT 0,          -- RFC 5280 revocation reason code
    revoked_by VARCHAR(255),                    -- Who revoked this certificate
    notes TEXT,                                 -- Optional notes about revocation
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for CRL generation (by CA and time)
CREATE INDEX IF NOT EXISTS idx_certificate_revocations_ca_id ON certificate_revocations(ca_id);
CREATE INDEX IF NOT EXISTS idx_certificate_revocations_revoked_at ON certificate_revocations(revoked_at);
CREATE INDEX IF NOT EXISTS idx_certificate_revocations_serial ON certificate_revocations(serial_number);

-- CRL cache table for performance (optional)
CREATE TABLE IF NOT EXISTS crl_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ca_id VARCHAR(255) NOT NULL UNIQUE,
    crl_der BYTEA NOT NULL,                     -- DER-encoded CRL
    this_update TIMESTAMP WITH TIME ZONE NOT NULL,
    next_update TIMESTAMP WITH TIME ZONE NOT NULL,
    crl_number BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
