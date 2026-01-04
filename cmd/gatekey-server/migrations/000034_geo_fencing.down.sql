-- Rollback geo-fencing migration

DROP TABLE IF EXISTS group_geo_fence_rules;
DROP TABLE IF EXISTS user_geo_fence_rules;
DROP TABLE IF EXISTS geo_fence_global;
DROP TABLE IF EXISTS geo_fence_rules;

DELETE FROM system_settings WHERE key IN ('geo_fencing_enabled', 'geo_fencing_enforce');
