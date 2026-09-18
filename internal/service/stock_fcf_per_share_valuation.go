package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

var ErrInvalidFCFPerShareAssumptions = errors.New("invalid fcf per share valuation assumptions")

type FCFPerShareValuationRepository interface {
	GetStock(context.Context, string) (repository.Stock, error)
	GetMasterData(context.Context, string) (repository.MasterData, error)
	GetFundamentals(context.Context, string) (repository.StockFundamentals, error)
	GetStockBeta(context.Context, string) (repository.StockBeta, error)
}

type FCFPerShareValuationAssumptions struct {
	GrowthRate         *float64
	ForecastYears      int
	TerminalGrowthRate float64
	EquityRiskPremium  float64
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
type FCFPerShareValuationService struct {
	repository FCFPerShareValuationRepository
}

func NewFCFPerShareValuationService(repo FCFPerShareValuationRepository) *FCFPerShareValuationService {
	return &FCFPerShareValuationService{repository: repo}
}

func (s *FCFPerShareValuationService) Calculate(ctx context.Context, ticker string, in FCFPerShareValuationAssumptions) (FCFPerShareValuation, error) {
	stock, err := s.repository.GetStock(ctx, strings.ToUpper(strings.TrimSpace(ticker)))
	if errors.Is(err, repository.ErrStockNotFound) {
		return FCFPerShareValuation{}, ErrStockNotFound
	}
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("fcfPerShareValuationService.Calculate -> GetStock: %w", err)
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
	bondYield, err := s.repository.GetMasterData(ctx, MasterDataKeyIndonesia10YearBondYield)
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("%w: %s is unavailable", ErrInvalidFCFPerShareAssumptions, MasterDataKeyIndonesia10YearBondYield)
	}
	riskFreeRate, err := percentagePointsToDecimal(bondYield.Value, MasterDataKeyIndonesia10YearBondYield)
	if err != nil {
		return FCFPerShareValuation{}, err
	}
	stockBeta, err := s.repository.GetStockBeta(ctx, stock.Ticker)
	if errors.Is(err, repository.ErrStockBetaNotFound) {
		return FCFPerShareValuation{}, fmt.Errorf("%w: stock beta is unavailable", ErrInvalidFCFPerShareAssumptions)
	}
	if err != nil {
		return FCFPerShareValuation{}, fmt.Errorf("fcfPerShareValuationService.Calculate -> GetStockBeta: %w", err)
	}
	beta := stockBeta.Value
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
	if err := validateFCFPerShare(in, *growth, years, riskFreeRate, beta); err != nil {
		return FCFPerShareValuation{}, err
	}
	ke := riskFreeRate + beta*in.EquityRiskPremium
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
	return FCFPerShareValuation{Ticker: stock.Ticker, Currency: "IDR", FCFPerShareTTM: metrics.FCFPerShareTTM, GrowthRate: *growth, GrowthRateSource: source, GrowthRateDerivation: derivation, RiskFreeRate: riskFreeRate, Beta: beta, CostOfEquity: ke, Forecast: forecast, TerminalValue: terminalValue, PresentTerminalValue: presentTerminalValue, FairValuePerShare: fairValue}, nil
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
	fcfPerShareTTM, found := findMetricByKeys(perShare,
		"free_cashflow_per_share_ttm",
		"free_cash_flow_per_share_ttm",
	)
	if !found || fcfPerShareTTM <= 0 {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: positive free cash flow per share ttm is missing", ErrInvalidFCFPerShareAssumptions)
	}
	eps, found := findMetric(perShare, func(l string) bool { return strings.Contains(l, "current eps") && strings.Contains(l, "ttm") })
	if !found || eps <= 0 {
		return fcfPerShareMetrics{}, fmt.Errorf("%w: positive current eps ttm is missing", ErrInvalidFCFPerShareAssumptions)
	}
	roe, found := findMetricByKeys(data["managementEffectiveness"],
		"return_on_equity_ttm",
		"return_on_equity",
		"roe_ttm",
		"roe",
	)
	if !found {
		// Older snapshots may have grouped ROE under profitability.
		roe, found = findMetricByKeys(data["profitability"],
			"return_on_equity_ttm",
			"return_on_equity",
			"roe_ttm",
			"roe",
		)
	}
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
		return fcfPerShareMetrics{}, fmt.Errorf("%w: %v", ErrInvalidFCFPerShareAssumptions, err)
	}
	return fcfPerShareMetrics{fcfPerShareTTM, roe, eps, dps}, nil
}
func validateFCFPerShare(in FCFPerShareValuationAssumptions, growth float64, years int, riskFreeRate, beta float64) error {
	if years < 1 || years > 10 {
		return fmt.Errorf("%w: forecast_years must be between 1 and 10", ErrInvalidFCFPerShareAssumptions)
	}
	if in.TerminalGrowthRate < 0 {
		return fmt.Errorf("%w: terminal_growth_rate must not be negative", ErrInvalidFCFPerShareAssumptions)
	}
	if in.EquityRiskPremium < 0 {
		return fmt.Errorf("%w: equity_risk_premium must not be negative", ErrInvalidFCFPerShareAssumptions)
	}
	if growth <= -1 {
		return fmt.Errorf("%w: growth_rate must be greater than -1", ErrInvalidFCFPerShareAssumptions)
	}
	if riskFreeRate <= 0 || riskFreeRate >= 1 {
		return fmt.Errorf("%w: normalized %s must be between 0 and 1", ErrInvalidFCFPerShareAssumptions, MasterDataKeyIndonesia10YearBondYield)
	}
	if math.IsNaN(beta) || math.IsInf(beta, 0) {
		return fmt.Errorf("%w: stored stock beta must be finite", ErrInvalidFCFPerShareAssumptions)
	}
	if math.IsNaN(growth) || math.IsInf(growth, 0) || math.IsNaN(in.TerminalGrowthRate) || math.IsInf(in.TerminalGrowthRate, 0) || math.IsNaN(in.EquityRiskPremium) || math.IsInf(in.EquityRiskPremium, 0) {
		return fmt.Errorf("%w: values must be finite", ErrInvalidFCFPerShareAssumptions)
	}
	if in.TerminalGrowthRate >= riskFreeRate+beta*in.EquityRiskPremium {
		return fmt.Errorf("%w: terminal_growth_rate must be less than cost_of_equity", ErrInvalidFCFPerShareAssumptions)
	}
	return nil
}
func percentagePointsToDecimal(value float64, field string) (float64, error) {
	if value <= 0 || value >= 100 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%w: %s must be between 0 and 100 percent", ErrInvalidFCFPerShareAssumptions, field)
	}
	return value / 100, nil
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
