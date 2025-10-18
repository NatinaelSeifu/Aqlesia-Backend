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

	// Generate reset token
	token, tokenHash, err := pr.generateResetToken()
	if err != nil {
		pr.logger.Error(ctx, "failed to generate reset token", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to generate reset token")
	}

	// Store token in database
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err = pr.userStorage.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt)
	if err != nil {
		pr.logger.Error(ctx, "failed to store reset token", zap.Error(err))
		return nil, errors.ErrWriteError.New("failed to store reset token")
	}

	// Send reset link via Telegram
	err = pr.telegramBot.SendResetLink(ctx, user.TelegramID.String, token)
	if err != nil {
		pr.logger.Error(ctx, "failed to send reset link via Telegram", 
			zap.Error(err), 
			zap.String("user_id", user.ID.String()))
		// Don't return error to user - they shouldn't know the exact failure
	} else {
		pr.logger.Info(ctx, "sent password reset link via Telegram", 
			zap.String("user_id", user.ID.String()))
	}

	return &dto.ForgotPasswordResponse{
		Message: "If your account exists and has a linked Telegram, you will receive a password reset link.",
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
	tokenHash := pr.hashToken(req.Token)

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
