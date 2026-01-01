-- Remove access rules linking from proxy applications
-- Access rules are only used for VPN gateway firewall rules, not proxy apps

DROP TABLE IF EXISTS proxy_application_access_rules;
