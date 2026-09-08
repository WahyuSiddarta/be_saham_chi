package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureStockBetaTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stock_betas (
  ticker TEXT NOT NULL REFERENCES stocks(ticker)
    ON UPDATE CASCADE
    ON DELETE CASCADE,
  value NUMERIC NOT NULL,
  period TEXT NOT NULL,
  interval TEXT NOT NULL,
  source TEXT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (ticker, period, interval, source),
  CONSTRAINT stock_betas_period_check CHECK (period = '5y'),
  CONSTRAINT stock_betas_interval_check CHECK (interval = '1mo'),
  CONSTRAINT stock_betas_source_check CHECK (source = 'yahoo_finance')
);

CREATE INDEX IF NOT EXISTS stock_betas_fetched_at_idx
  ON stock_betas (fetched_at DESC);
`)
	return err
}
