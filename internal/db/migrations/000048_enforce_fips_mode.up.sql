-- Add enforce_fips_mode column to gateways table
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS enforce_fips_mode BOOLEAN DEFAULT false NOT NULL;

-- Add enforce_fips_mode column to mesh_hubs table
ALTER TABLE mesh_hubs ADD COLUMN IF NOT EXISTS enforce_fips_mode BOOLEAN DEFAULT false NOT NULL;

-- Add enforce_fips_mode column to mesh_gateways (spokes) table
ALTER TABLE mesh_gateways ADD COLUMN IF NOT EXISTS enforce_fips_mode BOOLEAN DEFAULT false NOT NULL;
