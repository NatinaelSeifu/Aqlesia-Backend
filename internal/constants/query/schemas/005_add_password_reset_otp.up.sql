-- Add password reset OTP table for Telegram-based password resets
-- This enables OTP-based password reset flow via Telegram

CREATE TABLE password_reset_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_hash CHAR(64) NOT NULL,  -- hex encoded HMAC-SHA256 hash
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    attempts INT NOT NULL DEFAULT 0  -- track verification attempts
);

-- Create indexes for efficient lookups
CREATE INDEX idx_password_reset_otps_user_id ON password_reset_otps(user_id);
CREATE INDEX idx_password_reset_otps_expires_at ON password_reset_otps(expires_at);
CREATE INDEX idx_password_reset_otps_user_id_otp_hash ON password_reset_otps(user_id, otp_hash);
