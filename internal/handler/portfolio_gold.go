package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

	"github.com/go-chi/chi/v5"
)

type GoldTransactionRequest struct {
	AccountID       string  `json:"account_id"`
	TransactionType string  `json:"transaction_type"`
	QuantityGrams   float64 `json:"quantity_grams"`
	Price           float64 `json:"price"`
	FeeAmount       float64 `json:"fee_amount"`
	TaxAmount       float64 `json:"tax_amount"`
	TransactionDate string  `json:"transaction_date"`
	Notes           string  `json:"notes"`
}

type GoldRequest struct {
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	QuantityGrams   float64 `json:"quantity_grams"`
	Price           float64 `json:"price"`
	FeeAmount       float64 `json:"fee_amount"`
	TaxAmount       float64 `json:"tax_amount"`
	TransactionDate string  `json:"transaction_date"`
	Notes           string  `json:"notes"`
}

func (h Handler) CreateGold(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	in, e := bindInitialGold(req)
	if e != nil {
		return response.Fail(w, http.StatusBadRequest, e.Error())
	}
	v, e := h.goldService.CreateGold(req.Context(), id, chi.URLParam(req, "portfolio_id"), in)
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to create gold")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusCreated, v)
}
func (h Handler) GetGold(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	v, e := h.goldService.GetGold(req.Context(), id, chi.URLParam(req, "portfolio_id"))
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to get gold")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) CreateGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	in, e := bindGold(req)
	if e != nil {
		return response.Fail(w, http.StatusBadRequest, e.Error())
	}
	v, e := h.goldService.CreateGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), in)
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to create gold transaction")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusCreated, v)
}
func (h Handler) ListGoldTransactions(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	v, e := h.goldService.ListGoldTransactions(req.Context(), id, chi.URLParam(req, "portfolio_id"))
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to list gold transactions")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) GetGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	v, e := h.goldService.GetGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to get gold transaction")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) UpdateGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	in, e := bindGold(req)
	if e != nil {
		return response.Fail(w, http.StatusBadRequest, e.Error())
	}
	v, e := h.goldService.UpdateGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), in)
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to update gold transaction")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) DeleteGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return response.Fail(w, http.StatusUnauthorized, e.Error())
	}
	e = h.goldService.DeleteGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if e != nil {
		status, message := portfolioHTTPError(e, "failed to delete gold transaction")
		h.logRequestError(req, status, message, e)
		return response.Fail(w, status, message)
	}
	return response.Success(w, http.StatusOK, nil)
}
