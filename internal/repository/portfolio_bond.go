package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/helper"

	"github.com/jmoiron/sqlx"
)

func (r *Repository) CreateBond(ctx context.Context, userID string, portfolioID string, input BondCommand, now time.Time) (PortfolioBond, error) {
	assetID, err := withinTx(ctx, r.db, func(tx *sqlx.Tx) (string, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return "", err
		}
		assetID, err := r.ensureBondAssetTx(ctx, tx, input.BondAssetCommand)
		if err != nil {
			return "", err
		}
		if err := r.upsertBondTermTx(ctx, tx, assetID, input.BondAssetCommand); err != nil {
			return "", err
		}
		accountID, err := r.resolveBondAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return "", err
		}
		transactionDate := input.TransactionDate
		if transactionDate.IsZero() {
			transactionDate = now
		}
		price := 1.0
		if input.PrincipalAmount > 0 {
			price = input.CostAmount / input.PrincipalAmount
		}

		if _, err := tx.ExecContext(ctx, `
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
			accrued_coupon_amount,
			fee_amount,
			net_amount,
			currency_code,
			notes
		)
		VALUES ($1, $2, $3, 'buy', $4, $5, $6, $7, $7, $8, $9, $7 + $8 + $9, 'IDR', $10)
		`, portfolioID, accountID, assetID, transactionDate, input.PrincipalAmount, price, input.CostAmount, input.AccruedCouponAmount, input.FeeAmount, input.Notes); err != nil {
			return "", err
		}
		if err := r.rebuildBondHoldingTx(ctx, tx, portfolioID, accountID, assetID); err != nil {
			return "", err
		}
		if input.MarketValue > 0 {
			if err := r.upsertBondHoldingValuationTx(ctx, tx, userID, portfolioID, accountID, assetID, now, BondValuationCommand{
				MarketValue: input.MarketValue,
				Notes:       input.Notes,
			}); err != nil {
				return "", err
			}
		}
		if err := r.refreshBondSnapshotTx(ctx, tx, portfolioID); err != nil {
			return "", err
		}
		return assetID, nil
	})
	if err != nil {
		return PortfolioBond{}, err
	}
	return r.GetBond(ctx, userID, portfolioID, assetID)
}

func (r *Repository) ListBonds(ctx context.Context, userID string, portfolioID string) ([]PortfolioBond, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryxContext(ctx, bondHoldingSelectSQL()+`
		WHERE ph.portfolio_id = $1
			AND p.user_id = $2
			AND ac.code = 'bond'
		ORDER BY a.symbol ASC, pa.name ASC
	`, portfolioID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBondHoldings(rows)
}

func (r *Repository) GetBond(ctx context.Context, userID string, portfolioID string, assetID string) (PortfolioBond, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return PortfolioBond{}, err
	}

	rows, err := r.db.QueryxContext(ctx, bondHoldingSelectSQL()+`
		WHERE ph.portfolio_id = $1
			AND p.user_id = $2
			AND ph.asset_id = $3
			AND ac.code = 'bond'
		ORDER BY pa.name ASC
	`, portfolioID, userID, assetID)
	if err != nil {
		return PortfolioBond{}, err
	}
	defer rows.Close()

	bonds, err := scanBondHoldings(rows)
	if err != nil {
		return PortfolioBond{}, err
	}
	if len(bonds) == 0 {
		return PortfolioBond{}, ErrBondAssetNotFound
	}
	return bonds[0], nil
}

func (r *Repository) UpdateBond(ctx context.Context, userID string, portfolioID string, assetID string, input BondAssetCommand) (PortfolioBond, error) {
	err := withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return err
		}
		if err := r.ensureBondHoldingAssetTx(ctx, tx, userID, portfolioID, assetID); err != nil {
			return err
		}
		tag, err := tx.ExecContext(ctx, `
		UPDATE assets a
		SET symbol = $1,
			name = $2,
			currency_code = 'IDR',
			pricing_method = 'manual',
			source = 'manual',
			status = TRUE,
			updated_at = NOW()
		FROM asset_classes ac
		WHERE a.asset_id = $3
			AND a.asset_class_id = ac.asset_class_id
			AND ac.code = 'bond'
		`, input.Symbol, input.Name, assetID)
		if err != nil {
			return err
		}
		if affected, err := tag.RowsAffected(); err != nil {
			return err
		} else if affected == 0 {
			return ErrBondAssetNotFound
		}
		if err := r.upsertBondTermTx(ctx, tx, assetID, input); err != nil {
			return err
		}
		return r.refreshBondSnapshotTx(ctx, tx, portfolioID)
	})
	if err != nil {
		return PortfolioBond{}, err
	}
	return r.GetBond(ctx, userID, portfolioID, assetID)
}

func (r *Repository) AdjustBondValuation(ctx context.Context, userID string, portfolioID string, assetID string, input BondValuationCommand, now time.Time) (PortfolioBond, error) {
	err := withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return err
		}
		if err := r.upsertBondHoldingValuationTx(ctx, tx, userID, portfolioID, input.AccountID, assetID, now, input); err != nil {
			return err
		}
		return r.refreshBondSnapshotTx(ctx, tx, portfolioID)
	})
	if err != nil {
		return PortfolioBond{}, err
	}
	return r.GetBond(ctx, userID, portfolioID, assetID)
}

func (r *Repository) CreateBondTransaction(ctx context.Context, userID string, portfolioID string, input BondTransactionCommand, now time.Time) (PortfolioBondTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioBondTransaction, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.ensureBondAssetExistsTx(ctx, tx, input.AssetID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		accountID, err := r.resolveBondAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.normalizeBondTransactionAmountsTx(ctx, tx, portfolioID, accountID, &input); err != nil {
			return PortfolioBondTransaction{}, err
		}
		transactionDate := input.TransactionDate
		if transactionDate.IsZero() {
			transactionDate = now
		}

		var bondTx PortfolioBondTransaction
		err = tx.GetContext(ctx, &bondTx.TransactionID, `
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
			accrued_coupon_amount,
			fee_amount,
			tax_amount,
			net_amount,
			currency_code,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'IDR', $14)
		RETURNING transaction_id::text
		`, portfolioID, accountID, input.AssetID, input.TransactionType, transactionDate, input.PrincipalAmount, input.Price, input.GrossAmount, input.CostAmount, input.AccruedCouponAmount, input.FeeAmount, input.TaxAmount, input.NetAmount, input.Notes)
		if err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.rebuildBondHoldingTx(ctx, tx, portfolioID, accountID, input.AssetID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.refreshBondSnapshotTx(ctx, tx, portfolioID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.getBondTransactionByIDTx(ctx, tx, userID, portfolioID, bondTx.TransactionID, &bondTx); err != nil {
			return PortfolioBondTransaction{}, err
		}
		return bondTx, nil
	})
}

func (r *Repository) ListBondTransactions(ctx context.Context, userID string, portfolioID string) ([]PortfolioBondTransaction, error) {
	if _, err := r.GetByID(ctx, userID, portfolioID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryxContext(ctx, bondTransactionSelectSQL()+`
		WHERE pt.portfolio_id = $1
			AND p.user_id = $2
			AND ac.code = 'bond'
		ORDER BY pt.transaction_date DESC, pt.created_at DESC
	`, portfolioID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBondTransactions(rows)
}

func (r *Repository) GetBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) (PortfolioBondTransaction, error) {
	var bondTx PortfolioBondTransaction
	err := r.db.GetContext(ctx, &bondTx, bondTransactionSelectSQL()+`
		WHERE pt.transaction_id = $1
			AND pt.portfolio_id = $2
			AND p.user_id = $3
			AND ac.code = 'bond'
	`, transactionID, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PortfolioBondTransaction{}, ErrBondTransactionNotFound
		}
		return PortfolioBondTransaction{}, err
	}
	return bondTx, nil
}

func (r *Repository) UpdateBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string, input BondTransactionCommand, now time.Time) (PortfolioBondTransaction, error) {
	return withinTx(ctx, r.db, func(tx *sqlx.Tx) (PortfolioBondTransaction, error) {
		if err := r.ensurePortfolioTx(ctx, tx, userID, portfolioID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		oldTx, err := r.getBondTransactionTx(ctx, tx, userID, portfolioID, transactionID)
		if err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.ensureBondAssetExistsTx(ctx, tx, input.AssetID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		accountID, err := r.resolveBondAccountTx(ctx, tx, portfolioID, input.AccountID, input.AccountName)
		if err != nil {
			return PortfolioBondTransaction{}, err
		}
		if err := r.normalizeBondTransactionAmountsTx(ctx, tx, portfolioID, accountID, &input); err != nil {
			return PortfolioBondTransaction{}, err
		}
		transactionDate := input.TransactionDate
		if transactionDate.IsZero() {
			transactionDate = oldTx.TransactionDate
		}

		tag, err := tx.ExecContext(ctx, `
		UPDATE portfolio_transactions
		SET account_id = $1,
			asset_id = $2,
			transaction_type = $3,
			transaction_date = $4,
			quantity = $5,
			price = $6,
			gross_amount = $7,
			cost_amount = $8,
			accrued_coupon_amount = $9,
			fee_amount = $10,
			tax_amount = $11,
			net_amount = $12,
			notes = $13,
			updated_at = NOW()
		WHERE transaction_id = $14
			AND portfolio_id = $15
		`, accountID, input.AssetID, input.TransactionType, transactionDate, input.PrincipalAmount, input.Price, input.GrossAmount, input.CostAmount, input.AccruedCouponAmount, input.FeeAmount, input.TaxAmount, input.NetAmount, input.Notes, transactionID, portfolioID)
		if err != nil {
			return PortfolioBondTransaction{}, err
		}
		if affected, err := tag.RowsAffected(); err != nil {
			return PortfolioBondTransaction{}, err
		} else if affected == 0 {
			return PortfolioBondTransaction{}, ErrBondTransactionNotFound
		}
		if err := r.rebuildBondHoldingTx(ctx, tx, portfolioID, oldTx.AccountID, oldTx.AssetID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		if accountID != oldTx.AccountID || input.AssetID != oldTx.AssetID {
			if err := r.rebuildBondHoldingTx(ctx, tx, portfolioID, accountID, input.AssetID); err != nil {
				return PortfolioBondTransaction{}, err
			}
		}
		if err := r.refreshBondSnapshotTx(ctx, tx, portfolioID); err != nil {
			return PortfolioBondTransaction{}, err
		}
		var bondTx PortfolioBondTransaction
		if err := r.getBondTransactionByIDTx(ctx, tx, userID, portfolioID, transactionID, &bondTx); err != nil {
			return PortfolioBondTransaction{}, err
		}
		return bondTx, nil
	})
}

func (r *Repository) DeleteBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) error {
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		oldTx, err := r.getBondTransactionTx(ctx, tx, userID, portfolioID, transactionID)
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
			return ErrBondTransactionNotFound
		}
		if err := r.rebuildBondHoldingTx(ctx, tx, portfolioID, oldTx.AccountID, oldTx.AssetID); err != nil {
			return err
		}
		return r.refreshBondSnapshotTx(ctx, tx, portfolioID)
	})
}

func (r *Repository) ListBondSnapshots(ctx context.Context, userID string, portfolioID string, from time.Time, to time.Time) ([]PortfolioBondSnapshot, error) {
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
			AND ac.code = 'bond'
			AND pacs.snapshot_date >= $2
			AND pacs.snapshot_date <= $3
		ORDER BY pacs.snapshot_date ASC
	`, portfolioID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]PortfolioBondSnapshot, 0)
	for rows.Next() {
		var snapshot PortfolioBondSnapshot
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

func (r *Repository) ensureBondAssetTx(ctx context.Context, tx *sqlx.Tx, input BondAssetCommand) (string, error) {
	var assetID string
	err := tx.GetContext(ctx, &assetID, `
		WITH bond_class AS (
			SELECT asset_class_id
			FROM asset_classes
			WHERE code = 'bond'
		),
		upserted AS (
			INSERT INTO assets (asset_class_id, symbol, name, currency_code, pricing_method, source)
			SELECT asset_class_id, $1, $2, 'IDR', 'manual', 'manual'
			FROM bond_class
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
	`, input.Symbol, input.Name)
	if err != nil {
		return "", err
	}
	return assetID, nil
}

func (r *Repository) ensureBondAssetExistsTx(ctx context.Context, tx *sqlx.Tx, assetID string) error {
	var exists bool
	err := tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1
			FROM assets a
			JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
			WHERE a.asset_id = $1
				AND ac.code = 'bond'
				AND a.status = TRUE
		)
	`, assetID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrBondAssetNotFound
	}
	return nil
}

func (r *Repository) ensureBondHoldingAssetTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string, assetID string) error {
	var exists bool
	err := tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1
			FROM portfolio_holdings ph
			JOIN portfolios p ON p.portfolio_id = ph.portfolio_id
			JOIN assets a ON a.asset_id = ph.asset_id
			JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
			WHERE ph.portfolio_id = $1
				AND p.user_id = $2
				AND ph.asset_id = $3
				AND ac.code = 'bond'
		)
	`, portfolioID, userID, assetID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrBondAssetNotFound
	}
	return nil
}

func (r *Repository) upsertBondTermTx(ctx context.Context, tx *sqlx.Tx, assetID string, input BondAssetCommand) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO asset_terms (
			asset_id,
			issue_date,
			maturity_date,
			annual_rate,
			coupon_frequency,
			principal_value,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (asset_id) DO UPDATE
		SET issue_date = EXCLUDED.issue_date,
			maturity_date = EXCLUDED.maturity_date,
			annual_rate = EXCLUDED.annual_rate,
			coupon_frequency = EXCLUDED.coupon_frequency,
			principal_value = EXCLUDED.principal_value,
			updated_at = NOW()
	`, assetID, helper.DateOrNil(input.IssueDate), helper.DateOrNil(input.MaturityDate), helper.NumberOrNil(input.AnnualRate), helper.EmptyStringOrNil(input.CouponFrequency), helper.NumberOrNil(input.PrincipalValue))
	return err
}

func (r *Repository) resolveBondAccountTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, accountID string, accountName string) (string, error) {
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
				return "", ErrBondAccountNotFound
			}
			return "", err
		}
		return resolvedID, nil
	}

	var resolvedID string
	err := tx.GetContext(ctx, &resolvedID, `
		INSERT INTO portfolio_accounts (portfolio_id, name, account_type, currency_code)
		VALUES ($1, $2, 'manual', 'IDR')
		ON CONFLICT (portfolio_id, name) DO UPDATE
		SET updated_at = NOW()
		RETURNING account_id::text
	`, portfolioID, accountName)
	if err != nil {
		return "", err
	}
	return resolvedID, nil
}

func (r *Repository) upsertBondHoldingValuationTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string, accountID string, assetID string, now time.Time, input BondValuationCommand) error {
	var quantity float64
	err := tx.GetContext(ctx, &quantity, `
		SELECT ph.quantity::float8
		FROM portfolio_holdings ph
		JOIN assets a ON a.asset_id = ph.asset_id
		JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
		WHERE ph.portfolio_id = $1
			AND ph.account_id = $2
			AND ph.asset_id = $3
			AND ac.code = 'bond'
	`, portfolioID, accountID, assetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBondHoldingNotFound
		}
		return err
	}
	if quantity <= 0 {
		return ErrBondHoldingQuantity
	}

	valuationDate := input.ValuationDate
	if valuationDate.IsZero() {
		valuationDate = now
	}
	valuationDate = time.Date(valuationDate.UTC().Year(), valuationDate.UTC().Month(), valuationDate.UTC().Day(), 0, 0, 0, 0, time.UTC)

	price := input.Price
	marketValue := input.MarketValue
	if marketValue == 0 && price > 0 {
		marketValue = quantity * price
	}
	if price == 0 && marketValue > 0 {
		price = marketValue / quantity
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO portfolio_holding_valuations (
			portfolio_id,
			account_id,
			asset_id,
			valuation_date,
			price,
			market_value,
			currency_code,
			source,
			notes,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'IDR', 'manual', $7, $8)
		ON CONFLICT (portfolio_id, account_id, asset_id, valuation_date, source) DO UPDATE
		SET price = EXCLUDED.price,
			market_value = EXCLUDED.market_value,
			notes = EXCLUDED.notes,
			created_by = EXCLUDED.created_by,
			created_at = NOW()
	`, portfolioID, accountID, assetID, valuationDate, price, marketValue, input.Notes, userID)
	return err
}

func (r *Repository) normalizeBondTransactionAmountsTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, accountID string, input *BondTransactionCommand) error {
	if input.Price == 0 && input.PrincipalAmount > 0 {
		input.Price = 1
	}
	if input.GrossAmount == 0 && input.PrincipalAmount > 0 {
		input.GrossAmount = input.PrincipalAmount * input.Price
	}
	if input.TransactionType != BondTransactionBuy && input.TransactionType != BondTransactionSell {
		input.AccruedCouponAmount = 0
	}
	if input.NetAmount == 0 {
		input.NetAmount = input.GrossAmount + input.AccruedCouponAmount + input.FeeAmount + input.TaxAmount
	}
	if input.CostAmount == 0 && input.TransactionType == BondTransactionBuy {
		input.CostAmount = input.GrossAmount
	}
	if input.CostAmount == 0 && (input.TransactionType == BondTransactionSell || input.TransactionType == BondTransactionMaturity) {
		var averageCost float64
		err := tx.GetContext(ctx, &averageCost, `
			SELECT ph.average_cost::float8
			FROM portfolio_holdings ph
			WHERE ph.portfolio_id = $1
				AND ph.account_id = $2
				AND ph.asset_id = $3
		`, portfolioID, accountID, input.AssetID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrBondHoldingNotFound
			}
			return err
		}
		input.CostAmount = averageCost * input.PrincipalAmount
	}
	return nil
}

func (r *Repository) rebuildBondHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID string, accountID string, assetID string) error {
	var totals holdingTotals
	err := tx.GetContext(ctx, &totals, `
		SELECT COALESCE(SUM(
				CASE
					WHEN transaction_type = 'buy' THEN quantity
					WHEN transaction_type IN ('sell', 'maturity') THEN -quantity
					ELSE 0
				END
			), 0)::float8 AS quantity, COALESCE(SUM(
				CASE
					WHEN transaction_type = 'buy' THEN cost_amount
					WHEN transaction_type IN ('sell', 'maturity') THEN -cost_amount
					ELSE 0
				END
			), 0)::float8 AS total_cost FROM portfolio_transactions
		WHERE portfolio_id = $1
			AND account_id = $2
			AND asset_id = $3
	`, portfolioID, accountID, assetID)
	quantity, totalCost := totals.Quantity, totals.TotalCost
	if err != nil {
		return err
	}
	if quantity < 0 || totalCost < 0 {
		return ErrBondHoldingQuantity
	}
	if quantity == 0 {
		return deleteHoldingTx(ctx, tx, portfolioID, accountID, assetID)
	}
	return upsertHoldingTx(ctx, tx, portfolioID, accountID, assetID, quantity, totalCost/quantity, totalCost)
}

func (r *Repository) refreshBondSnapshotTx(ctx context.Context, tx *sqlx.Tx, portfolioID string) error {
	_, err := tx.ExecContext(ctx, `
		WITH bond_class AS (
			SELECT asset_class_id
			FROM asset_classes
			WHERE code = 'bond'
		),
		bond_totals AS (
			SELECT
				bc.asset_class_id,
				COALESCE(SUM(ph.total_cost), 0) AS total_cost,
				COALESCE(SUM(COALESCE(latest.market_value, ph.quantity)), 0) AS market_value
			FROM bond_class bc
			LEFT JOIN assets a ON a.asset_class_id = bc.asset_class_id
			LEFT JOIN portfolio_holdings ph ON ph.asset_id = a.asset_id
				AND ph.portfolio_id = $1
			LEFT JOIN LATERAL (
				SELECT phv.market_value
				FROM portfolio_holding_valuations phv
				WHERE phv.portfolio_id = ph.portfolio_id
					AND phv.account_id = ph.account_id
					AND phv.asset_id = ph.asset_id
					AND phv.source = 'manual'
				ORDER BY phv.valuation_date DESC, phv.created_at DESC
				LIMIT 1
			) latest ON TRUE
			GROUP BY bc.asset_class_id
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
			market_value,
			market_value - total_cost,
			0,
			market_value - total_cost,
			CASE WHEN total_cost = 0 THEN 0 ELSE ((market_value - total_cost) / total_cost) * 100 END,
			'IDR',
			NOW()
		FROM bond_totals
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

func (r *Repository) getBondTransactionTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string, transactionID string) (PortfolioBondTransaction, error) {
	var bondTx PortfolioBondTransaction
	if err := r.getBondTransactionByIDTx(ctx, tx, userID, portfolioID, transactionID, &bondTx); err != nil {
		return PortfolioBondTransaction{}, err
	}
	return bondTx, nil
}

func (r *Repository) getBondTransactionByIDTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string, transactionID string, bondTx *PortfolioBondTransaction) error {
	err := tx.GetContext(ctx, bondTx, bondTransactionSelectSQL()+`
		WHERE pt.transaction_id = $1
			AND pt.portfolio_id = $2
			AND p.user_id = $3
			AND ac.code = 'bond'
	`, transactionID, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBondTransactionNotFound
		}
		return err
	}
	return nil
}

func bondHoldingSelectSQL() string {
	return `
		SELECT ph.portfolio_id::text AS portfolio_id, ph.asset_id::text AS asset_id, a.symbol AS symbol, a.name AS name, ph.account_id::text AS account_id, pa.name AS account_name, ph.quantity::float8 AS principal_amount, ph.total_cost::float8 AS total_cost, COALESCE(latest.market_value, ph.quantity)::float8 AS market_value, ph.updated_at AS updated_at, COALESCE(at.issue_date::text, '') AS issue_date, COALESCE(at.maturity_date::text, '') AS maturity_date, COALESCE(at.annual_rate, 0)::float8 AS annual_rate, COALESCE(at.coupon_frequency, '') AS coupon_frequency, COALESCE(at.principal_value, 0)::float8 AS principal_value, COALESCE(latest.valuation_date::text, '') AS valuation_date, COALESCE(latest.price, 0)::float8 AS valuation_price, COALESCE(latest.market_value, 0)::float8 AS valuation_market, COALESCE(latest.source, '') AS valuation_source, COALESCE(latest.notes, '') AS valuation_notes FROM portfolio_holdings ph
		JOIN portfolios p ON p.portfolio_id = ph.portfolio_id
		JOIN portfolio_accounts pa ON pa.account_id = ph.account_id
		JOIN assets a ON a.asset_id = ph.asset_id
		JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
		LEFT JOIN asset_terms at ON at.asset_id = a.asset_id
		LEFT JOIN LATERAL (
			SELECT valuation_date, price, market_value, source, notes
			FROM portfolio_holding_valuations phv
			WHERE phv.portfolio_id = ph.portfolio_id
				AND phv.account_id = ph.account_id
				AND phv.asset_id = ph.asset_id
				AND phv.source = 'manual'
			ORDER BY phv.valuation_date DESC, phv.created_at DESC
			LIMIT 1
		) latest ON TRUE
	`
}

func scanBondHoldings(rows *sqlx.Rows) ([]PortfolioBond, error) {
	bondByAsset := make(map[string]*PortfolioBond)
	order := make([]string, 0)

	for rows.Next() {
		var row bondHoldingRow
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		account := row.PortfolioBondAccount
		bond := PortfolioBond{PortfolioID: row.PortfolioID, AssetID: row.AssetID, Symbol: row.Symbol, Name: row.Name,
			Term: PortfolioBondTerm{AnnualRate: row.AnnualRate, CouponFrequency: row.CouponFrequency, PrincipalValue: row.PrincipalValue}}
		issueDate, maturityDate := row.IssueDate, row.MaturityDate
		valuationDate, valuationPrice, valuationMarket := row.ValuationDate, row.ValuationPrice, row.ValuationMarket
		valuationSource, valuationNotes := row.ValuationSource, row.ValuationNotes
		bond.Term.IssueDate = helper.ParseDateString(issueDate)
		bond.Term.MaturityDate = helper.ParseDateString(maturityDate)
		bond.CurrencyCode = PortfolioCurrencyIDR
		account.RecalculatePnL()

		if valuationDate != "" {
			account.LatestValuation = &PortfolioBondValuation{
				ValuationDate: helper.ParseDateString(valuationDate),
				Price:         valuationPrice,
				MarketValue:   valuationMarket,
				Source:        valuationSource,
				Notes:         valuationNotes,
			}
		}

		current, ok := bondByAsset[bond.AssetID]
		if !ok {
			bond.Accounts = []PortfolioBondAccount{}
			bondByAsset[bond.AssetID] = &bond
			order = append(order, bond.AssetID)
			current = &bond
		}
		current.PrincipalAmount += account.PrincipalAmount
		current.TotalCost += account.TotalCost
		current.MarketValue += account.MarketValue
		if account.UpdatedAt.After(current.UpdatedAt) {
			current.UpdatedAt = account.UpdatedAt
		}
		if account.LatestValuation != nil {
			current.LatestValuation = account.LatestValuation
		}
		current.Accounts = append(current.Accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	bonds := make([]PortfolioBond, 0, len(order))
	for _, assetID := range order {
		bond := *bondByAsset[assetID]
		if len(bond.Accounts) == 1 {
			bond.AccountID = bond.Accounts[0].AccountID
			bond.AccountName = bond.Accounts[0].AccountName
		}
		bond.RecalculatePnL()
		bonds = append(bonds, bond)
	}
	return bonds, nil
}

func bondTransactionSelectSQL() string {
	return `
		SELECT pt.transaction_id::text AS transaction_id, pt.portfolio_id::text AS portfolio_id, pt.account_id::text AS account_id, pa.name AS account_name, pt.asset_id::text AS asset_id, a.symbol AS symbol, a.name AS name, pt.transaction_type AS transaction_type, pt.transaction_date AS transaction_date, pt.quantity::float8 AS principal_amount, pt.price::float8 AS price, pt.gross_amount::float8 AS gross_amount, pt.cost_amount::float8 AS cost_amount, pt.accrued_coupon_amount::float8 AS accrued_coupon_amount, pt.fee_amount::float8 AS fee_amount, pt.tax_amount::float8 AS tax_amount, pt.net_amount::float8 AS net_amount, pt.currency_code AS currency_code, COALESCE(pt.notes, '') AS notes, pt.created_at AS created_at, pt.updated_at AS updated_at FROM portfolio_transactions pt
		JOIN portfolios p ON p.portfolio_id = pt.portfolio_id
		JOIN portfolio_accounts pa ON pa.account_id = pt.account_id
		JOIN assets a ON a.asset_id = pt.asset_id
		JOIN asset_classes ac ON ac.asset_class_id = a.asset_class_id
	`
}

func scanBondTransactions(rows *sqlx.Rows) ([]PortfolioBondTransaction, error) {
	transactions := make([]PortfolioBondTransaction, 0)
	for rows.Next() {
		var bondTx PortfolioBondTransaction
		if err := rows.StructScan(&bondTx); err != nil {
			return nil, err
		}
		transactions = append(transactions, bondTx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}

type bondHoldingRow struct {
	PortfolioBondAccount
	PortfolioID     string  `db:"portfolio_id"`
	AssetID         string  `db:"asset_id"`
	Symbol          string  `db:"symbol"`
	Name            string  `db:"name"`
	IssueDate       string  `db:"issue_date"`
	MaturityDate    string  `db:"maturity_date"`
	AnnualRate      float64 `db:"annual_rate"`
	CouponFrequency string  `db:"coupon_frequency"`
	PrincipalValue  float64 `db:"principal_value"`
	ValuationDate   string  `db:"valuation_date"`
	ValuationPrice  float64 `db:"valuation_price"`
	ValuationMarket float64 `db:"valuation_market"`
	ValuationSource string  `db:"valuation_source"`
	ValuationNotes  string  `db:"valuation_notes"`
}
