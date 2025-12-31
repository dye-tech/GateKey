-- Revert: make public_ip NOT NULL again
-- This may fail if there are NULL values
ALTER TABLE gateways ALTER COLUMN public_ip SET NOT NULL;
