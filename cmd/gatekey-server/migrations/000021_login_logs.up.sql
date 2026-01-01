-- Login logs table for monitoring user authentication
CREATE TABLE IF NOT EXISTS login_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    provider VARCHAR(50) NOT NULL,  -- 'oidc', 'saml', 'local'
    provider_name VARCHAR(100),     -- specific provider name (e.g., 'keycloak', 'okta')
    ip_address INET NOT NULL,
    user_agent TEXT,
    country VARCHAR(100),
    city VARCHAR(100),
    success BOOLEAN NOT NULL DEFAULT true,
    failure_reason VARCHAR(255),
    session_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_login_logs_user_email ON login_logs(user_email);
CREATE INDEX idx_login_logs_user_id ON login_logs(user_id);
CREATE INDEX idx_login_logs_created_at ON login_logs(created_at DESC);
CREATE INDEX idx_login_logs_ip_address ON login_logs(ip_address);
CREATE INDEX idx_login_logs_success ON login_logs(success);

-- System settings for log retention (if not exists, add to existing settings)
INSERT INTO system_settings (key, value, description)
VALUES ('login_log_retention_days', '30', 'Number of days to retain login logs (0 = forever)')
ON CONFLICT (key) DO NOTHING;
