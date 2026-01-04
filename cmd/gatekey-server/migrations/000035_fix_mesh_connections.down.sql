-- Revert mesh_connections changes
ALTER TABLE mesh_connections DROP COLUMN IF EXISTS user_type;
ALTER TABLE mesh_connections ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
ALTER TABLE mesh_connections ADD CONSTRAINT mesh_connections_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
