package password_reset

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/internal/telegram"
	"aqlesia/platform/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type passwordReset struct {
	userStorage     storage.User
	telegramBot     *telegram.BotService
	logger          logger.Logger
	tokenPepper     string
	appBaseURL      string
}

func Init(logger logger.Logger, userStorage storage.User, telegramBot *telegram.BotService, tokenPepper string, appBaseURL string) module.PasswordReset {
	return &passwordReset{
		userStorage: userStorage,
		telegramBot: telegramBot,
		logger:      logger,
		tokenPepper: tokenPepper,
		appBaseURL:  appBaseURL,
	}
}

// ForgotPassword initiates the password reset process
func (pr *passwordReset) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	if err := req.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		pr.logger.Error(ctx, "validation failed", zap.Error(err))
		return nil, err
	}

	normalizedPhone := req.NormalizePhoneNumber()

	// Check if user exists and has verified Telegram
	user, err := pr.userStorage.GetUserByPhoneWithPassword(ctx, normalizedPhone)
	if err != nil {
		// Don't reveal whether user exists - always return success
		pr.logger.Info(ctx, "forgot password request for non-existent user", 
			zap.String("phone", normalizedPhone))
		return &dto.ForgotPasswordResponse{
			Message: "If your account exists and has a linked Telegram, you will receive a password reset link.",
		}, nil
	}

	// Check if user has verified Telegram
	if !user.TelegramID.Valid || !user.TelegramVerified {
		pr.logger.Info(ctx, "forgot password request for user without verified Telegram", 
			zap.String("user_id", user.ID.String()))
		return &dto.ForgotPasswordResponse{
			Message: "If your account exists and has a linked Telegram, you will receive a password reset link.",
		}, nil
	}

	// Generate OTP for verification
	otp, otpHash, err := pr.generateOTP()
	if err != nil {
		pr.logger.Error(ctx, "failed to generate OTP", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to generate OTP")
	}

	// Store OTP in database
	otpExpiresAt := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes
	_, err = pr.userStorage.CreatePasswordResetOTP(ctx, user.ID, otpHash, otpExpiresAt)
	if err != nil {
		pr.logger.Error(ctx, "failed to store OTP", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to store OTP")
	}

	// Send OTP via Telegram
	err = pr.telegramBot.SendOTP(ctx, user.TelegramID.String, otp)
	if err != nil {
		pr.logger.Error(ctx, "failed to send OTP via Telegram", 
			zap.Error(err), 
			zap.String("user_id", user.ID.String()))
		// Don't return error to user - they shouldn't know the exact failure
	} else {
		pr.logger.Info(ctx, "sent password reset OTP via Telegram", 
			zap.String("user_id", user.ID.String()))
	}

	return &dto.ForgotPasswordResponse{
		Message: "If your account exists and has a linked Telegram, you will receive a 6-digit OTP code.",
	}, nil
}

// VerifyOTP verifies the OTP and returns a reset token if valid
func (pr *passwordReset) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (*dto.VerifyOTPResponse, error) {
	if err := req.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		pr.logger.Error(ctx, "validation failed", zap.Error(err))
		return nil, err
	}

	normalizedPhone := req.NormalizePhoneNumber()

	// Get user by phone number
	user, err := pr.userStorage.GetUserByPhoneWithPassword(ctx, normalizedPhone)
	if err != nil {
		pr.logger.Warn(ctx, "OTP verification for non-existent user", 
			zap.String("phone", normalizedPhone))
		return &dto.VerifyOTPResponse{
			Valid:   false,
			Message: "Invalid OTP or phone number",
		}, nil
	}

	// Hash the provided OTP
	otpHash := pr.hashOTP(req.OTP)

	// Get and validate OTP
	resetOTP, err := pr.userStorage.GetValidPasswordResetOTP(ctx, user.ID, otpHash)
	if err != nil {
		pr.logger.Warn(ctx, "invalid or expired OTP used", 
			zap.String("user_id", user.ID.String()),
			zap.String("otp_hash", otpHash))
		return &dto.VerifyOTPResponse{
			Valid:   false,
			Message: "Invalid or expired OTP",
		}, nil
	}

	// Generate reset token for password reset
	resetToken, tokenHash, err := pr.generateResetToken()
	if err != nil {
		pr.logger.Error(ctx, "failed to generate reset token", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to generate reset token")
	}

	// Store reset token (expires in 10 minutes)
	tokenExpiresAt := time.Now().Add(10 * time.Minute)
	_, err = pr.userStorage.CreatePasswordResetToken(ctx, user.ID, tokenHash, tokenExpiresAt)
	if err != nil {
		pr.logger.Error(ctx, "failed to store reset token", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to store reset token")
	}

	// Mark OTP as used
	err = pr.userStorage.MarkPasswordResetOTPUsed(ctx, resetOTP.ID)
	if err != nil {
		pr.logger.Error(ctx, "failed to mark OTP as used", 
			zap.Error(err), 
			zap.String("otp_id", resetOTP.ID.String()))
		// Continue anyway as token was generated
	}

	pr.logger.Info(ctx, "OTP verified successfully", 
		zap.String("user_id", user.ID.String()))

	return &dto.VerifyOTPResponse{
		Valid:      true,
		Message:    "OTP verified successfully. Use the reset token to set your new password.",
		ResetToken: &resetToken,
	}, nil
}

// ResetPassword completes the password reset process
func (pr *passwordReset) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error) {
	if err := req.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		pr.logger.Error(ctx, "validation failed", zap.Error(err))
		return nil, err
	}

	// Hash the provided token
	tokenHash := pr.hashToken(req.ResetToken)

	// Get and validate token
	resetToken, err := pr.userStorage.GetValidPasswordResetToken(ctx, tokenHash)
	if err != nil {
		pr.logger.Warn(ctx, "invalid or expired reset token used", zap.String("token_hash", tokenHash))
		return nil, errors.ErrInvalidUserInput.New("invalid or expired reset token")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		pr.logger.Error(ctx, "failed to hash password", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to process password")
	}

	// Update user password
	_, err = pr.userStorage.ResetUserPassword(ctx, string(hashedPassword), resetToken.UserID)
	if err != nil {
		pr.logger.Error(ctx, "failed to update user password", 
			zap.Error(err), 
			zap.String("user_id", resetToken.UserID.String()))
		return nil, errors.ErrWriteError.New("failed to update password")
	}

	// Mark token as used
	err = pr.userStorage.MarkPasswordResetTokenUsed(ctx, resetToken.ID)
	if err != nil {
		pr.logger.Error(ctx, "failed to mark reset token as used", 
			zap.Error(err), 
			zap.String("token_id", resetToken.ID.String()))
		// Don't return error as the password was already updated
	}

	pr.logger.Info(ctx, "password reset completed successfully", 
		zap.String("user_id", resetToken.UserID.String()))

	return &dto.ResetPasswordResponse{
		Message: "Password has been reset successfully. You can now login with your new password.",
	}, nil
}

// LinkTelegram generates a link code for Telegram account linking
func (pr *passwordReset) LinkTelegram(ctx context.Context, req dto.TelegramLinkRequest) (*dto.TelegramLinkResponse, error) {
	if err := req.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		pr.logger.Error(ctx, "validation failed", zap.Error(err))
		return nil, err
	}

	normalizedPhone := req.NormalizePhoneNumber()

	// Check if user exists
	_, err := pr.userStorage.GetUserByPhoneWithPassword(ctx, normalizedPhone)
	if err != nil {
		pr.logger.Warn(ctx, "telegram link request for non-existent user", 
			zap.String("phone", normalizedPhone))
		return nil, errors.ErrInvalidUserInput.New("user not found")
	}

	// Generate link code
	linkCode, telegramURL, err := pr.telegramBot.GenerateLinkCode(normalizedPhone)
	if err != nil {
		pr.logger.Error(ctx, "failed to generate link code", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to generate link code")
	}

	pr.logger.Info(ctx, "generated Telegram link code", 
		zap.String("phone", normalizedPhone))

	return &dto.TelegramLinkResponse{
		LinkCode:    linkCode,
		TelegramURL: telegramURL,
		Message:     "Click the Telegram link or manually start a chat with the bot using the provided code. The link will expire in 5 minutes.",
	}, nil
}

// CheckTelegramVerification checks if a user has verified Telegram without sending OTP
func (pr *passwordReset) CheckTelegramVerification(ctx context.Context, req dto.TelegramLinkRequest) (map[string]interface{}, error) {
	if err := req.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		pr.logger.Error(ctx, "validation failed", zap.Error(err))
		return nil, err
	}

	normalizedPhone := req.NormalizePhoneNumber()

	// Check if user exists and has verified Telegram
	user, err := pr.userStorage.GetUserByPhoneWithPassword(ctx, normalizedPhone)
	if err != nil {
		// Don't reveal whether user exists - return neutral message
		pr.logger.Info(ctx, "telegram verification check for non-existent user",
			zap.String("phone", normalizedPhone))
		return map[string]interface{}{
			"verified": false,
			"message":  "Please ensure your account exists and Telegram is properly linked.",
		}, nil
	}

	// Check telegram verification status
	if !user.TelegramID.Valid || !user.TelegramVerified {
		pr.logger.Info(ctx, "telegram verification check for user without verified Telegram",
			zap.String("user_id", user.ID.String()))
		return map[string]interface{}{
			"verified": false,
			"message":  "Telegram account not linked or verified. Please complete the linking process.",
		}, nil
	}

	pr.logger.Info(ctx, "telegram verification check successful",
		zap.String("user_id", user.ID.String()))

	return map[string]interface{}{
		"verified": true,
		"message":  "Telegram account is successfully linked and verified. You can receive OTP codes for password reset.",
	}, nil
}

// CleanupExpiredTokens removes expired password reset tokens
func (pr *passwordReset) CleanupExpiredTokens(ctx context.Context) error {
	err := pr.userStorage.CleanupExpiredResetTokens(ctx)
	if err != nil {
		pr.logger.Error(ctx, "failed to cleanup expired reset tokens", zap.Error(err))
		return err
	}

	pr.logger.Debug(ctx, "cleaned up expired reset tokens")
	return nil
}

// generateResetToken creates a secure random token and its hash
func (pr *passwordReset) generateResetToken() (token string, tokenHash string, err error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	token = base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash = pr.hashToken(token)

	return token, tokenHash, nil
}

// hashToken creates HMAC-SHA256 hash of the token
func (pr *passwordReset) hashToken(token string) string {
	mac := hmac.New(sha256.New, []byte(pr.tokenPepper))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// generateOTP creates a 6-digit OTP and its hash
func (pr *passwordReset) generateOTP() (otp string, otpHash string, err error) {
	// Generate random 6-digit OTP
	otpBytes := make([]byte, 3)
	if _, err = rand.Read(otpBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random OTP: %w", err)
	}

	// Convert to 6-digit string
	otpInt := int(otpBytes[0])<<16 | int(otpBytes[1])<<8 | int(otpBytes[2])
	otp = fmt.Sprintf("%06d", otpInt%1000000)
	otpHash = pr.hashOTP(otp)

	return otp, otpHash, nil
}

// hashOTP creates HMAC-SHA256 hash of the OTP
func (pr *passwordReset) hashOTP(otp string) string {
	mac := hmac.New(sha256.New, []byte(pr.tokenPepper+"_otp"))
	mac.Write([]byte(otp))
	return hex.EncodeToString(mac.Sum(nil))
}
