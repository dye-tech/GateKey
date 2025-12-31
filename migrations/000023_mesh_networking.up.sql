-- Mesh Networking Migration
-- Adds tables for Hub and Mesh Gateway management

-- Mesh Hubs table
-- Stores standalone hubs that run OpenVPN server and accept gateway connections
CREATE TABLE mesh_hubs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',

    -- Hub endpoint configuration
    public_endpoint VARCHAR(255) NOT NULL,  -- hostname:port that gateways connect to
    vpn_port INTEGER NOT NULL DEFAULT 1194,
    vpn_protocol VARCHAR(10) NOT NULL DEFAULT 'udp',
    vpn_subnet CIDR NOT NULL DEFAULT '172.30.0.0/16',  -- Mesh network subnet

    -- Crypto configuration
    crypto_profile VARCHAR(50) NOT NULL DEFAULT 'fips',
    tls_auth_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tls_auth_key TEXT,

    -- PKI - Hub's own CA for mesh network
    ca_cert TEXT,
    ca_key TEXT,
    server_cert TEXT,
    server_key TEXT,
    dh_params TEXT,

    -- Control plane communication
    api_token VARCHAR(64) NOT NULL,  -- Token for hub to authenticate with control plane
    control_plane_url TEXT NOT NULL,  -- URL of the GateKey control plane

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, online, offline, error
    status_message TEXT,
    last_heartbeat TIMESTAMPTZ,
    connected_gateways INTEGER NOT NULL DEFAULT 0,
    connected_clients INTEGER NOT NULL DEFAULT 0,

    -- Config versioning for auto-updates
    config_version VARCHAR(64),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mesh_hubs_name ON mesh_hubs(name);
CREATE INDEX idx_mesh_hubs_status ON mesh_hubs(status);

CREATE TRIGGER update_mesh_hubs_updated_at
    BEFORE UPDATE ON mesh_hubs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Mesh Gateways table
-- Stores remote gateways that connect TO a hub (gateway acts as OpenVPN client)
CREATE TABLE mesh_gateways (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Networks behind this gateway (advertised via iroute)
    local_networks TEXT[] NOT NULL DEFAULT '{}',  -- Array of CIDRs like {'10.0.0.0/8', '192.168.1.0/24'}

    -- Assigned tunnel IP (assigned by hub)
    tunnel_ip INET,

    -- Client certificates for connecting to hub
    client_cert TEXT,
    client_key TEXT,

    -- Gateway token for provisioning/authentication
    token VARCHAR(64) NOT NULL,

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, connected, disconnected, error
    status_message TEXT,
    last_seen TIMESTAMPTZ,
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,

    -- The remote public IP when connected (for diagnostics)
    remote_ip INET,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(hub_id, name)
);

CREATE INDEX idx_mesh_gateways_hub_id ON mesh_gateways(hub_id);
CREATE INDEX idx_mesh_gateways_status ON mesh_gateways(status);
CREATE INDEX idx_mesh_gateways_token ON mesh_gateways(token);

CREATE TRIGGER update_mesh_gateways_updated_at
    BEFORE UPDATE ON mesh_gateways
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Mesh Hub Access table
-- Controls which users/groups can access routes through the mesh
CREATE TABLE mesh_hub_users (
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (hub_id, user_id)
);

CREATE TABLE mesh_hub_groups (
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    group_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (hub_id, group_name)
);

-- Mesh Gateway Access table
-- Controls which users/groups can access specific gateway routes
CREATE TABLE mesh_gateway_users (
    gateway_id UUID NOT NULL REFERENCES mesh_gateways(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gateway_id, user_id)
);

CREATE TABLE mesh_gateway_groups (
    gateway_id UUID NOT NULL REFERENCES mesh_gateways(id) ON DELETE CASCADE,
    group_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gateway_id, group_name)
);

-- Mesh client connections table
-- Tracks users connected to the mesh hub
CREATE TABLE mesh_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hub_id UUID NOT NULL REFERENCES mesh_hubs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Connection details
    client_ip INET NOT NULL,
    tunnel_ip INET NOT NULL,

    -- Traffic stats
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,

    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    disconnect_reason VARCHAR(100)
);

CREATE INDEX idx_mesh_connections_hub_id ON mesh_connections(hub_id);
CREATE INDEX idx_mesh_connections_user_id ON mesh_connections(user_id);
CREATE INDEX idx_mesh_connections_active ON mesh_connections(disconnected_at) WHERE disconnected_at IS NULL;
