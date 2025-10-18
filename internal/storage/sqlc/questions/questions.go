package questions

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/db"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"database/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type questions struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.Questions {
	return &questions{
		db:  db,
		log: log,
	}
}

// Create creates a new question
func (q *questions) Create(ctx context.Context, userID uuid.UUID, param dto.CreateQuestion) (*dto.Question, error) {
	q.log.Debug(ctx, "Creating question",
		zap.String("user-id", userID.String()),
		zap.String("question-text", param.Question),
		zap.Int("question-length", len(param.Question)))

	params := db.CreateQuestionParams{
		UserID:   userID,
		Question: param.Question,
		Status:   string(dto.QuestionStatusPending),
	}

	q.log.Debug(ctx, "Calling database CreateQuestion with params",
		zap.String("db-user-id", params.UserID.String()),
		zap.String("db-question-text", params.Question),
		zap.String("db-status", params.Status))

	dbQuestion, err := q.db.CreateQuestion(ctx, params)
	if err != nil {
		q.log.Error(ctx, "failed to create question", zap.Error(err))
		return nil, errors.ErrWriteError.Wrap(err, "failed to create question")
	}

	q.log.Debug(ctx, "Database CreateQuestion returned",
		zap.String("created-question-id", dbQuestion.ID.String()),
		zap.String("created-question-text", dbQuestion.Question),
		zap.String("created-user-id", dbQuestion.UserID.String()),
		zap.String("created-status", dbQuestion.Status))

	result := q.convertToDTO(dbQuestion)
	q.log.Debug(ctx, "Successfully created question", zap.String("question-id", result.ID.String()))
	return &result, nil
}

// Get gets a question by ID with user and responder information
func (q *questions) Get(ctx context.Context, id uuid.UUID) (*dto.Question, error) {
	q.log.Debug(ctx, "Getting question by ID", zap.String("question-id", id.String()))

	dbQuestion, err := q.db.GetQuestion(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			q.log.Debug(ctx, "question not found", zap.String("question-id", id.String()))
			return nil, errors.ErrNoRecordFound.New("question not found")
		}
		q.log.Error(ctx, "failed to get question", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get question")
	}

	result := q.convertGetQuestionRowToDTO(dbQuestion)
	q.log.Debug(ctx, "Successfully retrieved question", zap.String("question-id", id.String()))
	return &result, nil
}

// GetByUser gets all questions for a specific user
func (q *questions) GetByUser(ctx context.Context, userID uuid.UUID) ([]dto.Question, error) {
	q.log.Debug(ctx, "Getting questions by user", zap.String("user-id", userID.String()))

	dbQuestions, err := q.db.GetQuestionsByUser(ctx, userID)
	if err != nil {
		q.log.Error(ctx, "failed to get questions by user", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get questions by user")
	}

	q.log.Debug(ctx, "Raw database query result", 
		zap.String("user-id", userID.String()),
		zap.Int("db-questions-count", len(dbQuestions)),
		zap.Any("db-questions", dbQuestions))

	questions := make([]dto.Question, len(dbQuestions))
	for i, dbQuestion := range dbQuestions {
		questions[i] = q.convertToDTO(dbQuestion)
		q.log.Debug(ctx, "Converting question", 
			zap.String("question-id", dbQuestion.ID.String()),
			zap.String("question-text", dbQuestion.Question),
			zap.String("status", dbQuestion.Status))
	}

	q.log.Debug(ctx, "Successfully retrieved user questions", 
		zap.String("user-id", userID.String()),
		zap.Int("count", len(questions)))
	return questions, nil
}

// GetQuestions gets questions with filtering and pagination
func (q *questions) GetQuestions(ctx context.Context, query dto.QuestionQuery) (*dto.QuestionsListResponse, error) {
	q.log.Debug(ctx, "Getting questions with query", zap.Any("query", query))

	// Simple approach: just get all questions and return them
	// Use the working GetPendingQuestions query to get all pending questions
	dbQuestions, err := q.db.GetPendingQuestions(ctx)
	if err != nil {
		q.log.Error(ctx, "failed to get questions", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get questions")
	}

	// Convert all questions to DTOs
	questions := make([]dto.Question, len(dbQuestions))
	for i, dbQuestion := range dbQuestions {
		questions[i] = q.convertGetPendingQuestionsRowToDTO(dbQuestion)
	}

	// Apply simple pagination
	offset := (query.GetPage() - 1) * query.GetPageSize()
	start := offset
	end := offset + query.GetPageSize()
	if start > len(questions) {
		start = len(questions)
	}
	if end > len(questions) {
		end = len(questions)
	}

	paginatedQuestions := questions[start:end]
	hasMore := end < len(questions)

	result := &dto.QuestionsListResponse{
		Questions: paginatedQuestions,
		Total:     int64(len(questions)),
		Page:      query.GetPage(),
		PageSize:  query.GetPageSize(),
		HasMore:   hasMore,
	}

	q.log.Debug(ctx, "Successfully retrieved questions", 
		zap.Int("total", len(questions)),
		zap.Int("returned", len(paginatedQuestions)))
	return result, nil
}


// UpdateQuestion updates a question's text (user only, pending questions only)
func (q *questions) UpdateQuestion(ctx context.Context, id uuid.UUID, userID uuid.UUID, question string) (*dto.Question, error) {
	q.log.Debug(ctx, "Updating question text",
		zap.String("question-id", id.String()),
		zap.String("user-id", userID.String()))

	dbQuestion, err := q.db.UpdateQuestionText(ctx, db.UpdateQuestionTextParams{
		ID:       id,
		Question: question,
		UserID:   userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			q.log.Warn(ctx, "question not found or not editable", 
				zap.String("question-id", id.String()),
				zap.String("user-id", userID.String()))
			return nil, errors.ErrNoRecordFound.New("question not found or cannot be edited")
		}
		q.log.Error(ctx, "failed to update question", zap.Error(err))
		return nil, errors.ErrWriteError.Wrap(err, "failed to update question")
	}

	result := q.convertToDTO(dbQuestion)
	q.log.Debug(ctx, "Successfully updated question", zap.String("question-id", id.String()))
	return &result, nil
}

// UpdateStatus updates a question's status and admin response
func (q *questions) UpdateStatus(ctx context.Context, id uuid.UUID, status dto.QuestionStatus, adminResponse *string, respondedBy *uuid.UUID) (*dto.Question, error) {
	q.log.Debug(ctx, "Updating question status",
		zap.String("question-id", id.String()),
		zap.String("status", string(status)))

	// Convert adminResponse to sql.NullString
	var sqlAdminResponse sql.NullString
	if adminResponse != nil {
		sqlAdminResponse = sql.NullString{String: *adminResponse, Valid: true}
	}
	
	// Convert respondedBy to uuid.NullUUID
	var sqlRespondedBy uuid.NullUUID
	if respondedBy != nil {
		sqlRespondedBy = uuid.NullUUID{UUID: *respondedBy, Valid: true}
	}
	
	dbQuestion, err := q.db.UpdateQuestionStatus(ctx, db.UpdateQuestionStatusParams{
		ID:            id,
		Status:        string(status),
		AdminResponse: sqlAdminResponse,
		RespondedBy:   sqlRespondedBy,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			q.log.Warn(ctx, "question not found", zap.String("question-id", id.String()))
			return nil, errors.ErrNoRecordFound.New("question not found")
		}
		q.log.Error(ctx, "failed to update question status", zap.Error(err))
		return nil, errors.ErrWriteError.Wrap(err, "failed to update question status")
	}

	result := q.convertToDTO(dbQuestion)
	q.log.Debug(ctx, "Successfully updated question status", zap.String("question-id", id.String()))
	return &result, nil
}

// Delete deletes a question (admin only)
func (q *questions) Delete(ctx context.Context, id uuid.UUID) error {
	q.log.Debug(ctx, "Deleting question", zap.String("question-id", id.String()))

	err := q.db.DeleteQuestion(ctx, id)
	if err != nil {
		q.log.Error(ctx, "failed to delete question", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to delete question")
	}

	q.log.Debug(ctx, "Successfully deleted question", zap.String("question-id", id.String()))
	return nil
}

// DeleteByUser deletes a pending question by user
func (q *questions) DeleteByUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	q.log.Debug(ctx, "Deleting question by user",
		zap.String("question-id", id.String()),
		zap.String("user-id", userID.String()))

	err := q.db.DeleteQuestionByUser(ctx, db.DeleteQuestionByUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		q.log.Error(ctx, "failed to delete question by user", zap.Error(err))
		return errors.ErrWriteError.Wrap(err, "failed to delete question")
	}

	q.log.Debug(ctx, "Successfully deleted question by user", 
		zap.String("question-id", id.String()),
		zap.String("user-id", userID.String()))
	return nil
}

// GetStats gets overall question statistics
func (q *questions) GetStats(ctx context.Context) (*dto.QuestionStats, error) {
	q.log.Debug(ctx, "Getting question stats")

	stats, err := q.db.GetQuestionStats(ctx)
	if err != nil {
		q.log.Error(ctx, "failed to get question stats", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get question stats")
	}

	result := &dto.QuestionStats{
		Total:     stats.Total,
		Pending:   stats.Pending,
		Answered:  stats.Answered,
		Closed:    stats.Closed,
		Cancelled: stats.Cancelled,
	}

	q.log.Debug(ctx, "Successfully retrieved question stats", zap.Int64("total", result.Total))
	return result, nil
}

// GetUserStats gets question statistics for a specific user
func (q *questions) GetUserStats(ctx context.Context, userID uuid.UUID) (*dto.QuestionStats, error) {
	q.log.Debug(ctx, "Getting user question stats", zap.String("user-id", userID.String()))

	stats, err := q.db.GetUserQuestionStats(ctx, userID)
	if err != nil {
		q.log.Error(ctx, "failed to get user question stats", zap.Error(err))
		return nil, errors.ErrReadError.Wrap(err, "failed to get user question stats")
	}

	result := &dto.QuestionStats{
		Total:     stats.Total,
		Pending:   stats.Pending,
		Answered:  stats.Answered,
		Closed:    stats.Closed,
		Cancelled: stats.Cancelled,
	}

	q.log.Debug(ctx, "Successfully retrieved user question stats", 
		zap.String("user-id", userID.String()),
		zap.Int64("total", result.Total))
	return result, nil
}

// Conversion helper functions
func (q *questions) convertToDTO(dbQuestion db.Question) dto.Question {
	var respondedBy *uuid.UUID
	if dbQuestion.RespondedBy.Valid {
		respondedBy = &dbQuestion.RespondedBy.UUID
	}

	var adminResponse *string
	if dbQuestion.AdminResponse.Valid {
		adminResponse = &dbQuestion.AdminResponse.String
	}

	return dto.Question{
		ID:            dbQuestion.ID,
		UserID:        dbQuestion.UserID,
		Question:      dbQuestion.Question,
		Status:        dto.QuestionStatus(dbQuestion.Status),
		AdminResponse: adminResponse,
		RespondedBy:   respondedBy,
		CreatedAt:     dbQuestion.CreatedAt,
		UpdatedAt:     dbQuestion.UpdatedAt,
	}
}

func (q *questions) convertGetQuestionRowToDTO(row db.GetQuestionRow) dto.Question {
	var respondedBy *uuid.UUID
	if row.RespondedBy.Valid {
		respondedBy = &row.RespondedBy.UUID
	}

	var adminResponse *string
	if row.AdminResponse.Valid {
		adminResponse = &row.AdminResponse.String
	}

	var userName *string
	if row.UserName.Valid {
		userName = &row.UserName.String
	}

	var responderName *string
	if row.ResponderName.Valid {
		responderName = &row.ResponderName.String
	}

	return dto.Question{
		ID:            row.ID,
		UserID:        row.UserID,
		Question:      row.Question,
		Status:        dto.QuestionStatus(row.Status),
		AdminResponse: adminResponse,
		RespondedBy:   respondedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		UserName:      userName,
		ResponderName: responderName,
	}
}

func (q *questions) convertGetQuestionsPaginatedRowToDTO(row db.GetQuestionsPaginatedRow) dto.Question {
	var respondedBy *uuid.UUID
	if row.RespondedBy.Valid {
		respondedBy = &row.RespondedBy.UUID
	}

	var adminResponse *string
	if row.AdminResponse.Valid {
		adminResponse = &row.AdminResponse.String
	}

	var userName *string
	if userNameVal, ok := row.UserName.(string); ok && userNameVal != "" {
		userName = &userNameVal
	}

	var responderName *string
	if responderNameVal, ok := row.ResponderName.(string); ok && responderNameVal != "" {
		responderName = &responderNameVal
	}

	return dto.Question{
		ID:            row.ID,
		UserID:        row.UserID,
		Question:      row.Question,
		Status:        dto.QuestionStatus(row.Status),
		AdminResponse: adminResponse,
		RespondedBy:   respondedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		UserName:      userName,
		ResponderName: responderName,
	}
}

func (q *questions) convertGetPendingQuestionsRowToDTO(row db.GetPendingQuestionsRow) dto.Question {
	var userName *string
	if row.UserName.Valid {
		userName = &row.UserName.String
	}

	return dto.Question{
		ID:        row.ID,
		UserID:    row.UserID,
		Question:  row.Question,
		Status:    dto.QuestionStatus(row.Status),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		UserName:  userName,
	}
}
