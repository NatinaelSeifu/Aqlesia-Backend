package initiator

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/handler/rest"
	appointmentHandler "aqlesia/internal/handler/rest/gin/appointment"
	authHandler "aqlesia/internal/handler/rest/gin/auth"
	communionHandler "aqlesia/internal/handler/rest/gin/communion"
	"aqlesia/internal/handler/rest/gin/user"
	"aqlesia/platform/logger"
	"time"
)

type Handler struct {
	user        rest.User
	auth        rest.Auth
	appointment rest.Appointment
	communion   rest.Communion
}

func InitHandler(module Module, persistence Persistence, log logger.Logger, timeout time.Duration, jwtManager *auth.JWTManager) Handler {
	return Handler{
		user:        user.Init(log.Named("user-handler"), module.user, timeout),
		auth:        authHandler.Init(log.Named("auth-handler"), persistence.user, jwtManager, timeout),
		appointment: appointmentHandler.Init(log.Named("appointment-handler"), module.appointment, timeout),
		communion:   communionHandler.Init(log.Named("communion-handler"), module.communion, timeout),
	}
}
