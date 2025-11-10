-- 004_add_engaged_marriage_status.down.sql
-- Purpose: Revert 'engaged' from marriage_status check constraint

BEGIN;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_marriage_status_check;
ALTER TABLE users 
  ADD CONSTRAINT users_marriage_status_check 
  CHECK (marriage_status IS NULL OR marriage_status IN ('single', 'married', 'divorced', 'widowed'));

COMMIT;
