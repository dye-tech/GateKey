-- Store CA certificate and key in database for consistency across pods
CREATE TABLE IF NOT EXISTS pki_ca (
    id VARCHAR(50) PRIMARY KEY DEFAULT 'default',
    certificate_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    serial_number VARCHAR(100),
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Ensure only one CA exists
CREATE UNIQUE INDEX IF NOT EXISTS idx_pki_ca_singleton ON pki_ca ((true));
