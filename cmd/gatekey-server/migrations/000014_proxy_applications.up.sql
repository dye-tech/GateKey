-- Proxy applications: web apps accessible through the reverse proxy
CREATE TABLE IF NOT EXISTS proxy_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    internal_url TEXT NOT NULL,
    icon_url TEXT,
    is_active BOOLEAN DEFAULT true,
    preserve_host_header BOOLEAN DEFAULT false,
    strip_prefix BOOLEAN DEFAULT true,
    inject_headers JSONB DEFAULT '{}',
    allowed_headers JSONB DEFAULT '["*"]',
    websocket_enabled BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 30,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Direct user assignments to proxy applications
CREATE TABLE IF NOT EXISTS user_proxy_applications (
    user_id VARCHAR(255) NOT NULL,
    proxy_app_id UUID NOT NULL REFERENCES proxy_applications(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, proxy_app_id)
);

-- Direct group assignments to proxy applications
CREATE TABLE IF NOT EXISTS group_proxy_applications (
    group_name VARCHAR(255) NOT NULL,
    proxy_app_id UUID NOT NULL REFERENCES proxy_applications(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_name, proxy_app_id)
);

-- Link proxy applications to existing access rules (hybrid access control)
CREATE TABLE IF NOT EXISTS proxy_application_access_rules (
    proxy_app_id UUID NOT NULL REFERENCES proxy_applications(id) ON DELETE CASCADE,
    access_rule_id UUID NOT NULL REFERENCES access_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (proxy_app_id, access_rule_id)
);

-- Proxy access audit log
CREATE TABLE IF NOT EXISTS proxy_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proxy_app_id UUID NOT NULL REFERENCES proxy_applications(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255),
    request_method VARCHAR(10) NOT NULL,
    request_path TEXT NOT NULL,
    response_status INTEGER,
    response_time_ms INTEGER,
    client_ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for proxy_applications
CREATE INDEX IF NOT EXISTS idx_proxy_applications_slug ON proxy_applications(slug);
CREATE INDEX IF NOT EXISTS idx_proxy_applications_active ON proxy_applications(is_active) WHERE is_active = true;

-- Indexes for user_proxy_applications
CREATE INDEX IF NOT EXISTS idx_user_proxy_apps_user ON user_proxy_applications(user_id);
CREATE INDEX IF NOT EXISTS idx_user_proxy_apps_app ON user_proxy_applications(proxy_app_id);

-- Indexes for group_proxy_applications
CREATE INDEX IF NOT EXISTS idx_group_proxy_apps_group ON group_proxy_applications(group_name);
CREATE INDEX IF NOT EXISTS idx_group_proxy_apps_app ON group_proxy_applications(proxy_app_id);

-- Indexes for proxy_application_access_rules
CREATE INDEX IF NOT EXISTS idx_proxy_app_rules_app ON proxy_application_access_rules(proxy_app_id);
CREATE INDEX IF NOT EXISTS idx_proxy_app_rules_rule ON proxy_application_access_rules(access_rule_id);

-- Indexes for proxy_access_logs
CREATE INDEX IF NOT EXISTS idx_proxy_access_logs_app ON proxy_access_logs(proxy_app_id);
CREATE INDEX IF NOT EXISTS idx_proxy_access_logs_user ON proxy_access_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_proxy_access_logs_time ON proxy_access_logs(created_at);

-- Update trigger for proxy_applications
CREATE TRIGGER proxy_applications_updated_at
    BEFORE UPDATE ON proxy_applications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
