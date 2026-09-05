package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func (r *Repository) GetFundamentals(ctx context.Context, ticker string) (StockFundamentals, error) {
	var fundamentals StockFundamentals
	err := r.db.GetContext(ctx, &fundamentals, `
		SELECT ticker AS ticker, payload AS payload, scraped_at AS scraped_at, updated_at AS updated_at FROM stock_fundamentals
		WHERE ticker = $1
	`, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return StockFundamentals{}, ErrStockNotFound
	}
	return fundamentals, err
}

type StockFundamentals struct {
	Ticker    string          `db:"ticker"`
	Payload   json.RawMessage `db:"payload"`
	ScrapedAt time.Time       `db:"scraped_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}
