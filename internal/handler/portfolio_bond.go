package handler

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

	"github.com/go-chi/chi/v5"
)

type BondRequest struct {
	AccountID           string  `json:"account_id"`
	AccountName         string  `json:"account_name"`
	Symbol              string  `json:"symbol"`
	Name                string  `json:"name"`
	PrincipalAmount     float64 `json:"principal_amount"`
	CostAmount          float64 `json:"cost_amount"`
	AccruedCouponAmount float64 `json:"accrued_coupon_amount"`
	FeeAmount           float64 `json:"fee_amount"`
	MarketValue         float64 `json:"market_value"`
	IssueDate           string  `json:"issue_date"`
	MaturityDate        string  `json:"maturity_date"`
	AnnualRate          float64 `json:"annual_rate"`
	CouponFrequency     string  `json:"coupon_frequency"`
	PrincipalValue      float64 `json:"principal_value"`
	TransactionDate     string  `json:"transaction_date"`
	Notes               string  `json:"notes"`
}

type BondAssetRequest struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	IssueDate       string  `json:"issue_date"`
	MaturityDate    string  `json:"maturity_date"`
	AnnualRate      float64 `json:"annual_rate"`
	CouponFrequency string  `json:"coupon_frequency"`
	PrincipalValue  float64 `json:"principal_value"`
}

type BondValuationRequest struct {
	AccountID     string  `json:"account_id"`
	ValuationDate string  `json:"valuation_date"`
	Price         float64 `json:"price"`
	MarketValue   float64 `json:"market_value"`
	Notes         string  `json:"notes"`
}

type BondTransactionRequest struct {
	AccountID           string  `json:"account_id"`
	AccountName         string  `json:"account_name"`
	AssetID             string  `json:"asset_id"`
	TransactionType     string  `json:"transaction_type"`
	PrincipalAmount     float64 `json:"principal_amount"`
	Price               float64 `json:"price"`
	GrossAmount         float64 `json:"gross_amount"`
	CostAmount          float64 `json:"cost_amount"`
	AccruedCouponAmount float64 `json:"accrued_coupon_amount"`
	FeeAmount           float64 `json:"fee_amount"`
	TaxAmount           float64 `json:"tax_amount"`
	NetAmount           float64 `json:"net_amount"`
	TransactionDate     string  `json:"transaction_date"`
	Notes               string  `json:"notes"`
}

func (h Handler) CreateBond(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindBondRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.CreateBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		status, message := bondHTTPError(err, "failed to create bond")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusCreated, NewBondResponse(bond))
}

func (h Handler) ListBonds(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	bonds, err := h.bondService.ListBonds(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := bondHTTPError(err, "failed to list bonds")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondListResponse(bonds))
}

func (h Handler) GetBond(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	bond, err := h.bondService.GetBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"))
	if err != nil {
		status, message := bondHTTPError(err, "failed to get bond")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) UpdateBond(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindBondAssetRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.UpdateBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"), input)
	if err != nil {
		status, message := bondHTTPError(err, "failed to update bond")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) AdjustBondValuation(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindBondValuationRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.AdjustBondValuation(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"), input)
	if err != nil {
		status, message := bondHTTPError(err, "failed to adjust bond valuation")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) ListBondSnapshots(w http.ResponseWriter, req *http.Request) error {
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

	snapshots, err := h.bondService.ListBondSnapshots(req.Context(), userID, chi.URLParam(req, "portfolio_id"), from, to)
	if err != nil {
		status, message := bondHTTPError(err, "failed to list bond snapshots")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondSnapshotListResponse(snapshots))
}

func (h Handler) CreateBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindBondTransactionRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	bondTx, err := h.bondService.CreateBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		status, message := bondHTTPError(err, "failed to create bond transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusCreated, NewBondTransactionResponse(bondTx))
}

func (h Handler) ListBondTransactions(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	transactions, err := h.bondService.ListBondTransactions(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		status, message := bondHTTPError(err, "failed to list bond transactions")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondTransactionListResponse(transactions))
}

func (h Handler) GetBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	bondTx, err := h.bondService.GetBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		status, message := bondHTTPError(err, "failed to get bond transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondTransactionResponse(bondTx))
}

func (h Handler) UpdateBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	input, err := bindBondTransactionRequest(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}

	bondTx, err := h.bondService.UpdateBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), input)
	if err != nil {
		status, message := bondHTTPError(err, "failed to update bond transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, NewBondTransactionResponse(bondTx))
}

func (h Handler) DeleteBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return response.Fail(w, http.StatusUnauthorized, err.Error())
	}

	err = h.bondService.DeleteBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		status, message := bondHTTPError(err, "failed to delete bond transaction")
		h.logRequestError(req, status, message, err)
		return response.Fail(w, status, message)
	}

	return response.Success(w, http.StatusOK, nil)
}

type BondListResponse = []BondItemResponse

type BondResponse struct {
	BondItemResponse
}

type BondItemResponse struct {
	PortfolioID                 string                 `json:"portfolio_id"`
	AssetID                     string                 `json:"asset_id"`
	AccountID                   string                 `json:"account_id,omitempty"`
	AccountName                 string                 `json:"account_name,omitempty"`
	Symbol                      string                 `json:"symbol"`
	Name                        string                 `json:"name"`
	PrincipalAmount             float64                `json:"principal_amount"`
	TotalCost                   float64                `json:"total_cost"`
	MarketValue                 float64                `json:"market_value"`
	UnrealizedPnL               float64                `json:"unrealized_pnl"`
	RealizedPnL                 float64                `json:"realized_pnl"`
	TotalPnL                    float64                `json:"total_pnl"`
	TotalPnLPercent             float64                `json:"total_pnl_percent"`
	CurrencyCode                string                 `json:"currency_code"`
	Term                        BondTermResponse       `json:"term"`
	MaturityDate                string                 `json:"maturity_date"`
	NextCouponDate              *string                `json:"next_coupon_date"`
	CouponAmountPerPeriod       float64                `json:"coupon_amount_per_period"`
	CouponPaymentsPerYear       int                    `json:"coupon_payments_per_year"`
	IsNextCouponAtMaturity      bool                   `json:"is_next_coupon_at_maturity"`
	PrincipalReturnedAtMaturity float64                `json:"principal_returned_at_maturity"`
	LatestValuation             *BondValuationResponse `json:"latest_valuation,omitempty"`
	Accounts                    []BondAccountResponse  `json:"accounts"`
	UpdatedAt                   string                 `json:"updated_at"`
}

type BondAccountResponse struct {
	AccountID                   string                 `json:"account_id"`
	AccountName                 string                 `json:"account_name"`
	PrincipalAmount             float64                `json:"principal_amount"`
	TotalCost                   float64                `json:"total_cost"`
	MarketValue                 float64                `json:"market_value"`
	UnrealizedPnL               float64                `json:"unrealized_pnl"`
	TotalPnL                    float64                `json:"total_pnl"`
	TotalPnLPercent             float64                `json:"total_pnl_percent"`
	MaturityDate                string                 `json:"maturity_date"`
	NextCouponDate              *string                `json:"next_coupon_date"`
	CouponAmountPerPeriod       float64                `json:"coupon_amount_per_period"`
	CouponPaymentsPerYear       int                    `json:"coupon_payments_per_year"`
	IsNextCouponAtMaturity      bool                   `json:"is_next_coupon_at_maturity"`
	PrincipalReturnedAtMaturity float64                `json:"principal_returned_at_maturity"`
	LatestValuation             *BondValuationResponse `json:"latest_valuation,omitempty"`
	UpdatedAt                   string                 `json:"updated_at"`
}

type BondTermResponse struct {
	IssueDate       string  `json:"issue_date"`
	MaturityDate    string  `json:"maturity_date"`
	AnnualRate      float64 `json:"annual_rate"`
	CouponFrequency string  `json:"coupon_frequency"`
	PrincipalValue  float64 `json:"principal_value"`
}

type BondValuationResponse struct {
	ValuationDate string  `json:"valuation_date"`
	Price         float64 `json:"price"`
	MarketValue   float64 `json:"market_value"`
	Source        string  `json:"source"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at,omitempty"`
}

type BondTransactionListResponse = []BondTransactionItemResponse

type BondTransactionResponse struct {
	BondTransactionItemResponse
}

type BondTransactionItemResponse struct {
	TransactionID       string  `json:"transaction_id"`
	PortfolioID         string  `json:"portfolio_id"`
	AccountID           string  `json:"account_id"`
	AccountName         string  `json:"account_name"`
	AssetID             string  `json:"asset_id"`
	Symbol              string  `json:"symbol"`
	Name                string  `json:"name"`
	TransactionType     string  `json:"transaction_type"`
	TransactionDate     string  `json:"transaction_date"`
	PrincipalAmount     float64 `json:"principal_amount"`
	Price               float64 `json:"price"`
	GrossAmount         float64 `json:"gross_amount"`
	CostAmount          float64 `json:"cost_amount"`
	AccruedCouponAmount float64 `json:"accrued_coupon_amount"`
	FeeAmount           float64 `json:"fee_amount"`
	TaxAmount           float64 `json:"tax_amount"`
	NetAmount           float64 `json:"net_amount"`
	CurrencyCode        string  `json:"currency_code"`
	Notes               string  `json:"notes"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type BondSnapshotListResponse = CashSnapshotListResponse
type BondSnapshotResponse = CashSnapshotResponse
