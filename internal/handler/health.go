package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
	"github.com/rs/zerolog"
)

type Handler struct {
	status      string
	log         *zerolog.Logger
	authService *service.AuthService
}

func New(status string, log *zerolog.Logger, authService *service.AuthService) Handler {
	return Handler{
		status:      status,
		log:         log,
		authService: authService,
	}
}

func (h Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := response.Success(w, http.StatusOK, map[string]string{
		"service": "healthy",
		"env":     h.status,
	}); err != nil {
		h.log.Error().Err(err).Msg("write health response")
	}
}
