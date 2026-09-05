package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
)

func (h Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := response.Success(w, http.StatusOK, map[string]string{
		"service": "healthy",
		"env":     h.status,
	}); err != nil {
		h.log.Error().Err(err).Msg("write health response")
	}
}

func (h Handler) ProtectedExample(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		h.fail(w, http.StatusUnauthorized, "missing authentication claims")
		return
	}

	if err := response.Success(w, http.StatusOK, map[string]any{
		"message": "authenticated",
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role_id": claims.RoleID,
		"rules":   claims.Rules,
	}); err != nil {
		h.log.Error().Err(err).Msg("write protected example response")
	}
}
