-- Align networks.cidr_v6 to the CIDR type intended by migration 000043.
--
-- Databases that applied an earlier revision of 000043 were left with
-- networks.cidr_v6 as TEXT and without the idx_networks_cidr_v6 GiST index
-- (000042 created the column as TEXT and the earlier 000043 did not coerce it).
-- Fresh installs already have it as CIDR with the index, so every statement
-- below is written to be idempotent and a no-op there.

-- GiST inet_ops indexing on inet/cidr requires btree_gist.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Coerce only if the column is not already CIDR. The cast is safe: cidr_v6
-- holds a single IPv6 CIDR (or NULL).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'networks'
          AND column_name = 'cidr_v6'
          AND udt_name <> 'cidr'
    ) THEN
        ALTER TABLE networks ALTER COLUMN cidr_v6 TYPE CIDR USING cidr_v6::CIDR;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_networks_cidr_v6
    ON networks USING gist (cidr_v6 inet_ops) WHERE cidr_v6 IS NOT NULL;
