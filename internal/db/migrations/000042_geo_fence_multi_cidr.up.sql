-- Change ip_range from CIDR (single) to TEXT (supports multiple comma-separated CIDRs)
-- Drop the GiST index first (only works with CIDR type)
DROP INDEX IF EXISTS idx_geo_fence_rules_ip;

-- Change column type from CIDR to TEXT
ALTER TABLE geo_fence_rules ALTER COLUMN ip_range TYPE TEXT USING ip_range::TEXT;

-- Add IPv6 support column for geo-fence rules
ALTER TABLE geo_fence_rules ADD COLUMN IF NOT EXISTS ipv6_range TEXT;

-- Add IPv6 support to networks table
ALTER TABLE networks ADD COLUMN IF NOT EXISTS cidr_v6 TEXT;

-- Add IPv6 support to access_rules for IP/CIDR type rules
ALTER TABLE access_rules ADD COLUMN IF NOT EXISTS value_v6 TEXT;
