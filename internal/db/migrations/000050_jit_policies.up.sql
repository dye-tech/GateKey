CREATE TABLE jit_access_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    resource_type VARCHAR(20) NOT NULL,       -- 'access_rule', 'network', 'gateway', 'proxy_app', 'mesh_hub', '*'
    resource_id UUID,                         -- NULL = all resources of that type
    max_duration_minutes INTEGER NOT NULL DEFAULT 480,
    default_duration_minutes INTEGER NOT NULL DEFAULT 60,
    request_expiry_minutes INTEGER NOT NULL DEFAULT 60,
    auto_approve BOOLEAN NOT NULL DEFAULT false,
    require_justification BOOLEAN NOT NULL DEFAULT true,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jit_policies_resource ON jit_access_policies(resource_type, resource_id);
CREATE INDEX idx_jit_policies_active ON jit_access_policies(is_active) WHERE is_active = true;
