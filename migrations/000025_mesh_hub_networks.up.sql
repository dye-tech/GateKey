-- Mesh Hub Networks table
-- Links mesh hubs to networks (for access rules, similar to gateway_networks)
CREATE TABLE mesh_hub_networks (
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (hub_id, network_id)
);

CREATE INDEX idx_mesh_hub_networks_hub ON mesh_hub_networks(hub_id);
CREATE INDEX idx_mesh_hub_networks_network ON mesh_hub_networks(network_id);

-- Add TLS auth settings to mesh hubs (already have tls_auth_enabled/tls_auth_key)
-- Add full tunnel and DNS settings to mesh hubs for client configs
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS full_tunnel_mode boolean NOT NULL DEFAULT false;
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS push_dns boolean NOT NULL DEFAULT false;
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS dns_servers text[] NOT NULL DEFAULT '{}';
