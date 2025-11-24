package rest

import "github.com/gin-gonic/gin"

type User interface {
	UpdateUser(ctx *gin.Context)
	GetUser(ctx *gin.Context)
	GetUsers(ctx *gin.Context)
	UpdateUserStatus(ctx *gin.Context)
	DeleteUser(ctx *gin.Context)
	UploadAvatar(ctx *gin.Context)
}

type Auth interface {
	Login(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Register(ctx *gin.Context)
	ChangePassword(ctx *gin.Context)
}

type Appointment interface {
	CreateAppointment(ctx *gin.Context)
	GetAppointment(ctx *gin.Context)
	UpdateAppointment(ctx *gin.Context)
	GetUserAppointments(ctx *gin.Context)
	GetAllAppointments(ctx *gin.Context)
	MarkAppointmentCompleted(ctx *gin.Context)
	CancelAppointment(ctx *gin.Context)
	DeleteAppointment(ctx *gin.Context)
	GetAppointmentStats(ctx *gin.Context)
	GetAvailableDates(ctx *gin.Context)
}

type Communion interface {
	CreateCommunion(ctx *gin.Context)
	GetCommunion(ctx *gin.Context)
	GetUserCommunions(ctx *gin.Context)
	GetAllCommunions(ctx *gin.Context)
	GetPendingCommunions(ctx *gin.Context)
	UpdateCommunionStatus(ctx *gin.Context)
	UpdateCommunion(ctx *gin.Context)
	DeleteCommunion(ctx *gin.Context)
}
