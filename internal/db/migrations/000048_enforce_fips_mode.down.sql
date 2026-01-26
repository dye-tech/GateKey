-- Remove enforce_fips_mode column from gateways table
ALTER TABLE gateways DROP COLUMN IF EXISTS enforce_fips_mode;

-- Remove enforce_fips_mode column from mesh_hubs table
ALTER TABLE mesh_hubs DROP COLUMN IF EXISTS enforce_fips_mode;

-- Remove enforce_fips_mode column from mesh_gateways (spokes) table
ALTER TABLE mesh_gateways DROP COLUMN IF EXISTS enforce_fips_mode;
