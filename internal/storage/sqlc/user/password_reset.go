package user

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/db"
	"aqlesia/internal/constants/model/dto"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UpdateTelegramInfo updates user's Telegram information
func (u *user) UpdateTelegramInfo(ctx context.Context, telegramID string, verified bool, phoneNumber string) (*dto.User, error) {
	telegramNullString := sql.NullString{String: telegramID, Valid: true}
	
	updatedUser, err := u.db.UpdateTelegramInfo(ctx, db.UpdateTelegramInfoParams{
		TelegramID:       telegramNullString,
		TelegramVerified: verified,
		PhoneNumber:      phoneNumber,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User not found for Telegram update", zap.String("phone", phoneNumber))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not update Telegram info")
		u.log.Error(ctx, "unable to update Telegram info", zap.Error(err), zap.String("phone", phoneNumber))
		return nil, err
	}

	// Convert optional fields for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if updatedUser.TelegramID.Valid {
		responseTelegramID = &updatedUser.TelegramID.String
	}
	if updatedUser.JobTitle.Valid {
		responseJobTitle = &updatedUser.JobTitle.String
	}
	if updatedUser.Education.Valid {
		responseEducation = &updatedUser.Education.String
	}
	if updatedUser.MarriageStatus.Valid {
		responseMarriageStatus = &updatedUser.MarriageStatus.String
	}
	if updatedUser.PartnerName.Valid {
		responsePartnerName = &updatedUser.PartnerName.String
	}

	u.log.Info(ctx, "Telegram info updated successfully", zap.String("phone", phoneNumber))

	return &dto.User{
		ID:             updatedUser.ID,
		Name:           updatedUser.Name,
		LastName:       updatedUser.Lastname,
		PhoneNumber:    updatedUser.PhoneNumber,
		Role:           updatedUser.Role,
		Status:         string(updatedUser.Status),
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  updatedUser.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      updatedUser.CreatedAt,
		UpdatedAt:      updatedUser.UpdatedAt,
	}, nil
}

// GetUserByTelegramID returns user by their Telegram ID
func (u *user) GetUserByTelegramID(ctx context.Context, telegramID string) (*db.User, error) {
	user, err := u.db.GetUserByTelegramID(ctx, sql.NullString{String: telegramID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User with this Telegram ID not found", zap.String("telegram_id", telegramID))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to get user by Telegram ID", zap.Error(err), zap.String("telegram_id", telegramID))
		return nil, err
	}

	return &user, nil
}

// CreatePasswordResetToken creates a new password reset token
func (u *user) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*dto.PasswordResetToken, error) {
	token, err := u.db.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not create password reset token")
		u.log.Error(ctx, "unable to create password reset token", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	return &dto.PasswordResetToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
		Used:      token.Used,
	}, nil
}

// GetValidPasswordResetToken retrieves a valid password reset token
func (u *user) GetValidPasswordResetToken(ctx context.Context, tokenHash string) (*dto.PasswordResetToken, error) {
	token, err := u.db.GetValidPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "token not found or expired")
			u.log.Info(ctx, "Password reset token not found or expired", zap.String("token_hash", tokenHash))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read password reset token")
		u.log.Error(ctx, "unable to get password reset token", zap.Error(err))
		return nil, err
	}

	return &dto.PasswordResetToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
		Used:      token.Used,
	}, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (u *user) MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	err := u.db.MarkPasswordResetTokenUsed(ctx, tokenID)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not mark token as used")
		u.log.Error(ctx, "unable to mark password reset token as used", zap.Error(err), zap.String("token_id", tokenID.String()))
		return err
	}

	u.log.Info(ctx, "Password reset token marked as used", zap.String("token_id", tokenID.String()))
	return nil
}

// ResetUserPassword resets a user's password
func (u *user) ResetUserPassword(ctx context.Context, hashedPassword string, userID uuid.UUID) (*dto.User, error) {
	updatedUser, err := u.db.ResetUserPassword(ctx, db.ResetUserPasswordParams{
		Password: hashedPassword,
		ID:       userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User not found for password reset", zap.String("user_id", userID.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not reset password")
		u.log.Error(ctx, "unable to reset user password", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	// Convert optional fields for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if updatedUser.TelegramID.Valid {
		responseTelegramID = &updatedUser.TelegramID.String
	}
	if updatedUser.JobTitle.Valid {
		responseJobTitle = &updatedUser.JobTitle.String
	}
	if updatedUser.Education.Valid {
		responseEducation = &updatedUser.Education.String
	}
	if updatedUser.MarriageStatus.Valid {
		responseMarriageStatus = &updatedUser.MarriageStatus.String
	}
	if updatedUser.PartnerName.Valid {
		responsePartnerName = &updatedUser.PartnerName.String
	}

	u.log.Info(ctx, "Password reset successfully", zap.String("user_id", userID.String()))

	return &dto.User{
		ID:             updatedUser.ID,
		Name:           updatedUser.Name,
		LastName:       updatedUser.Lastname,
		PhoneNumber:    updatedUser.PhoneNumber,
		Role:           updatedUser.Role,
		Status:         string(updatedUser.Status),
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  updatedUser.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      updatedUser.CreatedAt,
		UpdatedAt:      updatedUser.UpdatedAt,
	}, nil
}

// CleanupExpiredResetTokens removes expired password reset tokens
func (u *user) CleanupExpiredResetTokens(ctx context.Context) error {
	err := u.db.CleanupExpiredResetTokens(ctx)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not cleanup expired tokens")
		u.log.Error(ctx, "unable to cleanup expired reset tokens", zap.Error(err))
		return err
	}

	u.log.Debug(ctx, "Cleaned up expired reset tokens")
	return nil
}

// CreatePasswordResetOTP creates a new password reset OTP
func (u *user) CreatePasswordResetOTP(ctx context.Context, userID uuid.UUID, otpHash string, expiresAt time.Time) (*dto.PasswordResetOTP, error) {
	otp, err := u.db.CreatePasswordResetOTP(ctx, db.CreatePasswordResetOTPParams{
		UserID:    userID,
		OtpHash:   otpHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not create password reset OTP")
		u.log.Error(ctx, "unable to create password reset OTP", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	return &dto.PasswordResetOTP{
		ID:        otp.ID,
		UserID:    otp.UserID,
		OTPHash:   otp.OtpHash,
		CreatedAt: otp.CreatedAt,
		ExpiresAt: otp.ExpiresAt,
		Used:      otp.Used,
		Attempts:  int(otp.Attempts),
	}, nil
}

// GetValidPasswordResetOTP retrieves a valid password reset OTP
func (u *user) GetValidPasswordResetOTP(ctx context.Context, userID uuid.UUID, otpHash string) (*dto.PasswordResetOTP, error) {
	otp, err := u.db.GetValidPasswordResetOTP(ctx, db.GetValidPasswordResetOTPParams{
		UserID:  userID,
		OtpHash: otpHash,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "OTP not found or expired")
			u.log.Info(ctx, "Password reset OTP not found or expired", zap.String("user_id", userID.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read password reset OTP")
		u.log.Error(ctx, "unable to get password reset OTP", zap.Error(err))
		return nil, err
	}

	return &dto.PasswordResetOTP{
		ID:        otp.ID,
		UserID:    otp.UserID,
		OTPHash:   otp.OtpHash,
		CreatedAt: otp.CreatedAt,
		ExpiresAt: otp.ExpiresAt,
		Used:      otp.Used,
		Attempts:  int(otp.Attempts),
	}, nil
}

// MarkPasswordResetOTPUsed marks a password reset OTP as used
func (u *user) MarkPasswordResetOTPUsed(ctx context.Context, otpID uuid.UUID) error {
	err := u.db.MarkPasswordResetOTPUsed(ctx, otpID)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not mark OTP as used")
		u.log.Error(ctx, "unable to mark password reset OTP as used", zap.Error(err), zap.String("otp_id", otpID.String()))
		return err
	}

	u.log.Info(ctx, "Password reset OTP marked as used", zap.String("otp_id", otpID.String()))
	return nil
}
