package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
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
	var input loginRequest
	if err := binding.BindJSON(r.Body, &input); err != nil {
		h.fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" {
		h.fail(w, http.StatusBadRequest, "email is required")
		return
	}
	parsedEmail, err := mail.ParseAddress(input.Email)
	if err != nil || parsedEmail.Address != input.Email {
		h.fail(w, http.StatusBadRequest, "invalid email")
		return
	}
	if input.Password == "" {
		h.fail(w, http.StatusBadRequest, "password is required")
		return
	}

	result, err := h.authService.Login(r.Context(), input.Email, input.Password, time.Now())
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

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h Handler) Register(w http.ResponseWriter, req *http.Request) error {
	var request RegisterRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if helper.CheckEmptyString(request.Email) {
		return newHTTPError(http.StatusBadRequest, "email is required")
	}
	if !helper.CheckEmail(request.Email) {
		return newHTTPError(http.StatusBadRequest, "invalid email")
	}
	if helper.CheckEmptyString(request.Password) {
		return newHTTPError(http.StatusBadRequest, "password is required")
	}

	user, portfolio, err := h.authService.Register(req.Context(), request.Email, request.Password)
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to register user").SetInternal(fmt.Errorf("AuthHandler.Register -> AuthService.Register: %w", err))
	}

	return response.JSON(w, http.StatusCreated, NewRegisterResponse(user, portfolio))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h Handler) LoginV2(w http.ResponseWriter, req *http.Request) error {
	var request LoginRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if helper.CheckEmptyString(request.Email) {
		return newHTTPError(http.StatusBadRequest, "email is required")
	}
	if !helper.CheckEmail(request.Email) {
		return newHTTPError(http.StatusBadRequest, "invalid email")
	}
	if helper.CheckEmptyString(request.Password) {
		return newHTTPError(http.StatusBadRequest, "password is required")
	}

	result, err := h.authService.Login(req.Context(), request.Email, request.Password, time.Now())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return newHTTPError(http.StatusUnauthorized, "invalid email or password").SetInternal(fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
		}
		if errors.Is(err, service.ErrUserBanned) {
			return newHTTPError(http.StatusForbidden, "user is banned").SetInternal(fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
		}
		return newHTTPError(http.StatusInternalServerError, "failed to login").SetInternal(fmt.Errorf("AuthHandler.Login -> AuthService.Login: %w", err))
	}

	return response.JSON(w, http.StatusOK, LoginResponse{
		Status:    "ok",
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339Nano),
		User:      NewUserBrief(result.User),
	})
}

type RegisterResponse struct {
	Status    string         `json:"status"`
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
	Status    string    `json:"status"`
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

func NewRegisterResponse(user repository.User, portfolio repository.Portfolio) RegisterResponse {
	return RegisterResponse{
		Status:    "ok",
		User:      NewUserBrief(user),
		Portfolio: NewPortfolioBrief(portfolio),
	}
}

func NewPortfolioBrief(portfolio repository.Portfolio) PortfolioBrief {
	return PortfolioBrief{
		PortfolioID:      portfolio.PortfolioID,
		Name:             portfolio.Name,
		BaseCurrencyCode: portfolio.BaseCurrencyCode,
		CreatedAt:        portfolio.CreatedAt.Format(time.RFC3339Nano),
	}
}

func NewUserBrief(user repository.User) UserBrief {
	return UserBrief{
		UserId:    user.ID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		Status:    user.Status,
		Rules:     user.Rules,
		CreatedAt: user.CreatedAt.Format(time.RFC3339Nano),
	}
}
