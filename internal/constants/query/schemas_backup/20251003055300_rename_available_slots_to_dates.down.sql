-- Revert available_dates table back to available_appointment_slots
ALTER TABLE available_dates RENAME TO available_appointment_slots;

-- Revert indexes with old names
DROP INDEX IF EXISTS idx_available_dates_date;
DROP INDEX IF EXISTS idx_available_dates_active;
DROP INDEX IF EXISTS idx_available_dates_availability;

CREATE INDEX idx_available_slots_date ON available_appointment_slots(slot_date);
CREATE INDEX idx_available_slots_active ON available_appointment_slots(is_active);
CREATE INDEX idx_available_slots_availability ON available_appointment_slots(slot_date, is_active) WHERE is_active = true;

-- Revert trigger references
DROP TRIGGER IF EXISTS update_available_dates_updated_at_trigger ON available_appointment_slots;
DROP FUNCTION IF EXISTS update_available_dates_updated_at();

-- Recreate trigger with old name
CREATE OR REPLACE FUNCTION update_available_slots_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_available_slots_updated_at_trigger
    BEFORE UPDATE ON available_appointment_slots
    FOR EACH ROW
    EXECUTE FUNCTION update_available_slots_updated_at();

-- Revert the booking count trigger to reference old table name
DROP TRIGGER IF EXISTS update_slot_booking_count_trigger ON appointments;
DROP FUNCTION IF EXISTS update_slot_booking_count();

CREATE OR REPLACE FUNCTION update_slot_booking_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        -- Update count for new/updated appointment
        UPDATE available_appointment_slots 
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
        UPDATE available_appointment_slots 
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
