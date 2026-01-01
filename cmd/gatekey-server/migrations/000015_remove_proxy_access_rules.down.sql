-- Recreate proxy_application_access_rules table

CREATE TABLE IF NOT EXISTS proxy_application_access_rules (
    proxy_app_id UUID NOT NULL REFERENCES proxy_applications(id) ON DELETE CASCADE,
    access_rule_id UUID NOT NULL REFERENCES access_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    PRIMARY KEY (proxy_app_id, access_rule_id)
);

CREATE INDEX IF NOT EXISTS idx_proxy_app_access_rules_app ON proxy_application_access_rules(proxy_app_id);
CREATE INDEX IF NOT EXISTS idx_proxy_app_access_rules_rule ON proxy_application_access_rules(access_rule_id);
