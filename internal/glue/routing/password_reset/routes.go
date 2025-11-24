package password_reset

import (
	"aqlesia/internal/module"
	"aqlesia/platform/logger"

	"github.com/gin-gonic/gin"
)

// InitRoutes initializes password reset routes
func InitRoutes(router *gin.RouterGroup, passwordResetModule module.PasswordReset, log logger.Logger) {
	handler := NewHandler(passwordResetModule, log.Named("password-reset-handler"))

	// Public routes (no authentication required)
	authGroup := router.Group("/auth")
	{
	// API endpoints
		authGroup.POST("/forgot-password", handler.ForgotPassword)
		authGroup.POST("/verify-otp", handler.VerifyOTP)
		authGroup.POST("/reset-password", handler.ResetPassword)
		authGroup.POST("/link-telegram", handler.LinkTelegram)
		authGroup.POST("/check-telegram", handler.CheckTelegramVerification)
		
		// Note: Reset form is handled by client-side application
		// Telegram links point directly to client app at /reset-password?token=...
	}
}
