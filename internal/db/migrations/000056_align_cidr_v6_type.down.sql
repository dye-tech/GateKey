-- Intentional no-op.
--
-- 000056 only aligns networks.cidr_v6 to the CIDR type that migration 000043
-- already establishes on fresh installs. Reverting the type here would diverge
-- previously-migrated databases from fresh ones, and the idx_networks_cidr_v6
-- index is owned by 000043's down migration. There is nothing to roll back.
SELECT 1;
