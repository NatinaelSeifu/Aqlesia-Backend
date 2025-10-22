package password_reset

import (
	"aqlesia/internal/constants"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/platform/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type handler struct {
	passwordResetModule module.PasswordReset
	logger              logger.Logger
}

func NewHandler(passwordResetModule module.PasswordReset, logger logger.Logger) *handler {
	return &handler{
		passwordResetModule: passwordResetModule,
		logger:              logger,
	}
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Initiate password reset process. A 6-digit OTP will be sent to the user's linked Telegram account if it exists
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Forgot password request"
// @Success 200 {object} dto.ForgotPasswordResponse "Success message (always returned regardless of user existence)"
// @Failure 400 {object} model.ErrorResponse "Validation error"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /auth/forgot-password [post]
func (h *handler) ForgotPassword(ctx *gin.Context) {
	var req dto.ForgotPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.logger.Error(ctx.Request.Context(), "failed to bind request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response, err := h.passwordResetModule.ForgotPassword(ctx.Request.Context(), req)
	if err != nil {
		h.logger.Error(ctx.Request.Context(), "forgot password failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, response, nil)
}

// VerifyOTP godoc
// @Summary Verify password reset OTP
// @Description Verify the OTP received via Telegram and get a reset token for password reset
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.VerifyOTPRequest true "OTP verification request"
// @Success 200 {object} dto.VerifyOTPResponse "OTP verification result with reset token if valid"
// @Failure 400 {object} model.ErrorResponse "Validation error or invalid OTP"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /auth/verify-otp [post]
func (h *handler) VerifyOTP(ctx *gin.Context) {
	var req dto.VerifyOTPRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.logger.Error(ctx.Request.Context(), "failed to bind request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response, err := h.passwordResetModule.VerifyOTP(ctx.Request.Context(), req)
	if err != nil {
		h.logger.Error(ctx.Request.Context(), "verify OTP failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, response, nil)
}

// ResetPassword godoc
// @Summary Complete password reset
// @Description Reset user password using a reset token obtained from OTP verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Reset password request"
// @Success 200 {object} dto.ResetPasswordResponse "Password reset successful"
// @Failure 400 {object} model.ErrorResponse "Invalid token or validation error"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /auth/reset-password [post]
func (h *handler) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.logger.Error(ctx.Request.Context(), "failed to bind request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response, err := h.passwordResetModule.ResetPassword(ctx.Request.Context(), req)
	if err != nil {
		h.logger.Error(ctx.Request.Context(), "reset password failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, response, nil)
}

// LinkTelegram godoc
// @Summary Generate Telegram link code
// @Description Generate a temporary link code to connect user's Telegram account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.TelegramLinkRequest true "Telegram link request"
// @Success 200 {object} dto.TelegramLinkResponse "Link code and Telegram URL generated"
// @Failure 400 {object} model.ErrorResponse "User not found or validation error"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /auth/link-telegram [post]
func (h *handler) LinkTelegram(ctx *gin.Context) {
	var req dto.TelegramLinkRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.logger.Error(ctx.Request.Context(), "failed to bind request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response, err := h.passwordResetModule.LinkTelegram(ctx.Request.Context(), req)
	if err != nil {
		h.logger.Error(ctx.Request.Context(), "telegram link failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, response, nil)
}

// CheckTelegramVerification godoc
// @Summary Check Telegram verification status
// @Description Check if a user has a verified Telegram account without sending OTP
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.TelegramLinkRequest true "Telegram verification check request"
// @Success 200 {object} map[string]interface{} "Verification status"
// @Failure 400 {object} model.ErrorResponse "Validation error"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /auth/check-telegram [post]
func (h *handler) CheckTelegramVerification(ctx *gin.Context) {
	var req dto.TelegramLinkRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.logger.Error(ctx.Request.Context(), "failed to bind request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response, err := h.passwordResetModule.CheckTelegramVerification(ctx.Request.Context(), req)
	if err != nil {
		h.logger.Error(ctx.Request.Context(), "telegram verification check failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, response, nil)
}

// ResetPasswordForm handler removed - password reset is now handled client-side
// Telegram reset links point directly to the client application
