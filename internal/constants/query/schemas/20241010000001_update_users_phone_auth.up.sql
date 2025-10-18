-- Remove old columns and add new ones for phone-based registration
ALTER TABLE users 
  DROP COLUMN first_name,
  DROP COLUMN middle_name,
  DROP COLUMN last_name,
  DROP COLUMN email,
  DROP COLUMN status;

-- Add new columns
ALTER TABLE users 
  ADD COLUMN name varchar NOT NULL DEFAULT '',
  ADD COLUMN lastname varchar NOT NULL DEFAULT '',
  ADD COLUMN phone_number varchar NOT NULL DEFAULT '',
  ADD COLUMN password varchar NOT NULL DEFAULT '',
  ADD COLUMN telegram_id varchar;

-- Add unique constraint on phone_number
ALTER TABLE users 
  ADD CONSTRAINT users_phone_number_unique UNIQUE (phone_number);

-- Drop the old email index
DROP INDEX IF EXISTS users_email_deleted_at_key;

-- Create new index for phone_number with soft delete support
CREATE UNIQUE INDEX users_phone_deleted_at_key ON users(phone_number, deleted_at) WHERE deleted_at IS NULL;
