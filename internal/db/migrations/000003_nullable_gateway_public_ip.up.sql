-- Make public_ip nullable for gateways
-- The public IP is not known until the gateway connects and reports it
ALTER TABLE gateways ALTER COLUMN public_ip DROP NOT NULL;
