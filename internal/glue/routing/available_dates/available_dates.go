package available_dates

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
	"go.uber.org/zap"
)

func Init(grp *gin.RouterGroup, log logger.Logger, availableDatesModule module.AvailableDates, jwtManager *auth.JWTManager, authzMiddleware *middleware2.AuthorizationMiddleware, userStorage storage.User) {
	// Create a REST handler for available dates
	availableDatesHandler := newAvailableDatesHandler(log, availableDatesModule)
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)

	// All available dates endpoints under /available-dates
	availableDatesGroup := grp.Group("available-dates")
	availableDatesRoutes := []routing.Router{
		// List available dates - all authenticated users can list
		{
			Method:      http.MethodGet,
			Path:        "",
			Handler:     availableDatesHandler.GetAvailableDates,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("available_dates:list")},
		},
		// Get specific date - all authenticated users can read
		{
			Method:      http.MethodGet,
			Path:        "/:date",
			Handler:     availableDatesHandler.GetAvailableDateByDate,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("available_dates:read")},
		},
		// Create available date - admin/manager only
		{
			Method:      http.MethodPost,
			Path:        "",
			Handler:     availableDatesHandler.CreateAvailableDate,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("available_dates:create")},
		},
		// Update available date - admin/manager only
		{
			Method:      http.MethodPut,
			Path:        "/:date",
			Handler:     availableDatesHandler.UpdateAvailableDate,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("available_dates:update")},
		},
		// Delete available date - admin/manager only
		{
			Method:      http.MethodDelete,
			Path:        "/:date",
			Handler:     availableDatesHandler.DeleteAvailableDate,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("available_dates:delete")},
		},
	}
	routing.RegisterRoutes(availableDatesGroup, availableDatesRoutes)
}

// Placeholder handler - in a real implementation, you would create a proper REST handler
type availableDatesHandler struct {
	log                  logger.Logger
	availableDatesModule module.AvailableDates
}

func newAvailableDatesHandler(log logger.Logger, availableDatesModule module.AvailableDates) *availableDatesHandler {
	return &availableDatesHandler{
		log:                  log,
		availableDatesModule: availableDatesModule,
	}
}

// GetAvailableDates gets available dates with optional admin features
// @Summary Get available dates
// @Description Get available dates within a date range for booking appointments. Admin/managers can include inactive dates.
// @Tags available-dates
// @Security BearerAuth
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Param include_inactive query bool false "Include inactive dates (default: false, admin/manager only)"
// @Param only_available query bool false "Only return dates with available capacity (default: false)"
// @Success 200 {object} model.Response{data=[]dto.AvailableDate} "Successfully retrieved available dates"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid parameters"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /available-dates [get]
func (h *availableDatesHandler) GetAvailableDates(ctx *gin.Context) {
	h.log.Debug(ctx, "Getting available dates")

	// Parse query parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")
	includeInactiveStr := ctx.DefaultQuery("include_inactive", "false")
	onlyAvailableStr := ctx.DefaultQuery("only_available", "false")

	// Validate required parameters
	if startDateStr == "" {
		err := errors.ErrInvalidUserInput.New("start_date is required")
		h.log.Warn(ctx, "missing start_date parameter")
		_ = ctx.Error(err)
		return
	}

	if endDateStr == "" {
		err := errors.ErrInvalidUserInput.New("end_date is required")
		h.log.Warn(ctx, "missing end_date parameter")
		_ = ctx.Error(err)
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.New("invalid start_date format, expected YYYY-MM-DD")
		h.log.Warn(ctx, "invalid start_date format", zap.String("start_date", startDateStr))
		_ = ctx.Error(err)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.New("invalid end_date format, expected YYYY-MM-DD")
		h.log.Warn(ctx, "invalid end_date format", zap.String("end_date", endDateStr))
		_ = ctx.Error(err)
		return
	}

	includeInactive, err := strconv.ParseBool(includeInactiveStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.New("invalid include_inactive value, expected true or false")
		h.log.Warn(ctx, "invalid include_inactive format", zap.String("include_inactive", includeInactiveStr))
		_ = ctx.Error(err)
		return
	}

	onlyAvailable, err := strconv.ParseBool(onlyAvailableStr)
	if err != nil {
		err = errors.ErrInvalidUserInput.New("invalid only_available value, expected true or false")
		h.log.Warn(ctx, "invalid only_available format", zap.String("only_available", onlyAvailableStr))
		_ = ctx.Error(err)
		return
	}

	// If include_inactive is requested, use the admin method with full query
	if includeInactive {
		// Build query for admin functionality
		query := dto.AvailableDateQuery{
			StartDate:       startDate,
			EndDate:         endDate,
			IncludeInactive: includeInactive,
			OnlyAvailable:   onlyAvailable,
		}

		// Get all available dates (including inactive)
		dates, err := h.availableDatesModule.GetAllAvailableDates(ctx, query)
		if err != nil {
			_ = ctx.Error(err)
			return
		}

		constants.SuccessResponse(ctx, http.StatusOK, dates, nil)
	} else {
		// Get available dates (public method)
		dates, err := h.availableDatesModule.GetAvailableDates(ctx, startDate, endDate, onlyAvailable)
		if err != nil {
			_ = ctx.Error(err)
			return
		}

		constants.SuccessResponse(ctx, http.StatusOK, dates, nil)
	}
}

// GetAvailableDateByDate gets a specific available date
// @Summary Get available date by date
// @Description Get a specific available date by its date
// @Tags available-dates
// @Security BearerAuth
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} model.Response{data=dto.AvailableDate} "Successfully retrieved available date"
// @Failure 400 {object} model.ErrorResponse "Bad request - invalid date format"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} model.ErrorResponse "Available date not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /available-dates/{date} [get]
func (h *availableDatesHandler) GetAvailableDateByDate(ctx *gin.Context) {
	dateStr := ctx.Param("date")
	h.log.Debug(ctx, "Getting available date by date", zap.String("date", dateStr))

	// Get available date
	date, err := h.availableDatesModule.GetAvailableDateByDate(ctx, dateStr)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, date, nil)
}

// CreateAvailableDate creates a new available date (Admin/Manager)
// @Summary Create new available date (Admin/Manager)
// @Description Create a new available date for appointments. Admins and managers can add availability dates beyond the default Monday/Wednesday/Friday schedule.
// @Tags available-dates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateAvailableDate true "Create available date request"
// @Success 201 {object} model.Response{data=dto.AvailableDate} "Successfully created available date"
// @Failure 400 {object} model.ErrorResponse "Bad request - validation errors (invalid date, capacity, etc.)"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 409 {object} model.ErrorResponse "Conflict - available date already exists for this date"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /available-dates [post]
func (h *availableDatesHandler) CreateAvailableDate(ctx *gin.Context) {
	h.log.Debug(ctx, "Creating available date")

	// Parse request body
	var param dto.CreateAvailableDate
	if err := ctx.ShouldBindJSON(&param); err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Create available date
	date, err := h.availableDatesModule.CreateAvailableDate(ctx, param)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusCreated, date, nil)
}

// UpdateAvailableDate updates an existing available date (Admin/Manager)
// @Summary Update available date (Admin/Manager)
// @Description Update an existing available date's capacity or active status
// @Tags available-dates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date path string true "Date (YYYY-MM-DD)"
// @Param request body dto.UpdateAvailableDate true "Update available date request"
// @Success 200 {object} model.Response{data=dto.AvailableDate} "Successfully updated available date"
// @Failure 400 {object} model.ErrorResponse "Bad request - validation errors or cannot reduce capacity below current bookings"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Available date not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /available-dates/{date} [put]
func (h *availableDatesHandler) UpdateAvailableDate(ctx *gin.Context) {
	dateStr := ctx.Param("date")
	h.log.Debug(ctx, "Updating available date", zap.String("date", dateStr))

	// Parse request body
	var param dto.UpdateAvailableDate
	if err := ctx.ShouldBindJSON(&param); err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid request body")
		h.log.Warn(ctx, "invalid request body", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Update available date
	date, err := h.availableDatesModule.UpdateAvailableDate(ctx, dateStr, param)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, date, nil)
}

// DeleteAvailableDate deletes an available date (Admin/Manager)
// @Summary Delete available date (Admin/Manager)
// @Description Delete an available date. Cannot delete dates with active appointments.
// @Tags available-dates
// @Security BearerAuth
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} model.Response "Successfully deleted available date"
// @Failure 400 {object} model.ErrorResponse "Bad request - cannot delete date with active bookings"
// @Failure 401 {object} model.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} model.ErrorResponse "Forbidden - requires admin or manager role"
// @Failure 404 {object} model.ErrorResponse "Available date not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /available-dates/{date} [delete]
func (h *availableDatesHandler) DeleteAvailableDate(ctx *gin.Context) {
	dateStr := ctx.Param("date")
	h.log.Debug(ctx, "Deleting available date", zap.String("date", dateStr))

	// Delete available date
	err := h.availableDatesModule.DeleteAvailableDate(ctx, dateStr)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Available date deleted successfully", nil)
}
