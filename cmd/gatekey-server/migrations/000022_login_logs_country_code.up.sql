-- Add country_code column to login_logs for flag display
ALTER TABLE login_logs ADD COLUMN IF NOT EXISTS country_code VARCHAR(2);
