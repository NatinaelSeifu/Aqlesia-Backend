package initiator

import (
	"aqlesia/platform/logger"
	"log"

	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	logg, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	return logger.New(logg)
}
