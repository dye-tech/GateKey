-- Remove login logs table and related objects
DROP INDEX IF EXISTS idx_login_logs_success;
DROP INDEX IF EXISTS idx_login_logs_ip_address;
DROP INDEX IF EXISTS idx_login_logs_created_at;
DROP INDEX IF EXISTS idx_login_logs_user_id;
DROP INDEX IF EXISTS idx_login_logs_user_email;
DROP TABLE IF EXISTS login_logs;

-- Remove the setting
DELETE FROM system_settings WHERE key = 'login_log_retention_days';
