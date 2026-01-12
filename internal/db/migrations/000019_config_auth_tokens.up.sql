-- Add auth token and revocation support for VPN configs
-- Each config gets a unique auth token that can be revoked

ALTER TABLE generated_configs ADD COLUMN IF NOT EXISTS auth_token VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE generated_configs ADD COLUMN IF NOT EXISTS is_revoked BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE generated_configs ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;
ALTER TABLE generated_configs ADD COLUMN IF NOT EXISTS revoked_reason VARCHAR(255);

-- Index for fast token lookups during authentication
CREATE INDEX IF NOT EXISTS idx_generated_configs_auth_token ON generated_configs(auth_token) WHERE auth_token != '';

-- Index for finding active (non-revoked) configs
CREATE INDEX IF NOT EXISTS idx_generated_configs_active ON generated_configs(user_id, is_revoked) WHERE is_revoked = FALSE;

COMMENT ON COLUMN generated_configs.auth_token IS 'Unique authentication token embedded in the .ovpn config for password auth';
COMMENT ON COLUMN generated_configs.is_revoked IS 'Whether this config has been revoked';
COMMENT ON COLUMN generated_configs.revoked_at IS 'When the config was revoked';
COMMENT ON COLUMN generated_configs.revoked_reason IS 'Reason for revocation';
