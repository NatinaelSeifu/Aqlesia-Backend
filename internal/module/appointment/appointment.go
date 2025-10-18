package appointment

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type appointment struct {
	appointmentStorage storage.Appointment
	userStorage        storage.User
	slotStorage        storage.AvailableSlot
	log                logger.Logger
}

func Init(appointmentStorage storage.Appointment, userStorage storage.User, slotStorage storage.AvailableSlot, log logger.Logger) module.Appointment {
	return &appointment{
		appointmentStorage: appointmentStorage,
		userStorage:        userStorage,
		slotStorage:        slotStorage,
		log:                log,
	}
}

func (a *appointment) Create(ctx context.Context, userID uuid.UUID, param dto.CreateAppointmentRequest) (*dto.Appointment, error) {
	a.log.Info(ctx, "Creating appointment", zap.String("user-id", userID.String()), zap.Any("request", param))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment request")
		a.log.Error(ctx, "invalid appointment request", zap.Error(err))
		return nil, err
	}

	// Verify user exists
	_, err := a.userStorage.Get(ctx, userID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "user not found")
		a.log.Error(ctx, "user not found for appointment creation", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Create appointment through storage layer (which handles business rules)
	appointment, err := a.appointmentStorage.Create(ctx, userID, param)
	if err != nil {
		a.log.Error(ctx, "failed to create appointment", zap.Error(err))
		return nil, err
	}

	a.log.Info(ctx, "Appointment created successfully", 
		zap.String("appointment-id", appointment.ID.String()),
		zap.String("user-id", userID.String()),
		zap.Time("appointment-date", appointment.AppointmentDate))

	return appointment, nil
}

func (a *appointment) Update(ctx context.Context, id string, userID uuid.UUID, param dto.UpdateAppointmentRequest) (*dto.Appointment, error) {
	a.log.Info(ctx, "Updating appointment", zap.String("appointment-id", id), zap.String("user-id", userID.String()))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment update request")
		a.log.Error(ctx, "invalid appointment update request", zap.Error(err))
		return nil, err
	}

	// Parse appointment ID
	appointmentID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment ID")
		a.log.Error(ctx, "invalid appointment ID", zap.Error(err), zap.String("appointment-id", id))
		return nil, err
	}

	// Get current appointment to verify ownership
	currentAppointment, err := a.appointmentStorage.Get(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to get appointment for update", zap.Error(err))
		return nil, err
	}

	// Verify ownership (users can only update their own appointments unless admin/manager)
	if currentAppointment.UserID != userID {
		err = errors.ErrInvalidUserInput.New("users can only update their own appointments")
		a.log.Warn(ctx, "user attempting to update another user's appointment", 
			zap.String("appointment-id", id),
			zap.String("appointment-owner", currentAppointment.UserID.String()),
			zap.String("requesting-user", userID.String()))
		return nil, err
	}

	// Check if appointment is in a state that allows updates
	if currentAppointment.Status == dto.AppointmentStatusCompleted {
		err = errors.ErrInvalidUserInput.New("cannot update completed appointments")
		a.log.Warn(ctx, "attempt to update completed appointment", zap.String("appointment-id", id))
		return nil, err
	}

	if currentAppointment.Status == dto.AppointmentStatusCancelled {
		err = errors.ErrInvalidUserInput.New("cannot update cancelled appointments")
		a.log.Warn(ctx, "attempt to update cancelled appointment", zap.String("appointment-id", id))
		return nil, err
	}

	// Check if appointment date has passed
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if currentAppointment.AppointmentDate.Before(today) {
		err = errors.ErrInvalidUserInput.New("cannot update appointments in the past")
		a.log.Warn(ctx, "attempt to update past appointment", 
			zap.String("appointment-id", id),
			zap.Time("appointment-date", currentAppointment.AppointmentDate))
		return nil, err
	}

	// Update appointment through storage layer
	updatedAppointment, err := a.appointmentStorage.Update(ctx, appointmentID, param)
	if err != nil {
		a.log.Error(ctx, "failed to update appointment", zap.Error(err))
		return nil, err
	}

	a.log.Info(ctx, "Appointment updated successfully", zap.String("appointment-id", id))
	return updatedAppointment, nil
}

func (a *appointment) Get(ctx context.Context, id string) (*dto.Appointment, error) {
	// Parse appointment ID
	appointmentID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment ID")
		a.log.Error(ctx, "invalid appointment ID", zap.Error(err), zap.String("appointment-id", id))
		return nil, err
	}

	// Get appointment with user information
	appointment, err := a.appointmentStorage.GetWithUser(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to get appointment", zap.Error(err))
		return nil, err
	}

	return appointment, nil
}

func (a *appointment) GetUserAppointments(ctx context.Context, userID uuid.UUID, page, pageSize int) (*dto.AppointmentListResponse, error) {
	a.log.Debug(ctx, "Getting user appointments", 
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

	// Get appointments from storage
	appointments, total, err := a.appointmentStorage.GetUserAppointments(ctx, userID, page, pageSize)
	if err != nil {
		a.log.Error(ctx, "failed to get user appointments", zap.Error(err))
		return nil, err
	}

	return &dto.AppointmentListResponse{
		Appointments: appointments,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func (a *appointment) GetAllAppointments(ctx context.Context, page, pageSize int) (*dto.AppointmentListResponse, error) {
	a.log.Debug(ctx, "Getting all appointments", zap.Int("page", page), zap.Int("page-size", pageSize))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	// Get appointments from storage
	appointments, total, err := a.appointmentStorage.GetAllAppointments(ctx, page, pageSize)
	if err != nil {
		a.log.Error(ctx, "failed to get all appointments", zap.Error(err))
		return nil, err
	}

	return &dto.AppointmentListResponse{
		Appointments: appointments,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func (a *appointment) MarkCompleted(ctx context.Context, id string, param dto.MarkAppointmentCompletedRequest) (*dto.Appointment, error) {
	a.log.Info(ctx, "Marking appointment as completed", zap.String("appointment-id", id))

	// Validate input
	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid completion request")
		a.log.Error(ctx, "invalid completion request", zap.Error(err))
		return nil, err
	}

	// Parse appointment ID
	appointmentID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment ID")
		a.log.Error(ctx, "invalid appointment ID", zap.Error(err), zap.String("appointment-id", id))
		return nil, err
	}

	// Get current appointment to verify state
	currentAppointment, err := a.appointmentStorage.Get(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to get appointment for completion", zap.Error(err))
		return nil, err
	}

	// Check if appointment is in a valid state for completion
	if currentAppointment.Status != dto.AppointmentStatusPending {
		err = errors.ErrInvalidUserInput.New(fmt.Sprintf("can only complete pending appointments, current status: %s", currentAppointment.Status))
		a.log.Warn(ctx, "attempt to complete non-pending appointment", 
			zap.String("appointment-id", id),
			zap.String("current-status", string(currentAppointment.Status)))
		return nil, err
	}

	// Mark appointment as completed through storage layer
	completedAppointment, err := a.appointmentStorage.MarkCompleted(ctx, appointmentID, param.Notes)
	if err != nil {
		a.log.Error(ctx, "failed to mark appointment as completed", zap.Error(err))
		return nil, err
	}

	a.log.Info(ctx, "Appointment marked as completed successfully", zap.String("appointment-id", id))
	return completedAppointment, nil
}

func (a *appointment) Cancel(ctx context.Context, id string, userID uuid.UUID) error {
	a.log.Info(ctx, "Cancelling appointment", zap.String("appointment-id", id), zap.String("user-id", userID.String()))

	// Parse appointment ID
	appointmentID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment ID")
		a.log.Error(ctx, "invalid appointment ID", zap.Error(err), zap.String("appointment-id", id))
		return err
	}

	// Get current appointment to verify ownership and state
	currentAppointment, err := a.appointmentStorage.Get(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to get appointment for cancellation", zap.Error(err))
		return err
	}

	// Verify ownership (users can only cancel their own appointments unless admin/manager)
	if currentAppointment.UserID != userID {
		err = errors.ErrInvalidUserInput.New("users can only cancel their own appointments")
		a.log.Warn(ctx, "user attempting to cancel another user's appointment", 
			zap.String("appointment-id", id),
			zap.String("appointment-owner", currentAppointment.UserID.String()),
			zap.String("requesting-user", userID.String()))
		return err
	}

	// Check if appointment can be cancelled
	if currentAppointment.Status == dto.AppointmentStatusCompleted {
		err = errors.ErrInvalidUserInput.New("cannot cancel completed appointments")
		a.log.Warn(ctx, "attempt to cancel completed appointment", zap.String("appointment-id", id))
		return err
	}

	if currentAppointment.Status == dto.AppointmentStatusCancelled {
		err = errors.ErrInvalidUserInput.New("appointment is already cancelled")
		a.log.Warn(ctx, "attempt to cancel already cancelled appointment", zap.String("appointment-id", id))
		return err
	}

	// Cancel appointment through storage layer
	err = a.appointmentStorage.Cancel(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to cancel appointment", zap.Error(err))
		return err
	}

	a.log.Info(ctx, "Appointment cancelled successfully", zap.String("appointment-id", id))
	return nil
}

func (a *appointment) Delete(ctx context.Context, id string) error {
	a.log.Info(ctx, "Deleting appointment", zap.String("appointment-id", id))

	// Parse appointment ID
	appointmentID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment ID")
		a.log.Error(ctx, "invalid appointment ID", zap.Error(err), zap.String("appointment-id", id))
		return err
	}

	// Delete appointment through storage layer
	err = a.appointmentStorage.Delete(ctx, appointmentID)
	if err != nil {
		a.log.Error(ctx, "failed to delete appointment", zap.Error(err))
		return err
	}

	a.log.Info(ctx, "Appointment deleted successfully", zap.String("appointment-id", id))
	return nil
}

func (a *appointment) GetAppointmentStats(ctx context.Context) (*dto.AppointmentStatsResponse, error) {
	a.log.Debug(ctx, "Getting appointment statistics")

	stats, err := a.appointmentStorage.GetAppointmentStats(ctx)
	if err != nil {
		a.log.Error(ctx, "failed to get appointment stats", zap.Error(err))
		return nil, err
	}

	return stats, nil
}

func (a *appointment) GetAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]time.Time, error) {
	a.log.Debug(ctx, "Getting available appointment dates from database slots", 
		zap.Time("start-date", startDate), 
		zap.Time("end-date", endDate))

	// Get available slots from the database
	slots, err := a.slotStorage.GetAvailableSlotsWithCapacity(ctx, startDate, endDate)
	if err != nil {
		a.log.Error(ctx, "failed to get available slots", zap.Error(err))
		return nil, err
	}

	// Extract dates from slots that have available capacity
	var availableDates []time.Time
	for _, slot := range slots {
		if slot.IsActive && slot.AvailableSpots > 0 {
			availableDates = append(availableDates, slot.SlotDate)
		}
	}

	a.log.Debug(ctx, "Found available dates from slots", 
		zap.Int("count", len(availableDates)),
		zap.Int("total-slots", len(slots)))
	
	if len(availableDates) == 0 {
		a.log.Info(ctx, "No available slots found. Make sure the appointment scheduler has created slots for the requested date range.")
	}
	
	return availableDates, nil
}
