package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrPortfolioStockTransactionNotFound = errors.New("stock transaction not found")
	ErrPortfolioStockAccount             = errors.New("stock account not found")
	ErrPortfolioStockQuantity            = errors.New("insufficient stock quantity at transaction date")
)

type StockTransactionCommand struct {
	AccountID       string    `json:"account_id"`
	AccountName     string    `json:"account_name"`
	Ticker          string    `json:"ticker"`
	TransactionType string    `json:"transaction_type"`
	Quantity        float64   `json:"quantity"`
	Price           float64   `json:"price"`
	FeeAmount       float64   `json:"fee_amount"`
	TaxAmount       float64   `json:"tax_amount"`
	TransactionDate time.Time `json:"transaction_date"`
	Notes           string    `json:"notes"`
}

type PortfolioStockTransaction struct {
	TransactionID   string    `db:"transaction_id" json:"transaction_id"`
	PortfolioID     string    `db:"portfolio_id" json:"portfolio_id"`
	AccountID       string    `db:"account_id" json:"account_id"`
	AccountName     string    `db:"account_name" json:"account_name"`
	AssetID         string    `db:"asset_id" json:"asset_id"`
	Ticker          string    `db:"ticker" json:"ticker"`
	Name            string    `db:"name" json:"name"`
	TransactionType string    `db:"transaction_type" json:"transaction_type"`
	TransactionDate time.Time `db:"transaction_date" json:"transaction_date"`
	Quantity        float64   `db:"quantity" json:"quantity"`
	Price           float64   `db:"price" json:"price"`
	GrossAmount     float64   `db:"gross_amount" json:"gross_amount"`
	CostAmount      float64   `db:"cost_amount" json:"cost_amount"`
	FeeAmount       float64   `db:"fee_amount" json:"fee_amount"`
	TaxAmount       float64   `db:"tax_amount" json:"tax_amount"`
	NetAmount       float64   `db:"net_amount" json:"net_amount"`
	RealizedPnL     float64   `db:"realized_pnl" json:"realized_pnl"`
	Notes           string    `db:"notes" json:"notes"`
}

type PortfolioStockHolding struct {
	AssetID        string     `db:"asset_id" json:"asset_id"`
	Ticker         string     `db:"ticker" json:"ticker"`
	Name           string     `db:"name" json:"name"`
	AccountID      string     `db:"account_id" json:"account_id"`
	AccountName    string     `db:"account_name" json:"account_name"`
	Quantity       float64    `db:"quantity" json:"quantity"`
	AverageCost    float64    `db:"average_cost" json:"average_cost"`
	TotalCost      float64    `db:"total_cost" json:"total_cost"`
	RealizedPnL    float64    `db:"realized_pnl" json:"realized_pnl"`
	Price          *float64   `json:"price"`
	PriceFetchedAt *time.Time `json:"price_fetched_at"`
	MarketValue    *float64   `json:"market_value"`
	UnrealizedPnL  *float64   `json:"unrealized_pnl"`
}

const stockTransactionSelect = `SELECT pt.transaction_id::text,pt.portfolio_id::text,pt.account_id::text,
 pa.name AS account_name,pt.asset_id::text,a.symbol AS ticker,a.name,pt.transaction_type,pt.transaction_date,
 pt.quantity::float8,pt.price::float8,pt.gross_amount::float8,pt.cost_amount::float8,
 pt.fee_amount::float8,pt.tax_amount::float8,pt.net_amount::float8,
 (CASE WHEN pt.transaction_type='sell' THEN pt.net_amount-pt.cost_amount ELSE 0 END)::float8 AS realized_pnl,
 COALESCE(pt.notes,'') AS notes FROM portfolio_transactions pt
 JOIN portfolios p ON p.portfolio_id=pt.portfolio_id
 JOIN portfolio_accounts pa ON pa.account_id=pt.account_id
 JOIN assets a ON a.asset_id=pt.asset_id JOIN asset_classes ac ON ac.asset_class_id=a.asset_class_id`

func (r *Repository) ListPortfolioStocks(ctx context.Context, userID, portfolioID string) ([]PortfolioStockHolding, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	items := []PortfolioStockHolding{}
	err := r.db.SelectContext(ctx, &items, `WITH positions AS (
 SELECT pt.account_id,pt.asset_id,SUM(CASE WHEN pt.transaction_type='sell' THEN pt.net_amount-pt.cost_amount ELSE 0 END) AS realized_pnl
 FROM portfolio_transactions pt JOIN assets a ON a.asset_id=pt.asset_id JOIN asset_classes ac ON ac.asset_class_id=a.asset_class_id
 WHERE pt.portfolio_id=$1 AND ac.code='stock' GROUP BY pt.account_id,pt.asset_id)
 SELECT x.asset_id::text,a.symbol AS ticker,a.name,x.account_id::text,pa.name AS account_name,
 COALESCE(h.quantity,0)::float8 AS quantity,COALESCE(h.average_cost,0)::float8 AS average_cost,
 COALESCE(h.total_cost,0)::float8 AS total_cost,x.realized_pnl::float8
 FROM positions x JOIN assets a ON a.asset_id=x.asset_id JOIN portfolio_accounts pa ON pa.account_id=x.account_id
 LEFT JOIN portfolio_holdings h ON h.portfolio_id=$1 AND h.account_id=x.account_id AND h.asset_id=x.asset_id
 ORDER BY a.symbol,pa.name`, portfolioID)
	return items, err
}

func (r *Repository) ListStockTransactions(ctx context.Context, userID, portfolioID string) ([]PortfolioStockTransaction, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	items := []PortfolioStockTransaction{}
	err := r.db.SelectContext(ctx, &items, stockTransactionSelect+` WHERE pt.portfolio_id=$1 AND p.user_id=$2 AND ac.code='stock' ORDER BY pt.transaction_date DESC,pt.created_at DESC,pt.transaction_id DESC`, portfolioID, userID)
	return items, err
}

// Lock the owning portfolio before reading or changing a ledger. This serializes
// stock writes, including backdated edits and moves between accounts or tickers.
func lockStockPortfolioTx(ctx context.Context, tx *sqlx.Tx, userID, portfolioID string) error {
	var id string
	err := tx.GetContext(ctx, &id, `SELECT portfolio_id::text FROM portfolios WHERE portfolio_id=$1 AND user_id=$2 FOR UPDATE`, portfolioID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPortfolioNotFound
	}
	return err
}

func stockTransactionTx(ctx context.Context, tx *sqlx.Tx, userID, portfolioID, id string) (PortfolioStockTransaction, error) {
	var item PortfolioStockTransaction
	err := tx.GetContext(ctx, &item, stockTransactionSelect+` WHERE pt.transaction_id=$1 AND pt.portfolio_id=$2 AND p.user_id=$3 AND ac.code='stock'`, id, portfolioID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrPortfolioStockTransactionNotFound
	}
	return item, err
}

func resolveStockAccountTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, input StockTransactionCommand) (string, error) {
	var id string
	if input.AccountID != "" {
		err := tx.GetContext(ctx, &id, `SELECT account_id::text FROM portfolio_accounts WHERE portfolio_id=$1 AND account_id=$2`, portfolioID, input.AccountID)
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPortfolioStockAccount
		}
		return id, err
	}
	err := tx.GetContext(ctx, &id, `INSERT INTO portfolio_accounts(portfolio_id,name,account_type,currency_code) VALUES($1,$2,'broker','IDR') ON CONFLICT(portfolio_id,name) DO UPDATE SET updated_at=NOW() RETURNING account_id::text`, portfolioID, input.AccountName)
	return id, err
}

func (r *Repository) SaveStockTransaction(ctx context.Context, userID, portfolioID, id string, input StockTransactionCommand) (PortfolioStockTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioStockTransaction, error) {
		empty := PortfolioStockTransaction{}
		if err := lockStockPortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return empty, err
		}
		var old PortfolioStockTransaction
		if id != "" {
			var err error
			old, err = stockTransactionTx(ctx, tx, userID, portfolioID, id)
			if err != nil {
				return empty, err
			}
		}
		accountID, err := resolveStockAccountTx(ctx, tx, portfolioID, input)
		if err != nil {
			return empty, err
		}
		var assetID string
		err = tx.GetContext(ctx, &assetID, `INSERT INTO assets(asset_class_id,symbol,name,currency_code,pricing_method,source,provider_symbol)
   SELECT ac.asset_class_id,s.ticker,s.name,'IDR','api','yahoo',s.ticker||'.JK' FROM stocks s CROSS JOIN asset_classes ac
   WHERE s.ticker=$1 AND ac.code='stock' ON CONFLICT(asset_class_id,symbol) DO UPDATE SET name=EXCLUDED.name RETURNING asset_id::text`, input.Ticker)
		if errors.Is(err, sql.ErrNoRows) {
			return empty, ErrStockNotFound
		}
		if err != nil {
			return empty, err
		}
		if input.TransactionDate.IsZero() {
			if id != "" {
				input.TransactionDate = old.TransactionDate
			} else {
				input.TransactionDate = time.Now().UTC()
			}
		}
		if id == "" {
			err = tx.GetContext(ctx, &id, `INSERT INTO portfolio_transactions(portfolio_id,account_id,asset_id,transaction_type,transaction_date,quantity,price,gross_amount,fee_amount,tax_amount,notes)
    VALUES($1,$2,$3,$4,$5,$6,$7,$6::numeric*$7::numeric,$8,$9,$10) RETURNING transaction_id::text`, portfolioID, accountID, assetID, input.TransactionType, input.TransactionDate, input.Quantity, input.Price, input.FeeAmount, input.TaxAmount, input.Notes)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE portfolio_transactions SET account_id=$1,asset_id=$2,transaction_type=$3,transaction_date=$4,quantity=$5,price=$6,gross_amount=$5::numeric*$6::numeric,fee_amount=$7,tax_amount=$8,notes=$9,updated_at=NOW() WHERE transaction_id=$10 AND portfolio_id=$11`, accountID, assetID, input.TransactionType, input.TransactionDate, input.Quantity, input.Price, input.FeeAmount, input.TaxAmount, input.Notes, id, portfolioID)
		}
		if err != nil {
			return empty, err
		}
		if old.TransactionID != "" && (old.AccountID != accountID || old.AssetID != assetID) {
			if err := rebuildStockHoldingTx(ctx, tx, portfolioID, old.AccountID, old.AssetID); err != nil {
				return empty, err
			}
		}
		if err := rebuildStockHoldingTx(ctx, tx, portfolioID, accountID, assetID); err != nil {
			return empty, err
		}
		return stockTransactionTx(ctx, tx, userID, portfolioID, id)
	})
}

func (r *Repository) DeleteStockTransaction(ctx context.Context, userID, portfolioID, id string) error {
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := lockStockPortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return err
		}
		old, err := stockTransactionTx(ctx, tx, userID, portfolioID, id)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM portfolio_transactions WHERE transaction_id=$1 AND portfolio_id=$2`, id, portfolioID); err != nil {
			return err
		}
		return rebuildStockHoldingTx(ctx, tx, portfolioID, old.AccountID, old.AssetID)
	})
}

func rebuildStockHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID, assetID string) error {
	// Stocks and gold use the same weighted-average buy/sell ledger arithmetic.
	err := rebuildGoldHoldingTx(ctx, tx, portfolioID, accountID, assetID)
	if errors.Is(err, ErrGoldHoldingQuantity) {
		return ErrPortfolioStockQuantity
	}
	return err
}
