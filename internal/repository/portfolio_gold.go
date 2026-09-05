package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

const goldAssetSymbol = "GOLD-GRAM"

func (r *Repository) CreateGold(ctx context.Context, userID, portfolioID string, input GoldCommand, now time.Time) (PortfolioGold, error) {
	err := withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return err
		}
		accountID, err := resolveGoldAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return err
		}
		assetID, err := ensureGoldAssetTx(ctx, tx)
		if err != nil {
			return err
		}
		if input.TransactionDate.IsZero() {
			input.TransactionDate = now
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO portfolio_transactions (portfolio_id, account_id, asset_id, transaction_type, transaction_date,
				quantity, price, gross_amount, cost_amount, fee_amount, tax_amount, net_amount, currency_code, notes)
			VALUES ($1,$2,$3,'buy',$4,$5,$6,$7,0,$8,$9,0,'IDR',$10)
		`, portfolioID, accountID, assetID, input.TransactionDate, input.QuantityGrams, input.Price,
			input.QuantityGrams*input.Price, input.FeeAmount, input.TaxAmount, input.Notes)
		if err != nil {
			return err
		}
		return rebuildGoldHoldingTx(ctx, tx, portfolioID, accountID, assetID)
	})
	if err != nil {
		return PortfolioGold{}, err
	}
	return r.GetGold(ctx, userID, portfolioID)
}

func (r *Repository) CreateGoldTransaction(ctx context.Context, userID, portfolioID string, input GoldTransactionCommand, now time.Time) (PortfolioGoldTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioGoldTransaction, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if err := ensureGoldAccountTx(ctx, tx, portfolioID, input.AccountID); err != nil {
			return PortfolioGoldTransaction{}, err
		}
		assetID, err := ensureGoldAssetTx(ctx, tx)
		if err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if input.TransactionDate.IsZero() {
			input.TransactionDate = now
		}
		var id string
		err = tx.GetContext(ctx, &id, `
			INSERT INTO portfolio_transactions (portfolio_id, account_id, asset_id, transaction_type, transaction_date,
				quantity, price, gross_amount, cost_amount, fee_amount, tax_amount, net_amount, currency_code, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,0,$9,$10,0,'IDR',$11)
			RETURNING transaction_id::text
		`, portfolioID, input.AccountID, assetID, input.TransactionType, input.TransactionDate, input.QuantityGrams, input.Price, input.QuantityGrams*input.Price, input.FeeAmount, input.TaxAmount, input.Notes)
		if err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if err := rebuildGoldHoldingTx(ctx, tx, portfolioID, input.AccountID, assetID); err != nil {
			return PortfolioGoldTransaction{}, err
		}
		return getGoldTransactionTx(ctx, tx, userID, portfolioID, id)
	})
}

func (r *Repository) UpdateGoldTransaction(ctx context.Context, userID, portfolioID, transactionID string, input GoldTransactionCommand, now time.Time) (PortfolioGoldTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioGoldTransaction, error) {
		old, err := getGoldTransactionTx(ctx, tx, userID, portfolioID, transactionID)
		if err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if err := ensureGoldAccountTx(ctx, tx, portfolioID, input.AccountID); err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if input.TransactionDate.IsZero() {
			input.TransactionDate = old.TransactionDate
		}
		tag, err := tx.ExecContext(ctx, `UPDATE portfolio_transactions SET account_id=$1, transaction_type=$2, transaction_date=$3,
			quantity=$4, price=$5, gross_amount=$6, cost_amount=0, fee_amount=$7, tax_amount=$8, net_amount=0,
			notes=$9, updated_at=$10 WHERE transaction_id=$11 AND portfolio_id=$12`, input.AccountID, input.TransactionType,
			input.TransactionDate, input.QuantityGrams, input.Price, input.QuantityGrams*input.Price, input.FeeAmount,
			input.TaxAmount, input.Notes, now, transactionID, portfolioID)
		if err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if affected, err := tag.RowsAffected(); err != nil {
			return PortfolioGoldTransaction{}, err
		} else if affected == 0 {
			return PortfolioGoldTransaction{}, ErrGoldTransactionNotFound
		}
		if err := rebuildGoldHoldingTx(ctx, tx, portfolioID, old.AccountID, old.AssetID); err != nil {
			return PortfolioGoldTransaction{}, err
		}
		if input.AccountID != old.AccountID {
			if err := rebuildGoldHoldingTx(ctx, tx, portfolioID, input.AccountID, old.AssetID); err != nil {
				return PortfolioGoldTransaction{}, err
			}
		}
		return getGoldTransactionTx(ctx, tx, userID, portfolioID, transactionID)
	})
}

func (r *Repository) DeleteGoldTransaction(ctx context.Context, userID, portfolioID, transactionID string) error {
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		old, err := getGoldTransactionTx(ctx, tx, userID, portfolioID, transactionID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM portfolio_transactions WHERE transaction_id=$1`, transactionID); err != nil {
			return err
		}
		return rebuildGoldHoldingTx(ctx, tx, portfolioID, old.AccountID, old.AssetID)
	})
}

func (r *Repository) GetGoldTransaction(ctx context.Context, userID, portfolioID, transactionID string) (PortfolioGoldTransaction, error) {
	return getGoldTransactionTx(ctx, r.db, userID, portfolioID, transactionID)
}

func (r *Repository) ListGoldTransactions(ctx context.Context, userID, portfolioID string) ([]PortfolioGoldTransaction, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryxContext(ctx, goldTransactionSelect+` WHERE pt.portfolio_id=$1 AND a.symbol=$2 ORDER BY pt.transaction_date DESC, pt.created_at DESC`, portfolioID, goldAssetSymbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PortfolioGoldTransaction{}
	for rows.Next() {
		var item PortfolioGoldTransaction
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetGold(ctx context.Context, userID, portfolioID string) (PortfolioGold, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return PortfolioGold{}, err
	}
	gold := PortfolioGold{PortfolioID: portfolioID, CurrencyCode: PortfolioCurrencyIDR, Accounts: []PortfolioGoldAccount{}}
	if err := r.db.GetContext(ctx, &gold, `SELECT a.asset_id::text AS asset_id, a.symbol AS symbol, a.name AS name, a.updated_at AS updated_at FROM assets a JOIN asset_classes ac ON ac.asset_class_id=a.asset_class_id WHERE a.symbol=$1 AND ac.code='commodity' AND a.status=TRUE`, goldAssetSymbol); err != nil {
		return PortfolioGold{}, err
	}
	rows, err := r.db.QueryxContext(ctx, `WITH gold_accounts AS (
		SELECT pt.portfolio_id,pt.account_id,pt.asset_id,MAX(pt.updated_at) AS updated_at,
			SUM(CASE WHEN pt.transaction_type='sell' THEN pt.net_amount-pt.cost_amount ELSE 0 END) AS realized_pnl
		FROM portfolio_transactions pt JOIN assets a ON a.asset_id=pt.asset_id
		WHERE pt.portfolio_id=$1 AND a.symbol=$3 GROUP BY pt.portfolio_id,pt.account_id,pt.asset_id
	) SELECT ga.asset_id::text AS asset_id, a.symbol AS symbol, a.name AS name, pa.account_id::text AS account_id, pa.name AS account_name, COALESCE(ph.quantity,0)::float8 AS quantity_grams, COALESCE(ph.average_cost,0)::float8 AS average_cost, COALESCE(ph.total_cost,0)::float8 AS total_cost, GREATEST(ga.updated_at,COALESCE(ph.updated_at,ga.updated_at)) AS updated_at, ga.realized_pnl::float8 AS realized_pnl FROM gold_accounts ga JOIN assets a ON a.asset_id=ga.asset_id JOIN portfolio_accounts pa ON pa.account_id=ga.account_id
		JOIN portfolios p ON p.portfolio_id=ga.portfolio_id LEFT JOIN portfolio_holdings ph ON ph.portfolio_id=ga.portfolio_id AND ph.account_id=ga.account_id AND ph.asset_id=ga.asset_id
		WHERE p.user_id=$2 ORDER BY pa.name`, portfolioID, userID, goldAssetSymbol)
	if err != nil {
		return PortfolioGold{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var row goldHoldingRow
		if err := rows.StructScan(&row); err != nil {
			return PortfolioGold{}, err
		}
		a := row.PortfolioGoldAccount
		gold.AssetID = row.AssetID
		gold.Symbol = row.Symbol
		gold.Name = row.Name
		gold.QuantityGrams += a.QuantityGrams
		gold.TotalCost += a.TotalCost
		gold.RealizedPnL += a.RealizedPnL
		if a.UpdatedAt.After(gold.UpdatedAt) {
			gold.UpdatedAt = a.UpdatedAt
		}
		gold.Accounts = append(gold.Accounts, a)
	}
	if err := rows.Err(); err != nil {
		return PortfolioGold{}, err
	}
	if gold.QuantityGrams > 0 {
		gold.AverageCost = gold.TotalCost / gold.QuantityGrams
	}
	return gold, nil
}

type goldQuerier interface {
	GetContext(context.Context, any, string, ...any) error
}

func getGoldTransactionTx(ctx context.Context, q goldQuerier, userID, portfolioID, id string) (PortfolioGoldTransaction, error) {
	var item PortfolioGoldTransaction
	err := q.GetContext(ctx, &item, goldTransactionSelect+` WHERE pt.transaction_id=$1 AND pt.portfolio_id=$2 AND p.user_id=$3 AND a.symbol=$4`, id, portfolioID, userID, goldAssetSymbol)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrGoldTransactionNotFound
	}
	return item, err
}

const goldTransactionSelect = `SELECT pt.transaction_id::text AS transaction_id, pt.portfolio_id::text AS portfolio_id, pt.account_id::text AS account_id, pa.name AS account_name, pt.asset_id::text AS asset_id, pt.transaction_type AS transaction_type, pt.transaction_date AS transaction_date, pt.quantity::float8 AS quantity_grams, pt.price::float8 AS price, pt.gross_amount::float8 AS gross_amount, pt.cost_amount::float8 AS cost_amount, pt.fee_amount::float8 AS fee_amount, pt.tax_amount::float8 AS tax_amount, pt.net_amount::float8 AS net_amount, (CASE WHEN pt.transaction_type='sell' THEN pt.net_amount-pt.cost_amount ELSE 0 END)::float8 AS realized_pnl, pt.currency_code AS currency_code, COALESCE(pt.notes,'') AS notes, pt.created_at AS created_at, pt.updated_at AS updated_at FROM portfolio_transactions pt JOIN portfolios p ON p.portfolio_id=pt.portfolio_id JOIN portfolio_accounts pa ON pa.account_id=pt.account_id JOIN assets a ON a.asset_id=pt.asset_id`

func ensureGoldAssetTx(ctx context.Context, tx *sqlx.Tx) (string, error) {
	var id string
	err := tx.GetContext(ctx, &id, `INSERT INTO assets(asset_class_id,symbol,name,currency_code,pricing_method,source,provider_symbol) SELECT asset_class_id,$1,'Gold','IDR','api','yahoo','GC=F' FROM asset_classes WHERE code='commodity' ON CONFLICT(asset_class_id,symbol) DO UPDATE SET status=TRUE,updated_at=NOW() RETURNING asset_id::text`, goldAssetSymbol)
	return id, err
}
func ensureGoldAccountTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID string) error {
	var ok bool
	err := tx.GetContext(ctx, &ok, `SELECT EXISTS(SELECT 1 FROM portfolio_accounts WHERE portfolio_id=$1 AND account_id=$2)`, portfolioID, accountID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGoldAccountNotFound
	}
	return nil
}

func resolveGoldAccountTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID, accountName string) (string, error) {
	if accountID != "" {
		if err := ensureGoldAccountTx(ctx, tx, portfolioID, accountID); err != nil {
			return "", err
		}
		return accountID, nil
	}
	var resolvedID string
	err := tx.GetContext(ctx, &resolvedID, `
		INSERT INTO portfolio_accounts (portfolio_id, name, account_type, currency_code)
		VALUES ($1, $2, 'manual', 'IDR')
		ON CONFLICT (portfolio_id, name) DO UPDATE SET updated_at=NOW()
		RETURNING account_id::text
	`, portfolioID, accountName)
	return resolvedID, err
}

func rebuildGoldHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID, assetID string) error {
	rows, err := tx.QueryxContext(ctx, `SELECT transaction_id::text,transaction_type,quantity::float8,gross_amount::float8,fee_amount::float8,tax_amount::float8 FROM portfolio_transactions WHERE portfolio_id=$1 AND account_id=$2 AND asset_id=$3 ORDER BY transaction_date,created_at,transaction_id`, portfolioID, accountID, assetID)
	if err != nil {
		return err
	}
	type replay struct {
		ID       string  `db:"transaction_id"`
		Kind     string  `db:"transaction_type"`
		Quantity float64 `db:"quantity"`
		Gross    float64 `db:"gross_amount"`
		Fee      float64 `db:"fee_amount"`
		Tax      float64 `db:"tax_amount"`
	}
	items := []replay{}
	for rows.Next() {
		var v replay
		if err := rows.StructScan(&v); err != nil {
			rows.Close()
			return err
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	replayItems := make([]goldReplayTransaction, 0, len(items))
	for _, v := range items {
		replayItems = append(replayItems, goldReplayTransaction{Kind: v.Kind, Quantity: v.Quantity, Gross: v.Gross, Fee: v.Fee, Tax: v.Tax})
	}
	result, err := calculateGoldMovingAverage(replayItems)
	if err != nil {
		return err
	}
	for i, v := range items {
		if _, err := tx.ExecContext(ctx, `UPDATE portfolio_transactions SET cost_amount=$1,net_amount=$2 WHERE transaction_id=$3`, result.Transactions[i].CostAmount, result.Transactions[i].NetAmount, v.ID); err != nil {
			return err
		}
	}
	quantity, totalCost := result.Quantity, result.TotalCost
	if quantity == 0 {
		return deleteHoldingTx(ctx, tx, portfolioID, accountID, assetID)
	}
	return upsertHoldingTx(ctx, tx, portfolioID, accountID, assetID, quantity, result.AverageCost, totalCost)
}

type goldReplayTransaction struct {
	Kind                      string `db:"kind"`
	Quantity, Gross, Fee, Tax float64
}
type goldReplayAmount struct{ CostAmount, NetAmount, RealizedPnL float64 }
type goldReplayResult struct {
	Quantity, TotalCost, AverageCost, RealizedPnL float64
	Transactions                                  []goldReplayAmount `db:"transactions"`
}

func calculateGoldMovingAverage(items []goldReplayTransaction) (goldReplayResult, error) {
	result := goldReplayResult{Transactions: make([]goldReplayAmount, 0, len(items))}
	for _, item := range items {
		amount := goldReplayAmount{}
		switch item.Kind {
		case GoldTransactionBuy:
			amount.CostAmount = item.Gross + item.Fee + item.Tax
			amount.NetAmount = amount.CostAmount
			result.Quantity += item.Quantity
			result.TotalCost += amount.CostAmount
		case GoldTransactionSell:
			if item.Quantity > result.Quantity+1e-8 {
				return goldReplayResult{}, ErrGoldHoldingQuantity
			}
			if result.Quantity > 0 {
				amount.CostAmount = item.Quantity * (result.TotalCost / result.Quantity)
			}
			amount.NetAmount = item.Gross - item.Fee - item.Tax
			if amount.NetAmount < 0 {
				return goldReplayResult{}, ErrGoldHoldingQuantity
			}
			amount.RealizedPnL = amount.NetAmount - amount.CostAmount
			result.Quantity -= item.Quantity
			result.TotalCost -= amount.CostAmount
			result.RealizedPnL += amount.RealizedPnL
			if result.Quantity < 1e-8 {
				result.Quantity, result.TotalCost = 0, 0
			}
		}
		result.Transactions = append(result.Transactions, amount)
	}
	if result.Quantity > 0 {
		result.AverageCost = result.TotalCost / result.Quantity
	}
	return result, nil
}

type goldHoldingRow struct {
	PortfolioGoldAccount
	AssetID string `db:"asset_id"`
	Symbol  string `db:"symbol"`
	Name    string `db:"name"`
}
