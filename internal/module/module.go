package module

import (
	"aqlesia/internal/constants/model/dto"
	"context"
	"time"

	"github.com/google/uuid"
)

type User interface {
	Create(ctx context.Context, param dto.RegisterUser) (*dto.User, error)
	Update(ctx context.Context, id string, param dto.UpdateUser) (*dto.User, error)
	GetAll(ctx context.Context, page, pageSize int) (*dto.UserListResponse, error)
	GetByStatus(ctx context.Context, status string, page, pageSize int) (*dto.UserListResponse, error)
	Get(ctx context.Context, id string) (*dto.User, error)
	UpdateStatus(ctx context.Context, id string, status string) (*dto.User, error)
	DeleteUser(ctx context.Context, userId string) error
	ChangePassword(ctx context.Context, userID string, param dto.ChangePasswordRequest) (*dto.User, error)
	UpdateProfileImage(ctx context.Context, id string, imageURL string) error
}

type Appointment interface {
	Create(ctx context.Context, userID uuid.UUID, param dto.CreateAppointmentRequest) (*dto.Appointment, error)
	Update(ctx context.Context, id string, userID uuid.UUID, param dto.UpdateAppointmentRequest) (*dto.Appointment, error)
	Get(ctx context.Context, id string) (*dto.Appointment, error)
	GetUserAppointments(ctx context.Context, userID uuid.UUID, page, pageSize int) (*dto.AppointmentListResponse, error)
	GetAllAppointments(ctx context.Context, page, pageSize int) (*dto.AppointmentListResponse, error)
	MarkCompleted(ctx context.Context, id string, param dto.MarkAppointmentCompletedRequest) (*dto.Appointment, error)
	Cancel(ctx context.Context, id string, userID uuid.UUID) error
	Delete(ctx context.Context, id string) error
	GetAppointmentStats(ctx context.Context) (*dto.AppointmentStatsResponse, error)
	GetAvailableDates(ctx context.Context, startDate, endDate time.Time) ([]time.Time, error)
}

type Communion interface {
	Create(ctx context.Context, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error)
	Get(ctx context.Context, id string) (*dto.Communion, error)
	GetUserCommunions(ctx context.Context, userID uuid.UUID, page, pageSize int) (*dto.CommunionListResponse, error)
	GetAllCommunions(ctx context.Context, page, pageSize int) (*dto.CommunionListResponse, error)
	GetPendingCommunions(ctx context.Context, page, pageSize int) (*dto.CommunionListResponse, error)
	UpdateStatus(ctx context.Context, id string, adminUserID uuid.UUID, param dto.UpdateCommunionStatusRequest) (*dto.Communion, error)
	Update(ctx context.Context, id string, userID uuid.UUID, param dto.CreateCommunionRequest) (*dto.Communion, error)
	Delete(ctx context.Context, id string) error
}

type AvailableDates interface {
	// Public endpoints (for all users)
	GetAvailableDates(ctx context.Context, startDate, endDate time.Time, onlyAvailable bool) ([]dto.AvailableDate, error)
	GetAvailableDateByDate(ctx context.Context, dateStr string) (*dto.AvailableDate, error)

	// Admin/Manager endpoints
	GetAllAvailableDates(ctx context.Context, query dto.AvailableDateQuery) ([]dto.AvailableDate, error)
	CreateAvailableDate(ctx context.Context, param dto.CreateAvailableDate) (*dto.AvailableDate, error)
	UpdateAvailableDate(ctx context.Context, dateStr string, param dto.UpdateAvailableDate) (*dto.AvailableDate, error)
	DeleteAvailableDate(ctx context.Context, dateStr string) error
}

type Questions interface {
	// User endpoints
	CreateQuestion(ctx context.Context, userID uuid.UUID, param dto.CreateQuestion) (*dto.Question, error)
	GetMyQuestions(ctx context.Context, userID uuid.UUID) ([]dto.Question, error)
	UpdateMyQuestion(ctx context.Context, id string, userID uuid.UUID, param dto.UpdateQuestion) (*dto.Question, error)
	DeleteMyQuestion(ctx context.Context, id string, userID uuid.UUID) error
	GetMyQuestionStats(ctx context.Context, userID uuid.UUID) (*dto.QuestionStats, error)

	// Admin/Manager endpoints
	GetQuestions(ctx context.Context, query dto.QuestionQuery) (*dto.QuestionsListResponse, error)
	GetQuestion(ctx context.Context, id string) (*dto.Question, error)
	RespondToQuestion(ctx context.Context, id string, adminUserID uuid.UUID, param dto.UpdateQuestion) (*dto.Question, error)
	UpdateQuestionStatus(ctx context.Context, id string, adminUserID uuid.UUID, status dto.QuestionStatus) (*dto.Question, error)
	DeleteQuestion(ctx context.Context, id string) error
	GetQuestionStats(ctx context.Context) (*dto.QuestionStats, error)
}

type PasswordReset interface {
	// Initiate password reset process via Telegram OTP
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error)
	// Verify OTP and get reset token
	VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (*dto.VerifyOTPResponse, error)
	// Complete password reset with token
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error)
	// Generate Telegram link code for account linking
	LinkTelegram(ctx context.Context, req dto.TelegramLinkRequest) (*dto.TelegramLinkResponse, error)
	// Check Telegram verification status without sending OTP
	CheckTelegramVerification(ctx context.Context, req dto.TelegramLinkRequest) (map[string]interface{}, error)
	// Cleanup expired reset tokens (for cron job)
	CleanupExpiredTokens(ctx context.Context) error
}
