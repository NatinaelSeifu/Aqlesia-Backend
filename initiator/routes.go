package initiator

import (
	"aqlesia/docs"
	"aqlesia/internal/auth"
	appointmentRouting "aqlesia/internal/glue/routing/appointment"
	availableDatesRouting "aqlesia/internal/glue/routing/available_dates"
	authRouting "aqlesia/internal/glue/routing/auth"
	communionRouting "aqlesia/internal/glue/routing/communion"
	passwordResetRouting "aqlesia/internal/glue/routing/password_reset"
	questionsRouting "aqlesia/internal/glue/routing/questions"
	"aqlesia/internal/glue/routing/user"
	"aqlesia/internal/middleware"
	"aqlesia/platform/logger"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(group *gin.RouterGroup, handler Handler, module Module, jwtManager *auth.JWTManager, authzMiddlewareInstance *middleware.AuthorizationMiddleware, persistence Persistence, log logger.Logger) {
	docs.SwaggerInfo.BasePath = "/v1"
	group.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authRouting.InitRoute(group, handler.auth, jwtManager, persistence.user)
	user.InitRoute(group, handler.user, jwtManager, authzMiddlewareInstance, persistence.user)
	appointmentRouting.InitRoute(group, handler.appointment, jwtManager, authzMiddlewareInstance, persistence.user)
	availableDatesRouting.Init(group, log.Named("available-dates-routes"), module.availableDates, jwtManager, authzMiddlewareInstance, persistence.user)
	communionRouting.InitRoute(group, handler.communion, jwtManager, authzMiddlewareInstance, persistence.user)
	questionsRouting.Init(group, log.Named("questions-routes"), module.questions, jwtManager, authzMiddlewareInstance, persistence.user)
	passwordResetRouting.InitRoutes(group, module.passwordReset, log.Named("password-reset-routes"))
}
