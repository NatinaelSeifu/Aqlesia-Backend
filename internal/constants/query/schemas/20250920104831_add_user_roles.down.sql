-- Drop the role index
DROP INDEX IF EXISTS idx_users_role;

-- Drop the role constraint
ALTER TABLE users 
  DROP CONSTRAINT IF EXISTS users_role_check;

-- Drop the role column
ALTER TABLE users 
  DROP COLUMN IF EXISTS role;
