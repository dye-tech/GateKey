-- Networks table: defines CIDR blocks that gateways can serve
CREATE TABLE IF NOT EXISTS networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    cidr CIDR NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Gateway-Network associations (many-to-many)
CREATE TABLE IF NOT EXISTS gateway_networks (
    gateway_id UUID NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (gateway_id, network_id)
);

-- Access rules: IP addresses and hostnames that users/groups can access
CREATE TABLE IF NOT EXISTS access_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('ip', 'cidr', 'hostname', 'hostname_wildcard')),
    value VARCHAR(512) NOT NULL,  -- IP address, CIDR, or hostname
    port_range VARCHAR(50),       -- Optional: e.g., "80", "443", "8000-9000", "*"
    protocol VARCHAR(20),         -- Optional: tcp, udp, icmp, or * for all
    network_id UUID REFERENCES networks(id) ON DELETE CASCADE,  -- Optional: restrict to specific network
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User access rule assignments
CREATE TABLE IF NOT EXISTS user_access_rules (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_rule_id UUID NOT NULL REFERENCES access_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, access_rule_id)
);

-- Group access rule assignments
CREATE TABLE IF NOT EXISTS group_access_rules (
    group_name VARCHAR(255) NOT NULL,
    access_rule_id UUID NOT NULL REFERENCES access_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (group_name, access_rule_id)
);

-- Indexes for performance
CREATE INDEX idx_networks_cidr ON networks USING gist (cidr inet_ops);
CREATE INDEX idx_gateway_networks_gateway ON gateway_networks(gateway_id);
CREATE INDEX idx_gateway_networks_network ON gateway_networks(network_id);
CREATE INDEX idx_access_rules_network ON access_rules(network_id);
CREATE INDEX idx_access_rules_type ON access_rules(rule_type);
CREATE INDEX idx_user_access_rules_user ON user_access_rules(user_id);
CREATE INDEX idx_group_access_rules_group ON group_access_rules(group_name);

-- Update trigger for networks
CREATE OR REPLACE FUNCTION update_networks_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER networks_updated_at
    BEFORE UPDATE ON networks
    FOR EACH ROW
    EXECUTE FUNCTION update_networks_updated_at();

-- Update trigger for access_rules
CREATE TRIGGER access_rules_updated_at
    BEFORE UPDATE ON access_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_networks_updated_at();
