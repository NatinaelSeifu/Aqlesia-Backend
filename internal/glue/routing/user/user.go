package user

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

func InitRoute(grp *gin.RouterGroup, user rest.User, jwtManager *auth.JWTManager, authzMiddleware *middleware2.AuthorizationMiddleware, userStorage storage.User) {
	users := grp.Group("users")
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)
	
	usersRoutes := []routing.Router{
		{
			Method:      http.MethodPatch,
			Path:        "/:id",
			Handler:     user.UpdateUser,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequireUserAccess("update")},
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Handler:     user.GetUser,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequireUserAccess("read")},
		},
		{
			Method:      http.MethodGet,
			Handler:     user.GetUsers,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("users:list")},
		},
		{
			Method:      http.MethodPatch,
			Path:        "/:id/status",
			Handler:     user.UpdateUserStatus,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequirePermission("users:approve")},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Handler:     user.DeleteUser,
			Middlewares: []gin.HandlerFunc{authMiddleware, authzMiddleware.RequireUserAccess("delete")},
		},
	}
	routing.RegisterRoutes(users, usersRoutes)
}
