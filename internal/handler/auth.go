package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt string    `json:"expires_at"`
	User      userBrief `json:"user"`
}

type userBrief struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	RoleID int      `json:"role_id"`
	Status bool     `json:"status"`
	Rules  []string `json:"rules"`
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		h.fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		h.fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	request.Email = strings.TrimSpace(request.Email)
	if request.Email == "" {
		h.fail(w, http.StatusBadRequest, "email is required")
		return
	}
	parsedEmail, err := mail.ParseAddress(request.Email)
	if err != nil || parsedEmail.Address != request.Email {
		h.fail(w, http.StatusBadRequest, "invalid email")
		return
	}
	if request.Password == "" {
		h.fail(w, http.StatusBadRequest, "password is required")
		return
	}

	result, err := h.authService.Login(r.Context(), request.Email, request.Password, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			h.fail(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, service.ErrUserBanned):
			h.fail(w, http.StatusForbidden, "user is banned")
		default:
			h.log.Error().Err(err).Msg("login")
			h.fail(w, http.StatusInternalServerError, "failed to login")
		}
		return
	}

	if err := response.Success(w, http.StatusOK, loginResponse{
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339Nano),
		User: userBrief{UserID: result.User.ID, Email: result.User.Email, RoleID: result.User.RoleID,
			Status: result.User.Status, Rules: result.User.Rules},
	}); err != nil {
		h.log.Error().Err(err).Msg("write login response")
	}
}
