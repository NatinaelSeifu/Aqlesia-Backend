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
        UNIQUE (user_id, communion_date) DEFERRABLE INITIALLY DEFERRED
);

-- Add indexes
CREATE INDEX idx_communion_user_id ON communion(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_communion_status ON communion(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_communion_approved_by ON communion(approved_by_user_id) WHERE deleted_at IS NULL;

-- Add trigger to update updated_at
CREATE OR REPLACE FUNCTION update_communion_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language plpgsql;

CREATE TRIGGER trigger_communion_updated_at
    BEFORE UPDATE ON communion
    FOR EACH ROW
    EXECUTE FUNCTION update_communion_updated_at();
