package initiator

import (
	"aqlesia/platform/logger"
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

func InitiateMigration(path, conn string, log logger.Logger) *migrate.Migrate {
	// Use the connection string as-is since it's already postgres://
	m, err := migrate.New(fmt.Sprintf("file://%s", path), conn)
	if err != nil {
		log.Fatal(context.Background(), "could not create migrator", zap.Error(err))
	}
	return m
}

func UpMigration(m *migrate.Migrate, log logger.Logger) {
	err := m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(context.Background(), "could not migrate", zap.Error(err))
	}
}
