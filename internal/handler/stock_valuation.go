package handler

import (
	"errors"
	"net/http"

	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
	"github.com/go-chi/chi/v5"
)

type FCFPerShareValuationRequest struct {
	GrowthRate         *float64 `json:"growth_rate"`
	ForecastYears      int      `json:"forecast_years"`
	TerminalGrowthRate float64  `json:"terminal_growth_rate"`
	EquityRiskPremium  float64  `json:"equity_risk_premium"`
}
type FCFPerShareValuationResponse struct {
	Ticker               string                                   `json:"ticker"`
	Currency             string                                   `json:"currency"`
	FCFPerShareTTM       float64                                  `json:"fcf_per_share_ttm"`
	GrowthRate           float64                                  `json:"growth_rate"`
	GrowthRateSource     string                                   `json:"growth_rate_source"`
	GrowthRateDerivation *service.FCFPerShareGrowthRateDerivation `json:"growth_rate_derivation"`
	RiskFreeRate         float64                                  `json:"risk_free_rate"`
	Beta                 float64                                  `json:"beta"`
	CostOfEquity         float64                                  `json:"cost_of_equity"`
	Forecast             []service.FCFPerShareForecast            `json:"forecast"`
	TerminalValue        float64                                  `json:"terminal_value"`
	PresentTerminalValue float64                                  `json:"present_terminal_value"`
	FairValuePerShare    float64                                  `json:"fair_value_per_share"`
}

type ResidualIncomeValuationRequest struct {
	ForecastYears      int     `json:"forecast_years"`
	TerminalROE        float64 `json:"terminal_roe"`
	TerminalGrowthRate float64 `json:"terminal_growth_rate"`
	EquityRiskPremium  float64 `json:"equity_risk_premium"`
}

type ResidualIncomeValuationResponse struct {
	Ticker                               string                           `json:"ticker"`
	Currency                             string                           `json:"currency"`
	BookValuePerShare                    float64                          `json:"book_value_per_share"`
	CurrentROE                           float64                          `json:"current_roe"`
	CurrentDividendPayoutRatio           float64                          `json:"current_dividend_payout_ratio"`
	RiskFreeRate                         float64                          `json:"risk_free_rate"`
	Beta                                 float64                          `json:"beta"`
	CostOfEquity                         float64                          `json:"cost_of_equity"`
	ForecastYears                        int                              `json:"forecast_years"`
	TerminalROE                          float64                          `json:"terminal_roe"`
	TerminalGrowthRate                   float64                          `json:"terminal_growth_rate"`
	TerminalRetentionRatio               float64                          `json:"terminal_retention_ratio"`
	Forecast                             []service.ResidualIncomeForecast `json:"forecast"`
	PresentValueOfForecastResidualIncome float64                          `json:"present_value_of_forecast_residual_income"`
	TerminalResidualIncome               float64                          `json:"terminal_residual_income"`
	TerminalValue                        float64                          `json:"terminal_value"`
	PresentTerminalValue                 float64                          `json:"present_terminal_value"`
	FairValuePerShare                    float64                          `json:"fair_value_per_share"`
}

func (h Handler) CalculateStockFCFPerShare(w http.ResponseWriter, req *http.Request) error {
	var body FCFPerShareValuationRequest
	if err := binding.BindJSON(req.Body, &body); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	valuation, err := h.fcfPerShareValuationService.Calculate(req.Context(), chi.URLParam(req, "ticker"), service.FCFPerShareValuationAssumptions{GrowthRate: body.GrowthRate, ForecastYears: body.ForecastYears, TerminalGrowthRate: body.TerminalGrowthRate, EquityRiskPremium: body.EquityRiskPremium})
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			return response.Fail(w, http.StatusNotFound, "stock not found")
		}
		if errors.Is(err, service.ErrInvalidFCFPerShareAssumptions) {
			return response.Fail(w, http.StatusBadRequest, err.Error())
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to calculate stock fcf per share valuation", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to calculate stock fcf per share valuation")
	}
	return response.Success(w, http.StatusOK, FCFPerShareValuationResponse{Ticker: valuation.Ticker, Currency: valuation.Currency, FCFPerShareTTM: valuation.FCFPerShareTTM, GrowthRate: valuation.GrowthRate, GrowthRateSource: valuation.GrowthRateSource, GrowthRateDerivation: valuation.GrowthRateDerivation, RiskFreeRate: valuation.RiskFreeRate, Beta: valuation.Beta, CostOfEquity: valuation.CostOfEquity, Forecast: valuation.Forecast, TerminalValue: valuation.TerminalValue, PresentTerminalValue: valuation.PresentTerminalValue, FairValuePerShare: valuation.FairValuePerShare})
}

func (h Handler) CalculateStockResidualIncome(w http.ResponseWriter, req *http.Request) error {
	var body ResidualIncomeValuationRequest
	if err := binding.BindJSON(req.Body, &body); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	valuation, err := h.residualIncomeValuationService.Calculate(req.Context(), chi.URLParam(req, "ticker"), service.ResidualIncomeValuationAssumptions{
		ForecastYears:      body.ForecastYears,
		TerminalROE:        body.TerminalROE,
		TerminalGrowthRate: body.TerminalGrowthRate,
		EquityRiskPremium:  body.EquityRiskPremium,
	})
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			return response.Fail(w, http.StatusNotFound, "stock not found")
		}
		if errors.Is(err, service.ErrInvalidResidualIncomeAssumptions) {
			return response.Fail(w, http.StatusBadRequest, err.Error())
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to calculate stock residual income valuation", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to calculate stock residual income valuation")
	}
	return response.Success(w, http.StatusOK, ResidualIncomeValuationResponse{
		Ticker:                               valuation.Ticker,
		Currency:                             valuation.Currency,
		BookValuePerShare:                    valuation.BookValuePerShare,
		CurrentROE:                           valuation.CurrentROE,
		CurrentDividendPayoutRatio:           valuation.CurrentDividendPayoutRatio,
		RiskFreeRate:                         valuation.RiskFreeRate,
		Beta:                                 valuation.Beta,
		CostOfEquity:                         valuation.CostOfEquity,
		ForecastYears:                        valuation.ForecastYears,
		TerminalROE:                          valuation.TerminalROE,
		TerminalGrowthRate:                   valuation.TerminalGrowthRate,
		TerminalRetentionRatio:               valuation.TerminalRetentionRatio,
		Forecast:                             valuation.Forecast,
		PresentValueOfForecastResidualIncome: valuation.PresentValueOfForecastResidualIncome,
		TerminalResidualIncome:               valuation.TerminalResidualIncome,
		TerminalValue:                        valuation.TerminalValue,
		PresentTerminalValue:                 valuation.PresentTerminalValue,
		FairValuePerShare:                    valuation.FairValuePerShare,
	})
}
