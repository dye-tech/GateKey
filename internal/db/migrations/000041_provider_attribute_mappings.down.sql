-- Remove attribute/claim mapping fields from OIDC and SAML providers

ALTER TABLE oidc_providers
DROP COLUMN IF EXISTS claim_mappings;

ALTER TABLE saml_providers
DROP COLUMN IF EXISTS attribute_mappings;
