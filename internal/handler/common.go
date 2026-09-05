package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"
	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type Handler struct {
	status            string
	log               *zerolog.Logger
	authService       *service.AuthService
	commodityService  *service.CommodityService
	stockService      *service.StockService
	portfolioService  *service.PortfolioService
	cashService       *service.CashService
	bondService       *service.BondService
	goldService       *service.GoldService
	masterDataService *service.MasterDataService
}

func New(status string, log *zerolog.Logger, authService *service.AuthService, domains Domains) Handler {
	return Handler{
		status:            status,
		log:               log,
		authService:       authService,
		commodityService:  domains.Commodity,
		stockService:      domains.Stock,
		portfolioService:  domains.Portfolio,
		cashService:       domains.Cash,
		bondService:       domains.Bond,
		goldService:       domains.Gold,
		masterDataService: domains.MasterData,
	}
}

func (h Handler) fail(w http.ResponseWriter, status int, message string) {
	if err := response.Fail(w, status, message); err != nil {
		h.log.Error().Err(err).Msg("write failure response")
	}
}

type Domains struct {
	Commodity  *service.CommodityService
	Stock      *service.StockService
	Portfolio  *service.PortfolioService
	Cash       *service.CashService
	Bond       *service.BondService
	Gold       *service.GoldService
	MasterData *service.MasterDataService
}

// Handle logs response errors and writes a fallback only before a response starts.
func (h Handler) Handle(next func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		if err := next(writer, r); err != nil {
			h.logRequestError(r, http.StatusInternalServerError, "response failed", err)
			if writer.Status() == 0 {
				h.fail(writer, http.StatusInternalServerError, "Internal Server Error")
			}
		}
	}
}

func (h Handler) logRequestError(req *http.Request, status int, message string, err error) {
	event := h.log.Warn()
	if status >= http.StatusInternalServerError {
		event = h.log.Error()
	}
	event.Err(err).Int("status_code", status).Str("request_id", chimiddleware.GetReqID(req.Context())).Str("method", req.Method).Str("path", req.URL.Path).Str("response_message", message).Msg("request failed")
}

type httpErrorRule struct {
	target  error
	status  int
	message string
}

func NewRegisterResponse(user repository.User, portfolio repository.Portfolio) RegisterResponse {
	return RegisterResponse{
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

func NewQuoteResponse(quote repository.MarketPrice) QuoteResponse {
	return QuoteResponse{
		Symbol:    quote.Symbol,
		Open:      quote.Open,
		High:      quote.High,
		Low:       quote.Low,
		Close:     quote.Close,
		Volume:    quote.Volume,
		Source:    quote.Source,
		FetchedAt: quote.FetchedAt,
	}
}

func NewKlineListResponse(klines []repository.MarketKline) KlineListResponse {
	return helper.MapSlice(klines, func(kline repository.MarketKline) KlineResponse {
		return KlineResponse{
			Symbol:    kline.Symbol,
			Interval:  kline.Interval,
			OpenTime:  kline.OpenTime,
			Open:      kline.Open,
			High:      kline.High,
			Low:       kline.Low,
			Close:     kline.Close,
			Volume:    kline.Volume,
			Source:    kline.Source,
			FetchedAt: kline.FetchedAt,
		}
	})
}

func newMasterDataResponse(item repository.MasterData) MasterDataResponse {
	return MasterDataResponse{
		Key:       item.Key,
		Value:     item.Value,
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func NewPortfolioListResponse(portfolios []repository.Portfolio) PortfolioListResponse {
	return helper.MapSlice(portfolios, newPortfolioItemResponse)
}

func NewPortfolioResponse(portfolio repository.Portfolio) PortfolioResponse {
	return PortfolioResponse{PortfolioItemResponse: newPortfolioItemResponse(portfolio)}
}

func newPortfolioItemResponse(portfolio repository.Portfolio) PortfolioItemResponse {
	return PortfolioItemResponse{
		PortfolioID:      portfolio.PortfolioID,
		UserID:           portfolio.UserID,
		Name:             portfolio.Name,
		BaseCurrencyCode: portfolio.BaseCurrencyCode,
		IsMain:           portfolio.IsMain,
		CreatedAt:        helper.FormatTime(portfolio.CreatedAt),
		UpdatedAt:        helper.FormatTime(portfolio.UpdatedAt),
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
	userID, ok := middleware.UserIDFromContext(req.Context())
	if !ok {
		return "", errors.New("missing user_id")
	}
	return userID, nil
}

func mapPortfolioHTTPError(err error, fallback string, rules ...httpErrorRule) (int, string) {
	for _, rule := range rules {
		if errors.Is(err, rule.target) {
			return rule.status, rule.message
		}
	}
	return http.StatusInternalServerError, fallback
}

func portfolioHTTPError(err error, fallback string) (int, string) {
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

func NewBondListResponse(bonds []repository.PortfolioBond) BondListResponse {
	return helper.MapSlice(bonds, newBondItemResponse)
}

func NewBondResponse(bond repository.PortfolioBond) BondResponse {
	return BondResponse{BondItemResponse: newBondItemResponse(bond)}
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
			MaturityDate:                helper.FormatDate(bond.Term.MaturityDate),
			NextCouponDate:              helper.FormatOptionalDate(schedule.NextCouponDate),
			CouponAmountPerPeriod:       schedule.CouponAmountPerPeriod,
			CouponPaymentsPerYear:       schedule.CouponPaymentsPerYear,
			IsNextCouponAtMaturity:      schedule.IsNextCouponAtMaturity,
			PrincipalReturnedAtMaturity: schedule.PrincipalReturnedAmount,
			LatestValuation:             newBondValuationResponse(account.LatestValuation),
			UpdatedAt:                   helper.FormatTime(account.UpdatedAt),
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
			IssueDate:       helper.FormatDate(bond.Term.IssueDate),
			MaturityDate:    helper.FormatDate(bond.Term.MaturityDate),
			AnnualRate:      bond.Term.AnnualRate,
			CouponFrequency: bond.Term.CouponFrequency,
			PrincipalValue:  bond.Term.PrincipalValue,
		},
		MaturityDate:                helper.FormatDate(bond.Term.MaturityDate),
		NextCouponDate:              helper.FormatOptionalDate(schedule.NextCouponDate),
		CouponAmountPerPeriod:       schedule.CouponAmountPerPeriod,
		CouponPaymentsPerYear:       schedule.CouponPaymentsPerYear,
		IsNextCouponAtMaturity:      schedule.IsNextCouponAtMaturity,
		PrincipalReturnedAtMaturity: schedule.PrincipalReturnedAmount,
		LatestValuation:             newBondValuationResponse(bond.LatestValuation),
		Accounts:                    accounts,
		UpdatedAt:                   helper.FormatTime(bond.UpdatedAt),
	}
}

func newBondValuationResponse(valuation *repository.PortfolioBondValuation) *BondValuationResponse {
	if valuation == nil {
		return nil
	}
	return &BondValuationResponse{
		ValuationDate: helper.FormatDate(valuation.ValuationDate),
		Price:         valuation.Price,
		MarketValue:   valuation.MarketValue,
		Source:        valuation.Source,
		Notes:         valuation.Notes,
		CreatedAt:     helper.FormatTime(valuation.CreatedAt),
	}
}

func NewBondTransactionListResponse(transactions []repository.PortfolioBondTransaction) BondTransactionListResponse {
	return helper.MapSlice(transactions, newBondTransactionItemResponse)
}

func NewBondTransactionResponse(bondTx repository.PortfolioBondTransaction) BondTransactionResponse {
	return BondTransactionResponse{BondTransactionItemResponse: newBondTransactionItemResponse(bondTx)}
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
		TransactionDate:     helper.FormatTime(bondTx.TransactionDate),
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
		CreatedAt:           helper.FormatTime(bondTx.CreatedAt),
		UpdatedAt:           helper.FormatTime(bondTx.UpdatedAt),
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
	transactionDate, err := helper.ParseOptionalRFC3339(request.TransactionDate, "transaction_date")
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
	issueDate, err := helper.ParseOptionalDate(request.IssueDate, "issue_date")
	if err != nil {
		return service.BondAssetInput{}, err
	}
	maturityDate, err := helper.ParseOptionalDate(request.MaturityDate, "maturity_date")
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
	valuationDate, err := helper.ParseOptionalDate(request.ValuationDate, "valuation_date")
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
	transactionDate, err := helper.ParseOptionalRFC3339(request.TransactionDate, "transaction_date")
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

func bondHTTPError(err error, fallback string) (int, string) {
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
			UpdatedAt:       helper.FormatTime(account.UpdatedAt),
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
		UpdatedAt:       helper.FormatTime(cash.UpdatedAt),
	}
}

func NewCashTransactionListResponse(transactions []repository.PortfolioCashTransaction) CashTransactionListResponse {
	return helper.MapSlice(transactions, newCashTransactionItemResponse)
}

func NewCashSnapshotListResponse(snapshots []repository.PortfolioCashSnapshot) CashSnapshotListResponse {
	return helper.MapSlice(snapshots, func(snapshot repository.PortfolioCashSnapshot) CashSnapshotResponse {
		return CashSnapshotResponse{
			PortfolioID:     snapshot.PortfolioID,
			AssetClassID:    snapshot.AssetClassID,
			AssetClassCode:  snapshot.AssetClassCode,
			SnapshotDate:    helper.FormatDate(snapshot.SnapshotDate),
			TotalCost:       snapshot.TotalCost,
			MarketValue:     snapshot.MarketValue,
			UnrealizedPnL:   snapshot.UnrealizedPnL,
			RealizedPnL:     snapshot.RealizedPnL,
			TotalPnL:        snapshot.TotalPnL,
			TotalPnLPercent: snapshot.TotalPnLPercent,
			CurrencyCode:    snapshot.CurrencyCode,
			CreatedAt:       helper.FormatTime(snapshot.CreatedAt),
			UpdatedAt:       helper.FormatTime(snapshot.UpdatedAt),
		}
	})
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
		TransactionDate: helper.FormatTime(cashTx.TransactionDate),
		Amount:          cashTx.Amount,
		CostAmount:      cashTx.CostAmount,
		CashFlowAmount:  cashTx.CashFlowAmount(),
		CostFlowAmount:  cashTx.CostFlowAmount(),
		PnLEffectAmount: cashTx.PnLEffectAmount(),
		CurrencyCode:    cashTx.CurrencyCode,
		Notes:           cashTx.Notes,
		CreatedAt:       helper.FormatTime(cashTx.CreatedAt),
		UpdatedAt:       helper.FormatTime(cashTx.UpdatedAt),
	}
}

func bindCashTransactionRequest(req *http.Request) (service.CashTransactionInput, error) {
	var request CashTransactionRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return service.CashTransactionInput{}, errors.New("invalid request body")
	}

	transactionDate, err := helper.ParseOptionalRFC3339(request.TransactionDate, "transaction_date")
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

func cashHTTPError(err error, fallback string) (int, string) {
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

func bindGold(req *http.Request) (service.GoldTransactionInput, error) {
	var r GoldTransactionRequest
	if err := binding.BindJSON(req.Body, &r); err != nil {
		return service.GoldTransactionInput{}, errors.New("invalid request body")
	}
	date, err := helper.ParseOptionalRFC3339(r.TransactionDate, "transaction_date")
	if err != nil {
		return service.GoldTransactionInput{}, err
	}
	return service.GoldTransactionInput{AccountID: r.AccountID, TransactionType: r.TransactionType, QuantityGrams: r.QuantityGrams, Price: r.Price, FeeAmount: r.FeeAmount, TaxAmount: r.TaxAmount, TransactionDate: date, Notes: r.Notes}, nil
}

func bindInitialGold(req *http.Request) (service.GoldInput, error) {
	var r GoldRequest
	if err := binding.BindJSON(req.Body, &r); err != nil {
		return service.GoldInput{}, errors.New("invalid request body")
	}
	date, err := helper.ParseOptionalRFC3339(r.TransactionDate, "transaction_date")
	if err != nil {
		return service.GoldInput{}, err
	}
	return service.GoldInput{AccountID: r.AccountID, AccountName: r.AccountName, QuantityGrams: r.QuantityGrams, Price: r.Price, FeeAmount: r.FeeAmount, TaxAmount: r.TaxAmount, TransactionDate: date, Notes: r.Notes}, nil
}

func newStockItemResponse(stock repository.Stock) StockItemResponse {
	return StockItemResponse{
		Ticker: stock.Ticker, Name: stock.Name, Active: stock.Active,
		CreatedAt: stock.CreatedAt, UpdatedAt: stock.UpdatedAt,
	}
}

func stockKlineDateRange(req *http.Request) (time.Time, time.Time, error) {
	to := time.Now().UTC()
	from := to.AddDate(0, -1, 0)
	var err error
	if raw := req.URL.Query().Get("from"); raw != "" {
		from, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("invalid from date")
		}
	}
	if raw := req.URL.Query().Get("to"); raw != "" {
		to, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("invalid to date")
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from must not be after to")
	}
	return from, to, nil
}

func stockKlinesToResponse(items []repository.StockKline) []StockKlineResponse {
	out := make([]StockKlineResponse, 0, len(items))
	for _, k := range items {
		out = append(out, StockKlineResponse{
			Symbol: k.Symbol, Interval: k.Interval, OpenTime: k.OpenTime,
			Open: k.Open, High: k.High, Low: k.Low, Close: k.Close,
			Volume: k.Volume, Source: k.Source, FetchedAt: k.FetchedAt,
		})
	}
	return out
}
