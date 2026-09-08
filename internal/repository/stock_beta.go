package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrStockBetaNotFound = errors.New("stock beta not found")

type StockBeta struct {
	Ticker    string    `db:"ticker"`
	Value     float64   `db:"value"`
	Period    string    `db:"period"`
	Interval  string    `db:"interval"`
	Source    string    `db:"source"`
	FetchedAt time.Time `db:"fetched_at"`
}

func (r *Repository) GetStockBeta(ctx context.Context, ticker string) (StockBeta, error) {
	var beta StockBeta
	err := r.db.GetContext(ctx, &beta, `
		SELECT ticker, value, period, interval, source, fetched_at
		FROM stock_betas
		WHERE ticker = $1
		  AND period = '5y'
		  AND interval = '1mo'
		  AND source = 'yahoo_finance'
		ORDER BY fetched_at DESC
		LIMIT 1
	`, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return StockBeta{}, ErrStockBetaNotFound
	}
	return beta, err
}
