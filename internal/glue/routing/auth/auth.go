package auth

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/handler/middleware"
	"aqlesia/internal/handler/rest"
	"aqlesia/internal/storage"

	"github.com/gin-gonic/gin"
)

func InitRoute(group *gin.RouterGroup, authHandler rest.Auth, jwtManager *auth.JWTManager, userStorage storage.User) {
	authGroup := group.Group("/auth")
	authMiddleware := middleware.AuthMiddleware(jwtManager, userStorage)
	{
		authGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/change-password", authMiddleware, authHandler.ChangePassword)
	}
}
