package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureMarketTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS market_klines (
			id BIGSERIAL PRIMARY KEY,
			symbol TEXT NOT NULL,
			source TEXT NOT NULL,
			interval TEXT NOT NULL,
			open_time DATE NOT NULL,
			open NUMERIC NOT NULL,
			high NUMERIC NOT NULL,
			low NUMERIC NOT NULL,
			close NUMERIC NOT NULL,
			volume BIGINT NOT NULL,
			fetched_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT market_klines_symbol_source_interval_open_time_key
				UNIQUE (symbol, source, interval, open_time)
		);

		CREATE INDEX IF NOT EXISTS market_klines_lookup_idx
			ON market_klines (symbol, source, interval, open_time);

		CREATE TABLE IF NOT EXISTS stock_klines (
			id BIGSERIAL PRIMARY KEY, symbol TEXT NOT NULL, source TEXT NOT NULL,
			interval TEXT NOT NULL, open_time TIMESTAMPTZ NOT NULL,
			open NUMERIC NOT NULL, high NUMERIC NOT NULL, low NUMERIC NOT NULL, close NUMERIC NOT NULL,
			volume BIGINT NOT NULL, fetched_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (symbol, source, interval, open_time)
		);
		CREATE INDEX IF NOT EXISTS stock_klines_lookup_idx
			ON stock_klines (symbol, source, interval, open_time);

		CREATE TABLE IF NOT EXISTS stocks (
			ticker TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		ALTER TABLE stocks ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;
		CREATE INDEX IF NOT EXISTS stocks_active_ticker_idx ON stocks (active, ticker);
	`)
	return err
}

func EnsureMasterDataTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS master_data (
			key TEXT PRIMARY KEY,
			value NUMERIC NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		INSERT INTO master_data (key, value)
		VALUES
			('usd_idr_rate', 0),
			('bi_rate', 0)
		ON CONFLICT (key) DO NOTHING;
	`)
	return err
}
