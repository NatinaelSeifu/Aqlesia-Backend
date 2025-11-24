package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// AppointmentStatus represents the status of an appointment
type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusCompleted AppointmentStatus = "completed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
)

// Appointment represents an appointment in the system
type Appointment struct {
	// ID is the unique identifier of the appointment
	ID uuid.UUID `json:"id"`
	// UserID is the ID of the user who booked the appointment
	UserID uuid.UUID `json:"user_id"`
	// User contains the user information (populated when needed)
	User *User `json:"user,omitempty"`
	// AppointmentDate is the date of the appointment (without time)
	AppointmentDate time.Time `json:"appointment_date"`
	// Status is the current status of the appointment
	Status AppointmentStatus `json:"status"`
	// Notes contains additional notes about the appointment
	Notes *string `json:"notes,omitempty"`
	// CreatedAt is the time when the appointment was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time when the appointment was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// DeletedAt is the time when the appointment was soft deleted
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// CreateAppointmentRequest represents the request payload for creating an appointment
type CreateAppointmentRequest struct {
	// AppointmentDate is the desired date for the appointment (format: YYYY-MM-DD)
	AppointmentDate string `json:"appointment_date"`
	// Notes contains optional notes about the appointment
	Notes *string `json:"notes,omitempty"`
}

func (c CreateAppointmentRequest) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.AppointmentDate,
			validation.Required.Error("appointment date is required"),
			validation.By(c.validateAppointmentDate)),
		validation.Field(&c.Notes,
			validation.When(c.Notes != nil, validation.Length(0, 500).Error("notes must be no more than 500 characters"))),
	)
}

// validateAppointmentDate validates the appointment date format and business rules
func (c CreateAppointmentRequest) validateAppointmentDate(value interface{}) error {
	dateStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_date_invalid_type", "appointment date must be a string")
	}

	// Parse the date in YYYY-MM-DD format
	appointmentDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return validation.NewError("validation_date_invalid_format", "appointment date must be in YYYY-MM-DD format")
	}

	// Check if the date is in the past
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if appointmentDate.Before(today) {
		return validation.NewError("validation_date_in_past", "appointment date cannot be in the past")
	}

	// Check if the date is too far in the future (more than 1 month)
	maxDate := today.AddDate(0, 1, 0)
	if appointmentDate.After(maxDate) {
		return validation.NewError("validation_date_too_far", "appointment date cannot be more than 1 month in the future")
	}

	// Check if it's a valid appointment day (Monday, Wednesday, Friday)
	weekday := appointmentDate.Weekday()
	if weekday != time.Monday && weekday != time.Wednesday && weekday != time.Friday {
		return validation.NewError("validation_date_invalid_day", "appointments are only available on Monday, Wednesday, and Friday")
	}

	return nil
}

// GetAppointmentDate returns the parsed appointment date
func (c CreateAppointmentRequest) GetAppointmentDate() (time.Time, error) {
	return time.Parse("2006-01-02", c.AppointmentDate)
}

// UpdateAppointmentRequest represents the request payload for updating an appointment
type UpdateAppointmentRequest struct {
	// AppointmentDate is the new date for the appointment (format: YYYY-MM-DD)
	AppointmentDate *string `json:"appointment_date,omitempty"`
	// Notes contains optional notes about the appointment
	Notes *string `json:"notes,omitempty"`
}

func (u UpdateAppointmentRequest) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.AppointmentDate,
			validation.When(u.AppointmentDate != nil, validation.By(u.validateAppointmentDate))),
		validation.Field(&u.Notes,
			validation.When(u.Notes != nil, validation.Length(0, 500).Error("notes must be no more than 500 characters"))),
	)
}

// validateAppointmentDate validates the appointment date format and business rules for updates
func (u UpdateAppointmentRequest) validateAppointmentDate(value interface{}) error {
	dateStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_date_invalid_type", "appointment date must be a string")
	}

	// Parse the date in YYYY-MM-DD format
	appointmentDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return validation.NewError("validation_date_invalid_format", "appointment date must be in YYYY-MM-DD format")
	}

	// Check if the date is in the past
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if appointmentDate.Before(today) {
		return validation.NewError("validation_date_in_past", "appointment date cannot be in the past")
	}

	// Check if the date is too far in the future (more than 1 month)
	maxDate := today.AddDate(0, 1, 0)
	if appointmentDate.After(maxDate) {
		return validation.NewError("validation_date_too_far", "appointment date cannot be more than 1 month in the future")
	}

	// Check if it's a valid appointment day (Monday, Wednesday, Friday)
	weekday := appointmentDate.Weekday()
	if weekday != time.Monday && weekday != time.Wednesday && weekday != time.Friday {
		return validation.NewError("validation_date_invalid_day", "appointments are only available on Monday, Wednesday, and Friday")
	}

	return nil
}

// GetAppointmentDate returns the parsed appointment date for updates
func (u UpdateAppointmentRequest) GetAppointmentDate() (*time.Time, error) {
	if u.AppointmentDate == nil {
		return nil, nil
	}
	date, err := time.Parse("2006-01-02", *u.AppointmentDate)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

// MarkAppointmentCompletedRequest represents the request to mark an appointment as completed
type MarkAppointmentCompletedRequest struct {
	// Notes contains optional completion notes
	Notes *string `json:"notes,omitempty"`
}

func (m MarkAppointmentCompletedRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.Notes,
			validation.When(m.Notes != nil, validation.Length(0, 500).Error("notes must be no more than 500 characters"))),
	)
}

// AppointmentListResponse represents a paginated list of appointments
type AppointmentListResponse struct {
	Appointments []Appointment `json:"appointments"`
	Total        int64         `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

// AppointmentStatsResponse represents appointment statistics
type AppointmentStatsResponse struct {
	TotalAppointments     int64 `json:"total_appointments"`
	PendingAppointments   int64 `json:"pending_appointments"`
	CompletedAppointments int64 `json:"completed_appointments"`
	CancelledAppointments int64 `json:"cancelled_appointments"`
}

// AvailableSlot represents an available appointment slot
type AvailableSlot struct {
	// ID is the unique identifier of the available slot
	ID uuid.UUID `json:"id"`
	// SlotDate is the date of the slot
	SlotDate time.Time `json:"slot_date"`
	// MaxCapacity is the maximum number of appointments for this slot
	MaxCapacity int32 `json:"max_capacity"`
	// CurrentBookings is the current number of bookings for this slot
	CurrentBookings int32 `json:"current_bookings"`
	// AvailableSpots is the remaining available spots (computed field)
	AvailableSpots int32 `json:"available_spots"`
	// IsActive indicates if the slot is active
	IsActive bool `json:"is_active"`
	// CreatedAt is the time when the slot was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time when the slot was last updated
	UpdatedAt time.Time `json:"updated_at"`
}
