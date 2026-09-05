package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrMasterDataNotFound = errors.New("master data not found")

func (r *Repository) GetMasterData(ctx context.Context, key string) (MasterData, error) {
	var item MasterData
	err := r.db.GetContext(ctx, &item, `
		SELECT key AS key, value AS value, updated_at AS updated_at FROM master_data
		WHERE key = $1
	`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return MasterData{}, ErrMasterDataNotFound
	}
	return item, err
}

func (r *Repository) ListMasterData(ctx context.Context) ([]MasterData, error) {
	items := make([]MasterData, 0)
	err := r.db.SelectContext(ctx, &items, "SELECT key, value, updated_at FROM master_data ORDER BY key")
	return items, err
}

func (r *Repository) UpdateMasterData(ctx context.Context, key string, value float64) (MasterData, error) {
	var item MasterData
	err := r.db.GetContext(ctx, &item, `
		UPDATE master_data
		SET value = $1, updated_at = NOW()
		WHERE key = $2
		RETURNING key AS key, value AS value, updated_at AS updated_at `, value, key)
	if errors.Is(err, sql.ErrNoRows) {
		return MasterData{}, ErrMasterDataNotFound
	}
	return item, err
}

type MasterData struct {
	Key       string    `db:"key"`
	Value     float64   `db:"value"`
	UpdatedAt time.Time `db:"updated_at"`
}
