-- Remove disconnect request tracking from connections tables
ALTER TABLE mesh_connections
    DROP COLUMN IF EXISTS disconnect_requested_by,
    DROP COLUMN IF EXISTS disconnect_requested_at;

ALTER TABLE gateway_connections
    DROP COLUMN IF EXISTS disconnect_requested_by,
    DROP COLUMN IF EXISTS disconnect_requested_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_pending_disconnects_cleanup;
DROP INDEX IF EXISTS idx_pending_disconnects_user;
DROP INDEX IF EXISTS idx_pending_disconnects_gateway;

-- Drop table
DROP TABLE IF EXISTS pending_disconnects;
