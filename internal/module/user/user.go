package user

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type user struct {
	userPersistent storage.User
	log            logger.Logger
}

func Init(log logger.Logger, userPersistent storage.User) module.User {
	return &user{
		userPersistent: userPersistent,
		log:            log,
	}
}

func (u *user) Create(ctx context.Context, param dto.RegisterUser) (*dto.User, error) {
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	exist, err := u.userPersistent.CheckUserExists(ctx, param)
	if err != nil {
		return nil, err
	} else if exist {
		err = errors.ErrDataExists.New("user with this phone number already exists")
		u.log.Error(ctx, "duplicated data", zap.String("user-phone", param.PhoneNumber))
		return nil, err
	}

	return u.userPersistent.Create(ctx, param)
}

func (u *user) DeleteUser(ctx context.Context, id string) error {
	userId, err := uuid.Parse(id)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed", zap.Error(err), zap.String("user-id", id))
		return err
	}

	return u.userPersistent.DeleteUser(ctx, userId)
}

func (u *user) Update(ctx context.Context, id string, param dto.UpdateUser) (*dto.User, error) {
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	uuidID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed", zap.Error(err), zap.String("user-id", id))
		return nil, err
	}

	return u.userPersistent.Update(ctx, uuidID, param)
}

func (u *user) Get(ctx context.Context, id string) (*dto.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed", zap.Error(err), zap.String("user-id", id))
		return nil, err
	}

	return u.userPersistent.Get(ctx, uuidID)
}

func (u *user) GetAll(ctx context.Context, page, pageSize int) (*dto.UserListResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	users, total, err := u.userPersistent.GetAll(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &dto.UserListResponse{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *user) ChangePassword(ctx context.Context, userID string, param dto.ChangePasswordRequest) (*dto.User, error) {
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		u.log.Error(ctx, "validation failed for password change", zap.Error(err), zap.String("user-id", userID))
		return nil, err
	}

	uuidID, err := uuid.Parse(userID)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed", zap.Error(err), zap.String("user-id", userID))
		return nil, err
	}

	// Check if new password is different from current password
	if param.CurrentPassword == param.NewPassword {
		err := errors.ErrInvalidUserInput.New("new password must be different from current password")
		u.log.Warn(ctx, "user tried to set same password", zap.String("user-id", userID))
		return nil, err
	}

	return u.userPersistent.ChangePassword(ctx, uuidID, param.CurrentPassword, param.NewPassword)
}

func (u *user) GetByStatus(ctx context.Context, status string, page, pageSize int) (*dto.UserListResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Validate status
	if status != "PENDING" && status != "ACTIVE" && status != "INACTIVE" {
		err := errors.ErrInvalidUserInput.New("invalid status: must be PENDING, ACTIVE, or INACTIVE")
		u.log.Error(ctx, "invalid status provided", zap.String("status", status))
		return nil, err
	}

	users, total, err := u.userPersistent.GetByStatus(ctx, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &dto.UserListResponse{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *user) UpdateStatus(ctx context.Context, id string, status string) (*dto.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed", zap.Error(err), zap.String("user-id", id))
		return nil, err
	}

	// Validate status
	if status != "ACTIVE" && status != "INACTIVE" {
		err := errors.ErrInvalidUserInput.New("invalid status: must be ACTIVE or INACTIVE")
		u.log.Error(ctx, "invalid status for update", zap.String("status", status), zap.String("user-id", id))
		return nil, err
	}

	return u.userPersistent.UpdateStatus(ctx, uuidID, status)
}

func (u *user) UpdateProfileImage(ctx context.Context, id string, imageURL string) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user id")
		u.log.Error(ctx, "parsing user id failed for profile image update", zap.Error(err), zap.String("user-id", id))
		return err
	}
	return u.userPersistent.UpdateProfileImage(ctx, uuidID, imageURL)
}
