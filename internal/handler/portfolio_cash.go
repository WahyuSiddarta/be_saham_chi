package handler

import (
	"errors"
	"net/http"

	applogger "github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
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
		return err
	}

	applogger.Info().
		Str("user_id", userID).
		Str("portfolio_id", chi.URLParam(req, "portfolio_id")).
		Msg("AddCash")

	var request AddCashRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
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
		return cashHTTPError(err, "failed to add portfolio cash")
	}

	cash, err := h.cashService.GetCash(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return cashHTTPError(err, "failed to get portfolio cash")
	}

	return response.Success(w, http.StatusCreated, NewPortfolioCashResponse(cash))
}

func (h Handler) GetCash(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	cash, err := h.cashService.GetCash(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return cashHTTPError(err, "failed to get portfolio cash")
	}

	return response.Success(w, http.StatusOK, NewPortfolioCashResponse(cash))
}

func (h Handler) ListCashSnapshots(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	from, err := parseDateQuery(req.URL.Query().Get("from"))
	if err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid from")
	}
	to, err := parseDateQuery(req.URL.Query().Get("to"))
	if err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid to")
	}

	snapshots, err := h.cashService.ListCashSnapshots(req.Context(), userID, chi.URLParam(req, "portfolio_id"), from, to)
	if err != nil {
		return cashHTTPError(err, "failed to list cash snapshots")
	}

	return response.Success(w, http.StatusOK, NewCashSnapshotListResponse(snapshots))
}

func (h Handler) CreateCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindCashTransactionRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	cashTx, err := h.cashService.CreateCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		return cashHTTPError(err, "failed to create cash transaction")
	}

	return response.Success(w, http.StatusCreated, NewCashTransactionResponse(cashTx))
}

func (h Handler) ListCashTransactions(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	transactions, err := h.cashService.ListCashTransactions(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return cashHTTPError(err, "failed to list cash transactions")
	}

	return response.Success(w, http.StatusOK, NewCashTransactionListResponse(transactions))
}

func (h Handler) GetCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	cashTx, err := h.cashService.GetCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		return cashHTTPError(err, "failed to get cash transaction")
	}

	return response.Success(w, http.StatusOK, NewCashTransactionResponse(cashTx))
}

func (h Handler) UpdateCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindCashTransactionRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	cashTx, err := h.cashService.UpdateCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), input)
	if err != nil {
		return cashHTTPError(err, "failed to update cash transaction")
	}

	return response.Success(w, http.StatusOK, NewCashTransactionResponse(cashTx))
}

func (h Handler) DeleteCashTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	err = h.cashService.DeleteCashTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		return cashHTTPError(err, "failed to delete cash transaction")
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

func NewPortfolioCashResponse(cash repository.PortfolioCash) PortfolioCashResponse {
	accounts := make([]PortfolioCashAccountResponse, 0, len(cash.Accounts))
	for _, account := range cash.Accounts {
		accounts = append(accounts, PortfolioCashAccountResponse{
			AccountID:       account.AccountID,
			AccountName:     account.AccountName,
			Quantity:        account.Quantity,
			TotalCost:       account.TotalCost,
			UnrealizedPnL:   account.UnrealizedPnL,
			TotalPnL:        account.TotalPnL,
			TotalPnLPercent: account.TotalPnLPercent,
			UpdatedAt:       formatTime(account.UpdatedAt),
		})
	}

	return PortfolioCashResponse{
		PortfolioID:     cash.PortfolioID,
		AssetID:         cash.AssetID,
		Symbol:          cash.Symbol,
		Name:            cash.Name,
		TotalCash:       cash.TotalCash,
		TotalCost:       cash.TotalCost,
		UnrealizedPnL:   cash.UnrealizedPnL,
		RealizedPnL:     cash.RealizedPnL,
		TotalPnL:        cash.TotalPnL,
		TotalPnLPercent: cash.TotalPnLPercent,
		CurrencyCode:    cash.CurrencyCode,
		Accounts:        accounts,
		UpdatedAt:       formatTime(cash.UpdatedAt),
	}
}

func NewCashTransactionListResponse(transactions []repository.PortfolioCashTransaction) CashTransactionListResponse {
	response := make([]CashTransactionItemResponse, 0, len(transactions))
	for _, cashTx := range transactions {
		response = append(response, newCashTransactionItemResponse(cashTx))
	}
	return response
}

func NewCashSnapshotListResponse(snapshots []repository.PortfolioCashSnapshot) CashSnapshotListResponse {
	response := make([]CashSnapshotResponse, 0, len(snapshots))
	for _, snapshot := range snapshots {
		response = append(response, CashSnapshotResponse{
			PortfolioID:     snapshot.PortfolioID,
			AssetClassID:    snapshot.AssetClassID,
			AssetClassCode:  snapshot.AssetClassCode,
			SnapshotDate:    formatDate(snapshot.SnapshotDate),
			TotalCost:       snapshot.TotalCost,
			MarketValue:     snapshot.MarketValue,
			UnrealizedPnL:   snapshot.UnrealizedPnL,
			RealizedPnL:     snapshot.RealizedPnL,
			TotalPnL:        snapshot.TotalPnL,
			TotalPnLPercent: snapshot.TotalPnLPercent,
			CurrencyCode:    snapshot.CurrencyCode,
			CreatedAt:       formatTime(snapshot.CreatedAt),
			UpdatedAt:       formatTime(snapshot.UpdatedAt),
		})
	}
	return response
}

func NewCashTransactionResponse(cashTx repository.PortfolioCashTransaction) CashTransactionResponse {
	return CashTransactionResponse{CashTransactionItemResponse: newCashTransactionItemResponse(cashTx)}
}

func newCashTransactionItemResponse(cashTx repository.PortfolioCashTransaction) CashTransactionItemResponse {
	return CashTransactionItemResponse{
		TransactionID:   cashTx.TransactionID,
		PortfolioID:     cashTx.PortfolioID,
		AccountID:       cashTx.AccountID,
		AccountName:     cashTx.AccountName,
		AssetID:         cashTx.AssetID,
		TransactionType: cashTx.TransactionType,
		TransactionDate: formatTime(cashTx.TransactionDate),
		Amount:          cashTx.Amount,
		CostAmount:      cashTx.CostAmount,
		CashFlowAmount:  cashTx.CashFlowAmount(),
		CostFlowAmount:  cashTx.CostFlowAmount(),
		PnLEffectAmount: cashTx.PnLEffectAmount(),
		CurrencyCode:    cashTx.CurrencyCode,
		Notes:           cashTx.Notes,
		CreatedAt:       formatTime(cashTx.CreatedAt),
		UpdatedAt:       formatTime(cashTx.UpdatedAt),
	}
}
func bindCashTransactionRequest(req *http.Request) (service.CashTransactionInput, error) {
	var request CashTransactionRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.CashTransactionInput{}, errors.New("invalid request body")
	}

	transactionDate, err := parseOptionalRFC3339(request.TransactionDate, "transaction_date")
	if err != nil {
		return service.CashTransactionInput{}, err
	}

	return service.CashTransactionInput{
		AccountID:       request.AccountID,
		AccountName:     request.AccountName,
		TransactionType: request.TransactionType,
		Amount:          request.Amount,
		CostAmount:      request.CostAmount,
		TransactionDate: transactionDate,
		Notes:           request.Notes,
	}, nil
}
func cashHTTPError(err error, fallback string) error {
	return mapPortfolioHTTPError(err, fallback,
		httpErrorRule{service.ErrInvalidPortfolioID, http.StatusBadRequest, "portfolio_id is required"},
		httpErrorRule{service.ErrInvalidTransactionID, http.StatusBadRequest, "transaction_id is required"},
		httpErrorRule{service.ErrInvalidCashAccount, http.StatusBadRequest, "valid account_id or account_name is required"},
		httpErrorRule{service.ErrInvalidCashAmount, http.StatusBadRequest, "amount must be greater than zero"},
		httpErrorRule{service.ErrInvalidTransactionType, http.StatusBadRequest, "invalid transaction_type"},
		httpErrorRule{service.ErrInvalidSnapshotRange, http.StatusBadRequest, "invalid snapshot range"},
		httpErrorRule{repository.ErrPortfolioNotFound, http.StatusNotFound, "portfolio not found"},
		httpErrorRule{repository.ErrCashTransactionNotFound, http.StatusNotFound, "cash transaction not found"},
	)
}
