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

-- CockroachDB: Removed PL/pgSQL functions and triggers
-- updated_at and current_bookings will be managed in application code
-- Booking count updates will be handled in Go transactions
