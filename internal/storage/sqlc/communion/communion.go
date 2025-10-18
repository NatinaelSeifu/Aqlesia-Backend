package communion

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/db"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type communion struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.Communion {
	return &communion{
		db:  db,
		log: log,
	}
}

func (c *communion) Create(ctx context.Context, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error) {
	// Parse communion date
	communionDate, err := param.GetCommunionDate()
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion date")
		c.log.Error(ctx, "unable to parse communion date", zap.Error(err))
		return nil, err
	}



	// Create the communion request
	communionDB, err := c.db.CreateCommunion(ctx, db.CreateCommunionParams{
		UserID:       userID,
		CommunionDate: communionDate,
		Status:       string(dto.CommunionStatusPending),
		RequestedAt:  time.Now(),
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not create communion")
		c.log.Error(ctx, "unable to create communion", zap.Error(err), zap.Any("communion", param))
		return nil, err
	}

	return c.convertToDTO(communionDB), nil
}

func (c *communion) Get(ctx context.Context, id uuid.UUID) (*dto.Communion, error) {
	communionDB, err := c.db.GetCommunion(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "communion not found")
			c.log.Info(ctx, "Communion not found", zap.String("communion-id", id.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read communion")
		c.log.Error(ctx, "unable to get communion", zap.Error(err), zap.String("communion-id", id.String()))
		return nil, err
	}

	return c.convertToDTO(communionDB), nil
}

func (c *communion) GetWithUser(ctx context.Context, id uuid.UUID) (*dto.Communion, error) {
	communionWithUser, err := c.db.GetCommunionWithUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "communion not found")
			c.log.Info(ctx, "Communion with user not found", zap.String("communion-id", id.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read communion with user")
		c.log.Error(ctx, "unable to get communion with user", zap.Error(err), zap.String("communion-id", id.String()))
		return nil, err
	}

	return c.convertWithUserRowToDTO(communionWithUser), nil
}

func (c *communion) GetUserCommunion(ctx context.Context, userID uuid.UUID) (*dto.Communion, error) {
	communionWithDetails, err := c.db.GetUserCommunionWithDetails(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No communion found for user
		}
		err = errors.ErrReadError.Wrap(err, "could not read user communion")
		c.log.Error(ctx, "unable to get user communion", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	return c.convertUserCommunionRowToDTO(communionWithDetails), nil
}

func (c *communion) GetAll(ctx context.Context, page, pageSize int) ([]dto.Communion, int64, error) {
	offset := (page - 1) * pageSize

	communions, err := c.db.GetAllCommunions(ctx, db.GetAllCommunionsParams{
		Offset: int32(offset),
		Limit:  int32(pageSize),
	})
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read all communions")
		c.log.Error(ctx, "unable to get all communions", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total, err := c.db.CountAllCommunions(ctx)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count all communions")
		c.log.Error(ctx, "unable to count all communions", zap.Error(err))
		return nil, 0, err
	}

	// Convert to DTOs
	communionDTOs := make([]dto.Communion, len(communions))
	for i, communion := range communions {
		communionDTOs[i] = *c.convertAllCommunionsRowToDTO(communion)
	}

	return communionDTOs, total, nil
}

func (c *communion) GetPending(ctx context.Context, page, pageSize int) ([]dto.Communion, int64, error) {
	offset := (page - 1) * pageSize

	communions, err := c.db.GetPendingCommunions(ctx, db.GetPendingCommunionsParams{
		Offset: int32(offset),
		Limit:  int32(pageSize),
	})
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read pending communions")
		c.log.Error(ctx, "unable to get pending communions", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total, err := c.db.CountPendingCommunions(ctx)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count pending communions")
		c.log.Error(ctx, "unable to count pending communions", zap.Error(err))
		return nil, 0, err
	}

	// Convert to DTOs
	communionDTOs := make([]dto.Communion, len(communions))
	for i, communion := range communions {
		communionDTOs[i] = *c.convertPendingCommunionRowToDTO(communion)
	}

	return communionDTOs, total, nil
}

func (c *communion) UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedByUserID uuid.UUID) (*dto.Communion, error) {
	updatedCommunion, err := c.db.UpdateCommunionStatus(ctx, db.UpdateCommunionStatusParams{
		Status:              status,
		ApprovedByUserID:    uuid.NullUUID{UUID: approvedByUserID, Valid: true},
		ID:                  id,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "communion not found")
			c.log.Info(ctx, "Communion not found for status update", zap.String("communion-id", id.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not update communion status")
		c.log.Error(ctx, "unable to update communion status", zap.Error(err), zap.String("communion-id", id.String()))
		return nil, err
	}

	return c.convertToDTO(updatedCommunion), nil
}

func (c *communion) Update(ctx context.Context, id uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error) {
	// Parse communion date
	communionDate, err := param.GetCommunionDate()
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid communion date")
		c.log.Error(ctx, "unable to parse communion date", zap.Error(err))
		return nil, err
	}

	updatedCommunion, err := c.db.UpdateCommunion(ctx, db.UpdateCommunionParams{
		CommunionDate: communionDate,
		ID:           id,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "communion not found")
			c.log.Info(ctx, "Communion not found for update", zap.String("communion-id", id.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not update communion")
		c.log.Error(ctx, "unable to update communion", zap.Error(err), zap.String("communion-id", id.String()))
		return nil, err
	}

	return c.convertToDTO(updatedCommunion), nil
}

func (c *communion) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := c.db.DeleteCommunion(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "communion not found")
			c.log.Info(ctx, "Communion not found for deletion", zap.String("communion-id", id.String()))
			return err
		}
		err = errors.ErrWriteError.Wrap(err, "could not delete communion")
		c.log.Error(ctx, "unable to delete communion", zap.Error(err), zap.String("communion-id", id.String()))
		return err
	}

	return nil
}

// Helper methods to convert database models to DTOs

func (c *communion) convertToDTO(communionDB db.Communion) *dto.Communion {
	var approvedAt *time.Time
	if communionDB.ApprovedAt.Valid {
		approvedAt = &communionDB.ApprovedAt.Time
	}

	var approvedByUserID *uuid.UUID
	if communionDB.ApprovedByUserID.Valid {
		approvedByUserID = &communionDB.ApprovedByUserID.UUID
	}

	var deletedAt *time.Time

	createdAt := communionDB.CreatedAt
	updatedAt := communionDB.UpdatedAt
	if communionDB.DeletedAt.Valid {
		deletedAt = &communionDB.DeletedAt.Time
	}

	return &dto.Communion{
		ID:               communionDB.ID,
		UserID:           communionDB.UserID,
		CommunionDate:    communionDB.CommunionDate,
		Status:           dto.CommunionStatus(communionDB.Status),
		RequestedAt:      communionDB.RequestedAt,
		ApprovedAt:       approvedAt,
		ApprovedByUserID: approvedByUserID,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedAt,
	}
}

func (c *communion) convertWithUserRowToDTO(row db.GetCommunionWithUserRow) *dto.Communion {
	var approvedAt *time.Time
	if row.ApprovedAt.Valid {
		approvedAt = &row.ApprovedAt.Time
	}

	var approvedByUserID *uuid.UUID
	var approvedBy *dto.User
	if row.ApprovedByUserID.Valid {
		approvedByUserID = &row.ApprovedByUserID.UUID
		if row.ApprovedByName.Valid && row.ApprovedByLastname.Valid {
			approvedBy = &dto.User{
				ID:       row.ApprovedByUserID.UUID,
				Name:     row.ApprovedByName.String,
				LastName: row.ApprovedByLastname.String,
			}
		}
	}

	var deletedAt *time.Time

	createdAt := row.CreatedAt
	updatedAt := row.UpdatedAt
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Communion{
		ID:               row.ID,
		UserID:           row.UserID,
		CommunionDate:    row.CommunionDate,
		Status:           dto.CommunionStatus(row.Status),
		RequestedAt:      row.RequestedAt,
		ApprovedAt:       approvedAt,
		ApprovedByUserID: approvedByUserID,
		ApprovedBy:       approvedBy,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}

func (c *communion) convertUserCommunionRowToDTO(row db.GetUserCommunionWithDetailsRow) *dto.Communion {
	var approvedAt *time.Time
	if row.ApprovedAt.Valid {
		approvedAt = &row.ApprovedAt.Time
	}

	var approvedByUserID *uuid.UUID
	var approvedBy *dto.User
	if row.ApprovedByUserID.Valid {
		approvedByUserID = &row.ApprovedByUserID.UUID
		if row.ApprovedByName.Valid && row.ApprovedByLastname.Valid {
			approvedBy = &dto.User{
				ID:       row.ApprovedByUserID.UUID,
				Name:     row.ApprovedByName.String,
				LastName: row.ApprovedByLastname.String,
			}
		}
	}

	var deletedAt *time.Time

	createdAt := row.CreatedAt
	updatedAt := row.UpdatedAt
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Communion{
		ID:               row.ID,
		UserID:           row.UserID,
		CommunionDate:    row.CommunionDate,
		Status:           dto.CommunionStatus(row.Status),
		RequestedAt:      row.RequestedAt,
		ApprovedAt:       approvedAt,
		ApprovedByUserID: approvedByUserID,
		ApprovedBy:       approvedBy,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedAt,
	}
}

func (c *communion) convertAllCommunionsRowToDTO(row db.GetAllCommunionsRow) *dto.Communion {
	var approvedAt *time.Time
	if row.ApprovedAt.Valid {
		approvedAt = &row.ApprovedAt.Time
	}

	var approvedByUserID *uuid.UUID
	var approvedBy *dto.User
	if row.ApprovedByUserID.Valid {
		approvedByUserID = &row.ApprovedByUserID.UUID
		if row.ApprovedByName.Valid && row.ApprovedByLastname.Valid {
			approvedBy = &dto.User{
				ID:       row.ApprovedByUserID.UUID,
				Name:     row.ApprovedByName.String,
				LastName: row.ApprovedByLastname.String,
			}
		}
	}

	var deletedAt *time.Time

	createdAt := row.CreatedAt
	updatedAt := row.UpdatedAt
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Communion{
		ID:               row.ID,
		UserID:           row.UserID,
		CommunionDate:    row.CommunionDate,
		Status:           dto.CommunionStatus(row.Status),
		RequestedAt:      row.RequestedAt,
		ApprovedAt:       approvedAt,
		ApprovedByUserID: approvedByUserID,
		ApprovedBy:       approvedBy,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}

func (c *communion) convertPendingCommunionRowToDTO(row db.GetPendingCommunionsRow) *dto.Communion {
	var approvedAt *time.Time
	if row.ApprovedAt.Valid {
		approvedAt = &row.ApprovedAt.Time
	}

	var approvedByUserID *uuid.UUID
	if row.ApprovedByUserID.Valid {
		approvedByUserID = &row.ApprovedByUserID.UUID
	}

	var deletedAt *time.Time

	createdAt := row.CreatedAt
	updatedAt := row.UpdatedAt
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Communion{
		ID:               row.ID,
		UserID:           row.UserID,
		CommunionDate:    row.CommunionDate,
		Status:           dto.CommunionStatus(row.Status),
		RequestedAt:      row.RequestedAt,
		ApprovedAt:       approvedAt,
		ApprovedByUserID: approvedByUserID,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}
