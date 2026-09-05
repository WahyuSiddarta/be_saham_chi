package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrPortfolioNotFound       = errors.New("portfolio not found")
	ErrMainPortfolioProtected  = errors.New("main portfolio cannot be deleted")
	ErrCashAccountNotFound     = errors.New("cash account not found")
	ErrCashTransactionNotFound = errors.New("cash transaction not found")
	ErrBondAssetNotFound       = errors.New("bond asset not found")
	ErrBondAccountNotFound     = errors.New("bond account not found")
	ErrBondHoldingNotFound     = errors.New("bond holding not found")
	ErrBondHoldingQuantity     = errors.New("bond holding quantity is insufficient")
	ErrBondTransactionNotFound = errors.New("bond transaction not found")
	ErrGoldAccountNotFound     = errors.New("gold account not found")
	ErrGoldHoldingQuantity     = errors.New("gold holding quantity is insufficient")
	ErrGoldTransactionNotFound = errors.New("gold transaction not found")
)

func (r *Repository) GetByID(ctx context.Context, userID string, portfolioID string) (Portfolio, error) {
	var portfolio Portfolio
	err := r.db.GetContext(ctx, &portfolio, `
		SELECT portfolio_id::text AS portfolio_id, user_id::text AS user_id, name AS name, base_currency_code AS base_currency_code, is_main AS is_main, created_at AS created_at, updated_at AS updated_at FROM portfolios
		WHERE portfolio_id = $1
			AND user_id = $2
	`, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Portfolio{}, ErrPortfolioNotFound
		}
		return Portfolio{}, err
	}

	return portfolio, nil
}

func (r *Repository) ListPortfolio(ctx context.Context, userID string) ([]Portfolio, error) {
	items := make([]Portfolio, 0)
	err := r.db.SelectContext(ctx, &items, `
SELECT portfolio_id::text, user_id::text, name, base_currency_code, is_main, created_at, updated_at
FROM portfolios WHERE user_id=$1 ORDER BY is_main DESC, created_at ASC
`, userID)
	return items, err
}

func (r *Repository) CreatePortfolio(ctx context.Context, userID string, input PortfolioCommand) (Portfolio, error) {
	var portfolio Portfolio
	err := r.db.GetContext(ctx, &portfolio, `
		INSERT INTO portfolios (user_id, name, base_currency_code)
		VALUES ($1, $2, $3)
		RETURNING portfolio_id::text AS portfolio_id, user_id::text AS user_id, name AS name, base_currency_code AS base_currency_code, is_main AS is_main, created_at AS created_at, updated_at AS updated_at `, userID, input.Name, PortfolioCurrencyIDR)
	if err != nil {
		return Portfolio{}, err
	}

	return portfolio, nil
}

func (r *Repository) UpdatePortfolio(ctx context.Context, userID string, portfolioID string, input PortfolioCommand) (Portfolio, error) {
	var portfolio Portfolio
	err := r.db.GetContext(ctx, &portfolio, `
		UPDATE portfolios
		SET name = $1,
			base_currency_code = $2,
			updated_at = NOW()
		WHERE portfolio_id = $3
			AND user_id = $4
		RETURNING portfolio_id::text AS portfolio_id, user_id::text AS user_id, name AS name, base_currency_code AS base_currency_code, is_main AS is_main, created_at AS created_at, updated_at AS updated_at `, input.Name, PortfolioCurrencyIDR, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Portfolio{}, ErrPortfolioNotFound
		}
		return Portfolio{}, err
	}

	return portfolio, nil
}

func (r *Repository) DeleteAndMove(ctx context.Context, userID string, portfolioID string, targetPortfolioID string) error {
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		source, err := r.getPortfolioTx(ctx, tx, userID, portfolioID)
		if err != nil {
			return err
		}
		if source.IsMain {
			return ErrMainPortfolioProtected
		}
		if _, err := r.getPortfolioTx(ctx, tx, userID, targetPortfolioID); err != nil {
			return err
		}
		if err := r.movePortfolioAccountsTx(ctx, tx, portfolioID, targetPortfolioID); err != nil {
			return err
		}
		if err := r.movePortfolioHoldingsTx(ctx, tx, portfolioID, targetPortfolioID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
		UPDATE portfolio_transactions
		SET portfolio_id = $1,
			updated_at = NOW()
		WHERE portfolio_id = $2
		`, targetPortfolioID, portfolioID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
		DELETE FROM portfolio_asset_class_snapshots
		WHERE portfolio_id = $1
		`, portfolioID); err != nil {
			return err
		}
		tag, err := tx.ExecContext(ctx, `
		DELETE FROM portfolios
		WHERE portfolio_id = $1
			AND user_id = $2
			AND is_main = FALSE
		`, portfolioID, userID)
		if err != nil {
			return err
		}
		if affected, err := tag.RowsAffected(); err != nil {
			return err
		} else if affected == 0 {
			return ErrPortfolioNotFound
		}
		if err := r.refreshCashSnapshotTx(ctx, tx, targetPortfolioID); err != nil {
			return err
		}
		return r.refreshBondSnapshotTx(ctx, tx, targetPortfolioID)
	})
}

func (r *Repository) ensurePortfolioTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string) error {
	var exists bool
	err := tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1
			FROM portfolios
			WHERE portfolio_id = $1
				AND user_id = $2
		)
	`, portfolioID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrPortfolioNotFound
	}
	return nil
}

func (r *Repository) getPortfolioTx(ctx context.Context, tx *sqlx.Tx, userID string, portfolioID string) (Portfolio, error) {
	var portfolio Portfolio
	err := tx.GetContext(ctx, &portfolio, `
		SELECT portfolio_id::text AS portfolio_id, user_id::text AS user_id, name AS name, base_currency_code AS base_currency_code, is_main AS is_main, created_at AS created_at, updated_at AS updated_at FROM portfolios
		WHERE portfolio_id = $1
			AND user_id = $2
	`, portfolioID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Portfolio{}, ErrPortfolioNotFound
		}
		return Portfolio{}, err
	}
	return portfolio, nil
}

func (r *Repository) movePortfolioAccountsTx(ctx context.Context, tx *sqlx.Tx, sourcePortfolioID string, targetPortfolioID string) error {
	_, err := tx.ExecContext(ctx, `
		WITH source_accounts AS (
			SELECT name, account_type, currency_code
			FROM portfolio_accounts
			WHERE portfolio_id = $1
		)
		INSERT INTO portfolio_accounts (portfolio_id, name, account_type, currency_code)
		SELECT $2, name, account_type, currency_code
		FROM source_accounts
		ON CONFLICT (portfolio_id, name) DO UPDATE
		SET updated_at = NOW()
	`, sourcePortfolioID, targetPortfolioID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		WITH account_map AS (
			SELECT
				source.account_id AS source_account_id,
				target.account_id AS target_account_id
			FROM portfolio_accounts source
			JOIN portfolio_accounts target ON target.portfolio_id = $2
				AND target.name = source.name
			WHERE source.portfolio_id = $1
		)
		UPDATE portfolio_transactions pt
		SET account_id = account_map.target_account_id,
			updated_at = NOW()
		FROM account_map
		WHERE pt.portfolio_id = $1
			AND pt.account_id = account_map.source_account_id
	`, sourcePortfolioID, targetPortfolioID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		WITH account_map AS (
			SELECT
				source.account_id AS source_account_id,
				target.account_id AS target_account_id
			FROM portfolio_accounts source
			JOIN portfolio_accounts target ON target.portfolio_id = $2
				AND target.name = source.name
			WHERE source.portfolio_id = $1
		)
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
			created_by,
			created_at
		)
		SELECT
			$2,
			account_map.target_account_id,
			phv.asset_id,
			phv.valuation_date,
			phv.price,
			phv.market_value,
			phv.currency_code,
			phv.source,
			phv.notes,
			phv.created_by,
			phv.created_at
		FROM portfolio_holding_valuations phv
		JOIN account_map ON account_map.source_account_id = phv.account_id
		WHERE phv.portfolio_id = $1
		ON CONFLICT (portfolio_id, account_id, asset_id, valuation_date, source) DO UPDATE
		SET price = EXCLUDED.price,
			market_value = EXCLUDED.market_value,
			notes = EXCLUDED.notes,
			created_by = EXCLUDED.created_by,
			created_at = EXCLUDED.created_at
	`, sourcePortfolioID, targetPortfolioID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM portfolio_holding_valuations
		WHERE portfolio_id = $1
	`, sourcePortfolioID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		WITH account_map AS (
			SELECT
				source.account_id AS source_account_id,
				target.account_id AS target_account_id
			FROM portfolio_accounts source
			JOIN portfolio_accounts target ON target.portfolio_id = $2
				AND target.name = source.name
			WHERE source.portfolio_id = $1
		)
		UPDATE portfolio_holdings ph
		SET account_id = account_map.target_account_id,
			updated_at = NOW()
		FROM account_map
		WHERE ph.portfolio_id = $1
			AND ph.account_id = account_map.source_account_id
	`, sourcePortfolioID, targetPortfolioID)
	return err
}

func (r *Repository) movePortfolioHoldingsTx(ctx context.Context, tx *sqlx.Tx, sourcePortfolioID string, targetPortfolioID string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO portfolio_holdings (
			portfolio_id,
			account_id,
			asset_id,
			quantity,
			average_cost,
			total_cost,
			currency_code,
			updated_at
		)
		SELECT
			$2,
			account_id,
			asset_id,
			quantity,
			average_cost,
			total_cost,
			currency_code,
			NOW()
		FROM portfolio_holdings
		WHERE portfolio_id = $1
			AND account_id IS NOT NULL
		ON CONFLICT (portfolio_id, account_id, asset_id)
			WHERE account_id IS NOT NULL
		DO UPDATE
		SET quantity = portfolio_holdings.quantity + EXCLUDED.quantity,
			total_cost = portfolio_holdings.total_cost + EXCLUDED.total_cost,
			average_cost = CASE
				WHEN portfolio_holdings.quantity + EXCLUDED.quantity = 0 THEN 0
				ELSE (portfolio_holdings.total_cost + EXCLUDED.total_cost) / (portfolio_holdings.quantity + EXCLUDED.quantity)
			END,
			updated_at = NOW()
	`, sourcePortfolioID, targetPortfolioID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO portfolio_holdings (
			portfolio_id,
			account_id,
			asset_id,
			quantity,
			average_cost,
			total_cost,
			currency_code,
			updated_at
		)
		SELECT
			$2,
			account_id,
			asset_id,
			quantity,
			average_cost,
			total_cost,
			currency_code,
			NOW()
		FROM portfolio_holdings
		WHERE portfolio_id = $1
			AND account_id IS NULL
		ON CONFLICT (portfolio_id, asset_id)
			WHERE account_id IS NULL
		DO UPDATE
		SET quantity = portfolio_holdings.quantity + EXCLUDED.quantity,
			total_cost = portfolio_holdings.total_cost + EXCLUDED.total_cost,
			average_cost = CASE
				WHEN portfolio_holdings.quantity + EXCLUDED.quantity = 0 THEN 0
				ELSE (portfolio_holdings.total_cost + EXCLUDED.total_cost) / (portfolio_holdings.quantity + EXCLUDED.quantity)
			END,
			updated_at = NOW()
	`, sourcePortfolioID, targetPortfolioID)
	return err
}
