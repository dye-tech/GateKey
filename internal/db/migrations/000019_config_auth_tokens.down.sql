-- Remove auth token and revocation columns
DROP INDEX IF EXISTS idx_generated_configs_auth_token;
DROP INDEX IF EXISTS idx_generated_configs_active;
ALTER TABLE generated_configs DROP COLUMN IF EXISTS auth_token;
ALTER TABLE generated_configs DROP COLUMN IF EXISTS is_revoked;
ALTER TABLE generated_configs DROP COLUMN IF EXISTS revoked_at;
ALTER TABLE generated_configs DROP COLUMN IF EXISTS revoked_reason;
