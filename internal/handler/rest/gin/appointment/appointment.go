package appointment

import (
	"aqlesia/internal/constants"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/handler/middleware"
	"aqlesia/internal/handler/rest"
	"aqlesia/internal/module"
	"aqlesia/platform/logger"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type appointment struct {
	logger            logger.Logger
	appointmentModule module.Appointment
	contextTimeout    time.Duration
}

func Init(logger logger.Logger, appointmentModule module.Appointment, contextTimeout time.Duration) rest.Appointment {
	return &appointment{
		appointmentModule: appointmentModule,
		logger:            logger,
		contextTimeout:    contextTimeout,
	}
}

// CreateAppointment creates a new appointment for the authenticated user.
//
//	@Summary		Create appointment
//	@Description	Create a new appointment for the authenticated user. Users can only have one active appointment at a time. Appointments are only available on Monday, Wednesday, and Friday. Maximum 10 appointments per day.
//	@Tags			Appointments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			appointment	body		dto.CreateAppointmentRequest	true	"Appointment details"
//	@Success		201			{object}	dto.Appointment					"Successfully created appointment"
//	@Failure		400			{object}	model.ErrorResponse				"Bad request - validation errors, user already has active appointment, or date at capacity"
//	@Failure		401			{object}	model.ErrorResponse				"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse				"Forbidden - insufficient permissions"
//	@Failure		500			{object}	model.ErrorResponse				"Internal server error"
//	@Router			/appointments [post]
func (a *appointment) CreateAppointment(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		a.logger.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	// Parse request body
	var appointmentRequest dto.CreateAppointmentRequest
	err := ctx.ShouldBind(&appointmentRequest)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "unable to bind appointment data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Create appointment
	userID := authContext.Claims.UserID
	userUUID, err := parseUserID(userID)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		a.logger.Error(ctx, "invalid user ID in token", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	appointment, err := a.appointmentModule.Create(cntx, userUUID, appointmentRequest)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusCreated, appointment, nil)
}

// GetAppointment gets an appointment by ID.
//
//	@Summary		Get appointment by ID
//	@Description	Retrieve appointment details by ID. Users can only access their own appointments unless they are admin/manager.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"Appointment ID (UUID)"
//	@Success		200	{object}	dto.Appointment		"Successfully retrieved appointment"
//	@Failure		400	{object}	model.ErrorResponse	"Bad request - invalid appointment ID"
//	@Failure		401	{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse	"Forbidden - cannot access this appointment"
//	@Failure		404	{object}	model.ErrorResponse	"Appointment not found"
//	@Router			/appointments/{id} [get]
func (a *appointment) GetAppointment(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	appointment, err := a.appointmentModule.Get(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, appointment, nil)
}

// UpdateAppointment updates an appointment by ID.
//
//	@Summary		Update appointment
//	@Description	Update appointment details. Users can only update their own appointments and cannot update past, completed, or cancelled appointments.
//	@Tags			Appointments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string							true	"Appointment ID (UUID)"
//	@Param			appointment	body		dto.UpdateAppointmentRequest	true	"Updated appointment details"
//	@Success		200			{object}	dto.Appointment					"Successfully updated appointment"
//	@Failure		400			{object}	model.ErrorResponse				"Bad request - validation errors or invalid state"
//	@Failure		401			{object}	model.ErrorResponse				"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse				"Forbidden - cannot update this appointment"
//	@Failure		404			{object}	model.ErrorResponse				"Appointment not found"
//	@Router			/appointments/{id} [patch]
func (a *appointment) UpdateAppointment(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		a.logger.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	// Parse request body
	var updateRequest dto.UpdateAppointmentRequest
	err := ctx.ShouldBind(&updateRequest)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "unable to bind appointment update data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Update appointment
	userID := authContext.Claims.UserID
	userUUID, err := parseUserID(userID)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		a.logger.Error(ctx, "invalid user ID in token", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	appointment, err := a.appointmentModule.Update(cntx, ctx.Param("id"), userUUID, updateRequest)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, appointment, nil)
}

// GetUserAppointments gets appointments for the authenticated user.
//
//	@Summary		Get user appointments
//	@Description	Retrieve appointments for the authenticated user with pagination.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int						false	"Page number (default: 1)"
//	@Param			page_size	query		int						false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	dto.AppointmentListResponse	"Successfully retrieved user appointments"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/appointments/my [get]
func (a *appointment) GetUserAppointments(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		a.logger.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	// Get user appointments
	userID := authContext.Claims.UserID
	userUUID, err := parseUserID(userID)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		a.logger.Error(ctx, "invalid user ID in token", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	appointments, err := a.appointmentModule.GetUserAppointments(cntx, userUUID, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, appointments, nil)
}

// GetAllAppointments gets all appointments (admin/manager only).
//
//	@Summary		Get all appointments
//	@Description	Retrieve all appointments with pagination. Requires admin or manager role.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int						false	"Page number (default: 1)"
//	@Param			page_size	query		int						false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	dto.AppointmentListResponse	"Successfully retrieved all appointments"
//	@Failure		401			{object}	model.ErrorResponse			"Unauthorized - invalid or missing token"
//	@Failure		403			{object}	model.ErrorResponse			"Forbidden - requires admin or manager role"
//	@Failure		500			{object}	model.ErrorResponse			"Internal server error"
//	@Router			/appointments [get]
func (a *appointment) GetAllAppointments(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	// Get all appointments
	appointments, err := a.appointmentModule.GetAllAppointments(cntx, page, pageSize)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, appointments, nil)
}

// MarkAppointmentCompleted marks an appointment as completed (admin/manager only).
//
//	@Summary		Mark appointment as completed
//	@Description	Mark an appointment as completed with optional notes. Only admin and manager can perform this action.
//	@Tags			Appointments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string								true	"Appointment ID (UUID)"
//	@Param			completion	body	dto.MarkAppointmentCompletedRequest	true	"Completion details"
//	@Success		200		{object}	dto.Appointment						"Successfully marked appointment as completed"
//	@Failure		400		{object}	model.ErrorResponse					"Bad request - invalid ID or appointment state"
//	@Failure		401		{object}	model.ErrorResponse					"Unauthorized - invalid or missing token"
//	@Failure		403		{object}	model.ErrorResponse					"Forbidden - requires admin or manager role"
//	@Failure		404		{object}	model.ErrorResponse					"Appointment not found"
//	@Router			/appointments/{id}/complete [post]
func (a *appointment) MarkAppointmentCompleted(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Parse request body
	var completionRequest dto.MarkAppointmentCompletedRequest
	err := ctx.ShouldBind(&completionRequest)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid input")
		a.logger.Error(ctx, "unable to bind completion request data", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Mark appointment as completed
	appointment, err := a.appointmentModule.MarkCompleted(cntx, ctx.Param("id"), completionRequest)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, appointment, nil)
}

// CancelAppointment cancels an appointment.
//
//	@Summary		Cancel appointment
//	@Description	Cancel an appointment. Users can only cancel their own appointments unless they are admin/manager.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"Appointment ID (UUID)"
//	@Success		200	{object}	model.SuccessResponse	"Successfully cancelled appointment"
//	@Failure		400	{object}	model.ErrorResponse		"Bad request - invalid ID or appointment state"
//	@Failure		401	{object}	model.ErrorResponse		"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse		"Forbidden - cannot cancel this appointment"
//	@Failure		404	{object}	model.ErrorResponse		"Appointment not found"
//	@Router			/appointments/{id}/cancel [post]
func (a *appointment) CancelAppointment(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Get authenticated user
	authContext, exists := middleware.GetAuthContext(ctx)
	if !exists {
		a.logger.Error(ctx, "no auth context found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"ok": false,
			"error": gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			},
		})
		return
	}

	// Cancel appointment
	userID := authContext.Claims.UserID
	userUUID, err := parseUserID(userID)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid user ID")
		a.logger.Error(ctx, "invalid user ID in token", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	err = a.appointmentModule.Cancel(cntx, ctx.Param("id"), userUUID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Appointment cancelled successfully", nil)
}

// DeleteAppointment deletes an appointment (admin only).
//
//	@Summary		Delete appointment
//	@Description	Delete an appointment permanently. Only admin can perform this action.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"Appointment ID (UUID)"
//	@Success		200	{object}	model.SuccessResponse	"Successfully deleted appointment"
//	@Failure		400	{object}	model.ErrorResponse		"Bad request - invalid appointment ID"
//	@Failure		401	{object}	model.ErrorResponse		"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse		"Forbidden - requires admin role"
//	@Failure		404	{object}	model.ErrorResponse		"Appointment not found"
//	@Router			/appointments/{id} [delete]
func (a *appointment) DeleteAppointment(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	err := a.appointmentModule.Delete(cntx, ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, "Appointment deleted successfully", nil)
}

// GetAppointmentStats gets appointment statistics (admin/manager only).
//
//	@Summary		Get appointment statistics
//	@Description	Retrieve appointment statistics including total, pending, completed, and cancelled counts. Requires admin or manager role.
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	dto.AppointmentStatsResponse	"Successfully retrieved appointment statistics"
//	@Failure		401	{object}	model.ErrorResponse				"Unauthorized - invalid or missing token"
//	@Failure		403	{object}	model.ErrorResponse				"Forbidden - requires admin or manager role"
//	@Failure		500	{object}	model.ErrorResponse				"Internal server error"
//	@Router			/appointments/stats [get]
func (a *appointment) GetAppointmentStats(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	stats, err := a.appointmentModule.GetAppointmentStats(cntx)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	constants.SuccessResponse(ctx, http.StatusOK, stats, nil)
}

// GetAvailableDates gets available appointment dates.
//
//	@Summary		Get available appointment dates
//	@Description	Retrieve available appointment dates within a specified date range. Only shows dates that are not at maximum capacity (10 appointments).
//	@Tags			Appointments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			start_date	query		string	false	"Start date (YYYY-MM-DD, default: today)"
//	@Param			end_date	query		string	false	"End date (YYYY-MM-DD, default: 1 month from today)"
//	@Success		200			{array}		string	"Successfully retrieved available dates"
//	@Failure		400			{object}	model.ErrorResponse	"Bad request - invalid date format"
//	@Failure		401			{object}	model.ErrorResponse	"Unauthorized - invalid or missing token"
//	@Failure		500			{object}	model.ErrorResponse	"Internal server error"
//	@Router			/appointments/available-dates [get]
func (a *appointment) GetAvailableDates(ctx *gin.Context) {
	cntx, cancel := context.WithTimeout(ctx, a.contextTimeout)
	defer cancel()

	// Parse date parameters
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Default start date is today
	startDateStr := ctx.DefaultQuery("start_date", today.Format("2006-01-02"))
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid start_date format, use YYYY-MM-DD")
		a.logger.Error(ctx, "invalid start_date format", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Default end date is 1 month from today
	endDateStr := ctx.DefaultQuery("end_date", today.AddDate(0, 1, 0).Format("2006-01-02"))
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		err := errors.ErrInvalidUserInput.Wrap(err, "invalid end_date format, use YYYY-MM-DD")
		a.logger.Error(ctx, "invalid end_date format", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		err := errors.ErrInvalidUserInput.New("end_date cannot be before start_date")
		a.logger.Error(ctx, "invalid date range", zap.Time("start", startDate), zap.Time("end", endDate))
		_ = ctx.Error(err)
		return
	}

	// Get available dates
	availableDates, err := a.appointmentModule.GetAvailableDates(cntx, startDate, endDate)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	// Convert to string format
	dateStrings := make([]string, len(availableDates))
	for i, date := range availableDates {
		dateStrings[i] = date.Format("2006-01-02")
	}

	constants.SuccessResponse(ctx, http.StatusOK, dateStrings, nil)
}

// Helper function to parse user ID from string (handle both UUID string and numeric formats)
func parseUserID(userIDStr string) (uuid.UUID, error) {
	// Try parsing as UUID first
	if userUUID, err := uuid.Parse(userIDStr); err == nil {
		return userUUID, nil
	}

	// If UUID parsing fails, return error
	return uuid.UUID{}, fmt.Errorf("invalid UUID format: %s", userIDStr)
}
