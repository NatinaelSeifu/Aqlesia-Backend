-- Drop appointments table and related objects
DROP TRIGGER IF EXISTS update_appointments_updated_at_trigger ON appointments;
DROP FUNCTION IF EXISTS update_appointments_updated_at();
DROP INDEX IF EXISTS idx_appointments_user_date_active;
DROP INDEX IF EXISTS idx_appointments_deleted_at;
DROP INDEX IF EXISTS idx_appointments_status;
DROP INDEX IF EXISTS idx_appointments_date;
DROP INDEX IF EXISTS idx_appointments_user_id;
DROP TABLE IF EXISTS appointments;
