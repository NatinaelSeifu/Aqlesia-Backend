package appointment

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

const (
	MaxAppointmentsPerDay = 10
)

type appointment struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.Appointment {
	return &appointment{
		db:  db,
		log: log,
	}
}

func (a *appointment) Create(ctx context.Context, userID uuid.UUID, param dto.CreateAppointmentRequest) (*dto.Appointment, error) {
	// Parse the appointment date
	appointmentDate, err := param.GetAppointmentDate()
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment date")
		a.log.Error(ctx, "unable to parse appointment date", zap.Error(err))
		return nil, err
	}

	// Check if user already has an active appointment
	activeAppointment, err := a.db.GetUserActiveAppointment(ctx, userID)
	if err != nil && err != sql.ErrNoRows {
		err = errors.ErrReadError.Wrap(err, "could not check user active appointments")
		a.log.Error(ctx, "unable to check user active appointments", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}
	if err == nil {
		// User already has an active appointment
		err = errors.ErrWriteError.New("user already has an active appointment")
		a.log.Warn(ctx, "user already has an active appointment", 
			zap.String("user-id", userID.String()),
			zap.Time("existing-appointment-date", activeAppointment.AppointmentDate),
			zap.String("existing-appointment-id", activeAppointment.ID.String()))
		return nil, err
	}

	// Check if the selected date has reached maximum capacity
	count, err := a.db.CountAppointmentsByDate(ctx, appointmentDate)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count appointments by date")
		a.log.Error(ctx, "unable to count appointments by date", zap.Error(err), zap.Time("date", appointmentDate))
		return nil, err
	}
	if count >= MaxAppointmentsPerDay {
		err = errors.ErrWriteError.New("selected date has reached maximum capacity")
		a.log.Warn(ctx, "appointment date at maximum capacity", 
			zap.Time("date", appointmentDate),
			zap.Int64("current-count", count),
			zap.Int("max-capacity", MaxAppointmentsPerDay))
		return nil, err
	}

	// Convert notes to sql.NullString
	var notes sql.NullString
	if param.Notes != nil {
		notes = sql.NullString{String: *param.Notes, Valid: true}
	}

	// Create the appointment
	appointmentDB, err := a.db.CreateAppointment(ctx, db.CreateAppointmentParams{
		UserID:          userID,
		AppointmentDate: appointmentDate,
		Status:          string(dto.AppointmentStatusPending),
		Notes:           notes,
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not create appointment")
		a.log.Error(ctx, "unable to create appointment", zap.Error(err), zap.Any("appointment", param))
		return nil, err
	}

	// Convert to DTO
	return a.convertToDTO(appointmentDB), nil
}

func (a *appointment) Update(ctx context.Context, id uuid.UUID, param dto.UpdateAppointmentRequest) (*dto.Appointment, error) {
	// Get current appointment to preserve existing values
	currentAppointment, err := a.db.GetAppointment(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment not found", zap.String("appointment-id", id.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read appointment")
		a.log.Error(ctx, "unable to get current appointment for update", zap.Error(err), zap.String("appointment-id", id.String()))
		return nil, err
	}

	// Prepare update parameters
	var appointmentDate time.Time
	var notes sql.NullString

	// Handle appointment date update
	if param.AppointmentDate != nil {
		newDate, err := param.GetAppointmentDate()
		if err != nil {
			err = errors.ErrInvalidUserInput.Wrap(err, "invalid appointment date")
			a.log.Error(ctx, "unable to parse new appointment date", zap.Error(err))
			return nil, err
		}
		
		// If the date is changing, check capacity for the new date
		if !newDate.Equal(currentAppointment.AppointmentDate) {
			count, err := a.db.CountAppointmentsByDate(ctx, *newDate)
			if err != nil {
				err = errors.ErrReadError.Wrap(err, "could not count appointments by date")
				a.log.Error(ctx, "unable to count appointments by date", zap.Error(err), zap.Time("date", *newDate))
				return nil, err
			}
			if count >= MaxAppointmentsPerDay {
				err = errors.ErrWriteError.New("selected date has reached maximum capacity")
				a.log.Warn(ctx, "new appointment date at maximum capacity", 
					zap.Time("date", *newDate),
					zap.Int64("current-count", count),
					zap.Int("max-capacity", MaxAppointmentsPerDay))
				return nil, err
			}
		}
		appointmentDate = *newDate
	} else {
		// Use current date if not updating
		appointmentDate = currentAppointment.AppointmentDate
	}

	// Handle notes update
	if param.Notes != nil {
		notes = sql.NullString{String: *param.Notes, Valid: true}
	} else {
		notes = currentAppointment.Notes // Preserve current value
	}

	// Update the appointment
	updatedAppointment, err := a.db.UpdateAppointment(ctx, db.UpdateAppointmentParams{
		ID:              id,
		AppointmentDate: appointmentDate,
		Notes:           notes,
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not update appointment")
		a.log.Error(ctx, "unable to update appointment", zap.Error(err), zap.String("appointment-id", id.String()))
		return nil, err
	}

	return a.convertToDTO(updatedAppointment), nil
}

func (a *appointment) Get(ctx context.Context, id uuid.UUID) (*dto.Appointment, error) {
	appointmentDB, err := a.db.GetAppointment(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment not found", zap.String("appointment-id", id.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read appointment")
		a.log.Error(ctx, "unable to get appointment", zap.Error(err), zap.String("appointment-id", id.String()))
		return nil, err
	}

	return a.convertToDTO(appointmentDB), nil
}

func (a *appointment) GetWithUser(ctx context.Context, id uuid.UUID) (*dto.Appointment, error) {
	appointmentWithUser, err := a.db.GetAppointmentWithUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment with user not found", zap.String("appointment-id", id.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read appointment with user")
		a.log.Error(ctx, "unable to get appointment with user", zap.Error(err), zap.String("appointment-id", id.String()))
		return nil, err
	}

	return a.convertWithUserRowToDTO(appointmentWithUser), nil
}

func (a *appointment) GetUserAppointments(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]dto.Appointment, int64, error) {
	offset := (page - 1) * pageSize
	
	appointments, err := a.db.GetUserAppointments(ctx, db.GetUserAppointmentsParams{
		UserID:      userID,
		OffsetCount: int32(offset),
		LimitCount:  int32(pageSize),
	})
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read user appointments")
		a.log.Error(ctx, "unable to get user appointments", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, 0, err
	}

	// Get total count
	total, err := a.db.CountUserAppointments(ctx, userID)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count user appointments")
		a.log.Error(ctx, "unable to count user appointments", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, 0, err
	}

	// Convert to DTOs
	appointmentDTOs := make([]dto.Appointment, len(appointments))
	for i, appointment := range appointments {
		appointmentDTOs[i] = *a.convertToDTO(appointment)
	}

	return appointmentDTOs, total, nil
}

func (a *appointment) GetAllAppointments(ctx context.Context, page, pageSize int) ([]dto.Appointment, int64, error) {
	offset := (page - 1) * pageSize
	
	appointments, err := a.db.GetAllAppointments(ctx, db.GetAllAppointmentsParams{
		OffsetCount: int32(offset),
		LimitCount:  int32(pageSize),
	})
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read all appointments")
		a.log.Error(ctx, "unable to get all appointments", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total, err := a.db.CountAllAppointments(ctx)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count all appointments")
		a.log.Error(ctx, "unable to count all appointments", zap.Error(err))
		return nil, 0, err
	}

	// Convert to DTOs
	appointmentDTOs := make([]dto.Appointment, len(appointments))
	for i, appointment := range appointments {
		appointmentDTOs[i] = *a.convertAllAppointmentsRowToDTO(appointment)
	}

	return appointmentDTOs, total, nil
}

func (a *appointment) GetUserActiveAppointment(ctx context.Context, userID uuid.UUID) (*dto.Appointment, error) {
	appointmentDB, err := a.db.GetUserActiveAppointment(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active appointment found
		}
		err = errors.ErrReadError.Wrap(err, "could not read user active appointment")
		a.log.Error(ctx, "unable to get user active appointment", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	return a.convertToDTO(appointmentDB), nil
}

func (a *appointment) CountAppointmentsByDate(ctx context.Context, date time.Time) (int64, error) {
	count, err := a.db.CountAppointmentsByDate(ctx, date)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count appointments by date")
		a.log.Error(ctx, "unable to count appointments by date", zap.Error(err), zap.Time("date", date))
		return 0, err
	}
	return count, nil
}

func (a *appointment) MarkCompleted(ctx context.Context, id uuid.UUID, notes *string) (*dto.Appointment, error) {
	var notesParam sql.NullString
	if notes != nil {
		notesParam = sql.NullString{String: *notes, Valid: true}
	}

	updatedAppointment, err := a.db.UpdateAppointmentStatus(ctx, db.UpdateAppointmentStatusParams{
		ID:     id,
		Status: string(dto.AppointmentStatusCompleted),
		Notes:  notesParam,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment not found for completion", zap.String("appointment-id", id.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not mark appointment as completed")
		a.log.Error(ctx, "unable to mark appointment as completed", zap.Error(err), zap.String("appointment-id", id.String()))
		return nil, err
	}

	return a.convertToDTO(updatedAppointment), nil
}

func (a *appointment) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.UpdateAppointmentStatus(ctx, db.UpdateAppointmentStatusParams{
		ID:     id,
		Status: string(dto.AppointmentStatusCancelled),
		Notes:  sql.NullString{}, // Keep existing notes
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment not found for cancellation", zap.String("appointment-id", id.String()))
			return err
		}
		err = errors.ErrWriteError.Wrap(err, "could not cancel appointment")
		a.log.Error(ctx, "unable to cancel appointment", zap.Error(err), zap.String("appointment-id", id.String()))
		return err
	}

	return nil
}

func (a *appointment) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.DeleteAppointment(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "appointment not found")
			a.log.Info(ctx, "Appointment not found for deletion", zap.String("appointment-id", id.String()))
			return err
		}
		err = errors.ErrWriteError.Wrap(err, "could not delete appointment")
		a.log.Error(ctx, "unable to delete appointment", zap.Error(err), zap.String("appointment-id", id.String()))
		return err
	}

	return nil
}

func (a *appointment) GetAppointmentStats(ctx context.Context) (*dto.AppointmentStatsResponse, error) {
	stats, err := a.db.GetAppointmentStats(ctx)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read appointment stats")
		a.log.Error(ctx, "unable to get appointment stats", zap.Error(err))
		return nil, err
	}

	return &dto.AppointmentStatsResponse{
		TotalAppointments:     stats.TotalAppointments,
		PendingAppointments:   stats.PendingAppointments,
		CompletedAppointments: stats.CompletedAppointments,
		CancelledAppointments: stats.CancelledAppointments,
	}, nil
}

func (a *appointment) GetAppointmentsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]dto.Appointment, error) {
	appointments, err := a.db.GetAppointmentsByDateRange(ctx, db.GetAppointmentsByDateRangeParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read appointments by date range")
		a.log.Error(ctx, "unable to get appointments by date range", zap.Error(err), 
			zap.Time("start-date", startDate), zap.Time("end-date", endDate))
		return nil, err
	}

	// Convert to DTOs
	appointmentDTOs := make([]dto.Appointment, len(appointments))
	for i, appointment := range appointments {
		appointmentDTOs[i] = *a.convertDateRangeRowToDTO(appointment)
	}

	return appointmentDTOs, nil
}

func (a *appointment) GetUpcomingAppointments(ctx context.Context, limit int) ([]dto.Appointment, error) {
	appointments, err := a.db.GetUpcomingAppointments(ctx, int32(limit))
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read upcoming appointments")
		a.log.Error(ctx, "unable to get upcoming appointments", zap.Error(err))
		return nil, err
	}

	// Convert to DTOs
	appointmentDTOs := make([]dto.Appointment, len(appointments))
	for i, appointment := range appointments {
		appointmentDTOs[i] = *a.convertUpcomingRowToDTO(appointment)
	}

	return appointmentDTOs, nil
}

// Helper methods to convert database models to DTOs

func (a *appointment) convertToDTO(appointmentDB db.Appointment) *dto.Appointment {
	var notes *string
	if appointmentDB.Notes.Valid {
		notes = &appointmentDB.Notes.String
	}

	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	if appointmentDB.CreatedAt.Valid {
		createdAt = appointmentDB.CreatedAt.Time
	}
	if appointmentDB.UpdatedAt.Valid {
		updatedAt = appointmentDB.UpdatedAt.Time
	}
	if appointmentDB.DeletedAt.Valid {
		deletedAt = &appointmentDB.DeletedAt.Time
	}

	return &dto.Appointment{
		ID:              appointmentDB.ID,
		UserID:          appointmentDB.UserID,
		AppointmentDate: appointmentDB.AppointmentDate,
		Status:          dto.AppointmentStatus(appointmentDB.Status),
		Notes:           notes,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       deletedAt,
	}
}

func (a *appointment) convertWithUserRowToDTO(row db.GetAppointmentWithUserRow) *dto.Appointment {
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Appointment{
		ID:              row.ID,
		UserID:          row.UserID,
		AppointmentDate: row.AppointmentDate,
		Status:          dto.AppointmentStatus(row.Status),
		Notes:           notes,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}

func (a *appointment) convertAllAppointmentsRowToDTO(row db.GetAllAppointmentsRow) *dto.Appointment {
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Appointment{
		ID:              row.ID,
		UserID:          row.UserID,
		AppointmentDate: row.AppointmentDate,
		Status:          dto.AppointmentStatus(row.Status),
		Notes:           notes,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}

func (a *appointment) convertDateRangeRowToDTO(row db.GetAppointmentsByDateRangeRow) *dto.Appointment {
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Appointment{
		ID:              row.ID,
		UserID:          row.UserID,
		AppointmentDate: row.AppointmentDate,
		Status:          dto.AppointmentStatus(row.Status),
		Notes:           notes,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}

func (a *appointment) convertUpcomingRowToDTO(row db.GetUpcomingAppointmentsRow) *dto.Appointment {
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return &dto.Appointment{
		ID:              row.ID,
		UserID:          row.UserID,
		AppointmentDate: row.AppointmentDate,
		Status:          dto.AppointmentStatus(row.Status),
		Notes:           notes,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       deletedAt,
		User: &dto.User{
			ID:          row.UserID,
			Name:        row.UserName,
			LastName:    row.UserLastname,
			PhoneNumber: row.UserPhone,
		},
	}
}
