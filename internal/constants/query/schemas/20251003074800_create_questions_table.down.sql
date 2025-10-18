-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_questions_updated_at ON questions;

-- Drop function
DROP FUNCTION IF EXISTS update_questions_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_questions_created_at;
DROP INDEX IF EXISTS idx_questions_status;
DROP INDEX IF EXISTS idx_questions_user_id;

-- Drop questions table
DROP TABLE IF EXISTS questions;
