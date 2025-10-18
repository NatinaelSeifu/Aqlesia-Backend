package initiator

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/storage"
	"aqlesia/internal/storage/sqlc/available_slot"
	"aqlesia/platform/logger"
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	// MaxAppointmentsPerDay defines the daily capacity for appointments read from config
	MaxAppointmentsPerDay = 10
	// SeedWeeks defines how many weeks ahead to seed slots
	SeedWeeks = 2
)

type CronScheduler struct {
	cron        *cron.Cron
	slotStorage storage.AvailableSlot
	log         logger.Logger
}

// InitCronScheduler initializes the cron scheduler for appointment slots
func InitCronScheduler(db dbinstance.DBInstance, log logger.Logger) *CronScheduler {
	slotStorage := available_slot.Init(db, log.Named("slot-storage"))

	// Create cron instance with timezone support
	c := cron.New(cron.WithLocation(time.Local))

	scheduler := &CronScheduler{
		cron:        c,
		slotStorage: slotStorage,
		log:         log.Named("cron-scheduler"),
	}

	// Schedule daily job to maintain 2-week window (runs at 12:01 AM every day)
	c.AddFunc("1 0 * * *", scheduler.maintainSlotWindow)

	return scheduler
}

// Start starts the cron scheduler
func (cs *CronScheduler) Start(ctx context.Context) {
	cs.log.Info(ctx, "Starting cron scheduler")
	cs.cron.Start()
	cs.log.Info(ctx, "Cron scheduler started")
}

// Stop stops the cron scheduler
func (cs *CronScheduler) Stop(ctx context.Context) {
	cs.log.Info(ctx, "Stopping cron scheduler")
	cs.cron.Stop()
	cs.log.Info(ctx, "Cron scheduler stopped")
}

// SeedInitialSlots seeds the initial appointment slots for the next 2 weeks
func (cs *CronScheduler) SeedInitialSlots(ctx context.Context) error {
	cs.log.Info(ctx, "Seeding initial appointment slots for the next 2 weeks")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Clean up old slots (older than today)
	err := cs.slotStorage.DeleteOldSlots(ctx, today)
	if err != nil {
		cs.log.Error(ctx, "Failed to clean up old slots during seeding", zap.Error(err))
		// Don't return error - continue with seeding
	}

	// Calculate end date (2 weeks from today)
	endDate := today.AddDate(0, 0, SeedWeeks*7)

	slotsCreated := 0
	current := today

	for current.Before(endDate) {
		weekday := current.Weekday()

		// Only create slots for Monday, Wednesday, Friday
		if weekday == time.Monday || weekday == time.Wednesday || weekday == time.Friday {
			err := cs.slotStorage.UpsertSlot(ctx, current, MaxAppointmentsPerDay)
			if err != nil {
				cs.log.Error(ctx, "Failed to create slot during seeding",
					zap.Error(err),
					zap.Time("date", current))
				// Continue with other slots even if one fails
			} else {
				slotsCreated++
				cs.log.Debug(ctx, "Created appointment slot",
					zap.Time("date", current),
					zap.Int32("capacity", MaxAppointmentsPerDay))
			}
		}

		current = current.AddDate(0, 0, 1)
	}

	cs.log.Info(ctx, "Initial slot seeding completed",
		zap.Int("slots_created", slotsCreated),
		zap.Time("from_date", today),
		zap.Time("to_date", endDate))

	return nil
}

// maintainSlotWindow maintains the 2-week rolling window of appointment slots
func (cs *CronScheduler) maintainSlotWindow() {
	ctx := context.Background()
	cs.log.Info(ctx, "Running scheduled slot maintenance")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Clean up old slots (older than today)
	err := cs.slotStorage.DeleteOldSlots(ctx, today)
	if err != nil {
		cs.log.Error(ctx, "Failed to clean up old slots", zap.Error(err))
	} else {
		cs.log.Debug(ctx, "Old slots cleaned up successfully")
	}

	// Calculate the end of current 2-week window
	endDate := today.AddDate(0, 0, SeedWeeks*7)

	// Add new slots to maintain the 2-week window
	// We'll check from today + 13 days to today + 14 days to add the next day
	newSlotDate := today.AddDate(0, 0, (SeedWeeks*7)-1)
	weekday := newSlotDate.Weekday()

	// Only create slot if it's Monday, Wednesday, or Friday
	if weekday == time.Monday || weekday == time.Wednesday || weekday == time.Friday {
		err := cs.slotStorage.UpsertSlot(ctx, newSlotDate, MaxAppointmentsPerDay)
		if err != nil {
			cs.log.Error(ctx, "Failed to create new slot during maintenance",
				zap.Error(err),
				zap.Time("date", newSlotDate))
		} else {
			cs.log.Info(ctx, "New appointment slot created",
				zap.Time("date", newSlotDate),
				zap.Int32("capacity", MaxAppointmentsPerDay))
		}
	}

	// Also ensure we have all slots for the current 2-week window
	current := today
	slotsChecked := 0
	slotsCreated := 0

	for current.Before(endDate) {
		weekday := current.Weekday()

		if weekday == time.Monday || weekday == time.Wednesday || weekday == time.Friday {
			slotsChecked++
			err := cs.slotStorage.UpsertSlot(ctx, current, MaxAppointmentsPerDay)
			if err != nil {
				cs.log.Error(ctx, "Failed to ensure slot exists",
					zap.Error(err),
					zap.Time("date", current))
			} else {
				slotsCreated++
				cs.log.Debug(ctx, "Ensured appointment slot exists",
					zap.Time("date", current))
			}
		}

		current = current.AddDate(0, 0, 1)
	}

	cs.log.Info(ctx, "Slot maintenance completed",
		zap.Int("slots_checked", slotsChecked),
		zap.Int("slots_created", slotsCreated),
		zap.Time("window_start", today),
		zap.Time("window_end", endDate))
}
