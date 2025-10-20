-- Drop cleanup function
DROP FUNCTION IF EXISTS cleanup_expired_reset_tokens();

-- Remove telegram_verified column from users table
ALTER TABLE users DROP COLUMN IF EXISTS telegram_verified;

-- Drop indexes
DROP INDEX IF EXISTS idx_password_reset_tokens_token_hash;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;

-- Drop password reset tokens table
DROP TABLE IF EXISTS password_reset_tokens;
