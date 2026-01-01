-- Remove revocation_reason column from api_keys table
ALTER TABLE api_keys DROP COLUMN IF EXISTS revocation_reason;
