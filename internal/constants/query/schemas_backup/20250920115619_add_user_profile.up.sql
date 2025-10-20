-- Add profile fields to users table
ALTER TABLE users 
  ADD COLUMN job_title varchar,
  ADD COLUMN education varchar,
  ADD COLUMN marriage_status varchar,
  ADD COLUMN childrens_name text[];

-- Add constraint for marriage_status
ALTER TABLE users 
  ADD CONSTRAINT users_marriage_status_check CHECK (marriage_status IS NULL OR marriage_status IN ('single', 'married', 'divorced', 'widowed'));
