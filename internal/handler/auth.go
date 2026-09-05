package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h Handler) Register(w http.ResponseWriter, req *http.Request) error {
	var request RegisterRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	if helper.CheckEmptyString(request.Email) {
		return response.Fail(w, http.StatusBadRequest, "email is required")
	}
	if !helper.CheckEmail(request.Email) {
		return response.Fail(w, http.StatusBadRequest, "invalid email")
	}
	if helper.CheckEmptyString(request.Password) {
		return response.Fail(w, http.StatusBadRequest, "password is required")
	}

	user, portfolio, err := h.authService.Register(req.Context(), request.Email, request.Password)
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to register user", fmt.Errorf("AuthHandler.Register -> AuthService.Register: %w", err))
		return response.Fail(w, http.StatusInternalServerError, "failed to register user")
	}

	return response.Success(w, http.StatusCreated, NewRegisterResponse(user, portfolio))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h Handler) Login(w http.ResponseWriter, req *http.Request) error {
	var request LoginRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	if helper.CheckEmptyString(request.Email) {
		return response.Fail(w, http.StatusBadRequest, "email is required")
	}
	if !helper.CheckEmail(request.Email) {
		return response.Fail(w, http.StatusBadRequest, "invalid email")
	}
	if helper.CheckEmptyString(request.Password) {
		return response.Fail(w, http.StatusBadRequest, "password is required")
	}

	result, err := h.authService.Login(req.Context(), request.Email, request.Password, time.Now())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			h.logRequestError(req, http.StatusUnauthorized, "invalid email or password", fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
			return response.Fail(w, http.StatusUnauthorized, "invalid email or password")
		}
		if errors.Is(err, service.ErrUserBanned) {
			h.logRequestError(req, http.StatusForbidden, "user is banned", fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
			return response.Fail(w, http.StatusForbidden, "user is banned")
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to login", fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
		return response.Fail(w, http.StatusInternalServerError, "failed to login")
	}

	return response.Success(w, http.StatusOK, LoginResponse{
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339Nano),
		User:      NewUserBrief(result.User),
	})
}

type RegisterResponse struct {
	User      UserBrief      `json:"user"`
	Portfolio PortfolioBrief `json:"portfolio"`
}

type UserBrief struct {
	UserId    string   `json:"user_id"`
	Email     string   `json:"email"`
	RoleID    int      `json:"role_id"`
	Status    bool     `json:"status"`
	Rules     []string `json:"rules"`
	CreatedAt string   `json:"created_at"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt string    `json:"expires_at"`
	User      UserBrief `json:"user"`
}

type PortfolioBrief struct {
	PortfolioID      string `json:"portfolio_id"`
	Name             string `json:"name"`
	BaseCurrencyCode string `json:"base_currency_code"`
	CreatedAt        string `json:"created_at"`
}
