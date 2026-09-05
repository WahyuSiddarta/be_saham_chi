package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrStockNotFound = errors.New("stock not found")

func (r *Repository) CreateStock(ctx context.Context, stock Stock) (Stock, error) {
	err := r.db.GetContext(ctx, &stock, `INSERT INTO stocks(ticker,name) VALUES($1,$2) RETURNING ticker AS ticker, name AS name, active AS active, created_at AS created_at, updated_at AS updated_at `, stock.Ticker, stock.Name)
	return stock, err
}
func (r *Repository) ListStocks(ctx context.Context) ([]Stock, error) {
	items := make([]Stock, 0)
	err := r.db.SelectContext(ctx, &items, "SELECT ticker,name,active,created_at,updated_at FROM stocks ORDER BY ticker")
	return items, err
}

func (r *Repository) SearchActiveStocks(ctx context.Context, query string, limit int) ([]Stock, error) {
	items := make([]Stock, 0)
	err := r.db.SelectContext(ctx, &items, "SELECT ticker,name FROM stocks WHERE active=TRUE AND ticker LIKE $1 ORDER BY CASE WHEN ticker=$2 THEN 0 ELSE 1 END,ticker LIMIT $3", query+"%", query, limit)
	return items, err
}

func (r *Repository) GetStock(ctx context.Context, ticker string) (Stock, error) {
	var stock Stock
	err := r.db.GetContext(ctx, &stock, `SELECT ticker AS ticker, name AS name, active AS active, created_at AS created_at, updated_at AS updated_at FROM stocks WHERE ticker=$1`, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return Stock{}, ErrStockNotFound
	}
	return stock, err
}
func (r *Repository) UpdateStockName(ctx context.Context, ticker, name string) (Stock, error) {
	var stock Stock
	err := r.db.GetContext(ctx, &stock, `UPDATE stocks SET name=$1,updated_at=NOW() WHERE ticker=$2 RETURNING ticker AS ticker, name AS name, active AS active, created_at AS created_at, updated_at AS updated_at `, name, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return Stock{}, ErrStockNotFound
	}
	return stock, err
}
func (r *Repository) UpdateStockStatus(ctx context.Context, ticker string, active bool) (Stock, error) {
	var stock Stock
	err := r.db.GetContext(ctx, &stock, `UPDATE stocks SET active=$1,updated_at=NOW() WHERE ticker=$2 RETURNING ticker AS ticker, name AS name, active AS active, created_at AS created_at, updated_at AS updated_at `, active, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return Stock{}, ErrStockNotFound
	}
	return stock, err
}
func (r *Repository) ListKlines(ctx context.Context, symbol string, source Source, interval string, from, to time.Time) ([]StockKline, error) {
	items := make([]StockKline, 0)
	err := r.db.SelectContext(ctx, &items, "SELECT symbol,interval,open_time,open,high,low,close,volume,source,fetched_at FROM stock_klines WHERE symbol=$1 AND source=$2 AND interval=$3 AND open_time >= $4 AND open_time <= $5 ORDER BY open_time ASC", symbol, string(source), interval, from, to)
	return items, err
}

func (r *Repository) UpsertKlines(ctx context.Context, items []StockKline) error {
	if len(items) == 0 {
		return nil
	}

	const query = `INSERT INTO stock_klines(symbol,source,interval,open_time,open,high,low,close,volume,fetched_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(symbol,source,interval,open_time) DO UPDATE SET open=EXCLUDED.open,high=EXCLUDED.high,low=EXCLUDED.low,close=EXCLUDED.close,volume=EXCLUDED.volume,fetched_at=EXCLUDED.fetched_at,updated_at=NOW()`
	return withinTxVoid(ctx, r.db, func(tx *sqlx.Tx) error {
		for _, k := range items {
			if _, err := tx.ExecContext(ctx, query, k.Symbol, string(k.Source), k.Interval, k.OpenTime, k.Open, k.High, k.Low, k.Close, k.Volume, k.FetchedAt); err != nil {
				return err
			}
		}
		return nil
	})
}
