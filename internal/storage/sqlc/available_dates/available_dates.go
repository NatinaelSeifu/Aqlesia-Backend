package available_dates

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

	"go.uber.org/zap"
)

type availableDates struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.AvailableDates {
	return &availableDates{
		db:  db,
		log: log,
	}
}

// GetAvailableDatesWithCapacity returns dates that have available capacity for booking
func (a *availableDates) GetAvailableDatesWithCapacity(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting available dates with capacity",
		zap.Time("start-date", startDate),
		zap.Time("end-date", endDate))

	dbDates, err := a.db.GetAvailableDatesWithCapacity(ctx, db.GetAvailableDatesWithCapacityParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		a.log.Error(ctx, "failed to get available dates with capacity", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get available dates")
	}

	dates := a.convertToDTO(dbDates)
	a.log.Debug(ctx, "Successfully retrieved available dates", zap.Int("count", len(dates)))
	return dates, nil
}

// GetAvailableDates returns all active dates in a range
func (a *availableDates) GetAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting available dates",
		zap.Time("start-date", startDate),
		zap.Time("end-date", endDate))

	dbDates, err := a.db.GetAvailableDates(ctx, db.GetAvailableDatesParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		a.log.Error(ctx, "failed to get available dates", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get available dates")
	}

	dates := a.convertToDTO(dbDates)
	a.log.Debug(ctx, "Successfully retrieved available dates", zap.Int("count", len(dates)))
	return dates, nil
}

// GetAllAvailableDates returns all dates (active and inactive) for admin/manager use
func (a *availableDates) GetAllAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting all available dates",
		zap.Time("start-date", startDate),
		zap.Time("end-date", endDate))

	dbDates, err := a.db.GetAllAvailableDates(ctx, db.GetAllAvailableDatesParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		a.log.Error(ctx, "failed to get all available dates", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get all available dates")
	}

	dates := a.convertToDTO(dbDates)
	a.log.Debug(ctx, "Successfully retrieved all available dates", zap.Int("count", len(dates)))
	return dates, nil
}

// CreateAvailableDate creates a new available date
func (a *availableDates) CreateAvailableDate(ctx context.Context, param dto.CreateAvailableDate) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Creating available date",
		zap.Time("slot-date", param.SlotDate.ToTime()),
		zap.Int32("max-capacity", param.MaxCapacity))

	dbDate, err := a.db.CreateAvailableDate(ctx, db.CreateAvailableDateParams{
		SlotDate:        param.SlotDate.ToTime(),
		MaxCapacity:     param.MaxCapacity,
		CurrentBookings: 0, // New dates start with 0 bookings
		IsActive:        param.GetIsActive(),
	})
	if err != nil {
		a.log.Error(ctx, "failed to create available date", zap.Error(err))
		return nil, errors.ErrWriteError.Wrap(err, "failed to create available date")
	}

	result := a.convertSingleToDTO(dbDate)
	a.log.Debug(ctx, "Successfully created available date", zap.Time("slot-date", param.SlotDate.ToTime()))
	return &result, nil
}

// UpdateAvailableDate updates an existing available date
func (a *availableDates) UpdateAvailableDate(ctx context.Context, slotDate time.Time, param dto.UpdateAvailableDate) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Updating available date",
		zap.Time("slot-date", slotDate),
		zap.Any("params", param))

	// Get current date to preserve values not being updated
	current, err := a.GetAvailableDateByDate(ctx, slotDate)
	if err != nil {
		return nil, err
	}

	maxCapacity := current.MaxCapacity
	if param.MaxCapacity != nil {
		maxCapacity = *param.MaxCapacity
	}

	isActive := current.IsActive
	if param.IsActive != nil {
		isActive = *param.IsActive
	}

	dbDate, err := a.db.UpdateAvailableDate(ctx, db.UpdateAvailableDateParams{
		SlotDate:    slotDate,
		MaxCapacity: maxCapacity,
		IsActive:    isActive,
	})
	if err != nil {
		a.log.Error(ctx, "failed to update available date", zap.Error(err))
		return nil, errors.ErrWriteError.Wrap(err, "failed to update available date")
	}

	result := a.convertSingleToDTO(dbDate)
	a.log.Debug(ctx, "Successfully updated available date", zap.Time("slot-date", slotDate))
	return &result, nil
}

// DeleteAvailableDate deletes an available date
func (a *availableDates) DeleteAvailableDate(ctx context.Context, slotDate time.Time) error {
	a.log.Debug(ctx, "Deleting available date", zap.Time("slot-date", slotDate))

	err := a.db.DeleteAvailableDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to delete available date", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to delete available date")
	}

	a.log.Debug(ctx, "Successfully deleted available date", zap.Time("slot-date", slotDate))
	return nil
}

// GetAvailableDateByDate gets a specific available date
func (a *availableDates) GetAvailableDateByDate(ctx context.Context, slotDate time.Time) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting available date by date", zap.Time("slot-date", slotDate))

	dbDate, err := a.db.GetAvailableDate(ctx, slotDate)
	if err != nil {
		if err == sql.ErrNoRows {
			a.log.Debug(ctx, "available date not found", zap.Time("slot-date", slotDate))
			return nil, errors.ErrNoRecordFound.New("available date not found")
		}
		a.log.Error(ctx, "failed to get available date by date", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get available date")
	}

	result := a.convertSingleToDTO(dbDate)
	a.log.Debug(ctx, "Successfully retrieved available date", zap.Time("slot-date", slotDate))
	return &result, nil
}

// UpsertAvailableDate creates or updates an available date
func (a *availableDates) UpsertAvailableDate(ctx context.Context, slotDate time.Time, maxCapacity int32) error {
	a.log.Debug(ctx, "Upserting available date",
		zap.Time("slot-date", slotDate),
		zap.Int32("max-capacity", maxCapacity))

	_, err := a.db.UpsertAvailableDate(ctx, db.UpsertAvailableDateParams{
		SlotDate:        slotDate,
		MaxCapacity:     maxCapacity,
		CurrentBookings: 0, // New dates start with 0 bookings
		IsActive:        true, // New dates are active by default
	})
	if err != nil {
		a.log.Error(ctx, "failed to upsert available date", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to upsert available date")
	}

	a.log.Debug(ctx, "Successfully upserted available date", zap.Time("slot-date", slotDate))
	return nil
}

// DeactivateDate deactivates an available date
func (a *availableDates) DeactivateDate(ctx context.Context, slotDate time.Time) error {
	a.log.Debug(ctx, "Deactivating available date", zap.Time("slot-date", slotDate))

	_, err := a.db.DeactivateDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to deactivate available date", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to deactivate available date")
	}

	a.log.Debug(ctx, "Successfully deactivated available date", zap.Time("slot-date", slotDate))
	return nil
}

// ActivateDate activates an available date
func (a *availableDates) ActivateDate(ctx context.Context, slotDate time.Time) error {
	a.log.Debug(ctx, "Activating available date", zap.Time("slot-date", slotDate))

	_, err := a.db.ActivateDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to activate available date", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to activate available date")
	}

	a.log.Debug(ctx, "Successfully activated available date", zap.Time("slot-date", slotDate))
	return nil
}

// DeleteOldDates deletes old available dates
func (a *availableDates) DeleteOldDates(ctx context.Context, beforeDate time.Time) error {
	a.log.Debug(ctx, "Deleting old available dates", zap.Time("before-date", beforeDate))

	err := a.db.DeleteOldDates(ctx, beforeDate)
	if err != nil {
		a.log.Error(ctx, "failed to delete old available dates", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to delete old available dates")
	}

	a.log.Debug(ctx, "Successfully deleted old available dates", zap.Time("before-date", beforeDate))
	return nil
}

// Helper functions to convert database models to DTOs
func (a *availableDates) convertToDTO(dbDates []db.AvailableDate) []dto.AvailableDate {
	dates := make([]dto.AvailableDate, len(dbDates))
	for i, dbDate := range dbDates {
		dates[i] = a.convertSingleToDTO(dbDate)
	}
	return dates
}

func (a *availableDates) convertSingleToDTO(dbDate db.AvailableDate) dto.AvailableDate {
	availableSpots := dbDate.MaxCapacity - dbDate.CurrentBookings
	if availableSpots < 0 {
		availableSpots = 0
	}

	// Handle nullable time fields
	var createdAt, updatedAt time.Time
	if dbDate.CreatedAt.Valid {
		createdAt = dbDate.CreatedAt.Time
	}
	if dbDate.UpdatedAt.Valid {
		updatedAt = dbDate.UpdatedAt.Time
	}

	return dto.AvailableDate{
		ID:              dbDate.ID,
		SlotDate:        dbDate.SlotDate,
		MaxCapacity:     dbDate.MaxCapacity,
		CurrentBookings: dbDate.CurrentBookings,
		AvailableSpots:  availableSpots,
		IsActive:        dbDate.IsActive,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}
