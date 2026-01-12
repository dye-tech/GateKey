-- User-Gateway direct assignment
-- Allows assigning users directly to specific gateways
-- user_id is VARCHAR to support both local admin usernames and SSO user IDs
CREATE TABLE user_gateways (
    user_id VARCHAR(255) NOT NULL,
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, gateway_id)
);

-- Group-Gateway direct assignment
-- Allows assigning groups (by name) to specific gateways
CREATE TABLE group_gateways (
    group_name VARCHAR(255) NOT NULL,
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (group_name, gateway_id)
);

-- Indexes for efficient lookups
CREATE INDEX idx_user_gateways_user ON user_gateways(user_id);
CREATE INDEX idx_user_gateways_gateway ON user_gateways(gateway_id);
CREATE INDEX idx_group_gateways_group ON group_gateways(group_name);
CREATE INDEX idx_group_gateways_gateway ON group_gateways(gateway_id);
