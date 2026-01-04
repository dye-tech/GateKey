-- Fix mesh_connections table to support both SSO and local users
-- Similar to gateway_connections which already supports both

-- Drop the foreign key constraint on user_id
ALTER TABLE mesh_connections DROP CONSTRAINT IF EXISTS mesh_connections_user_id_fkey;

-- Change user_id from UUID to VARCHAR to support both user types
ALTER TABLE mesh_connections ALTER COLUMN user_id TYPE VARCHAR(255);

-- Add user_type column to distinguish between SSO and local users
ALTER TABLE mesh_connections ADD COLUMN IF NOT EXISTS user_type VARCHAR(10) NOT NULL DEFAULT 'sso' CHECK (user_type IN ('sso', 'local'));
