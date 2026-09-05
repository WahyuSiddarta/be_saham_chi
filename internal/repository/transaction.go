package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func withinTx[T any](ctx context.Context, pool *sqlx.DB, run func(*sqlx.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := pool.BeginTxx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()

	result, err := run(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(); err != nil {
		return zero, err
	}
	return result, nil
}

func withinTxVoid(ctx context.Context, pool *sqlx.DB, run func(*sqlx.Tx) error) error {
	_, err := withinTx(ctx, pool, func(tx *sqlx.Tx) (struct{}, error) {
		return struct{}{}, run(tx)
	})
	return err
}
