-- Revert: restore original default
ALTER TABLE gateways ALTER COLUMN is_active SET DEFAULT TRUE;
