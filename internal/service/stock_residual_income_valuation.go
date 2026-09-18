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

var ErrInvalidResidualIncomeAssumptions = errors.New("invalid residual income valuation assumptions")

type ResidualIncomeValuationRepository interface {
	GetStock(context.Context, string) (repository.Stock, error)
	GetMasterData(context.Context, string) (repository.MasterData, error)
	GetFundamentals(context.Context, string) (repository.StockFundamentals, error)
	GetStockBeta(context.Context, string) (repository.StockBeta, error)
}

type ResidualIncomeValuationAssumptions struct {
	ForecastYears      int
	TerminalROE        float64
	TerminalGrowthRate float64
	EquityRiskPremium  float64
}

type ResidualIncomeForecast struct {
	Year                   int     `json:"year"`
	OpeningBookValue       float64 `json:"opening_book_value_per_share"`
	ROE                    float64 `json:"roe"`
	EarningsPerShare       float64 `json:"earnings_per_share"`
	DividendPayoutRatio    float64 `json:"dividend_payout_ratio"`
	DividendPerShare       float64 `json:"dividend_per_share"`
	ClosingBookValue       float64 `json:"closing_book_value_per_share"`
	ResidualIncomePerShare float64 `json:"residual_income_per_share"`
	PresentValue           float64 `json:"present_value"`
}

type ResidualIncomeValuation struct {
	Ticker                               string
	Currency                             string
	BookValuePerShare                    float64
	CurrentROE                           float64
	CurrentDividendPayoutRatio           float64
	RiskFreeRate                         float64
	Beta                                 float64
	CostOfEquity                         float64
	ForecastYears                        int
	TerminalROE                          float64
	TerminalGrowthRate                   float64
	TerminalRetentionRatio               float64
	Forecast                             []ResidualIncomeForecast
	PresentValueOfForecastResidualIncome float64
	TerminalResidualIncome               float64
	TerminalValue                        float64
	PresentTerminalValue                 float64
	FairValuePerShare                    float64
}

type ResidualIncomeValuationService struct {
	repository ResidualIncomeValuationRepository
}

func NewResidualIncomeValuationService(repo ResidualIncomeValuationRepository) *ResidualIncomeValuationService {
	return &ResidualIncomeValuationService{repository: repo}
}

func (s *ResidualIncomeValuationService) Calculate(ctx context.Context, ticker string, in ResidualIncomeValuationAssumptions) (ResidualIncomeValuation, error) {
	stock, err := s.repository.GetStock(ctx, strings.ToUpper(strings.TrimSpace(ticker)))
	if errors.Is(err, repository.ErrStockNotFound) {
		return ResidualIncomeValuation{}, ErrStockNotFound
	}
	if err != nil {
		return ResidualIncomeValuation{}, fmt.Errorf("residualIncomeValuationService.Calculate -> GetStock: %w", err)
	}
	if !stock.Active {
		return ResidualIncomeValuation{}, ErrInactiveStock
	}

	fundamentals, err := s.repository.GetFundamentals(ctx, stock.Ticker)
	if errors.Is(err, repository.ErrStockNotFound) {
		return ResidualIncomeValuation{}, fmt.Errorf("%w: stored fundamentals are unavailable", ErrInvalidResidualIncomeAssumptions)
	}
	if err != nil {
		return ResidualIncomeValuation{}, fmt.Errorf("residualIncomeValuationService.Calculate -> GetFundamentals: %w", err)
	}
	metrics, err := readResidualIncomeMetrics(fundamentals.Payload, fundamentals.ScrapedAt)
	if err != nil {
		return ResidualIncomeValuation{}, err
	}

	bondYield, err := s.repository.GetMasterData(ctx, MasterDataKeyIndonesia10YearBondYield)
	if errors.Is(err, repository.ErrMasterDataNotFound) {
		return ResidualIncomeValuation{}, fmt.Errorf("%w: %s is unavailable", ErrInvalidResidualIncomeAssumptions, MasterDataKeyIndonesia10YearBondYield)
	}
	if err != nil {
		return ResidualIncomeValuation{}, fmt.Errorf("residualIncomeValuationService.Calculate -> GetMasterData: %w", err)
	}
	riskFreeRate, err := residualIncomePercentagePointsToDecimal(bondYield.Value, MasterDataKeyIndonesia10YearBondYield)
	if err != nil {
		return ResidualIncomeValuation{}, err
	}
	stockBeta, err := s.repository.GetStockBeta(ctx, stock.Ticker)
	if errors.Is(err, repository.ErrStockBetaNotFound) {
		return ResidualIncomeValuation{}, fmt.Errorf("%w: stock beta is unavailable", ErrInvalidResidualIncomeAssumptions)
	}
	if err != nil {
		return ResidualIncomeValuation{}, fmt.Errorf("residualIncomeValuationService.Calculate -> GetStockBeta: %w", err)
	}

	years := in.ForecastYears
	if years == 0 {
		years = 5
	}
	beta := stockBeta.Value
	currentPayout := metrics.DividendPerShareTTM / metrics.EPSTTM
	if err := validateResidualIncome(in, metrics, currentPayout, years, riskFreeRate, beta); err != nil {
		return ResidualIncomeValuation{}, err
	}

	costOfEquity := riskFreeRate + beta*in.EquityRiskPremium
	terminalRetention := in.TerminalGrowthRate / in.TerminalROE
	terminalPayout := 1 - terminalRetention
	openingBookValue := metrics.BookValuePerShare
	forecast := make([]ResidualIncomeForecast, 0, years)
	presentForecastResidualIncome := 0.0

	for year := 1; year <= years; year++ {
		progress := float64(year) / float64(years)
		roe := metrics.ROE + (in.TerminalROE-metrics.ROE)*progress
		payout := currentPayout + (terminalPayout-currentPayout)*progress
		earnings := roe * openingBookValue
		dividend := payout * earnings
		closingBookValue := openingBookValue + earnings - dividend
		if closingBookValue <= 0 || !isFinite(closingBookValue) {
			return ResidualIncomeValuation{}, fmt.Errorf("%w: projected book value per share must remain positive and finite", ErrInvalidResidualIncomeAssumptions)
		}
		residualIncome := (roe - costOfEquity) * openingBookValue
		presentValue := residualIncome / math.Pow(1+costOfEquity, float64(year))
		forecast = append(forecast, ResidualIncomeForecast{
			Year:                   year,
			OpeningBookValue:       openingBookValue,
			ROE:                    roe,
			EarningsPerShare:       earnings,
			DividendPayoutRatio:    payout,
			DividendPerShare:       dividend,
			ClosingBookValue:       closingBookValue,
			ResidualIncomePerShare: residualIncome,
			PresentValue:           presentValue,
		})
		presentForecastResidualIncome += presentValue
		openingBookValue = closingBookValue
	}

	terminalResidualIncome := (in.TerminalROE - costOfEquity) * openingBookValue
	terminalValue := terminalResidualIncome / (costOfEquity - in.TerminalGrowthRate)
	presentTerminalValue := terminalValue / math.Pow(1+costOfEquity, float64(years))
	fairValue := metrics.BookValuePerShare + presentForecastResidualIncome + presentTerminalValue

	return ResidualIncomeValuation{
		Ticker:                               stock.Ticker,
		Currency:                             "IDR",
		BookValuePerShare:                    metrics.BookValuePerShare,
		CurrentROE:                           metrics.ROE,
		CurrentDividendPayoutRatio:           currentPayout,
		RiskFreeRate:                         riskFreeRate,
		Beta:                                 beta,
		CostOfEquity:                         costOfEquity,
		ForecastYears:                        years,
		TerminalROE:                          in.TerminalROE,
		TerminalGrowthRate:                   in.TerminalGrowthRate,
		TerminalRetentionRatio:               terminalRetention,
		Forecast:                             forecast,
		PresentValueOfForecastResidualIncome: presentForecastResidualIncome,
		TerminalResidualIncome:               terminalResidualIncome,
		TerminalValue:                        terminalValue,
		PresentTerminalValue:                 presentTerminalValue,
		FairValuePerShare:                    fairValue,
	}, nil
}

type residualIncomeMetrics struct {
	BookValuePerShare   float64
	ROE                 float64
	EPSTTM              float64
	DividendPerShareTTM float64
}

func readResidualIncomeMetrics(payload json.RawMessage, scrapedAt time.Time) (residualIncomeMetrics, error) {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return residualIncomeMetrics{}, fmt.Errorf("%w: stored fundamentals payload is invalid", ErrInvalidResidualIncomeAssumptions)
	}
	perShare, ok := data["perShare"]
	if !ok {
		return residualIncomeMetrics{}, fmt.Errorf("%w: per-share metrics are missing", ErrInvalidResidualIncomeAssumptions)
	}
	bookValue, found := findMetricByKeys(perShare, "current_book_value_per_share", "book_value_per_share")
	if !found || bookValue <= 0 {
		return residualIncomeMetrics{}, fmt.Errorf("%w: positive current book value per share is missing", ErrInvalidResidualIncomeAssumptions)
	}
	eps, found := findMetricByKeys(perShare, "current_eps_ttm", "eps_ttm")
	if !found || eps <= 0 {
		return residualIncomeMetrics{}, fmt.Errorf("%w: positive current eps ttm is missing", ErrInvalidResidualIncomeAssumptions)
	}
	roe, found := findPercentageMetricByKeys(data["managementEffectiveness"], "return_on_equity_ttm", "return_on_equity", "roe_ttm", "roe")
	if !found {
		roe, found = findPercentageMetricByKeys(data["profitability"], "return_on_equity_ttm", "return_on_equity", "roe_ttm", "roe")
	}
	if !found {
		return residualIncomeMetrics{}, fmt.Errorf("%w: return on equity is missing", ErrInvalidResidualIncomeAssumptions)
	}
	history, ok := data["dividendHistory"].([]any)
	if !ok {
		return residualIncomeMetrics{}, fmt.Errorf("%w: dividend history is missing", ErrInvalidResidualIncomeAssumptions)
	}
	dps, err := dividendPerShareTTM(history, scrapedAt)
	if err != nil {
		return residualIncomeMetrics{}, fmt.Errorf("%w: %v", ErrInvalidResidualIncomeAssumptions, err)
	}
	return residualIncomeMetrics{BookValuePerShare: bookValue, ROE: roe, EPSTTM: eps, DividendPerShareTTM: dps}, nil
}

func findPercentageMetricByKeys(value any, keys ...string) (float64, bool) {
	metrics, ok := value.(map[string]any)
	if !ok {
		return 0, false
	}
	for _, key := range keys {
		metric, ok := metrics[key].(map[string]any)
		if !ok {
			continue
		}
		raw, ok := metric["value"].(string)
		if !ok {
			continue
		}
		parsed, err := parseNumber(raw)
		if err != nil {
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(raw), "%") {
			parsed /= 100
		}
		return parsed, true
	}
	return 0, false
}

func validateResidualIncome(in ResidualIncomeValuationAssumptions, metrics residualIncomeMetrics, currentPayout float64, years int, riskFreeRate, beta float64) error {
	if years < 1 || years > 10 {
		return fmt.Errorf("%w: forecast_years must be between 1 and 10", ErrInvalidResidualIncomeAssumptions)
	}
	if !isFinite(in.TerminalROE) || in.TerminalROE <= 0 || in.TerminalROE > 1 {
		return fmt.Errorf("%w: terminal_roe must be greater than 0 and at most 1", ErrInvalidResidualIncomeAssumptions)
	}
	if !isFinite(in.TerminalGrowthRate) || in.TerminalGrowthRate < 0 {
		return fmt.Errorf("%w: terminal_growth_rate must be a non-negative finite decimal", ErrInvalidResidualIncomeAssumptions)
	}
	if in.TerminalGrowthRate > in.TerminalROE {
		return fmt.Errorf("%w: terminal_growth_rate must not exceed terminal_roe", ErrInvalidResidualIncomeAssumptions)
	}
	if !isFinite(in.EquityRiskPremium) || in.EquityRiskPremium < 0 {
		return fmt.Errorf("%w: equity_risk_premium must be a non-negative finite decimal", ErrInvalidResidualIncomeAssumptions)
	}
	if !isFinite(metrics.ROE) || !isFinite(currentPayout) || currentPayout < 0 {
		return fmt.Errorf("%w: stored roe and dividend payout ratio must be finite, and payout must not be negative", ErrInvalidResidualIncomeAssumptions)
	}
	if !isFinite(beta) {
		return fmt.Errorf("%w: stored stock beta must be finite", ErrInvalidResidualIncomeAssumptions)
	}
	costOfEquity := riskFreeRate + beta*in.EquityRiskPremium
	if !isFinite(costOfEquity) || costOfEquity <= in.TerminalGrowthRate {
		return fmt.Errorf("%w: terminal_growth_rate must be less than cost_of_equity", ErrInvalidResidualIncomeAssumptions)
	}
	return nil
}

func residualIncomePercentagePointsToDecimal(value float64, field string) (float64, error) {
	if value <= 0 || value >= 100 || !isFinite(value) {
		return 0, fmt.Errorf("%w: %s must be between 0 and 100 percent", ErrInvalidResidualIncomeAssumptions, field)
	}
	return value / 100, nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
