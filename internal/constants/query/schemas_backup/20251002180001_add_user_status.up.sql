-- Add status column back to users table for user approval workflow
-- CockroachDB compatible: using VARCHAR instead of ENUM
ALTER TABLE users 
  ADD COLUMN status varchar(20) NOT NULL DEFAULT 'PENDING';

-- Update existing users to ACTIVE status (they were already using the system)
UPDATE users SET status = 'ACTIVE';

-- Create index for status-based queries
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;

-- Create index for pending users (most common admin query)
CREATE INDEX idx_users_pending ON users(status) WHERE status = 'PENDING' AND deleted_at IS NULL;
