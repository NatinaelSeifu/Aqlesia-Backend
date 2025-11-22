package user

import (
	"aqlesia/internal/constants"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/handler/rest"
	"aqlesia/internal/module"
	"aqlesia/platform/logger"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type user struct {
	logger         logger.Logger
	userModule     module.User
	contextTimeout time.Duration
}

func Init(logger logger.Logger, userModule module.User, contextTimeout time.Duration) rest.User {
	return &user{
		userModule:     userModule,
		logger:         logger,
		contextTimeout: contextTimeout,
	}
}

// UpdateUser updates a user profile by ID.
//
//	@Summary		Update user profile
//	@Description	Update user profile information including name, lastname, phone_number, job_title, education, marriage_status, childrens_name, and telegram_id. Users can update their own profile, admins/managers can update any user. All fields are optional - only provided fields will be updated.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"User ID (UUID)"
//	@Param			user	body		dto.UpdateUser		true	"Updated user profile details (all fields optional)"
//	@Success		200		{object}	dto.User			"Successfully updated user profile"
//	@Failure		400		{object}	model.ErrorResponse	"Bad request - validation errors (invalid phone, marriage_status, etc.)"
//	@Failure		401		{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		403		{object}	model.ErrorResponse	"Forbidden - can only update own profile unless admin/manager"
//	@Failure		404		{object}	model.ErrorResponse	"User not found"
//	@Router			/users/{id} [patch]
func (u *user) UpdateUser(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	user := dto.UpdateUser{}
	err := ctx.ShouldBind(&user)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.logger.Error(ctx, "unable to bind user data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	updatedUser, err := u.userModule.Update(cntx, ctx.Param("id"), user)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Check if the user is updating their own profile
	authUserInterface, exists := ctx.Get("auth_user")
	if exists {
		if authUser, ok := authUserInterface.(*dto.User); ok {
			// If the authenticated user is updating their own profile, hide the role
			if authUser.ID.String() == ctx.Param("id") {
				// Create a copy of the user without the role field
				userCopy := *updatedUser
				userCopy.Role = "" // Hide the role field
				constants.SuccessResponse(ctx, http.StatusOK, &userCopy, nil)
				return
			}
		}
	}

	// For admin/manager updating other users or any other case, return full user info
	constants.SuccessResponse(ctx, http.StatusOK, updatedUser, nil)
}

// GetUser gets a user by ID.
//
//	@Summary		Get user by ID
//	@Description	Retrieve user information by user ID. Users can read their own profile, admins and managers can read any user. Note: when users view their own profile, the role field is omitted from the response.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"User ID (UUID)"
//	@Success		200	{object}	dto.User			"Successfully retrieved user"
//	@Failure		400	{object}	model.ErrorResponse	"Bad request - invalid user ID"
//	@Failure		401	{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse	"Forbidden - can only access own profile unless admin/manager"
//	@Failure		404	{object}	model.ErrorResponse	"User not found"
//	@Router			/users/{id} [get]
func (u *user) GetUser(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	user, err := u.userModule.Get(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Check if the user is viewing their own profile
	authUserInterface, exists := ctx.Get("auth_user")
	if exists {
		if authUser, ok := authUserInterface.(*dto.User); ok {
			// If the authenticated user is viewing their own profile, hide the role
			if authUser.ID.String() == ctx.Param("id") {
				// Create a copy of the user without the role field
				userCopy := *user
				userCopy.Role = "" // Hide the role field
				constants.SuccessResponse(ctx, http.StatusOK, &userCopy, nil)
				return
			}
		}
	}

	// For admin/manager viewing other users or any other case, return full user info
	constants.SuccessResponse(ctx, http.StatusOK, user, nil)
}

// GetUsers gets a list of users.
//
//	@Summary		Get all users
//	@Description	Retrieve a list of all registered users with pagination. Requires authentication and 'users:list' permission. Only admins and managers can access this endpoint.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int						false	"Page number (default: 1)"
//	@Param			page_size	query		int						false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	dto.UserListResponse	"Successfully retrieved users"
//	@Failure		401			{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse	"Forbidden - insufficient permissions (requires admin or manager role)"
//	@Failure		500			{object}	model.ErrorResponse	"Internal server error"
//	@Router			/users [get]
func (u *user) GetUsers(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	users, err := u.userModule.GetAll(cntx, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, users, nil)
}

// DeleteUser is used to delete a user.
//
//	@Summary		Delete user
//	@Description	Soft delete a user by ID. Requires 'users:delete' permission (admin role only).
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string				true	"User ID (UUID)"
//	@Success		200	{object}	model.SuccessResponse	"Successfully deleted the user"
//	@Failure		400	{object}	model.ErrorResponse	"Bad request - invalid user ID"
//	@Failure		401	{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse	"Forbidden - requires admin role"
//	@Failure		404	{object}	model.ErrorResponse	"User not found"
//	@Router			/users/{id} [delete]
func (u *user) DeleteUser(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	err := u.userModule.DeleteUser(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "User deleted successfully", nil)
}

// UpdateUserStatus updates a user's status (approve/reject).
//
//	@Summary		Update user status
//	@Description	Update a user's account status. Used by admins/managers to approve or reject pending user registrations. Status can be ACTIVE (approved) or INACTIVE (rejected). When users register, they start with PENDING status and cannot login until approved.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"User ID (UUID)"
//	@Param			status	body		dto.UserStatusUpdateRequest	true	"New status details (ACTIVE or INACTIVE)"
//	@Success		200			{object}	dto.User				"Successfully updated user status"
//	@Failure		400			{object}	model.ErrorResponse		"Bad request - validation errors or invalid user ID"
//	@Failure		401			{object}	model.ErrorResponse		"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse		"Forbidden - requires admin or manager role"
//	@Failure		404			{object}	model.ErrorResponse		"User not found"
//	@Router			/users/{id}/status [patch]
func (u *user) UpdateUserStatus(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	var statusRequest dto.UserStatusUpdateRequest
	err := ctx.ShouldBind(&statusRequest)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.logger.Error(ctx, "unable to bind status update data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate input
	if err := statusRequest.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.logger.Error(ctx, "validation failed for status update", zap.Error(err), zap.String("user-id", ctx.Param("id")))
		_ = ctx.Error(err)
		return
	}

	// Update user status
	updatedUser, err := u.userModule.UpdateStatus(cntx, ctx.Param("id"), statusRequest.Status)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Log the admin action
	authUserInterface, exists := ctx.Get("auth_user")
	if exists {
		if authUser, ok := authUserInterface.(*dto.User); ok {
			u.logger.Info(ctx, "user status updated by admin",
				zap.String("target-user-id", ctx.Param("id")),
				zap.String("admin-user-id", authUser.ID.String()),
				zap.String("new-status", statusRequest.Status))
		}
	}

	constants.SuccessResponse(ctx, http.StatusOK, updatedUser, nil)
}

// UploadAvatar uploads and sets the user's profile picture.
//
//	@Summary		Upload user avatar
//	@Description	Upload a profile picture to DigitalOcean Spaces and set it for the user
//	@Tags			Users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path	string	true	"User ID (UUID)"
//	@Param			file		formData	file	true	"Image file (max 5MB)"
//	@Success		200			{object}	map[string]string	"Successfully uploaded avatar"
//	@Failure		400			{object}	model.ErrorResponse	"Bad request - invalid image or input"
//	@Failure		401			{object}	model.ErrorResponse	"Unauthorized"
//	@Failure		403			{object}	model.ErrorResponse	"Forbidden"
//	@Failure		404			{object}	model.ErrorResponse	"User not found"
//	@Router			/users/{id}/avatar [post]
func (u *user) UploadAvatar(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "file is required")
		u.logger.Error(ctx, "unable to read upload file", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Basic size limit: 5MB
	const maxSize int64 = 5 * 1024 * 1024
	if fileHeader.Size > maxSize {
		err := errors.ErrInvalidUserInput.New("file is too large (max 5MB)")
		_ = ctx.Error(err)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "could not open uploaded file")
		_ = ctx.Error(err)
		return
	}
	defer f.Close()

	// Detect content type
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := f.Seek(0, 0); err != nil {
		_ = ctx.Error(err)
		return
	}
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		err := errors.ErrInvalidUserInput.New("unsupported image type (allowed: jpeg, png, webp)")
		_ = ctx.Error(err)
		return
	}

	// Build object key
	userID := ctx.Param("id")
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}
	objectKey := "Sewasew-SGT/" + userID + ext

	// S3 config from viper
	endpoint := viper.GetString("spaces.endpoint")
	region := viper.GetString("spaces.region")
	bucket := viper.GetString("spaces.bucket")
	accessKey := viper.GetString("spaces.access_key")
	secretKey := viper.GetString("spaces.secret_key")
	cdnBase := viper.GetString("spaces.cdn_base_url")

	if endpoint == "" || region == "" || bucket == "" || accessKey == "" || secretKey == "" {
		u.logger.Error(ctx, "spaces config is incomplete")
		_ = ctx.Error(errors.ErrInvalidUserInput.New("object storage is not configured"))
		return
	}

	// Build AWS SDK v2 config for DO Spaces
	awsCfg, err := awsconfig.LoadDefaultConfig(cntx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		u.logger.Error(ctx, "failed to load AWS config", zap.Error(err))
		_ = ctx.Error(err)
		return
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = false
		o.BaseEndpoint = aws.String(endpoint)
	})
	uploader := manager.NewUploader(s3Client)

	// Upload
	uploadInput := &s3.PutObjectInput{
		Bucket:       &bucket,
		Key:          &objectKey,
		Body:         f,
		ContentType:  &contentType,
		CacheControl: aws.String("public, max-age=31536000, immutable"),
		ACL:          "public-read",
	}
	_, err = uploader.Upload(cntx, uploadInput)
	if err != nil {
		u.logger.Error(ctx, "failed to upload to spaces", zap.Error(err))
		_ = ctx.Error(errors.ErrWriteError.Wrap(err, "failed to upload image"))
		return
	}

	// Construct public URL
	imageURL := ""
	if cdnBase != "" {
		imageURL = strings.TrimRight(cdnBase, "/") + "/" + objectKey
	} else {
		// Fallback to default Spaces URL format
		imageURL = strings.TrimRight(endpoint, "/") + "/" + bucket + "/" + objectKey
	}

	// Persist URL in DB
	if err := u.userModule.UpdateProfileImage(cntx, userID, imageURL); err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, map[string]string{"image_url": imageURL}, nil)
}
