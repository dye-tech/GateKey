-- DNS rules per network
CREATE TABLE dns_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    dns_servers TEXT[] NOT NULL DEFAULT '{}',
    search_domains TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_dns_rules_network ON dns_rules(network_id);

-- Static DNS records
CREATE TABLE dns_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dns_records_network ON dns_records(network_id);
CREATE INDEX idx_dns_records_hostname ON dns_records(hostname);
