package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// AvailableDate represents an available date for appointments
type AvailableDate struct {
	// ID is the unique identifier of the available date
	ID uuid.UUID `json:"id"`
	// SlotDate is the date available for appointments
	SlotDate time.Time `json:"slot_date"`
	// MaxCapacity is the maximum number of appointments allowed on this date
	MaxCapacity int32 `json:"max_capacity"`
	// CurrentBookings is the current number of booked appointments
	CurrentBookings int32 `json:"current_bookings"`
	// AvailableSpots is the number of available spots (calculated field)
	AvailableSpots int32 `json:"available_spots"`
	// IsActive indicates if this date is active for booking
	IsActive bool `json:"is_active"`
	// CreatedAt is the time when the available date was created
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt is the time when the available date was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// CreateAvailableDate represents the request to create a new available date
type CreateAvailableDate struct {
	// SlotDate is the date to make available for appointments
	SlotDate Date `json:"slot_date"`
	// MaxCapacity is the maximum number of appointments allowed on this date (defaults to 10)
	MaxCapacity int32 `json:"max_capacity"`
	// IsActive indicates if this date should be active immediately (defaults to true)
	IsActive *bool `json:"is_active,omitempty"`
}

func (c CreateAvailableDate) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.SlotDate, 
			validation.Required.Error("slot date is required"),
			validation.By(c.validateFutureDate)),
		validation.Field(&c.MaxCapacity, 
			validation.Required.Error("max capacity is required"),
			validation.Min(1).Error("max capacity must be at least 1"),
			validation.Max(100).Error("max capacity cannot exceed 100")),
	)
}

// validateFutureDate ensures the slot date is not in the past
func (c CreateAvailableDate) validateFutureDate(value interface{}) error {
	var slotDate time.Time
	
	// Handle both Date and time.Time types
	switch v := value.(type) {
	case Date:
		slotDate = v.Time
	case time.Time:
		slotDate = v
	default:
		return validation.NewError("validation_date_invalid_type", "slot date must be a valid date")
	}
	
	// Allow dates from today onwards
	today := time.Now().Truncate(24 * time.Hour)
	if slotDate.Before(today) {
		return validation.NewError("validation_date_in_past", "slot date cannot be in the past")
	}
	
	return nil
}

// GetIsActive returns the is_active value, defaulting to true if not specified
func (c CreateAvailableDate) GetIsActive() bool {
	if c.IsActive == nil {
		return true
	}
	return *c.IsActive
}

// UpdateAvailableDate represents the request to update an existing available date
type UpdateAvailableDate struct {
	// MaxCapacity is the maximum number of appointments allowed on this date
	MaxCapacity *int32 `json:"max_capacity,omitempty"`
	// IsActive indicates if this date should be active for booking
	IsActive *bool `json:"is_active,omitempty"`
}

func (u UpdateAvailableDate) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.MaxCapacity, validation.When(u.MaxCapacity != nil,
			validation.Min(int32(1)).Error("max capacity must be at least 1"),
			validation.Max(int32(100)).Error("max capacity cannot exceed 100"))),
	)
}

// AvailableDatesListResponse represents a paginated list of available dates
type AvailableDatesListResponse struct {
	Dates    []AvailableDate `json:"dates"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// AvailableDateQuery represents query parameters for filtering available dates
type AvailableDateQuery struct {
	// StartDate is the start of the date range to query
	StartDate time.Time `json:"start_date"`
	// EndDate is the end of the date range to query  
	EndDate time.Time `json:"end_date"`
	// IncludeInactive includes inactive dates in the results
	IncludeInactive bool `json:"include_inactive,omitempty"`
	// OnlyAvailable only returns dates with available capacity
	OnlyAvailable bool `json:"only_available,omitempty"`
}

func (q AvailableDateQuery) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.StartDate, validation.Required.Error("start date is required")),
		validation.Field(&q.EndDate, validation.Required.Error("end date is required")),
		validation.Field(&q.EndDate, validation.By(q.validateEndDate)),
	)
}

// validateEndDate ensures end date is not before start date
func (q AvailableDateQuery) validateEndDate(value interface{}) error {
	endDate, ok := value.(time.Time)
	if !ok {
		return validation.NewError("validation_date_invalid_type", "end date must be a valid date")
	}
	
	if endDate.Before(q.StartDate) {
		return validation.NewError("validation_date_range_invalid", "end date must be after start date")
	}
	
	return nil
}
