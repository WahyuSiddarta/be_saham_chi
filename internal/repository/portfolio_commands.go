package repository

import "time"

const PortfolioCurrencyIDR = "IDR"

type PortfolioCommand struct {
	Name string `db:"name"`
}

type CashTransactionCommand struct {
	AccountID       string    `db:"account_id"`
	AccountName     string    `db:"account_name"`
	TransactionType string    `db:"transaction_type"`
	Amount          float64   `db:"amount"`
	CostAmount      float64   `db:"cost_amount"`
	TransactionDate time.Time `db:"transaction_date"`
	Notes           string    `db:"notes"`
}

type GoldTransactionCommand struct {
	AccountID       string    `db:"account_id"`
	TransactionType string    `db:"transaction_type"`
	QuantityGrams   float64   `db:"quantity_grams"`
	Price           float64   `db:"price"`
	FeeAmount       float64   `db:"fee_amount"`
	TaxAmount       float64   `db:"tax_amount"`
	TransactionDate time.Time `db:"transaction_date"`
	Notes           string    `db:"notes"`
}

type GoldCommand struct {
	AccountID       string    `db:"account_id"`
	AccountName     string    `db:"account_name"`
	QuantityGrams   float64   `db:"quantity_grams"`
	Price           float64   `db:"price"`
	FeeAmount       float64   `db:"fee_amount"`
	TaxAmount       float64   `db:"tax_amount"`
	TransactionDate time.Time `db:"transaction_date"`
	Notes           string    `db:"notes"`
}

type BondAssetCommand struct {
	Symbol          string    `db:"symbol"`
	Name            string    `db:"name"`
	IssueDate       time.Time `db:"issue_date"`
	MaturityDate    time.Time `db:"maturity_date"`
	AnnualRate      float64   `db:"annual_rate"`
	CouponFrequency string    `db:"coupon_frequency"`
	PrincipalValue  float64   `db:"principal_value"`
}

type BondCommand struct {
	BondAssetCommand
	AccountID           string    `db:"account_id"`
	AccountName         string    `db:"account_name"`
	PrincipalAmount     float64   `db:"principal_amount"`
	CostAmount          float64   `db:"cost_amount"`
	AccruedCouponAmount float64   `db:"accrued_coupon_amount"`
	FeeAmount           float64   `db:"fee_amount"`
	MarketValue         float64   `db:"market_value"`
	TransactionDate     time.Time `db:"transaction_date"`
	Notes               string    `db:"notes"`
}

type BondValuationCommand struct {
	AccountID     string    `db:"account_id"`
	ValuationDate time.Time `db:"valuation_date"`
	Price         float64   `db:"price"`
	MarketValue   float64   `db:"market_value"`
	Notes         string    `db:"notes"`
}

type BondTransactionCommand struct {
	AccountID           string    `db:"account_id"`
	AccountName         string    `db:"account_name"`
	AssetID             string    `db:"asset_id"`
	TransactionType     string    `db:"transaction_type"`
	PrincipalAmount     float64   `db:"principal_amount"`
	Price               float64   `db:"price"`
	GrossAmount         float64   `db:"gross_amount"`
	CostAmount          float64   `db:"cost_amount"`
	AccruedCouponAmount float64   `db:"accrued_coupon_amount"`
	FeeAmount           float64   `db:"fee_amount"`
	TaxAmount           float64   `db:"tax_amount"`
	NetAmount           float64   `db:"net_amount"`
	TransactionDate     time.Time `db:"transaction_date"`
	Notes               string    `db:"notes"`
}
