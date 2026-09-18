package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type stubFCFPerShareValuationRepository struct {
	stock           repository.Stock
	fundamentals    repository.StockFundamentals
	bondYield       repository.MasterData
	masterDataKey   string
	stockBeta       repository.StockBeta
	stockErr        error
	fundamentalsErr error
	masterDataErr   error
	stockBetaErr    error
}

func (r *stubFCFPerShareValuationRepository) GetStock(context.Context, string) (repository.Stock, error) {
	return r.stock, r.stockErr
}

func (r *stubFCFPerShareValuationRepository) GetMasterData(_ context.Context, key string) (repository.MasterData, error) {
	r.masterDataKey = key
	return r.bondYield, r.masterDataErr
}

func (r *stubFCFPerShareValuationRepository) GetFundamentals(context.Context, string) (repository.StockFundamentals, error) {
	return r.fundamentals, r.fundamentalsErr
}

func (r *stubFCFPerShareValuationRepository) GetStockBeta(context.Context, string) (repository.StockBeta, error) {
	return r.stockBeta, r.stockBetaErr
}

func TestCalculateFCFPerShareUsesTTMMetricInsteadOfNegativeQuarter(t *testing.T) {
	scrapedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"perShare": map[string]any{
			"free_cashflow_per_share_ttm": map[string]any{
				"label": "Free Cashflow Per Share (TTM)",
				"value": "538.71",
			},
			"free_cashflow_per_share_quarter": map[string]any{
				"label": "Free Cashflow Per Share (Quarter)",
				"value": "-10.00",
			},
			"current_eps_ttm": map[string]any{
				"label": "Current EPS (TTM)",
				"value": "500",
			},
		},
		"managementEffectiveness": map[string]any{
			"return_on_equity_ttm": map[string]any{
				"label": "Return on Equity (TTM)",
				"value": "20%",
			},
		},
		"dividendHistory": []any{
			map[string]any{"exDate": "01 Jan 26", "dividend": "100"},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	repo := &stubFCFPerShareValuationRepository{
		stock:        repository.Stock{Ticker: "BBCA", Active: true},
		fundamentals: repository.StockFundamentals{Ticker: "BBCA", Payload: payload, ScrapedAt: scrapedAt},
		bondYield:    repository.MasterData{Key: MasterDataKeyIndonesia10YearBondYield, Value: 6},
		stockBeta:    repository.StockBeta{Ticker: "BBCA", Value: 1.2, Period: "5y", Interval: "1mo", Source: "yahoo_finance"},
	}
	valuationService := NewFCFPerShareValuationService(repo)

	valuation, err := valuationService.Calculate(context.Background(), "BBCA", FCFPerShareValuationAssumptions{
		ForecastYears:      5,
		TerminalGrowthRate: 0.03,
		EquityRiskPremium:  0.06,
	})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if valuation.FCFPerShareTTM != 538.71 {
		t.Fatalf("FCFPerShareTTM = %v, want 538.71", valuation.FCFPerShareTTM)
	}
	if valuation.RiskFreeRate != 0.06 {
		t.Fatalf("RiskFreeRate = %v, want 0.06", valuation.RiskFreeRate)
	}
	if repo.masterDataKey != MasterDataKeyIndonesia10YearBondYield {
		t.Fatalf("master data key = %q, want %q", repo.masterDataKey, MasterDataKeyIndonesia10YearBondYield)
	}
	if valuation.Beta != 1.2 {
		t.Fatalf("Beta = %v, want 1.2", valuation.Beta)
	}
	if valuation.CostOfEquity != 0.132 {
		t.Fatalf("CostOfEquity = %v, want 0.132", valuation.CostOfEquity)
	}
}
