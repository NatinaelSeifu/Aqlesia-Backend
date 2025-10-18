package communion

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type communion struct {
	communionStorage storage.Communion
	userStorage      storage.User
	log              logger.Logger
}

func Init(communionStorage storage.Communion, userStorage storage.User, log logger.Logger) module.Communion {
	return &communion{
		communionStorage: communionStorage,
		userStorage:      userStorage,
		log:              log,
	}
}

func (c *communion) Create(ctx context.Context, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error) {
	c.log.Info(ctx, "Creating communion request", zap.String("user-id", userID.String()), zap.Any("request", param))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion request")
		c.log.Error(ctx, "invalid communion request", zap.Error(err))
		return nil, err
	}

	// Verify user exists
	_, err := c.userStorage.Get(ctx, userID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "user not found")
		c.log.Error(ctx, "user not found for communion request creation", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Parse communion date
	communionDate, err := param.GetCommunionDate()
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion date format")
		c.log.Error(ctx, "invalid communion date format", zap.Error(err))
		return nil, err
	}

	// Create communion request through storage layer
	communion, err := c.communionStorage.Create(ctx, userID, param)
	if err != nil {
		c.log.Error(ctx, "failed to create communion request", zap.Error(err))
		return nil, err
	}

	c.log.Info(ctx, "Communion request created successfully", 
		zap.String("communion-id", communion.ID.String()),
		zap.String("user-id", userID.String()),
		zap.Time("communion-date", communionDate))

	return communion, nil
}

func (c *communion) Get(ctx context.Context, id string) (*dto.Communion, error) {
	c.log.Debug(ctx, "Getting communion request", zap.String("communion-id", id))

	// Parse communion ID
	communionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion ID")
		c.log.Error(ctx, "invalid communion ID", zap.Error(err), zap.String("communion-id", id))
		return nil, err
	}

	// Get communion with user information
	communion, err := c.communionStorage.GetWithUser(ctx, communionID)
	if err != nil {
		c.log.Error(ctx, "failed to get communion request", zap.Error(err))
		return nil, err
	}

	c.log.Debug(ctx, "Communion request retrieved successfully", zap.String("communion-id", id))
	return communion, nil
}

func (c *communion) GetUserCommunions(ctx context.Context, userID uuid.UUID, page, pageSize int) (*dto.CommunionListResponse, error) {
	c.log.Debug(ctx, "Getting user communion requests", 
		zap.String("user-id", userID.String()),
		zap.Int("page", page),
		zap.Int("page-size", pageSize))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	// Verify user exists
	_, err := c.userStorage.Get(ctx, userID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "user not found")
		c.log.Error(ctx, "user not found for communion requests", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Get communion requests from storage - for user requests we'll need to add this method
	// For now, we'll get all and filter (this should be optimized with a proper storage method)
	allCommunions, _, err := c.communionStorage.GetAll(ctx, 1, 1000) // Get all for filtering
	if err != nil {
		c.log.Error(ctx, "failed to get user communion requests", zap.Error(err))
		return nil, err
	}

	// Filter by user ID
	var userCommunions []dto.Communion
	for _, communion := range allCommunions {
		if communion.UserID == userID {
			userCommunions = append(userCommunions, communion)
		}
	}

	// Apply pagination to filtered results
	totalFiltered := int64(len(userCommunions))
	start := (page - 1) * pageSize
	end := start + pageSize
	
	if start >= len(userCommunions) {
		userCommunions = []dto.Communion{}
	} else {
		if end > len(userCommunions) {
			end = len(userCommunions)
		}
		userCommunions = userCommunions[start:end]
	}

	c.log.Debug(ctx, "User communion requests retrieved", 
		zap.String("user-id", userID.String()),
		zap.Int64("total", totalFiltered),
		zap.Int("returned", len(userCommunions)))

	return &dto.CommunionListResponse{
		Communions: userCommunions,
		Total:      totalFiltered,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (c *communion) GetAllCommunions(ctx context.Context, page, pageSize int) (*dto.CommunionListResponse, error) {
	c.log.Debug(ctx, "Getting all communion requests", zap.Int("page", page), zap.Int("page-size", pageSize))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	// Get communion requests from storage
	communions, total, err := c.communionStorage.GetAll(ctx, page, pageSize)
	if err != nil {
		c.log.Error(ctx, "failed to get all communion requests", zap.Error(err))
		return nil, err
	}

	c.log.Debug(ctx, "All communion requests retrieved", 
		zap.Int64("total", total),
		zap.Int("returned", len(communions)))

	return &dto.CommunionListResponse{
		Communions: communions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (c *communion) GetPendingCommunions(ctx context.Context, page, pageSize int) (*dto.CommunionListResponse, error) {
	c.log.Debug(ctx, "Getting pending communion requests", zap.Int("page", page), zap.Int("page-size", pageSize))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	// Get pending communion requests from storage
	communions, total, err := c.communionStorage.GetPending(ctx, page, pageSize)
	if err != nil {
		c.log.Error(ctx, "failed to get pending communion requests", zap.Error(err))
		return nil, err
	}

	c.log.Debug(ctx, "Pending communion requests retrieved", 
		zap.Int64("total", total),
		zap.Int("returned", len(communions)))

	return &dto.CommunionListResponse{
		Communions: communions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (c *communion) UpdateStatus(ctx context.Context, id string, adminUserID uuid.UUID, param dto.UpdateCommunionStatusRequest) (*dto.Communion, error) {
	c.log.Info(ctx, "Updating communion status", 
		zap.String("communion-id", id), 
		zap.String("admin-user-id", adminUserID.String()),
		zap.String("status", param.Status))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid status update request")
		c.log.Error(ctx, "invalid status update request", zap.Error(err))
		return nil, err
	}

	// Parse communion ID
	communionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion ID")
		c.log.Error(ctx, "invalid communion ID", zap.Error(err), zap.String("communion-id", id))
		return nil, err
	}

	// Verify admin user exists
	adminUser, err := c.userStorage.Get(ctx, adminUserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "admin user not found")
		c.log.Error(ctx, "admin user not found for communion status update", zap.Error(err), zap.String("admin-user-id", adminUserID.String()))
		return nil, err
	}

	// Verify admin has proper role (admin only)
	if adminUser.Role != "admin" {
		err = errors.ErrInvalidUserInput.New("only admins can update communion status")
		c.log.Warn(ctx, "unauthorized attempt to update communion status", 
			zap.String("user-id", adminUserID.String()),
			zap.String("user-role", adminUser.Role))
		return nil, err
	}

	// Get current communion to verify state
	currentCommunion, err := c.communionStorage.Get(ctx, communionID)
	if err != nil {
		c.log.Error(ctx, "failed to get communion for status update", zap.Error(err))
		return nil, err
	}

	// Check if communion is in a valid state for status update
	if currentCommunion.Status != dto.CommunionStatusPending {
		err = errors.ErrInvalidUserInput.New(fmt.Sprintf("can only update status of pending communion requests, current status: %s", currentCommunion.Status))
		c.log.Warn(ctx, "attempt to update non-pending communion status", 
			zap.String("communion-id", id),
			zap.String("current-status", string(currentCommunion.Status)))
		return nil, err
	}

	// Update communion status through storage layer
	updatedCommunion, err := c.communionStorage.UpdateStatus(ctx, communionID, param.Status, adminUserID)
	if err != nil {
		c.log.Error(ctx, "failed to update communion status", zap.Error(err))
		return nil, err
	}

	c.log.Info(ctx, "Communion status updated successfully", 
		zap.String("communion-id", id),
		zap.String("new-status", param.Status),
		zap.String("approved-by", adminUserID.String()))

	return updatedCommunion, nil
}

func (c *communion) Update(ctx context.Context, id string, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error) {
	c.log.Info(ctx, "Updating communion request", zap.String("communion-id", id), zap.String("user-id", userID.String()))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion update request")
		c.log.Error(ctx, "invalid communion update request", zap.Error(err))
		return nil, err
	}

	// Parse communion ID
	communionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion ID")
		c.log.Error(ctx, "invalid communion ID", zap.Error(err), zap.String("communion-id", id))
		return nil, err
	}

	// Get current communion to verify ownership and state
	currentCommunion, err := c.communionStorage.Get(ctx, communionID)
	if err != nil {
		c.log.Error(ctx, "failed to get communion for update", zap.Error(err))
		return nil, err
	}

	// Verify ownership (users can only update their own communion requests unless admin/manager)
	if currentCommunion.UserID != userID {
		err = errors.ErrInvalidUserInput.New("users can only update their own communion requests")
		c.log.Warn(ctx, "user attempting to update another user's communion request", 
			zap.String("communion-id", id),
			zap.String("communion-owner", currentCommunion.UserID.String()),
			zap.String("requesting-user", userID.String()))
		return nil, err
	}

	// Check if communion is in a state that allows updates (only pending can be updated)
	if currentCommunion.Status != dto.CommunionStatusPending {
		err = errors.ErrInvalidUserInput.New(fmt.Sprintf("can only update pending communion requests, current status: %s", currentCommunion.Status))
		c.log.Warn(ctx, "attempt to update non-pending communion", zap.String("communion-id", id), zap.String("status", string(currentCommunion.Status)))
		return nil, err
	}

	// Update communion request through storage layer
	updatedCommunion, err := c.communionStorage.Update(ctx, communionID, param)
	if err != nil {
		c.log.Error(ctx, "failed to update communion request", zap.Error(err))
		return nil, err
	}

	c.log.Info(ctx, "Communion request updated successfully", zap.String("communion-id", id))
	return updatedCommunion, nil
}

func (c *communion) Delete(ctx context.Context, id string) error {
	c.log.Info(ctx, "Deleting communion request", zap.String("communion-id", id))

	// Parse communion ID
	communionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion ID")
		c.log.Error(ctx, "invalid communion ID", zap.Error(err), zap.String("communion-id", id))
		return err
	}

	// Delete communion request through storage layer
	err = c.communionStorage.Delete(ctx, communionID)
	if err != nil {
		c.log.Error(ctx, "failed to delete communion request", zap.Error(err))
		return err
	}

	c.log.Info(ctx, "Communion request deleted successfully", zap.String("communion-id", id))
	return nil
}
