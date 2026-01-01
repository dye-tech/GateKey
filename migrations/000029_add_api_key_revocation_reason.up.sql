-- Add missing revocation_reason column to api_keys table
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS revocation_reason TEXT;
