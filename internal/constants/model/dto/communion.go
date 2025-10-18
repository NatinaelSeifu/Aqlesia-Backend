package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// CommunionStatus represents the status of a communion request
type CommunionStatus string

const (
	CommunionStatusPending  CommunionStatus = "pending"
	CommunionStatusApproved CommunionStatus = "approved"
	CommunionStatusRejected CommunionStatus = "rejected"
)

// Communion represents a communion request in the system
type Communion struct {
	// ID is the unique identifier of the communion request
	ID uuid.UUID `json:"id"`
	// UserID is the ID of the user who made the communion request
	UserID uuid.UUID `json:"user_id"`
	// User contains the user information (populated when needed)
	User *User `json:"user,omitempty"`
	// CommunionDate is the requested communion date (must be in the past)
	CommunionDate time.Time `json:"communion_date"`
	// Status is the current status of the communion request
	Status CommunionStatus `json:"status"`
	// RequestedAt is the time when the communion was requested
	RequestedAt time.Time `json:"requested_at"`
	// ApprovedAt is the time when the communion was approved (if applicable)
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	// ApprovedByUserID is the ID of the admin who approved the communion
	ApprovedByUserID *uuid.UUID `json:"approved_by_user_id,omitempty"`
	// ApprovedBy contains the admin user information (populated when needed)
	ApprovedBy *User `json:"approved_by,omitempty"`
	// CreatedAt is the time when the communion record was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time when the communion record was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// DeletedAt is the time when the communion record was soft deleted
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// CreateCommunionRequest represents the request payload for creating a communion request
type CreateCommunionRequest struct {
	// CommunionDate is the desired communion date (format: YYYY-MM-DD) - must be in the past
	CommunionDate string `json:"communion_date"`
}

func (c CreateCommunionRequest) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.CommunionDate,
			validation.Required.Error("communion date is required"),
			validation.By(c.validateCommunionDate)),
	)
}

// validateCommunionDate validates the communion date format and business rules
func (c CreateCommunionRequest) validateCommunionDate(value interface{}) error {
	dateStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_date_invalid_type", "communion date must be a string")
	}

	// Parse the date in YYYY-MM-DD format
	communionDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return validation.NewError("validation_date_invalid_format", "communion date must be in YYYY-MM-DD format")
	}

	// Check if the date is in the future (only past dates are allowed)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if communionDate.After(today) || communionDate.Equal(today) {
		return validation.NewError("validation_date_not_past", "communion date must be in the past")
	}

	return nil
}

// GetCommunionDate returns the parsed communion date
func (c CreateCommunionRequest) GetCommunionDate() (time.Time, error) {
	return time.Parse("2006-01-02", c.CommunionDate)
}

// UpdateCommunionStatusRequest represents the request payload for updating communion status
type UpdateCommunionStatusRequest struct {
	// Status is the new status for the communion request
	Status string `json:"status"`
}

func (u UpdateCommunionStatusRequest) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Status,
			validation.Required.Error("status is required"),
			validation.In("approved", "rejected").Error("status must be either approved or rejected")),
	)
}

// CommunionListResponse represents a paginated list of communion requests
type CommunionListResponse struct {
	Communions []Communion `json:"communions"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
}
