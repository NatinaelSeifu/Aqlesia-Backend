-- Drop available_appointment_slots table and related objects
DROP TRIGGER IF EXISTS update_slot_booking_count_trigger ON appointments;
DROP FUNCTION IF EXISTS update_slot_booking_count();
DROP TRIGGER IF EXISTS update_available_slots_updated_at_trigger ON available_appointment_slots;
DROP FUNCTION IF EXISTS update_available_slots_updated_at();
DROP INDEX IF EXISTS idx_available_slots_availability;
DROP INDEX IF EXISTS idx_available_slots_active;
DROP INDEX IF EXISTS idx_available_slots_date;
DROP TABLE IF EXISTS available_appointment_slots;
