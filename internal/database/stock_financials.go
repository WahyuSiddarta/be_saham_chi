package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureStockFinancialsTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stock_financials (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  ticker TEXT NOT NULL REFERENCES stocks(ticker) ON UPDATE CASCADE ON DELETE CASCADE,
  source_url TEXT,
  scraped_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- The nullable legacy column is used only until old captures are backfilled.
ALTER TABLE stock_financials ADD COLUMN IF NOT EXISTS payload JSONB;
ALTER TABLE stock_financials ALTER COLUMN payload DROP NOT NULL;
ALTER TABLE stock_financials ADD COLUMN IF NOT EXISTS source_url TEXT;
CREATE INDEX IF NOT EXISTS stock_financials_ticker_scraped_at_idx
  ON stock_financials (ticker, scraped_at DESC);

CREATE TABLE IF NOT EXISTS stock_financial_values (
  capture_id BIGINT NOT NULL REFERENCES stock_financials(id) ON DELETE CASCADE,
  ticker TEXT NOT NULL REFERENCES stocks(ticker) ON UPDATE CASCADE ON DELETE CASCADE,
  scraped_at TIMESTAMPTZ NOT NULL,
  table_id TEXT NOT NULL,
  table_kind TEXT NOT NULL,
  table_title TEXT NOT NULL,
  row_id TEXT NOT NULL,
  row_role TEXT NOT NULL,
  row_left TEXT,
  row_right TEXT,
  row_classes TEXT NOT NULL,
  metric_key TEXT NOT NULL,
  metric_name TEXT NOT NULL,
  label TEXT NOT NULL,
  label_en TEXT,
  label_id TEXT,
  period_key TEXT NOT NULL,
  period_label TEXT NOT NULL,
  period_year INTEGER,
  period_quarter SMALLINT CHECK (period_quarter BETWEEN 1 AND 4),
  period_type TEXT NOT NULL,
  period_basis TEXT NOT NULL,
  currency TEXT,
  value NUMERIC,
  is_missing BOOLEAN NOT NULL,
  raw_value TEXT,
  display_value TEXT NOT NULL,
  source_value_idr TEXT,
  source_value_usd TEXT,
  percentage NUMERIC,
  PRIMARY KEY (capture_id, table_id, row_id, period_key),
  CHECK (NOT is_missing OR value IS NULL)
);
ALTER TABLE stock_financial_values ADD COLUMN IF NOT EXISTS raw_percentage TEXT;
CREATE INDEX IF NOT EXISTS stock_financial_values_ticker_metric_period_idx
  ON stock_financial_values (ticker, metric_key, period_year, period_quarter, scraped_at DESC);
CREATE INDEX IF NOT EXISTS stock_financial_values_metric_period_value_idx
  ON stock_financial_values (metric_key, period_year, period_quarter, value)
  WHERE value IS NOT NULL;
`)
	return err
}
