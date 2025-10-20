-- Initial database schema setup
-- This consolidates the initial user management system

-- Create users table with all current fields
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar NOT NULL DEFAULT '',
    lastname varchar NOT NULL DEFAULT '',
    phone_number varchar NOT NULL DEFAULT '',
    password varchar NOT NULL DEFAULT '',
    telegram_id varchar,
    role varchar NOT NULL DEFAULT 'user',
    status varchar(20) NOT NULL DEFAULT 'PENDING',
    job_title varchar,
    education varchar,
    marriage_status varchar,
    partner_name varchar,
    childrens_name text[],
    telegram_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL default now(),
    updated_at timestamptz NOT NULL default now(),
    deleted_at timestamptz
);

-- Add constraints
ALTER TABLE users 
  ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'manager', 'user'));

ALTER TABLE users 
  ADD CONSTRAINT users_marriage_status_check CHECK (marriage_status IS NULL OR marriage_status IN ('single', 'married', 'divorced', 'widowed'));

-- Create indexes for users table
CREATE UNIQUE INDEX users_phone_deleted_at_key ON users(phone_number, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_pending ON users(status) WHERE status = 'PENDING' AND deleted_at IS NULL;
