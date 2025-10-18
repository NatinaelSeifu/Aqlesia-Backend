package scheduler

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/constants/model/db"
	"aqlesia/platform/logger"
	"context"
	"time"

	"go.uber.org/zap"
)

// AppointmentScheduler handles scheduling of appointment slots
type AppointmentScheduler struct {
	log logger.Logger
	db  dbinstance.DBInstance
}

// NewAppointmentScheduler creates a new appointment scheduler
func NewAppointmentScheduler(log logger.Logger, db dbinstance.DBInstance) *AppointmentScheduler {
	return &AppointmentScheduler{
		log: log,
		db:  db,
	}
}

// GenerateAppointmentSlots creates appointment slots in the database for the next 2 weeks
// This is called by a cron job every Monday, Wednesday, Friday
func (s *AppointmentScheduler) GenerateAppointmentSlots(ctx context.Context) error {
	s.log.Info(ctx, "Generating appointment slots for next 2 weeks")
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := today.AddDate(0, 0, 14) // 2 weeks from today
	
	current := today
	createdSlots := 0
	updatedSlots := 0
	
	for current.Before(endDate) || current.Equal(endDate) {
		// Check if it's Monday, Wednesday, or Friday
		weekday := current.Weekday()
		if weekday == time.Monday || weekday == time.Wednesday || weekday == time.Friday {
			// Create or update the available slot
			slot, err := s.db.UpsertAvailableDate(ctx, db.UpsertAvailableDateParams{
				SlotDate:        current,
				MaxCapacity:     10, // Default max capacity
				CurrentBookings: 0,  // Will be updated by database triggers
				IsActive:        true,
			})
			
			if err != nil {
				s.log.Error(ctx, "Failed to create/update appointment slot", 
					zap.Error(err), 
					zap.Time("date", current))
				// Continue with other dates even if one fails
			} else {
				if slot.CreatedAt.Time.After(now.Add(-time.Minute)) {
					createdSlots++
					s.log.Debug(ctx, "Created new appointment slot", zap.Time("date", current))
				} else {
					updatedSlots++
					s.log.Debug(ctx, "Updated existing appointment slot", zap.Time("date", current))
				}
			}
		}
		current = current.AddDate(0, 0, 1)
	}
	
	// Clean up old slots (older than today)
	err := s.db.DeleteOldDates(ctx, today)
	if err != nil {
		s.log.Error(ctx, "Failed to delete old slots", zap.Error(err))
	} else {
		s.log.Debug(ctx, "Cleaned up old appointment slots")
	}
	
	s.log.Info(ctx, "Completed generating appointment slots", 
		zap.Int("created", createdSlots),
		zap.Int("updated", updatedSlots))
	return nil
}

// StartScheduler starts the cron job scheduler (placeholder implementation)
func (s *AppointmentScheduler) StartScheduler(ctx context.Context) {
	s.log.Info(ctx, "Starting appointment scheduler")
	
	// In a real implementation, you would use a cron library like:
	// - github.com/robfig/cron/v3
	// - or a job queue system like Redis/RabbitMQ
	
	// Example with a simple ticker for demonstration
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			s.log.Info(ctx, "Stopping appointment scheduler")
			return
		case <-ticker.C:
			// Check if today is Monday, Wednesday, or Friday
			now := time.Now()
			weekday := now.Weekday()
			if weekday == time.Monday || weekday == time.Wednesday || weekday == time.Friday {
				if err := s.GenerateAppointmentSlots(ctx); err != nil {
					s.log.Error(ctx, "Failed to generate appointment slots", zap.Error(err))
				}
			}
		}
	}
}
