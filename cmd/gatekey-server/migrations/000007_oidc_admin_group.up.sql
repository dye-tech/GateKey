-- Add admin_group column to oidc_providers and saml_providers
-- When a user logs in via OIDC/SAML and their groups contain this value, they will be made an admin

ALTER TABLE oidc_providers ADD COLUMN admin_group VARCHAR(255) DEFAULT NULL;
ALTER TABLE saml_providers ADD COLUMN admin_group VARCHAR(255) DEFAULT NULL;

COMMENT ON COLUMN oidc_providers.admin_group IS 'Group name that grants admin access when user is a member';
COMMENT ON COLUMN saml_providers.admin_group IS 'Group name that grants admin access when user is a member';
