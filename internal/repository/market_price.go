package repository

import "time"

type Source string

const (
	SourceYahooFinance Source = "yahoo_finance"
	SourceStockbit     Source = "stockbit"
)

type MarketPrice struct {
	Symbol    string    `db:"symbol"`
	Open      float64   `db:"open"`
	High      float64   `db:"high"`
	Low       float64   `db:"low"`
	Close     float64   `db:"close"`
	Volume    int64     `db:"volume"`
	Source    Source    `db:"source"`
	FetchedAt time.Time `db:"fetched_at"`
}

type Stock struct {
	Ticker    string    `db:"ticker"`
	Name      string    `db:"name"`
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type MarketKline struct {
	Symbol    string    `db:"symbol"`
	Interval  string    `db:"interval"`
	OpenTime  time.Time `db:"open_time"`
	Open      float64   `db:"open"`
	High      float64   `db:"high"`
	Low       float64   `db:"low"`
	Close     float64   `db:"close"`
	Volume    int64     `db:"volume"`
	Source    Source    `db:"source"`
	FetchedAt time.Time `db:"fetched_at"`
}

// StockKline uses the same OHLCV shape as the Yahoo market data it stores.
type StockKline = MarketKline
