-- Remove partner_name field
ALTER TABLE users 
DROP COLUMN IF EXISTS partner_name;
