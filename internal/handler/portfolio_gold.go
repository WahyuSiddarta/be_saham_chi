package handler

import (
	"net/http"

	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

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

func bindGold(req *http.Request) (service.GoldTransactionInput, error) {
	var r GoldTransactionRequest
	if err := binding.BindJSON(req.Body, &r); err != nil {
		return service.GoldTransactionInput{}, newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	date, err := parseOptionalRFC3339(r.TransactionDate, "transaction_date")
	if err != nil {
		return service.GoldTransactionInput{}, newHTTPError(http.StatusBadRequest, err.Error())
	}
	return service.GoldTransactionInput{AccountID: r.AccountID, TransactionType: r.TransactionType, QuantityGrams: r.QuantityGrams, Price: r.Price, FeeAmount: r.FeeAmount, TaxAmount: r.TaxAmount, TransactionDate: date, Notes: r.Notes}, nil
}

func bindInitialGold(req *http.Request) (service.GoldInput, error) {
	var r GoldRequest
	if err := binding.BindJSON(req.Body, &r); err != nil {
		return service.GoldInput{}, newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	date, err := parseOptionalRFC3339(r.TransactionDate, "transaction_date")
	if err != nil {
		return service.GoldInput{}, newHTTPError(http.StatusBadRequest, err.Error())
	}
	return service.GoldInput{AccountID: r.AccountID, AccountName: r.AccountName, QuantityGrams: r.QuantityGrams, Price: r.Price, FeeAmount: r.FeeAmount, TaxAmount: r.TaxAmount, TransactionDate: date, Notes: r.Notes}, nil
}

func (h Handler) CreateGold(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	in, e := bindInitialGold(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.CreateGold(req.Context(), id, chi.URLParam(req, "portfolio_id"), in)
	if e != nil {
		return portfolioHTTPError(e, "failed to create gold")
	}
	return response.Success(w, http.StatusCreated, v)
}
func (h Handler) GetGold(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.GetGold(req.Context(), id, chi.URLParam(req, "portfolio_id"))
	if e != nil {
		return portfolioHTTPError(e, "failed to get gold")
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) CreateGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	in, e := bindGold(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.CreateGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), in)
	if e != nil {
		return portfolioHTTPError(e, "failed to create gold transaction")
	}
	return response.Success(w, http.StatusCreated, v)
}
func (h Handler) ListGoldTransactions(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.ListGoldTransactions(req.Context(), id, chi.URLParam(req, "portfolio_id"))
	if e != nil {
		return portfolioHTTPError(e, "failed to list gold transactions")
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) GetGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.GetGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if e != nil {
		return portfolioHTTPError(e, "failed to get gold transaction")
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) UpdateGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	in, e := bindGold(req)
	if e != nil {
		return e
	}
	v, e := h.goldService.UpdateGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), in)
	if e != nil {
		return portfolioHTTPError(e, "failed to update gold transaction")
	}
	return response.Success(w, http.StatusOK, v)
}
func (h Handler) DeleteGoldTransaction(w http.ResponseWriter, req *http.Request) error {
	id, e := requiredUserID(req)
	if e != nil {
		return e
	}
	e = h.goldService.DeleteGoldTransaction(req.Context(), id, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if e != nil {
		return portfolioHTTPError(e, "failed to delete gold transaction")
	}
	return response.Success(w, http.StatusOK, nil)
}
