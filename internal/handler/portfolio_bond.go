package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

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
		return err
	}

	input, err := bindBondRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.CreateBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		return bondHTTPError(err, "failed to create bond")
	}

	return response.JSON(w, http.StatusCreated, NewBondResponse(bond))
}

func (h Handler) ListBonds(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	bonds, err := h.bondService.ListBonds(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return bondHTTPError(err, "failed to list bonds")
	}

	return response.JSON(w, http.StatusOK, NewBondListResponse(bonds))
}

func (h Handler) GetBond(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	bond, err := h.bondService.GetBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"))
	if err != nil {
		return bondHTTPError(err, "failed to get bond")
	}

	return response.JSON(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) UpdateBond(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindBondAssetRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.UpdateBond(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"), input)
	if err != nil {
		return bondHTTPError(err, "failed to update bond")
	}

	return response.JSON(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) AdjustBondValuation(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindBondValuationRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	bond, err := h.bondService.AdjustBondValuation(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "asset_id"), input)
	if err != nil {
		return bondHTTPError(err, "failed to adjust bond valuation")
	}

	return response.JSON(w, http.StatusOK, NewBondResponse(bond))
}

func (h Handler) ListBondSnapshots(w http.ResponseWriter, req *http.Request) error {
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

	snapshots, err := h.bondService.ListBondSnapshots(req.Context(), userID, chi.URLParam(req, "portfolio_id"), from, to)
	if err != nil {
		return bondHTTPError(err, "failed to list bond snapshots")
	}

	return response.JSON(w, http.StatusOK, NewBondSnapshotListResponse(snapshots))
}

func (h Handler) CreateBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindBondTransactionRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	bondTx, err := h.bondService.CreateBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), input)
	if err != nil {
		return bondHTTPError(err, "failed to create bond transaction")
	}

	return response.JSON(w, http.StatusCreated, NewBondTransactionResponse(bondTx))
}

func (h Handler) ListBondTransactions(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	transactions, err := h.bondService.ListBondTransactions(req.Context(), userID, chi.URLParam(req, "portfolio_id"))
	if err != nil {
		return bondHTTPError(err, "failed to list bond transactions")
	}

	return response.JSON(w, http.StatusOK, NewBondTransactionListResponse(transactions))
}

func (h Handler) GetBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	bondTx, err := h.bondService.GetBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		return bondHTTPError(err, "failed to get bond transaction")
	}

	return response.JSON(w, http.StatusOK, NewBondTransactionResponse(bondTx))
}

func (h Handler) UpdateBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	input, err := bindBondTransactionRequest(req)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, err.Error())
	}

	bondTx, err := h.bondService.UpdateBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"), input)
	if err != nil {
		return bondHTTPError(err, "failed to update bond transaction")
	}

	return response.JSON(w, http.StatusOK, NewBondTransactionResponse(bondTx))
}

func (h Handler) DeleteBondTransaction(w http.ResponseWriter, req *http.Request) error {
	userID, err := requiredUserID(req)
	if err != nil {
		return err
	}

	err = h.bondService.DeleteBondTransaction(req.Context(), userID, chi.URLParam(req, "portfolio_id"), chi.URLParam(req, "transaction_id"))
	if err != nil {
		return bondHTTPError(err, "failed to delete bond transaction")
	}

	return response.NoContent(w, http.StatusNoContent)
}

type BondListResponse struct {
	Status string             `json:"status"`
	Data   []BondItemResponse `json:"data"`
}

type BondResponse struct {
	Status string `json:"status"`
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

type BondTransactionListResponse struct {
	Status string                        `json:"status"`
	Data   []BondTransactionItemResponse `json:"data"`
}

type BondTransactionResponse struct {
	Status string `json:"status"`
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

func NewBondListResponse(bonds []repository.PortfolioBond) BondListResponse {
	response := make([]BondItemResponse, 0, len(bonds))
	for _, bond := range bonds {
		response = append(response, newBondItemResponse(bond))
	}
	return BondListResponse{
		Status: "ok",
		Data:   response,
	}
}

func NewBondResponse(bond repository.PortfolioBond) BondResponse {
	return BondResponse{Status: "ok", BondItemResponse: newBondItemResponse(bond)}
}

func newBondItemResponse(bond repository.PortfolioBond) BondItemResponse {
	now := time.Now().UTC()
	accounts := make([]BondAccountResponse, 0, len(bond.Accounts))
	for _, account := range bond.Accounts {
		schedule := bond.Term.CouponScheduleSummary(account.PrincipalAmount, now)
		accounts = append(accounts, BondAccountResponse{
			AccountID:                   account.AccountID,
			AccountName:                 account.AccountName,
			PrincipalAmount:             account.PrincipalAmount,
			TotalCost:                   account.TotalCost,
			MarketValue:                 account.MarketValue,
			UnrealizedPnL:               account.UnrealizedPnL,
			TotalPnL:                    account.TotalPnL,
			TotalPnLPercent:             account.TotalPnLPercent,
			MaturityDate:                formatDate(bond.Term.MaturityDate),
			NextCouponDate:              formatOptionalDate(schedule.NextCouponDate),
			CouponAmountPerPeriod:       schedule.CouponAmountPerPeriod,
			CouponPaymentsPerYear:       schedule.CouponPaymentsPerYear,
			IsNextCouponAtMaturity:      schedule.IsNextCouponAtMaturity,
			PrincipalReturnedAtMaturity: schedule.PrincipalReturnedAmount,
			LatestValuation:             newBondValuationResponse(account.LatestValuation),
			UpdatedAt:                   formatTime(account.UpdatedAt),
		})
	}
	schedule := bond.Term.CouponScheduleSummary(bond.PrincipalAmount, now)

	return BondItemResponse{
		PortfolioID:     bond.PortfolioID,
		AssetID:         bond.AssetID,
		AccountID:       bond.AccountID,
		AccountName:     bond.AccountName,
		Symbol:          bond.Symbol,
		Name:            bond.Name,
		PrincipalAmount: bond.PrincipalAmount,
		TotalCost:       bond.TotalCost,
		MarketValue:     bond.MarketValue,
		UnrealizedPnL:   bond.UnrealizedPnL,
		RealizedPnL:     bond.RealizedPnL,
		TotalPnL:        bond.TotalPnL,
		TotalPnLPercent: bond.TotalPnLPercent,
		CurrencyCode:    bond.CurrencyCode,
		Term: BondTermResponse{
			IssueDate:       formatDate(bond.Term.IssueDate),
			MaturityDate:    formatDate(bond.Term.MaturityDate),
			AnnualRate:      bond.Term.AnnualRate,
			CouponFrequency: bond.Term.CouponFrequency,
			PrincipalValue:  bond.Term.PrincipalValue,
		},
		MaturityDate:                formatDate(bond.Term.MaturityDate),
		NextCouponDate:              formatOptionalDate(schedule.NextCouponDate),
		CouponAmountPerPeriod:       schedule.CouponAmountPerPeriod,
		CouponPaymentsPerYear:       schedule.CouponPaymentsPerYear,
		IsNextCouponAtMaturity:      schedule.IsNextCouponAtMaturity,
		PrincipalReturnedAtMaturity: schedule.PrincipalReturnedAmount,
		LatestValuation:             newBondValuationResponse(bond.LatestValuation),
		Accounts:                    accounts,
		UpdatedAt:                   formatTime(bond.UpdatedAt),
	}
}

func newBondValuationResponse(valuation *repository.PortfolioBondValuation) *BondValuationResponse {
	if valuation == nil {
		return nil
	}
	return &BondValuationResponse{
		ValuationDate: formatDate(valuation.ValuationDate),
		Price:         valuation.Price,
		MarketValue:   valuation.MarketValue,
		Source:        valuation.Source,
		Notes:         valuation.Notes,
		CreatedAt:     formatTime(valuation.CreatedAt),
	}
}

func NewBondTransactionListResponse(transactions []repository.PortfolioBondTransaction) BondTransactionListResponse {
	response := make([]BondTransactionItemResponse, 0, len(transactions))
	for _, bondTx := range transactions {
		response = append(response, newBondTransactionItemResponse(bondTx))
	}
	return BondTransactionListResponse{
		Status: "ok",
		Data:   response,
	}
}

func NewBondTransactionResponse(bondTx repository.PortfolioBondTransaction) BondTransactionResponse {
	return BondTransactionResponse{Status: "ok", BondTransactionItemResponse: newBondTransactionItemResponse(bondTx)}
}

func newBondTransactionItemResponse(bondTx repository.PortfolioBondTransaction) BondTransactionItemResponse {
	return BondTransactionItemResponse{
		TransactionID:       bondTx.TransactionID,
		PortfolioID:         bondTx.PortfolioID,
		AccountID:           bondTx.AccountID,
		AccountName:         bondTx.AccountName,
		AssetID:             bondTx.AssetID,
		Symbol:              bondTx.Symbol,
		Name:                bondTx.Name,
		TransactionType:     bondTx.TransactionType,
		TransactionDate:     formatTime(bondTx.TransactionDate),
		PrincipalAmount:     bondTx.PrincipalAmount,
		Price:               bondTx.Price,
		GrossAmount:         bondTx.GrossAmount,
		CostAmount:          bondTx.CostAmount,
		AccruedCouponAmount: bondTx.AccruedCouponAmount,
		FeeAmount:           bondTx.FeeAmount,
		TaxAmount:           bondTx.TaxAmount,
		NetAmount:           bondTx.NetAmount,
		CurrencyCode:        bondTx.CurrencyCode,
		Notes:               bondTx.Notes,
		CreatedAt:           formatTime(bondTx.CreatedAt),
		UpdatedAt:           formatTime(bondTx.UpdatedAt),
	}
}

func NewBondSnapshotListResponse(snapshots []repository.PortfolioBondSnapshot) BondSnapshotListResponse {
	return NewCashSnapshotListResponse(snapshots)
}
func bindBondRequest(req *http.Request) (service.BondInput, error) {
	var request BondRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.BondInput{}, errors.New("invalid request body")
	}
	assetInput, err := bondAssetInputFromRequest(BondAssetRequest{
		Symbol:          request.Symbol,
		Name:            request.Name,
		IssueDate:       request.IssueDate,
		MaturityDate:    request.MaturityDate,
		AnnualRate:      request.AnnualRate,
		CouponFrequency: request.CouponFrequency,
		PrincipalValue:  request.PrincipalValue,
	})
	if err != nil {
		return service.BondInput{}, err
	}
	transactionDate, err := parseOptionalRFC3339(request.TransactionDate, "transaction_date")
	if err != nil {
		return service.BondInput{}, err
	}
	return service.BondInput{
		BondAssetInput:      assetInput,
		AccountID:           request.AccountID,
		AccountName:         request.AccountName,
		PrincipalAmount:     request.PrincipalAmount,
		CostAmount:          request.CostAmount,
		AccruedCouponAmount: request.AccruedCouponAmount,
		FeeAmount:           request.FeeAmount,
		MarketValue:         request.MarketValue,
		TransactionDate:     transactionDate,
		Notes:               request.Notes,
	}, nil
}

func bindBondAssetRequest(req *http.Request) (service.BondAssetInput, error) {
	var request BondAssetRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.BondAssetInput{}, errors.New("invalid request body")
	}
	return bondAssetInputFromRequest(request)
}

func bondAssetInputFromRequest(request BondAssetRequest) (service.BondAssetInput, error) {
	issueDate, err := parseOptionalDate(request.IssueDate, "issue_date")
	if err != nil {
		return service.BondAssetInput{}, err
	}
	maturityDate, err := parseOptionalDate(request.MaturityDate, "maturity_date")
	if err != nil {
		return service.BondAssetInput{}, err
	}
	return service.BondAssetInput{
		Symbol:          request.Symbol,
		Name:            request.Name,
		IssueDate:       issueDate,
		MaturityDate:    maturityDate,
		AnnualRate:      request.AnnualRate,
		CouponFrequency: request.CouponFrequency,
		PrincipalValue:  request.PrincipalValue,
	}, nil
}

func bindBondValuationRequest(req *http.Request) (service.BondValuationInput, error) {
	var request BondValuationRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.BondValuationInput{}, errors.New("invalid request body")
	}
	valuationDate, err := parseOptionalDate(request.ValuationDate, "valuation_date")
	if err != nil {
		return service.BondValuationInput{}, err
	}
	return service.BondValuationInput{
		AccountID:     request.AccountID,
		ValuationDate: valuationDate,
		Price:         request.Price,
		MarketValue:   request.MarketValue,
		Notes:         request.Notes,
	}, nil
}

func bindBondTransactionRequest(req *http.Request) (service.BondTransactionInput, error) {
	var request BondTransactionRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.BondTransactionInput{}, errors.New("invalid request body")
	}
	transactionDate, err := parseOptionalRFC3339(request.TransactionDate, "transaction_date")
	if err != nil {
		return service.BondTransactionInput{}, err
	}
	return service.BondTransactionInput{
		AccountID:           request.AccountID,
		AccountName:         request.AccountName,
		AssetID:             request.AssetID,
		TransactionType:     request.TransactionType,
		PrincipalAmount:     request.PrincipalAmount,
		Price:               request.Price,
		GrossAmount:         request.GrossAmount,
		CostAmount:          request.CostAmount,
		AccruedCouponAmount: request.AccruedCouponAmount,
		FeeAmount:           request.FeeAmount,
		TaxAmount:           request.TaxAmount,
		NetAmount:           request.NetAmount,
		TransactionDate:     transactionDate,
		Notes:               request.Notes,
	}, nil
}
func bondHTTPError(err error, fallback string) error {
	return mapPortfolioHTTPError(err, fallback,
		httpErrorRule{service.ErrInvalidPortfolioID, http.StatusBadRequest, "portfolio_id is required"},
		httpErrorRule{service.ErrInvalidTransactionID, http.StatusBadRequest, "transaction_id is required"},
		httpErrorRule{service.ErrInvalidBondAsset, http.StatusBadRequest, "valid bond asset is required"},
		httpErrorRule{service.ErrInvalidBondAccount, http.StatusBadRequest, "valid account_id or account_name is required"},
		httpErrorRule{service.ErrInvalidBondAmount, http.StatusBadRequest, "bond amounts must be valid and non-negative"},
		httpErrorRule{service.ErrInvalidBondTerm, http.StatusBadRequest, "invalid bond term"},
		httpErrorRule{service.ErrInvalidBondCouponFrequency, http.StatusBadRequest, "coupon_frequency must be one of monthly, quarterly, semiannual, or annual"},
		httpErrorRule{service.ErrInvalidBondValuation, http.StatusBadRequest, "price or market_value is required"},
		httpErrorRule{service.ErrInvalidTransactionType, http.StatusBadRequest, "invalid transaction_type"},
		httpErrorRule{service.ErrInvalidSnapshotRange, http.StatusBadRequest, "invalid snapshot range"},
		httpErrorRule{repository.ErrPortfolioNotFound, http.StatusNotFound, "portfolio not found"},
		httpErrorRule{repository.ErrBondAssetNotFound, http.StatusNotFound, "bond asset not found"},
		httpErrorRule{repository.ErrBondAccountNotFound, http.StatusNotFound, "bond account not found"},
		httpErrorRule{repository.ErrBondHoldingNotFound, http.StatusNotFound, "bond holding not found"},
		httpErrorRule{repository.ErrBondTransactionNotFound, http.StatusNotFound, "bond transaction not found"},
		httpErrorRule{repository.ErrBondHoldingQuantity, http.StatusConflict, "bond holding quantity is insufficient"},
	)
}
