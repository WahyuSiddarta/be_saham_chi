package service

import (
	"context"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type stubStockRepository struct {
	query        string
	limit        int
	stock        repository.Stock
	listed       [][]repository.StockKline
	listCalls    int
	upserted     []repository.StockKline
	listSource   repository.Source
	listInterval string
	fundamentals repository.StockFundamentals
}

func (r *stubStockRepository) CreateStock(context.Context, repository.Stock) (repository.Stock, error) {
	return repository.Stock{}, nil
}
func (r *stubStockRepository) ListStocks(context.Context) ([]repository.Stock, error) {
	return nil, nil
}
func (r *stubStockRepository) SearchActiveStocks(_ context.Context, query string, limit int) ([]repository.Stock, error) {
	r.query, r.limit = query, limit
	return []repository.Stock{{Ticker: "BBCA", Name: "Bank Central Asia Tbk"}}, nil
}
func (r *stubStockRepository) GetStock(context.Context, string) (repository.Stock, error) {
	return r.stock, nil
}
func (r *stubStockRepository) UpdateStockName(context.Context, string, string) (repository.Stock, error) {
	return repository.Stock{}, nil
}
func (r *stubStockRepository) UpdateStockStatus(context.Context, string, bool) (repository.Stock, error) {
	return repository.Stock{}, nil
}
func (r *stubStockRepository) ListKlines(_ context.Context, _ string, source repository.Source, interval string, _ time.Time, _ time.Time) ([]repository.StockKline, error) {
	r.listSource, r.listInterval = source, interval
	index := r.listCalls
	r.listCalls++
	if index < len(r.listed) {
		return r.listed[index], nil
	}
	return nil, nil
}
func (r *stubStockRepository) UpsertKlines(_ context.Context, items []repository.StockKline) error {
	r.upserted = items
	return nil
}
func (r *stubStockRepository) GetFundamentals(context.Context, string) (repository.StockFundamentals, error) {
	return r.fundamentals, nil
}

type stubYahooStockProvider struct {
	symbol     string
	klineCalls int
	klines     []repository.StockKline
}

func (p *stubYahooStockProvider) GetQuote(_ context.Context, symbol string) (repository.MarketPrice, error) {
	p.symbol = symbol
	return repository.MarketPrice{Symbol: symbol, Close: 9300, Source: repository.SourceYahooFinance}, nil
}
func (p *stubYahooStockProvider) GetKlines(context.Context, string, time.Time, time.Time) ([]repository.StockKline, error) {
	p.klineCalls++
	return p.klines, nil
}

func TestSearchTickersNormalizesQueryAndDefaultsLimit(t *testing.T) {
	repoStub := &stubStockRepository{}
	service := NewStockService(nil, repoStub)

	stocks, err := service.SearchTickers(context.Background(), " bb ", 0)
	if err != nil {
		t.Fatalf("SearchTickers returned error: %v", err)
	}
	if repoStub.query != "BB" || repoStub.limit != 10 {
		t.Fatalf("expected query BB and limit 10, got query %q and limit %d", repoStub.query, repoStub.limit)
	}
	if len(stocks) != 1 || stocks[0].Ticker != "BBCA" {
		t.Fatalf("unexpected stocks: %#v", stocks)
	}
}

func TestSearchTickersCapsLimitAtFifty(t *testing.T) {
	repoStub := &stubStockRepository{}
	service := NewStockService(nil, repoStub)

	if _, err := service.SearchTickers(context.Background(), "BB", 100); err != nil {
		t.Fatalf("SearchTickers returned error: %v", err)
	}
	if repoStub.limit != 50 {
		t.Fatalf("expected limit 50, got %d", repoStub.limit)
	}
}

func TestGetQuoteUsesYahooJakartaSymbol(t *testing.T) {
	repoStub := &stubStockRepository{stock: repository.Stock{Ticker: "BBCA", Active: true}}
	provider := &stubYahooStockProvider{}
	stockService := NewStockService(provider, repoStub)

	quote, err := stockService.GetQuote(context.Background(), "bbca")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if provider.symbol != "BBCA.JK" || quote.Symbol != "BBCA.JK" {
		t.Fatalf("expected BBCA.JK, provider got %q and quote got %q", provider.symbol, quote.Symbol)
	}
}

func TestGetFundamentalsReadsStoredStockbitSnapshot(t *testing.T) {
	scrapedAt := time.Date(2026, time.August, 28, 1, 2, 3, 0, time.UTC)
	repoStub := &stubStockRepository{
		stock: repository.Stock{Ticker: "BBCA", Active: true},
		fundamentals: repository.StockFundamentals{
			Ticker:    "BBCA",
			Payload:   []byte(`{"currentValuation":{}}`),
			ScrapedAt: scrapedAt,
		},
	}
	stockService := NewStockService(nil, repoStub)

	fundamentals, err := stockService.GetFundamentals(context.Background(), " bbca ")
	if err != nil {
		t.Fatalf("GetFundamentals returned error: %v", err)
	}
	if fundamentals.Ticker != "BBCA" || len(fundamentals.Payload) == 0 || !fundamentals.ScrapedAt.Equal(scrapedAt) {
		t.Fatalf("unexpected fundamentals: %#v", fundamentals)
	}
}

func TestGetKlinesReturnsYahooCacheWithoutProviderCall(t *testing.T) {
	cached := []repository.StockKline{{Symbol: "BBCA.JK", Interval: "1d", Close: 9300}}
	repoStub := &stubStockRepository{stock: repository.Stock{Ticker: "BBCA", Active: true}, listed: [][]repository.StockKline{cached}}
	provider := &stubYahooStockProvider{}
	stockService := NewStockService(provider, repoStub)

	items, err := stockService.GetKlines(context.Background(), "BBCA", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("GetKlines returned error: %v", err)
	}
	if provider.klineCalls != 0 || len(items) != 1 || repoStub.listSource != repository.SourceYahooFinance || repoStub.listInterval != "1d" {
		t.Fatalf("unexpected cache behavior: calls=%d items=%#v source=%s interval=%s", provider.klineCalls, items, repoStub.listSource, repoStub.listInterval)
	}
}

func TestGetKlinesFetchesPersistsAndRereadsOnCacheMiss(t *testing.T) {
	fetched := []repository.StockKline{{Symbol: "BBCA.JK", Interval: "1d", Close: 9300}}
	repoStub := &stubStockRepository{stock: repository.Stock{Ticker: "BBCA", Active: true}, listed: [][]repository.StockKline{nil, fetched}}
	provider := &stubYahooStockProvider{klines: fetched}
	stockService := NewStockService(provider, repoStub)

	items, err := stockService.GetKlines(context.Background(), "BBCA", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("GetKlines returned error: %v", err)
	}
	if provider.klineCalls != 1 || repoStub.listCalls != 2 || len(repoStub.upserted) != 1 || len(items) != 1 {
		t.Fatalf("unexpected cache fill: provider=%d lists=%d upserted=%d items=%d", provider.klineCalls, repoStub.listCalls, len(repoStub.upserted), len(items))
	}
}
