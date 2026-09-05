package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func EnsureAuthTables(ctx context.Context, pool *sqlx.DB) error {
	_, err := pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS t_role (
			role_id INTEGER PRIMARY KEY,
			name VARCHAR NOT NULL
		);

		CREATE TABLE IF NOT EXISTS t_user (
			user_id UUID PRIMARY KEY,
			role INTEGER NOT NULL DEFAULT 1,
			email VARCHAR NOT NULL UNIQUE,
			password VARCHAR,
			status BOOLEAN NOT NULL DEFAULT TRUE,
			role_id INTEGER REFERENCES t_role(role_id)
				ON UPDATE CASCADE
				ON DELETE SET DEFAULT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		ALTER TABLE t_user
			ADD COLUMN IF NOT EXISTS status BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE t_user ADD COLUMN IF NOT EXISTS role_id INTEGER REFERENCES t_role(role_id);

		CREATE TABLE IF NOT EXISTS t_acl_rule (
			acl_rule_id BIGSERIAL PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT TRUE
		);

		ALTER TABLE t_acl_rule
			ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;

		CREATE TABLE IF NOT EXISTS t_role_acl_rule (
			role_id INTEGER NOT NULL REFERENCES t_role(role_id),
			acl_rule_id BIGINT NOT NULL REFERENCES t_acl_rule(acl_rule_id),
			PRIMARY KEY (role_id, acl_rule_id)
		);

		INSERT INTO t_role (role_id, name)
		VALUES
			(1, 'user'),
			(2, 'admin')
		ON CONFLICT (role_id) DO UPDATE
		SET name = EXCLUDED.name;

		INSERT INTO t_acl_rule (code, description)
		VALUES
			('market.gold.read', 'Read gold market data'),
			('market.wti.read', 'Read WTI crude oil market data'),
			('market.crude.read', 'Read Brent crude oil market data'),
			('market.stock.read', 'Read stock market data'),
			('stock.manage', 'Create and manage stock provider configuration'),
			('portfolio.read', 'Read portfolios'),
			('portfolio.create', 'Create portfolios'),
			('portfolio.update', 'Update portfolios'),
			('portfolio.delete', 'Delete portfolios'),
			('portfolio.cash.read', 'Read portfolio cash, snapshots, and cash transactions'),
			('portfolio.cash.create', 'Create portfolio cash and cash transactions'),
			('portfolio.cash.update', 'Update portfolio cash transactions'),
			('portfolio.cash.delete', 'Delete portfolio cash transactions'),
			('portfolio.bond.read', 'Read portfolio bonds, snapshots, and bond transactions'),
			('portfolio.bond.create', 'Create portfolio bonds and bond transactions'),
			('portfolio.bond.update', 'Update portfolio bonds, valuations, and bond transactions'),
			('portfolio.bond.delete', 'Delete portfolio bond transactions'),
			('portfolio.commodity.read', 'Read portfolio commodities and transactions'),
			('portfolio.commodity.create', 'Create portfolio commodity transactions'),
			('portfolio.commodity.update', 'Update portfolio commodity transactions'),
			('portfolio.commodity.delete', 'Delete portfolio commodity transactions'),
			('master_data.read', 'Read master data'),
			('master_data.update', 'Update master data')
		ON CONFLICT (code) DO UPDATE
		SET description = EXCLUDED.description,
			active = TRUE;

		INSERT INTO t_role_acl_rule (role_id, acl_rule_id)
		SELECT 1, ar.acl_rule_id
		FROM t_acl_rule ar
		WHERE ar.code IN (
			'market.gold.read',
			'market.wti.read',
			'market.crude.read',
			'market.stock.read',
			'portfolio.read',
			'portfolio.create',
			'portfolio.update',
			'portfolio.delete',
			'portfolio.cash.read',
			'portfolio.cash.create',
			'portfolio.cash.update',
			'portfolio.cash.delete',
			'portfolio.bond.read',
			'portfolio.bond.create',
			'portfolio.bond.update',
			'portfolio.bond.delete',
			'portfolio.commodity.read',
			'portfolio.commodity.create',
			'portfolio.commodity.update',
			'portfolio.commodity.delete'
		)
		ON CONFLICT (role_id, acl_rule_id) DO NOTHING;

		INSERT INTO t_role_acl_rule (role_id, acl_rule_id)
		SELECT 2, ar.acl_rule_id
		FROM t_acl_rule ar
		ON CONFLICT (role_id, acl_rule_id) DO NOTHING;
	`)
	return err
}
