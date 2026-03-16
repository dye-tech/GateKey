CREATE TABLE network_flow_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gateway_id UUID NOT NULL,
    gateway_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    source_ip INET NOT NULL,
    dest_ip INET NOT NULL,
    dest_port INTEGER,
    protocol VARCHAR(10),
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,
    flow_start TIMESTAMPTZ NOT NULL,
    flow_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_flow_logs_gateway ON network_flow_logs(gateway_id);
CREATE INDEX idx_flow_logs_user ON network_flow_logs(user_id);
CREATE INDEX idx_flow_logs_dest ON network_flow_logs(dest_ip);
CREATE INDEX idx_flow_logs_created ON network_flow_logs(created_at DESC);
CREATE INDEX idx_flow_logs_user_dest ON network_flow_logs(user_id, dest_ip);
