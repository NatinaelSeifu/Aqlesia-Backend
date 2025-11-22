-- 007_add_profile_image_url.up.sql
-- Purpose: Add profile_image_url column to users

BEGIN;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS profile_image_url varchar;

COMMIT;