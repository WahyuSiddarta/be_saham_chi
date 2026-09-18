package service

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

const stockTestPortfolioID = "11111111-1111-4111-8111-111111111111"

func TestValidateStockTransaction(t *testing.T) {
	valid := repository.StockTransactionCommand{AccountName: " Broker ", Ticker: " bbca ", TransactionType: " BUY ", Quantity: 100, Price: 9000, FeeAmount: 1500}
	normalized, err := validateStockTransaction(stockTestPortfolioID, "", valid)
	if err != nil || normalized.Ticker != "BBCA" || normalized.AccountName != "Broker" || normalized.TransactionType != "buy" {
		t.Fatalf("normalization: %+v, %v", normalized, err)
	}
	tests := map[string]func(*repository.StockTransactionCommand){
		"fractional shares": func(v *repository.StockTransactionCommand) { v.Quantity = 0.5 },
		"zero shares":       func(v *repository.StockTransactionCommand) { v.Quantity = 0 },
		"zero price":        func(v *repository.StockTransactionCommand) { v.Price = 0 },
		"nan":               func(v *repository.StockTransactionCommand) { v.Price = math.NaN() },
		"infinite":          func(v *repository.StockTransactionCommand) { v.Quantity = math.Inf(1) },
		"overflow":          func(v *repository.StockTransactionCommand) { v.Price = 1e20 },
		"negative fee":      func(v *repository.StockTransactionCommand) { v.FeeAmount = -1 },
		"negative tax":      func(v *repository.StockTransactionCommand) { v.TaxAmount = -1 },
		"no account":        func(v *repository.StockTransactionCommand) { v.AccountName = " " },
		"invalid account":   func(v *repository.StockTransactionCommand) { v.AccountID = "invalid" },
		"invalid ticker":    func(v *repository.StockTransactionCommand) { v.Ticker = "BBCA.JK" },
		"unsupported type":  func(v *repository.StockTransactionCommand) { v.TransactionType = "dividend" },
		"negative proceeds": func(v *repository.StockTransactionCommand) { v.TransactionType = "sell"; v.FeeAmount = 1e6 },
		"future date":       func(v *repository.StockTransactionCommand) { v.TransactionDate = time.Now().Add(time.Hour) },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid
			change(&input)
			if _, err := validateStockTransaction(stockTestPortfolioID, "", input); !errors.Is(err, ErrInvalidStockTransaction) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
	if _, err := validateStockTransaction("bad-id", "", valid); err == nil {
		t.Fatal("accepted invalid portfolio ID")
	}
	if _, err := validateStockTransaction(stockTestPortfolioID, "bad-id", valid); err == nil {
		t.Fatal("accepted invalid transaction ID")
	}
}

type portfolioStockRepoStub struct {
	PortfolioStockRepository
	items []repository.PortfolioStockHolding
	err   error
}

func (s portfolioStockRepoStub) ListPortfolioStocks(context.Context, string, string) ([]repository.PortfolioStockHolding, error) {
	return s.items, s.err
}

type portfolioStockQuoteStub struct {
	mu    sync.Mutex
	calls map[string]int
}

func (s *portfolioStockQuoteStub) GetQuote(_ context.Context, symbol string) (repository.MarketPrice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[symbol]++
	if symbol == "FAIL.JK" {
		return repository.MarketPrice{}, errors.New("provider unavailable")
	}
	return repository.MarketPrice{Close: 1200, FetchedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func TestPortfolioStocksValuationKeepsUnavailablePricesNull(t *testing.T) {
	repo := portfolioStockRepoStub{items: []repository.PortfolioStockHolding{
		{Ticker: "BBCA", Quantity: 100, TotalCost: 100000},
		{Ticker: "BBCA", Quantity: 200, TotalCost: 200000},
		{Ticker: "FAIL", Quantity: 100, TotalCost: 90000},
		{Ticker: "CLOSED", Quantity: 0, RealizedPnL: 5000},
	}}
	quotes := &portfolioStockQuoteStub{calls: map[string]int{}}
	got, err := NewPortfolioStockService(repo, quotes).ListHoldings(context.Background(), "user", stockTestPortfolioID)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].MarketValue == nil || *got[0].MarketValue != 120000 || *got[0].UnrealizedPnL != 20000 {
		t.Fatalf("wrong valuation: %+v", got[0])
	}
	if quotes.calls["BBCA.JK"] != 1 || quotes.calls["CLOSED.JK"] != 0 {
		t.Fatalf("unexpected quote calls: %v", quotes.calls)
	}
	if got[2].MarketValue != nil || got[2].UnrealizedPnL != nil || got[2].Price != nil {
		t.Fatal("failed quote must not fabricate valuation")
	}
	if got[3].MarketValue == nil || *got[3].MarketValue != 0 || got[3].RealizedPnL != 5000 {
		t.Fatal("closed position must retain realized PnL and zero market value")
	}
}
func TestPortfolioStocksDoNotFetchQuotesAfterOwnershipFailure(t *testing.T) {
	quotes := &portfolioStockQuoteStub{calls: map[string]int{}}
	_, err := NewPortfolioStockService(portfolioStockRepoStub{err: repository.ErrPortfolioNotFound}, quotes).ListHoldings(context.Background(), "other-user", stockTestPortfolioID)
	if !errors.Is(err, repository.ErrPortfolioNotFound) || len(quotes.calls) != 0 {
		t.Fatalf("ownership failure: %v, quotes: %v", err, quotes.calls)
	}
}
func TestStockValidationRunsBeforeRepository(t *testing.T) {
	svc := NewPortfolioStockService(nil, nil)
	if _, err := svc.SaveTransaction(context.Background(), "user", stockTestPortfolioID, "", repository.StockTransactionCommand{}); !errors.Is(err, ErrInvalidStockTransaction) {
		t.Fatal(err)
	}
	if err := svc.DeleteTransaction(context.Background(), "user", stockTestPortfolioID, "bad-id"); !errors.Is(err, ErrInvalidStockTransaction) {
		t.Fatal(err)
	}
}
