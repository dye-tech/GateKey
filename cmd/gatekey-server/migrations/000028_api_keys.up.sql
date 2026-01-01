-- API Keys table for programmatic authentication
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    key_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 hash
    key_prefix VARCHAR(12) NOT NULL,       -- "gk_xxxx..." for display
    scopes JSONB NOT NULL DEFAULT '[]',
    is_admin_provisioned BOOLEAN DEFAULT FALSE,
    provisioned_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    last_used_ip INET,
    is_revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id),
    revocation_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_is_revoked ON api_keys(is_revoked);
