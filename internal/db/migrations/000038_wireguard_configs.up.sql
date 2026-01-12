-- WireGuard client configurations table
-- Similar to generated_configs for OpenVPN, but with WireGuard-specific fields
CREATE TABLE IF NOT EXISTS wireguard_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    gateway_name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    config_data BYTEA NOT NULL,
    -- WireGuard-specific fields
    client_private_key TEXT NOT NULL,
    client_public_key TEXT NOT NULL,
    assigned_ip CIDR NOT NULL,
    preshared_key TEXT,
    -- Standard tracking fields
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    downloaded_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_wg_configs_user_id ON wireguard_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_wg_configs_gateway_id ON wireguard_configs(gateway_id);
CREATE INDEX IF NOT EXISTS idx_wg_configs_expires_at ON wireguard_configs(expires_at);
CREATE INDEX IF NOT EXISTS idx_wg_configs_client_public_key ON wireguard_configs(client_public_key);
CREATE INDEX IF NOT EXISTS idx_wg_configs_revoked ON wireguard_configs(revoked_at) WHERE revoked_at IS NULL;

-- WireGuard active peers table for tracking connections on gateway
CREATE TABLE IF NOT EXISTS wireguard_peers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES wireguard_configs(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    user_type VARCHAR(10) NOT NULL DEFAULT 'sso',
    public_key TEXT NOT NULL,
    assigned_ip CIDR NOT NULL,
    allowed_ips TEXT[] NOT NULL DEFAULT '{}',
    endpoint TEXT,
    last_handshake TIMESTAMPTZ,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    UNIQUE(gateway_id, public_key)
);

CREATE INDEX IF NOT EXISTS idx_wg_peers_gateway_id ON wireguard_peers(gateway_id);
CREATE INDEX IF NOT EXISTS idx_wg_peers_user_id ON wireguard_peers(user_id);
CREATE INDEX IF NOT EXISTS idx_wg_peers_config_id ON wireguard_peers(config_id);
CREATE INDEX IF NOT EXISTS idx_wg_peers_active ON wireguard_peers(gateway_id) WHERE disconnected_at IS NULL;
