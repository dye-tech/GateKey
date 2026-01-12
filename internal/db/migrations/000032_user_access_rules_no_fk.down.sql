-- Revert: restore foreign key constraint on user_access_rules.user_id
-- Note: This will fail if there are any local user assignments

-- Drop the primary key constraint
ALTER TABLE user_access_rules DROP CONSTRAINT user_access_rules_pkey;

-- Change user_id back to UUID with foreign key
ALTER TABLE user_access_rules ALTER COLUMN user_id TYPE UUID USING user_id::UUID;

-- Re-add the primary key constraint
ALTER TABLE user_access_rules ADD PRIMARY KEY (user_id, access_rule_id);

-- Re-add foreign key constraint
ALTER TABLE user_access_rules ADD CONSTRAINT user_access_rules_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Drop and recreate index
DROP INDEX IF EXISTS idx_user_access_rules_user;
CREATE INDEX idx_user_access_rules_user ON user_access_rules(user_id);
