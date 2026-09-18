package service

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

var ErrInvalidStockTransaction = errors.New("invalid stock transaction: use a known ticker, account, buy/sell, whole positive shares, positive price and non-negative fees and taxes")
var portfolioUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var portfolioTickerPattern = regexp.MustCompile(`^[A-Z0-9-]{1,20}$`)

type PortfolioStockRepository interface {
	ListPortfolioStocks(context.Context, string, string) ([]repository.PortfolioStockHolding, error)
	ListStockTransactions(context.Context, string, string) ([]repository.PortfolioStockTransaction, error)
	SaveStockTransaction(context.Context, string, string, string, repository.StockTransactionCommand) (repository.PortfolioStockTransaction, error)
	DeleteStockTransaction(context.Context, string, string, string) error
}
type PortfolioStockQuoteProvider interface {
	GetQuote(context.Context, string) (repository.MarketPrice, error)
}
type PortfolioStockService struct {
	repository PortfolioStockRepository
	quotes     PortfolioStockQuoteProvider
}

func NewPortfolioStockService(repo PortfolioStockRepository, quotes PortfolioStockQuoteProvider) *PortfolioStockService {
	return &PortfolioStockService{repository: repo, quotes: quotes}
}

func validateStockTransaction(portfolioID, id string, input repository.StockTransactionCommand) (repository.StockTransactionCommand, error) {
	if !portfolioUUIDPattern.MatchString(portfolioID) || (id != "" && !portfolioUUIDPattern.MatchString(id)) {
		return input, ErrInvalidStockTransaction
	}
	input.Ticker = strings.ToUpper(strings.TrimSpace(input.Ticker))
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.Notes = strings.TrimSpace(input.Notes)
	input.TransactionType = strings.ToLower(strings.TrimSpace(input.TransactionType))
	if !portfolioTickerPattern.MatchString(input.Ticker) || (input.AccountID == "" && input.AccountName == "") || (input.AccountID != "" && !portfolioUUIDPattern.MatchString(input.AccountID)) || len(input.AccountName) > 200 || len(input.Notes) > 2000 {
		return input, ErrInvalidStockTransaction
	}
	if input.TransactionType != "buy" && input.TransactionType != "sell" {
		return input, ErrInvalidStockTransaction
	}
	for _, v := range []float64{input.Quantity, input.Price, input.FeeAmount, input.TaxAmount, input.Quantity*input.Price + input.FeeAmount + input.TaxAmount} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1e15 {
			return input, ErrInvalidStockTransaction
		}
	}
	if input.Quantity <= 0 || math.Trunc(input.Quantity) != input.Quantity || input.Price <= 0 || (input.TransactionType == "sell" && input.Quantity*input.Price < input.FeeAmount+input.TaxAmount) {
		return input, ErrInvalidStockTransaction
	}
	if !input.TransactionDate.IsZero() && input.TransactionDate.After(time.Now().UTC()) {
		return input, ErrInvalidStockTransaction
	}
	return input, nil
}
func (s *PortfolioStockService) SaveTransaction(ctx context.Context, userID, portfolioID, id string, input repository.StockTransactionCommand) (repository.PortfolioStockTransaction, error) {
	input, err := validateStockTransaction(portfolioID, id, input)
	if err != nil {
		return repository.PortfolioStockTransaction{}, err
	}
	return s.repository.SaveStockTransaction(ctx, userID, portfolioID, id, input)
}
func (s *PortfolioStockService) DeleteTransaction(ctx context.Context, userID, portfolioID, id string) error {
	if !portfolioUUIDPattern.MatchString(portfolioID) || !portfolioUUIDPattern.MatchString(id) {
		return ErrInvalidStockTransaction
	}
	return s.repository.DeleteStockTransaction(ctx, userID, portfolioID, id)
}
func (s *PortfolioStockService) ListTransactions(ctx context.Context, userID, portfolioID string) ([]repository.PortfolioStockTransaction, error) {
	if !portfolioUUIDPattern.MatchString(portfolioID) {
		return nil, ErrInvalidStockTransaction
	}
	return s.repository.ListStockTransactions(ctx, userID, portfolioID)
}
func (s *PortfolioStockService) ListHoldings(ctx context.Context, userID, portfolioID string) ([]repository.PortfolioStockHolding, error) {
	if !portfolioUUIDPattern.MatchString(portfolioID) {
		return nil, ErrInvalidStockTransaction
	}
	items, err := s.repository.ListPortfolioStocks(ctx, userID, portfolioID)
	if err != nil {
		return nil, err
	}
	// Quote failures must not hide the ledger or fabricate a market valuation.
	quotes := map[string]repository.MarketPrice{}
	if s.quotes != nil {
		quoteCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		tickers := map[string]bool{}
		for _, item := range items {
			if item.Quantity > 0 {
				tickers[item.Ticker] = true
			}
		}
		var mu sync.Mutex
		var wg sync.WaitGroup
		slots := make(chan struct{}, 4)
		for ticker := range tickers {
			select {
			case slots <- struct{}{}:
			case <-quoteCtx.Done():
				continue
			}
			wg.Add(1)
			go func(ticker string) {
				defer wg.Done()
				defer func() { <-slots }()
				q, e := s.quotes.GetQuote(quoteCtx, yahooStockSymbol(ticker))
				if e == nil && q.Close > 0 && !math.IsNaN(q.Close) && !math.IsInf(q.Close, 0) {
					mu.Lock()
					quotes[ticker] = q
					mu.Unlock()
				}
			}(ticker)
		}
		wg.Wait()
	}
	for i := range items {
		item := &items[i]
		if item.Quantity == 0 {
			zero := 0.0
			item.MarketValue = &zero
			item.UnrealizedPnL = &zero
			continue
		}
		if q, ok := quotes[item.Ticker]; ok {
			price := q.Close
			value := item.Quantity * price
			pnl := value - item.TotalCost
			item.Price = &price
			item.PriceFetchedAt = &q.FetchedAt
			item.MarketValue = &value
			item.UnrealizedPnL = &pnl
		}
	}
	return items, nil
}
