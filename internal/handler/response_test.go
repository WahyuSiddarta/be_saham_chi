package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func TestDomainPayloadsHaveOneEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
		list bool
	}{
		{"registration", NewRegisterResponse(repository.User{Status: true}, repository.Portfolio{}), false},
		{"portfolio", NewPortfolioResponse(repository.Portfolio{}), false},
		{"portfolios", NewPortfolioListResponse(nil), true},
		{"quote", NewQuoteResponse(repository.MarketPrice{}), false},
		{"klines", NewKlineListResponse(nil), true},
		{"cash", NewPortfolioCashResponse(repository.PortfolioCash{}), false},
		{"cash transaction", NewCashTransactionResponse(repository.PortfolioCashTransaction{}), false},
		{"cash transactions", NewCashTransactionListResponse(nil), true},
		{"cash snapshots", NewCashSnapshotListResponse(nil), true},
		{"bond", NewBondResponse(repository.PortfolioBond{}), false},
		{"bonds", NewBondListResponse(nil), true},
		{"bond transaction", NewBondTransactionResponse(repository.PortfolioBondTransaction{}), false},
		{"bond transactions", NewBondTransactionListResponse(nil), true},
		{"bond snapshots", NewBondSnapshotListResponse(nil), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			if err := response.Success(res, 200, tc.data); err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 2 || body["status"] != "ok" {
				t.Fatalf("wrong envelope: %v", body)
			}
			if tc.list {
				if _, ok := body["data"].([]any); !ok {
					t.Fatalf("data must be array: %v", body)
				}
			} else {
				data, ok := body["data"].(map[string]any)
				if !ok {
					t.Fatalf("data must be object: %v", body)
				}
				if _, exists := data["status"]; exists {
					t.Fatalf("nested status: %v", data)
				}
				if _, exists := data["data"]; exists {
					t.Fatalf("nested envelope: %v", data)
				}
			}
		})
	}
}

type deletePortfolioStore struct {
	service.PortfolioRepository
	userID, portfolioID, targetID string
}

func (s *deletePortfolioStore) DeleteAndMove(_ context.Context, userID, portfolioID, targetID string) error {
	s.userID, s.portfolioID, s.targetID = userID, portfolioID, targetID
	return nil
}
func TestDeleteReturnsSuccessEnvelope(t *testing.T) {
	store := &deletePortfolioStore{}
	log := zerolog.New(io.Discard)
	h := New("test", &log, nil, Domains{Portfolio: service.NewPortfolioService(store)})
	config := auth.Config{Secret: "test-secret", Issuer: "test", TTL: time.Hour}
	token, _, err := auth.GenerateToken(config, repository.User{ID: "u-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(auth.Middleware(config))
	router.Delete("/portfolios/{portfolio_id}", h.Handle(h.DeletePortfolio))
	req := httptest.NewRequest(http.MethodDelete, "/portfolios/p-1", strings.NewReader(`{"target_portfolio_id":"p-2"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 200 || res.Body.String() != `{"status":"ok","data":null}` {
		t.Fatalf("status=%d body=%s", res.Code, res.Body)
	}
	if store.userID != "u-1" || store.portfolioID != "p-1" || store.targetID != "p-2" {
		t.Fatalf("wrong delete arguments: %+v", store)
	}
}
