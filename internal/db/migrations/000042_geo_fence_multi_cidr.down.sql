-- Remove IPv6 columns
ALTER TABLE access_rules DROP COLUMN IF EXISTS value_v6;
ALTER TABLE networks DROP COLUMN IF EXISTS cidr_v6;
ALTER TABLE geo_fence_rules DROP COLUMN IF EXISTS ipv6_range;

-- Revert to single CIDR (note: this will fail if any row has multiple CIDRs)
ALTER TABLE geo_fence_rules ALTER COLUMN ip_range TYPE CIDR USING ip_range::CIDR;

-- Recreate the GiST index
CREATE INDEX idx_geo_fence_rules_ip ON geo_fence_rules USING gist (ip_range inet_ops);
