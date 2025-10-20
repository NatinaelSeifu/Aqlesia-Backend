-- Restore the unique constraint (note: this will prevent deleted users from re-registering)
-- This is kept for rollback purposes, but the partial index is the better solution
ALTER TABLE users ADD CONSTRAINT users_phone_number_unique UNIQUE (phone_number);
