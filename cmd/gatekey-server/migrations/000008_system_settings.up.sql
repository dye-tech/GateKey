-- System settings table for configurable options
CREATE TABLE system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default settings
INSERT INTO system_settings (key, value, description) VALUES
    ('session_duration_hours', '12', 'Session duration in hours'),
    ('secure_cookies', 'true', 'Use secure cookies (HTTPS only)'),
    ('vpn_cert_validity_hours', '24', 'VPN certificate validity in hours');

-- Trigger to update timestamp
CREATE TRIGGER update_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
