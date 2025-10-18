package initiator

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/storage"
	"aqlesia/internal/storage/sqlc/appointment"
	"aqlesia/internal/storage/sqlc/available_dates"
	"aqlesia/internal/storage/sqlc/available_slot"
	"aqlesia/internal/storage/sqlc/communion"
	"aqlesia/internal/storage/sqlc/questions"
	"aqlesia/internal/storage/sqlc/user"
	"aqlesia/platform/logger"
)

type Persistence struct {
	user            storage.User
	appointment     storage.Appointment
	slot            storage.AvailableSlot
	availableDates  storage.AvailableDates
	communion       storage.Communion
	questions       storage.Questions
}

func InitPersistence(db dbinstance.DBInstance, log logger.Logger) Persistence {
	return Persistence{
		user:           user.Init(db, log.Named("user-persistence")),
		appointment:    appointment.Init(db, log.Named("appointment-persistence")),
		slot:           available_slot.Init(db, log.Named("slot-persistence")),
		availableDates: available_dates.Init(db, log.Named("available-dates-persistence")),
		communion:      communion.Init(db, log.Named("communion-persistence")),
		questions:      questions.Init(db, log.Named("questions-persistence")),
	}
}
