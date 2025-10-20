-- Remove the redundant unique constraint that prevents deleted users from re-registering
-- The partial unique index users_phone_deleted_at_key already handles uniqueness for active users
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_number_unique;
