-- Add attribute/claim mapping fields to OIDC and SAML providers
-- These allow customization of how IdP attributes map to GateKey user fields

-- Add claim mappings to OIDC providers with sensible defaults
ALTER TABLE oidc_providers
ADD COLUMN IF NOT EXISTS claim_mappings JSONB DEFAULT '{
    "email": "email",
    "name": "name",
    "given_name": "given_name",
    "family_name": "family_name",
    "groups": "groups"
}'::jsonb;

-- Add attribute mappings to SAML providers with sensible defaults
ALTER TABLE saml_providers
ADD COLUMN IF NOT EXISTS attribute_mappings JSONB DEFAULT '{
    "email": "email",
    "name": "displayName",
    "given_name": "givenName",
    "family_name": "sn",
    "groups": "groups"
}'::jsonb;

-- Add comments for documentation
COMMENT ON COLUMN oidc_providers.claim_mappings IS 'Maps GateKey fields to OIDC claim names from the IdP';
COMMENT ON COLUMN saml_providers.attribute_mappings IS 'Maps GateKey fields to SAML attribute names from the IdP';
