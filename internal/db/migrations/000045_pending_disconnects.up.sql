-- Pending disconnects table for immediate user session termination
-- Gateway agents poll this table to disconnect specific users
CREATE TABLE IF NOT EXISTS pending_disconnects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Target identification (individual user, NOT group)
    user_id UUID NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    -- Gateway/Hub where user is connected (NULL = all gateways)
    gateway_id UUID,
    node_type VARCHAR(20) NOT NULL DEFAULT 'gateway', -- 'gateway' or 'hub'
    -- Connection details for precise targeting
    connection_id UUID, -- Specific connection to disconnect (optional)
    vpn_address VARCHAR(45), -- User's VPN IP for identification
    -- Admin action metadata
    reason VARCHAR(500) NOT NULL DEFAULT 'Disconnected by administrator',
    requested_by VARCHAR(255) NOT NULL, -- Admin who initiated
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Execution tracking
    executed_at TIMESTAMP WITH TIME ZONE, -- When gateway executed disconnect
    executed_by_gateway UUID, -- Which gateway executed it
    -- Expiration (old requests are cleaned up)
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() + INTERVAL '5 minutes',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for gateway polling (find pending disconnects for a specific gateway)
CREATE INDEX idx_pending_disconnects_gateway ON pending_disconnects(gateway_id, node_type)
    WHERE executed_at IS NULL AND expires_at > NOW();

-- Index for user lookup (find all pending disconnects for a user)
CREATE INDEX idx_pending_disconnects_user ON pending_disconnects(user_id)
    WHERE executed_at IS NULL;

-- Index for cleanup of expired/executed records
CREATE INDEX idx_pending_disconnects_cleanup ON pending_disconnects(expires_at)
    WHERE executed_at IS NOT NULL OR expires_at < NOW();

-- Add disconnect_requested_at to gateway_connections for tracking
ALTER TABLE gateway_connections
    ADD COLUMN IF NOT EXISTS disconnect_requested_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS disconnect_requested_by VARCHAR(255);

-- Add disconnect_requested_at to mesh_connections for tracking
ALTER TABLE mesh_connections
    ADD COLUMN IF NOT EXISTS disconnect_requested_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS disconnect_requested_by VARCHAR(255);
