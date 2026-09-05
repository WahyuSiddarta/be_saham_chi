package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsurePortfolioTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		CREATE TABLE IF NOT EXISTS asset_classes (
			asset_class_id SMALLSERIAL PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		INSERT INTO asset_classes (code, name)
		VALUES
			('stock', 'Stock'),
			('bond', 'Bond'),
			('cash', 'Cash'),
			('commodity', 'Commodity')
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name;

		CREATE TABLE IF NOT EXISTS assets (
			asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			asset_class_id SMALLINT NOT NULL REFERENCES asset_classes(asset_class_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT,
			symbol TEXT NOT NULL,
			name TEXT NOT NULL,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			pricing_method TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'manual',
			provider_symbol TEXT,
			status BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT assets_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT assets_pricing_method_check
				CHECK (pricing_method IN ('api', 'manual', 'fixed')),
			CONSTRAINT assets_symbol_class_key
				UNIQUE (asset_class_id, symbol)
		);

		CREATE INDEX IF NOT EXISTS assets_class_status_idx
			ON assets (asset_class_id, status);

		INSERT INTO assets (asset_class_id, symbol, name, currency_code, pricing_method, source, provider_symbol)
		SELECT asset_class_id, 'GOLD-GRAM', 'Gold', 'IDR', 'api', 'yahoo', 'GC=F'
		FROM asset_classes
		WHERE code = 'commodity'
		ON CONFLICT (asset_class_id, symbol) DO UPDATE
		SET name = EXCLUDED.name,
			pricing_method = EXCLUDED.pricing_method,
			source = EXCLUDED.source,
			provider_symbol = EXCLUDED.provider_symbol,
			status = TRUE;

		CREATE TABLE IF NOT EXISTS asset_terms (
			asset_term_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			asset_id UUID NOT NULL REFERENCES assets(asset_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			issue_date DATE,
			maturity_date DATE,
			annual_rate NUMERIC(12, 6),
			coupon_frequency TEXT,
			principal_value NUMERIC(30, 2),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT asset_terms_asset_key
				UNIQUE (asset_id),
			CONSTRAINT asset_terms_dates_check
				CHECK (
					issue_date IS NULL
					OR maturity_date IS NULL
					OR maturity_date >= issue_date
				),
			CONSTRAINT asset_terms_annual_rate_check
				CHECK (annual_rate IS NULL OR annual_rate >= 0),
			CONSTRAINT asset_terms_coupon_frequency_check
				CHECK (coupon_frequency IS NULL OR coupon_frequency IN ('monthly', 'quarterly', 'semiannual', 'annual')),
			CONSTRAINT asset_terms_principal_value_check
				CHECK (principal_value IS NULL OR principal_value >= 0)
		);


		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'asset_terms_coupon_frequency_check'
			) THEN
				ALTER TABLE asset_terms
					ADD CONSTRAINT asset_terms_coupon_frequency_check
					CHECK (coupon_frequency IS NULL OR coupon_frequency IN ('monthly', 'quarterly', 'semiannual', 'annual'));
			END IF;
		END $$;

		CREATE INDEX IF NOT EXISTS asset_terms_maturity_date_idx
			ON asset_terms (maturity_date);

		CREATE TABLE IF NOT EXISTS asset_valuations (
			valuation_id BIGSERIAL PRIMARY KEY,
			asset_id UUID NOT NULL REFERENCES assets(asset_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			valuation_date DATE NOT NULL,
			price NUMERIC(30, 8) NOT NULL,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			source TEXT NOT NULL DEFAULT 'manual',
			notes TEXT,
			created_by UUID REFERENCES t_user(user_id)
				ON UPDATE CASCADE
				ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT asset_valuations_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT asset_valuations_price_check
				CHECK (price >= 0),
			CONSTRAINT asset_valuations_asset_date_source_key
				UNIQUE (asset_id, valuation_date, source)
		);

		CREATE INDEX IF NOT EXISTS asset_valuations_latest_idx
			ON asset_valuations (asset_id, valuation_date DESC);

		CREATE TABLE IF NOT EXISTS portfolios (
			portfolio_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES t_user(user_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			name TEXT NOT NULL,
			base_currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			is_main BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolios_base_currency_code_check
				CHECK (base_currency_code = 'IDR')
		);

		ALTER TABLE portfolios
			ADD COLUMN IF NOT EXISTS is_main BOOLEAN NOT NULL DEFAULT FALSE;


		CREATE INDEX IF NOT EXISTS portfolios_user_idx
			ON portfolios (user_id);

		CREATE UNIQUE INDEX IF NOT EXISTS portfolios_user_name_key
			ON portfolios (user_id, name);

		CREATE TABLE IF NOT EXISTS portfolio_accounts (
			account_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			portfolio_id UUID NOT NULL REFERENCES portfolios(portfolio_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			name TEXT NOT NULL,
			account_type TEXT NOT NULL,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolio_accounts_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT portfolio_accounts_account_type_check
				CHECK (account_type IN ('broker', 'bank', 'manual'))
		);

		CREATE INDEX IF NOT EXISTS portfolio_accounts_portfolio_idx
			ON portfolio_accounts (portfolio_id);

		CREATE UNIQUE INDEX IF NOT EXISTS portfolio_accounts_portfolio_name_key
			ON portfolio_accounts (portfolio_id, name);

		CREATE TABLE IF NOT EXISTS portfolio_transactions (
			transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			portfolio_id UUID NOT NULL REFERENCES portfolios(portfolio_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			account_id UUID REFERENCES portfolio_accounts(account_id)
				ON UPDATE CASCADE
				ON DELETE SET NULL,
			asset_id UUID NOT NULL REFERENCES assets(asset_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT,
			transaction_type TEXT NOT NULL,
			transaction_date TIMESTAMPTZ NOT NULL,
			quantity NUMERIC(30, 8) NOT NULL DEFAULT 0,
			price NUMERIC(30, 8) NOT NULL DEFAULT 0,
			gross_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			cost_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			accrued_coupon_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			fee_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			tax_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			net_amount NUMERIC(30, 2) NOT NULL DEFAULT 0,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			notes TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolio_transactions_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT portfolio_transactions_type_check
				CHECK (
					transaction_type IN (
						'buy',
						'sell',
						'deposit',
						'withdrawal',
						'dividend',
						'coupon',
						'interest',
						'fee',
						'tax',
						'maturity'
					)
				),
			CONSTRAINT portfolio_transactions_amounts_check
				CHECK (
					quantity >= 0
					AND price >= 0
					AND gross_amount >= 0
					AND cost_amount >= 0
					AND accrued_coupon_amount >= 0
					AND fee_amount >= 0
					AND tax_amount >= 0
					AND net_amount >= 0
				)
		);

		ALTER TABLE portfolio_transactions
			ADD COLUMN IF NOT EXISTS cost_amount NUMERIC(30, 2) NOT NULL DEFAULT 0;

		ALTER TABLE portfolio_transactions
			ADD COLUMN IF NOT EXISTS accrued_coupon_amount NUMERIC(30, 2) NOT NULL DEFAULT 0;

		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'portfolio_transactions_cost_amount_check'
			) THEN
				ALTER TABLE portfolio_transactions
					ADD CONSTRAINT portfolio_transactions_cost_amount_check
					CHECK (cost_amount >= 0);
			END IF;
		END $$;

		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'portfolio_transactions_accrued_coupon_amount_check'
			) THEN
				ALTER TABLE portfolio_transactions
					ADD CONSTRAINT portfolio_transactions_accrued_coupon_amount_check
					CHECK (accrued_coupon_amount >= 0);
			END IF;
		END $$;

		CREATE INDEX IF NOT EXISTS portfolio_transactions_portfolio_date_idx
			ON portfolio_transactions (portfolio_id, transaction_date DESC);

		CREATE INDEX IF NOT EXISTS portfolio_transactions_asset_idx
			ON portfolio_transactions (asset_id);

		CREATE TABLE IF NOT EXISTS portfolio_holdings (
			holding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			portfolio_id UUID NOT NULL REFERENCES portfolios(portfolio_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			account_id UUID REFERENCES portfolio_accounts(account_id)
				ON UPDATE CASCADE
				ON DELETE SET NULL,
			asset_id UUID NOT NULL REFERENCES assets(asset_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT,
			quantity NUMERIC(30, 8) NOT NULL DEFAULT 0,
			average_cost NUMERIC(30, 8) NOT NULL DEFAULT 0,
			total_cost NUMERIC(30, 2) NOT NULL DEFAULT 0,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolio_holdings_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT portfolio_holdings_amounts_check
				CHECK (
					quantity >= 0
					AND average_cost >= 0
					AND total_cost >= 0
				)
		);

		CREATE INDEX IF NOT EXISTS portfolio_holdings_portfolio_idx
			ON portfolio_holdings (portfolio_id);

		CREATE INDEX IF NOT EXISTS portfolio_holdings_asset_idx
			ON portfolio_holdings (asset_id);

		CREATE UNIQUE INDEX IF NOT EXISTS portfolio_holdings_account_asset_key
			ON portfolio_holdings (portfolio_id, account_id, asset_id)
			WHERE account_id IS NOT NULL;

		CREATE UNIQUE INDEX IF NOT EXISTS portfolio_holdings_no_account_asset_key
			ON portfolio_holdings (portfolio_id, asset_id)
			WHERE account_id IS NULL;

		CREATE TABLE IF NOT EXISTS portfolio_asset_class_snapshots (
			snapshot_id BIGSERIAL PRIMARY KEY,
			portfolio_id UUID NOT NULL REFERENCES portfolios(portfolio_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			asset_class_id SMALLINT NOT NULL REFERENCES asset_classes(asset_class_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT,
			snapshot_date DATE NOT NULL,
			total_cost NUMERIC(30, 2) NOT NULL DEFAULT 0,
			market_value NUMERIC(30, 2) NOT NULL DEFAULT 0,
			unrealized_pnl NUMERIC(30, 2) NOT NULL DEFAULT 0,
			realized_pnl NUMERIC(30, 2) NOT NULL DEFAULT 0,
			total_pnl NUMERIC(30, 2) NOT NULL DEFAULT 0,
			total_pnl_percent NUMERIC(18, 8) NOT NULL DEFAULT 0,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolio_asset_class_snapshots_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT portfolio_asset_class_snapshots_amounts_check
				CHECK (
					total_cost >= 0
					AND market_value >= 0
				),
			CONSTRAINT portfolio_asset_class_snapshots_day_key
				UNIQUE (portfolio_id, asset_class_id, snapshot_date)
		);

		CREATE INDEX IF NOT EXISTS portfolio_asset_class_snapshots_lookup_idx
			ON portfolio_asset_class_snapshots (portfolio_id, snapshot_date, asset_class_id);

		CREATE TABLE IF NOT EXISTS portfolio_holding_valuations (
			valuation_id BIGSERIAL PRIMARY KEY,
			portfolio_id UUID NOT NULL REFERENCES portfolios(portfolio_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			account_id UUID NOT NULL REFERENCES portfolio_accounts(account_id)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			asset_id UUID NOT NULL REFERENCES assets(asset_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT,
			valuation_date DATE NOT NULL,
			price NUMERIC(30, 8) NOT NULL,
			market_value NUMERIC(30, 2) NOT NULL,
			currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
			source TEXT NOT NULL DEFAULT 'manual',
			notes TEXT,
			created_by UUID REFERENCES t_user(user_id)
				ON UPDATE CASCADE
				ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT portfolio_holding_valuations_currency_code_check
				CHECK (currency_code = 'IDR'),
			CONSTRAINT portfolio_holding_valuations_amounts_check
				CHECK (
					price >= 0
					AND market_value >= 0
				),
			CONSTRAINT portfolio_holding_valuations_day_source_key
				UNIQUE (portfolio_id, account_id, asset_id, valuation_date, source)
		);

		CREATE INDEX IF NOT EXISTS portfolio_holding_valuations_latest_idx
			ON portfolio_holding_valuations (portfolio_id, account_id, asset_id, valuation_date DESC);
	`)
	return err
}
