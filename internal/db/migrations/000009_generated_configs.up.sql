-- Generated VPN configurations table
CREATE TABLE IF NOT EXISTS generated_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    gateway_name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    config_data BYTEA NOT NULL,
    serial_number VARCHAR(255) NOT NULL,
    fingerprint VARCHAR(255) NOT NULL,
    cli_callback_url VARCHAR(1024),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    downloaded_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_generated_configs_user_id ON generated_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_generated_configs_gateway_id ON generated_configs(gateway_id);
CREATE INDEX IF NOT EXISTS idx_generated_configs_serial_number ON generated_configs(serial_number);
CREATE INDEX IF NOT EXISTS idx_generated_configs_expires_at ON generated_configs(expires_at);
