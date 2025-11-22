-- 007_add_profile_image_url.down.sql
-- Purpose: Drop profile_image_url column from users

BEGIN;

ALTER TABLE users
  DROP COLUMN IF EXISTS profile_image_url;

COMMIT;