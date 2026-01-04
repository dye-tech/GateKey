-- Rollback local groups tables
DROP TRIGGER IF EXISTS local_groups_updated_at ON local_groups;
DROP FUNCTION IF EXISTS update_local_groups_updated_at();
DROP TABLE IF EXISTS local_group_members;
DROP TABLE IF EXISTS local_groups;
