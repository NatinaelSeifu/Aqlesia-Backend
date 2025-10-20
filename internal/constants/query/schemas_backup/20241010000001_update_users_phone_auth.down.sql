-- Revert the users table changes
DROP INDEX IF EXISTS users_phone_deleted_at_key;
DROP INDEX IF EXISTS users_phone_number_unique;

ALTER TABLE users 
  DROP COLUMN name,
  DROP COLUMN lastname,  
  DROP COLUMN phone_number,
  DROP COLUMN password,
  DROP COLUMN telegram_id;

ALTER TABLE users 
  ADD COLUMN first_name varchar NOT NULL,
  ADD COLUMN middle_name varchar NOT NULL,
  ADD COLUMN last_name varchar NOT NULL,
  ADD COLUMN email varchar NOT NULL,
  ADD COLUMN status status NOT NULL DEFAULT 'ACTIVE';

CREATE UNIQUE INDEX users_email_deleted_at_key ON users(email,deleted_at) WHERE deleted_at IS NULL;
