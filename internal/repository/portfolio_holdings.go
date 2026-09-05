package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func deleteHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID, assetID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM portfolio_holdings
		WHERE portfolio_id = $1
			AND account_id = $2
			AND asset_id = $3
	`, portfolioID, accountID, assetID)
	return err
}

func upsertHoldingTx(ctx context.Context, tx *sqlx.Tx, portfolioID, accountID, assetID string, quantity, averageCost, totalCost float64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO portfolio_holdings (
			portfolio_id, account_id, asset_id, quantity, average_cost,
			total_cost, currency_code, updated_at
		)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric, 'IDR', NOW())
		ON CONFLICT (portfolio_id, account_id, asset_id)
			WHERE account_id IS NOT NULL
		DO UPDATE
		SET quantity = EXCLUDED.quantity,
			average_cost = EXCLUDED.average_cost,
			total_cost = EXCLUDED.total_cost,
			updated_at = NOW()
	`, portfolioID, accountID, assetID, quantity, averageCost, totalCost)
	return err
}

type holdingTotals struct {
	Quantity  float64 `db:"quantity"`
	TotalCost float64 `db:"total_cost"`
}
