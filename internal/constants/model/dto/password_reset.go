package dto

import (
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/dongri/phonenumber"
	"github.com/google/uuid"
)

// ForgotPasswordRequest represents the forgot password request payload
type ForgotPasswordRequest struct {
	// PhoneNumber is the Ethiopian phone number of the user
	PhoneNumber string `json:"phone_number"`
}

func (f ForgotPasswordRequest) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PhoneNumber,
			validation.Required.Error("phone number is required"),
			validation.By(f.validateEthiopianPhoneNumber)),
	)
}

// validateEthiopianPhoneNumber validates Ethiopian phone numbers for forgot password
func (f ForgotPasswordRequest) validateEthiopianPhoneNumber(value interface{}) error {
	phoneStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
	}

	// Parse and validate the phone number for Ethiopia (ET)
	parsedNumber := phonenumber.Parse(phoneStr, "ET")

	// Check if the parsed number is empty (invalid)
	if parsedNumber == "" {
		return validation.NewError("validation_phone_invalid", "phone number must be a valid Ethiopian number")
	}

	// Verify it's actually an Ethiopian number (should start with 251)
	if len(parsedNumber) < 12 || parsedNumber[:3] != "251" {
		return validation.NewError("validation_phone_not_ethiopian", "phone number must be a valid Ethiopian number")
	}

	return nil
}

// NormalizePhoneNumber returns the phone number in E.164 format
func (f ForgotPasswordRequest) NormalizePhoneNumber() string {
	return phonenumber.Parse(f.PhoneNumber, "ET")
}

// ForgotPasswordResponse represents the forgot password response
type ForgotPasswordResponse struct {
	// Message is a generic success message (doesn't reveal if user exists)
	Message string `json:"message"`
}

// VerifyOTPRequest represents the OTP verification request
type VerifyOTPRequest struct {
	// PhoneNumber is the Ethiopian phone number of the user
	PhoneNumber string `json:"phone_number"`
	// OTP is the one-time password received via Telegram
	OTP string `json:"otp"`
}

func (v VerifyOTPRequest) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.PhoneNumber,
			validation.Required.Error("phone number is required"),
			validation.By(v.validateEthiopianPhoneNumber)),
		validation.Field(&v.OTP,
			validation.Required.Error("OTP is required"),
			validation.Match(regexp.MustCompile(`^[0-9]{6}$`)).Error("OTP must be 6 digits")),
	)
}

// validateEthiopianPhoneNumber validates Ethiopian phone numbers for OTP verification
func (v VerifyOTPRequest) validateEthiopianPhoneNumber(value interface{}) error {
	phoneStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
	}

	// Parse and validate the phone number for Ethiopia (ET)
	parsedNumber := phonenumber.Parse(phoneStr, "ET")

	// Check if the parsed number is empty (invalid)
	if parsedNumber == "" {
		return validation.NewError("validation_phone_invalid", "phone number must be a valid Ethiopian number")
	}

	// Verify it's actually an Ethiopian number (should start with 251)
	if len(parsedNumber) < 12 || parsedNumber[:3] != "251" {
		return validation.NewError("validation_phone_not_ethiopian", "phone number must be a valid Ethiopian number")
	}

	return nil
}

// NormalizePhoneNumber returns the phone number in E.164 format
func (v VerifyOTPRequest) NormalizePhoneNumber() string {
	return phonenumber.Parse(v.PhoneNumber, "ET")
}

// VerifyOTPResponse represents the OTP verification response
type VerifyOTPResponse struct {
	// Valid indicates if the OTP was valid
	Valid bool `json:"valid"`
	// Message provides feedback to the user
	Message string `json:"message"`
	// ResetToken is provided only if OTP is valid (for password reset)
	ResetToken *string `json:"reset_token,omitempty"`
}

// ResetPasswordRequest represents the password reset request payload
type ResetPasswordRequest struct {
	// ResetToken is the token received after OTP verification
	ResetToken string `json:"reset_token"`
	// NewPassword is the new password to set
	NewPassword string `json:"new_password"`
}

func (r ResetPasswordRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ResetToken,
			validation.Required.Error("reset token is required")),
		validation.Field(&r.NewPassword,
			validation.Required.Error("new password is required"),
			validation.Length(6, 100).Error("new password must be at least 6 characters")),
	)
}

// ResetPasswordResponse represents the password reset response
type ResetPasswordResponse struct {
	// Message is the success message
	Message string `json:"message"`
}

// TelegramLinkRequest represents the request to link Telegram account
type TelegramLinkRequest struct {
	// PhoneNumber is the user's phone number
	PhoneNumber string `json:"phone_number"`
}

func (t TelegramLinkRequest) Validate() error {
	return validation.ValidateStruct(&t,
		validation.Field(&t.PhoneNumber,
			validation.Required.Error("phone number is required"),
			validation.By(t.validateEthiopianPhoneNumber)),
	)
}

// validateEthiopianPhoneNumber validates Ethiopian phone numbers for Telegram linking
func (t TelegramLinkRequest) validateEthiopianPhoneNumber(value interface{}) error {
	phoneStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
	}

	// Parse and validate the phone number for Ethiopia (ET)
	parsedNumber := phonenumber.Parse(phoneStr, "ET")

	// Check if the parsed number is empty (invalid)
	if parsedNumber == "" {
		return validation.NewError("validation_phone_invalid", "phone number must be a valid Ethiopian number")
	}

	// Verify it's actually an Ethiopian number (should start with 251)
	if len(parsedNumber) < 12 || parsedNumber[:3] != "251" {
		return validation.NewError("validation_phone_not_ethiopian", "phone number must be a valid Ethiopian number")
	}

	return nil
}

// NormalizePhoneNumber returns the phone number in E.164 format
func (t TelegramLinkRequest) NormalizePhoneNumber() string {
	return phonenumber.Parse(t.PhoneNumber, "ET")
}

// TelegramLinkResponse represents the response for Telegram linking
type TelegramLinkResponse struct {
	// LinkCode is the one-time code for linking Telegram account
	LinkCode string `json:"link_code"`
	// TelegramURL is the deep link URL to open in Telegram
	TelegramURL string `json:"telegram_url"`
	// Message provides instructions to the user
	Message string `json:"message"`
}

// PasswordResetToken represents a password reset token (for internal use)
type PasswordResetToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"` // Never serialize the hash
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// PasswordResetOTP represents a password reset OTP (for internal use)
type PasswordResetOTP struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	OTPHash   string    `json:"-"` // Never serialize the hash
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	Attempts  int       `json:"attempts"`
}
