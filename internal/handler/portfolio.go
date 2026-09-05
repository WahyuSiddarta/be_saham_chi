package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

type PortfolioRequest struct {
	Name string `json:"name"`
}

type DeletePortfolioRequest struct {
	TargetPortfolioID string `json:"target_portfolio_id"`
}

func (h Handler) ListPortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	portfolios, err := h.portfolioService.ListPortfolio(req.Context(), userID)
	if err != nil {
		return portfolioHTTPError(err, "failed to list portfolios")
	}

	return response.JSON(w, http.StatusOK, NewPortfolioListResponse(portfolios))
}

func (h Handler) GetPortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	portfolio, err := h.portfolioService.Get(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return portfolioHTTPError(err, "failed to get portfolio")
	}

	return response.JSON(w, http.StatusOK, NewPortfolioResponse(portfolio))
}

func (h Handler) CreatePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindPortfolioRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	portfolio, err := h.portfolioService.CreatePortfolio(req.Context(), userID, input)
	if err != nil {
		return portfolioHTTPError(err, "failed to create portfolio")
	}

	return response.JSON(w, http.StatusCreated, NewPortfolioResponse(portfolio))
}

func (h Handler) UpdatePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindPortfolioRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		return portfolioHTTPError(err, "failed to update portfolio")
	}

	return response.JSON(w, http.StatusOK, NewPortfolioResponse(portfolio))
}

func (h Handler) DeletePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	var request DeletePortfolioRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}

	err = h.portfolioService.Delete(req.Context(), userID, chi.URLParam(req, "portfolio_id"), request.TargetPortfolioID)
	if err != nil {
		return portfolioHTTPError(err, "failed to delete portfolio")
	}

	return response.NoContent(w, http.StatusNoContent)
}

type PortfolioListResponse struct {
	Status string                  `json:"status"`
	Data   []PortfolioItemResponse `json:"data"`
}

type PortfolioResponse struct {
	Status string `json:"status"`
	PortfolioItemResponse
}

type PortfolioItemResponse struct {
	PortfolioID      string `json:"portfolio_id"`
	UserID           string `json:"user_id"`
	Name             string `json:"name"`
	BaseCurrencyCode string `json:"base_currency_code"`
	IsMain           bool   `json:"is_main"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func NewPortfolioListResponse(portfolios []repository.Portfolio) PortfolioListResponse {
	response := make([]PortfolioItemResponse, 0, len(portfolios))
	for _, portfolio := range portfolios {
		response = append(response, newPortfolioItemResponse(portfolio))
	}
	return PortfolioListResponse{
		Status: "ok",
		Data:   response,
	}
}

func NewPortfolioResponse(portfolio repository.Portfolio) PortfolioResponse {
	return PortfolioResponse{Status: "ok", PortfolioItemResponse: newPortfolioItemResponse(portfolio)}
}

func newPortfolioItemResponse(portfolio repository.Portfolio) PortfolioItemResponse {
	return PortfolioItemResponse{
		PortfolioID:      portfolio.PortfolioID,
		UserID:           portfolio.UserID,
		Name:             portfolio.Name,
		BaseCurrencyCode: portfolio.BaseCurrencyCode,
		IsMain:           portfolio.IsMain,
		CreatedAt:        formatTime(portfolio.CreatedAt),
		UpdatedAt:        formatTime(portfolio.UpdatedAt),
	}
}
func bindPortfolioRequest(req *http.Request) (service.PortfolioInput, error) {
	var request PortfolioRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.PortfolioInput{}, errors.New("invalid request body")
	}
	return service.PortfolioInput{
		Name: request.Name,
	}, nil
}

func requiredUserID(req *http.Request) (string, error) {
	userID, ok := auth.UserIDFromContext(req.Context())
	if !ok {
		return "", newHTTPError(http.StatusUnauthorized, "missing user_id")
	}
	return userID, nil
}

type httpErrorRule struct {
	target  error
	status  int
	message string
}

func mapPortfolioHTTPError(err error, fallback string, rules ...httpErrorRule) error {
	for _, rule := range rules {
		if errors.Is(err, rule.target) {
			return newHTTPError(rule.status, rule.message).SetInternal(err)
		}
	}
	return internalServerError(fallback, err)
}

func portfolioHTTPError(err error, fallback string) error {
	return mapPortfolioHTTPError(err, fallback,
		httpErrorRule{service.ErrInvalidPortfolioID, http.StatusBadRequest, "portfolio_id and target_portfolio_id are required"},
		httpErrorRule{service.ErrInvalidPortfolioName, http.StatusBadRequest, "name is required"},
		httpErrorRule{service.ErrInvalidPortfolioMove, http.StatusBadRequest, "target_portfolio_id must be different from portfolio_id"},
		httpErrorRule{service.ErrDuplicatePortfolioName, http.StatusConflict, "portfolio name already exists"},
		httpErrorRule{repository.ErrMainPortfolioProtected, http.StatusConflict, "main portfolio cannot be deleted"},
		httpErrorRule{repository.ErrPortfolioNotFound, http.StatusNotFound, "portfolio not found"},
		httpErrorRule{service.ErrInvalidGoldAccount, http.StatusBadRequest, "account_id is invalid for this portfolio"},
		httpErrorRule{service.ErrInvalidGoldTransaction, http.StatusBadRequest, "gold quantity and price must be positive; fee and tax must be non-negative"},
		httpErrorRule{service.ErrInvalidTransactionType, http.StatusBadRequest, "transaction_type must be buy or sell"},
		httpErrorRule{service.ErrInvalidTransactionID, http.StatusBadRequest, "transaction_id is required"},
		httpErrorRule{service.ErrInsufficientGoldQuantity, http.StatusConflict, "insufficient gold quantity at transaction date"},
		httpErrorRule{repository.ErrGoldTransactionNotFound, http.StatusNotFound, "gold transaction not found"},
	)
}
func internalServerError(message string, err error) error {
	return newHTTPError(http.StatusInternalServerError, message).SetInternal(err)
}

func parseDateQuery(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", value)
}

func parseOptionalDate(value string, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, errors.New("invalid " + field)
	}
	return parsed, nil
}

func parseOptionalRFC3339(value string, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New("invalid " + field)
	}
	return parsed, nil
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatOptionalDate(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	formatted := formatDate(t)
	return &formatted
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
