-- Drop indexes
DROP INDEX IF EXISTS idx_users_pending;
DROP INDEX IF EXISTS idx_users_status;

-- Remove status column from users table
ALTER TABLE users DROP COLUMN status;
