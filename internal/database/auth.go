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
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS t_acl_rule (
			acl_rule_id BIGSERIAL PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT TRUE
		);

		CREATE TABLE IF NOT EXISTS t_role_acl_rule (
			role_id INTEGER NOT NULL REFERENCES t_role(role_id),
			acl_rule_id BIGINT NOT NULL REFERENCES t_acl_rule(acl_rule_id),
			PRIMARY KEY (role_id, acl_rule_id)
		);

		INSERT INTO t_role (role_id, name)
		VALUES (1, 'user'), (2, 'admin')
		ON CONFLICT (role_id) DO UPDATE SET name = EXCLUDED.name;
	`)
	return err
}
