package initiator

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/handler/middleware"
	authzMiddleware "aqlesia/internal/middleware"
	"aqlesia/internal/opa"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"syscall"

	// ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Initiate
//
//	@title			Kesis Ethiopian Phone Authentication API
//	@version		1.0
//	@description	A comprehensive authentication API supporting Ethiopian phone numbers with JWT-based security
//	@termsOfService	http://swagger.io/terms/
//	@contact.name	API Support
//	@contact.url	http://www.example.com/support
//	@contact.email	support@example.com
//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT
//	@host			localhost:8000
//	@BasePath		/v1
//	@schemes		http https
//	@securityDefinitions.apikey	BearerAuth
//	@in				header
//	@name			Authorization
//	@description	Enter the token with the `Bearer: ` prefix, e.g. "Bearer abcde12345"
func Initiator(ctx context.Context) {

	log := InitLogger()
	log.Info(ctx, "logger initialized")

	log.Info(ctx, "initializing config")
	InitConfig("config", "config", log)
	log.Info(ctx, "config initialized")

	log.Info(ctx, "initializing database")
	Conn := InitDB(viper.GetString("database.url"), log)
	log.Info(ctx, "database initialized")

	if viper.GetBool("migration.active") {
		log.Info(ctx, "initializing migration")
		m := InitiateMigration(viper.GetString("migration.path"), viper.GetString("database.url"), log)
		UpMigration(m, log)
		log.Info(ctx, "migration initialized")
	} else {
		log.Info(ctx, "migration skipped (disabled in config)")
	}

	log.Info(ctx, "initializing persistence layer")
	persistence := InitPersistence(dbinstance.New(Conn), log)
	log.Info(ctx, "persistence layer initialized")

	log.Info(ctx, "initializing module")
	module := InitModule(persistence, log)
	log.Info(ctx, "module initialized")

	log.Info(ctx, "initializing JWT manager")
	jwtManager := auth.NewJWTManager(viper.GetString("jwt.secret"))
	log.Info(ctx, "JWT manager initialized")

	log.Info(ctx, "initializing OPA service")
	opaService, err := opa.NewOPAService(log.Named("opa"))
	if err != nil {
		log.Fatal(ctx, "failed to initialize OPA service", zap.Error(err))
	}
	log.Info(ctx, "OPA service initialized")

	log.Info(ctx, "initializing authorization middleware")
	authzMiddlewareInstance := authzMiddleware.NewAuthorizationMiddleware(opaService, log.Named("authz"))
	log.Info(ctx, "authorization middleware initialized")

	log.Info(ctx, "initializing cron scheduler")
	cronScheduler := InitCronScheduler(dbinstance.New(Conn), log)
	log.Info(ctx, "cron scheduler initialized")

	log.Info(ctx, "seeding initial appointment slots")
	if err := cronScheduler.SeedInitialSlots(ctx); err != nil {
		log.Error(ctx, "failed to seed initial appointment slots", zap.Error(err))
		// Don't fail startup - just log the error
	} else {
		log.Info(ctx, "initial appointment slots seeded successfully")
	}

	log.Info(ctx, "starting cron scheduler")
	cronScheduler.Start(ctx)

	// Start Telegram bot service
	log.Info(ctx, "starting Telegram bot service")
	go func() {
		module.telegramBot.Start()
	}()
	log.Info(ctx, "Telegram bot service started")

	log.Info(ctx, "initializing handler")
	handler := InitHandler(module, persistence, log, viper.GetDuration("server.timeout"), jwtManager)
	log.Info(ctx, "handler initialized")

	log.Info(ctx, "initializing server")
	server := gin.New()
	gin.SetMode(gin.ReleaseMode)

	server.Use(middleware.GinLogger(log.Named("gin")))
	server.Use(middleware.RecoveryWithZap(log.Named("gin.recovery"), true))
	server.Use(middleware.ErrorHandler())
	server.Use(InitCORS())
	log.Info(ctx, "server initialized")

	log.Info(ctx, "initializing router")
	v1 := server.Group("/v1")
	InitRouter(v1, handler, module, jwtManager, authzMiddlewareInstance, persistence, log)
	log.Info(ctx, "router initialized")

	srv := &http.Server{
		Addr:    viper.GetString("server.host") + ":" + viper.GetString("server.port"),
		Handler: server,
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	go func() {
		log.Info(ctx, "server started",
			zap.String("host", viper.GetString("server.host")),
			zap.Int("port", viper.GetInt("server.port")))
		log.Info(ctx, fmt.Sprintf("server stopped with error %v", srv.ListenAndServe()))
	}()
	sig := <-quit
	log.Info(ctx, fmt.Sprintf("server shutting down with signal %v", sig))
	ctx, cancel := context.WithTimeout(ctx, viper.GetDuration("server.timeout"))
	defer cancel()

	log.Info(ctx, "shutting down cron scheduler")
	cronScheduler.Stop(ctx)

	log.Info(ctx, "shutting down Telegram bot service")
	module.telegramBot.Stop()

	log.Info(ctx, "shutting down server")
	err = srv.Shutdown(ctx)
	if err != nil {
		log.Fatal(ctx, fmt.Sprintf("error while shutting down server: %v", err))
	} else {
		log.Info(ctx, "server shutdown complete")
	}
}
