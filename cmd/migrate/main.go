package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/database"
	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"

	"github.com/joho/godotenv"
)

func main() {
	log := logger.Configure(os.Stdout)
	if err := run(); err != nil {
		log.Error().Err(err).Msg("database migration and initialization failed")
		os.Exit(1)
	}
	log.Info().Msg("database migration and initialization complete")
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("load environment: %w", err)
	}
	poolConfig, err := database.LoadPoolConfigFromEnv()
	if err != nil {
		return fmt.Errorf("load PostgreSQL pool configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectionContext, cancelConnection := context.WithTimeout(ctx, 10*time.Second)
	pool, err := database.NewPostgreSQLPool(connectionContext, os.Getenv("DATABASE_URL"), poolConfig)
	cancelConnection()
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()

	schemaContext, cancelSchema := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelSchema()
	if err := database.EnsureTables(schemaContext, pool); err != nil {
		return fmt.Errorf("initialize application schema and seed data: %w", err)
	}
	return nil
}
