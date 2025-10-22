-- Add missing partner_name column to users table (if it doesn't exist)
-- Note: CockroachDB doesn't support conditional DDL, so we use IF NOT EXISTS
ALTER TABLE users ADD COLUMN IF NOT EXISTS partner_name varchar;
