package questions

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/constants"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/glue/routing"
	"aqlesia/internal/handler/middleware"
	middleware2 "aqlesia/internal/middleware"
	"aqlesia/internal/module"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Init(grp *gin.RouterGroup, log logger.Logger, questionsModule module.Questions, jwtManager *auth.JWTManager, authzMiddleware *middleware2.AuthorizationMiddleware, userStorage storage.User) {
	// Create a REST handler for questions
	questionsHandler := newQuestionsHandler(log, questionsModule)
	
	// Set up authentication middleware
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)

	// User questions endpoints under /questions
	questionsGroup := grp.Group("questions")
	questionsRoutes := []routing.Router{
		// User endpoints (require authentication)
		{
			Method:      http.MethodPost,
			Path:        "",
			Handler:     questionsHandler.CreateQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users can create questions
		},
		{
			Method:      http.MethodGet,
			Path:        "/my",
			Handler:     questionsHandler.GetMyQuestions,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users can view their own questions
		},
		{
			Method:      http.MethodPut,
			Path:        "/my/:id",
			Handler:     questionsHandler.UpdateMyQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users can update their own questions
		},
		{
			Method:      http.MethodDelete,
			Path:        "/my/:id",
			Handler:     questionsHandler.DeleteMyQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users can delete their own questions
		},
		{
			Method:      http.MethodGet,
			Path:        "/my/stats",
			Handler:     questionsHandler.GetMyQuestionStats,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users can view their own stats
		},
		// Admin endpoints (require admin/manager authorization)
		{
			Method:      http.MethodGet,
			Path:        "",
			Handler:     questionsHandler.GetQuestions,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:list")},
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Handler:     questionsHandler.GetQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:read")},
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/respond",
			Handler:     questionsHandler.RespondToQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:respond")},
		},
		{
			Method:      http.MethodPatch,
			Path:        "/:id/status",
			Handler:     questionsHandler.UpdateQuestionStatus,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:update")},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Handler:     questionsHandler.DeleteQuestion,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:delete")},
		},
		{
			Method:      http.MethodGet,
			Path:        "/stats",
			Handler:     questionsHandler.GetQuestionStats,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("questions:stats")},
		},
	}
	routing.RegisterRoutes(questionsGroup, questionsRoutes)
}

// REST handler for questions
type questionsHandler struct {
	log             logger.Logger
	questionsModule module.Questions
}

func newQuestionsHandler(log logger.Logger, questionsModule module.Questions) *questionsHandler {
	return &questionsHandler{
		log:             log,
		questionsModule: questionsModule,
	}
}

// User endpoints

// CreateQuestion creates a new question for the authenticated user
// @Summary Create new question
// @Description Create a new question for the authenticated user
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateQuestion true "Create question request"
// @Success 201 {object} model.Response{data=dto.Question} "Successfully created question"
// @Failure 400 {object} model.ErrorResponse "Bad request - validation errors or maximum pending questions reached"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions [post]
func (h *questionsHandler) CreateQuestion(ctx *gin.Context) {
	h.log.Debug(ctx, "Starting CreateQuestion handler")

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	h.log.Debug(ctx, "Auth context found", 
		zap.String("user_id", authContext.UserID),
		zap.String("phone_number", authContext.PhoneNumber))

	userID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	h.log.Debug(ctx, "Parsed user ID", zap.String("parsed_user_id", userID.String()))

	// Parse request body
	var param dto.CreateQuestion
	if err := ctx.ShouldBindJSON(&param); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	h.log.Debug(ctx, "Parsed request body", 
		zap.String("question_text", param.Question),
		zap.Int("question_length", len(param.Question)))

	// Create question
	h.log.Debug(ctx, "Calling questionsModule.CreateQuestion")
	question, err := h.questionsModule.CreateQuestion(ctx, userID, param)
	if err != nil {
		h.log.Error(ctx, "Failed to create question", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	h.log.Debug(ctx, "Successfully created question", 
		zap.String("question_id", question.ID.String()),
		zap.String("question_text", question.Question))

	constants.SuccessResponse(ctx, http.StatusCreated, question, nil)
}

// GetMyQuestions gets all questions for the authenticated user
// @Summary Get my questions
// @Description Get all questions for the authenticated user
// @Tags questions
// @Security BearerAuth
// @Success 200 {object} model.Response{data=[]dto.Question} "Successfully retrieved questions"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/my [get]
func (h *questionsHandler) GetMyQuestions(ctx *gin.Context) {
	h.log.Debug(ctx, "Starting GetMyQuestions handler")

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	h.log.Debug(ctx, "Auth context found for GetMyQuestions", 
		zap.String("user_id", authContext.UserID),
		zap.String("phone_number", authContext.PhoneNumber))

	userID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	h.log.Debug(ctx, "Parsed user ID for GetMyQuestions", zap.String("parsed_user_id", userID.String()))

	// Get user questions
	h.log.Debug(ctx, "Calling questionsModule.GetMyQuestions")
	questions, err := h.questionsModule.GetMyQuestions(ctx, userID)
	if err != nil {
		h.log.Error(ctx, "Failed to get user questions", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	h.log.Debug(ctx, "Successfully retrieved user questions", 
		zap.Int("questions_count", len(questions)))

	constants.SuccessResponse(ctx, http.StatusOK, questions, nil)
}

// UpdateMyQuestion updates a user's own question
// @Summary Update my question
// @Description Update a user's own pending question
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Param request body dto.UpdateQuestion true "Update question request"
// @Success 200 {object} model.Response{data=dto.Question} "Successfully updated question"
// @Failure 400 {object} model.ErrorResponse "Bad request - validation errors or invalid question ID"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - cannot update non-pending questions"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/my/{id} [put]
func (h *questionsHandler) UpdateMyQuestion(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Updating user question", zap.String("question-id", questionID))

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	userID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	// Parse request body
	var param dto.UpdateQuestion
	if err := ctx.ShouldBindJSON(&param); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Update question
	question, err := h.questionsModule.UpdateMyQuestion(ctx, questionID, userID, param)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, question, nil)
}

// DeleteMyQuestion deletes a user's own pending question
// @Summary Delete my question
// @Description Delete a user's own pending question
// @Tags questions
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Success 200 {object} model.Response "Successfully deleted question"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid question ID"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - cannot delete non-pending questions"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/my/{id} [delete]
func (h *questionsHandler) DeleteMyQuestion(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Deleting user question", zap.String("question-id", questionID))

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	userID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	// Delete question
	err = h.questionsModule.DeleteMyQuestion(ctx, questionID, userID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Question deleted successfully", nil)
}

// GetMyQuestionStats gets question statistics for the authenticated user
// @Summary Get my question statistics
// @Description Get question statistics for the authenticated user
// @Tags questions
// @Security BearerAuth
// @Success 200 {object} model.Response{data=dto.QuestionStats} "Successfully retrieved statistics"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/my/stats [get]
func (h *questionsHandler) GetMyQuestionStats(ctx *gin.Context) {
	h.log.Debug(ctx, "Getting user question stats")

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	userID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	// Get user question stats
	stats, err := h.questionsModule.GetMyQuestionStats(ctx, userID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, stats, nil)
}

// Admin endpoints

// GetQuestions gets questions with filtering and pagination (admin only)
// @Summary Get questions (Admin/Manager)
// @Description Get questions with filtering and pagination for admin/manager use
// @Tags questions
// @Security BearerAuth
// @Param user_id query string false "Filter by user ID"
// @Param status query string false "Filter by status (pending, answered, closed, cancelled)"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Param include_user_names query bool false "Include user names in response (default: false)"
// @Success 200 {object} model.Response{data=dto.QuestionsListResponse} "Successfully retrieved questions"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid parameters"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions [get]
func (h *questionsHandler) GetQuestions(ctx *gin.Context) {
	h.log.Debug(ctx, "Getting questions with filters")

	// Parse query parameters
	var query dto.QuestionQuery

	// User ID filter
	if userIDStr := ctx.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			query.UserID = &userID
		} else {
			err = errors.ErrInvalidUserInput.New("invalid user_id format")
			h.log.Warn(ctx, "invalid user_id parameter", zap.String("user_id", userIDStr))
			_ = ctx.Error(err)
			return
		}
	}

	// Status filter
	if statusStr := ctx.Query("status"); statusStr != "" {
		status := dto.QuestionStatus(statusStr)
		query.Status = &status
	}

	// Date filters
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			query.StartDate = &startDate
		} else {
			err = errors.ErrInvalidUserInput.New("invalid start_date format, expected YYYY-MM-DD")
			h.log.Warn(ctx, "invalid start_date parameter", zap.String("start_date", startDateStr))
			_ = ctx.Error(err)
			return
		}
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			query.EndDate = &endDate
		} else {
			err = errors.ErrInvalidUserInput.New("invalid end_date format, expected YYYY-MM-DD")
			h.log.Warn(ctx, "invalid end_date parameter", zap.String("end_date", endDateStr))
			_ = ctx.Error(err)
			return
		}
	}

	// Pagination
	if pageStr := ctx.DefaultQuery("page", "1"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			query.Page = page
		} else {
			err = errors.ErrInvalidUserInput.New("invalid page parameter")
			h.log.Warn(ctx, "invalid page parameter", zap.String("page", pageStr))
			_ = ctx.Error(err)
			return
		}
	}

	if pageSizeStr := ctx.DefaultQuery("page_size", "20"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			query.PageSize = pageSize
		} else {
			err = errors.ErrInvalidUserInput.New("invalid page_size parameter")
			h.log.Warn(ctx, "invalid page_size parameter", zap.String("page_size", pageSizeStr))
			_ = ctx.Error(err)
			return
		}
	}

	// Include user names
	if includeStr := ctx.DefaultQuery("include_user_names", "false"); includeStr != "" {
		if include, err := strconv.ParseBool(includeStr); err == nil {
			query.IncludeUserNames = include
		}
	}

	// Get questions
	result, err := h.questionsModule.GetQuestions(ctx, query)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, result, nil)
}

// GetQuestion gets a specific question by ID (admin only)
// @Summary Get question by ID (Admin/Manager)
// @Description Get a specific question by ID for admin/manager use
// @Tags questions
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Success 200 {object} model.Response{data=dto.Question} "Successfully retrieved question"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid question ID"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/{id} [get]
func (h *questionsHandler) GetQuestion(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Getting question by ID", zap.String("question-id", questionID))

	// Get question
	question, err := h.questionsModule.GetQuestion(ctx, questionID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, question, nil)
}


// RespondToQuestion responds to a question with admin response (admin only)
// @Summary Respond to question (Admin/Manager)
// @Description Respond to a pending question with admin response
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Param request body dto.UpdateQuestion true "Response request (admin_response field required)"
// @Success 200 {object} model.Response{data=dto.Question} "Successfully responded to question"
// @Failure 400 {object} model.ErrorResponse "Bad request - validation errors or question not pending"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/{id}/respond [post]
func (h *questionsHandler) RespondToQuestion(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Responding to question", zap.String("question-id", questionID))

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	adminUserID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	// Parse request body
	var param dto.UpdateQuestion
	if err := ctx.ShouldBindJSON(&param); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Respond to question
	question, err := h.questionsModule.RespondToQuestion(ctx, questionID, adminUserID, param)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, question, nil)
}

// UpdateQuestionStatus updates a question's status (admin only)
// @Summary Update question status (Admin/Manager)
// @Description Update a question's status
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Param request body object{status=string} true "Status update request"
// @Success 200 {object} model.Response{data=dto.Question} "Successfully updated question status"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid status"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/{id}/status [patch]
func (h *questionsHandler) UpdateQuestionStatus(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Updating question status", zap.String("question-id", questionID))

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		h.log.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	adminUserID, err := uuid.Parse(authContext.UserID)
	if err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		h.log.Warn(ctx, "invalid user ID", zap.String("user_id", authContext.UserID))
		_ = ctx.Error(err)
		return
	}

	// Parse request body
	var request struct {
		Status string `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		err = errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Update question status
	status := dto.QuestionStatus(request.Status)
	question, err := h.questionsModule.UpdateQuestionStatus(ctx, questionID, adminUserID, status)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, question, nil)
}

// DeleteQuestion deletes a question (admin only)
// @Summary Delete question (Admin/Manager)
// @Description Delete a question permanently
// @Tags questions
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Success 200 {object} model.Response "Successfully deleted question"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid question ID"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Question not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/{id} [delete]
func (h *questionsHandler) DeleteQuestion(ctx *gin.Context) {
	questionID := ctx.Param("id")
	h.log.Debug(ctx, "Deleting question", zap.String("question-id", questionID))

	// Delete question
	err := h.questionsModule.DeleteQuestion(ctx, questionID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Question deleted successfully", nil)
}

// GetQuestionStats gets overall question statistics (admin only)
// @Summary Get question statistics (Admin/Manager)
// @Description Get overall question statistics for admin/manager dashboard
// @Tags questions
// @Security BearerAuth
// @Success 200 {object} model.Response{data=dto.QuestionStats} "Successfully retrieved statistics"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /questions/stats [get]
func (h *questionsHandler) GetQuestionStats(ctx *gin.Context) {
	h.log.Debug(ctx, "Getting question stats")

	// Get question stats
	stats, err := h.questionsModule.GetQuestionStats(ctx)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, stats, nil)
}
