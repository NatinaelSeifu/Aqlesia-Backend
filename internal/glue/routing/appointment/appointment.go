package appointment

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/glue/routing"
	"aqlesia/internal/handler/middleware"
	"aqlesia/internal/handler/rest"
	middleware2 "aqlesia/internal/middleware"
	"aqlesia/internal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppointmentAccessMiddleware creates a middleware that validates appointment access permissions
func AppointmentAccessMiddleware(authzMiddleware *middleware2.AuthorizationMiddleware, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the appointment ID from URL parameter
		appointmentID := c.Param("id")
		if appointmentID == "" {
			// If no ID in path, continue (this might be for list endpoints)
			c.Next()
			return
		}

		// Prepare resource context
		resource := map[string]interface{}{
			"appointment_id": appointmentID,
		}

		// Use the authorization middleware's permission check
		permissionMiddleware := authzMiddleware.RequirePermission("appointments:" + action)
		
		// If the permission check fails, try the "own" permission
		c.Set("appointment_resource", resource)
		permissionMiddleware(c)
	}
}


// InitRoute sets up the appointment routes with proper authentication and authorization
func InitRoute(grp *gin.RouterGroup, appointmentHandler rest.Appointment, jwtManager *auth.JWTManager, authzMiddleware *middleware2.AuthorizationMiddleware, userStorage storage.User) {
	appointments := grp.Group("appointments")
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)

	appointmentRoutes := []routing.Router{
		// Create appointment - users can create their own appointments
		{
			Method:      http.MethodPost,
			Path:        "",
			Handler:     appointmentHandler.CreateAppointment,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:create")},
		},
		
		// Get appointment by ID - users can read their own appointments, admin/manager can read any
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Handler:     appointmentHandler.GetAppointment,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequireUserAccess("read")},
		},
		
		// Update appointment - users can update their own appointments
		{
			Method:      http.MethodPatch,
			Path:        "/:id",
			Handler:     appointmentHandler.UpdateAppointment,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequireUserAccess("update")},
		},
		
		// Delete appointment - admin only
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Handler:     appointmentHandler.DeleteAppointment,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:delete")},
		},
		
		// Get user's own appointments
		{
			Method:      http.MethodGet,
			Path:        "/my",
			Handler:     appointmentHandler.GetUserAppointments,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // No additional authorization needed as users access their own data
		},
		
		// Get all appointments - admin/manager only
		{
			Method:      http.MethodGet,
			Path:        "",
			Handler:     appointmentHandler.GetAllAppointments,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:list")},
		},
		
		// Mark appointment as completed - admin/manager only
		{
			Method:      http.MethodPost,
			Path:        "/:id/complete",
			Handler:     appointmentHandler.MarkAppointmentCompleted,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:complete")},
		},
		
		// Cancel appointment - users can cancel their own
		{
			Method:      http.MethodPost,
			Path:        "/:id/cancel",
			Handler:     appointmentHandler.CancelAppointment,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:cancel_own")},
		},
		
		// Get appointment statistics - admin/manager only
		{
			Method:      http.MethodGet,
			Path:        "/stats",
			Handler:     appointmentHandler.GetAppointmentStats,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:stats")},
		},
		
		// Get available dates - all authenticated users
		{
			Method:      http.MethodGet,
			Path:        "/available-dates",
			Handler:     appointmentHandler.GetAvailableDates,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("appointments:list_available")},
		},
	}
	
	routing.RegisterRoutes(appointments, appointmentRoutes)
}
