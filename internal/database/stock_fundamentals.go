package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureStockFundamentalsTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stock_fundamentals (
  ticker TEXT PRIMARY KEY REFERENCES stocks(ticker)
    ON UPDATE CASCADE
    ON DELETE CASCADE,
  payload JSONB NOT NULL DEFAULT '{}'::JSONB,
  scraped_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE stock_fundamentals
  ADD COLUMN IF NOT EXISTS payload JSONB;

DO $migration$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'stock_fundamentals'
      AND column_name = 'current_valuation'
  ) THEN
    EXECUTE $sql$
      UPDATE stock_fundamentals
      SET payload = jsonb_build_object(
        'currentValuation', current_valuation,
        'perShare', per_share,
        'solvency', solvency,
        'managementEffectiveness', management_effectiveness,
        'profitability', profitability,
        'growth', growth,
        'dividend', dividend,
        'marketRank', market_rank,
        'incomeStatement', income_statement,
        'balanceSheet', balance_sheet,
        'cashFlowStatement', cash_flow_statement,
        'dividendHistory', dividend_history,
        'marketOverview', market_overview
      )
      WHERE payload IS NULL
    $sql$;
  END IF;
END;
$migration$;

UPDATE stock_fundamentals SET payload = '{}'::JSONB WHERE payload IS NULL;
ALTER TABLE stock_fundamentals
  ALTER COLUMN payload SET DEFAULT '{}'::JSONB,
  ALTER COLUMN payload SET NOT NULL,
  DROP CONSTRAINT IF EXISTS stock_fundamentals_source_check,
  DROP CONSTRAINT IF EXISTS stock_fundamentals_dividend_history_check,
  DROP COLUMN IF EXISTS source,
  DROP COLUMN IF EXISTS current_valuation,
  DROP COLUMN IF EXISTS per_share,
  DROP COLUMN IF EXISTS solvency,
  DROP COLUMN IF EXISTS management_effectiveness,
  DROP COLUMN IF EXISTS profitability,
  DROP COLUMN IF EXISTS growth,
  DROP COLUMN IF EXISTS dividend,
  DROP COLUMN IF EXISTS market_rank,
  DROP COLUMN IF EXISTS income_statement,
  DROP COLUMN IF EXISTS balance_sheet,
  DROP COLUMN IF EXISTS cash_flow_statement,
  DROP COLUMN IF EXISTS dividend_history,
  DROP COLUMN IF EXISTS market_overview;

CREATE INDEX IF NOT EXISTS stock_fundamentals_scraped_at_idx
  ON stock_fundamentals (scraped_at DESC);
`)
	return err
}
