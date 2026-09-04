package main

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/handler"
	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"
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
}

func (app Application) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(logger.ChiLogFormatter()))
	r.Use(middleware.Recoverer)

	handlers := handler.New(app.config.status, Log)
	r.Get("/health", handlers.Health)

	return r
}
