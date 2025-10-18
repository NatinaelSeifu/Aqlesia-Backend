-- Add role column to users table
ALTER TABLE users 
  ADD COLUMN role varchar NOT NULL DEFAULT 'user';

-- Add constraint to ensure valid roles
ALTER TABLE users 
  ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'manager', 'user'));

-- Create index for role-based queries
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;

-- Update existing users to have 'user' role (if any exist)
UPDATE users SET role = 'user' WHERE role IS NULL OR role = '';
