package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/database"
	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var Log *zerolog.Logger

func main() {
	Log = logger.Configure(os.Stdout)
	if err := godotenv.Load(); err != nil {
		Log.Fatal().Err(err).Msg("load environment")
	}

	app := Application{config: Config{
		addr:        net.JoinHostPort(os.Getenv("APP_HOST"), os.Getenv("APP_PORT")),
		databaseURL: os.Getenv("DATABASE_URL"),
		logFile:     os.Getenv("APP_LOG_FILE"),
		status:      os.Getenv("APP_ENV"),
	}}
	if app.config.addr == ":" {
		Log.Fatal().Msg("APP_HOST and APP_PORT must be set")
	}
	if app.config.logFile == "" {
		Log.Fatal().Msg("APP_LOG_FILE must be set")
	}
	jwtConfig, err := loadJWTConfig()
	if err != nil {
		Log.Fatal().Err(err).Msg("load JWT configuration")
	}
	app.config.jwt = jwtConfig

	logFile, err := logger.ConfigureFile(os.Stdout, app.config.logFile)
	if err != nil {
		Log.Fatal().Err(err).Str("path", app.config.logFile).Msg("configure file logger")
	}
	Log = logger.Get()
	defer func() {
		if err := logFile.Close(); err != nil {
			Log.Error().Err(err).Str("path", app.config.logFile).Msg("close log file")
		}
	}()

	databaseContext, cancelDatabaseConnection := context.WithTimeout(context.Background(), 10*time.Second)
	databasePool, err := database.NewPostgreSQLPool(databaseContext, app.config.databaseURL)
	cancelDatabaseConnection()
	if err != nil {
		Log.Fatal().Err(err).Msg("connect PostgreSQL")
	}
	app.database = databasePool
	defer app.database.Close()

	schemaContext, cancelSchema := context.WithTimeout(context.Background(), 10*time.Second)
	err = database.EnsureAuthTables(schemaContext, app.database)
	cancelSchema()
	if err != nil {
		Log.Fatal().Err(err).Msg("ensure auth tables")
	}

	server := &http.Server{
		Addr:    app.config.addr,
		Handler: app.routes(),
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	Log.Info().Msg("PostgreSQL pool connected")
	Log.Info().Str("addr", app.config.addr).Msg("API server starting")
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			Log.Fatal().Err(err).Msg("API server stopped unexpectedly")
		}
	case shutdownSignal := <-shutdownSignals:
		Log.Info().Str("signal", shutdownSignal.String()).Msg("API server shutting down")

		shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			Log.Error().Err(err).Msg("API server graceful shutdown failed")
			if closeErr := server.Close(); closeErr != nil {
				Log.Error().Err(closeErr).Msg("API server forced shutdown failed")
			}
		}

		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			Log.Error().Err(err).Msg("API server stopped with an error")
		}
		Log.Info().Msg("API server stopped")
	}
}
