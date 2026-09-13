package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

func TestCalculateResidualIncomeRollsBookValueAndPreservesStoredBeta(t *testing.T) {
	scrapedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"perShare": map[string]any{
			"current_book_value_per_share": map[string]any{"label": "Current Book Value Per Share", "value": "2,193.77"},
			"current_eps_ttm":              map[string]any{"label": "Current EPS (TTM)", "value": "500"},
		},
		"managementEffectiveness": map[string]any{
			"return_on_equity_ttm": map[string]any{"label": "Return on Equity (TTM)", "value": "20%"},
		},
		"dividendHistory": []any{
			map[string]any{"exDate": "01 Jan 26", "dividend": "100"},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	valuationService := NewStockValuationService(&stubFCFPerShareValuationRepository{
		stock:        repository.Stock{Ticker: "BBCA", Active: true},
		fundamentals: repository.StockFundamentals{Ticker: "BBCA", Payload: payload, ScrapedAt: scrapedAt},
		biRate:       repository.MasterData{Key: "bi_rate", Value: 5.75},
		stockBeta:    repository.StockBeta{Ticker: "BBCA", Value: -0.04, Period: "5y", Interval: "1mo", Source: "yahoo_finance"},
	})

	valuation, err := valuationService.CalculateResidualIncome(context.Background(), "bbca", ResidualIncomeValuationAssumptions{
		ForecastYears:      5,
		TerminalROE:        0.12,
		TerminalGrowthRate: 0.03,
		EquityRiskPremium:  0.06,
	})
	if err != nil {
		t.Fatalf("CalculateResidualIncome returned error: %v", err)
	}
	if valuation.Beta != -0.04 {
		t.Fatalf("Beta = %v, want -0.04", valuation.Beta)
	}
	assertFloatClose(t, valuation.CostOfEquity, 0.0551)
	assertFloatClose(t, valuation.CurrentROE, 0.20)
	assertFloatClose(t, valuation.CurrentDividendPayoutRatio, 0.20)
	assertFloatClose(t, valuation.TerminalRetentionRatio, 0.25)
	if len(valuation.Forecast) != 5 {
		t.Fatalf("len(Forecast) = %d, want 5", len(valuation.Forecast))
	}
	assertFloatClose(t, valuation.Forecast[0].ROE, 0.184)
	assertFloatClose(t, valuation.Forecast[4].ROE, 0.12)
	assertFloatClose(t, valuation.Forecast[4].DividendPayoutRatio, 0.75)
	for i, year := range valuation.Forecast {
		if year.ClosingBookValue <= 0 {
			t.Fatalf("forecast[%d] closing book value = %v, want positive", i, year.ClosingBookValue)
		}
		if i > 0 {
			assertFloatClose(t, year.OpeningBookValue, valuation.Forecast[i-1].ClosingBookValue)
		}
	}
	assertFloatClose(t, valuation.FairValuePerShare, valuation.BookValuePerShare+valuation.PresentValueOfForecastResidualIncome+valuation.PresentTerminalValue)
}

func TestCalculateResidualIncomeRejectsInconsistentTerminalAssumptions(t *testing.T) {
	scrapedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"perShare": map[string]any{
			"current_book_value_per_share": map[string]any{"value": "1000"},
			"current_eps_ttm":              map[string]any{"value": "100"},
		},
		"managementEffectiveness": map[string]any{
			"return_on_equity_ttm": map[string]any{"value": "10%"},
		},
		"dividendHistory": []any{map[string]any{"exDate": "01 Jan 26", "dividend": "20"}},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	valuationService := NewStockValuationService(&stubFCFPerShareValuationRepository{
		stock:        repository.Stock{Ticker: "TEST", Active: true},
		fundamentals: repository.StockFundamentals{Ticker: "TEST", Payload: payload, ScrapedAt: scrapedAt},
		biRate:       repository.MasterData{Key: "bi_rate", Value: 5},
		stockBeta:    repository.StockBeta{Ticker: "TEST", Value: 1},
	})

	_, err = valuationService.CalculateResidualIncome(context.Background(), "TEST", ResidualIncomeValuationAssumptions{
		TerminalROE:        0.02,
		TerminalGrowthRate: 0.03,
		EquityRiskPremium:  0.06,
	})
	if !errors.Is(err, ErrInvalidResidualIncomeAssumptions) {
		t.Fatalf("error = %v, want ErrInvalidResidualIncomeAssumptions", err)
	}
}

func TestReadResidualIncomeMetricsUsesPercentSuffix(t *testing.T) {
	for _, tc := range []struct {
		name, roe string
		want      float64
	}{
		{name: "below one percent", roe: "0.80%", want: 0.008},
		{name: "negative below one percent", roe: "-0.50%", want: -0.005},
		{name: "bare decimal", roe: "0.80", want: 0.80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := readResidualIncomeMetrics(residualIncomePayload(t, tc.roe, []any{}), time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("readResidualIncomeMetrics returned error: %v", err)
			}
			assertFloatClose(t, metrics.ROE, tc.want)
		})
	}
}

func TestReadResidualIncomeMetricsAllowsNoTTMDividends(t *testing.T) {
	asOf := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		history []any
	}{
		{name: "empty history", history: []any{}},
		{name: "only old dividends", history: []any{map[string]any{"exDate": "01 Jan 20", "dividend": "50"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := readResidualIncomeMetrics(residualIncomePayload(t, "10%", tc.history), asOf)
			if err != nil {
				t.Fatalf("readResidualIncomeMetrics returned error: %v", err)
			}
			if metrics.DividendPerShareTTM != 0 {
				t.Fatalf("DividendPerShareTTM = %v, want 0", metrics.DividendPerShareTTM)
			}
		})
	}
}

func TestReadResidualIncomeMetricsRejectsMalformedDividendHistory(t *testing.T) {
	_, err := readResidualIncomeMetrics(
		residualIncomePayload(t, "10%", []any{map[string]any{"exDate": "not-a-date", "dividend": "20"}}),
		time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, ErrInvalidResidualIncomeAssumptions) || !strings.Contains(err.Error(), "malformed dividend history") {
		t.Fatalf("error = %v, want invalid assumptions with malformed dividend history", err)
	}
}

func TestCalculateResidualIncomeClassifiesRepositoryErrors(t *testing.T) {
	infrastructureErr := errors.New("database unavailable")
	validPayload := residualIncomePayload(t, "10%", []any{})
	base := stubFCFPerShareValuationRepository{
		stock:        repository.Stock{Ticker: "TEST", Active: true},
		fundamentals: repository.StockFundamentals{Ticker: "TEST", Payload: validPayload, ScrapedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
		biRate:       repository.MasterData{Key: "bi_rate", Value: 5},
		stockBeta:    repository.StockBeta{Ticker: "TEST", Value: 1},
	}
	assumptions := ResidualIncomeValuationAssumptions{TerminalROE: 0.10, TerminalGrowthRate: 0.03, EquityRiskPremium: 0.06}

	for _, tc := range []struct {
		name        string
		configure   func(*stubFCFPerShareValuationRepository)
		wantInvalid bool
		wantCause   error
	}{
		{name: "missing fundamentals", configure: func(r *stubFCFPerShareValuationRepository) { r.fundamentalsErr = repository.ErrStockNotFound }, wantInvalid: true},
		{name: "fundamentals infrastructure", configure: func(r *stubFCFPerShareValuationRepository) { r.fundamentalsErr = infrastructureErr }, wantCause: infrastructureErr},
		{name: "missing BI rate", configure: func(r *stubFCFPerShareValuationRepository) { r.masterDataErr = repository.ErrMasterDataNotFound }, wantInvalid: true},
		{name: "BI rate infrastructure", configure: func(r *stubFCFPerShareValuationRepository) { r.masterDataErr = infrastructureErr }, wantCause: infrastructureErr},
		{name: "missing beta", configure: func(r *stubFCFPerShareValuationRepository) { r.stockBetaErr = repository.ErrStockBetaNotFound }, wantInvalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := base
			tc.configure(&repo)
			_, err := NewStockValuationService(&repo).CalculateResidualIncome(context.Background(), "TEST", assumptions)
			if tc.wantInvalid && !errors.Is(err, ErrInvalidResidualIncomeAssumptions) {
				t.Fatalf("error = %v, want ErrInvalidResidualIncomeAssumptions", err)
			}
			if tc.wantCause != nil {
				if !errors.Is(err, tc.wantCause) {
					t.Fatalf("error = %v, want underlying cause %v", err, tc.wantCause)
				}
				if errors.Is(err, ErrInvalidResidualIncomeAssumptions) {
					t.Fatalf("infrastructure error must not be invalid assumptions: %v", err)
				}
			}
		})
	}
}

func TestCalculateResidualIncomeDefaultsForecastAndMatchesIndependentExample(t *testing.T) {
	scrapedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	repo := &stubFCFPerShareValuationRepository{
		stock:        repository.Stock{Ticker: "TEST", Active: true},
		fundamentals: repository.StockFundamentals{Ticker: "TEST", Payload: residualIncomePayload(t, "20%", []any{map[string]any{"exDate": "01 Jan 26", "dividend": "120"}}), ScrapedAt: scrapedAt},
		biRate:       repository.MasterData{Key: "bi_rate", Value: 5},
		stockBeta:    repository.StockBeta{Ticker: "TEST", Value: 1},
	}
	valuation, err := NewStockValuationService(repo).CalculateResidualIncome(context.Background(), "TEST", ResidualIncomeValuationAssumptions{
		TerminalROE:        0.10,
		TerminalGrowthRate: 0.02,
		EquityRiskPremium:  0.05,
	})
	if err != nil {
		t.Fatalf("CalculateResidualIncome returned error: %v", err)
	}
	if valuation.ForecastYears != 5 || len(valuation.Forecast) != 5 {
		t.Fatalf("forecast years = %d, len = %d; want 5", valuation.ForecastYears, len(valuation.Forecast))
	}
	// Independently calculated from BVPS 1000, EPS 100, DPS 120, ROE fading
	// 20% to 10%, payout fading 120% to 80%, and a 10% cost of equity.
	assertFloatClose(t, valuation.Forecast[0].ClosingBookValue, 978.4)
	assertFloatClose(t, valuation.Forecast[0].ResidualIncomePerShare, 80)
	assertFloatClose(t, valuation.PresentValueOfForecastResidualIncome, 163.81228525570654)
	assertFloatClose(t, valuation.PresentTerminalValue, 0)
	assertFloatClose(t, valuation.FairValuePerShare, 1163.8122852557065)
}

func residualIncomePayload(t *testing.T, roe string, history []any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"perShare": map[string]any{
			"current_book_value_per_share": map[string]any{"label": "Current Book Value Per Share", "value": "1000"},
			"current_eps_ttm":              map[string]any{"label": "Current EPS (TTM)", "value": "100"},
		},
		"managementEffectiveness": map[string]any{
			"return_on_equity_ttm": map[string]any{"label": "Return on Equity (TTM)", "value": roe},
		},
		"dividendHistory": history,
	})
	if err != nil {
		t.Fatalf("marshal residual income payload: %v", err)
	}
	return payload
}

func assertFloatClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
