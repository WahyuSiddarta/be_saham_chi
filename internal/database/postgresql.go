package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type PoolConfig struct {
	MaxOpenConnections int
	MaxIdleConnections int
	ConnMaxIdleTime    time.Duration
	ConnMaxLifetime    time.Duration
}

// LoadPoolConfigFromEnv loads explicit database/sql pool settings.
func LoadPoolConfigFromEnv() (PoolConfig, error) {
	maxOpenConnections, err := positiveIntEnv("DATABASE_MAX_OPEN_CONNS")
	if err != nil {
		return PoolConfig{}, err
	}
	maxIdleConnections, err := nonNegativeIntEnv("DATABASE_MAX_IDLE_CONNS")
	if err != nil {
		return PoolConfig{}, err
	}
	if maxIdleConnections > maxOpenConnections {
		return PoolConfig{}, fmt.Errorf("DATABASE_MAX_IDLE_CONNS cannot exceed DATABASE_MAX_OPEN_CONNS")
	}
	connMaxIdleTime, err := positiveDurationEnv("DATABASE_CONN_MAX_IDLE_TIME")
	if err != nil {
		return PoolConfig{}, err
	}
	connMaxLifetime, err := positiveDurationEnv("DATABASE_CONN_MAX_LIFETIME")
	if err != nil {
		return PoolConfig{}, err
	}

	return PoolConfig{
		MaxOpenConnections: maxOpenConnections,
		MaxIdleConnections: maxIdleConnections,
		ConnMaxIdleTime:    connMaxIdleTime,
		ConnMaxLifetime:    connMaxLifetime,
	}, nil
}

// NewPostgreSQLPool creates and verifies a PostgreSQL connection pool.
func NewPostgreSQLPool(ctx context.Context, databaseURL string, config PoolConfig) (*sqlx.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	pool, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	pool.SetMaxOpenConns(config.MaxOpenConnections)
	pool.SetMaxIdleConns(config.MaxIdleConnections)
	pool.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	pool.SetConnMaxLifetime(config.ConnMaxLifetime)

	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL pool: %w", err)
	}

	return pool, nil
}

func positiveIntEnv(name string) (int, error) {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func nonNegativeIntEnv(name string) (int, error) {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return value, nil
}

func positiveDurationEnv(name string) (time.Duration, error) {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}
