package auth

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/constants"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/handler/rest"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type authHandler struct {
	logger         logger.Logger
	userStorage    storage.User
	jwtManager     *auth.JWTManager
	contextTimeout time.Duration
}

func Init(logger logger.Logger, userStorage storage.User, jwtManager *auth.JWTManager, contextTimeout time.Duration) rest.Auth {
	return &authHandler{
		logger:         logger,
		userStorage:    userStorage,
		jwtManager:     jwtManager,
		contextTimeout: contextTimeout,
	}
}

// Login authenticates a user with phone number and password
//
//	@Summary		Login with Ethiopian phone number
//	@Description	Authenticate with Ethiopian phone number and password. Returns JWT access and refresh tokens. Supports multiple phone formats: +251912345678, 0912345678, or 912345678.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		dto.LoginRequest	true	"Login credentials (phone number + password)"
//	@Success		200			{object}	dto.AuthResponse	"Successfully authenticated - returns access token (15min) and refresh token (7 days)"
//	@Failure		400			{object}	model.ErrorResponse	"Bad request - validation errors"
//	@Failure		401			{object}	model.ErrorResponse	"Unauthorized - invalid phone number or password"
//	@Router			/auth/login [post]
func (a *authHandler) Login(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	var loginReq dto.LoginRequest
	err := ctx.ShouldBind(&loginReq)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid login request")
		a.logger.Error(ctx, "unable to bind login data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate input
	if err := loginReq.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", loginReq))
		_ = ctx.Error(err)
		return
	}

	// Normalize phone number for consistent lookup
	normalizedPhone := loginReq.NormalizePhoneNumber()
	
	// Get user by phone number (with password for authentication)
	user, err := a.userStorage.GetUserByPhoneWithPassword(cntx, normalizedPhone)
	if err != nil {
		// Don't reveal if user exists or not for security
		err := errors.ErrUnauthorized.New("invalid phone number or password")
		a.logger.Warn(ctx, "login attempt with invalid phone", zap.String("phone", normalizedPhone))
		_ = ctx.Error(err)
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		err := errors.ErrUnauthorized.New("invalid phone number or password")
		a.logger.Warn(ctx, "login attempt with invalid password", zap.String("phone", normalizedPhone))
		_ = ctx.Error(err)
		return
	}

	// Get user status by querying through storage layer
	userDTO, err := a.userStorage.GetUserByPhone(cntx, normalizedPhone)
	if err != nil {
		err := errors.ErrUnauthorized.New("invalid phone number or password")
		a.logger.Warn(ctx, "failed to get user for status check", zap.String("phone", normalizedPhone))
		_ = ctx.Error(err)
		return
	}

	// Check if user account is approved
	if userDTO.Status != "ACTIVE" {
		err := errors.ErrUnauthorized.New("account not yet approved by admin")
		a.logger.Warn(ctx, "login attempt with non-active account", 
			zap.String("phone", normalizedPhone), 
			zap.String("status", userDTO.Status))
		_ = ctx.Error(err)
		return
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := a.jwtManager.GenerateTokenPair(user.ID.String(), user.PhoneNumber)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "failed to generate tokens")
		a.logger.Error(ctx, "token generation failed", zap.Error(err), zap.String("user_id", user.ID.String()))
		_ = ctx.Error(err)
		return
	}

	// Use the userDTO we already fetched for status check
	userResponse := userDTO

	// Prepare auth response
	authResponse := dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(auth.AccessTokenDuration.Seconds()),
		User:         userResponse,
	}

	a.logger.Info(ctx, "user logged in successfully", zap.String("user_id", user.ID.String()), zap.String("phone", user.PhoneNumber))
	constants.SuccessResponse(ctx, http.StatusOK, authResponse, nil)
}

// RefreshToken generates a new access token using a refresh token
//
//	@Summary		Refresh access token
//	@Description	Generate a new access token using a valid refresh token. The refresh token remains valid for its full 7-day lifetime.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			refresh		body		dto.RefreshTokenRequest	true	"Refresh token request"
//	@Success		200			{object}	dto.AuthResponse		"Successfully refreshed - returns new access token (15min) with same refresh token"
//	@Failure		400			{object}	model.ErrorResponse		"Bad request - missing or malformed refresh token"
//	@Failure		401			{object}	model.ErrorResponse		"Unauthorized - invalid or expired refresh token"
//	@Router			/auth/refresh [post]
func (a *authHandler) RefreshToken(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	var refreshReq dto.RefreshTokenRequest
	err := ctx.ShouldBind(&refreshReq)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid refresh request")
		a.logger.Error(ctx, "unable to bind refresh data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate input
	if err := refreshReq.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "validation failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Generate new access token
	newAccessToken, err := a.jwtManager.RefreshAccessToken(refreshReq.RefreshToken)
	if err != nil {
		err := errors.ErrUnauthorized.Wrap(err, "invalid or expired refresh token")
		a.logger.Warn(ctx, "refresh token validation failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate the refresh token to get user info
	claims, err := a.jwtManager.ValidateToken(refreshReq.RefreshToken)
	if err != nil {
		err := errors.ErrUnauthorized.Wrap(err, "invalid refresh token")
		a.logger.Error(ctx, "refresh token validation failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Get user info for response
	user, err := a.userStorage.GetUserByPhone(cntx, claims.PhoneNumber)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "failed to get user")
		a.logger.Error(ctx, "failed to get user during refresh", zap.Error(err), zap.String("user_id", claims.UserID))
		_ = ctx.Error(err)
		return
	}

	// Prepare user response (user is already a DTO from GetUserByPhone)
	userResponse := user

	// Prepare auth response
	authResponse := dto.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshReq.RefreshToken, // Keep the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    int64(auth.AccessTokenDuration.Seconds()),
		User:         userResponse,
	}

	a.logger.Info(ctx, "token refreshed successfully", zap.String("user_id", claims.UserID))
	constants.SuccessResponse(ctx, http.StatusOK, authResponse, nil)
}

// Register creates a new user account
//
//	@Summary		Register a new user
//	@Description	Register a new user with Ethiopian phone number. Supports multiple formats: +251912345678, 0912345678, or 912345678. Phone numbers are automatically normalized to E.164 format. Users default to 'user' role.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			user	body		dto.RegisterUser	true	"User registration details"
//	@Success		201		{object}	dto.User			"Successfully created user"
//	@Failure		400		{object}	model.ErrorResponse	"Bad request - validation errors or duplicate phone number"
//	@Router			/auth/register [post]
func (a *authHandler) Register(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	var registerReq dto.RegisterUser
	err := ctx.ShouldBind(&registerReq)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid registration request")
		a.logger.Error(ctx, "unable to bind registration data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate input
	if err := registerReq.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", registerReq))
		_ = ctx.Error(err)
		return
	}

	// Check if user already exists
	exists, err := a.userStorage.CheckUserExists(cntx, registerReq)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "failed to check user existence")
		a.logger.Error(ctx, "failed to check user existence", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	if exists {
		err := errors.ErrDataExists.New("user with this phone number already exists")
		a.logger.Warn(ctx, "registration attempt with existing phone number", zap.String("phone", registerReq.NormalizePhoneNumber()))
		_ = ctx.Error(err)
		return
	}

	// Create the user
	createdUser, err := a.userStorage.Create(cntx, registerReq)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	a.logger.Info(ctx, "user registered successfully", zap.String("user_id", createdUser.ID.String()), zap.String("phone", createdUser.PhoneNumber))
	constants.SuccessResponse(ctx, http.StatusCreated, createdUser, nil)
}

// ChangePassword allows a user to change their password.
//
//	@Summary		Change user password
//	@Description	Change the authenticated user's password. Requires the current password for verification.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			password	body		dto.ChangePasswordRequest	true	"Current and new password details"
//	@Success		200			{object}	dto.User					"Successfully changed password"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - validation errors or incorrect current password"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/auth/change-password [post]
func (a *authHandler) ChangePassword(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Get authenticated user from context
	authUserInterface, exists := ctx.Get("auth_user")
	if !exists {
		a.logger.Error(ctx, "no authenticated user found in context")
		_ = ctx.Error(errors.ErrInvalidUserInput.New("authentication required"))
		return
	}

	authUser, ok := authUserInterface.(*dto.User)
	if !ok {
		a.logger.Error(ctx, "invalid user type in context")
		_ = ctx.Error(errors.ErrInvalidUserInput.New("invalid authentication context"))
		return
	}

	// Parse request body
	var passwordRequest dto.ChangePasswordRequest
	err := ctx.ShouldBind(&passwordRequest)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "unable to bind password change data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate input
	if err := passwordRequest.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "validation failed for password change", zap.Error(err), zap.String("user-id", authUser.ID.String()))
		_ = ctx.Error(err)
		return
	}

	// Check if new password is different from current password
	if passwordRequest.CurrentPassword == passwordRequest.NewPassword {
		err := errors.ErrInvalidUserInput.New("new password must be different from current password")
		a.logger.Warn(ctx, "user tried to set same password", zap.String("user-id", authUser.ID.String()))
		_ = ctx.Error(err)
		return
	}

	// Change password
	updatedUser, err := a.userStorage.ChangePassword(cntx, authUser.ID, passwordRequest.CurrentPassword, passwordRequest.NewPassword)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Hide the role field for the user's own response
	userCopy := *updatedUser
	userCopy.Role = ""
	a.logger.Info(ctx, "password changed successfully", zap.String("user-id", authUser.ID.String()))
	constants.SuccessResponse(ctx, http.StatusOK, &userCopy, nil)
}
