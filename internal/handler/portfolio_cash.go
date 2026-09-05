package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"
	applogger "github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

type AddCashRequest struct {
	AccountID   string  `json:"account_id"`
	AccountName string  `json:"account_name"`
	Amount      float64 `json:"amount"`
	Notes       string  `json:"notes"`
}
type CashTransactionRequest struct {
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	TransactionType string  `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	CostAmount      float64 `json:"cost_amount"`
	TransactionDate string  `json:"transaction_date"`
	Notes           string  `json:"notes"`
}

func (h Handler) AddCash(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	applogger.Info().
		Str("user_id", userID).
		Str("portfolio_id", chi.URLParam(req, "portfolio_id")).
		Msg("AddCash")

	var request AddCashRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}

	accountName := request.AccountName
	if accountName == "" && request.AccountID == "" {
		accountName = "Cash"
	}

	_, err = h.cashService.CreateCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), service.CashTransactionInput{
		AccountID:       request.AccountID,
		AccountName:     accountName,
		TransactionType: "deposit",
		Amount:          request.Amount,
		Notes:           request.Notes,
	})
	if err != nil {
		status, message := cashHTTPError(err, "failed to add portfolio cash")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	cash, err := h.cashService.GetCash(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := cashHTTPError(err, "failed to get portfolio cash")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusCreated, NewPortfolioCashResponse(cash))
}

func (h Handler) GetCash(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	cash, err := h.cashService.GetCash(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := cashHTTPError(err, "failed to get portfolio cash")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewPortfolioCashResponse(cash))
}

func (h Handler) ListCashSnapshots(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	from, err := helper.ParseDate(req.URL.Query().Get("from"))
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid from")
	}
	to, err := helper.ParseDate(req.URL.Query().Get("to"))
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid to")
	}

	snapshots, err := h.cashService.ListCashSnapshots(req.Context(), userID, chi.URLParam(req, "portfolio_id"), from, to)
	if err != nil {
		status, message := cashHTTPError(err, "failed to list cash snapshots")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewCashSnapshotListResponse(snapshots))
}

func (h Handler) CreateCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindCashTransactionRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	cashTx, err := h.cashService.CreateCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		status, message := cashHTTPError(err, "failed to create cash transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusCreated, NewCashTransactionResponse(cashTx))
}

func (h Handler) ListCashTransactions(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	transactions, err := h.cashService.ListCashTransactions(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := cashHTTPError(err, "failed to list cash transactions")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewCashTransactionListResponse(transactions))
}

func (h Handler) GetCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	cashTx, err := h.cashService.GetCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		status, message := cashHTTPError(err, "failed to get cash transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewCashTransactionResponse(cashTx))
}

func (h Handler) UpdateCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindCashTransactionRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	cashTx, err := h.cashService.UpdateCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), input)
	if err != nil {
		status, message := cashHTTPError(err, "failed to update cash transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewCashTransactionResponse(cashTx))
}

func (h Handler) DeleteCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	err = h.cashService.DeleteCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		status, message := cashHTTPError(err, "failed to delete cash transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, nil)
}

type PortfolioCashResponse struct {
	PortfolioID     string                         `json:"portfolio_id"`
	AssetID         string                         `json:"asset_id"`
	Symbol          string                         `json:"symbol"`
	Name            string                         `json:"name"`
	TotalCash       float64                        `json:"total_cash"`
	TotalCost       float64                        `json:"total_cost"`
	UnrealizedPnL   float64                        `json:"unrealized_pnl"`
	RealizedPnL     float64                        `json:"realized_pnl"`
	TotalPnL        float64                        `json:"total_pnl"`
	TotalPnLPercent float64                        `json:"total_pnl_percent"`
	CurrencyCode    string                         `json:"currency_code"`
	Accounts        []PortfolioCashAccountResponse `json:"accounts"`
	UpdatedAt       string                         `json:"updated_at"`
}
type PortfolioCashAccountResponse struct {
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	Quantity        float64 `json:"quantity"`
	TotalCost       float64 `json:"total_cost"`
	UnrealizedPnL   float64 `json:"unrealized_pnl"`
	TotalPnL        float64 `json:"total_pnl"`
	TotalPnLPercent float64 `json:"total_pnl_percent"`
	UpdatedAt       string  `json:"updated_at"`
}

type CashTransactionListResponse = []CashTransactionItemResponse

type CashSnapshotListResponse = []CashSnapshotResponse

type CashTransactionResponse struct {
	CashTransactionItemResponse
}

type CashTransactionItemResponse struct {
	TransactionID   string  `json:"transaction_id"`
	PortfolioID     string  `json:"portfolio_id"`
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	AssetID         string  `json:"asset_id"`
	TransactionType string  `json:"transaction_type"`
	TransactionDate string  `json:"transaction_date"`
	Amount          float64 `json:"amount"`
	CostAmount      float64 `json:"cost_amount"`
	CashFlowAmount  float64 `json:"cash_flow_amount"`
	CostFlowAmount  float64 `json:"cost_flow_amount"`
	PnLEffectAmount float64 `json:"pnl_effect_amount"`
	CurrencyCode    string  `json:"currency_code"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type CashSnapshotResponse struct {
	PortfolioID     string  `json:"portfolio_id"`
	AssetClassID    int     `json:"asset_class_id"`
	AssetClassCode  string  `json:"asset_class_code"`
	SnapshotDate    string  `json:"snapshot_date"`
	TotalCost       float64 `json:"total_cost"`
	MarketValue     float64 `json:"market_value"`
	UnrealizedPnL   float64 `json:"unrealized_pnl"`
	RealizedPnL     float64 `json:"realized_pnl"`
	TotalPnL        float64 `json:"total_pnl"`
	TotalPnLPercent float64 `json:"total_pnl_percent"`
	CurrencyCode    string  `json:"currency_code"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}
