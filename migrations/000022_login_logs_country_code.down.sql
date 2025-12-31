-- Remove country_code column from login_logs
ALTER TABLE login_logs DROP COLUMN IF EXISTS country_code;
