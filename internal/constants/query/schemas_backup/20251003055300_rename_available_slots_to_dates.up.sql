-- Rename available_appointment_slots table to available_dates
ALTER TABLE available_appointment_slots RENAME TO available_dates;

-- Update indexes with new names
DROP INDEX IF EXISTS idx_available_slots_date;
DROP INDEX IF EXISTS idx_available_slots_active;
DROP INDEX IF EXISTS idx_available_slots_availability;

CREATE INDEX idx_available_dates_date ON available_dates(slot_date);
CREATE INDEX idx_available_dates_active ON available_dates(is_active);
CREATE INDEX idx_available_dates_availability ON available_dates(slot_date, is_active) WHERE is_active = true;

-- CockroachDB: Remove PL/pgSQL functions and triggers
-- Drop any existing triggers and functions from PostgreSQL
DROP TRIGGER IF EXISTS update_available_slots_updated_at_trigger ON available_dates;
DROP FUNCTION IF EXISTS update_available_slots_updated_at();
DROP TRIGGER IF EXISTS update_slot_booking_count_trigger ON appointments;
DROP FUNCTION IF EXISTS update_slot_booking_count();

-- updated_at and current_bookings will be managed in application code
