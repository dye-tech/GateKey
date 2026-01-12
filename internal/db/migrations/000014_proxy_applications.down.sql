-- Drop trigger first
DROP TRIGGER IF EXISTS proxy_applications_updated_at ON proxy_applications;

-- Drop indexes
DROP INDEX IF EXISTS idx_proxy_access_logs_time;
DROP INDEX IF EXISTS idx_proxy_access_logs_user;
DROP INDEX IF EXISTS idx_proxy_access_logs_app;
DROP INDEX IF EXISTS idx_proxy_app_rules_rule;
DROP INDEX IF EXISTS idx_proxy_app_rules_app;
DROP INDEX IF EXISTS idx_group_proxy_apps_app;
DROP INDEX IF EXISTS idx_group_proxy_apps_group;
DROP INDEX IF EXISTS idx_user_proxy_apps_app;
DROP INDEX IF EXISTS idx_user_proxy_apps_user;
DROP INDEX IF EXISTS idx_proxy_applications_active;
DROP INDEX IF EXISTS idx_proxy_applications_slug;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS proxy_access_logs;
DROP TABLE IF EXISTS proxy_application_access_rules;
DROP TABLE IF EXISTS group_proxy_applications;
DROP TABLE IF EXISTS user_proxy_applications;
DROP TABLE IF EXISTS proxy_applications;
