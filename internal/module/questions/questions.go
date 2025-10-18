package questions

import (
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"database/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type questions struct {
	questionsPersistent storage.Questions
	log                 logger.Logger
}

func Init(questionsPersistent storage.Questions, log logger.Logger) module.Questions {
	return &questions{
		questionsPersistent: questionsPersistent,
		log:                 log,
	}
}

// User endpoints

// CreateQuestion creates a new question for a user
func (q *questions) CreateQuestion(ctx context.Context, userID uuid.UUID, param dto.CreateQuestion) (*dto.Question, error) {
	q.log.Debug(ctx, "Creating question", zap.String("user-id", userID.String()))

	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		q.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	// Check if user has too many pending questions (business rule)
	userQuestions, err := q.questionsPersistent.GetByUser(ctx, userID)
	if err != nil {
		q.log.Error(ctx, "failed to get user questions", zap.Error(err))
		return nil, err
	}

	pendingCount := 0
	for _, question := range userQuestions {
		if question.Status == dto.QuestionStatusPending {
			pendingCount++
		}
	}

	// Business rule: Max 5 pending questions per user
	if pendingCount >= 5 {
		err = errors.ErrBusinessConstraintViolation.New("maximum pending questions limit reached (5)")
		q.log.Warn(ctx, "user has too many pending questions", 
			zap.String("user-id", userID.String()),
			zap.Int("pending-count", pendingCount))
		return nil, err
	}

	result, err := q.questionsPersistent.Create(ctx, userID, param)
	if err != nil {
		q.log.Error(ctx, "failed to create question", zap.Error(err))
		return nil, err
	}

	q.log.Info(ctx, "successfully created question", 
		zap.String("user-id", userID.String()),
		zap.String("question-id", result.ID.String()))
	return result, nil
}

// GetMyQuestions gets all questions for a specific user
func (q *questions) GetMyQuestions(ctx context.Context, userID uuid.UUID) ([]dto.Question, error) {
	q.log.Debug(ctx, "Getting user questions", zap.String("user-id", userID.String()))

	questions, err := q.questionsPersistent.GetByUser(ctx, userID)
	if err != nil {
		q.log.Error(ctx, "failed to get user questions", zap.Error(err))
		return nil, err
	}

	q.log.Debug(ctx, "successfully retrieved user questions", 
		zap.String("user-id", userID.String()),
		zap.Int("count", len(questions)))
	return questions, nil
}

// UpdateMyQuestion updates a user's own question (only pending questions)
func (q *questions) UpdateMyQuestion(ctx context.Context, id string, userID uuid.UUID, param dto.UpdateQuestion) (*dto.Question, error) {
	q.log.Debug(ctx, "Updating user question", 
		zap.String("question-id", id),
		zap.String("user-id", userID.String()))

	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		q.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return nil, err
	}

	// Users can only update the question text, not status or admin response
	if param.Status != nil || param.AdminResponse != nil {
		err = errors.ErrUnauthorized.New("users cannot update question status or admin response")
		q.log.Warn(ctx, "user attempted to update restricted fields", 
			zap.String("user-id", userID.String()),
			zap.String("question-id", id))
		return nil, err
	}

	if param.Question == nil {
		err = errors.ErrInvalidUserInput.New("question text is required")
		q.log.Warn(ctx, "missing question text", zap.String("question-id", id))
		return nil, err
	}

	result, err := q.questionsPersistent.UpdateQuestion(ctx, questionID, userID, *param.Question)
	if err != nil {
		q.log.Error(ctx, "failed to update user question", zap.Error(err))
		return nil, err
	}

	q.log.Info(ctx, "successfully updated user question", 
		zap.String("user-id", userID.String()),
		zap.String("question-id", id))
	return result, nil
}

// DeleteMyQuestion deletes a user's own pending question
func (q *questions) DeleteMyQuestion(ctx context.Context, id string, userID uuid.UUID) error {
	q.log.Debug(ctx, "Deleting user question", 
		zap.String("question-id", id),
		zap.String("user-id", userID.String()))

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return err
	}

	err = q.questionsPersistent.DeleteByUser(ctx, questionID, userID)
	if err != nil {
		q.log.Error(ctx, "failed to delete user question", zap.Error(err))
		return err
	}

	q.log.Info(ctx, "successfully deleted user question", 
		zap.String("user-id", userID.String()),
		zap.String("question-id", id))
	return nil
}

// GetMyQuestionStats gets question statistics for a specific user
func (q *questions) GetMyQuestionStats(ctx context.Context, userID uuid.UUID) (*dto.QuestionStats, error) {
	q.log.Debug(ctx, "Getting user question stats", zap.String("user-id", userID.String()))

	stats, err := q.questionsPersistent.GetUserStats(ctx, userID)
	if err != nil {
		q.log.Error(ctx, "failed to get user question stats", zap.Error(err))
		return nil, err
	}

	q.log.Debug(ctx, "successfully retrieved user question stats", 
		zap.String("user-id", userID.String()),
		zap.Int64("total", stats.Total))
	return stats, nil
}

// Admin/Manager endpoints

// GetQuestions gets questions with filtering and pagination (admin only)
func (q *questions) GetQuestions(ctx context.Context, query dto.QuestionQuery) (*dto.QuestionsListResponse, error) {
	q.log.Debug(ctx, "Getting questions with query", zap.Any("query", query))

	if err := query.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid query parameters")
		q.log.Warn(ctx, "validation failed", zap.Error(err), zap.Any("query", query))
		return nil, err
	}

	result, err := q.questionsPersistent.GetQuestions(ctx, query)
	if err != nil {
		q.log.Error(ctx, "failed to get questions", zap.Error(err))
		return nil, err
	}

	q.log.Debug(ctx, "successfully retrieved questions", 
		zap.Int("count", len(result.Questions)),
		zap.Int64("total", result.Total))
	return result, nil
}

// GetQuestion gets a specific question by ID (admin only)
func (q *questions) GetQuestion(ctx context.Context, id string) (*dto.Question, error) {
	q.log.Debug(ctx, "Getting question by ID", zap.String("question-id", id))

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return nil, err
	}

	result, err := q.questionsPersistent.Get(ctx, questionID)
	if err != nil {
		q.log.Error(ctx, "failed to get question", zap.Error(err))
		return nil, err
	}

	q.log.Debug(ctx, "successfully retrieved question", zap.String("question-id", id))
	return result, nil
}


// RespondToQuestion responds to a question with admin response (admin only)
func (q *questions) RespondToQuestion(ctx context.Context, id string, adminUserID uuid.UUID, param dto.UpdateQuestion) (*dto.Question, error) {
	q.log.Debug(ctx, "Responding to question", 
		zap.String("question-id", id),
		zap.String("admin-id", adminUserID.String()))

	if err := param.Validate(); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		q.log.Error(ctx, "validation failed", zap.Error(err), zap.Any("input", param))
		return nil, err
	}

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return nil, err
	}

	// Check if question exists and is still pending
	existing, err := q.questionsPersistent.Get(ctx, questionID)
	if err != nil {
		if isNoRecordFoundError(err) {
			err = errors.ErrNoRecordFound.New("question not found")
			q.log.Warn(ctx, "question not found", zap.String("question-id", id))
			return nil, err
		}
		q.log.Error(ctx, "failed to get existing question", zap.Error(err))
		return nil, err
	}

	// Business rule: Can only respond to pending questions
	if existing.Status != dto.QuestionStatusPending {
		err = errors.ErrBusinessConstraintViolation.New("can only respond to pending questions")
		q.log.Warn(ctx, "attempted to respond to non-pending question", 
			zap.String("question-id", id),
			zap.String("status", string(existing.Status)))
		return nil, err
	}

	// Set status to answered and provide admin response
	status := dto.QuestionStatusAnswered
	result, err := q.questionsPersistent.UpdateStatus(ctx, questionID, status, param.AdminResponse, &adminUserID)
	if err != nil {
		q.log.Error(ctx, "failed to respond to question", zap.Error(err))
		return nil, err
	}

	q.log.Info(ctx, "successfully responded to question", 
		zap.String("admin-id", adminUserID.String()),
		zap.String("question-id", id))
	return result, nil
}

// UpdateQuestionStatus updates a question's status (admin only)
func (q *questions) UpdateQuestionStatus(ctx context.Context, id string, adminUserID uuid.UUID, status dto.QuestionStatus) (*dto.Question, error) {
	q.log.Debug(ctx, "Updating question status", 
		zap.String("question-id", id),
		zap.String("admin-id", adminUserID.String()),
		zap.String("status", string(status)))

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return nil, err
	}

	// Validate status
	validStatuses := []dto.QuestionStatus{
		dto.QuestionStatusPending,
		dto.QuestionStatusAnswered,
		dto.QuestionStatusClosed,
		dto.QuestionStatusCancelled,
	}
	isValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		err = errors.ErrInvalidUserInput.New("invalid question status")
		q.log.Warn(ctx, "invalid status", zap.String("status", string(status)))
		return nil, err
	}

	var respondedBy *uuid.UUID
	if status == dto.QuestionStatusAnswered || status == dto.QuestionStatusClosed {
		respondedBy = &adminUserID
	}

	result, err := q.questionsPersistent.UpdateStatus(ctx, questionID, status, nil, respondedBy)
	if err != nil {
		q.log.Error(ctx, "failed to update question status", zap.Error(err))
		return nil, err
	}

	q.log.Info(ctx, "successfully updated question status", 
		zap.String("admin-id", adminUserID.String()),
		zap.String("question-id", id),
		zap.String("status", string(status)))
	return result, nil
}

// DeleteQuestion deletes a question (admin only)
func (q *questions) DeleteQuestion(ctx context.Context, id string) error {
	q.log.Debug(ctx, "Deleting question", zap.String("question-id", id))

	// Parse question ID
	questionID, err := uuid.Parse(id)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid question ID")
		q.log.Warn(ctx, "invalid question ID", zap.String("id", id))
		return err
	}

	err = q.questionsPersistent.Delete(ctx, questionID)
	if err != nil {
		q.log.Error(ctx, "failed to delete question", zap.Error(err))
		return err
	}

	q.log.Info(ctx, "successfully deleted question", zap.String("question-id", id))
	return nil
}

// GetQuestionStats gets overall question statistics (admin only)
func (q *questions) GetQuestionStats(ctx context.Context) (*dto.QuestionStats, error) {
	q.log.Debug(ctx, "Getting question stats")

	stats, err := q.questionsPersistent.GetStats(ctx)
	if err != nil {
		q.log.Error(ctx, "failed to get question stats", zap.Error(err))
		return nil, err
	}

	q.log.Debug(ctx, "successfully retrieved question stats", zap.Int64("total", stats.Total))
	return stats, nil
}

// Helper function to check if error is a "no record found" error
func isNoRecordFoundError(err error) bool {
	return err == sql.ErrNoRows || errors.IsErrorCode(err, errors.NoRecordFound)
}
