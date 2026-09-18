package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type residualIncomeHandlerRepository struct {
	fundamentalsErr error
	stockBetaErr    error
}

func (r *residualIncomeHandlerRepository) GetStock(context.Context, string) (repository.Stock, error) {
	return repository.Stock{Ticker: "TEST", Active: true}, nil
}

func (r *residualIncomeHandlerRepository) GetMasterData(context.Context, string) (repository.MasterData, error) {
	return repository.MasterData{Key: service.MasterDataKeyIndonesia10YearBondYield, Value: 5}, nil
}

func (r *residualIncomeHandlerRepository) GetFundamentals(context.Context, string) (repository.StockFundamentals, error) {
	payload, err := json.Marshal(map[string]any{
		"perShare": map[string]any{
			"current_book_value_per_share": map[string]any{"value": "1000"},
			"current_eps_ttm":              map[string]any{"value": "100"},
		},
		"managementEffectiveness": map[string]any{
			"return_on_equity_ttm": map[string]any{"value": "10%"},
		},
		"dividendHistory": []any{},
	})
	if err != nil {
		return repository.StockFundamentals{}, err
	}
	return repository.StockFundamentals{Ticker: "TEST", Payload: payload, ScrapedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)}, r.fundamentalsErr
}

func (r *residualIncomeHandlerRepository) GetStockBeta(context.Context, string) (repository.StockBeta, error) {
	return repository.StockBeta{Ticker: "TEST", Value: 1}, r.stockBetaErr
}

func TestCalculateStockResidualIncomeResponseClassification(t *testing.T) {
	databaseErr := errors.New("private database diagnostic")
	for _, tc := range []struct {
		name       string
		repo       *residualIncomeHandlerRepository
		wantStatus int
		wantData   string
		wantLog    string
	}{
		{
			name:       "missing beta is invalid input",
			repo:       &residualIncomeHandlerRepository{stockBetaErr: repository.ErrStockBetaNotFound},
			wantStatus: http.StatusBadRequest,
			wantData:   "invalid residual income valuation assumptions: stock beta is unavailable",
		},
		{
			name:       "repository failure is internal",
			repo:       &residualIncomeHandlerRepository{fundamentalsErr: databaseErr},
			wantStatus: http.StatusInternalServerError,
			wantData:   "failed to calculate stock residual income valuation",
			wantLog:    databaseErr.Error(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			logWriter := io.Writer(io.Discard)
			if tc.wantLog != "" {
				logWriter = &logs
			}
			log := zerolog.New(logWriter)
			h := New("test", &log, nil, Domains{ResidualIncomeValuation: service.NewResidualIncomeValuationService(tc.repo)})
			router := chi.NewRouter()
			router.Post("/stocks/{ticker}/valuation/residual-income", h.Handle(h.CalculateStockResidualIncome))
			req := httptest.NewRequest(http.MethodPost, "/stocks/TEST/valuation/residual-income", strings.NewReader(`{"terminal_roe":0.10,"terminal_growth_rate":0.03,"equity_risk_premium":0.06}`))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			assertFailure(t, res, tc.wantStatus, tc.wantData)
			if tc.wantLog != "" && !strings.Contains(logs.String(), tc.wantLog) {
				t.Fatalf("internal cause missing from logs: %s", &logs)
			}
		})
	}
}

func TestCalculateStockResidualIncomeSuccessEnvelope(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := New("test", &log, nil, Domains{ResidualIncomeValuation: service.NewResidualIncomeValuationService(&residualIncomeHandlerRepository{})})
	router := chi.NewRouter()
	router.Post("/stocks/{ticker}/valuation/residual-income", h.Handle(h.CalculateStockResidualIncome))
	req := httptest.NewRequest(http.MethodPost, "/stocks/TEST/valuation/residual-income", strings.NewReader(`{"terminal_roe":0.10,"terminal_growth_rate":0.03,"equity_risk_premium":0.06}`))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if body["status"] != "ok" || !ok || data["ticker"] != "TEST" || data["forecast_years"] != float64(5) {
		t.Fatalf("unexpected success envelope: %v", body)
	}
}
