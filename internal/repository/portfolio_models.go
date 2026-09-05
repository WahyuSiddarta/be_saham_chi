package repository

import "time"

const (
	CashTransactionDeposit    = "deposit"
	CashTransactionWithdrawal = "withdrawal"
	CashTransactionInterest   = "interest"
	CashTransactionDividend   = "dividend"
	CashTransactionCoupon     = "coupon"
	CashTransactionFee        = "fee"
	CashTransactionTax        = "tax"
	CashTransactionMaturity   = "maturity"

	BondTransactionBuy      = "buy"
	BondTransactionSell     = "sell"
	BondTransactionCoupon   = "coupon"
	BondTransactionFee      = "fee"
	BondTransactionTax      = "tax"
	BondTransactionMaturity = "maturity"

	GoldTransactionBuy  = "buy"
	GoldTransactionSell = "sell"
)

type PortfolioGold struct {
	PortfolioID   string                 `db:"portfolio_id" json:"portfolio_id"`
	AssetID       string                 `db:"asset_id" json:"asset_id"`
	Symbol        string                 `db:"symbol" json:"symbol"`
	Name          string                 `db:"name" json:"name"`
	QuantityGrams float64                `db:"quantity_grams" json:"quantity_grams"`
	AverageCost   float64                `db:"average_cost" json:"average_cost"`
	TotalCost     float64                `db:"total_cost" json:"total_cost"`
	RealizedPnL   float64                `db:"realized_pnl" json:"realized_pnl"`
	CurrencyCode  string                 `db:"currency_code" json:"currency_code"`
	Accounts      []PortfolioGoldAccount `db:"accounts" json:"accounts"`
	UpdatedAt     time.Time              `db:"updated_at" json:"updated_at"`
}

type PortfolioGoldAccount struct {
	AccountID     string    `db:"account_id" json:"account_id"`
	AccountName   string    `db:"account_name" json:"account_name"`
	QuantityGrams float64   `db:"quantity_grams" json:"quantity_grams"`
	AverageCost   float64   `db:"average_cost" json:"average_cost"`
	TotalCost     float64   `db:"total_cost" json:"total_cost"`
	RealizedPnL   float64   `db:"realized_pnl" json:"realized_pnl"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type PortfolioGoldTransaction struct {
	TransactionID   string    `db:"transaction_id" json:"transaction_id"`
	PortfolioID     string    `db:"portfolio_id" json:"portfolio_id"`
	AccountID       string    `db:"account_id" json:"account_id"`
	AccountName     string    `db:"account_name" json:"account_name"`
	AssetID         string    `db:"asset_id" json:"asset_id"`
	TransactionType string    `db:"transaction_type" json:"transaction_type"`
	TransactionDate time.Time `db:"transaction_date" json:"transaction_date"`
	QuantityGrams   float64   `db:"quantity_grams" json:"quantity_grams"`
	Price           float64   `db:"price" json:"price"`
	GrossAmount     float64   `db:"gross_amount" json:"gross_amount"`
	CostAmount      float64   `db:"cost_amount" json:"cost_amount"`
	FeeAmount       float64   `db:"fee_amount" json:"fee_amount"`
	TaxAmount       float64   `db:"tax_amount" json:"tax_amount"`
	NetAmount       float64   `db:"net_amount" json:"net_amount"`
	RealizedPnL     float64   `db:"realized_pnl" json:"realized_pnl"`
	CurrencyCode    string    `db:"currency_code" json:"currency_code"`
	Notes           string    `db:"notes" json:"notes"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type Portfolio struct {
	PortfolioID      string    `db:"portfolio_id" json:"portfolio_id"`
	UserID           string    `db:"user_id" json:"user_id"`
	Name             string    `db:"name" json:"name"`
	BaseCurrencyCode string    `db:"base_currency_code" json:"base_currency_code"`
	IsMain           bool      `db:"is_main" json:"is_main"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type PortfolioCash struct {
	PortfolioID     string                 `db:"portfolio_id" json:"portfolio_id"`
	AssetID         string                 `db:"asset_id" json:"asset_id"`
	Symbol          string                 `db:"symbol" json:"symbol"`
	Name            string                 `db:"name" json:"name"`
	TotalCash       float64                `db:"total_cash" json:"total_cash"`
	TotalCost       float64                `db:"total_cost" json:"total_cost"`
	UnrealizedPnL   float64                `db:"unrealized_pnl" json:"unrealized_pnl"`
	RealizedPnL     float64                `db:"realized_pnl" json:"realized_pnl"`
	TotalPnL        float64                `db:"total_pnl" json:"total_pnl"`
	TotalPnLPercent float64                `db:"total_pnl_percent" json:"total_pnl_percent"`
	CurrencyCode    string                 `db:"currency_code" json:"currency_code"`
	Accounts        []PortfolioCashAccount `db:"accounts" json:"accounts"`
	UpdatedAt       time.Time              `db:"updated_at" json:"updated_at"`
}

type PortfolioCashAccount struct {
	AccountID       string    `db:"account_id" json:"account_id"`
	AccountName     string    `db:"account_name" json:"account_name"`
	Quantity        float64   `db:"quantity" json:"quantity"`
	TotalCost       float64   `db:"total_cost" json:"total_cost"`
	UnrealizedPnL   float64   `db:"unrealized_pnl" json:"unrealized_pnl"`
	TotalPnL        float64   `db:"total_pnl" json:"total_pnl"`
	TotalPnLPercent float64   `db:"total_pnl_percent" json:"total_pnl_percent"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

func (a *PortfolioCashAccount) RecalculatePnL() {
	a.UnrealizedPnL = a.Quantity - a.TotalCost
	a.TotalPnL = a.UnrealizedPnL
	if a.TotalCost == 0 {
		a.TotalPnLPercent = 0
		return
	}
	a.TotalPnLPercent = (a.TotalPnL / a.TotalCost) * 100
}

func (c *PortfolioCash) RecalculatePnL() {
	c.UnrealizedPnL = c.TotalCash - c.TotalCost
	c.TotalPnL = c.UnrealizedPnL + c.RealizedPnL
	if c.TotalCost == 0 {
		c.TotalPnLPercent = 0
		return
	}
	c.TotalPnLPercent = (c.TotalPnL / c.TotalCost) * 100
}

type PortfolioCashTransaction struct {
	TransactionID   string    `db:"transaction_id" json:"transaction_id"`
	PortfolioID     string    `db:"portfolio_id" json:"portfolio_id"`
	AccountID       string    `db:"account_id" json:"account_id"`
	AccountName     string    `db:"account_name" json:"account_name"`
	AssetID         string    `db:"asset_id" json:"asset_id"`
	TransactionType string    `db:"transaction_type" json:"transaction_type"`
	TransactionDate time.Time `db:"transaction_date" json:"transaction_date"`
	Amount          float64   `db:"amount" json:"amount"`
	CostAmount      float64   `db:"cost_amount" json:"cost_amount"`
	CurrencyCode    string    `db:"currency_code" json:"currency_code"`
	Notes           string    `db:"notes" json:"notes"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

func (t PortfolioCashTransaction) CashFlowAmount() float64 {
	if IsNegativeCashTransactionType(t.TransactionType) {
		return -t.Amount
	}
	return t.Amount
}

func (t PortfolioCashTransaction) CostFlowAmount() float64 {
	if IsNoCostCashTransactionType(t.TransactionType) {
		return 0
	}
	if IsNegativeCashTransactionType(t.TransactionType) {
		return -t.CostAmount
	}
	return t.CostAmount
}

func (t PortfolioCashTransaction) PnLEffectAmount() float64 {
	return t.CashFlowAmount() - t.CostFlowAmount()
}

func IsCashTransactionType(transactionType string) bool {
	switch transactionType {
	case CashTransactionDeposit,
		CashTransactionWithdrawal,
		CashTransactionInterest,
		CashTransactionDividend,
		CashTransactionCoupon,
		CashTransactionFee,
		CashTransactionTax,
		CashTransactionMaturity:
		return true
	default:
		return false
	}
}

func IsNegativeCashTransactionType(transactionType string) bool {
	switch transactionType {
	case CashTransactionWithdrawal, CashTransactionFee, CashTransactionTax:
		return true
	default:
		return false
	}
}

func IsNoCostCashTransactionType(transactionType string) bool {
	switch transactionType {
	case CashTransactionInterest,
		CashTransactionDividend,
		CashTransactionCoupon,
		CashTransactionFee,
		CashTransactionTax:
		return true
	default:
		return false
	}
}

type PortfolioCashSnapshot struct {
	PortfolioID     string    `db:"portfolio_id" json:"portfolio_id"`
	AssetClassID    int       `db:"asset_class_id" json:"asset_class_id"`
	AssetClassCode  string    `db:"asset_class_code" json:"asset_class_code"`
	SnapshotDate    time.Time `db:"snapshot_date" json:"snapshot_date"`
	TotalCost       float64   `db:"total_cost" json:"total_cost"`
	MarketValue     float64   `db:"market_value" json:"market_value"`
	UnrealizedPnL   float64   `db:"unrealized_pnl" json:"unrealized_pnl"`
	RealizedPnL     float64   `db:"realized_pnl" json:"realized_pnl"`
	TotalPnL        float64   `db:"total_pnl" json:"total_pnl"`
	TotalPnLPercent float64   `db:"total_pnl_percent" json:"total_pnl_percent"`
	CurrencyCode    string    `db:"currency_code" json:"currency_code"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type PortfolioBond struct {
	PortfolioID     string                  `db:"portfolio_id" json:"portfolio_id"`
	AssetID         string                  `db:"asset_id" json:"asset_id"`
	AccountID       string                  `db:"account_id" json:"account_id,omitempty"`
	AccountName     string                  `db:"account_name" json:"account_name,omitempty"`
	Symbol          string                  `db:"symbol" json:"symbol"`
	Name            string                  `db:"name" json:"name"`
	PrincipalAmount float64                 `db:"principal_amount" json:"principal_amount"`
	TotalCost       float64                 `db:"total_cost" json:"total_cost"`
	MarketValue     float64                 `db:"market_value" json:"market_value"`
	UnrealizedPnL   float64                 `db:"unrealized_pnl" json:"unrealized_pnl"`
	RealizedPnL     float64                 `db:"realized_pnl" json:"realized_pnl"`
	TotalPnL        float64                 `db:"total_pnl" json:"total_pnl"`
	TotalPnLPercent float64                 `db:"total_pnl_percent" json:"total_pnl_percent"`
	CurrencyCode    string                  `db:"currency_code" json:"currency_code"`
	Term            PortfolioBondTerm       `db:"term" json:"term"`
	LatestValuation *PortfolioBondValuation `db:"latest_valuation" json:"latest_valuation,omitempty"`
	Accounts        []PortfolioBondAccount  `db:"accounts" json:"accounts"`
	UpdatedAt       time.Time               `db:"updated_at" json:"updated_at"`
}

type PortfolioBondAccount struct {
	AccountID       string                  `db:"account_id" json:"account_id"`
	AccountName     string                  `db:"account_name" json:"account_name"`
	PrincipalAmount float64                 `db:"principal_amount" json:"principal_amount"`
	TotalCost       float64                 `db:"total_cost" json:"total_cost"`
	MarketValue     float64                 `db:"market_value" json:"market_value"`
	UnrealizedPnL   float64                 `db:"unrealized_pnl" json:"unrealized_pnl"`
	TotalPnL        float64                 `db:"total_pnl" json:"total_pnl"`
	TotalPnLPercent float64                 `db:"total_pnl_percent" json:"total_pnl_percent"`
	LatestValuation *PortfolioBondValuation `db:"latest_valuation" json:"latest_valuation,omitempty"`
	UpdatedAt       time.Time               `db:"updated_at" json:"updated_at"`
}

func (a *PortfolioBondAccount) RecalculatePnL() {
	a.UnrealizedPnL = a.MarketValue - a.TotalCost
	a.TotalPnL = a.UnrealizedPnL
	if a.TotalCost == 0 {
		a.TotalPnLPercent = 0
		return
	}
	a.TotalPnLPercent = (a.TotalPnL / a.TotalCost) * 100
}

func (b *PortfolioBond) RecalculatePnL() {
	b.UnrealizedPnL = b.MarketValue - b.TotalCost
	b.TotalPnL = b.UnrealizedPnL + b.RealizedPnL
	if b.TotalCost == 0 {
		b.TotalPnLPercent = 0
		return
	}
	b.TotalPnLPercent = (b.TotalPnL / b.TotalCost) * 100
}

type PortfolioBondTerm struct {
	IssueDate       time.Time `db:"issue_date" json:"issue_date"`
	MaturityDate    time.Time `db:"maturity_date" json:"maturity_date"`
	AnnualRate      float64   `db:"annual_rate" json:"annual_rate"`
	CouponFrequency string    `db:"coupon_frequency" json:"coupon_frequency"`
	PrincipalValue  float64   `db:"principal_value" json:"principal_value"`
}

type BondCouponScheduleSummary struct {
	NextCouponDate          time.Time `db:"next_coupon_date"`
	CouponAmountPerPeriod   float64   `db:"coupon_amount_per_period"`
	CouponPaymentsPerYear   int       `db:"coupon_payments_per_year"`
	IsNextCouponAtMaturity  bool      `db:"is_next_coupon_at_maturity"`
	PrincipalReturnedAmount float64   `db:"principal_returned_amount"`
}

func BondCouponPaymentsPerYear(couponFrequency string) int {
	switch couponFrequency {
	case "monthly":
		return 12
	case "quarterly":
		return 4
	case "semiannual":
		return 2
	case "annual":
		return 1
	default:
		return 0
	}
}

func IsBondCouponFrequency(couponFrequency string) bool {
	return BondCouponPaymentsPerYear(couponFrequency) > 0
}

func (t PortfolioBondTerm) CouponIntervalMonths() int {
	paymentsPerYear := BondCouponPaymentsPerYear(t.CouponFrequency)
	if paymentsPerYear == 0 {
		return 0
	}
	return 12 / paymentsPerYear
}

func (t PortfolioBondTerm) CouponScheduleSummary(principalAmount float64, now time.Time) BondCouponScheduleSummary {
	paymentsPerYear := BondCouponPaymentsPerYear(t.CouponFrequency)
	if principalAmount <= 0 || t.AnnualRate <= 0 || paymentsPerYear == 0 {
		return BondCouponScheduleSummary{}
	}

	summary := BondCouponScheduleSummary{
		CouponAmountPerPeriod: (principalAmount * (t.AnnualRate / 100)) / float64(paymentsPerYear),
		CouponPaymentsPerYear: paymentsPerYear,
	}
	if t.IssueDate.IsZero() || t.MaturityDate.IsZero() {
		return summary
	}

	nowDate := truncateBondDate(now)
	maturityDate := truncateBondDate(t.MaturityDate)
	if !nowDate.Before(maturityDate) {
		return summary
	}

	couponDate := truncateBondDate(t.IssueDate)
	intervalMonths := t.CouponIntervalMonths()
	for {
		couponDate = couponDate.AddDate(0, intervalMonths, 0)
		if couponDate.After(maturityDate) {
			couponDate = maturityDate
		}
		if couponDate.After(nowDate) {
			summary.NextCouponDate = couponDate
			summary.IsNextCouponAtMaturity = couponDate.Equal(maturityDate)
			if summary.IsNextCouponAtMaturity {
				summary.PrincipalReturnedAmount = principalAmount
			}
			return summary
		}
		if couponDate.Equal(maturityDate) {
			return summary
		}
	}
}

func truncateBondDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

type PortfolioBondValuation struct {
	ValuationDate time.Time `db:"valuation_date" json:"valuation_date"`
	Price         float64   `db:"price" json:"price"`
	MarketValue   float64   `db:"market_value" json:"market_value"`
	Source        string    `db:"source" json:"source"`
	Notes         string    `db:"notes" json:"notes"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type PortfolioBondTransaction struct {
	TransactionID       string    `db:"transaction_id" json:"transaction_id"`
	PortfolioID         string    `db:"portfolio_id" json:"portfolio_id"`
	AccountID           string    `db:"account_id" json:"account_id"`
	AccountName         string    `db:"account_name" json:"account_name"`
	AssetID             string    `db:"asset_id" json:"asset_id"`
	Symbol              string    `db:"symbol" json:"symbol"`
	Name                string    `db:"name" json:"name"`
	TransactionType     string    `db:"transaction_type" json:"transaction_type"`
	TransactionDate     time.Time `db:"transaction_date" json:"transaction_date"`
	PrincipalAmount     float64   `db:"principal_amount" json:"principal_amount"`
	Price               float64   `db:"price" json:"price"`
	GrossAmount         float64   `db:"gross_amount" json:"gross_amount"`
	CostAmount          float64   `db:"cost_amount" json:"cost_amount"`
	AccruedCouponAmount float64   `db:"accrued_coupon_amount" json:"accrued_coupon_amount"`
	FeeAmount           float64   `db:"fee_amount" json:"fee_amount"`
	TaxAmount           float64   `db:"tax_amount" json:"tax_amount"`
	NetAmount           float64   `db:"net_amount" json:"net_amount"`
	CurrencyCode        string    `db:"currency_code" json:"currency_code"`
	Notes               string    `db:"notes" json:"notes"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

type PortfolioBondSnapshot = PortfolioCashSnapshot

func IsBondTransactionType(transactionType string) bool {
	switch transactionType {
	case BondTransactionBuy,
		BondTransactionSell,
		BondTransactionCoupon,
		BondTransactionFee,
		BondTransactionTax,
		BondTransactionMaturity:
		return true
	default:
		return false
	}
}
