-- Make hostname nullable so gateways can use either hostname OR IP address
ALTER TABLE gateways ALTER COLUMN hostname DROP NOT NULL;

-- Add a check constraint to ensure at least one of hostname or public_ip is provided
ALTER TABLE gateways ADD CONSTRAINT chk_gateway_address
    CHECK (hostname IS NOT NULL OR public_ip IS NOT NULL);
