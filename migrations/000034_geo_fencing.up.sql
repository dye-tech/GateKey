-- Geo-fencing feature migration
-- Adds IP-based geo-fencing with global, user, and group level rules

-- Geo-fencing rules (allowed IP ranges)
CREATE TABLE geo_fence_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    ip_range CIDR NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_geo_fence_rules_active ON geo_fence_rules(is_active);
CREATE INDEX idx_geo_fence_rules_ip ON geo_fence_rules USING gist (ip_range inet_ops);

-- Trigger for updated_at
CREATE TRIGGER update_geo_fence_rules_updated_at
    BEFORE UPDATE ON geo_fence_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Global rules (apply to everyone when no user/group rule matches)
CREATE TABLE geo_fence_global (
    rule_id UUID NOT NULL REFERENCES geo_fence_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (rule_id)
);

-- User-level geo-fencing (overrides global for specific users)
CREATE TABLE user_geo_fence_rules (
    user_id VARCHAR(255) NOT NULL,
    rule_id UUID NOT NULL REFERENCES geo_fence_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, rule_id)
);

CREATE INDEX idx_user_geo_fence_user ON user_geo_fence_rules(user_id);

-- Group-level geo-fencing (overrides global for group members)
CREATE TABLE group_geo_fence_rules (
    group_name VARCHAR(255) NOT NULL,
    rule_id UUID NOT NULL REFERENCES geo_fence_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (group_name, rule_id)
);

CREATE INDEX idx_group_geo_fence_group ON group_geo_fence_rules(group_name);

-- Insert default settings (disabled by default)
INSERT INTO system_settings (key, value, description) VALUES
    ('geo_fencing_enabled', 'false', 'Enable IP-based geo-fencing for VPN connections'),
    ('geo_fencing_enforce', 'audit', 'Geo-fencing mode: enforce (block) or audit (log only)')
ON CONFLICT (key) DO NOTHING;
