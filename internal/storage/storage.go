package storage

import (
	"aqlesia/internal/constants/model/db"
	"aqlesia/internal/constants/model/dto"
	"context"
	"time"

	"github.com/google/uuid"
)

type User interface {
	Create(ctx context.Context, param dto.RegisterUser) (*dto.User, error)
	Update(ctx context.Context, id uuid.UUID, param dto.UpdateUser) (*dto.User, error)
	CheckUserExists(ctx context.Context, param dto.RegisterUser) (bool, error)
	GetAll(ctx context.Context, page, pageSize int) ([]dto.User, int64, error)
	GetByStatus(ctx context.Context, status string, page, pageSize int) ([]dto.User, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*dto.User, error)
	GetUserByPhone(ctx context.Context, phoneNumber string) (*dto.User, error)
	GetUserByPhoneWithPassword(ctx context.Context, phoneNumber string) (*db.User, error)
	UpdateStatus(ctx context.Context, userID uuid.UUID, status string) (*dto.User, error)
	DeleteUser(ctx context.Context, userId uuid.UUID) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) (*dto.User, error)
	
	// Telegram integration methods
	UpdateTelegramInfo(ctx context.Context, telegramID string, verified bool, phoneNumber string) (*dto.User, error)
	GetUserByTelegramID(ctx context.Context, telegramID string) (*db.User, error)
	
	// Password reset methods
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*dto.PasswordResetToken, error)
	GetValidPasswordResetToken(ctx context.Context, tokenHash string) (*dto.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	ResetUserPassword(ctx context.Context, hashedPassword string, userID uuid.UUID) (*dto.User, error)
	CleanupExpiredResetTokens(ctx context.Context) error
}

type Appointment interface {
	Create(ctx context.Context, userID uuid.UUID, param dto.CreateAppointmentRequest) (*dto.Appointment, error)
	Update(ctx context.Context, id uuid.UUID, param dto.UpdateAppointmentRequest) (*dto.Appointment, error)
	Get(ctx context.Context, id uuid.UUID) (*dto.Appointment, error)
	GetWithUser(ctx context.Context, id uuid.UUID) (*dto.Appointment, error)
	GetUserAppointments(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]dto.Appointment, int64, error)
	GetAllAppointments(ctx context.Context, page, pageSize int) ([]dto.Appointment, int64, error)
	GetUserActiveAppointment(ctx context.Context, userID uuid.UUID) (*dto.Appointment, error)
	CountAppointmentsByDate(ctx context.Context, date time.Time) (int64, error)
	MarkCompleted(ctx context.Context, id uuid.UUID, notes *string) (*dto.Appointment, error)
	Cancel(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAppointmentStats(ctx context.Context) (*dto.AppointmentStatsResponse, error)
	GetAppointmentsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]dto.Appointment, error)
	GetUpcomingAppointments(ctx context.Context, limit int) ([]dto.Appointment, error)
}

type AvailableSlot interface {
	GetAvailableSlotsWithCapacity(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableSlot, error)
	UpsertSlot(ctx context.Context, slotDate time.Time, maxCapacity int32) error
	DeactivateSlot(ctx context.Context, slotDate time.Time) error
	ActivateSlot(ctx context.Context, slotDate time.Time) error
	DeleteOldSlots(ctx context.Context, beforeDate time.Time) error
	GetSlotByDate(ctx context.Context, slotDate time.Time) (*dto.AvailableSlot, error)
}

type AvailableDates interface {
	// Basic operations
	GetAvailableDatesWithCapacity(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error)
	GetAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error)
	GetAvailableDateByDate(ctx context.Context, slotDate time.Time) (*dto.AvailableDate, error)
	UpsertAvailableDate(ctx context.Context, slotDate time.Time, maxCapacity int32) error
	DeactivateDate(ctx context.Context, slotDate time.Time) error
	ActivateDate(ctx context.Context, slotDate time.Time) error
	DeleteOldDates(ctx context.Context, beforeDate time.Time) error
	
	// Admin/Manager operations
	GetAllAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]dto.AvailableDate, error)
	CreateAvailableDate(ctx context.Context, param dto.CreateAvailableDate) (*dto.AvailableDate, error)
	UpdateAvailableDate(ctx context.Context, slotDate time.Time, param dto.UpdateAvailableDate) (*dto.AvailableDate, error)
	DeleteAvailableDate(ctx context.Context, slotDate time.Time) error
}

type Communion interface {
	Create(ctx context.Context, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error)
	Get(ctx context.Context, id uuid.UUID) (*dto.Communion, error)
	GetWithUser(ctx context.Context, id uuid.UUID) (*dto.Communion, error)
	GetUserCommunion(ctx context.Context, userID uuid.UUID) (*dto.Communion, error)
	GetAll(ctx context.Context, page, pageSize int) ([]dto.Communion, int64, error)
	GetPending(ctx context.Context, page, pageSize int) ([]dto.Communion, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedByUserID uuid.UUID) (*dto.Communion, error)
	Update(ctx context.Context, id uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Questions interface {
	// Basic operations
	Create(ctx context.Context, userID uuid.UUID, param dto.CreateQuestion) (*dto.Question, error)
	Get(ctx context.Context, id uuid.UUID) (*dto.Question, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]dto.Question, error)
	GetQuestions(ctx context.Context, query dto.QuestionQuery) (*dto.QuestionsListResponse, error)
	
	// Update operations
	UpdateQuestion(ctx context.Context, id uuid.UUID, userID uuid.UUID, question string) (*dto.Question, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status dto.QuestionStatus, adminResponse *string, respondedBy *uuid.UUID) (*dto.Question, error)
	
	// Delete operations
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	
	// Stats
	GetStats(ctx context.Context) (*dto.QuestionStats, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*dto.QuestionStats, error)
}
