-- Remove last_seen_at column from mesh_connections and gateway_connections
DROP INDEX IF EXISTS idx_mesh_connections_last_seen;
ALTER TABLE mesh_connections DROP COLUMN IF EXISTS last_seen_at;

DROP INDEX IF EXISTS idx_gateway_connections_last_seen;
ALTER TABLE gateway_connections DROP COLUMN IF EXISTS last_seen_at;
