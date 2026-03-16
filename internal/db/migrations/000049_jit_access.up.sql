CREATE TABLE jit_access_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_id VARCHAR(255) NOT NULL,
    requester_email VARCHAR(255) NOT NULL,
    resource_type VARCHAR(20) NOT NULL,
    resource_id UUID NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    justification TEXT NOT NULL,
    requested_duration_minutes INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approver_id VARCHAR(255),
    approver_email VARCHAR(255),
    approval_note TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ
);

CREATE INDEX idx_jit_requests_requester ON jit_access_requests(requester_id);
CREATE INDEX idx_jit_requests_status ON jit_access_requests(status);
CREATE INDEX idx_jit_requests_expires ON jit_access_requests(expires_at) WHERE status = 'pending';

CREATE TABLE jit_access_grants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES jit_access_requests(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(20) NOT NULL,
    resource_id UUID NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(255),
    revoke_reason VARCHAR(255)
);

CREATE INDEX idx_jit_grants_user ON jit_access_grants(user_id);
CREATE INDEX idx_jit_grants_active ON jit_access_grants(is_active, expires_at) WHERE is_active = true;
CREATE INDEX idx_jit_grants_resource ON jit_access_grants(resource_type, resource_id);
