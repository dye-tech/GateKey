-- Gatex Initial Schema Rollback
-- This migration drops all core tables.

-- Drop triggers first
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_gateways_updated_at ON gateways;
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS configs;
DROP TABLE IF EXISTS connections;
DROP TABLE IF EXISTS policy_rules;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS certificates;
DROP TABLE IF EXISTS gateways;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

-- Note: We don't drop the uuid-ossp extension as it may be used by other schemas
