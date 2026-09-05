package handler

import (
	"net/http"

	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

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
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	portfolios, err := h.portfolioService.ListPortfolio(req.Context(), userID)
	if err != nil {
		status, message := portfolioHTTPError(err, "failed to list portfolios")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewPortfolioListResponse(portfolios))
}

func (h Handler) GetPortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	portfolio, err := h.portfolioService.Get(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := portfolioHTTPError(err, "failed to get portfolio")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewPortfolioResponse(portfolio))
}

func (h Handler) CreatePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindPortfolioRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	portfolio, err := h.portfolioService.CreatePortfolio(req.Context(), userID, input)
	if err != nil {
		status, message := portfolioHTTPError(err, "failed to create portfolio")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusCreated, NewPortfolioResponse(portfolio))
}

func (h Handler) UpdatePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindPortfolioRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		status, message := portfolioHTTPError(err, "failed to update portfolio")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewPortfolioResponse(portfolio))
}

func (h Handler) DeletePortfolio(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	var request DeletePortfolioRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}

	err = h.portfolioService.Delete(req.Context(), userID, chi.URLParam(req, "portfolio_id"), request.TargetPortfolioID)
	if err != nil {
		status, message := portfolioHTTPError(err, "failed to delete portfolio")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, nil)
}

type PortfolioListResponse = []PortfolioItemResponse

type PortfolioResponse struct {
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
