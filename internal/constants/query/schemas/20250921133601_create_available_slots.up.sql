-- Create available_appointment_slots table
CREATE TABLE available_appointment_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_date DATE NOT NULL UNIQUE,
    max_capacity INTEGER NOT NULL DEFAULT 10,
    current_bookings INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for performance
CREATE INDEX idx_available_slots_date ON available_appointment_slots(slot_date);
CREATE INDEX idx_available_slots_active ON available_appointment_slots(is_active);
CREATE INDEX idx_available_slots_availability ON available_appointment_slots(slot_date, is_active) WHERE is_active = true;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_available_slots_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_available_slots_updated_at_trigger
    BEFORE UPDATE ON available_appointment_slots
    FOR EACH ROW
    EXECUTE FUNCTION update_available_slots_updated_at();

-- Function to automatically update current_bookings when appointments are created/cancelled
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

-- Trigger to automatically update slot booking counts
CREATE TRIGGER update_slot_booking_count_trigger
    AFTER INSERT OR UPDATE OR DELETE ON appointments
    FOR EACH ROW
    EXECUTE FUNCTION update_slot_booking_count();
