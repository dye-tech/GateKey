-- Mesh Generated Configs table - stores mesh hub VPN configs like gateway configs
CREATE TABLE IF NOT EXISTS mesh_generated_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    hub_name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    config_data BYTEA NOT NULL,
    serial_number VARCHAR(255) NOT NULL,
    fingerprint VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    downloaded_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(255)
);

-- Indexes for mesh_generated_configs
CREATE INDEX idx_mesh_generated_configs_user_id ON mesh_generated_configs(user_id);
CREATE INDEX idx_mesh_generated_configs_hub_id ON mesh_generated_configs(hub_id);
CREATE INDEX idx_mesh_generated_configs_expires_at ON mesh_generated_configs(expires_at);
CREATE INDEX idx_mesh_generated_configs_serial_number ON mesh_generated_configs(serial_number);
CREATE INDEX idx_mesh_generated_configs_active ON mesh_generated_configs(user_id, is_revoked) WHERE is_revoked = FALSE;
