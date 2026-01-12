-- Local Groups: Admin-managed groups independent of IdP
-- Local group names are prefixed with 'local:' when returned to clients

-- Local group definitions
CREATE TABLE local_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for name lookups
CREATE INDEX idx_local_groups_name ON local_groups(name);

-- Group membership (supports both SSO and local users)
-- user_id is VARCHAR to support both SSO user UUIDs and local user IDs
-- member_type distinguishes between 'sso' and 'local' users
CREATE TABLE local_group_members (
    group_id UUID NOT NULL REFERENCES local_groups(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    member_type VARCHAR(10) NOT NULL CHECK (member_type IN ('sso', 'local')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id, member_type)
);

-- Indexes for efficient lookups
CREATE INDEX idx_local_group_members_group ON local_group_members(group_id);
CREATE INDEX idx_local_group_members_user ON local_group_members(user_id, member_type);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_local_groups_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER local_groups_updated_at
    BEFORE UPDATE ON local_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_local_groups_updated_at();
