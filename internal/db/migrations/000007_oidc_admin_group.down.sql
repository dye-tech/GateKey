-- Remove admin_group column from oidc_providers and saml_providers
ALTER TABLE oidc_providers DROP COLUMN admin_group;
ALTER TABLE saml_providers DROP COLUMN admin_group;
