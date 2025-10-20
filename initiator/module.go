package initiator

import (
	"aqlesia/internal/module"
	"aqlesia/internal/module/appointment"
	"aqlesia/internal/module/available_dates"
	"aqlesia/internal/module/communion"
	"aqlesia/internal/module/password_reset"
	"aqlesia/internal/module/questions"
	"aqlesia/internal/module/user"
	"aqlesia/internal/telegram"
	"aqlesia/platform/logger"
	"context"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Module struct {
	user          module.User
	appointment   module.Appointment
	availableDates module.AvailableDates
	communion     module.Communion
	questions     module.Questions
	passwordReset module.PasswordReset
	telegramBot   *telegram.BotService
}

func InitModule(persistence Persistence, log logger.Logger) Module {
	// Initialize Telegram bot service
	telegramConfig := telegram.Config{
		BotToken:      viper.GetString("telegram.bot_token"),
		AppBaseURL:    viper.GetString("telegram.app_base_url"),
		TokenPepper:   viper.GetString("telegram.token_pepper"),
		LinkCodeTTL:   viper.GetDuration("telegram.link_code_ttl"),
	}

	telegramBot, err := telegram.NewBotService(telegramConfig, persistence.user, log.Named("telegram-bot"))
	if err != nil {
		log.Panic(context.Background(), "failed to initialize Telegram bot", zap.Error(err))
	}

	// Initialize password reset module
	passwordResetModule := password_reset.Init(
		log.Named("password-reset-module"),
		persistence.user,
		telegramBot,
		telegramConfig.TokenPepper,
		telegramConfig.AppBaseURL,
	)

	return Module{
		user:          user.Init(log.Named("user-module"), persistence.user),
		appointment:   appointment.Init(persistence.appointment, persistence.user, persistence.slot, persistence.availableDates, log.Named("appointment-module")),
		availableDates: available_dates.Init(log.Named("available-dates-module"), persistence.availableDates),
		communion:     communion.Init(persistence.communion, persistence.user, log.Named("communion-module")),
		questions:     questions.Init(persistence.questions, log.Named("questions-module")),
		passwordReset: passwordResetModule,
		telegramBot:   telegramBot,
	}
}
