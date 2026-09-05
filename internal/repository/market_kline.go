package repository

import (
	"context"
	"time"
)

func (r *Repository) ListMarketKlines(ctx context.Context, symbol string, source Source, interval string, startDate time.Time) ([]MarketKline, error) {
	items := make([]MarketKline, 0)
	err := r.db.SelectContext(ctx, &items, `
SELECT symbol, interval, open_time, open, high, low, close, volume, source, fetched_at
FROM market_klines WHERE symbol=$1 AND source=$2 AND interval=$3 AND open_time >= $4 ORDER BY open_time ASC
`, symbol, string(source), interval, startDate)
	return items, err
}

func (r *Repository) UpsertMany(ctx context.Context, klines []MarketKline) error {
	for _, kline := range klines {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO market_klines (
				symbol,
				source,
				interval,
				open_time,
				open,
				high,
				low,
				close,
				volume,
				fetched_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (symbol, source, interval, open_time)
			DO UPDATE SET
				open = EXCLUDED.open,
				high = EXCLUDED.high,
				low = EXCLUDED.low,
				close = EXCLUDED.close,
				volume = EXCLUDED.volume,
				fetched_at = EXCLUDED.fetched_at,
				updated_at = NOW()
		`,
			kline.Symbol,
			string(kline.Source),
			kline.Interval,
			kline.OpenTime,
			kline.Open,
			kline.High,
			kline.Low,
			kline.Close,
			kline.Volume,
			kline.FetchedAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
