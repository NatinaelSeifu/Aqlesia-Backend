-- Add missing partner_name column to users table
ALTER TABLE users ADD COLUMN partner_name varchar;
