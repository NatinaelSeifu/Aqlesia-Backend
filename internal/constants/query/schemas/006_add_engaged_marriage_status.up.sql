-- 004_add_engaged_marriage_status.up.sql
-- Purpose: Allow 'engaged' as a valid marriage_status value

BEGIN;

-- Drop existing constraint if present, then re-add including 'engaged'
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_marriage_status_check;
ALTER TABLE users 
  ADD CONSTRAINT users_marriage_status_check 
  CHECK (marriage_status IS NULL OR marriage_status IN ('single', 'married', 'divorced', 'widowed', 'engaged'));

COMMIT;
