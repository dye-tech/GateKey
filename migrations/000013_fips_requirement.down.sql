-- Remove FIPS requirement setting
DELETE FROM system_settings WHERE key = 'require_fips';
