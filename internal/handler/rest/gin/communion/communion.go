package communion

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
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type communion struct {
	logger         logger.Logger
	communionModule module.Communion
	contextTimeout time.Duration
}

func Init(logger logger.Logger, communionModule module.Communion, contextTimeout time.Duration) rest.Communion {
	return &communion{
		communionModule: communionModule,
		logger:         logger,
		contextTimeout: contextTimeout,
	}
}

// CreateCommunion creates a new communion request.
//
//	@Summary		Create communion request
//	@Description	Create a new communion request for the authenticated user. The communion date must be in the past (format: YYYY-MM-DD). Users can create multiple communion requests.
//	@Tags			Communion
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			communion	body		dto.CreateCommunionRequest	true	"Communion request details"
//	@Success		201			{object}	dto.Communion				"Successfully created communion request"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - validation errors (invalid date format, future date, etc.)"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/communion [post]
func (c *communion) CreateCommunion(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Get authenticated user from context
	authUserInterface, exists := ctx.Get("auth_user")
	if !exists {
		err := errors.ErrInvalidUserInput.New("authentication required")
		c.logger.Error(ctx, "authentication required for communion creation", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	authUser, ok := authUserInterface.(*dto.User)
	if !ok {
		err := errors.ErrInvalidUserInput.New("invalid authentication data")
		c.logger.Error(ctx, "invalid authentication data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	communion := dto.CreateCommunionRequest{}
	err := ctx.ShouldBind(&communion)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid communion request data")
		c.logger.Error(ctx, "unable to bind communion data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	c.logger.Info(ctx, "Creating communion request", 
		zap.String("user-id", authUser.ID.String()),
		zap.String("communion-date", communion.CommunionDate))

	createdCommunion, err := c.communionModule.Create(cntx, authUser.ID, communion)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusCreated, createdCommunion, nil)
}

// GetCommunion gets a communion request by ID.
//
//	@Summary		Get communion request by ID
//	@Description	Retrieve communion request information by ID. Users can view their own requests, admins can view any communion request.
//	@Tags			Communion
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Communion ID (UUID)"
//	@Success		200	{object}	dto.Communion			"Successfully retrieved communion request"
//	@Failure		400	{object}	model.ErrorResponse		"Bad request - invalid communion ID"
//	@Failure		401	{object}	model.ErrorResponse		"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse		"Forbidden - can only access own requests unless admin"
//	@Failure		404	{object}	model.ErrorResponse		"Communion request not found"
//	@Router			/communion/{id} [get]
func (c *communion) GetCommunion(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	communion, err := c.communionModule.Get(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Check if user is authorized to view this communion request
	authUserInterface, exists := ctx.Get("auth_user")
	if exists {
		if authUser, ok := authUserInterface.(*dto.User); ok {
			// Users can only view their own communion requests unless they're admin
			if authUser.Role != "admin" && communion.UserID != authUser.ID {
				err := errors.ErrInvalidUserInput.New("users can only view their own communion requests")
				c.logger.Warn(ctx, "unauthorized attempt to view communion request", 
					zap.String("communion-id", ctx.Param("id")),
					zap.String("communion-owner", communion.UserID.String()),
					zap.String("requesting-user", authUser.ID.String()),
					zap.String("user-role", authUser.Role))
				_ = ctx.Error(err)
				return
			}
		}
	}

	constants.SuccessResponse(ctx, http.StatusOK, communion, nil)
}

// GetUserCommunions gets communion requests for the authenticated user.
//
//	@Summary		Get user's communion requests
//	@Description	Retrieve a paginated list of communion requests for the authenticated user.
//	@Tags			Communion
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default: 1)"
//	@Param			page_size	query		int	false	"Number of items per page (default: 20, max: 100)"
//	@Success		200			{object}	dto.CommunionListResponse	"Successfully retrieved user communion requests"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - invalid pagination parameters"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/communion/user [get]
func (c *communion) GetUserCommunions(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Get authenticated user from context
	authUserInterface, exists := ctx.Get("auth_user")
	if !exists {
		err := errors.ErrInvalidUserInput.New("authentication required")
		c.logger.Error(ctx, "authentication required for user communions", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	authUser, ok := authUserInterface.(*dto.User)
	if !ok {
		err := errors.ErrInvalidUserInput.New("invalid authentication data")
		c.logger.Error(ctx, "invalid authentication data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}

	communions, err := c.communionModule.GetUserCommunions(cntx, authUser.ID, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, communions, nil)
}

// GetAllCommunions gets all communion requests (admin/manager only).
//
//	@Summary		Get all communion requests
//	@Description	Retrieve a paginated list of all communion requests. Requires 'communions:list' permission (admin role only).
//	@Tags			Communion
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default: 1)"
//	@Param			page_size	query		int	false	"Number of items per page (default: 20, max: 100)"
//	@Success		200			{object}	dto.CommunionListResponse	"Successfully retrieved all communion requests"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - invalid pagination parameters"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse			"Forbidden - insufficient permissions (requires admin role)"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/communion/all [get]
func (c *communion) GetAllCommunions(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}

	communions, err := c.communionModule.GetAllCommunions(cntx, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, communions, nil)
}

// GetPendingCommunions gets pending communion requests (admin/manager only).
//
//	@Summary		Get pending communion requests
//	@Description	Retrieve a paginated list of pending communion requests that require approval. Requires 'communions:manage' permission (admin role only).
//	@Tags			Communion
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default: 1)"
//	@Param			page_size	query		int	false	"Number of items per page (default: 20, max: 100)"
//	@Success		200			{object}	dto.CommunionListResponse	"Successfully retrieved pending communion requests"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - invalid pagination parameters"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse			"Forbidden - insufficient permissions (requires admin role)"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/communion/pending [get]
func (c *communion) GetPendingCommunions(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}

	communions, err := c.communionModule.GetPendingCommunions(cntx, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, communions, nil)
}

// UpdateCommunionStatus updates the status of a communion request (admin/manager only).
//
//	@Summary		Update communion request status
//	@Description	Update the status of a communion request to approved or rejected. Requires 'communions:manage' permission (admin role only). Only pending requests can have their status updated.
//	@Tags			Communion
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Communion ID (UUID)"
//	@Param			status	body		dto.UpdateCommunionStatusRequest	true	"Status update details (approved/rejected)"
//	@Success		200		{object}	dto.Communion					"Successfully updated communion status"
//	@Failure		400		{object}	model.ErrorResponse				"Bad request - validation errors (invalid status, non-pending request, etc.)"
//	@Failure		401		{object}	model.ErrorResponse				"Unauthorized - invalid or missing token"
//	@Failure		403		{object}	model.ErrorResponse				"Forbidden - insufficient permissions (requires admin role)"
//	@Failure		404		{object}	model.ErrorResponse				"Communion request not found"
//	@Router			/communion/{id}/status [patch]
func (c *communion) UpdateCommunionStatus(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Get authenticated user from context
	authUserInterface, exists := ctx.Get("auth_user")
	if !exists {
		err := errors.ErrInvalidUserInput.New("authentication required")
		c.logger.Error(ctx, "authentication required for communion status update", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	authUser, ok := authUserInterface.(*dto.User)
	if !ok {
		err := errors.ErrInvalidUserInput.New("invalid authentication data")
		c.logger.Error(ctx, "invalid authentication data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	statusUpdate := dto.UpdateCommunionStatusRequest{}
	err := ctx.ShouldBind(&statusUpdate)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid status update request")
		c.logger.Error(ctx, "unable to bind status update data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	c.logger.Info(ctx, "Updating communion status", 
		zap.String("communion-id", ctx.Param("id")),
		zap.String("admin-user-id", authUser.ID.String()),
		zap.String("status", statusUpdate.Status))

	updatedCommunion, err := c.communionModule.UpdateStatus(cntx, ctx.Param("id"), authUser.ID, statusUpdate)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, updatedCommunion, nil)
}

// UpdateCommunion updates a communion request (user can update their own pending requests).
//
//	@Summary		Update communion request
//	@Description	Update a communion request. Users can only update their own pending requests. Only the communion date can be updated.
//	@Tags			Communion
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string						true	"Communion ID (UUID)"
//	@Param			communion	body		dto.CreateCommunionRequest	true	"Updated communion request details"
//	@Success		200			{object}	dto.Communion				"Successfully updated communion request"
//	@Failure		400			{object}	model.ErrorResponse			"Bad request - validation errors (invalid date, non-pending request, etc.)"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse			"Forbidden - can only update own pending requests"
//	@Failure		404			{object}	model.ErrorResponse			"Communion request not found"
//	@Router			/communion/{id} [patch]
func (c *communion) UpdateCommunion(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	// Get authenticated user from context
	authUserInterface, exists := ctx.Get("auth_user")
	if !exists {
		err := errors.ErrInvalidUserInput.New("authentication required")
		c.logger.Error(ctx, "authentication required for communion update", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	authUser, ok := authUserInterface.(*dto.User)
	if !ok {
		err := errors.ErrInvalidUserInput.New("invalid authentication data")
		c.logger.Error(ctx, "invalid authentication data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	communion := dto.CreateCommunionRequest{}
	err := ctx.ShouldBind(&communion)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid communion update data")
		c.logger.Error(ctx, "unable to bind communion update data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	c.logger.Info(ctx, "Updating communion request", 
		zap.String("communion-id", ctx.Param("id")),
		zap.String("user-id", authUser.ID.String()),
		zap.String("new-communion-date", communion.CommunionDate))

	updatedCommunion, err := c.communionModule.Update(cntx, ctx.Param("id"), authUser.ID, communion)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, updatedCommunion, nil)
}

// DeleteCommunion deletes a communion request (admin only).
//
//	@Summary		Delete communion request
//	@Description	Soft delete a communion request by ID. Requires 'communions:delete' permission (admin role only).
//	@Tags			Communion
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Communion ID (UUID)"
//	@Success		200	{object}	model.SuccessResponse	"Successfully deleted the communion request"
//	@Failure		400	{object}	model.ErrorResponse		"Bad request - invalid communion ID"
//	@Failure		401	{object}	model.ErrorResponse		"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse		"Forbidden - requires admin role"
//	@Failure		404	{object}	model.ErrorResponse		"Communion request not found"
//	@Router			/communion/{id} [delete]
func (c *communion) DeleteCommunion(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, c.contextTimeout)
	defer cancel()

	c.logger.Info(ctx, "Deleting communion request", zap.String("communion-id", ctx.Param("id")))

	err := c.communionModule.Delete(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Communion request deleted successfully", nil)
}
