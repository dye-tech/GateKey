-- Add last_seen_at column to mesh_connections and gateway_connections for real-time monitoring
-- This enables detection of stale connections when the hub/gateway stops reporting them

-- Mesh connections
ALTER TABLE mesh_connections ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ DEFAULT NOW();
UPDATE mesh_connections SET last_seen_at = NOW() WHERE disconnected_at IS NULL AND last_seen_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mesh_connections_last_seen ON mesh_connections(last_seen_at) WHERE disconnected_at IS NULL;

-- Gateway connections
ALTER TABLE gateway_connections ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ DEFAULT NOW();
UPDATE gateway_connections SET last_seen_at = NOW() WHERE disconnected_at IS NULL AND last_seen_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_gateway_connections_last_seen ON gateway_connections(last_seen_at) WHERE disconnected_at IS NULL;
