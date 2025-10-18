package communion

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

// InitRoute sets up the communion routes with proper authentication and authorization
func InitRoute(grp *gin.RouterGroup, communionHandler rest.Communion, jwtManager *auth.JWTManager, authzMiddleware *middleware2.AuthorizationMiddleware, userStorage storage.User) {
	communions := grp.Group("communion")
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)

	communionRoutes := []routing.Router{
		// Create communion request - users can create communion requests
		{
			Method:      http.MethodPost,
			Path:        "",
			Handler:     communionHandler.CreateCommunion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Only authentication required
		},

		// Get communion request by ID - users can view their own, admins can view all
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Handler:     communionHandler.GetCommunion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Authorization handled in handler
		},

		// Update communion request - users can update their own pending requests
		{
			Method:      http.MethodPatch,
			Path:        "/:id",
			Handler:     communionHandler.UpdateCommunion,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Authorization handled in handler
		},

		// Get user's own communion requests
		{
			Method:      http.MethodGet,
			Path:        "/user",
			Handler:     communionHandler.GetUserCommunions,
			Middlewares: []gin.HandlerFunc{authMiddleware}, // Users access their own data
		},

		// Get all communion requests - admin only
		{
			Method:      http.MethodGet,
			Path:        "/all",
			Handler:     communionHandler.GetAllCommunions,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("communions:list")},
		},

		// Get pending communion requests - admin only
		{
			Method:      http.MethodGet,
			Path:        "/pending",
			Handler:     communionHandler.GetPendingCommunions,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("communions:manage")},
		},

		// Update communion status (approve/reject) - admin only
		{
			Method:      http.MethodPatch,
			Path:        "/:id/status",
			Handler:     communionHandler.UpdateCommunionStatus,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("communions:manage")},
		},

		// Delete communion request - admin only
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Handler:     communionHandler.DeleteCommunion,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("communions:delete")},
		},
	}

	routing.RegisterRoutes(communions, communionRoutes)
}
