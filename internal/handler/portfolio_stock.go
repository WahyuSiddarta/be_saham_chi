package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/go-chi/chi/v5"
)

func (h Handler) stockPortfolioResult(w http.ResponseWriter, req *http.Request, status int, value any, err error) error {
	if err != nil {
		code, message := portfolioHTTPError(err, "failed to process stock portfolio")
		h.logRequestError(req, code, message, err)
		return response.Fail(w, code, message)
	}
	return response.Success(w, status, value)
}
func (h Handler) ListPortfolioStocks(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}
	items, err := h.portfolioStockService.ListHoldings(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	return h.stockPortfolioResult(w, req, http.StatusOK, items, err)
}
func (h Handler) ListStockTransactions(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}
	items, err := h.portfolioStockService.ListTransactions(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	return h.stockPortfolioResult(w, req, http.StatusOK, items, err)
}
func (h Handler) SaveStockTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}
	var input repository.StockTransactionCommand
	if err := binding.BindJSON(req.Body, &input); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body; transaction_date must be RFC3339")
	}
	item, err := h.portfolioStockService.SaveTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), input)
	status := http.StatusOK
	if req.Method == http.MethodPost {
		status = http.StatusCreated
	}
	return h.stockPortfolioResult(w, req, status, item, err)
}
func (h Handler) DeleteStockTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}
	err = h.portfolioStockService.DeleteTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	return h.stockPortfolioResult(w, req, http.StatusOK, nil, err)
}
