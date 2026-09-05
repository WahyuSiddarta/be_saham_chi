package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

func (r *Repository) CreateCashTransaction(ctx context.Context, userID string, portfolioID string, input CashTransactionCommand, now time.Time) (PortfolioCashTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioCashTransaction, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		assetID, err := r.ensureCashAssetTx(ctx, tx, userID, now)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		accountID, err := r.resolveCashAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		transactionDate := input.TransactionDate
		if transactionDate.IsZero() {
			transactionDate = now
		}

		var cashTx PortfolioCashTransaction
		err = tx.GetContext(ctx, &cashTx, `
		INSERT INTO portfolio_transactions (
			portfolio_id,
			account_id,
			asset_id,
			transaction_type,
			transaction_date,
			quantity,
			price,
			gross_amount,
			cost_amount,
			net_amount,
			currency_code,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, 1, $6, $7, $6, 'IDR', $8)
		RETURNING transaction_id::text AS transaction_id, portfolio_id::text AS portfolio_id, account_id::text AS account_id, (SELECT name FROM portfolio_accounts WHERE account_id = portfolio_transactions.account_id) AS account_name, asset_id::text AS asset_id, transaction_type AS transaction_type, transaction_date AS transaction_date, net_amount::float8 AS amount, cost_amount::float8 AS cost_amount, currency_code AS currency_code, COALESCE(notes, '') AS notes, created_at AS created_at, updated_at AS updated_at `, portfolioID, accountID, assetID, input.TransactionType, transactionDate, input.Amount, input.CostAmount, input.Notes)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		if err := r.rebuildCashHoldingTx(ctx, tx, portfolioID, accountID, assetID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		if err := r.refreshCashSnapshotTx(ctx, tx, portfolioID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		return cashTx, nil
	})
}

func (r *Repository) ListCashTransactions(ctx context.Context, userID string, portfolioID string) ([]PortfolioCashTransaction, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryxContext(ctx, cashTransactionSelectSQL()+`
		WHERE pt.portfolio_id = $1
			AND p.user_id = $2
			AND ac.code = 'cash'
		ORDER BY pt.transaction_date DESC, pt.created_at DESC
	`, portfolioID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCashTransactions(rows)
}

func (r *Repository) GetCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) (PortfolioCashTransaction, error) {
	var cashTx PortfolioCashTransaction
	err := r.db.GetContext(ctx, &cashTx, cashTransactionSelectSQL()+`
		WHERE pt.transaction_id = $1
			AND pt.portfolio_id = $2
			AND p.user_id = $3
			AND ac.code = 'cash'
	`, transactionID, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PortfolioCashTransaction{}, ErrCashTransactionNotFound
		}
		return PortfolioCashTransaction{}, err
	}

	return cashTx, nil
}

func (r *Repository) UpdateCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string, input CashTransactionCommand, now time.Time) (PortfolioCashTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioCashTransaction, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		oldTx, err := r.getCashTransactionTx(ctx, tx, userID, portfolioID, transactionID)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		accountID, err := r.resolveCashAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		transactionDate := input.TransactionDate
		if transactionDate.IsZero() {
			transactionDate = oldTx.TransactionDate
		}

		var cashTx PortfolioCashTransaction
		err = tx.GetContext(ctx, &cashTx, `
		UPDATE portfolio_transactions
		SET account_id = $1,
			transaction_type = $2,
			transaction_date = $3,
			quantity = $4,
			price = 1,
			gross_amount = $4,
			cost_amount = $5,
			net_amount = $4,
			notes = $6,
			updated_at = NOW()
		WHERE transaction_id = $7
			AND portfolio_id = $8
		RETURNING transaction_id::text AS transaction_id, portfolio_id::text AS portfolio_id, account_id::text AS account_id, (SELECT name FROM portfolio_accounts WHERE account_id = portfolio_transactions.account_id) AS account_name, asset_id::text AS asset_id, transaction_type AS transaction_type, transaction_date AS transaction_date, net_amount::float8 AS amount, cost_amount::float8 AS cost_amount, currency_code AS currency_code, COALESCE(notes, '') AS notes, created_at AS created_at, updated_at AS updated_at `, accountID, input.TransactionType, transactionDate, input.Amount, input.CostAmount, input.Notes, transactionID, portfolioID)
		if err != nil {
			return PortfolioCashTransaction{}, err
		}
		if err := r.rebuildCashHoldingTx(ctx, tx, portfolioID, oldTx.AccountID, oldTx.AssetID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		if accountID != oldTx.AccountID {
			if err := r.rebuildCashHoldingTx(ctx, tx, portfolioID, accountID, cashTx.AssetID); err != nil {
				return PortfolioCashTransaction{}, err
			}
		}
		if err := r.refreshCashSnapshotTx(ctx, tx, portfolioID); err != nil {
			return PortfolioCashTransaction{}, err
		}
		return cashTx, nil
	})
}

func (r *Repository) DeleteCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) error {
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		oldTx, err := r.getCashTransactionTx(ctx, tx, userID, portfolioID, transactionID)
		if err != nil {
			return err
		}
		tag, err := tx.ExecContext(ctx, `
		DELETE FROM portfolio_transactions
		WHERE transaction_id = $1
			AND portfolio_id = $2
		`, transactionID, portfolioID)
		if err != nil {
			return err
		}
		if affected, err := tag.RowsAffected(); err != nil {
			return err
		} else if affected == 0 {
			return ErrCashTransactionNotFound
		}
		if err := r.rebuildCashHoldingTx(ctx, tx, portfolioID, oldTx.AccountID, oldTx.AssetID); err != nil {
			return err
		}
		return r.refreshCashSnapshotTx(ctx, tx, portfolioID)
	})
}

func (r *Repository) GetCash(ctx context.Context, userID string, portfolioID string) (PortfolioCash, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return PortfolioCash{}, err
	}

	cash := PortfolioCash{
		PortfolioID:  portfolioID,
		Symbol:       "IDR-CASH",
		Name:         "Indonesian Rupiah Cash",
		CurrencyCode: PortfolioCurrencyIDR,
		Accounts:     []PortfolioCashAccount{},
	}

	rows, err := r.db.QueryxContext(ctx, `
		SELECT ph.asset_id::text AS asset_id, a.symbol AS symbol, a.name AS name, pa.account_id::text AS account_id, pa.name AS account_name, ph.quantity::float8 AS quantity, ph.total_cost::float8 AS total_cost, ph.updated_at AS updated_at FROM portfolio_holdings ph
		JOIN assets a ON a.asset_id = ph.asset_id
		JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
		JOIN portfolio_accounts pa ON pa.account_id = ph.account_id
		WHERE ph.portfolio_id = $1
			AND ac.code = 'cash'
		ORDER BY pa.name ASC
	`, portfolioID)
	if err != nil {
		return PortfolioCash{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var row cashHoldingRow
		err := rows.StructScan(&row)
		account := row.PortfolioCashAccount
		cash.AssetID, cash.Symbol, cash.Name = row.AssetID, row.Symbol, row.Name
		if err != nil {
			return PortfolioCash{}, err
		}
		cash.TotalCash += account.Quantity
		cash.TotalCost += account.TotalCost
		account.RecalculatePnL()
		if account.UpdatedAt.After(cash.UpdatedAt) {
			cash.UpdatedAt = account.UpdatedAt
		}
		cash.Accounts = append(cash.Accounts, account)
	}
	if err := rows.Err(); err != nil {
		return PortfolioCash{}, err
	}
	cash.RecalculatePnL()

	return cash, nil
}

func (r *Repository) ListCashSnapshots(ctx context.Context, userID string, portfolioID string, from time.Time, to time.Time) ([]PortfolioCashSnapshot, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryxContext(ctx, `
		SELECT
			pacs.portfolio_id::text,
			pacs.asset_class_id,
			ac.code AS asset_class_code,
			pacs.snapshot_date,
			pacs.total_cost::float8,
			pacs.market_value::float8,
			pacs.unrealized_pnl::float8,
			pacs.realized_pnl::float8,
			pacs.total_pnl::float8,
			pacs.total_pnl_percent::float8,
			pacs.currency_code,
			pacs.created_at,
			pacs.updated_at
		FROM portfolio_asset_class_snapshots pacs
		JOIN asset_classes ac ON ac.asset_class_id = pacs.asset_class_id
		WHERE pacs.portfolio_id = $1
			AND ac.code = 'cash'
			AND pacs.snapshot_date >= $2
			AND pacs.snapshot_date <= $3
		ORDER BY pacs.snapshot_date ASC
	`, portfolioID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]PortfolioCashSnapshot, 0)
	for rows.Next() {
		var snapshot PortfolioCashSnapshot
		if err := rows.StructScan(&snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (r *Repository) ensureCashAssetTx(ctx context.Context, tx *sqlx.Tx, userID string, now time.Time) (string, error) {
	var assetID string
	err := tx.GetContext(ctx, &assetID, `
		WITH cash_class AS (
			SELECT asset_class_id
			FROM asset_classes
			WHERE code = 'cash'
		),
		upserted AS (
			INSERT INTO assets (asset_class_id, symbol, name, currency_code, pricing_method, source)
			SELECT asset_class_id, 'IDR-CASH', 'Indonesian Rupiah Cash', 'IDR', 'manual', 'manual'
			FROM cash_class
			ON CONFLICT (asset_class_id, symbol) DO UPDATE
			SET name = EXCLUDED.name,
				currency_code = EXCLUDED.currency_code,
				pricing_method = EXCLUDED.pricing_method,
				source = EXCLUDED.source,
				status = TRUE,
				updated_at = NOW()
			RETURNING asset_id
		)
		SELECT asset_id::text
		FROM upserted
	`)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO asset_valuations (asset_id, valuation_date, price, currency_code, source, notes, created_by)
		VALUES ($1, $2, 1, 'IDR', 'manual', 'Cash uses manual fixed value', $3)
		ON CONFLICT (asset_id, valuation_date, source) DO UPDATE
		SET price = EXCLUDED.price,
			notes = EXCLUDED.notes,
			created_by = EXCLUDED.created_by
	`, assetID, now.UTC(), userID)
	if err != nil {
		return "", err
	}

	return assetID, nil
}

func (r *Repository) resolveCashAccountTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, accountID string, accountName string) (string, error) {
	if accountID != "" {
		var resolvedID string
		err := tx.GetContext(ctx, &resolvedID, `
			SELECT account_id::text
			FROM portfolio_accounts
			WHERE account_id = $1
				AND portfolio_id = $2
		`, accountID, portfolioID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrCashAccountNotFound
			}
			return "", err
		}
		return resolvedID, nil
	}

	var resolvedID string
	err := tx.GetContext(ctx, &resolvedID, `
		INSERT INTO portfolio_accounts (portfolio_id, name, account_type, currency_code)
		VALUES ($1, $2, 'bank', 'IDR')
		ON CONFLICT (portfolio_id, name) DO UPDATE
		SET updated_at = NOW()
		RETURNING account_id::text
	`, portfolioID, accountName)
	if err != nil {
		return "", err
	}
	return resolvedID, nil
}

func (r *Repository) rebuildCashHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, accountID string, assetID string) error {
	var totals holdingTotals
	err := tx.GetContext(ctx, &totals, `
		SELECT COALESCE(SUM(
			CASE
				WHEN transaction_type IN ('deposit', 'dividend', 'coupon', 'interest', 'maturity') THEN net_amount
				WHEN transaction_type IN ('withdrawal', 'fee', 'tax') THEN -net_amount
				ELSE 0
			END
		), 0)::float8 AS quantity, COALESCE(SUM(
			CASE
				WHEN transaction_type IN ('deposit', 'maturity') THEN cost_amount
				WHEN transaction_type = 'withdrawal' THEN -cost_amount
				ELSE 0
			END
		), 0)::float8 AS total_cost FROM portfolio_transactions
		WHERE portfolio_id = $1
			AND account_id = $2
			AND asset_id = $3
	`, portfolioID, accountID, assetID)
	balance, totalCost := totals.Quantity, totals.TotalCost
	if err != nil {
		return err
	}

	if balance <= 0 {
		return deleteHoldingTx(ctx, tx, portfolioID, accountID, assetID)
	}
	return upsertHoldingTx(ctx, tx, portfolioID, accountID, assetID, balance, totalCost/balance, totalCost)
}

func (r *Repository) refreshCashSnapshotTx(ctx context.Context, tx *sqlx.Tx, portfolioID string) error {
	_, err := tx.ExecContext(ctx, `
		WITH cash_class AS (
			SELECT asset_class_id
			FROM asset_classes
			WHERE code = 'cash'
		),
		cash_totals AS (
			SELECT
				cc.asset_class_id,
				COALESCE(SUM(ph.total_cost), 0) AS total_cost,
				COALESCE(SUM(ph.quantity), 0) AS total_cash
			FROM cash_class cc
			LEFT JOIN assets a ON a.asset_class_id = cc.asset_class_id
			LEFT JOIN portfolio_holdings ph ON ph.asset_id = a.asset_id
				AND ph.portfolio_id = $1
			GROUP BY cc.asset_class_id
		)
		INSERT INTO portfolio_asset_class_snapshots (
			portfolio_id,
			asset_class_id,
			snapshot_date,
			total_cost,
			market_value,
			unrealized_pnl,
			realized_pnl,
			total_pnl,
			total_pnl_percent,
			currency_code,
			updated_at
		)
		SELECT
			$1,
			asset_class_id,
			(NOW() AT TIME ZONE 'UTC')::date,
			total_cost,
			total_cash,
			total_cash - total_cost,
			0,
			total_cash - total_cost,
			CASE WHEN total_cost = 0 THEN 0 ELSE ((total_cash - total_cost) / total_cost) * 100 END,
			'IDR',
			NOW()
		FROM cash_totals
		ON CONFLICT (portfolio_id, asset_class_id, snapshot_date) DO UPDATE
		SET total_cost = EXCLUDED.total_cost,
			market_value = EXCLUDED.market_value,
			unrealized_pnl = EXCLUDED.unrealized_pnl,
			realized_pnl = EXCLUDED.realized_pnl,
			total_pnl = EXCLUDED.total_pnl,
			total_pnl_percent = EXCLUDED.total_pnl_percent,
			updated_at = NOW()
	`, portfolioID)
	return err
}

func (r *Repository) getCashTransactionTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string, transactionID string) (PortfolioCashTransaction, error) {
	var cashTx PortfolioCashTransaction
	err := tx.GetContext(ctx, &cashTx, cashTransactionSelectSQL()+`
		WHERE pt.transaction_id = $1
			AND pt.portfolio_id = $2
			AND p.user_id = $3
			AND ac.code = 'cash'
	`, transactionID, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PortfolioCashTransaction{}, ErrCashTransactionNotFound
		}
		return PortfolioCashTransaction{}, err
	}
	return cashTx, nil
}

func cashTransactionSelectSQL() string {
	return `
		SELECT pt.transaction_id::text AS transaction_id, pt.portfolio_id::text AS portfolio_id, pt.account_id::text AS account_id, pa.name AS account_name, pt.asset_id::text AS asset_id, pt.transaction_type AS transaction_type, pt.transaction_date AS transaction_date, pt.net_amount::float8 AS amount, pt.cost_amount::float8 AS cost_amount, pt.currency_code AS currency_code, COALESCE(pt.notes, '') AS notes, pt.created_at AS created_at, pt.updated_at AS updated_at FROM portfolio_transactions pt
		JOIN portfolios p ON p.portfolio_id = pt.portfolio_id
		JOIN portfolio_accounts pa ON pa.account_id = pt.account_id
		JOIN assets a ON a.asset_id = pt.asset_id
		JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
	`
}

func scanCashTransactions(rows *sqlx.Rows) ([]PortfolioCashTransaction, error) {
	transactions := make([]PortfolioCashTransaction, 0)
	for rows.Next() {
		var cashTx PortfolioCashTransaction
		if err := rows.StructScan(&cashTx); err != nil {
			return nil, err
		}
		transactions = append(transactions, cashTx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}

type cashHoldingRow struct {
	PortfolioCashAccount
	AssetID string `db:"asset_id"`
	Symbol  string `db:"symbol"`
	Name    string `db:"name"`
}
