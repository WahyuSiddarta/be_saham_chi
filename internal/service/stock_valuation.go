package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

const defaultFCFPerShareBeta = 0.8

var ErrInvalidFCFPerShareAssumptions = errors.New("invalid fcf per share valuation assumptions")

type FCFPerShareValuationRepository interface {
	GetStock(context.Context, string) (repository.Stock, error)
	GetMasterData(context.Context, string) (repository.MasterData, error)
	GetFundamentals(context.Context, string) (repository.StockFundamentals, error)
}

type FCFPerShareValuationAssumptions struct {
	GrowthRate         *float64
	ForecastYears      int
	TerminalGrowthRate float64
	EquityRiskPremium  float64
	Beta               *float64
}

type FCFPerShareForecast struct {
	Year         int     `json:"year"`
	FCFPerShare  float64 `json:"fcf_per_share"`
	PresentValue float64 `json:"present_value"`
}
type FCFPerShareGrowthRateDerivation struct {
	ROE                 float64 `json:"roe"`
	DividendPerShareTTM float64 `json:"dividend_per_share_ttm"`
	EPSTTM              float64 `json:"eps_ttm"`
	DividendPayoutRatio float64 `json:"dividend_payout_ratio"`
}
type FCFPerShareValuation struct {
	Ticker               string
	Currency             string
	FCFPerShareTTM       float64
	GrowthRate           float64
	GrowthRateSource     string
	GrowthRateDerivation *FCFPerShareGrowthRateDerivation
	RiskFreeRate         float64
	Beta                 float64
	CostOfEquity         float64
	Forecast             []FCFPerShareForecast
	TerminalValue        float64
	PresentTerminalValue float64
	FairValuePerShare    float64
}
type StockValuationService struct {
	repository FCFPerShareValuationRepository
}

func NewStockValuationService(repo FCFPerShareValuationRepository) *StockValuationService {
	return &StockValuationService{repository: repo}
}

func (s *StockValuationService) CalculateFCFPerShare(ctx context.Context, ticker string, in FCFPerShareValuationAssumptions) (FCFPerShareValuation, error) {
	stock, err := s.repository.GetStock(ctx, strings.ToUpper(strings.TrimSpace(ticker)))
	if errors.Is(err, repository.ErrStockNotFound) {
		return FCFPerShareValuation{}, ErrStockNotFound
	}
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("stockValuationService.CalculateFCFPerShare -> GetStock: %w", err)
	}
	if !stock.Active {
		return FCFPerShareValuation{}, ErrInactiveStock
	}
	fundamentals, err := s.repository.GetFundamentals(ctx, stock.Ticker)
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("%w: stored fundamentals are unavailable", ErrInvalidFCFPerShareAssumptions)
	}
	metrics, err := readFCFPerShareMetrics(fundamentals.Payload, fundamentals.ScrapedAt)
	if err != nil {
		return FCFPerShareValuation{}, err
	}
	biRate, err := s.repository.GetMasterData(ctx, "bi_rate")
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("%w: bi_rate is unavailable", ErrInvalidFCFPerShareAssumptions)
	}
	years := in.ForecastYears
	if years == 0 {
		years = 5
	}
	growth := in.GrowthRate
	source := "request"
	var derivation *FCFPerShareGrowthRateDerivation
	if growth == nil {
		derived := metrics.ROE * (1 - metrics.DividendPerShareTTM/metrics.EPSTTM)
		growth = &derived
		source = "fundamentals_roe_x_retention"
		derivation = &FCFPerShareGrowthRateDerivation{metrics.ROE, metrics.DividendPerShareTTM, metrics.EPSTTM, metrics.DividendPerShareTTM / metrics.EPSTTM}
	}
	if err := validateFCFPerShare(in, *growth, years, biRate.Value); err != nil {
		return FCFPerShareValuation{}, err
	}
	beta := defaultFCFPerShareBeta
	if in.Beta != nil {
		beta = *in.Beta
	}
	ke := biRate.Value + beta*in.EquityRiskPremium
	forecast := make([]FCFPerShareForecast, 0, years)
	fairValue := 0.0
	for year := 1; year <= years; year++ {
		fcf := metrics.FCFPerShareTTM * math.Pow(1+*growth, float64(year))
		pv := fcf / math.Pow(1+ke, float64(year))
		forecast = append(forecast, FCFPerShareForecast{year, fcf, pv})
		fairValue += pv
	}
	terminalValue := forecast[len(forecast)-1].FCFPerShare * (1 + in.TerminalGrowthRate) / (ke - in.TerminalGrowthRate)
	presentTerminalValue := terminalValue / math.Pow(1+ke, float64(years))
	fairValue += presentTerminalValue
	return FCFPerShareValuation{Ticker: stock.Ticker, Currency: "IDR", FCFPerShareTTM: metrics.FCFPerShareTTM, GrowthRate: *growth, GrowthRateSource: source, GrowthRateDerivation: derivation, RiskFreeRate: biRate.Value, Beta: beta, CostOfEquity: ke, Forecast: forecast, TerminalValue: terminalValue, PresentTerminalValue: presentTerminalValue, FairValuePerShare: fairValue}, nil
}

type fcfPerShareMetrics struct {
	FCFPerShareTTM      float64
	ROE                 float64
	EPSTTM              float64
	DividendPerShareTTM float64
}

func readFCFPerShareMetrics(payload json.RawMessage, scrapedAt time.Time) (fcfPerShareMetrics, error) {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: stored fundamentals payload is invalid", ErrInvalidFCFPerShareAssumptions)
	}
	perShare, ok := data["perShare"]
	if !ok {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: per-share metrics are missing", ErrInvalidFCFPerShareAssumptions)
	}
	fcf, found := findMetric(perShare, func(l string) bool {
		return strings.Contains(l, "free cash flow") && strings.Contains(l, "per share") && strings.Contains(l, "ttm")
	})
	if !found || fcf <= 0 {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: positive free cash flow per share ttm is missing", ErrInvalidFCFPerShareAssumptions)
	}
	eps, found := findMetric(perShare, func(l string) bool { return strings.Contains(l, "current eps") && strings.Contains(l, "ttm") })
	if !found || eps <= 0 {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: positive current eps ttm is missing", ErrInvalidFCFPerShareAssumptions)
	}
	roe, found := findMetric(data["profitability"], func(l string) bool { return strings.Contains(l, "return on equity") || strings.HasPrefix(l, "roe") })
	if !found || roe <= 0 {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: positive return on equity is missing", ErrInvalidFCFPerShareAssumptions)
	}
	if roe > 1 {
		roe /= 100
	}
	history, ok := data["dividendHistory"].([]any)
	if !ok {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: dividend history is missing", ErrInvalidFCFPerShareAssumptions)
	}
	dps, err := dividendPerShareTTM(history, scrapedAt)
	if err != nil {
		return fcfPerShareMetrics{}, err
	}
	return fcfPerShareMetrics{fcf, roe, eps, dps}, nil
}
func validateFCFPerShare(in FCFPerShareValuationAssumptions, growth float64, years int, bi float64) error {
	if years < 1 || years > 10 || in.TerminalGrowthRate < 0 || in.EquityRiskPremium < 0 || growth <= -1 || bi <= 0 || bi >= 1 {
		return fmt.Errorf("%w: years must be 1-10 and rates must be decimal values", ErrInvalidFCFPerShareAssumptions)
	}
	beta := defaultFCFPerShareBeta
	if in.Beta != nil {
		beta = *in.Beta
	}
	if beta <= 0 || math.IsNaN(beta) || math.IsInf(beta, 0) {
		return fmt.Errorf("%w: beta must be positive", ErrInvalidFCFPerShareAssumptions)
	}
	if math.IsNaN(growth) || math.IsInf(growth, 0) || math.IsNaN(in.TerminalGrowthRate) || math.IsInf(in.TerminalGrowthRate, 0) || math.IsNaN(in.EquityRiskPremium) || math.IsInf(in.EquityRiskPremium, 0) {
		return fmt.Errorf("%w: values must be finite", ErrInvalidFCFPerShareAssumptions)
	}
	if in.TerminalGrowthRate >= bi+beta*in.EquityRiskPremium {
		return fmt.Errorf("%w: terminal_growth_rate must be less than cost_of_equity", ErrInvalidFCFPerShareAssumptions)
	}
	return nil
}
func findMetric(value any, matches func(string) bool) (float64, bool) {
	switch item := value.(type) {
	case map[string]any:
		if label, ok := item["label"].(string); ok && matches(strings.ToLower(label)) {
			raw, ok := item["value"].(string)
			if !ok {
				return 0, false
			}
			parsed, err := parseNumber(raw)
			return parsed, err == nil
		}
		for _, child := range item {
			if v, ok := findMetric(child, matches); ok {
				return v, true
			}
		}
	case []any:
		for _, child := range item {
			if v, ok := findMetric(child, matches); ok {
				return v, true
			}
		}
	}
	return 0, false
}
func dividendPerShareTTM(history []any, asOf time.Time) (float64, error) {
	cutoff := asOf.AddDate(-1, 0, 0)
	total := 0.0
	count := 0
	for _, raw := range history {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		dateRaw, ok := item["exDate"].(string)
		if !ok {
			continue
		}
		date, err := time.Parse("02 Jan 06", dateRaw)
		if err != nil || date.Before(cutoff) || date.After(asOf) {
			continue
		}
		dividend, ok := item["dividend"].(string)
		if !ok {
			continue
		}
		amount, err := parseNumber(dividend)
		if err != nil {
			continue
		}
		total += amount
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("%w: no parseable TTM dividends", ErrInvalidFCFPerShareAssumptions)
	}
	return total, nil
}
func parseNumber(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(strings.NewReplacer(",", "", "%", "", "Rp", "", "IDR", "").Replace(raw)), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("invalid number")
	}
	return value, nil
}
