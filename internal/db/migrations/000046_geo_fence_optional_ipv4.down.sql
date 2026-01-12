-- Rollback: Make ip_range required again
-- First drop the check constraint
ALTER TABLE geo_fence_rules DROP CONSTRAINT IF EXISTS geo_fence_rules_at_least_one_range;

-- Then make ip_range NOT NULL (will fail if any rows have NULL ip_range)
ALTER TABLE geo_fence_rules ALTER COLUMN ip_range SET NOT NULL;
