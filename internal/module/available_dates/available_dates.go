package available_dates

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
	"github.com/joomcode/errorx"
)

type availableDates struct {
	availableDatesPersistent storage.AvailableDates
	log                      logger.Logger
}

func Init(log logger.Logger, availableDatesPersistent storage.AvailableDates) module.AvailableDates {
	return &availableDates{
		availableDatesPersistent: availableDatesPersistent,
		log:                      log,
	}
}

// GetAvailableDates returns available dates for regular users (booking)
func (a *availableDates) GetAvailableDates(ctx context.Context, startDate, endDate time.Time, onlyAvailable bool) ([]dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting available dates for booking",
		zap.Time("start-date", startDate),
		zap.Time("end-date", endDate),
		zap.Bool("only-available", onlyAvailable))

	// Validate date range
	if endDate.Before(startDate) {
		err := errors.ErrInvalidUserInput.New("end date must be after start date")
		a.log.Warn(ctx, "invalid date range", zap.Time("start", startDate), zap.Time("end", endDate))
		return nil, err
	}

	// Limit range to prevent abuse (max 3 months)
	maxRange := startDate.AddDate(0, 3, 0)
	if endDate.After(maxRange) {
		err := errors.ErrInvalidUserInput.New("date range cannot exceed 3 months")
		a.log.Warn(ctx, "date range too large", zap.Time("end", endDate), zap.Time("max", maxRange))
		return nil, err
	}

	var dates []dto.AvailableDate
	var err error

	if onlyAvailable {
		dates, err = a.availableDatesPersistent.GetAvailableDatesWithCapacity(ctx, startDate, endDate)
	} else {
		dates, err = a.availableDatesPersistent.GetAvailableDates(ctx, startDate, endDate)
	}

	if err != nil {
		a.log.Error(ctx, "failed to get available dates", zap.Error(err))
		return nil, err
	}

	a.log.Debug(ctx, "successfully retrieved available dates", zap.Int("count", len(dates)))
	return dates, nil
}

// GetAllAvailableDates returns all dates including inactive ones for admin/manager
func (a *availableDates) GetAllAvailableDates(ctx context.Context, query dto.AvailableDateQuery) ([]dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting all available dates for admin/manager", zap.Any("query", query))

	if err := query.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid query parameters")
		a.log.Warn(ctx, "validation failed", zap.Error(err), zap.Any("query", query))
		return nil, err
	}

	// Limit range to prevent abuse (max 1 year for admins)
	maxRange := query.StartDate.AddDate(1, 0, 0)
	if query.EndDate.After(maxRange) {
		err := errors.ErrInvalidUserInput.New("date range cannot exceed 1 year")
		a.log.Warn(ctx, "admin date range too large", zap.Time("end", query.EndDate), zap.Time("max", maxRange))
		return nil, err
	}

	var dates []dto.AvailableDate
	var err error

	if query.OnlyAvailable {
		dates, err = a.availableDatesPersistent.GetAvailableDatesWithCapacity(ctx, query.StartDate, query.EndDate)
	} else if query.IncludeInactive {
		dates, err = a.availableDatesPersistent.GetAllAvailableDates(ctx, query.StartDate, query.EndDate)
	} else {
		dates, err = a.availableDatesPersistent.GetAvailableDates(ctx, query.StartDate, query.EndDate)
	}

	if err != nil {
		a.log.Error(ctx, "failed to get all available dates", zap.Error(err))
		return nil, err
	}

	a.log.Debug(ctx, "successfully retrieved all available dates", zap.Int("count", len(dates)))
	return dates, nil
}

// CreateAvailableDate creates a new available date (admin/manager only)
func (a *availableDates) CreateAvailableDate(ctx context.Context, param dto.CreateAvailableDate) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Creating available date", zap.Any("param", param))

	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	// Check if date already exists
	existing, err := a.availableDatesPersistent.GetAvailableDateByDate(ctx, param.SlotDate.ToTime())
	if err != nil {
		// If it's not a "not found" error, it's a real error
		if !isNoRecordFoundError(err) {
			a.log.Error(ctx, "failed to check existing date", zap.Error(err))
			return nil, err
		}
		// If it's a "not found" error, that's what we want (date doesn't exist yet)
	}
	if existing != nil {
		err = errors.ErrDataExists.New("available date already exists for this date")
		a.log.Warn(ctx, "duplicate date", zap.Time("date", param.SlotDate.ToTime()))
		return nil, err
	}

	result, err := a.availableDatesPersistent.CreateAvailableDate(ctx, param)
	if err != nil {
		a.log.Error(ctx, "failed to create available date", zap.Error(err))
		return nil, err
	}

	a.log.Info(ctx, "successfully created available date", zap.Time("date", param.SlotDate.ToTime()))
	return result, nil
}

// UpdateAvailableDate updates an existing available date (admin/manager only)
func (a *availableDates) UpdateAvailableDate(ctx context.Context, dateStr string, param dto.UpdateAvailableDate) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Updating available date", zap.String("date", dateStr), zap.Any("param", param))

	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	// Parse date
	slotDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid date format, expected YYYY-MM-DD")
		a.log.Error(ctx, "date parsing failed", zap.Error(err), zap.String("date", dateStr))
		return nil, err
	}

	// Check if date exists
	existing, err := a.availableDatesPersistent.GetAvailableDateByDate(ctx, slotDate)
	if err != nil {
		if isNoRecordFoundError(err) {
			err = errors.ErrNoRecordFound.New("available date not found")
			a.log.Warn(ctx, "date not found", zap.Time("date", slotDate))
			return nil, err
		}
		a.log.Error(ctx, "failed to get existing date", zap.Error(err))
		return nil, err
	}

	// Business rule: cannot reduce capacity below current bookings
	if param.MaxCapacity != nil && *param.MaxCapacity < existing.CurrentBookings {
		err = errors.ErrInvalidUserInput.New("cannot reduce capacity below current bookings")
		a.log.Warn(ctx, "capacity reduction not allowed", 
			zap.Int32("requested", *param.MaxCapacity),
			zap.Int32("current-bookings", existing.CurrentBookings))
		return nil, err
	}

	result, err := a.availableDatesPersistent.UpdateAvailableDate(ctx, slotDate, param)
	if err != nil {
		a.log.Error(ctx, "failed to update available date", zap.Error(err))
		return nil, err
	}

	a.log.Info(ctx, "successfully updated available date", zap.Time("date", slotDate))
	return result, nil
}

// DeleteAvailableDate deletes an available date (admin/manager only)
func (a *availableDates) DeleteAvailableDate(ctx context.Context, dateStr string) error {
	a.log.Debug(ctx, "Deleting available date", zap.String("date", dateStr))

	// Parse date
	slotDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid date format, expected YYYY-MM-DD")
		a.log.Error(ctx, "date parsing failed", zap.Error(err), zap.String("date", dateStr))
		return err
	}

	// Check if date exists and has bookings
	existing, err := a.availableDatesPersistent.GetAvailableDateByDate(ctx, slotDate)
	if err != nil {
		if isNoRecordFoundError(err) {
			err = errors.ErrNoRecordFound.New("available date not found")
			a.log.Warn(ctx, "date not found", zap.Time("date", slotDate))
			return err
		}
		a.log.Error(ctx, "failed to get existing date", zap.Error(err))
		return err
	}

	// Business rule: cannot delete dates with active bookings
	if existing.CurrentBookings > 0 {
		err = errors.ErrInvalidUserInput.New("cannot delete date with active bookings")
		a.log.Warn(ctx, "deletion not allowed - has bookings", 
			zap.Time("date", slotDate),
			zap.Int32("bookings", existing.CurrentBookings))
		return err
	}

	err = a.availableDatesPersistent.DeleteAvailableDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to delete available date", zap.Error(err))
		return err
	}

	a.log.Info(ctx, "successfully deleted available date", zap.Time("date", slotDate))
	return nil
}

// GetAvailableDateByDate gets a specific available date
func (a *availableDates) GetAvailableDateByDate(ctx context.Context, dateStr string) (*dto.AvailableDate, error) {
	a.log.Debug(ctx, "Getting available date by date", zap.String("date", dateStr))

	// Parse date
	slotDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid date format, expected YYYY-MM-DD")
		a.log.Error(ctx, "date parsing failed", zap.Error(err), zap.String("date", dateStr))
		return nil, err
	}

	result, err := a.availableDatesPersistent.GetAvailableDateByDate(ctx, slotDate)
	if err != nil {
		if isNoRecordFoundError(err) {
			a.log.Debug(ctx, "date not found", zap.Time("date", slotDate))
			return nil, err
		}
		a.log.Error(ctx, "failed to get available date", zap.Error(err))
		return nil, err
	}

	a.log.Debug(ctx, "successfully retrieved available date", zap.Time("date", slotDate))
	return result, nil
}

// Helper function to check if an error is a "no record found" error
func isNoRecordFoundError(err error) bool {
	// Check for our custom error type
	if errorx.IsOfType(err, errors.ErrNoRecordFound) {
		return true
	}
	// Check for SQL no rows error
	if err == sql.ErrNoRows {
		return true
	}
	return false
}

// Helper function to check if an error is an "invalid user input" error
func isInvalidUserInputError(err error) bool {
	return errorx.IsOfType(err, errors.ErrInvalidUserInput)
}

// Helper function to check if an error is a "data exists" error
func isDataExistsError(err error) bool {
	return errorx.IsOfType(err, errors.ErrDataExists)
}
