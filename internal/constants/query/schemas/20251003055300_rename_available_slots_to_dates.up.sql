-- Rename available_appointment_slots table to available_dates
ALTER TABLE available_appointment_slots RENAME TO available_dates;

-- Update indexes with new names
DROP INDEX IF EXISTS idx_available_slots_date;
DROP INDEX IF EXISTS idx_available_slots_active;
DROP INDEX IF EXISTS idx_available_slots_availability;

CREATE INDEX idx_available_dates_date ON available_dates(slot_date);
CREATE INDEX idx_available_dates_active ON available_dates(is_active);
CREATE INDEX idx_available_dates_availability ON available_dates(slot_date, is_active) WHERE is_active = true;

-- Update trigger references
DROP TRIGGER IF EXISTS update_available_slots_updated_at_trigger ON available_dates;
DROP FUNCTION IF EXISTS update_available_slots_updated_at();

-- Recreate trigger with new name
CREATE OR REPLACE FUNCTION update_available_dates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_available_dates_updated_at_trigger
    BEFORE UPDATE ON available_dates
    FOR EACH ROW
    EXECUTE FUNCTION update_available_dates_updated_at();

-- Update the booking count trigger to reference new table name
DROP TRIGGER IF EXISTS update_slot_booking_count_trigger ON appointments;
DROP FUNCTION IF EXISTS update_slot_booking_count();

CREATE OR REPLACE FUNCTION update_slot_booking_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        -- Update count for new/updated appointment
        UPDATE available_dates 
        SET current_bookings = (
            SELECT COUNT(*) 
            FROM appointments 
            WHERE appointment_date = NEW.appointment_date 
              AND status IN ('pending', 'completed') 
              AND deleted_at IS NULL
        )
        WHERE slot_date = NEW.appointment_date;
        
        RETURN NEW;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        -- Update count for deleted appointment
        UPDATE available_dates 
        SET current_bookings = (
            SELECT COUNT(*) 
            FROM appointments 
            WHERE appointment_date = OLD.appointment_date 
              AND status IN ('pending', 'completed') 
              AND deleted_at IS NULL
        )
        WHERE slot_date = OLD.appointment_date;
        
        RETURN OLD;
    END IF;
    
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Recreate trigger
CREATE TRIGGER update_slot_booking_count_trigger
    AFTER INSERT OR UPDATE OR DELETE ON appointments
    FOR EACH ROW
    EXECUTE FUNCTION update_slot_booking_count();
