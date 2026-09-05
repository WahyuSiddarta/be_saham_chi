package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureStockFundamentalsTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stock_fundamentals (
 ticker TEXT PRIMARY KEY REFERENCES stocks(ticker) ON UPDATE CASCADE ON DELETE CASCADE,
 payload JSONB NOT NULL DEFAULT '{}'::JSONB,
 scraped_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS stock_fundamentals_scraped_at_idx ON stock_fundamentals (scraped_at DESC);
`)
	return err
}
