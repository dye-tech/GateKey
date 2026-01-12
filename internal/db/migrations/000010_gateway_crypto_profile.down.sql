-- Remove crypto_profile column from gateways table
ALTER TABLE gateways DROP COLUMN IF EXISTS crypto_profile;
