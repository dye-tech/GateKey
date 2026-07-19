-- Remove foreign key constraint on user_access_rules.user_id
-- This allows assigning access rules to both SSO users (users table) and local users (local_users table)

-- Drop the foreign key to users(id) first: user_id is being widened to
-- VARCHAR to hold both SSO (users.id UUID) and local (local_users.id) ids, and
-- an FK to a uuid column cannot survive the type change.
ALTER TABLE user_access_rules DROP CONSTRAINT IF EXISTS user_access_rules_user_id_fkey;

-- First, drop the primary key constraint (it depends on the column)
ALTER TABLE user_access_rules DROP CONSTRAINT user_access_rules_pkey;

-- Change user_id from UUID to VARCHAR to support both user types
ALTER TABLE user_access_rules ALTER COLUMN user_id TYPE VARCHAR(255);

-- Re-add the primary key constraint
ALTER TABLE user_access_rules ADD PRIMARY KEY (user_id, access_rule_id);

-- Drop the old index that was created in migration 000002
DROP INDEX IF EXISTS idx_user_access_rules_user;

-- Re-create the index on the new column type
CREATE INDEX idx_user_access_rules_user ON user_access_rules(user_id);
