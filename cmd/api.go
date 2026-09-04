package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/handler"
	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Application struct {
	config   Config
	database *pgxpool.Pool
}

type Config struct {
	addr        string
	databaseURL string
	logFile     string
	status      string
	jwt         auth.Config
}

func (app Application) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(logger.ChiLogFormatter()))
	r.Use(middleware.Recoverer)

	repositories := repository.New(app.database)
	authService := service.NewAuthService(repositories, app.config.jwt)
	handlers := handler.New(app.config.status, Log, authService)
	r.Get("/health", handlers.Health)
	r.Post("/auth/login", handlers.Login)
	r.With(auth.Middleware(app.config.jwt)).Get("/protected", handlers.ProtectedExample)

	return r
}

func loadJWTConfig() (auth.Config, error) {
	ttl, err := time.ParseDuration(os.Getenv("JWT_TTL"))
	if err != nil || ttl <= 0 {
		return auth.Config{}, fmt.Errorf("JWT_TTL must be a positive duration")
	}

	config := auth.Config{
		Secret: os.Getenv("JWT_SECRET"),
		Issuer: os.Getenv("JWT_ISSUER"),
		TTL:    ttl,
	}
	if config.Secret == "" || config.Issuer == "" {
		return auth.Config{}, fmt.Errorf("JWT_SECRET and JWT_ISSUER must be set")
	}

	return config, nil
}
