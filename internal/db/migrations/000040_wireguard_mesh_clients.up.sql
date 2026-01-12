-- Add WireGuard mesh client support
-- This allows mobile/desktop clients to connect directly to WireGuard mesh hubs

-- Add assigned_ip and is_wireguard columns to mesh_generated_configs
ALTER TABLE mesh_generated_configs ADD COLUMN IF NOT EXISTS assigned_ip INET;
ALTER TABLE mesh_generated_configs ADD COLUMN IF NOT EXISTS is_wireguard BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE mesh_generated_configs ADD COLUMN IF NOT EXISTS public_key TEXT;
ALTER TABLE mesh_generated_configs ADD COLUMN IF NOT EXISTS preshared_key TEXT;

-- Create index for finding WireGuard configs by hub
CREATE INDEX IF NOT EXISTS idx_mesh_generated_configs_wg_hub
    ON mesh_generated_configs(hub_id, is_wireguard)
    WHERE is_wireguard = TRUE;

-- Create index for finding assigned IPs
CREATE INDEX IF NOT EXISTS idx_mesh_generated_configs_assigned_ip
    ON mesh_generated_configs(hub_id, assigned_ip)
    WHERE assigned_ip IS NOT NULL;

-- Create table for tracking active WireGuard mesh peers (for hub to sync)
CREATE TABLE IF NOT EXISTS wg_mesh_peers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    config_id UUID REFERENCES mesh_generated_configs(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    public_key TEXT NOT NULL,
    preshared_key TEXT,
    assigned_ip INET NOT NULL,
    allowed_ips TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT unique_wg_mesh_peer_pubkey UNIQUE (hub_id, public_key)
);

-- Indexes for wg_mesh_peers
CREATE INDEX IF NOT EXISTS idx_wg_mesh_peers_hub_id ON wg_mesh_peers(hub_id);
CREATE INDEX IF NOT EXISTS idx_wg_mesh_peers_user_id ON wg_mesh_peers(user_id);
CREATE INDEX IF NOT EXISTS idx_wg_mesh_peers_expires_at ON wg_mesh_peers(expires_at);
CREATE INDEX IF NOT EXISTS idx_wg_mesh_peers_active
    ON wg_mesh_peers(hub_id, is_revoked, expires_at)
    WHERE is_revoked = FALSE;
