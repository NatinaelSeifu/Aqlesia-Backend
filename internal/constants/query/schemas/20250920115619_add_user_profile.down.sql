-- Drop the marriage_status constraint
ALTER TABLE users 
  DROP CONSTRAINT IF EXISTS users_marriage_status_check;

-- Drop the profile columns
ALTER TABLE users 
  DROP COLUMN IF EXISTS job_title,
  DROP COLUMN IF EXISTS education,
  DROP COLUMN IF EXISTS marriage_status,
  DROP COLUMN IF EXISTS childrens_name;
