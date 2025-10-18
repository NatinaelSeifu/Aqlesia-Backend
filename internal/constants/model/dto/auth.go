package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/dongri/phonenumber"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	// PhoneNumber is the Ethiopian phone number (+251...)
	PhoneNumber string `json:"phone_number"`
	// Password is the user's password
	Password string `json:"password"`
}

func (l LoginRequest) Validate() error {
	return validation.ValidateStruct(&l,
		validation.Field(&l.PhoneNumber, 
			validation.Required.Error("phone number is required"),
			validation.By(l.validateEthiopianPhoneNumber)),
		validation.Field(&l.Password, 
			validation.Required.Error("password is required")),
	)
}

// validateEthiopianPhoneNumber validates Ethiopian phone numbers for login
func (l LoginRequest) validateEthiopianPhoneNumber(value interface{}) error {
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
func (l LoginRequest) NormalizePhoneNumber() string {
	return phonenumber.Parse(l.PhoneNumber, "ET")
}

// AuthResponse represents the authentication response with tokens
type AuthResponse struct {
	// AccessToken is the JWT access token
	AccessToken string `json:"access_token"`
	// RefreshToken is the JWT refresh token
	RefreshToken string `json:"refresh_token"`
	// TokenType is the type of token (Bearer)
	TokenType string `json:"token_type"`
	// ExpiresIn is the token expiration time in seconds
	ExpiresIn int64 `json:"expires_in"`
	// User is the authenticated user information
	User *User `json:"user"`
}

// RefreshTokenRequest represents the refresh token request payload
type RefreshTokenRequest struct {
	// RefreshToken is the JWT refresh token
	RefreshToken string `json:"refresh_token"`
}

func (r RefreshTokenRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RefreshToken, validation.Required.Error("refresh token is required")),
	)
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID      string `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	TokenType   string `json:"token_type"` // "access" or "refresh"
	ExpiresAt   int64  `json:"exp"`
	IssuedAt    int64  `json:"iat"`
}

// AuthContext represents the authentication context stored in request context
type AuthContext struct {
	UserID      string
	PhoneNumber string
	Claims      *JWTClaims
}
