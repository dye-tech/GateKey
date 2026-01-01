-- Remove the check constraint
ALTER TABLE gateways DROP CONSTRAINT IF EXISTS chk_gateway_address;

-- Set empty hostnames to a default value before making NOT NULL
UPDATE gateways SET hostname = COALESCE(host(public_ip), name) WHERE hostname IS NULL OR hostname = '';

-- Make hostname NOT NULL again
ALTER TABLE gateways ALTER COLUMN hostname SET NOT NULL;
