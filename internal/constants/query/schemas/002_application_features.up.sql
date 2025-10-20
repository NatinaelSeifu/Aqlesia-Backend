-- Application features schema
-- This includes appointments, available dates, communion, and questions functionality

-- Create available_dates table (renamed from available_appointment_slots)
CREATE TABLE available_dates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_date DATE NOT NULL UNIQUE,
    max_capacity INTEGER NOT NULL DEFAULT 10,
    current_bookings INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for available_dates
CREATE INDEX idx_available_dates_date ON available_dates(slot_date);
CREATE INDEX idx_available_dates_active ON available_dates(is_active);
CREATE INDEX idx_available_dates_availability ON available_dates(slot_date, is_active) WHERE is_active = true;

-- Create appointments table
CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    appointment_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for appointments
CREATE INDEX idx_appointments_user_id ON appointments(user_id);
CREATE INDEX idx_appointments_date ON appointments(appointment_date);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_appointments_deleted_at ON appointments(deleted_at);

-- Create unique constraint to prevent double bookings
CREATE UNIQUE INDEX idx_appointments_user_date_active 
ON appointments(user_id, appointment_date) 
WHERE deleted_at IS NULL AND status != 'cancelled';

-- Create communion table
CREATE TABLE communion (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    communion_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    approved_by_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT fk_communion_user_id 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_communion_approved_by 
        FOREIGN KEY (approved_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    
    -- Ensure a user can't have duplicate communion dates
    CONSTRAINT unique_user_communion_date 
        UNIQUE (user_id, communion_date)
);

-- Add indexes for communion
CREATE INDEX idx_communion_user_id ON communion(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_communion_status ON communion(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_communion_approved_by ON communion(approved_by_user_id) WHERE deleted_at IS NULL;

-- Create questions table
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    admin_response TEXT,
    responded_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for questions
CREATE INDEX idx_questions_user_id ON questions(user_id);
CREATE INDEX idx_questions_status ON questions(status);
CREATE INDEX idx_questions_created_at ON questions(created_at);
