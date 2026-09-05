package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type StockYahooProvider interface {
	GetQuote(context.Context, string) (repository.MarketPrice, error)
	GetKlines(context.Context, string, time.Time, time.Time) ([]repository.StockKline, error)
}
type StockRepository interface {
	CreateStock(context.Context, repository.Stock) (repository.Stock, error)
	ListStocks(context.Context) ([]repository.Stock, error)
	SearchActiveStocks(context.Context, string, int) ([]repository.Stock, error)
	GetStock(context.Context, string) (repository.Stock, error)
	UpdateStockName(context.Context, string, string) (repository.Stock, error)
	UpdateStockStatus(context.Context, string, bool) (repository.Stock, error)
	ListKlines(context.Context, string, repository.Source, string, time.Time, time.Time) ([]repository.StockKline, error)
	UpsertKlines(context.Context, []repository.StockKline) error
	GetFundamentals(context.Context, string) (repository.StockFundamentals, error)
}
type StockService struct {
	yahooProvider StockYahooProvider
	repository    StockRepository
}

func NewStockService(yahooProvider StockYahooProvider, repo StockRepository) *StockService {
	return &StockService{yahooProvider: yahooProvider, repository: repo}
}

var (
	ErrStockNotFound    = errors.New("stock not found")
	ErrInvalidStock     = errors.New("ticker and name are required")
	ErrInvalidStockName = errors.New("name is required")
	ErrInactiveStock    = errors.New("stock is inactive")
)

func (s *StockService) CreateStock(ctx context.Context, ticker, name string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	name = strings.TrimSpace(name)
	if ticker == "" || name == "" {
		return repository.Stock{}, ErrInvalidStock
	}
	stock, err := s.repository.CreateStock(ctx, repository.Stock{Ticker: ticker, Name: name})
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.CreateStock: %w", err)
	}
	return stock, nil
}
func (s *StockService) ListStocks(ctx context.Context) ([]repository.Stock, error) {
	stocks, err := s.repository.ListStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("stockService.ListStocks: %w", err)
	}
	return stocks, nil
}
func (s *StockService) SearchTickers(ctx context.Context, query string, limit int) ([]repository.Stock, error) {
	query = strings.ToUpper(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	stocks, err := s.repository.SearchActiveStocks(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("stockService.SearchTickers: %w", err)
	}
	return stocks, nil
}
func (s *StockService) GetStock(ctx context.Context, ticker string) (repository.Stock, error) {
	stock, err := s.repository.GetStock(ctx, strings.ToUpper(strings.TrimSpace(ticker)))
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.GetStock: %w", err)
	}
	return stock, nil
}
func (s *StockService) UpdateStockName(ctx context.Context, ticker, name string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	name = strings.TrimSpace(name)
	if name == "" {
		return repository.Stock{}, ErrInvalidStockName
	}
	stock, err := s.repository.UpdateStockName(ctx, ticker, name)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.UpdateStockName: %w", err)
	}
	return stock, nil
}
func (s *StockService) UpdateStockStatus(ctx context.Context, ticker string, active bool) (repository.Stock, error) {
	stock, err := s.repository.UpdateStockStatus(ctx, strings.ToUpper(strings.TrimSpace(ticker)), active)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.UpdateStockStatus: %w", err)
	}
	return stock, nil
}
func (s *StockService) GetKlines(ctx context.Context, ticker string, from, to time.Time) ([]repository.StockKline, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return nil, err
	}
	symbol := yahooStockSymbol(stock.Ticker)
	items, err := s.repository.ListKlines(ctx, symbol, repository.SourceYahooFinance, "1d", from, to)
	if err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> StockRepository.ListKlines: %w", err)
	}
	if len(items) > 0 {
		return items, nil
	}

	fetched, err := s.yahooProvider.GetKlines(ctx, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> Yahoo.GetKlines: %w", err)
	}
	if err := s.repository.UpsertKlines(ctx, fetched); err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> StockRepository.UpsertKlines: %w", err)
	}
	items, err = s.repository.ListKlines(ctx, symbol, repository.SourceYahooFinance, "1d", from, to)
	if err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> StockRepository.ListKlinesAfterFetch: %w", err)
	}
	return items, nil
}

func (s *StockService) GetQuote(ctx context.Context, ticker string) (repository.MarketPrice, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return repository.MarketPrice{}, err
	}
	quote, err := s.yahooProvider.GetQuote(ctx, yahooStockSymbol(stock.Ticker))
	if err != nil {
		return repository.MarketPrice{}, fmt.Errorf("stockService.GetQuote -> Yahoo.GetQuote: %w", err)
	}
	return quote, nil
}

func (s *StockService) GetFundamentals(ctx context.Context, ticker string) (repository.StockFundamentals, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return repository.StockFundamentals{}, err
	}
	fundamentals, err := s.repository.GetFundamentals(ctx, stock.Ticker)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.StockFundamentals{}, ErrStockNotFound
	}
	if err != nil {
		return repository.StockFundamentals{}, fmt.Errorf("stockService.GetFundamentals -> GetFundamentals: %w", err)
	}
	return fundamentals, nil
}

func (s *StockService) activeStock(ctx context.Context, ticker string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return repository.Stock{}, fmt.Errorf("ticker is required")
	}
	stock, err := s.repository.GetStock(ctx, ticker)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, fmt.Errorf("%w: %s", ErrStockNotFound, ticker)
	}
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.activeStock -> GetStock: %w", err)
	}
	if !stock.Active {
		return repository.Stock{}, ErrInactiveStock
	}
	return stock, nil
}

func yahooStockSymbol(ticker string) string {
	return strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(ticker)), ".JK") + ".JK"
}
