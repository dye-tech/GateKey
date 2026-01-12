-- Make ip_range nullable in geo_fence_rules to allow IPv6-only rules
-- At least one of ip_range or ipv6_range must be provided (enforced by application)

ALTER TABLE geo_fence_rules ALTER COLUMN ip_range DROP NOT NULL;

-- Add a check constraint to ensure at least one range is provided
ALTER TABLE geo_fence_rules ADD CONSTRAINT geo_fence_rules_at_least_one_range
    CHECK (ip_range IS NOT NULL OR ipv6_range IS NOT NULL);
