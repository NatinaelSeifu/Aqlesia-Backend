package available_slot

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

type availableSlot struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.AvailableSlot {
	return &availableSlot{
		db:  db,
		log: log,
	}
}

func (a *availableSlot) GetAvailableSlotsWithCapacity(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableSlot, error) {
	a.log.Debug(ctx, "Getting available slots with capacity",
		zap.Time("start-date", startDate),
		zap.Time("end-date", endDate))

	dbSlots, err := a.db.GetAvailableDatesWithCapacity(ctx, db.GetAvailableDatesWithCapacityParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		a.log.Error(ctx, "failed to get available slots with capacity", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get available slots")
	}

	slots := make([]dto.AvailableSlot, len(dbSlots))
	for i, dbSlot := range dbSlots {
		availableSpots := dbSlot.MaxCapacity - dbSlot.CurrentBookings
		if availableSpots < 0 {
			availableSpots = 0
		}

		// Handle nullable time fields
		var createdAt, updatedAt time.Time
		if dbSlot.CreatedAt.Valid {
			createdAt = dbSlot.CreatedAt.Time
		}
		if dbSlot.UpdatedAt.Valid {
			updatedAt = dbSlot.UpdatedAt.Time
		}

		slots[i] = dto.AvailableSlot{
			ID:              dbSlot.ID,
			SlotDate:        dbSlot.SlotDate,
			MaxCapacity:     dbSlot.MaxCapacity,
			CurrentBookings: dbSlot.CurrentBookings,
			AvailableSpots:  availableSpots,
			IsActive:        dbSlot.IsActive,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		}
	}

	a.log.Debug(ctx, "Successfully retrieved available slots", zap.Int("count", len(slots)))
	return slots, nil
}

func (a *availableSlot) UpsertSlot(ctx context.Context, slotDate time.Time, maxCapacity int32) error {
	a.log.Debug(ctx, "Upserting slot",
		zap.Time("slot-date", slotDate),
		zap.Int32("max-capacity", maxCapacity))

	_, err := a.db.UpsertAvailableDate(ctx, db.UpsertAvailableDateParams{
		SlotDate:        slotDate,
		MaxCapacity:     maxCapacity,
		CurrentBookings: 0, // New slots start with 0 bookings
		IsActive:        true, // New slots are active by default
	})
	if err != nil {
		a.log.Error(ctx, "failed to upsert slot", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to upsert slot")
	}

	a.log.Debug(ctx, "Successfully upserted slot", zap.Time("slot-date", slotDate))
	return nil
}

func (a *availableSlot) DeactivateSlot(ctx context.Context, slotDate time.Time) error {
	a.log.Debug(ctx, "Deactivating slot", zap.Time("slot-date", slotDate))

	_, err := a.db.DeactivateDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to deactivate slot", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to deactivate slot")
	}

	a.log.Debug(ctx, "Successfully deactivated slot", zap.Time("slot-date", slotDate))
	return nil
}

func (a *availableSlot) ActivateSlot(ctx context.Context, slotDate time.Time) error {
	a.log.Debug(ctx, "Activating slot", zap.Time("slot-date", slotDate))

	_, err := a.db.ActivateDate(ctx, slotDate)
	if err != nil {
		a.log.Error(ctx, "failed to activate slot", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to activate slot")
	}

	a.log.Debug(ctx, "Successfully activated slot", zap.Time("slot-date", slotDate))
	return nil
}

func (a *availableSlot) DeleteOldSlots(ctx context.Context, beforeDate time.Time) error {
	a.log.Debug(ctx, "Deleting old slots", zap.Time("before-date", beforeDate))

	err := a.db.DeleteOldDates(ctx, beforeDate)
	if err != nil {
		a.log.Error(ctx, "failed to delete old slots", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to delete old slots")
	}

	a.log.Debug(ctx, "Successfully deleted old slots", zap.Time("before-date", beforeDate))
	return nil
}

func (a *availableSlot) GetSlotByDate(ctx context.Context, slotDate time.Time) (*dto.AvailableSlot, error) {
	a.log.Debug(ctx, "Getting slot by date", zap.Time("slot-date", slotDate))

	dbSlot, err := a.db.GetAvailableDate(ctx, slotDate)
	if err != nil {
		if err == sql.ErrNoRows {
			a.log.Debug(ctx, "slot not found", zap.Time("slot-date", slotDate))
			return nil, errors.ErrNoRecordFound.New("slot not found")
		}
		a.log.Error(ctx, "failed to get slot by date", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get slot")
	}

	availableSpots := dbSlot.MaxCapacity - dbSlot.CurrentBookings
	if availableSpots < 0 {
		availableSpots = 0
	}

	// Handle nullable time fields
	var createdAt, updatedAt time.Time
	if dbSlot.CreatedAt.Valid {
		createdAt = dbSlot.CreatedAt.Time
	}
	if dbSlot.UpdatedAt.Valid {
		updatedAt = dbSlot.UpdatedAt.Time
	}

	slot := &dto.AvailableSlot{
		ID:              dbSlot.ID,
		SlotDate:        dbSlot.SlotDate,
		MaxCapacity:     dbSlot.MaxCapacity,
		CurrentBookings: dbSlot.CurrentBookings,
		AvailableSpots:  availableSpots,
		IsActive:        dbSlot.IsActive,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	a.log.Debug(ctx, "Successfully retrieved slot", zap.Time("slot-date", slotDate))
	return slot, nil
}
