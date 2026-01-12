-- Gateway client connections table
-- Tracks users connected to gateways (similar to mesh_connections)
CREATE TABLE gateway_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,  -- Can be SSO user UUID or local user UUID
    user_type VARCHAR(10) NOT NULL DEFAULT 'sso' CHECK (user_type IN ('sso', 'local')),

    -- Connection details
    client_ip INET NOT NULL,
    tunnel_ip INET,

    -- Traffic stats
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,

    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    disconnect_reason VARCHAR(100)
);

CREATE INDEX idx_gateway_connections_gateway_id ON gateway_connections(gateway_id);
CREATE INDEX idx_gateway_connections_user_id ON gateway_connections(user_id);
CREATE INDEX idx_gateway_connections_active ON gateway_connections(disconnected_at) WHERE disconnected_at IS NULL;
