package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// QuestionStatus represents the possible states of a question
type QuestionStatus string

const (
	QuestionStatusPending   QuestionStatus = "pending"
	QuestionStatusAnswered  QuestionStatus = "answered"
	QuestionStatusClosed    QuestionStatus = "closed"
	QuestionStatusCancelled QuestionStatus = "cancelled"
)

// Question represents a question submitted by a user
type Question struct {
	// ID is the unique identifier of the question
	ID uuid.UUID `json:"id"`
	// UserID is the ID of the user who submitted the question
	UserID uuid.UUID `json:"user_id"`
	// Question is the text of the question
	Question string `json:"question"`
	// Status is the current status of the question
	Status QuestionStatus `json:"status"`
	// AdminResponse is the admin's response to the question
	AdminResponse *string `json:"admin_response,omitempty"`
	// RespondedBy is the ID of the admin who responded
	RespondedBy *uuid.UUID `json:"responded_by,omitempty"`
	// CreatedAt is the time when the question was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time when the question was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// UserName is the name of the user who submitted the question (populated in some queries)
	UserName *string `json:"user_name,omitempty"`
	// ResponderName is the name of the admin who responded (populated in some queries)
	ResponderName *string `json:"responder_name,omitempty"`
}

// CreateQuestion represents the request to create a new question
type CreateQuestion struct {
	// Question is the text of the question
	Question string `json:"question"`
}

func (c CreateQuestion) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Question, 
			validation.Required.Error("question is required"),
			validation.Length(5, 1000).Error("question must be between 5 and 1000 characters")),
	)
}

// UpdateQuestion represents the request to update an existing question
type UpdateQuestion struct {
	// Question is the new text of the question (only for users updating their own questions)
	Question *string `json:"question,omitempty"`
	// Status is the new status of the question (admin only)
	Status *QuestionStatus `json:"status,omitempty"`
	// AdminResponse is the admin's response to the question (admin only)
	AdminResponse *string `json:"admin_response,omitempty"`
}

func (u UpdateQuestion) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Question, validation.When(u.Question != nil,
			validation.Length(5, 1000).Error("question must be between 5 and 1000 characters"))),
		validation.Field(&u.Status, validation.When(u.Status != nil,
			validation.In(QuestionStatusPending, QuestionStatusAnswered, QuestionStatusClosed, QuestionStatusCancelled).Error("invalid status"))),
		validation.Field(&u.AdminResponse, validation.When(u.AdminResponse != nil,
			validation.Length(1, 2000).Error("admin response must be between 1 and 2000 characters"))),
	)
}

// QuestionQuery represents query parameters for filtering questions
type QuestionQuery struct {
	// UserID filters questions by user (optional)
	UserID *uuid.UUID `json:"user_id,omitempty"`
	// Status filters questions by status (optional)
	Status *QuestionStatus `json:"status,omitempty"`
	// StartDate filters questions created after this date (optional)
	StartDate *time.Time `json:"start_date,omitempty"`
	// EndDate filters questions created before this date (optional)
	EndDate *time.Time `json:"end_date,omitempty"`
	// Page is the page number for pagination (starts from 1)
	Page int `json:"page"`
	// PageSize is the number of questions per page
	PageSize int `json:"page_size"`
	// IncludeUserNames includes user names in the response
	IncludeUserNames bool `json:"include_user_names,omitempty"`
}

func (q QuestionQuery) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.Page, validation.Min(1).Error("page must be at least 1")),
		validation.Field(&q.PageSize, validation.Min(1).Error("page size must be at least 1"), 
			validation.Max(100).Error("page size cannot exceed 100")),
		validation.Field(&q.Status, validation.When(q.Status != nil,
			validation.In(QuestionStatusPending, QuestionStatusAnswered, QuestionStatusClosed, QuestionStatusCancelled).Error("invalid status"))),
		validation.Field(&q.EndDate, validation.When(q.StartDate != nil && q.EndDate != nil,
			validation.By(q.validateDateRange))),
	)
}

// validateDateRange ensures end date is after start date
func (q QuestionQuery) validateDateRange(value interface{}) error {
	if q.StartDate != nil && q.EndDate != nil && q.EndDate.Before(*q.StartDate) {
		return validation.NewError("validation_date_range_invalid", "end date must be after start date")
	}
	return nil
}

// GetPage returns the page number, defaulting to 1 if not set
func (q QuestionQuery) GetPage() int {
	if q.Page <= 0 {
		return 1
	}
	return q.Page
}

// GetPageSize returns the page size, defaulting to 20 if not set
func (q QuestionQuery) GetPageSize() int {
	if q.PageSize <= 0 {
		return 20
	}
	return q.PageSize
}

// GetOffset calculates the offset for pagination
func (q QuestionQuery) GetOffset() int {
	return (q.GetPage() - 1) * q.GetPageSize()
}

// QuestionsListResponse represents a paginated list of questions
type QuestionsListResponse struct {
	Questions []Question `json:"questions"`
	Total     int64      `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	HasMore   bool       `json:"has_more"`
}

// QuestionStats represents statistics about questions
type QuestionStats struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Answered  int64 `json:"answered"`
	Closed    int64 `json:"closed"`
	Cancelled int64 `json:"cancelled"`
}
